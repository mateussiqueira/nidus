import { Controller, Get, Header } from "@nestjs/common"
import { MetricsService } from "./metrics.service"

@Controller("api/metrics")
export class MetricsController {
  constructor(private readonly metrics: MetricsService) {}

  @Get()
  getMetrics() {
    return this.metrics.getMetrics()
  }

  @Get("prometheus")
  @Header("Content-Type", "text/plain")
  getPrometheusMetrics() {
    return this.metrics.getPrometheusMetrics()
  }
}
