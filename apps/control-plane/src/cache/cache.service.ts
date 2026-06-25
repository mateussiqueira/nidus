import { Injectable, Logger } from "@nestjs/common"
import { PrismaService } from "../prisma/prisma.service"

const CACHE_TTL = 30

@Injectable()
export class CacheService {
  private readonly logger = new Logger(CacheService.name)
  private cache = new Map<string, { data: any; expires: number }>()

  constructor(private readonly prisma: PrismaService) {}

  private get(key: string): any | null {
    const entry = this.cache.get(key)
    if (!entry) return null
    if (Date.now() > entry.expires) {
      this.cache.delete(key)
      return null
    }
    return entry.data
  }

  private set(key: string, data: any, ttl = CACHE_TTL): void {
    this.cache.set(key, { data, expires: Date.now() + ttl * 1000 })
  }

  invalidate(pattern: string): void {
    for (const key of this.cache.keys()) {
      if (key.includes(pattern)) {
        this.cache.delete(key)
      }
    }
  }

  async getProjects(userId: string) {
    const key = `projects:${userId}`
    const cached = this.get(key)
    if (cached) return cached

    const result = await this.prisma.db.query(
      `SELECT id, name, slug, status, framework, domain, created_at, updated_at
       FROM projects WHERE user_id = $1 ORDER BY updated_at DESC`,
      [userId],
    )
    const data = result.rows.map((row: any) => ({
      id: row.id,
      name: row.name,
      slug: row.slug,
      status: row.status,
      framework: row.framework,
      domain: row.domain,
      createdAt: row.created_at,
      updatedAt: row.updated_at,
    }))
    this.set(key, data, 60)
    return data
  }

  async getProject(projectId: string, userId: string) {
    const key = `project:${projectId}:${userId}`
    const cached = this.get(key)
    if (cached) return cached

    const result = await this.prisma.db.query(
      `SELECT id, name, slug, user_id, repo_url, branch, framework, status, domain, env_vars, created_at, updated_at
       FROM projects WHERE id = $1 AND user_id = $2`,
      [projectId, userId],
    )
    if (!result.rows[0]) return null
    
    const row = result.rows[0]
    const data = {
      id: row.id,
      name: row.name,
      slug: row.slug,
      userId: row.user_id,
      repoUrl: row.repo_url,
      branch: row.branch,
      framework: row.framework,
      status: row.status,
      domain: row.domain,
      envVars: row.env_vars,
      createdAt: row.created_at,
      updatedAt: row.updated_at,
    }
    this.set(key, data, 30)
    return data
  }

  async getDeployments(projectId: string) {
    const key = `deployments:${projectId}`
    const cached = this.get(key)
    if (cached) return cached

    const result = await this.prisma.db.query(
      `SELECT id, status, url, logs, branch, type, created_at, finished_at
       FROM deployments WHERE project_id = $1
       ORDER BY created_at DESC LIMIT 50`,
      [projectId],
    )
    const data = result.rows.map((row: any) => ({
      id: row.id,
      status: row.status,
      url: row.url,
      logs: row.logs,
      branch: row.branch,
      type: row.type,
      createdAt: row.created_at,
      finishedAt: row.finished_at,
    }))
    this.set(key, data, 10)
    return data
  }
}
