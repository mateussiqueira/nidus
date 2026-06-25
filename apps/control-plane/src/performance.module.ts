import { Module, Global, MiddlewareConsumer, NestModule } from "@nestjs/common"
import { APP_INTERCEPTOR } from "@nestjs/core"
import { CacheModule } from "./cache/cache.module"
import { CacheInterceptor } from "./cache/cache.interceptor"
import { TimingInterceptor } from "./interceptors/timing.interceptor"
import { CompressionMiddleware } from "./middleware/compression.middleware"

@Global()
@Module({
  imports: [CacheModule],
  providers: [
    {
      provide: APP_INTERCEPTOR,
      useClass: TimingInterceptor,
    },
    {
      provide: APP_INTERCEPTOR,
      useClass: CacheInterceptor,
    },
  ],
})
export class PerformanceModule implements NestModule {
  configure(consumer: MiddlewareConsumer) {
    consumer.apply(CompressionMiddleware).forRoutes("*")
  }
}
