import { SetMetadata } from "@nestjs/common"

export const CACHE_TTL_KEY = "cache_ttl"
export const CACHE_KEY = "cache_key"

export const Cacheable = (ttl: number = 60, key?: string) => {
  return (target: any, propertyKey?: string, descriptor?: PropertyDescriptor) => {
    if (descriptor) {
      Reflect.defineMetadata(CACHE_TTL_KEY, ttl, descriptor.value)
      if (key) {
        Reflect.defineMetadata(CACHE_KEY, key, descriptor.value)
      }
    }
    return descriptor
  }
}

export const CacheInvalidate = (pattern: string) => {
  return (target: any, propertyKey?: string, descriptor?: PropertyDescriptor) => {
    if (descriptor) {
      Reflect.defineMetadata("cache_invalidate", pattern, descriptor.value)
    }
    return descriptor
  }
}
