import type { ProjectConfig, DeploymentResult } from "@canopy/shared";
export declare function buildProject(config: ProjectConfig, sourceDir: string): Promise<DeploymentResult>;
export declare function deployProject(config: ProjectConfig, port: number): Promise<DeploymentResult>;
export declare function stopProject(slug: string): Promise<void>;
//# sourceMappingURL=index.d.ts.map