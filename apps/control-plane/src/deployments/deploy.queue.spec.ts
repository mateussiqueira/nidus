import { sanitizeBranch, sanitizeShell, detectFramework, generateDockerfile, getExposedPort } from "./deploy.queue"

describe("deploy.queue - Pure Functions", () => {
  describe("sanitizeBranch", () => {
    it("lowercases branch names", () => {
      expect(sanitizeBranch("MAIN")).toBe("main")
      expect(sanitizeBranch("Feature/Test")).toBe("feature-test")
    })

    it("replaces invalid characters with dashes", () => {
      expect(sanitizeBranch("feature@#$%")).toBe("feature----")
      expect(sanitizeBranch("branch with spaces")).toBe("branch-with-spaces")
    })

    it("truncates to 50 characters", () => {
      const long = "a".repeat(100)
      expect(sanitizeBranch(long)).toHaveLength(50)
    })

    it("allows dots, dashes, and underscores", () => {
      expect(sanitizeBranch("feature-1.0_test")).toBe("feature-1.0_test")
    })

    it("handles empty string", () => {
      expect(sanitizeBranch("")).toBe("")
    })
  })

  describe("sanitizeShell", () => {
    it("removes shell-dangerous characters", () => {
      expect(sanitizeShell("hello; rm -rf /")).toBe("hellorm-rf/")
      expect(sanitizeShell("test`whoami`")).toBe("testwhoami")
    })

    it("allows safe characters", () => {
      expect(sanitizeShell("https://github.com/user/repo.git")).toBe("https//github.com/user/repo.git")
      expect(sanitizeShell("file:///tmp/test")).toBe("file///tmp/test")
    })

    it("removes spaces", () => {
      expect(sanitizeShell("hello world")).toBe("helloworld")
    })
  })

  describe("detectFramework", () => {
    it("returns 'static' for unknown directory", () => {
      expect(detectFramework("/nonexistent")).toBe("static")
    })
  })

  describe("generateDockerfile", () => {
    it("generates Dockerfile for nextjs", () => {
      const df = generateDockerfile("nextjs")
      expect(df).toContain("FROM node:20-alpine")
      expect(df).toContain("npm run build")
      expect(df).toContain("EXPOSE 3000")
    })

    it("generates Dockerfile for nuxt", () => {
      const df = generateDockerfile("nuxt")
      expect(df).toContain("FROM node:20-alpine")
      expect(df).toContain("npm run build")
    })

    it("generates Dockerfile for vite", () => {
      const df = generateDockerfile("vite")
      expect(df).toContain("FROM node:20-alpine")
      expect(df).toContain("npm run build")
      expect(df).toContain("nginx")
    })

    it("generates Dockerfile for angular", () => {
      const df = generateDockerfile("angular")
      expect(df).toContain("npm run build")
      expect(df).toContain("--configuration=production")
    })

    it("generates static Dockerfile for unknown framework", () => {
      const df = generateDockerfile("unknown")
      expect(df).toContain("FROM nginx:alpine")
    })

    it("generates default Dockerfile for python", () => {
      const df = generateDockerfile("python")
      expect(df).toContain("FROM nginx:alpine")
    })
  })

  describe("getExposedPort", () => {
    it("returns 3000 for nextjs", () => {
      expect(getExposedPort("nextjs")).toBe(3000)
    })

    it("returns 3000 for nuxt", () => {
      expect(getExposedPort("nuxt")).toBe(3000)
    })

    it("returns 80 for other frameworks", () => {
      expect(getExposedPort("vite")).toBe(80)
      expect(getExposedPort("static")).toBe(80)
      expect(getExposedPort("python")).toBe(80)
    })
  })
})
