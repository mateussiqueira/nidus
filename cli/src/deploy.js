import { readFileSync, existsSync } from "fs"
import { execSync } from "child_process"
import chalk from "chalk"
import { api } from "./config.js"

export async function deploy(options) {
  const pkg = existsSync("package.json") ? JSON.parse(readFileSync("package.json", "utf-8")) : {}
  const slug = options.project || pkg.name
  if (!slug) {
    console.log(chalk.red("  ✗ Informe o projeto com --project ou configure package.json com 'name'"))
    return
  }

  const framework = detectFramework()
  console.log(chalk.cyan(`\n  🚀 Deploying ${chalk.bold(slug)} (${framework})...\n`))

  try {
    let repoUrl = ""
    try { repoUrl = execSync("git remote get-url origin 2>/dev/null").toString().trim() } catch {}

    // Create project if not exists
    let projects = await api("/api/projects")
    let project = projects.find(p => p.slug === slug)
    if (!project) {
      project = await api("/api/projects", {
        method: "POST",
        body: JSON.stringify({ name: slug, slug, repoUrl, framework }),
      })
      console.log(chalk.gray(`  ✓ Projeto ${slug} criado`))
    }

    const result = await api(`/api/projects/${project.id}/deploy`, { method: "POST" })
    if (result.status === "success") {
      console.log(chalk.green(`  ✅ Deploy concluído!`))
      if (result.url) console.log(chalk.cyan(`  🔗 ${result.url}`))
    } else {
      console.log(chalk.red(`  ❌ Falhou: ${result.error || "erro desconhecido"}`))
    }
  } catch (err) {
    console.log(chalk.red(`  ✗ ${err.message}`))
  }
}

function detectFramework() {
  if (existsSync("next.config.mjs") || existsSync("next.config.js")) return "nextjs"
  if (existsSync("pubspec.yaml")) return "vaden"
  if (existsSync("vite.config.ts") || existsSync("vite.config.js")) return "vite"
  if (existsSync("index.html")) return "static"
  if (existsSync("package.json")) {
    const pkg = JSON.parse(readFileSync("package.json", "utf-8"))
    if (pkg.dependencies?.express) return "express"
    if (pkg.dependencies?.fastify) return "fastify"
  }
  return "generic"
}
