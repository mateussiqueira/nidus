import { Controller, Post, Body, Headers } from "@nestjs/common"
import { DeploymentsService } from "../deployments/deployments.service"
import { PrismaService } from "../prisma/prisma.service"
import { Logger } from "@nestjs/common"

@Controller("api/webhook")
export class WebhookController {
  private readonly logger = new Logger(WebhookController.name)

  constructor(
    private readonly deployments: DeploymentsService,
    private readonly prisma: PrismaService,
  ) {}

  @Post("github")
  async github(@Body() body: any, @Headers("x-github-event") event: string) {
    if (event === "ping") return { ok: true, msg: "pong" }
    if (event !== "push") return { ok: false, msg: `event ${event} ignored` }

    const repoUrl = body.repository?.clone_url
    const branch = body.ref?.replace("refs/heads/", "")
    if (!repoUrl || !branch) return { ok: false, msg: "missing repo_url or branch" }

    const projects = await this.prisma.db.query(
      "SELECT id, name, slug, env_vars FROM projects WHERE repo_url = $1 AND branch = $2",
      [repoUrl, branch],
    )

    const results = []
    for (const project of projects.rows) {
      this.logger.log(`Auto-deploying ${project.name} from push to ${branch}`)
      const result = await this.deployments.deploy(project.id)
      results.push({ project: project.name, slug: project.slug, status: result.status, url: result.url })
    }

    return { ok: true, deployed: results.length, results }
  }
}
