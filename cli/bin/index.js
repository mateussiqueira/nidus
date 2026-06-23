#!/usr/bin/env node
import { Command } from "commander"
import { deploy } from "../src/deploy.js"
import { login, whoami, logout } from "../src/auth.js"
import { list } from "../src/projects.js"

const program = new Command()

program
  .name("nidus")
  .description("Nidus CLI — deploy como a Vercel, melhor que a Vercel")
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
  .action(deploy)

program
  .command("projects")
  .description("Listar projetos")
  .alias("ls")
  .action(list)

program.parse()
