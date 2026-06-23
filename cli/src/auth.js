import chalk from "chalk"
import { loadConfig, saveConfig, api } from "./config.js"
import { createInterface } from "readline"

function question(query) {
  return new Promise((resolve) => {
    const rl = createInterface({ input: process.stdin, output: process.stdout })
    rl.question(query, (answer) => { rl.close(); resolve(answer) })
  })
}

export async function login(token) {
  if (token) {
    saveConfig({ ...loadConfig(), token: token.trim() })
    console.log(chalk.green("✓") + " Token salvo!")
    return
  }
  console.log(chalk.cyan("\n  Nidus Login\n"))
  const email = await question("  Email: ")
  const password = await question("  Senha: ")
  try {
    const data = await api("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    })
    saveConfig({ ...loadConfig(), token: data.token })
    console.log(chalk.green("\n  ✓ Login efetuado como " + chalk.bold(data.user.name)))
  } catch (err) {
    console.log(chalk.red("\n  ✗ " + err.message))
  }
}

export async function whoami() {
  const config = loadConfig()
  if (!config.token) {
    console.log(chalk.yellow("  ⚠ Não logado. Use " + chalk.bold("nidus login")))
    return
  }
  try {
    const user = await api("/api/auth/me")
    console.log(chalk.green("  ✓ ") + chalk.bold(user.name) + " <" + user.email + ">")
  } catch {
    console.log(chalk.red("  ✗ Token inválido. Use " + chalk.bold("nidus login")))
  }
}

export async function logout() {
  saveConfig({})
  console.log(chalk.green("  ✓ Logout efetuado"))
}
