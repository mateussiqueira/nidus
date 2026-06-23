import { Injectable } from "@nestjs/common"
import { PrismaService } from "../prisma/prisma.service"

@Injectable()
export class DeploymentsService {
  constructor(private readonly prisma: PrismaService) {}

  async listByProject(projectId: string) {
    const result = await this.prisma.db.query(
      "SELECT id, status, url, created_at, finished_at FROM deployments WHERE project_id = $1 ORDER BY created_at DESC LIMIT 20",
      [projectId],
    )
    return result.rows.map((row: any) => ({
      id: row.id,
      status: row.status,
      url: row.url,
      createdAt: row.created_at,
      finishedAt: row.finished_at,
    }))
  }
}
