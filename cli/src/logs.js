import chalk from "chalk"
import { API, loadConfig } from "./config.js"

export async function logs(projectSlug, options) {
  const config = loadConfig()
  const projects = await fetch(`${API}/api/projects`, {
    headers: { Authorization: `Bearer ${config.token}` },
  }).then((r) => r.json())

  const project = projects.find((p) => p.slug === projectSlug || p.id === projectSlug)
  if (!project) {
    console.log(chalk.red(`  ✗ Projeto "${projectSlug}" não encontrado`))
    return
  }

  const deps = await fetch(`${API}/api/projects/${project.id}/deployments`, {
    headers: { Authorization: `Bearer ${config.token}` },
  }).then((r) => r.json())

  if (deps.length === 0) {
    console.log(chalk.yellow("  ⚠ Nenhum deployment encontrado"))
    return
  }

  if (options.list) {
    console.log(chalk.cyan(`\n  Deployments de ${chalk.bold(project.name)}:\n`))
    for (const d of deps.slice(0, 10)) {
      const icon = d.status === "success" ? chalk.green("●") : d.status === "failed" ? chalk.red("●") : chalk.yellow("●")
      const date = new Date(d.createdAt).toLocaleString("pt-BR")
      console.log(`  ${icon} ${chalk.bold(d.id.slice(0, 8))} — ${d.status} — ${date}`)
    }
    console.log()
    return
  }

  const target = options.deployment || deps[0].id
  const dep = deps.find((d) => d.id === target || d.id.startsWith(target))
  if (!dep) {
    console.log(chalk.red(`  ✗ Deployment "${target}" não encontrado`))
    return
  }

  try {
    const logRes = await fetch(`${API}/api/projects/${project.id}/deployments/${dep.id}/logs`, {
      headers: { Authorization: `Bearer ${config.token}` },
    })
    if (logRes.ok) {
      const text = await logRes.text()
      console.log(chalk.cyan(`\n  Logs de ${chalk.bold(dep.id.slice(0, 8))}:\n`))
      console.log(text)
    } else {
      console.log(chalk.yellow("  ⚠ Nenhum log disponível para este deployment"))
    }
  } catch (err) {
    console.log(chalk.red(`  ✗ Erro ao buscar logs: ${err.message}`))
  }
}
