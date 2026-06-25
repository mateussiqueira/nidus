import { Injectable, Logger, OnModuleDestroy } from "@nestjs/common"
import { PrismaService } from "../prisma/prisma.service"
import { execSync } from "child_process"
import { deployQueue, deployWorker, sanitizeBranch, DeployJobData } from "./deploy.queue"

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
      `SELECT id, status, url, logs, branch, type, created_at, finished_at
       FROM deployments WHERE project_id = $1
       ORDER BY created_at DESC LIMIT 50`,
      [projectId],
    )
    return result.rows.map((row: any) => ({
      id: row.id,
      status: row.status,
      url: row.url,
      logs: row.logs,
      branch: row.branch,
      type: row.type,
      createdAt: row.created_at,
      finishedAt: row.finished_at,
    }))
  }

  async listPreviews(projectId: string) {
    const result = await this.prisma.db.query(
      `SELECT id, status, url, logs, branch, type, created_at, finished_at
       FROM deployments WHERE project_id = $1 AND type = 'preview'
       ORDER BY created_at DESC LIMIT 20`,
      [projectId],
    )
    return result.rows.map((row: any) => ({
      id: row.id,
      status: row.status,
      url: row.url,
      logs: row.logs,
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
      const inspect = JSON.parse(execSync(`docker inspect ${containerName} 2>/dev/null || echo '{}'`).toString())
      if (!inspect || inspect.length === 0) return { status: "stopped", cpu: 0, memory: 0, uptime: 0 }

      const state = inspect[0].State || {}
      const stats = JSON.parse(execSync(`docker stats ${containerName} --no-stream --format '{{json .}}' 2>/dev/null || echo '{}'`).toString())

      return {
        status: state.Status || "unknown",
        running: state.Running || false,
        startedAt: state.StartedAt || null,
        uptime: state.StartedAt ? Math.floor((Date.now() - new Date(state.StartedAt).getTime()) / 1000) : 0,
        cpu: parseFloat(stats?.CPUPerc?.replace("%", "") || "0"),
        memory: {
          usage: stats?.MemUsage?.split("/")[0]?.trim() || "0",
          limit: stats?.MemUsage?.split("/")[1]?.trim() || "0",
          percent: parseFloat(stats?.MemPerc?.replace("%", "") || "0"),
        },
        network: stats?.NetIO || "0 / 0",
        blockIO: stats?.BlockIO || "0 / 0",
        restartCount: state.RestartCount || 0,
        exitCode: state.ExitCode ?? null,
      }
    } catch {
      return { status: "error", cpu: 0, memory: 0 }
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

    const job = await deployQueue.add("deploy", jobData, {
      jobId: depId,
    })

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
      `SELECT id, status, url, logs, branch, type, created_at, finished_at
       FROM deployments WHERE id = $1`,
      [deploymentId],
    )
    if (!result.rows[0]) return null

    const row = result.rows[0]
    return {
      id: row.id,
      status: row.status,
      url: row.url,
      logs: row.logs,
      branch: row.branch,
      type: row.type,
      createdAt: row.created_at,
      finishedAt: row.finished_at,
    }
  }
}
