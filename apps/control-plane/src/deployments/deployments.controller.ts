import { Controller, Get, Post, Param, Req, UseGuards, Query } from "@nestjs/common"
import { ApiTags, ApiOperation, ApiBearerAuth, ApiQuery } from "@nestjs/swagger"
import { DeploymentsService } from "./deployments.service"
import { ProjectsService } from "../projects/projects.service"
import { JwtGuard } from "../auth/jwt.guard"

@ApiTags("deployments")
@ApiBearerAuth()
@UseGuards(JwtGuard)
@Controller("api/projects/:projectId")
export class DeploymentsController {
  constructor(
    private readonly deployments: DeploymentsService,
    private readonly projects: ProjectsService,
  ) {}

  @Get("deployments")
  @ApiOperation({ summary: "List deployments for a project" })
  async list(@Param("projectId") projectId: string, @Req() req: any) {
    const project = await this.projects.get(projectId, req.user.sub)
    if (!project) return []
    return this.deployments.listByProject(projectId)
  }

  @Get("previews")
  @ApiOperation({ summary: "List preview deployments" })
  async listPreviews(@Param("projectId") projectId: string, @Req() req: any) {
    const project = await this.projects.get(projectId, req.user.sub)
    if (!project) return []
    return this.deployments.listPreviews(projectId)
  }

  @Get("metrics")
  @ApiOperation({ summary: "Get container metrics (CPU, RAM)" })
  @ApiQuery({ name: "branch", required: false })
  async metrics(
    @Param("projectId") projectId: string,
    @Req() req: any,
    @Query("branch") branch?: string,
  ) {
    const project = await this.projects.get(projectId, req.user.sub)
    if (!project) return { status: "not_found" }
    return this.deployments.metrics(project.slug, branch)
  }

  @Post("deploy")
  @ApiOperation({ summary: "Trigger a new deployment" })
  @ApiQuery({ name: "branch", required: false, default: "main" })
  async deploy(
    @Param("projectId") projectId: string,
    @Req() req: any,
    @Query("branch") branch?: string,
  ) {
    const project = await this.projects.get(projectId, req.user.sub)
    if (!project) throw new Error("Projeto não encontrado")
    return this.deployments.deploy(projectId, branch || "main")
  }

  @Get("deployments/:deploymentId")
  @ApiOperation({ summary: "Get deployment status by ID" })
  async getDeploymentStatus(
    @Param("projectId") projectId: string,
    @Param("deploymentId") deploymentId: string,
    @Req() req: any,
  ) {
    const project = await this.projects.get(projectId, req.user.sub)
    if (!project) return null
    return this.deployments.getJobStatus(deploymentId)
  }

  @Get("deployments/:deploymentId/logs")
  @ApiOperation({ summary: "Get deployment logs" })
  async getDeploymentLogs(
    @Param("projectId") projectId: string,
    @Param("deploymentId") deploymentId: string,
    @Req() req: any,
  ) {
    const project = await this.projects.get(projectId, req.user.sub)
    if (!project) return null
    return this.deployments.getLogs(deploymentId)
  }
}
