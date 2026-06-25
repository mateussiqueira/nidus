import { Injectable, LoggerService, Logger } from "@nestjs/common"

export interface LogContext {
  requestId?: string
  userId?: string
  projectId?: string
  action?: string
  duration?: number
  status?: number
  error?: string
}

@Injectable()
export class StructuredLogger implements LoggerService {
  private readonly logger = new Logger("App")
  private readonly context?: string

  constructor(context?: string) {
    this.context = context
  }

  private formatMessage(level: string, message: string, context?: LogContext): string {
    const timestamp = new Date().toISOString()
    const ctx = this.context || context?.action || ""
    const meta = context ? JSON.stringify(context) : ""
    return `[${timestamp}] [${level}] [${ctx}] ${message} ${meta}`.trim()
  }

  log(message: string, context?: LogContext) {
    this.logger.log(this.formatMessage("INFO", message, context))
  }

  error(message: string, trace?: string, context?: LogContext) {
    this.logger.error(this.formatMessage("ERROR", message, { ...context, error: trace }))
  }

  warn(message: string, context?: LogContext) {
    this.logger.warn(this.formatMessage("WARN", message, context))
  }

  debug(message: string, context?: LogContext) {
    this.logger.debug(this.formatMessage("DEBUG", message, context))
  }

  verbose(message: string, context?: LogContext) {
    this.logger.verbose(this.formatMessage("VERBOSE", message, context))
  }

  logRequest(method: string, path: string, status: number, duration: number, requestId?: string) {
    const level = status >= 500 ? "ERROR" : status >= 400 ? "WARN" : "INFO"
    this.logger.log(this.formatMessage(level, `${method} ${path} ${status} ${duration}ms`, {
      requestId,
      status,
      duration,
    }))
  }

  logDeploy(projectId: string, action: string, status: string, duration?: number) {
    this.logger.log(this.formatMessage("INFO", `Deploy ${action}: ${status}`, {
      projectId,
      action,
      duration,
    }))
  }

  logPerformance(operation: string, duration: number, metadata?: Record<string, any>) {
    const level = duration > 1000 ? "WARN" : "INFO"
    this.logger.log(this.formatMessage(level, `${operation} completed in ${duration}ms`, {
      action: operation,
      duration,
      ...metadata,
    }))
  }
}

export const createLogger = (context?: string) => new StructuredLogger(context)
