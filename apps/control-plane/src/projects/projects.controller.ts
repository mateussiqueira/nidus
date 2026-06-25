import { Controller, Get, Post, Patch, Body, Param, Req, UseGuards } from "@nestjs/common"
import { ProjectsService } from "./projects.service"
import { DeploymentsService } from "../deployments/deployments.service"
import { JwtGuard } from "../auth/jwt.guard"

@UseGuards(JwtGuard)
@Controller("api/projects")
export class ProjectsController {
  constructor(
    private readonly projects: ProjectsService,
    private readonly deployments: DeploymentsService,
  ) {}

  @Get()
  list(@Req() req: any) {
    return this.projects.list(req.user.sub)
  }

  @Get(":id")
  get(@Param("id") id: string, @Req() req: any) {
    return this.projects.get(id, req.user.sub)
  }

  @Post()
  create(@Req() req: any, @Body() body: { name: string; slug?: string; repoUrl?: string; framework?: string }) {
    const slug = body.slug || body.name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "")
    return this.projects.create(req.user.sub, { ...body, slug })
  }

  @Patch(":id")
  update(@Param("id") id: string, @Req() req: any, @Body() body: { envVars?: string; domain?: string; repoUrl?: string }) {
    return this.projects.update(id, req.user.sub, body)
  }

  @Get(":id/deployments")
  async listDeployments(@Param("id") id: string, @Req() req: any) {
    const project = await this.projects.get(id, req.user.sub)
    if (!project) return []
    return this.deployments.listByProject(id)
  }

  @Get(":id/metrics")
  async metrics(@Param("id") id: string, @Req() req: any) {
    const project = await this.projects.get(id, req.user.sub)
    if (!project) return { status: "not_found" }
    return this.deployments.metrics(project.slug)
  }

  @Post(":id/deploy")
  async deploy(@Param("id") id: string, @Req() req: any) {
    const project = await this.projects.get(id, req.user.sub)
    if (!project) throw new Error("Projeto não encontrado")
    return this.deployments.deploy(id)
  }
}
