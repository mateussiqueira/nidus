import { NestFactory } from "@nestjs/core"
import { AppModule } from "./app.module"
import { Logger } from "@nestjs/common"
import helmet from "helmet"

async function bootstrap() {
  const app = await NestFactory.create(AppModule, {
    logger: ["log", "error", "warn", "debug", "verbose"],
  })

  app.use(helmet({ contentSecurityPolicy: false }))
  app.enableCors({
    origin: ["http://localhost:3000", "http://2.24.204.31:3000"],
    credentials: true,
  })

  const port = process.env.API_PORT ?? 3001
  await app.listen(port)

  Logger.log(`🚀 Control Plane rodando em http://localhost:${port}`, "Bootstrap")
}

bootstrap()
