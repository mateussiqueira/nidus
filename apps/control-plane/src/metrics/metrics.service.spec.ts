import { MetricsService } from "./metrics.service"

describe("MetricsService", () => {
  let service: MetricsService

  beforeEach(() => {
    service = new MetricsService()
  })

  describe("recordRequest", () => {
    it("records request duration", () => {
      service.recordRequest(100, true)
      const metrics = service.getMetrics()
      expect(metrics.requests.total).toBe(1)
    })

    it("records success and error counts", () => {
      service.recordRequest(100, true)
      service.recordRequest(200, true)
      service.recordRequest(300, false)
      const metrics = service.getMetrics()
      expect(metrics.requests.success).toBe(2)
      expect(metrics.requests.error).toBe(1)
    })

    it("limits stored durations to 1000", () => {
      for (let i = 0; i < 1500; i++) {
        service.recordRequest(i, true)
      }
      const metrics = service.getMetrics()
      expect(metrics.requests.total).toBe(1000)
    })

    it("calculates average duration", () => {
      service.recordRequest(100, true)
      service.recordRequest(200, true)
      service.recordRequest(300, true)
      const metrics = service.getMetrics()
      expect(metrics.requests.avgDuration).toBe(200)
    })
  })

  describe("recordCacheHit/Miss", () => {
    it("records cache hits", () => {
      service.recordCacheHit()
      service.recordCacheHit()
      service.recordCacheHit()
      const metrics = service.getMetrics()
      expect(metrics.cache.hits).toBe(3)
      expect(metrics.cache.misses).toBe(0)
    })

    it("records cache misses", () => {
      service.recordCacheMiss()
      service.recordCacheMiss()
      const metrics = service.getMetrics()
      expect(metrics.cache.misses).toBe(2)
    })

    it("calculates hit rate", () => {
      service.recordCacheHit()
      service.recordCacheHit()
      service.recordCacheHit()
      service.recordCacheMiss()
      const metrics = service.getMetrics()
      expect(metrics.cache.hitRate).toBe(75)
    })

    it("returns 0 hit rate when no requests", () => {
      const metrics = service.getMetrics()
      expect(metrics.cache.hitRate).toBe(0)
    })
  })

  describe("recordDbQuery", () => {
    it("records query count and duration", () => {
      service.recordDbQuery(10)
      service.recordDbQuery(20)
      service.recordDbQuery(30)
      const metrics = service.getMetrics()
      expect(metrics.database.queries).toBe(3)
      expect(metrics.database.avgQueryTime).toBe(20)
    })
  })

  describe("percentile", () => {
    it("calculates p50 correctly", () => {
      for (let i = 1; i <= 100; i++) {
        service.recordRequest(i, true)
      }
      const metrics = service.getMetrics()
      expect(metrics.requests.p50).toBe(50)
    })

    it("calculates p95 correctly", () => {
      for (let i = 1; i <= 100; i++) {
        service.recordRequest(i, true)
      }
      const metrics = service.getMetrics()
      expect(metrics.requests.p95).toBe(95)
    })

    it("calculates p99 correctly", () => {
      for (let i = 1; i <= 100; i++) {
        service.recordRequest(i, true)
      }
      const metrics = service.getMetrics()
      expect(metrics.requests.p99).toBe(99)
    })

    it("returns 0 for empty array", () => {
      const metrics = service.getMetrics()
      expect(metrics.requests.p50).toBe(0)
      expect(metrics.requests.p95).toBe(0)
      expect(metrics.requests.p99).toBe(0)
    })
  })

  describe("getMetrics", () => {
    it("returns memory usage", () => {
      const metrics = service.getMetrics()
      expect(metrics.memory.heapUsed).toBeGreaterThan(0)
      expect(metrics.memory.heapTotal).toBeGreaterThan(0)
      expect(metrics.memory.rss).toBeGreaterThan(0)
    })

    it("returns uptime", () => {
      const metrics = service.getMetrics()
      expect(metrics.uptime).toBeGreaterThanOrEqual(0)
    })
  })

  describe("getPrometheusMetrics", () => {
    it("returns valid Prometheus format", () => {
      service.recordRequest(100, true)
      service.recordCacheHit()
      const output = service.getPrometheusMetrics()
      expect(output).toContain("# HELP nidus_requests_total")
      expect(output).toContain("# TYPE nidus_requests_total counter")
      expect(output).toContain("nidus_cache_hits_total 1")
      expect(output).toContain("nidus_uptime_seconds")
    })

    it("includes all metric types", () => {
      const output = service.getPrometheusMetrics()
      expect(output).toContain("nidus_request_duration_seconds_bucket")
      expect(output).toContain("nidus_cache_hit_rate")
      expect(output).toContain("nidus_memory_heap_used_bytes")
      expect(output).toContain("nidus_memory_rss_bytes")
    })
  })
})
