package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// detectFramework identifies the project framework by checking config files and dependencies.
func detectFramework(repoDir string) string {
	// Dart/Vaden detection first (pubspec.yaml is unique)
	if _, err := os.Stat(filepath.Join(repoDir, "pubspec.yaml")); err == nil {
		data, _ := os.ReadFile(filepath.Join(repoDir, "pubspec.yaml"))
		if strings.Contains(string(data), "vaden") || strings.Contains(string(data), "shelf") {
			return "vaden"
		}
		if strings.Contains(string(data), "flutter") {
			return "flutter"
		}
		return "dart"
	}
	if _, err := os.Stat(filepath.Join(repoDir, "pubspec.yml")); err == nil {
		return "dart"
	}

	configs := map[string]string{
		"next.config.js":   "nextjs",
		"next.config.ts":   "nextjs",
		"next.config.mjs":  "nextjs",
		"nuxt.config.js":   "nuxt",
		"nuxt.config.ts":   "nuxt",
		"vite.config.js":   "vite",
		"vite.config.ts":   "vite",
		"angular.json":     "angular",
		"svelte.config.js": "svelte",
		"astro.config.mjs": "astro",
		"astro.config.ts":  "astro",
	}
	for cfg, fw := range configs {
		if _, err := os.Stat(filepath.Join(repoDir, cfg)); err == nil {
			return fw
		}
	}

	pkgPath := filepath.Join(repoDir, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		var pkg struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if json.Unmarshal(data, &pkg) == nil {
			all := make(map[string]string, len(pkg.Dependencies)+len(pkg.DevDependencies))
			for k, v := range pkg.Dependencies {
				all[k] = v
			}
			for k, v := range pkg.DevDependencies {
				all[k] = v
			}
			for _, f := range []struct{ dep, fw string }{
				{"next", "nextjs"}, {"nuxt", "nuxt"}, {"vite", "vite"},
				{"@angular/core", "angular"}, {"svelte", "svelte"},
				{"astro", "astro"}, {"react", "vite"}, {"vue", "vite"}, {"express", "express"},
			} {
				if _, ok := all[f.dep]; ok {
					return f.fw
				}
			}
		}
	}

	return "static"
}

// generateDockerfile creates an optimized multi-stage Dockerfile for the given framework.
func generateDockerfile(framework string) string {
	dockerfiles := map[string]string{
		"nextjs": `FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline --no-audit
COPY . .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
EXPOSE 3000
CMD ["node", "server.js"]`,
		"nuxt": `FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline --no-audit
COPY . .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/.output ./.output
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]`,
		"vite": `FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline --no-audit
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80`,
		"angular": `FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline --no-audit
COPY . .
RUN npm run build --configuration=production

FROM nginx:alpine
COPY --from=builder /app/dist/browser /usr/share/nginx/html
EXPOSE 80`,
		"svelte": `FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline --no-audit
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/build /usr/share/nginx/html
EXPOSE 80`,
		"astro": `FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline --no-audit
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80`,
		"vaden": `FROM dart:stable AS builder
WORKDIR /app
COPY pubspec.* ./
RUN dart pub get
COPY . .
RUN dart compile exe bin/server.dart -o /app/server

FROM scratch
COPY --from=builder /app/server /server
COPY --from=builder /app/public /public
EXPOSE 8080
CMD ["/server"]`,
		"dart": `FROM dart:stable AS builder
WORKDIR /app
COPY pubspec.* ./
RUN dart pub get
COPY . .
RUN dart compile exe bin/server.dart -o /app/server 2>/dev/null || \
    dart compile kernel bin/server.dart -o /app/server.dill 2>/dev/null || true

FROM dart:stable
WORKDIR /app
COPY --from=builder /app/server /app/server 2>/dev/null || true
COPY --from=builder /app/server.dill /app/server.dill 2>/dev/null || true
COPY --from=builder /app/public ./public
COPY . .
EXPOSE 8080
CMD ["dart", "run", "bin/server.dart"]`,
		"flutter": `FROM ubuntu:22.04 AS builder
RUN apt-get update && apt-get install -y curl git unzip xz-utils zip libglu1-mesa
RUN curl -sSL https://storage.googleapis.com/flutter_infra_release/releases/stable/linux/flutter_linux_3.22.0-stable.tar.xz | tar xJ -C /opt
ENV PATH="/opt/flutter/bin:${PATH}"
WORKDIR /app
COPY . .
RUN flutter build web --release

FROM nginx:alpine
COPY --from=builder /app/build/web /usr/share/nginx/html
EXPOSE 80`,
		"static": `FROM nginx:alpine
COPY . /usr/share/nginx/html
EXPOSE 80`,
	}
	if df, ok := dockerfiles[framework]; ok {
		return df
	}
	return dockerfiles["static"]
}

// getExposedPort returns the internal port used by the framework's runtime.
func getExposedPort(framework string) int {
	switch framework {
	case "nextjs", "nuxt", "express", "nodejs":
		return 3000
	case "vaden", "dart":
		return 8080
	case "flutter":
		return 80
	default:
		return 80
	}
}

// sanitizeBranch converts a git branch name to a safe string for container names.
func sanitizeBranch(branch string) string {
	reg := regexp.MustCompile(`[^a-z0-9\-_.]`)
	s := reg.ReplaceAllString(strings.ToLower(branch), "-")
	if len(s) > 50 {
		s = s[:50]
	}
	return s
}

// sanitizeShell removes potentially dangerous characters for shell commands.
func sanitizeShell(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9._\/\-:]`)
	return reg.ReplaceAllString(s, "")
}
