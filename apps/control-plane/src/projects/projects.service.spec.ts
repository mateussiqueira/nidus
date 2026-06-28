import { ProjectsService } from "./projects.service"

describe("ProjectsService", () => {
  let service: ProjectsService
  let mockPrisma: any

  beforeEach(() => {
    mockPrisma = {
      db: {
        query: jest.fn(),
      },
    }
    service = new ProjectsService(mockPrisma)
  })

  describe("list", () => {
    it("returns mapped projects", async () => {
      mockPrisma.db.query.mockResolvedValue({
        rows: [
          { id: "1", name: "App", slug: "app", framework: "nextjs", status: "ACTIVE", domain: null, repo_url: "https://github.com/test", env_vars: null, created_at: new Date() },
        ],
      })

      const result = await service.list("user1")
      expect(result).toHaveLength(1)
      expect(result[0].name).toBe("App")
    })

    it("returns empty array when no projects", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [] })
      const result = await service.list("user1")
      expect(result).toEqual([])
    })

    it("maps snake_case to camelCase", async () => {
      mockPrisma.db.query.mockResolvedValue({
        rows: [{ id: "1", name: "Test", slug: "test", framework: null, status: "ACTIVE", domain: null, repo_url: "url", env_vars: null, created_at: new Date() }],
      })

      const result = await service.list("user1")
      expect(result[0]).toHaveProperty("repoUrl")
      expect(result[0]).toHaveProperty("envVars")
      expect(result[0]).toHaveProperty("createdAt")
    })
  })

  describe("get", () => {
    it("returns project by id", async () => {
      mockPrisma.db.query.mockResolvedValue({
        rows: [{ id: "1", name: "App", slug: "app", framework: "nextjs", status: "ACTIVE", domain: null, repo_url: null, env_vars: null, created_at: new Date() }],
      })

      const result = await service.get("1", "user1")
      expect(result).not.toBeNull()
      expect(result!.id).toBe("1")
    })

    it("returns null when not found", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [] })
      const result = await service.get("nonexistent", "user1")
      expect(result).toBeNull()
    })
  })

  describe("create", () => {
    it("creates and returns project", async () => {
      mockPrisma.db.query.mockResolvedValue({
        rows: [{ id: "1", name: "New App", slug: "new-app", framework: "nextjs", status: "ACTIVE", domain: null, repo_url: "https://github.com/test", env_vars: null, created_at: new Date() }],
      })

      const result = await service.create("user1", { name: "New App", slug: "new-app", repoUrl: "https://github.com/test", framework: "nextjs" })
      expect(result.name).toBe("New App")
    })

    it("handles optional fields", async () => {
      mockPrisma.db.query.mockResolvedValue({
        rows: [{ id: "1", name: "App", slug: "app", framework: null, status: "ACTIVE", domain: null, repo_url: null, env_vars: null, created_at: new Date() }],
      })

      const result = await service.create("user1", { name: "App", slug: "app" })
      expect(result).toBeDefined()
    })
  })

  describe("update", () => {
    it("updates project fields", async () => {
      mockPrisma.db.query.mockResolvedValue({
        rows: [{ id: "1", name: "App", slug: "app", framework: null, status: "ACTIVE", domain: "new.domain.com", repo_url: null, env_vars: null, created_at: new Date() }],
      })

      const result = await service.update("1", "user1", { domain: "new.domain.com" })
      expect(result).toBeDefined()
    })

    it("returns current state when no fields to update", async () => {
      mockPrisma.db.query.mockResolvedValue({
        rows: [{ id: "1", name: "App", slug: "app", framework: null, status: "ACTIVE", domain: null, repo_url: null, env_vars: null, created_at: new Date() }],
      })

      const result = await service.update("1", "user1", {})
      expect(result).toBeDefined()
    })
  })
})
