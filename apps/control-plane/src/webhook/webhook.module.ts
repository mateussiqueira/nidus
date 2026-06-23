import { Module } from "@nestjs/common"
import { WebhookController } from "./webhook.controller"
import { DeploymentsService } from "../deployments/deployments.service"

@Module({
  controllers: [WebhookController],
  providers: [DeploymentsService],
})
export class WebhookModule {}
