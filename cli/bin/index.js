#!/usr/bin/env node

import { Command } from "commander"
import { deploy } from "../src/deploy.js"
import { login, whoami, logout } from "../src/auth.js"
import { list } from "../src/projects.js"
import { logs } from "../src/logs.js"
import { envList, envSet, envDelete } from "../src/env.js"
import { dbList, dbCreate } from "../src/db.js"

const program = new Command()

program
  .name("nidus")
  .description("Nidus CLI — deploy como a Vercel, melhor que a Vercel")
  .version("0.1.0")

program
  .command("login")
  .description("Autenticar na Nidus (interativo)")
  .action(login)

program
  .command("whoami")
  .description("Mostrar usuário logado")
  .action(whoami)

program
  .command("logout")
  .description("Sair da conta")
  .action(logout)

program
  .command("deploy")
  .description("Fazer deploy do diretório atual")
  .option("-p, --project <slug>", "Slug do projeto")
  .action(deploy)

program
  .command("projects")
  .description("Listar projetos")
  .alias("ls")
  .action(list)

program
  .command("logs")
  .description("Ver logs de um projeto")
  .argument("<project>", "Slug ou ID do projeto")
  .option("-d, --deployment <id>", "ID do deployment específico")
  .option("-l, --list", "Listar deployments do projeto")
  .action(logs)

program
  .command("env")
  .description("Gerenciar variáveis de ambiente")
  .argument("<action>", "list, set, ou delete")
  .argument("<project>", "Slug ou ID do projeto")
  .argument("[key]", "Nome da variável")
  .argument("[value]", "Valor da variável")
  .action((action, project, key, value) => {
    switch (action) {
      case "list":
        envList(project)
        break
      case "set":
        if (!key || !value) {
          console.log("  Uso: nidus env set <projeto> <key> <value>")
          return
        }
        envSet(project, key, value)
        break
      case "delete":
        if (!key) {
          console.log("  Uso: nidus env delete <projeto> <key>")
          return
        }
        envDelete(project, key)
        break
      default:
        console.log("  Ações: list, set, delete")
    }
  })

program
  .command("db")
  .description("Gerenciar bancos de dados")
  .argument("<action>", "list ou create")
  .argument("[name]", "Nome do banco (para create)")
  .option("-t, --type <type>", "Tipo do banco (postgres, mysql, redis)", "postgres")
  .option("-p, --project <slug>", "Associar a um projeto")
  .action((action, name, opts) => {
    switch (action) {
      case "list":
        dbList()
        break
      case "create":
        if (!name) {
          console.log("  Uso: nidus db create <nome> [-t postgres|mysql|redis] [-p projeto]")
          return
        }
        dbCreate(name, opts)
        break
      default:
        console.log("  Ações: list, create")
    }
  })

program.parse()
