import { Injectable, NestInterceptor, ExecutionContext, CallHandler, Logger } from "@nestjs/common"
import { Observable } from "rxjs"
import { tap } from "rxjs/operators"

@Injectable()
export class TimingInterceptor implements NestInterceptor {
  private readonly logger = new Logger("Timing")

  intercept(context: ExecutionContext, next: CallHandler): Observable<any> {
    const request = context.switchToHttp().getRequest()
    const { method, url } = request
    const requestId = request.id
    const start = Date.now()

    return next.handle().pipe(
      tap(() => {
        const duration = Date.now() - start
        const status = context.switchToHttp().getResponse().statusCode
        
        if (duration > 1000) {
          this.logger.warn(`${method} ${url} ${status} ${duration}ms [${requestId}]`)
        } else if (duration > 100) {
          this.logger.log(`${method} ${url} ${status} ${duration}ms [${requestId}]`)
        }
      })
    )
  }
}
