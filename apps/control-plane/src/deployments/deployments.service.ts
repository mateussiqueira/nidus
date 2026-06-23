import { Injectable } from "@nestjs/common"
import { PrismaService } from "../prisma/prisma.service"
import { execSync } from "child_process"
import { writeFileSync, existsSync, mkdirSync } from "fs"
import { join } from "path"

const DEPLOYS_DIR = "/root/nidus/deploys"

@Injectable()
export class DeploymentsService {
  constructor(private readonly prisma: PrismaService) {
    if (!existsSync(DEPLOYS_DIR)) mkdirSync(DEPLOYS_DIR, { recursive: true })
  }

  async listByProject(projectId: string) {
    const result = await this.prisma.db.query(
      "SELECT id, status, url, logs, created_at, finished_at FROM deployments WHERE project_id = $1 ORDER BY created_at DESC LIMIT 20",
      [projectId],
    )
    return result.rows.map((row: any) => ({
      id: row.id,
      status: row.status,
      url: row.url,
      logs: row.logs,
      createdAt: row.created_at,
      finishedAt: row.finished_at,
    }))
  }

  async metrics(slug: string) {
    const containerName = `nidus-${slug}`
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

  async deploy(projectId: string) {
    const project = await this.prisma.db.query(
      "SELECT id, name, slug, repo_url, domain FROM projects WHERE id = $1",
      [projectId],
    )
    if (!project.rows[0]) throw new Error("Projeto não encontrado")
    const p = project.rows[0]

    const dep = await this.prisma.db.query(
      `INSERT INTO deployments (id, project_id, status, created_at)
       VALUES (gen_random_uuid(), $1, 'building', NOW())
       RETURNING id`,
      [projectId],
    )
    const depId = dep.rows[0].id

    try {
      const logs: string[] = []
      const log = (msg: string) => { logs.push(msg) }

      log(`🚀 Iniciando deploy de ${p.name}...`)

      let repoDir = join(DEPLOYS_DIR, p.slug)
      if (p.repo_url) {
        if (!existsSync(repoDir)) {
          log(`📦 Clonando repositório...`)
          execSync(`git clone ${p.repo_url} ${repoDir} 2>&1`, { timeout: 120000 })
        } else {
          log(`🔄 Atualizando repositório...`)
          execSync(`git pull`, { cwd: repoDir, timeout: 60000 })
        }
        log(`✅ Repositório atualizado`)
      } else {
        log(`⚠️ Sem repositório configurado`)
        repoDir = join(DEPLOYS_DIR, p.slug, "src")
        if (!existsSync(repoDir)) mkdirSync(repoDir, { recursive: true })
        writeFileSync(join(repoDir, "index.html"), `<h1>${p.name}</h1><p>Deploy #${depId.slice(0,8)}</p>`)
        log(`📄 Projeto criado sem repositório`)
      }

      const tag = `nidus-${p.slug}:latest`
      const containerName = `nidus-${p.slug}`

      log(`🐳 Build da imagem Docker...`)
      execSync(`docker build -t ${tag} -f- ${repoDir} 2>&1`, {
        input: `FROM nginx:alpine\nCOPY . /usr/share/nginx/html`,
        timeout: 120000,
      })
      log(`✅ Build concluído`)

      log(`🔄 Removendo container antigo...`)
      execSync(`docker rm -f ${containerName} 2>/dev/null || true`)
      log(`🚀 Iniciando container...`)
      execSync(`docker run -d --name ${containerName} -p 0:80 --restart unless-stopped ${tag} 2>&1`, { timeout: 30000 })

      const port = execSync(`docker port ${containerName} 80 | cut -d: -f2`).toString().trim()
      const url = p.domain ? `http://${p.domain}` : `http://2.24.204.31:${port}`
      log(`✅ Deploy concluído em ${url}`)

      await this.prisma.db.query(
        "UPDATE deployments SET status = 'success', url = $1, logs = $2, finished_at = NOW() WHERE id = $3",
        [url, logs.join("\n"), depId],
      )
      await this.prisma.db.query("UPDATE projects SET status = 'ACTIVE' WHERE id = $1", [projectId])

      return { id: depId, status: "success", url, logs: logs.join("\n") }
    } catch (err: any) {
      const errorMsg = err.message || "Erro desconhecido"
      await this.prisma.db.query(
        "UPDATE deployments SET status = 'failed', logs = $1, finished_at = NOW() WHERE id = $2",
        [errorMsg, depId],
      )
      await this.prisma.db.query("UPDATE projects SET status = 'FAILED' WHERE id = $1", [projectId])
      return { id: depId, status: "failed", error: errorMsg }
    }
  }
}
