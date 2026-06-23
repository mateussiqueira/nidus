import { Module } from "@nestjs/common"
import { ThrottlerModule } from "@nestjs/throttler"
import { PrismaModule } from "./prisma/prisma.module"
import { AuthModule } from "./auth/auth.module"
import { ProjectsModule } from "./projects/projects.module"
import { WebhookModule } from "./webhook/webhook.module"
import { HealthController } from "./health/health.controller"

@Module({
  imports: [
    ThrottlerModule.forRoot([
      {
        ttl: parseInt(process.env.AUTH_THROTTLE_TTL ?? "60", 10) * 1000,
        limit: parseInt(process.env.AUTH_THROTTLE_LIMIT ?? "10", 10),
      },
    ]),
    PrismaModule,
    AuthModule,
    ProjectsModule,
    WebhookModule,
  ],
  controllers: [HealthController],
})
export class AppModule {}
