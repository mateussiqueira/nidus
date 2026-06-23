import { Controller, Post, Get, Body, UseGuards, Req } from "@nestjs/common"
import { AuthService } from "./auth.service"
import { JwtGuard } from "./jwt.guard"

@Controller("api/auth")
export class AuthController {
  constructor(private readonly auth: AuthService) {}

  @Post("register")
  async register(@Body() body: { email: string; name: string; password: string }) {
    return this.auth.register(body.email, body.name, body.password)
  }

  @Post("login")
  async login(@Body() body: { email: string; password: string }) {
    return this.auth.login(body.email, body.password)
  }

  @UseGuards(JwtGuard)
  @Get("me")
  async me(@Req() req: any) {
    return this.auth.me(req.user.sub)
  }
}
