import { Injectable } from "@nestjs/common"
import { PrismaService } from "../prisma/prisma.service"
import { execSync } from "child_process"

const PSQL = "/opt/homebrew/bin/psql -d nidus"
const CREATEDB = "/opt/homebrew/bin/createdb"
const DROPDB = "/opt/homebrew/bin/dropdb"

@Injectable()
export class DatabasesService {
  constructor(private readonly prisma: PrismaService) {}

  async list(userId: string) {
    const result = await this.prisma.db.query(
      `SELECT d.id, d.name, d.url, d.project_id, d.created_at
       FROM databases d
       JOIN projects p ON d.project_id = p.id
       WHERE p.user_id = $1
       ORDER BY d.created_at DESC`,
      [userId],
    )
    return result.rows.map(this.mapDatabase)
  }

  async get(id: string, userId: string) {
    const result = await this.prisma.db.query(
      `SELECT d.id, d.name, d.url, d.project_id, d.created_at
       FROM databases d
       JOIN projects p ON d.project_id = p.id
       WHERE d.id = $1 AND p.user_id = $2`,
      [id, userId],
    )
    return result.rows[0] ? this.mapDatabase(result.rows[0]) : null
  }

  async create(userId: string, data: { projectId: string; name: string }) {
    // Verify project belongs to user
    const project = await this.prisma.db.query(
      "SELECT id FROM projects WHERE id = $1 AND user_id = $2",
      [data.projectId, userId],
    )
    if (!project.rows[0]) throw new Error("Projeto não encontrado")

    // Create database in PostgreSQL
    const dbName = `nidus_${data.name}`
    const dbUser = `nidus_${data.name}`
    const dbPassword = Math.random().toString(36).slice(-16)

    try {
      // Create database
      execSync(`${CREATEDB} ${dbName} 2>&1`, { timeout: 10000 })
      // Create user
      execSync(`${PSQL} -c "CREATE USER ${dbUser} WITH PASSWORD '${dbPassword}';" 2>&1`, { timeout: 10000 })
      // Grant permissions
      execSync(`${PSQL} -c "GRANT ALL PRIVILEGES ON DATABASE ${dbName} TO ${dbUser};" 2>&1`, { timeout: 10000 })
      execSync(`${PSQL} -d ${dbName} -c "GRANT ALL ON SCHEMA public TO ${dbUser};" 2>&1`, { timeout: 10000 })

      const url = `postgresql://${dbUser}:${dbPassword}@localhost:5432/${dbName}`

      // Save to database
      const result = await this.prisma.db.query(
        `INSERT INTO databases (id, project_id, name, url, created_at)
         VALUES (gen_random_uuid(), $1, $2, $3, NOW())
         RETURNING id, name, url, project_id, created_at`,
        [data.projectId, data.name, url],
      )

      // Update project with database reference
      await this.prisma.db.query(
        "UPDATE projects SET database_id = $1 WHERE id = $2",
        [result.rows[0].id, data.projectId],
      )

      return this.mapDatabase(result.rows[0])
    } catch (err: any) {
      throw new Error(`Erro ao criar banco: ${err.message}`)
    }
  }

  async delete(id: string, userId: string) {
    const db = await this.get(id, userId)
    if (!db) throw new Error("Banco não encontrado")

    const dbName = `nidus_${db.name}`
    try {
      execSync(`${DROPDB} ${dbName} 2>&1`, { timeout: 10000 })
      execSync(`${PSQL} -c "DROP USER IF EXISTS nidus_${db.name};" 2>&1`, { timeout: 10000 })
    } catch { /* ignore */ }

    await this.prisma.db.query("DELETE FROM databases WHERE id = $1", [id])
    return { success: true }
  }

  private mapDatabase(row: any) {
    return {
      id: row.id,
      name: row.name,
      url: row.url,
      projectId: row.project_id,
      createdAt: row.created_at,
    }
  }
}
