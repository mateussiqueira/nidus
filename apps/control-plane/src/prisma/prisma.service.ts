import { Injectable, OnModuleInit, OnModuleDestroy, Logger } from "@nestjs/common"
import { PrismaClient } from "@prisma/client"
import { PrismaPg } from "@prisma/adapter-pg"
import { Pool } from "pg"

const DATABASE_URL = "postgresql://broto:broto@localhost:5432/nidus?schema=public"

@Injectable()
export class PrismaService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(PrismaService.name)
  public client: PrismaClient
  public pool: Pool

  constructor() {
    const url = process.env.DATABASE_URL ?? DATABASE_URL
    this.pool = new Pool({ connectionString: url })
    const adapter = new PrismaPg(this.pool)
    this.client = new PrismaClient({ adapter })
  }

  get db() {
    return this.pool
  }

  async onModuleInit() {
    await this.client.$connect()
    await this.pool.query("SELECT 1")
    this.logger.log("Database connected")
  }

  async onModuleDestroy() {
    await this.client.$disconnect()
    await this.pool.end()
  }
}
