import { Controller, Get, Post, Param, Req, UseGuards } from "@nestjs/common"
import { DeploymentsService } from "./deployments.service"
import { ProjectsService } from "../projects/projects.service"
import { JwtGuard } from "../auth/jwt.guard"

@UseGuards(JwtGuard)
@Controller("api/projects/:projectId")
export class DeploymentsController {
  constructor(
    private readonly deployments: DeploymentsService,
    private readonly projects: ProjectsService,
  ) {}

  @Get("deployments")
  async list(@Param("projectId") projectId: string, @Req() req: any) {
    const project = await this.projects.get(projectId, req.user.sub)
    if (!project) return []
    return this.deployments.listByProject(projectId)
  }

  @Get("metrics")
  async metrics(@Param("projectId") projectId: string, @Req() req: any) {
    const project = await this.projects.get(projectId, req.user.sub)
    if (!project) return { status: "not_found" }
    return this.deployments.metrics(project.slug)
  }

  @Post("deploy")
  async deploy(@Param("projectId") projectId: string, @Req() req: any) {
    const project = await this.projects.get(projectId, req.user.sub)
    if (!project) throw new Error("Projeto não encontrado")
    return this.deployments.deploy(projectId)
  }
}
