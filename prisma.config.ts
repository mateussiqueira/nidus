import "dotenv/config"
import { defineConfig } from "prisma/config"

export default defineConfig({
  schema: "apps/control-plane/prisma/schema.prisma",
  migrations: {
    path: "apps/control-plane/prisma/migrations",
  },
  datasource: {
    url: process.env["DATABASE_URL"] ?? "postgresql://canopy:canopy@localhost:5433/canopy?schema=public",
  },
})
