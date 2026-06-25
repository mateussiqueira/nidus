import { Injectable, NestInterceptor, ExecutionContext, CallHandler } from "@nestjs/common"
import { Observable, of } from "rxjs"
import { tap, map } from "rxjs/operators"
import { RedisCacheService } from "./redis-cache.service"

@Injectable()
export class CacheInterceptor implements NestInterceptor {
  private cache = new Map<string, { data: any; expires: number }>()

  constructor(private readonly redisCache?: RedisCacheService) {}

  async intercept(context: ExecutionContext, next: CallHandler): Promise<Observable<any>> {
    const request = context.switchToHttp().getRequest()
    const response = context.switchToHttp().getResponse()

    if (request.method !== "GET") {
      return next.handle()
    }

    const cacheKey = this.generateCacheKey(request)
    const ttl = Reflect.getMetadata("cache_ttl", context.getHandler()) || 60

    const cached = this.cache.get(cacheKey)
    if (cached && Date.now() < cached.expires) {
      response.set("X-Cache", "HIT")
      return of(cached.data)
    }

    if (this.redisCache) {
      const redisCached = await this.redisCache.get(cacheKey)
      if (redisCached) {
        this.cache.set(cacheKey, { data: redisCached, expires: Date.now() + ttl * 1000 })
        response.set("X-Cache", "HIT")
        return of(redisCached)
      }
    }

    return next.handle().pipe(
      map((data) => {
        this.cache.set(cacheKey, { data, expires: Date.now() + ttl * 1000 })
        if (this.redisCache) {
          this.redisCache.set(cacheKey, data, ttl)
        }
        response.set("X-Cache", "MISS")
        return data
      })
    )
  }

  private generateCacheKey(request: any): string {
    const { method, url, query, user } = request
    const userId = user?.sub || "anonymous"
    return `${method}:${url}:${JSON.stringify(query)}:${userId}`
  }

  invalidate(pattern: string): void {
    for (const key of this.cache.keys()) {
      if (key.includes(pattern)) {
        this.cache.delete(key)
      }
    }
  }
}
