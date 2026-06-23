import { Controller, Get, Post, Body, Param, Req, UseGuards } from "@nestjs/common"
import { ProjectsService } from "./projects.service"
import { JwtGuard } from "../auth/jwt.guard"

@UseGuards(JwtGuard)
@Controller("api/projects")
export class ProjectsController {
  constructor(private readonly projects: ProjectsService) {}

  @Get()
  list(@Req() req: any) {
    return this.projects.list(req.user.sub)
  }

  @Get(":id")
  get(@Param("id") id: string, @Req() req: any) {
    return this.projects.get(id, req.user.sub)
  }

  @Post()
  create(@Req() req: any, @Body() body: { name: string; slug: string; repoUrl?: string }) {
    return this.projects.create(req.user.sub, body)
  }
}
