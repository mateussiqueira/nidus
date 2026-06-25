import { Queue, Worker, Job } from "bullmq"
import { execSync } from "child_process"
import { writeFileSync, existsSync, mkdirSync } from "fs"
import { join } from "path"

const REDIS_URL = process.env.REDIS_URL || "redis://localhost:6379"
const DEPLOYS_DIR = process.env.NIDUS_DEPLOYS_DIR || "/tmp/nidus-deploys"
const HOST = process.env.NIDUS_HOST || "localhost"

const redisOpts = { host: new URL(REDIS_URL).hostname, port: Number(new URL(REDIS_URL).port) || 6379 }

const deployQueue = new Queue("deploy-queue", {
  connection: redisOpts,
  defaultJobOptions: {
    attempts: 2,
    backoff: {
      type: "exponential",
      delay: 5000,
    },
    removeOnComplete: { age: 86400 },
    removeOnFail: { age: 86400 },
  },
})

export interface DeployJobData {
  deploymentId: string
  projectId: string
  projectName: string
  projectSlug: string
  repoUrl: string | null
  domain: string | null
  branch: string
  deployType: string
  containerName: string
  imageTag: string
  isPreview: boolean
  safeBranch: string
}

export interface DeployJobResult {
  status: "success" | "failed"
  url?: string
  logs: string
  error?: string
}

function sanitizeBranch(branch: string): string {
  return branch.toLowerCase().replace(/[^a-z0-9-_.]/g, "-").slice(0, 50)
}

function detectFramework(repoDir: string): string {
  const pkgJson = join(repoDir, "package.json")
  const nextConfig = join(repoDir, "next.config.js")
  const nextConfigTs = join(repoDir, "next.config.ts")
  const nuxtConfig = join(repoDir, "nuxt.config.js")
  const nuxtConfigTs = join(repoDir, "nuxt.config.ts")
  const viteConfig = join(repoDir, "vite.config.js")
  const viteConfigTs = join(repoDir, "vite.config.ts")
  const angularJson = join(repoDir, "angular.json")

  if (existsSync(nextConfig) || existsSync(nextConfigTs)) return "nextjs"
  if (existsSync(nuxtConfig) || existsSync(nuxtConfigTs)) return "nuxt"
  if (existsSync(viteConfig) || existsSync(viteConfigTs)) return "vite"
  if (existsSync(angularJson)) return "angular"
  if (existsSync(pkgJson)) {
    try {
      const pkg = JSON.parse(require("fs").readFileSync(pkgJson, "utf-8"))
      const deps = { ...pkg.dependencies, ...pkg.devDependencies }
      if (deps["next"]) return "nextjs"
      if (deps["nuxt"]) return "nuxt"
      if (deps["vite"]) return "vite"
      if (deps["@angular/core"]) return "angular"
      if (deps["react"]) return "vite"
    } catch { /* ignore */ }
  }
  return "static"
}

function generateDockerfile(framework: string): string {
  switch (framework) {
    case "nextjs":
      return `FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/package.json ./
COPY --from=builder /app/node_modules ./node_modules
EXPOSE 3000
CMD ["npm", "start"]`
    case "nuxt":
      return `FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/.output ./.output
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]`
    case "vite":
      return `FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80`
    case "angular":
      return `FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build --configuration=production

FROM nginx:alpine
COPY --from=builder /app/dist/browser /usr/share/nginx/html
EXPOSE 80`
    default:
      return `FROM nginx:alpine
COPY . /usr/share/nginx/html`
  }
}

function getExposedPort(framework: string): number {
  return framework === "nextjs" || framework === "nuxt" ? 3000 : 80
}

const deployWorker = new Worker(
  "deploy-queue",
  async (job: Job<DeployJobData>): Promise<DeployJobResult> => {
    const {
      deploymentId,
      projectId,
      projectName,
      projectSlug,
      repoUrl,
      domain,
      branch,
      deployType,
      containerName,
      imageTag,
      isPreview,
      safeBranch,
    } = job.data

    const logs: string[] = []
    const log = (msg: string) => {
      logs.push(msg)
      job.updateProgress({ logs: logs.join("\n") })
    }

    const pg = await import("pg")
    const pool = new pg.Pool({
      connectionString: process.env.DATABASE_URL || "postgresql://broto:broto@localhost:5432/nidus?schema=public",
    })

    try {
      log(`🚀 Iniciando deploy de ${projectName} (${branch})...`)
      await pool.query(`UPDATE deployments SET status = 'building', logs = $1 WHERE id = $2`, [logs.join("\n"), deploymentId])

      let repoDir = join(DEPLOYS_DIR, projectSlug)
      if (repoUrl) {
        if (!existsSync(repoDir)) {
          log(`📦 Clonando repositorio...`)
          execSync(`git clone ${repoUrl} ${repoDir} 2>&1`, { timeout: 120000 })
        } else {
          log(`🔄 Actualizando repositorio...`)
          execSync(`git fetch --all 2>&1`, { cwd: repoDir, timeout: 60000 })
        }

        execSync(`git checkout ${branch} 2>&1 && git pull origin ${branch} 2>&1`, {
          cwd: repoDir, timeout: 60000,
        })
        log(`✅ Branch ${branch} actualizada`)
      } else {
        log(`⚠️ Sin repositorio configurado`)
        repoDir = join(DEPLOYS_DIR, projectSlug, "src")
        if (!existsSync(repoDir)) mkdirSync(repoDir, { recursive: true })
        writeFileSync(join(repoDir, "index.html"), `<h1>${projectName}</h1><p>Deploy #${deploymentId.slice(0, 8)} (${branch})</p>`)
        log(`📄 Proyecto creado sin repositorio`)
      }

      const framework = detectFramework(repoDir)
      log(`Framework detectado: ${framework}`)
      log(`🐳 Build da imagen Docker...`)

      const dockerfile = generateDockerfile(framework)
      execSync(`docker build -t ${imageTag} -f- ${repoDir} 2>&1`, {
        input: dockerfile,
        timeout: 300000,
      })
      log(`✅ Build concluido`)

      log(`🔄 Removendo container anterior...`)
      execSync(`docker rm -f ${containerName} 2>/dev/null || true`)
      log(`🚀 Iniciando container...`)

      const exposedPort = getExposedPort(framework)
      execSync(
        `docker run -d --name ${containerName} -p 0:${exposedPort} --restart unless-stopped ${imageTag} 2>&1`,
        { timeout: 30000 },
      )

      const port = execSync(`docker port ${containerName} ${exposedPort} | cut -d: -f2`).toString().trim()
      const url = domain && !isPreview ? `http://${domain}` : `http://${HOST}:${port}`
      log(`✅ Deploy concluido em ${url}`)

      await pool.query(
        `UPDATE deployments SET status = 'success', url = $1, logs = $2, finished_at = NOW() WHERE id = $3`,
        [url, logs.join("\n"), deploymentId],
      )

      if (!isPreview) {
        await pool.query("UPDATE projects SET status = 'ACTIVE' WHERE id = $1", [projectId])
      }

      return { status: "success", url, logs: logs.join("\n") }
    } catch (err: any) {
      const errorMsg = err.message || "Error desconocido"
      await pool.query(
        `UPDATE deployments SET status = 'failed', logs = $1, finished_at = NOW() WHERE id = $2`,
        [logs.join("\n"), deploymentId],
      )

      await pool.query("UPDATE projects SET status = 'FAILED' WHERE id = $1", [projectId])

      return { status: "failed", error: errorMsg, logs: logs.join("\n") }
    } finally {
      await pool.end()
    }
  },
  {
    connection: redisOpts,
    concurrency: 4,
    limiter: {
      max: 10,
      duration: 60000,
    },
  },
)

deployWorker.on("completed", (job) => {
  console.log(`[deploy-worker] Job ${job.id} completed: ${job.data.deploymentId}`)
})

deployWorker.on("failed", (job, err) => {
  console.error(`[deploy-worker] Job ${job?.id} failed: ${err.message}`)
})

export { deployQueue, deployWorker, sanitizeBranch }
