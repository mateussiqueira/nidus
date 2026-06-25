import { Injectable, Logger, OnModuleDestroy } from "@nestjs/common"
import Redis from "ioredis"

const DEFAULT_TTL = 30

@Injectable()
export class RedisCacheService implements OnModuleDestroy {
  private readonly logger = new Logger(RedisCacheService.name)
  private readonly redis: Redis
  private readonly prefix = "nidus:"

  constructor() {
    this.redis = new Redis(process.env.REDIS_URL || "redis://localhost:6379", {
      maxRetriesPerRequest: 3,
      retryStrategy(times) {
        return Math.min(times * 50, 2000)
      },
      lazyConnect: true,
    })

    this.redis.on("error", (err) => {
      this.logger.error(`Redis error: ${err.message}`)
    })

    this.redis.connect().catch(() => {
      this.logger.warn("Redis not available, falling back to memory cache")
    })
  }

  async onModuleDestroy() {
    await this.redis.quit()
  }

  private key(pattern: string): string {
    return `${this.prefix}${pattern}`
  }

  async get<T>(key: string): Promise<T | null> {
    try {
      const data = await this.redis.get(this.key(key))
      if (!data) return null
      return JSON.parse(data) as T
    } catch {
      return null
    }
  }

  async set(key: string, value: unknown, ttl = DEFAULT_TTL): Promise<void> {
    try {
      await this.redis.setex(this.key(key), ttl, JSON.stringify(value))
    } catch (err) {
      this.logger.error(`Redis set error: ${err}`)
    }
  }

  async del(pattern: string): Promise<void> {
    try {
      const keys = await this.redis.keys(this.key(`${pattern}*`))
      if (keys.length > 0) {
        await this.redis.del(...keys)
      }
    } catch (err) {
      this.logger.error(`Redis del error: ${err}`)
    }
  }

  async invalidateProject(projectId: string): Promise<void> {
    await this.del(`project:${projectId}`)
    await this.del(`deployments:${projectId}`)
  }

  async invalidateUser(userId: string): Promise<void> {
    await this.del(`projects:${userId}`)
  }

  async getOrSet<T>(key: string, fetcher: () => Promise<T>, ttl = DEFAULT_TTL): Promise<T> {
    const cached = await this.get<T>(key)
    if (cached !== null) return cached

    const data = await fetcher()
    await this.set(key, data, ttl)
    return data
  }

  async healthCheck(): Promise<boolean> {
    try {
      const result = await this.redis.ping()
      return result === "PONG"
    } catch {
      return false
    }
  }
}
