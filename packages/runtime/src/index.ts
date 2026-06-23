import { execSync, spawn } from "child_process"
import { writeFileSync, existsSync, mkdirSync } from "fs"
import { join } from "path"
import type { ProjectConfig, DeploymentResult, Framework } from "@canopy/shared"

const DOCKER_DIR = join(process.cwd(), "..", "..", "docker")

function generateDockerfile(framework: Framework, outputDir?: string): string {
  switch (framework) {
    case "nextjs":
      return `
FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/static ./.next/static
EXPOSE 3000
CMD ["node", "server.js"]
`
    case "vaden":
      return `
FROM dart:stable AS builder
WORKDIR /app
COPY pubspec.* ./
RUN dart pub get
COPY . .
RUN dart compile exe bin/main.dart -o /app/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/server ./
EXPOSE 8080
CMD ["./server"]
`
    case "express":
      return `
FROM node:22-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
EXPOSE 3000
CMD ["node", "index.js"]
`
    default:
      return `
FROM alpine:latest
WORKDIR /app
COPY . .
EXPOSE 8080
CMD ["./start.sh"]
`
  }
}

export async function buildProject(config: ProjectConfig, sourceDir: string): Promise<DeploymentResult> {
  try {
    const dockerfile = generateDockerfile(config.framework, config.outputDir)
    const dockerfilePath = join(sourceDir, "Dockerfile.canopy")
    writeFileSync(dockerfilePath, dockerfile)

    const tag = `canopy/${config.slug}:latest`
    execSync(`docker build -t ${tag} -f ${dockerfilePath} .`, {
      cwd: sourceDir,
      stdio: "pipe",
    })

    return { success: true, logs: `Build concluído: ${tag}` }
  } catch (err: any) {
    return { success: false, error: err.message, logs: err.stderr?.toString() }
  }
}

export async function deployProject(config: ProjectConfig, port: number): Promise<DeploymentResult> {
  try {
    const tag = `canopy/${config.slug}:latest`
    const containerName = `canopy-${config.slug}`

    execSync(`docker rm -f ${containerName} 2>/dev/null || true`, { stdio: "pipe" })

    const container = spawn("docker", [
      "run", "-d",
      "--name", containerName,
      "--label", `canopy.project=${config.slug}`,
      ...(config.env ? Object.entries(config.env).flatMap(([k, v]) => ["-e", `${k}=${v}`]) : []),
      "-p", `${port}:8080`,
      "--restart", "unless-stopped",
      tag,
    ])

    return new Promise((resolve) => {
      container.on("close", (code) => {
        if (code === 0) {
          resolve({
            success: true,
            url: `http://localhost:${port}`,
            logs: `Container ${containerName} iniciado na porta ${port}`,
          })
        } else {
          resolve({ success: false, error: `Docker run falhou com código ${code}` })
        }
      })
    })
  } catch (err: any) {
    return { success: false, error: err.message }
  }
}

export async function stopProject(slug: string): Promise<void> {
  execSync(`docker rm -f canopy-${slug} 2>/dev/null || true`, { stdio: "pipe" })
}
