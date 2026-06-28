import { StructuredLogger, createLogger } from "./structured-logger"

describe("StructuredLogger", () => {
  let logger: StructuredLogger

  beforeEach(() => {
    logger = new StructuredLogger("TestContext")
  })

  describe("log", () => {
    it("logs message with context", () => {
      expect(() => logger.log("Test message", { action: "test" })).not.toThrow()
    })

    it("logs message without context", () => {
      expect(() => logger.log("Test message")).not.toThrow()
    })
  })

  describe("error", () => {
    it("logs error with trace", () => {
      expect(() => logger.error("Error message", "stack trace", { action: "test" })).not.toThrow()
    })
  })

  describe("warn", () => {
    it("logs warning", () => {
      expect(() => logger.warn("Warning message", { action: "test" })).not.toThrow()
    })
  })

  describe("debug", () => {
    it("logs debug message", () => {
      expect(() => logger.debug("Debug message", { action: "test" })).not.toThrow()
    })
  })

  describe("verbose", () => {
    it("logs verbose message", () => {
      expect(() => logger.verbose("Verbose message", { action: "test" })).not.toThrow()
    })
  })

  describe("logRequest", () => {
    it("logs request with INFO level for 2xx", () => {
      expect(() => logger.logRequest("GET", "/api/test", 200, 50, "req-1")).not.toThrow()
    })

    it("logs request with WARN level for 4xx", () => {
      expect(() => logger.logRequest("GET", "/api/test", 404, 10, "req-1")).not.toThrow()
    })

    it("logs request with ERROR level for 5xx", () => {
      expect(() => logger.logRequest("POST", "/api/test", 500, 100, "req-1")).not.toThrow()
    })
  })

  describe("logDeploy", () => {
    it("logs deploy action", () => {
      expect(() => logger.logDeploy("project-1", "deploy", "success", 5000)).not.toThrow()
    })
  })

  describe("logPerformance", () => {
    it("logs with INFO level for fast operations", () => {
      expect(() => logger.logPerformance("test-op", 500)).not.toThrow()
    })

    it("logs with WARN level for slow operations", () => {
      expect(() => logger.logPerformance("test-op", 1500)).not.toThrow()
    })
  })
})

describe("createLogger", () => {
  it("creates logger with context", () => {
    const logger = createLogger("MyContext")
    expect(logger).toBeInstanceOf(StructuredLogger)
  })

  it("creates logger without context", () => {
    const logger = createLogger()
    expect(logger).toBeInstanceOf(StructuredLogger)
  })
})
