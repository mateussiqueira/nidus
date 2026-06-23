import { Controller, Get, Param, Req, UseGuards } from "@nestjs/common"
import { DeploymentsService } from "./deployments.service"
import { ProjectsService } from "../projects/projects.service"
import { JwtGuard } from "../auth/jwt.guard"

@UseGuards(JwtGuard)
@Controller("api/projects/:projectId/deployments")
export class DeploymentsController {
  constructor(
    private readonly deployments: DeploymentsService,
    private readonly projects: ProjectsService,
  ) {}

  @Get()
  async list(@Param("projectId") projectId: string, @Req() req: any) {
    const project = await this.projects.get(projectId, req.user.sub)
    if (!project) return []
    return this.deployments.listByProject(projectId)
  }
}
