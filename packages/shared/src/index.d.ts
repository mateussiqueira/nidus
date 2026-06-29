export type Framework = "nextjs" | "vite" | "vaden" | "express" | "fastify" | "flask" | "django" | "generic";
export type Runtime = "node" | "python" | "go" | "dart" | "rust" | "deno";
export type ProjectStatus = "active" | "building" | "deploying" | "failed" | "paused";
export type DeploymentStatus = "pending" | "building" | "deploying" | "success" | "failed";
export interface ProjectConfig {
    name: string;
    slug: string;
    framework: Framework;
    buildCommand?: string;
    outputDir?: string;
    startCommand?: string;
    env?: Record<string, string>;
}
export interface DeploymentResult {
    success: boolean;
    url?: string;
    logs?: string;
    error?: string;
}
export declare function detectFramework(files: string[]): Framework;
//# sourceMappingURL=index.d.ts.map