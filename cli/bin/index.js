#!/usr/bin/env node
import { Command } from "commander"
import { deploy } from "../src/deploy.js"
import { login, whoami, logout } from "../src/auth.js"
import { list, create } from "../src/projects.js"
import { logs } from "../src/logs.js"
import { env } from "../src/env.js"

const program = new Command()

program
  .name("nidus")
  .description("Nidus CLI — deploy como a Vercel")
  .version("0.1.0")

program
  .command("login")
  .description("Autenticar na Nidus")
  .argument("[token]", "Token de API (opcional)")
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
  .option("-m, --message <msg>", "Mensagem do deploy")
  .option("--prod", "Deploy em produção")
  .action(deploy)

program
  .command("projects")
  .description("Listar projetos")
  .action(list)

program
  .command("create")
  .description("Criar novo projeto")
  .argument("<name>", "Nome do projeto")
  .option("--framework <fw>", "Framework (nextjs, express, vaden, static)")
  .action(create)

program
  .command("logs")
  .description("Ver logs de deploy")
  .argument("[project]", "Slug do projeto")
  .option("-n, --lines <n>", "Número de linhas", "50")
  .action(logs)

program
  .command("env")
  .description("Gerenciar variáveis de ambiente")
  .argument("<action>", "list, set, delete")
  .argument("[key]", "Nome da variável")
  .argument("[value]", "Valor da variável")
  .option("-p, --project <slug>", "Slug do projeto")
  .action(env)

program.parse()
