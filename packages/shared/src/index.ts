export type Framework =
  | "nextjs"
  | "vite"
  | "vaden"
  | "express"
  | "fastify"
  | "flask"
  | "django"
  | "generic"

export type Runtime = "node" | "python" | "go" | "dart" | "rust" | "deno"

export type ProjectStatus = "active" | "building" | "deploying" | "failed" | "paused"
export type DeploymentStatus = "pending" | "building" | "deploying" | "success" | "failed"

export interface ProjectConfig {
  name: string
  slug: string
  framework: Framework
  buildCommand?: string
  outputDir?: string
  startCommand?: string
  env?: Record<string, string>
}

export interface DeploymentResult {
  success: boolean
  url?: string
  logs?: string
  error?: string
}

export function detectFramework(files: string[]): Framework {
  if (files.includes("next.config.mjs") || files.includes("next.config.js")) return "nextjs"
  if (files.includes("vite.config.ts") || files.includes("vite.config.js")) return "vite"
  if (files.includes("pubspec.yaml")) return "vaden"
  if (files.includes("package.json")) return "express"
  if (files.includes("requirements.txt") || files.includes("Pipfile")) return "flask"
  return "generic"
}
