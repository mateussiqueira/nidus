import { Injectable } from "@nestjs/common"
import { PrismaService } from "../prisma/prisma.service"

@Injectable()
export class ProjectsService {
  constructor(private readonly prisma: PrismaService) {}

  async list(userId: string) {
    const result = await this.prisma.db.query(
      "SELECT id, name, slug, framework, status, domain, repo_url, env_vars, created_at FROM projects WHERE user_id = $1 ORDER BY created_at DESC",
      [userId],
    )
    return result.rows.map(this.mapProject)
  }

  async get(id: string, userId: string) {
    const result = await this.prisma.db.query(
      "SELECT id, name, slug, framework, status, domain, repo_url, env_vars, created_at FROM projects WHERE id = $1 AND user_id = $2",
      [id, userId],
    )
    return result.rows[0] ? this.mapProject(result.rows[0]) : null
  }

  async create(userId: string, data: { name: string; slug: string; repoUrl?: string; framework?: string }) {
    const result = await this.prisma.db.query(
      `INSERT INTO projects (id, name, slug, user_id, repo_url, framework, status, created_at, updated_at)
       VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, 'ACTIVE', NOW(), NOW())
       RETURNING id, name, slug, framework, status, domain, repo_url, env_vars, created_at`,
      [data.name, data.slug, userId, data.repoUrl ?? null, data.framework ?? null],
    )
    return this.mapProject(result.rows[0])
  }

  async update(id: string, userId: string, data: { envVars?: string; domain?: string; repoUrl?: string }) {
    const sets: string[] = []
    const params: any[] = []
    let idx = 1

    if (data.envVars !== undefined) { sets.push(`env_vars = $${idx++}`); params.push(data.envVars) }
    if (data.domain !== undefined) { sets.push(`domain = $${idx++}`); params.push(data.domain) }
    if (data.repoUrl !== undefined) { sets.push(`repo_url = $${idx++}`); params.push(data.repoUrl) }
    if (sets.length === 0) return this.get(id, userId)

    sets.push(`updated_at = NOW()`)
    params.push(id, userId)
    await this.prisma.db.query(
      `UPDATE projects SET ${sets.join(", ")} WHERE id = $${idx++} AND user_id = $${idx}`,
      params,
    )
    return this.get(id, userId)
  }

  private mapProject(row: any) {
    return {
      id: row.id,
      name: row.name,
      slug: row.slug,
      framework: row.framework,
      status: row.status,
      domain: row.domain,
      repoUrl: row.repo_url,
      envVars: row.env_vars,
      createdAt: row.created_at,
    }
  }
}
