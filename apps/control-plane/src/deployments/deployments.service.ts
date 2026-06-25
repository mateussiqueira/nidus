import { Injectable, Logger, OnModuleDestroy } from "@nestjs/common"
import { PrismaService } from "../prisma/prisma.service"
import { deployQueue, deployWorker, sanitizeBranch, DeployJobData } from "./deploy.queue"
import Docker from "dockerode"

const docker = new Docker({ socketPath: "/var/run/docker.sock" })
const DEPLOYS_DIR = process.env.NIDUS_DEPLOYS_DIR || "/tmp/nidus-deploys"
const HOST = process.env.NIDUS_HOST || "localhost"

@Injectable()
export class DeploymentsService implements OnModuleDestroy {
  private readonly logger = new Logger(DeploymentsService.name)

  constructor(private readonly prisma: PrismaService) {}

  async onModuleDestroy() {
    await deployWorker.close()
    await deployQueue.close()
  }

  async listByProject(projectId: string) {
    const result = await this.prisma.db.query(
      `SELECT id, status, url, branch, type, created_at, finished_at
       FROM deployments WHERE project_id = $1
       ORDER BY created_at DESC LIMIT 50`,
      [projectId],
    )
    return result.rows.map((row: any) => ({
      id: row.id,
      status: row.status,
      url: row.url,
      branch: row.branch,
      type: row.type,
      createdAt: row.created_at,
      finishedAt: row.finished_at,
    }))
  }

  async getLogs(deploymentId: string) {
    const result = await this.prisma.db.query(
      "SELECT logs FROM deployments WHERE id = $1",
      [deploymentId],
    )
    return result.rows[0]?.logs || ""
  }

  async listPreviews(projectId: string) {
    const result = await this.prisma.db.query(
      `SELECT id, status, url, branch, type, created_at, finished_at
       FROM deployments WHERE project_id = $1 AND type = 'preview'
       ORDER BY created_at DESC LIMIT 20`,
      [projectId],
    )
    return result.rows.map((row: any) => ({
      id: row.id,
      status: row.status,
      url: row.url,
      branch: row.branch,
      type: row.type,
      createdAt: row.created_at,
      finishedAt: row.finished_at,
    }))
  }

  async metrics(slug: string, branch?: string) {
    const containerName = branch
      ? `nidus-${slug}-preview-${sanitizeBranch(branch)}`
      : `nidus-${slug}`
    try {
      const container = docker.getContainer(containerName)
      const inspect = await container.inspect()
      const stats = await container.stats({ stream: false })

      const state = inspect.State || {}
      return {
        status: state.Status || "unknown",
        running: state.Running || false,
        startedAt: state.StartedAt || null,
        uptime: state.StartedAt ? Math.floor((Date.now() - new Date(state.StartedAt).getTime()) / 1000) : 0,
        cpu: stats.cpu_stats ? ((stats.cpu_stats.cpu_usage.total_usage / stats.system_cpu_usage) * 100).toFixed(2) : "0",
        memory: {
          usage: stats.memory_stats ? `${(stats.memory_stats.usage / 1024 / 1024).toFixed(1)}MB` : "0",
          limit: stats.memory_stats ? `${(stats.memory_stats.limit / 1024 / 1024).toFixed(1)}MB` : "0",
          percent: stats.memory_stats?.usage && stats.memory_stats?.limit
            ? ((stats.memory_stats.usage / stats.memory_stats.limit) * 100).toFixed(1)
            : "0",
        },
        network: stats.networks ? Object.values(stats.networks).map((n: any) => `${n.rx_bytes}/${n.tx_bytes}`).join(", ") : "0",
        restartCount: state.RestartCount || 0,
        exitCode: state.ExitCode ?? null,
      }
    } catch {
      return { status: "stopped", cpu: 0, memory: 0 }
    }
  }

  async deploy(projectId: string, branch = "main") {
    const project = await this.prisma.db.query(
      "SELECT id, name, slug, repo_url, domain FROM projects WHERE id = $1",
      [projectId],
    )
    if (!project.rows[0]) throw new Error("Projeto não encontrado")
    const p = project.rows[0]

    const isPreview = branch !== "main"
    const safeBranch = sanitizeBranch(branch)
    const deployType = isPreview ? "preview" : "production"
    const containerName = isPreview
      ? `nidus-${p.slug}-preview-${safeBranch}`
      : `nidus-${p.slug}`
    const imageTag = isPreview
      ? `nidus-${p.slug}:preview-${safeBranch}`
      : `nidus-${p.slug}:latest`

    const dep = await this.prisma.db.query(
      `INSERT INTO deployments (id, project_id, branch, type, status, created_at)
       VALUES (gen_random_uuid(), $1, $2, $3, 'building', NOW())
       RETURNING id`,
      [projectId, branch, deployType],
    )
    const depId = dep.rows[0].id

    const jobData: DeployJobData = {
      deploymentId: depId,
      projectId,
      projectName: p.name,
      projectSlug: p.slug,
      repoUrl: p.repo_url,
      domain: p.domain,
      branch,
      deployType,
      containerName,
      imageTag,
      isPreview,
      safeBranch,
    }

    const job = await deployQueue.add("deploy", jobData, { jobId: depId })
    this.logger.log(`Deploy job ${job.id} queued for ${p.name} (${branch})`)

    return {
      id: depId,
      status: "queued",
      jobId: job.id,
      branch,
      type: deployType,
    }
  }

  async getJobStatus(deploymentId: string) {
    const result = await this.prisma.db.query(
      `SELECT id, status, url, branch, type, created_at, finished_at
       FROM deployments WHERE id = $1`,
      [deploymentId],
    )
    if (!result.rows[0]) return null
    const row = result.rows[0]
    return {
      id: row.id,
      status: row.status,
      url: row.url,
      branch: row.branch,
      type: row.type,
      createdAt: row.created_at,
      finishedAt: row.finished_at,
    }
  }
}
