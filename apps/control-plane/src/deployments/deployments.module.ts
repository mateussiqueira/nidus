import { Module } from "@nestjs/common"
import { DeploymentsController } from "./deployments.controller"
import { DeploymentsService } from "./deployments.service"
import { ProjectsModule } from "../projects/projects.module"

@Module({
  imports: [ProjectsModule],
  controllers: [DeploymentsController],
  providers: [DeploymentsService],
})
export class DeploymentsModule {}
