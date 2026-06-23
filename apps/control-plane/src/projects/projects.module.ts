import { Module } from "@nestjs/common"
import { ProjectsController } from "./projects.controller"
import { ProjectsService } from "./projects.service"
import { DeploymentsService } from "../deployments/deployments.service"

@Module({
  controllers: [ProjectsController],
  providers: [ProjectsService, DeploymentsService],
  exports: [ProjectsService, DeploymentsService],
})
export class ProjectsModule {}
