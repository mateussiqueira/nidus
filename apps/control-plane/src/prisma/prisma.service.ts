import { Injectable, OnModuleInit, OnModuleDestroy, Logger } from "@nestjs/common"
import { PrismaClient } from "@prisma/client"
import { PrismaPg } from "@prisma/adapter-pg"
import { Pool } from "pg"

@Injectable()
export class PrismaService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(PrismaService.name)
  public client: PrismaClient
  public pool: Pool

  constructor() {
    const url = process.env.DATABASE_URL
    if (!url) throw new Error("DATABASE_URL is required")
    this.pool = new Pool({
      connectionString: url,
      max: 20,
      min: 5,
      idleTimeoutMillis: 30000,
      connectionTimeoutMillis: 5000,
      statement_timeout: 10000,
      query_timeout: 15000,
    })
    const adapter = new PrismaPg(this.pool)
    this.client = new PrismaClient({ adapter })

    this.pool.on("error", (err: Error) => {
      this.logger.error(`PostgreSQL pool error: ${err.message}`)
    })
  }

  get db() {
    return this.pool
  }

  async onModuleInit() {
    const maxRetries = 5
    for (let i = 0; i < maxRetries; i++) {
      try {
        await this.client.$connect()
        await this.pool.query("SELECT 1")
        this.logger.log("Database connected")
        return
      } catch (err: any) {
        this.logger.warn(`Database connection attempt ${i + 1}/${maxRetries} failed: ${err.message}`)
        if (i === maxRetries - 1) throw err
        await new Promise((r) => setTimeout(r, 2000 * (i + 1)))
      }
    }
  }

  async onModuleDestroy() {
    await this.client.$disconnect()
    await this.pool.end()
  }
}
