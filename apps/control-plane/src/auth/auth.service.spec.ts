import { AuthService } from "./auth.service"
import { UnauthorizedException, ConflictException, NotFoundException } from "@nestjs/common"

describe("AuthService", () => {
  let service: AuthService
  let mockPrisma: any
  let mockJwt: any

  beforeEach(() => {
    mockPrisma = {
      db: {
        query: jest.fn(),
      },
    }
    mockJwt = {
      sign: jest.fn().mockReturnValue("mock-token"),
    }
    service = new AuthService(mockPrisma, mockJwt)
  })

  describe("register", () => {
    it("creates a new user and returns token", async () => {
      mockPrisma.db.query
        .mockResolvedValueOnce({ rows: [] }) // Check existing
        .mockResolvedValueOnce({ rows: [{ id: "1", name: "Test", email: "test@test.com" }] }) // Insert

      const result = await service.register("test@test.com", "Test", "password123")
      expect(result.token).toBe("mock-token")
      expect(result.user.email).toBe("test@test.com")
    })

    it("throws ConflictException if email already exists", async () => {
      mockPrisma.db.query.mockResolvedValueOnce({ rows: [{ id: "existing" }] })

      await expect(service.register("existing@test.com", "Test", "pass")).rejects.toThrow(ConflictException)
    })

    it("hashes password before storing", async () => {
      mockPrisma.db.query
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [{ id: "1", name: "Test", email: "test@test.com" }] })

      await service.register("test@test.com", "Test", "password123")
      const insertCall = mockPrisma.db.query.mock.calls[1]
      expect(insertCall[1][2]).not.toBe("password123")
      expect(insertCall[1][2]).toMatch(/^\$2[aby]?\$/)
    })

    it("signs JWT with user id and email", async () => {
      mockPrisma.db.query
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [{ id: "user-123", name: "Test", email: "test@test.com" }] })

      await service.register("test@test.com", "Test", "pass")
      expect(mockJwt.sign).toHaveBeenCalledWith({ sub: "user-123", email: "test@test.com" })
    })
  })

  describe("login", () => {
    it("returns token for valid credentials", async () => {
      const bcrypt = require("bcryptjs")
      const hashed = await bcrypt.hash("correct-password", 10)
      mockPrisma.db.query.mockResolvedValueOnce({
        rows: [{ id: "1", name: "Test", email: "test@test.com", password: hashed }],
      })

      const result = await service.login("test@test.com", "correct-password")
      expect(result.token).toBe("mock-token")
      expect(result.user.email).toBe("test@test.com")
    })

    it("throws UnauthorizedException for wrong email", async () => {
      mockPrisma.db.query.mockResolvedValueOnce({ rows: [] })

      await expect(service.login("wrong@test.com", "pass")).rejects.toThrow(UnauthorizedException)
    })

    it("throws UnauthorizedException for wrong password", async () => {
      const bcrypt = require("bcryptjs")
      const hashed = await bcrypt.hash("correct-password", 10)
      mockPrisma.db.query.mockResolvedValueOnce({
        rows: [{ id: "1", name: "Test", email: "test@test.com", password: hashed }],
      })

      await expect(service.login("test@test.com", "wrong-password")).rejects.toThrow(UnauthorizedException)
    })
  })

  describe("me", () => {
    it("returns user data", async () => {
      mockPrisma.db.query.mockResolvedValueOnce({
        rows: [{ id: "1", name: "Test", email: "test@test.com", avatar: null, created_at: new Date() }],
      })

      const result = await service.me("1")
      expect(result.name).toBe("Test")
    })

    it("throws NotFoundException for non-existent user", async () => {
      mockPrisma.db.query.mockResolvedValueOnce({ rows: [] })

      await expect(service.me("nonexistent")).rejects.toThrow(NotFoundException)
    })
  })
})
