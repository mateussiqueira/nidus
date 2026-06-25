import { Controller, Get, Post, Delete, Body, Param, Req, UseGuards } from "@nestjs/common"
import { DatabasesService } from "./databases.service"
import { JwtGuard } from "../auth/jwt.guard"

@UseGuards(JwtGuard)
@Controller("api/databases")
export class DatabasesController {
  constructor(private readonly databases: DatabasesService) {}

  @Get()
  list(@Req() req: any) {
    return this.databases.list(req.user.sub)
  }

  @Get(":id")
  get(@Param("id") id: string, @Req() req: any) {
    return this.databases.get(id, req.user.sub)
  }

  @Post()
  create(@Req() req: any, @Body() body: { projectId: string; name: string }) {
    return this.databases.create(req.user.sub, body)
  }

  @Delete(":id")
  delete(@Param("id") id: string, @Req() req: any) {
    return this.databases.delete(id, req.user.sub)
  }
}
