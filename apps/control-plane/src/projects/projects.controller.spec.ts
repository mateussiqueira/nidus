import { ProjectsController } from "./projects.controller"

describe("ProjectsController", () => {
  let controller: ProjectsController
  let mockProjects: any
  let mockDeployments: any

  beforeEach(() => {
    mockProjects = {
      list: jest.fn(),
      get: jest.fn(),
      create: jest.fn(),
      update: jest.fn(),
    }
    mockDeployments = {
      listByProject: jest.fn(),
      metrics: jest.fn(),
      deploy: jest.fn(),
    }
    controller = new ProjectsController(mockProjects, mockDeployments)
  })

  describe("list", () => {
    it("returns user projects", async () => {
      mockProjects.list.mockResolvedValue([{ id: "1", name: "App" }])
      const result = await controller.list({ user: { sub: "user1" } })
      expect(result).toHaveLength(1)
      expect(mockProjects.list).toHaveBeenCalledWith("user1")
    })
  })

  describe("get", () => {
    it("returns project by id", async () => {
      mockProjects.get.mockResolvedValue({ id: "1", name: "App" })
      const result = await controller.get("1", { user: { sub: "user1" } })
      expect(result).not.toBeNull()
      expect(result!.id).toBe("1")
    })

    it("returns null for non-existent project", async () => {
      mockProjects.get.mockResolvedValue(null)
      const result = await controller.get("nonexistent", { user: { sub: "user1" } })
      expect(result).toBeNull()
    })
  })

  describe("create", () => {
    it("creates project with auto-generated slug", async () => {
      mockProjects.create.mockResolvedValue({ id: "1", name: "My App", slug: "my-app" })
      const result = await controller.create({ user: { sub: "user1" } }, { name: "My App" })
      expect(result.slug).toBe("my-app")
    })

    it("uses provided slug", async () => {
      mockProjects.create.mockResolvedValue({ id: "1", name: "App", slug: "custom-slug" })
      const result = await controller.create({ user: { sub: "user1" } }, { name: "App", slug: "custom-slug" })
      expect(result.slug).toBe("custom-slug")
    })
  })

  describe("update", () => {
    it("updates project", async () => {
      mockProjects.update.mockResolvedValue({ id: "1", domain: "new.com" })
      const result = await controller.update("1", { user: { sub: "user1" } }, { domain: "new.com" })
      expect(result).not.toBeNull()
      expect(result!.domain).toBe("new.com")
    })
  })

  describe("listDeployments", () => {
    it("returns deployments for project", async () => {
      mockProjects.get.mockResolvedValue({ id: "1" })
      mockDeployments.listByProject.mockResolvedValue([{ id: "d1", status: "success" }])
      const result = await controller.listDeployments("1", { user: { sub: "user1" } })
      expect(result).toHaveLength(1)
    })

    it("returns empty array for non-existent project", async () => {
      mockProjects.get.mockResolvedValue(null)
      const result = await controller.listDeployments("nonexistent", { user: { sub: "user1" } })
      expect(result).toEqual([])
    })
  })

  describe("metrics", () => {
    it("returns metrics for project", async () => {
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
    it("deploys project", async () => {
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
})
