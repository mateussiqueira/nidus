"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buildProject = buildProject;
exports.deployProject = deployProject;
exports.stopProject = stopProject;
const child_process_1 = require("child_process");
const fs_1 = require("fs");
const path_1 = require("path");
const DOCKER_DIR = (0, path_1.join)(process.cwd(), "..", "..", "docker");
function generateDockerfile(framework, outputDir) {
    switch (framework) {
        case "nextjs":
            return `
FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/static ./.next/static
EXPOSE 3000
CMD ["node", "server.js"]
`;
        case "vaden":
            return `
FROM dart:stable AS builder
WORKDIR /app
COPY pubspec.* ./
RUN dart pub get
COPY . .
RUN dart compile exe bin/main.dart -o /app/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/server ./
EXPOSE 8080
CMD ["./server"]
`;
        case "express":
            return `
FROM node:22-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
EXPOSE 3000
CMD ["node", "index.js"]
`;
        default:
            return `
FROM alpine:latest
WORKDIR /app
COPY . .
EXPOSE 8080
CMD ["./start.sh"]
`;
    }
}
async function buildProject(config, sourceDir) {
    try {
        const dockerfile = generateDockerfile(config.framework, config.outputDir);
        const dockerfilePath = (0, path_1.join)(sourceDir, "Dockerfile.canopy");
        (0, fs_1.writeFileSync)(dockerfilePath, dockerfile);
        const tag = `canopy/${config.slug}:latest`;
        (0, child_process_1.execSync)(`docker build -t ${tag} -f ${dockerfilePath} .`, {
            cwd: sourceDir,
            stdio: "pipe",
        });
        return { success: true, logs: `Build concluído: ${tag}` };
    }
    catch (err) {
        return { success: false, error: err.message, logs: err.stderr?.toString() };
    }
}
async function deployProject(config, port) {
    try {
        const tag = `canopy/${config.slug}:latest`;
        const containerName = `canopy-${config.slug}`;
        (0, child_process_1.execSync)(`docker rm -f ${containerName} 2>/dev/null || true`, { stdio: "pipe" });
        const container = (0, child_process_1.spawn)("docker", [
            "run", "-d",
            "--name", containerName,
            "--label", `canopy.project=${config.slug}`,
            ...(config.env ? Object.entries(config.env).flatMap(([k, v]) => ["-e", `${k}=${v}`]) : []),
            "-p", `${port}:8080`,
            "--restart", "unless-stopped",
            tag,
        ]);
        return new Promise((resolve) => {
            container.on("close", (code) => {
                if (code === 0) {
                    resolve({
                        success: true,
                        url: `http://localhost:${port}`,
                        logs: `Container ${containerName} iniciado na porta ${port}`,
                    });
                }
                else {
                    resolve({ success: false, error: `Docker run falhou com código ${code}` });
                }
            });
        });
    }
    catch (err) {
        return { success: false, error: err.message };
    }
}
async function stopProject(slug) {
    (0, child_process_1.execSync)(`docker rm -f canopy-${slug} 2>/dev/null || true`, { stdio: "pipe" });
}
//# sourceMappingURL=index.js.map