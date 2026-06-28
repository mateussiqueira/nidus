import { CacheService } from "./cache.service"

describe("CacheService", () => {
  let service: CacheService
  let mockPrisma: any

  beforeEach(() => {
    mockPrisma = {
      db: {
        query: jest.fn(),
      },
    }
    service = new CacheService(mockPrisma)
  })

  describe("getProjects", () => {
    it("returns cached data on second call", async () => {
      const mockRows = [
        { id: "1", name: "test", slug: "test", status: "ACTIVE", framework: "nextjs", domain: null, created_at: new Date(), updated_at: new Date() },
      ]
      mockPrisma.db.query.mockResolvedValue({ rows: mockRows })

      const first = await service.getProjects("user1")
      const second = await service.getProjects("user1")

      expect(first).toEqual(second)
      expect(mockPrisma.db.query).toHaveBeenCalledTimes(1)
    })

    it("maps database rows correctly", async () => {
      const mockRows = [
        { id: "1", name: "My App", slug: "my-app", status: "ACTIVE", framework: "nextjs", domain: "app.test.com", created_at: new Date("2024-01-01"), updated_at: new Date("2024-01-02") },
      ]
      mockPrisma.db.query.mockResolvedValue({ rows: mockRows })

      const result = await service.getProjects("user1")
      expect(result[0]).toEqual({
        id: "1",
        name: "My App",
        slug: "my-app",
        status: "ACTIVE",
        framework: "nextjs",
        domain: "app.test.com",
        createdAt: new Date("2024-01-01"),
        updatedAt: new Date("2024-01-02"),
      })
    })

    it("returns empty array when no projects", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [] })
      const result = await service.getProjects("user1")
      expect(result).toEqual([])
    })
  })

  describe("getProject", () => {
    it("returns cached project on second call", async () => {
      const mockRow = {
        id: "1", name: "test", slug: "test", user_id: "user1", repo_url: "https://github.com/test",
        branch: "main", framework: "nextjs", status: "ACTIVE", domain: null, env_vars: null,
        created_at: new Date(), updated_at: new Date(),
      }
      mockPrisma.db.query.mockResolvedValue({ rows: [mockRow] })

      const first = await service.getProject("1", "user1")
      const second = await service.getProject("1", "user1")

      expect(first).toEqual(second)
      expect(mockPrisma.db.query).toHaveBeenCalledTimes(1)
    })

    it("returns null for non-existent project", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [] })
      const result = await service.getProject("nonexistent", "user1")
      expect(result).toBeNull()
    })
  })

  describe("getDeployments", () => {
    it("caches deployments with 10s TTL", async () => {
      const mockRows = [
        { id: "d1", status: "success", url: "http://test.com", logs: "ok", branch: "main", type: "production", created_at: new Date(), finished_at: new Date() },
      ]
      mockPrisma.db.query.mockResolvedValue({ rows: mockRows })

      const first = await service.getDeployments("project1")
      expect(first).toHaveLength(1)
      expect(mockPrisma.db.query).toHaveBeenCalledTimes(1)

      const second = await service.getDeployments("project1")
      expect(mockPrisma.db.query).toHaveBeenCalledTimes(1)
    })
  })

  describe("invalidate", () => {
    it("removes entries matching pattern", async () => {
      const mockRows = [{ id: "1", name: "test", slug: "test", status: "ACTIVE", framework: null, domain: null, created_at: new Date(), updated_at: new Date() }]
      mockPrisma.db.query.mockResolvedValue({ rows: mockRows })

      await service.getProjects("user1")
      service.invalidate("projects:user1")

      mockPrisma.db.query.mockResolvedValue({ rows: mockRows })
      await service.getProjects("user1")
      expect(mockPrisma.db.query).toHaveBeenCalledTimes(2)
    })

    it("does not remove unrelated entries", async () => {
      const mockRows = [{ id: "1", name: "test", slug: "test", status: "ACTIVE", framework: null, domain: null, created_at: new Date(), updated_at: new Date() }]
      mockPrisma.db.query.mockResolvedValue({ rows: mockRows })

      await service.getProjects("user1")
      service.invalidate("something-else")

      await service.getProjects("user1")
      expect(mockPrisma.db.query).toHaveBeenCalledTimes(1)
    })
  })
})
