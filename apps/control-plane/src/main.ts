import { NestFactory } from "@nestjs/core"
import { AppModule } from "./app.module"
import { Logger, ValidationPipe } from "@nestjs/common"
import helmet from "helmet"
import compression from "compression"
import { DocumentBuilder, SwaggerModule } from "@nestjs/swagger"
import { randomUUID } from "crypto"

async function bootstrap() {
  const app = await NestFactory.create(AppModule, {
    logger: ["log", "error", "warn"],
  })

  app.use(helmet({ contentSecurityPolicy: false }))
  app.use(compression({
    level: 6,
    threshold: 1024,
  }))

  app.use((req: any, _res: any, next: any) => {
    req.id = req.headers["x-request-id"] || randomUUID()
    next()
  })

  app.useGlobalPipes(new ValidationPipe({
    whitelist: true,
    forbidNonWhitelisted: true,
    transform: true,
  }))

  const corsOrigins = process.env.CORS_ORIGINS?.split(",") ?? ["http://localhost:3000"]
  app.enableCors({
    origin: corsOrigins,
    credentials: true,
    methods: ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"],
    maxAge: 86400,
  })

  if (process.env.NODE_ENV !== "production") {
    const config = new DocumentBuilder()
      .setTitle("Nidus API")
      .setDescription("Self-hosted PaaS control plane API")
      .setVersion("0.1.0")
      .addBearerAuth()
      .build()
    const document = SwaggerModule.createDocument(app, config)
    SwaggerModule.setup("api/docs", app, document)
  }

  app.enableShutdownHooks()

  const port = process.env.API_PORT ?? 3001
  await app.listen(port)
  Logger.log(`🚀 Nidus API rodando em http://localhost:${port}`, "Bootstrap")
}

bootstrap()
