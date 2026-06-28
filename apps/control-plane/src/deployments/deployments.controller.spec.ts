import { DeploymentsController } from "./deployments.controller"

describe("DeploymentsController", () => {
  let controller: DeploymentsController
  let mockDeployments: any
  let mockProjects: any

  beforeEach(() => {
    mockDeployments = {
      listByProject: jest.fn(),
      listPreviews: jest.fn(),
      metrics: jest.fn(),
      deploy: jest.fn(),
      getJobStatus: jest.fn(),
      getLogs: jest.fn(),
    }
    mockProjects = {
      get: jest.fn(),
    }
    controller = new DeploymentsController(mockDeployments, mockProjects)
  })

  describe("list", () => {
    it("returns deployments for project", async () => {
      mockProjects.get.mockResolvedValue({ id: "1" })
      mockDeployments.listByProject.mockResolvedValue([{ id: "d1" }])
      const result = await controller.list("1", { user: { sub: "user1" } })
      expect(result).toHaveLength(1)
    })

    it("returns empty for non-existent project", async () => {
      mockProjects.get.mockResolvedValue(null)
      const result = await controller.list("nonexistent", { user: { sub: "user1" } })
      expect(result).toEqual([])
    })
  })

  describe("listPreviews", () => {
    it("returns preview deployments", async () => {
      mockProjects.get.mockResolvedValue({ id: "1" })
      mockDeployments.listPreviews.mockResolvedValue([{ id: "d1", type: "preview" }])
      const result = await controller.listPreviews("1", { user: { sub: "user1" } })
      expect(result).toHaveLength(1)
    })
  })

  describe("metrics", () => {
    it("returns container metrics", async () => {
      mockProjects.get.mockResolvedValue({ id: "1", slug: "app" })
      mockDeployments.metrics.mockResolvedValue({ status: "running" })
      const result = await controller.metrics("1", { user: { sub: "user1" } })
      expect(result.status).toBe("running")
    })

    it("returns not_found for non-existent project", async () => {
      mockProjects.get.mockResolvedValue(null)
      const result = await controller.metrics("nonexistent", { user: { sub: "user1" } })
      expect(result.status).toBe("not_found")
    })
  })

  describe("deploy", () => {
    it("triggers deployment", async () => {
      mockProjects.get.mockResolvedValue({ id: "1" })
      mockDeployments.deploy.mockResolvedValue({ id: "d1", status: "queued" })
      const result = await controller.deploy("1", { user: { sub: "user1" } })
      expect(result.status).toBe("queued")
    })

    it("throws for non-existent project", async () => {
      mockProjects.get.mockResolvedValue(null)
      await expect(controller.deploy("nonexistent", { user: { sub: "user1" } })).rejects.toThrow()
    })
  })

  describe("getDeploymentStatus", () => {
    it("returns deployment status", async () => {
      mockProjects.get.mockResolvedValue({ id: "1" })
      mockDeployments.getJobStatus.mockResolvedValue({ status: "success" })
      const result = await controller.getDeploymentStatus("1", "d1", { user: { sub: "user1" } })
      expect(result).not.toBeNull()
      expect(result!.status).toBe("success")
    })

    it("returns null for non-existent project", async () => {
      mockProjects.get.mockResolvedValue(null)
      const result = await controller.getDeploymentStatus("nonexistent", "d1", { user: { sub: "user1" } })
      expect(result).toBeNull()
    })
  })

  describe("getDeploymentLogs", () => {
    it("returns deployment logs", async () => {
      mockProjects.get.mockResolvedValue({ id: "1" })
      mockDeployments.getLogs.mockResolvedValue("Build successful")
      const result = await controller.getDeploymentLogs("1", "d1", { user: { sub: "user1" } })
      expect(result).toBe("Build successful")
    })

    it("returns null for non-existent project", async () => {
      mockProjects.get.mockResolvedValue(null)
      const result = await controller.getDeploymentLogs("nonexistent", "d1", { user: { sub: "user1" } })
      expect(result).toBeNull()
    })
  })
})
