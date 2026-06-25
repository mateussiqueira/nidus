import { Module } from "@nestjs/common"
import { DatabasesController } from "./databases.controller"
import { DatabasesService } from "./databases.service"
import { PrismaModule } from "../prisma/prisma.module"

@Module({
  imports: [PrismaModule],
  controllers: [DatabasesController],
  providers: [DatabasesService],
  exports: [DatabasesService],
})
export class DatabasesModule {}
