import { HealthController } from "./health.controller"

describe("HealthController", () => {
  let controller: HealthController
  let mockPrisma: any

  beforeEach(() => {
    mockPrisma = {
      db: {
        query: jest.fn(),
      },
    }
    controller = new HealthController(mockPrisma)
  })

  describe("check", () => {
    it("returns ok status when database is connected", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [{ "?column?": 1 }] })

      const result = await controller.check()
      expect(result.status).toBe("ok")
      expect(result.dbConnected).toBe(true)
      expect(result.name).toBe("nidus-control-plane")
      expect(result).toHaveProperty("version")
      expect(result).toHaveProperty("timestamp")
    })

    it("returns error status when database fails", async () => {
      mockPrisma.db.query.mockRejectedValue(new Error("Connection refused"))

      const result = await controller.check()
      expect(result.status).toBe("error")
      expect(result.message).toBe("Connection refused")
    })
  })
})
