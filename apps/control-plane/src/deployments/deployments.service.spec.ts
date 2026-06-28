import { DeploymentsService } from "./deployments.service"

jest.mock("dockerode", () => {
  const mockContainer = {
    inspect: jest.fn().mockResolvedValue({
      State: { Status: "running", Running: true, StartedAt: new Date().toISOString(), RestartCount: 0, ExitCode: 0 },
    }),
    stats: jest.fn().mockResolvedValue({
      cpu_stats: { cpu_usage: { total_usage: 1000000 }, system_cpu_usage: 10000000 },
      memory_stats: { usage: 50 * 1024 * 1024, limit: 100 * 1024 * 1024 },
      networks: { eth0: { rx_bytes: 1000, tx_bytes: 2000 } },
    }),
  }
  
  const failingContainer = {
    inspect: jest.fn().mockRejectedValue(new Error("No such container")),
    stats: jest.fn().mockRejectedValue(new Error("No such container")),
  }
  
  return jest.fn().mockImplementation(() => ({
    getContainer: jest.fn().mockImplementation((name: string) => {
      if (name === "nidus-nonexistent") return failingContainer
      return mockContainer
    }),
  }))
})

jest.mock("./deploy.queue", () => ({
  getDeployQueue: jest.fn().mockReturnValue({
    add: jest.fn().mockResolvedValue({ id: "job-1" }),
    close: jest.fn(),
  }),
  sanitizeBranch: jest.fn((b: string) => b.toLowerCase().replace(/[^a-z0-9-]/g, "-")),
}))

describe("DeploymentsService", () => {
  let service: DeploymentsService
  let mockPrisma: any

  beforeEach(() => {
    mockPrisma = {
      db: {
        query: jest.fn(),
      },
    }
    service = new DeploymentsService(mockPrisma)
  })

  describe("listByProject", () => {
    it("returns deployments mapped correctly", async () => {
      mockPrisma.db.query.mockResolvedValue({
        rows: [
          { id: "d1", status: "success", url: "http://app.test.com", branch: "main", type: "production", created_at: new Date(), finished_at: new Date() },
        ],
      })

      const result = await service.listByProject("project1")
      expect(result).toHaveLength(1)
      expect(result[0].id).toBe("d1")
      expect(result[0].status).toBe("success")
    })

    it("returns empty array when no deployments", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [] })
      const result = await service.listByProject("project1")
      expect(result).toEqual([])
    })
  })

  describe("getLogs", () => {
    it("returns deployment logs", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [{ logs: "Build successful\nDeployed" }] })
      const result = await service.getLogs("d1")
      expect(result).toBe("Build successful\nDeployed")
    })

    it("returns empty string when no logs", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [{}] })
      const result = await service.getLogs("d1")
      expect(result).toBe("")
    })

    it("returns empty string when deployment not found", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [] })
      const result = await service.getLogs("nonexistent")
      expect(result).toBe("")
    })
  })

  describe("listPreviews", () => {
    it("returns only preview deployments", async () => {
      mockPrisma.db.query.mockResolvedValue({
        rows: [
          { id: "d1", status: "success", url: null, branch: "feature-x", type: "preview", created_at: new Date(), finished_at: null },
        ],
      })

      const result = await service.listPreviews("project1")
      expect(result).toHaveLength(1)
      expect(result[0].type).toBe("preview")
    })
  })

  describe("getJobStatus", () => {
    it("returns deployment status", async () => {
      mockPrisma.db.query.mockResolvedValue({
        rows: [{ id: "d1", status: "success", url: "http://app.test.com", branch: "main", type: "production", created_at: new Date(), finished_at: new Date() }],
      })

      const result = await service.getJobStatus("d1")
      expect(result).not.toBeNull()
      expect(result!.status).toBe("success")
    })

    it("returns null for non-existent deployment", async () => {
      mockPrisma.db.query.mockResolvedValue({ rows: [] })
      const result = await service.getJobStatus("nonexistent")
      expect(result).toBeNull()
    })
  })

  describe("deploy", () => {
    it("creates deployment and queues job", async () => {
      mockPrisma.db.query
        .mockResolvedValueOnce({
          rows: [{ id: "p1", name: "My App", slug: "my-app", repo_url: "https://github.com/test", domain: null }],
        })
        .mockResolvedValueOnce({ rows: [{ id: "d1" }] })

      const result = await service.deploy("p1", "main")
      expect(result.status).toBe("queued")
      expect(result.branch).toBe("main")
      expect(result.type).toBe("production")
    })

    it("marks as preview for non-main branches", async () => {
      mockPrisma.db.query
        .mockResolvedValueOnce({
          rows: [{ id: "p1", name: "My App", slug: "my-app", repo_url: "https://github.com/test", domain: null }],
        })
        .mockResolvedValueOnce({ rows: [{ id: "d1" }] })

      const result = await service.deploy("p1", "feature-x")
      expect(result.type).toBe("preview")
    })

    it("throws when project not found", async () => {
      mockPrisma.db.query.mockResolvedValueOnce({ rows: [] })
      await expect(service.deploy("nonexistent")).rejects.toThrow("Projeto não encontrado")
    })
  })

  describe("metrics", () => {
    it("returns container metrics", async () => {
      const result = await service.metrics("my-app")
      expect(result.status).toBe("running")
      expect(result.running).toBe(true)
    })

    it("returns stopped status on error", async () => {
      const result = await service.metrics("nonexistent")
      expect(result.status).toBe("stopped")
    })
  })
})
