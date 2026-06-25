import { Module, Global } from "@nestjs/common"
import { RedisCacheService } from "./redis-cache.service"
import { CacheService } from "./cache.service"
import { PrismaModule } from "../prisma/prisma.module"

@Global()
@Module({
  imports: [PrismaModule],
  providers: [RedisCacheService, CacheService],
  exports: [RedisCacheService, CacheService],
})
export class CacheModule {}
