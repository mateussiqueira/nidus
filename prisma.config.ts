import { defineConfig } from "prisma/config"
import { config } from "dotenv"
import { resolve } from "path"

config({ path: resolve(__dirname, ".env") })

export default defineConfig({
  schema: "apps/control-plane/prisma/schema.prisma",
  migrations: {
    path: "apps/control-plane/prisma/migrations",
  },
  datasource: {
    url: process.env["DATABASE_URL"] ?? "postgresql://broto:broto@localhost:5432/nidus?schema=public",
  },
})
