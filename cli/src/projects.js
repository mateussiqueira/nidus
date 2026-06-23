import chalk from "chalk"
import { api } from "./config.js"

export async function list() {
  const projects = await api("/api/projects")
  if (projects.length === 0) {
    console.log(chalk.yellow("  Nenhum projeto ainda. Use " + chalk.bold("nidus create <nome>")))
    return
  }
  console.log(chalk.cyan("\n  Projetos:\n"))
  for (const p of projects) {
    const status = p.status === "ACTIVE" ? chalk.green("●") : p.status === "FAILED" ? chalk.red("●") : chalk.yellow("●")
    console.log(`  ${status} ${chalk.bold(p.name)} (${p.slug}) — ${p.framework || "sem framework"}`)
  }
  console.log()
}

export async function create(name, options) {
  const slug = name.toLowerCase().replace(/[^a-z0-9-]/g, "-")
  const project = await api("/api/projects", {
    method: "POST",
    body: JSON.stringify({ name, slug, framework: options.framework }),
  })
  console.log(chalk.green(`  ✓ Projeto ${chalk.bold(project.name)} criado! Slug: ${project.slug}`))
}
