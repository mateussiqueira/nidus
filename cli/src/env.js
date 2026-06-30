import chalk from "chalk"
import { API, loadConfig } from "./config.js"

async function api(path, options = {}) {
  const config = loadConfig()
  const res = await fetch(`${API}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(config.token ? { Authorization: `Bearer ${config.token}` } : {}),
      ...options.headers,
    },
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }))
    throw new Error(err.message || "Erro desconhecido")
  }
  return res.json()
}

async function resolveProject(identifier) {
  const projects = await api("/api/projects")
  const project = projects.find((p) => p.slug === identifier || p.id === identifier || p.name === identifier)
  if (!project) throw new Error(`Projeto "${identifier}" não encontrado`)
  return project
}

export async function envList(projectSlug) {
  try {
    const project = await resolveProject(projectSlug)
    const envs = await api(`/api/projects/${project.id}/envs`)
    if (envs.length === 0) {
      console.log(chalk.yellow(`  ⚠ Nenhuma variável de ambiente em ${chalk.bold(project.name)}`))
      return
    }
    console.log(chalk.cyan(`\n  Variáveis de ambiente — ${chalk.bold(project.name)}:\n`))
    for (const e of envs) {
      const masked = e.value ? e.value.slice(0, 4) + "••••" : "(vazio)"
      console.log(`  ${chalk.green("●")} ${chalk.bold(e.key)} = ${masked}`)
    }
    console.log()
  } catch (err) {
    console.log(chalk.red(`  ✗ ${err.message}`))
  }
}

export async function envSet(projectSlug, key, value) {
  try {
    const project = await resolveProject(projectSlug)
    const envs = await api(`/api/projects/${project.id}/envs`)
    const existing = envs.find((e) => e.key === key)
    if (existing) {
      await api(`/api/projects/${project.id}/envs/${existing.id}`, {
        method: "PATCH",
        body: JSON.stringify({ key, value }),
      })
      console.log(chalk.green(`  ✓ ${key} atualizado em ${chalk.bold(project.name)}`))
    } else {
      await api(`/api/projects/${project.id}/envs`, {
        method: "POST",
        body: JSON.stringify({ key, value }),
      })
      console.log(chalk.green(`  ✓ ${key} criado em ${chalk.bold(project.name)}`))
    }
  } catch (err) {
    console.log(chalk.red(`  ✗ ${err.message}`))
  }
}

export async function envDelete(projectSlug, key) {
  try {
    const project = await resolveProject(projectSlug)
    const envs = await api(`/api/projects/${project.id}/envs`)
    const existing = envs.find((e) => e.key === key)
    if (!existing) {
      console.log(chalk.yellow(`  ⚠ Variável ${key} não encontrada`))
      return
    }
    await api(`/api/projects/${project.id}/envs/${existing.id}`, { method: "DELETE" })
    console.log(chalk.green(`  ✓ ${key} removido de ${chalk.bold(project.name)}`))
  } catch (err) {
    console.log(chalk.red(`  ✗ ${err.message}`))
  }
}
