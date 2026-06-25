import { defineConfig } from "prisma/config"
import { config } from "dotenv"
import { resolve } from "path"

config({ path: resolve(__dirname, ".env") })

const url = process.env["DATABASE_URL"]
if (!url) throw new Error("DATABASE_URL is required in .env or environment")

export default defineConfig({
  schema: "apps/control-plane/prisma/schema.prisma",
  migrations: {
    path: "apps/control-plane/prisma/migrations",
  },
  datasource: {
    url,
  },
})
