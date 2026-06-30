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

export async function dbList() {
  try {
    const dbs = await api("/api/databases")
    if (dbs.length === 0) {
      console.log(chalk.yellow("  ⚠ Nenhum banco de dados provisionado"))
      return
    }
    console.log(chalk.cyan("\n  Bancos de dados:\n"))
    for (const db of dbs) {
      const icon = db.status === "ACTIVE" ? chalk.green("●") : chalk.yellow("●")
      console.log(`  ${icon} ${chalk.bold(db.name)} — ${db.type || "postgres"} — ${db.host}:${db.port}`)
    }
    console.log()
  } catch (err) {
    console.log(chalk.red(`  ✗ ${err.message}`))
  }
}

export async function dbCreate(name, options) {
  try {
    const body = { name, type: options.type || "postgres" }
    if (options.project) body.projectSlug = options.project
    const db = await api("/api/databases", {
      method: "POST",
      body: JSON.stringify(body),
    })
    console.log(chalk.green(`  ✓ Banco ${chalk.bold(db.name)} criado!`))
    console.log(chalk.cyan(`  Host: ${db.host}:${db.port}`))
    console.log(chalk.cyan(`  Database: ${db.databaseName}`))
    console.log(chalk.cyan(`  User: ${db.username}`))
    console.log(chalk.yellow("  ⚠ A senha foi exibida apenas no momento da criação"))
  } catch (err) {
    console.log(chalk.red(`  ✗ ${err.message}`))
  }
}
