import { Injectable, NestMiddleware } from "@nestjs/common"
import { Request, Response, NextFunction } from "express"
import compression from "compression"

@Injectable()
export class CompressionMiddleware implements NestMiddleware {
  private compressor = compression({
    level: 6,
    threshold: 1024,
    filter: (req: Request, res: Response) => {
      if (req.headers["x-no-compression"]) {
        return false
      }
      return compression.filter(req, res)
    },
  })

  use(req: Request, res: Response, next: NextFunction) {
    this.compressor(req, res, next)
  }
}
