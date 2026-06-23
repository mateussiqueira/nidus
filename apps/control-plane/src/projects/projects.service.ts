import { Injectable } from "@nestjs/common"
import { PrismaService } from "../prisma/prisma.service"

@Injectable()
export class ProjectsService {
  constructor(private readonly prisma: PrismaService) {}

  async list(userId: string) {
    const result = await this.prisma.db.query(
      "SELECT id, name, slug, framework, status, domain, repo_url, created_at FROM projects WHERE user_id = $1 ORDER BY created_at DESC",
      [userId],
    )
    return result.rows.map(this.mapProject)
  }

  async get(id: string, userId: string) {
    const result = await this.prisma.db.query(
      "SELECT id, name, slug, framework, status, domain, repo_url, created_at FROM projects WHERE id = $1 AND user_id = $2",
      [id, userId],
    )
    return result.rows[0] ? this.mapProject(result.rows[0]) : null
  }

  async create(userId: string, data: { name: string; slug: string; repoUrl?: string }) {
    const result = await this.prisma.db.query(
      `INSERT INTO projects (id, name, slug, user_id, repo_url, status, created_at, updated_at)
       VALUES (gen_random_uuid(), $1, $2, $3, $4, 'ACTIVE', NOW(), NOW())
       RETURNING id, name, slug, framework, status, domain, repo_url, created_at`,
      [data.name, data.slug, userId, data.repoUrl ?? null],
    )
    return this.mapProject(result.rows[0])
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
      createdAt: row.created_at,
    }
  }
}
