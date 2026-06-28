import { DatabasesService } from "./databases.service"

jest.mock("child_process", () => ({
  execSync: jest.fn(),
}))

describe("DatabasesService", () => {
  let service: DatabasesService
  let mockPrisma: any

  beforeEach(() => {
    mockPrisma = {
      db: {
        query: jest.fn(),
      },
    }
    service = new DatabasesService(mockPrisma)
  })

  describe("list", () => {
    it("returns databases for user", async () => {
      mockPrisma.db.query.mockResolvedValue({
        rows: [{ id: "1", name: "mydb", url: "postgresql://...", project_id: "p1", created_at: new Date() }],
      })
      const result = await service.list("user1")
      expect(result).toHaveLength(1)
      expect(result[0].name).toBe("mydb")
    })

    it("returns empty array when no databases", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [] })
      const result = await service.list("user1")
      expect(result).toEqual([])
    })
  })

  describe("get", () => {
    it("returns database by id", async () => {
      mockPrisma.db.query.mockResolvedValue({
        rows: [{ id: "1", name: "mydb", url: "postgresql://...", project_id: "p1", created_at: new Date() }],
      })
      const result = await service.get("1", "user1")
      expect(result).not.toBeNull()
      expect(result!.name).toBe("mydb")
    })

    it("returns null when not found", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [] })
      const result = await service.get("nonexistent", "user1")
      expect(result).toBeNull()
    })
  })

  describe("delete", () => {
    it("deletes database", async () => {
      mockPrisma.db.query
        .mockResolvedValueOnce({
          rows: [{ id: "1", name: "mydb", url: "postgresql://...", project_id: "p1", created_at: new Date() }],
        })
        .mockResolvedValueOnce({ rows: [] })

      const result = await service.delete("1", "user1")
      expect(result.success).toBe(true)
    })

    it("throws when database not found", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [] })
      await expect(service.delete("nonexistent", "user1")).rejects.toThrow("Banco não encontrado")
    })
  })
})
