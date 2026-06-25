import { Module, Global } from "@nestjs/common"
import { CacheService } from "./cache.service"
import { PrismaModule } from "../prisma/prisma.module"

@Global()
@Module({
  imports: [PrismaModule],
  providers: [CacheService],
  exports: [CacheService],
})
export class CacheModule {}
