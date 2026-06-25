import { Controller, Get, Logger } from "@nestjs/common"
import { PrismaService } from "../prisma/prisma.service"

@Controller()
export class HealthController {
  private readonly logger = new Logger(HealthController.name)

  constructor(private readonly prisma: PrismaService) {}

  @Get("health")
  async check() {
    try {
      await this.prisma.db.query("SELECT 1")
      return {
        status: "ok",
        name: "nidus-control-plane",
        version: "0.1.0",
        timestamp: new Date().toISOString(),
        dbConnected: true,
      }
    } catch (err: any) {
      this.logger.error(`Health check failed: ${err.message}`)
      return { status: "error", message: err.message }
    }
  }
}
