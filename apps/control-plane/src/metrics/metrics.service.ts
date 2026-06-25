import { Injectable, Logger } from "@nestjs/common"

export interface Metric {
  name: string
  value: number
  unit: string
  timestamp: number
  tags?: Record<string, string>
}

export interface PerformanceMetrics {
  requests: {
    total: number
    success: number
    error: number
    avgDuration: number
    p50: number
    p95: number
    p99: number
  }
  cache: {
    hits: number
    misses: number
    hitRate: number
    size: number
  }
  database: {
    connections: number
    queries: number
    avgQueryTime: number
  }
  memory: {
    heapUsed: number
    heapTotal: number
    rss: number
    external: number
  }
  uptime: number
}

@Injectable()
export class MetricsService {
  private readonly logger = new Logger(MetricsService.name)
  private metrics: Metric[] = []
  private requestDurations: number[] = []
  private cacheHits = 0
  private cacheMisses = 0
  private dbQueries = 0
  private dbQueryTimes: number[] = []
  private startTime = Date.now()

  recordRequest(duration: number, success: boolean) {
    this.requestDurations.push(duration)
    if (this.requestDurations.length > 1000) {
      this.requestDurations = this.requestDurations.slice(-1000)
    }
    
    this.metrics.push({
      name: "request_duration",
      value: duration,
      unit: "ms",
      timestamp: Date.now(),
      tags: { success: String(success) },
    })
  }

  recordCacheHit() {
    this.cacheHits++
  }

  recordCacheMiss() {
    this.cacheMisses++
  }

  recordDbQuery(duration: number) {
    this.dbQueries++
    this.dbQueryTimes.push(duration)
    if (this.dbQueryTimes.length > 1000) {
      this.dbQueryTimes = this.dbQueryTimes.slice(-1000)
    }
  }

  private percentile(arr: number[], p: number): number {
    if (arr.length === 0) return 0
    const sorted = [...arr].sort((a, b) => a - b)
    const index = Math.ceil((p / 100) * sorted.length) - 1
    return sorted[Math.max(0, index)]
  }

  private average(arr: number[]): number {
    if (arr.length === 0) return 0
    return arr.reduce((a, b) => a + b, 0) / arr.length
  }

  getMetrics(): PerformanceMetrics {
    const mem = process.memoryUsage()
    
    return {
      requests: {
        total: this.requestDurations.length,
        success: this.metrics.filter(m => m.tags?.success === "true").length,
        error: this.metrics.filter(m => m.tags?.success === "false").length,
        avgDuration: this.average(this.requestDurations),
        p50: this.percentile(this.requestDurations, 50),
        p95: this.percentile(this.requestDurations, 95),
        p99: this.percentile(this.requestDurations, 99),
      },
      cache: {
        hits: this.cacheHits,
        misses: this.cacheMisses,
        hitRate: this.cacheHits + this.cacheMisses > 0 
          ? (this.cacheHits / (this.cacheHits + this.cacheMisses)) * 100 
          : 0,
        size: 0,
      },
      database: {
        connections: 0,
        queries: this.dbQueries,
        avgQueryTime: this.average(this.dbQueryTimes),
      },
      memory: {
        heapUsed: mem.heapUsed / 1024 / 1024,
        heapTotal: mem.heapTotal / 1024 / 1024,
        rss: mem.rss / 1024 / 1024,
        external: mem.external / 1024 / 1024,
      },
      uptime: (Date.now() - this.startTime) / 1000,
    }
  }

  getPrometheusMetrics(): string {
    const metrics = this.getMetrics()
    
    return `# HELP nidus_requests_total Total number of requests
# TYPE nidus_requests_total counter
nidus_requests_total ${metrics.requests.total}

# HELP nidus_request_duration_seconds Request duration in seconds
# TYPE nidus_request_duration_seconds histogram
nidus_request_duration_seconds_bucket{le="0.1"} ${metrics.requests.p50}
nidus_request_duration_seconds_bucket{le="0.5"} ${metrics.requests.p95}
nidus_request_duration_seconds_bucket{le="1.0"} ${metrics.requests.p99}
nidus_request_duration_seconds_bucket{le="+Inf"} ${metrics.requests.total}

# HELP nidus_cache_hits_total Total cache hits
# TYPE nidus_cache_hits_total counter
nidus_cache_hits_total ${metrics.cache.hits}

# HELP nidus_cache_misses_total Total cache misses
# TYPE nidus_cache_misses_total counter
nidus_cache_misses_total ${metrics.cache.misses}

# HELP nidus_cache_hit_rate Cache hit rate
# TYPE nidus_cache_hit_rate gauge
nidus_cache_hit_rate ${metrics.cache.hitRate}

# HELP nidus_memory_heap_used_bytes Heap memory used
# TYPE nidus_memory_heap_used_bytes gauge
nidus_memory_heap_used_bytes ${metrics.memory.heapUsed * 1024 * 1024}

# HELP nidus_memory_rss_bytes Resident set size
# TYPE nidus_memory_rss_bytes gauge
nidus_memory_rss_bytes ${metrics.memory.rss * 1024 * 1024}

# HELP nidus_uptime_seconds Uptime in seconds
# TYPE nidus_uptime_seconds gauge
nidus_uptime_seconds ${metrics.uptime}
`
  }
}
