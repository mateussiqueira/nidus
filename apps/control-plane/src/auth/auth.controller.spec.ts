import { AuthController } from "./auth.controller"

describe("AuthController", () => {
  let controller: AuthController
  let mockAuth: any

  beforeEach(() => {
    mockAuth = {
      register: jest.fn(),
      login: jest.fn(),
      me: jest.fn(),
    }
    controller = new AuthController(mockAuth)
  })

  describe("register", () => {
    it("calls auth.register with body data", async () => {
      mockAuth.register.mockResolvedValue({ token: "token", user: { id: "1" } })
      const result = await controller.register({ email: "test@test.com", name: "Test", password: "pass" })
      expect(mockAuth.register).toHaveBeenCalledWith("test@test.com", "Test", "pass")
      expect(result.token).toBe("token")
    })
  })

  describe("login", () => {
    it("calls auth.login with body data", async () => {
      mockAuth.login.mockResolvedValue({ token: "token", user: { id: "1" } })
      const result = await controller.login({ email: "test@test.com", password: "pass" })
      expect(mockAuth.login).toHaveBeenCalledWith("test@test.com", "pass")
      expect(result.token).toBe("token")
    })
  })

  describe("me", () => {
    it("calls auth.me with user id from request", async () => {
      mockAuth.me.mockResolvedValue({ id: "1", name: "Test" })
      const result = await controller.me({ user: { sub: "1" } })
      expect(mockAuth.me).toHaveBeenCalledWith("1")
      expect(result.name).toBe("Test")
    })
  })
})
