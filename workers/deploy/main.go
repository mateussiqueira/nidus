package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeployJob struct {
	DeploymentID  string `json:"deployment_id"`
	ProjectID     string `json:"project_id"`
	ProjectName   string `json:"project_name"`
	ProjectSlug   string `json:"project_slug"`
	RepoURL       string `json:"repo_url"`
	Domain        string `json:"domain"`
	Branch        string `json:"branch"`
	DeployType    string `json:"deploy_type"`
	ContainerName string `json:"container_name"`
	ImageTag      string `json:"image_tag"`
	IsPreview     bool   `json:"is_preview"`
	SafeBranch    string `json:"safe_branch"`
}

type DeployResult struct {
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
	Logs   string `json:"logs"`
	Error  string `json:"error,omitempty"`
}

var (
	deploysDir = getEnv("NIDUS_DEPLOYS_DIR", "/tmp/nidus-deploys")
	host       = getEnv("NIDUS_HOST", "localhost")
	redisURL   = getEnv("REDIS_URL", "redis://localhost:6379")
	dbURL      = getEnv("DATABASE_URL", "postgresql://nidus:nidus@localhost:5432/nidus")
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func sanitizeBranch(branch string) string {
	reg := regexp.MustCompile(`[^a-z0-9\-_.]`)
	sanitized := reg.ReplaceAllString(strings.ToLower(branch), "-")
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}
	return sanitized
}

func detectFramework(repoDir string) string {
	configs := map[string]string{
		"next.config.js":  "nextjs",
		"next.config.ts":  "nextjs",
		"nuxt.config.js":  "nuxt",
		"nuxt.config.ts":  "nuxt",
		"vite.config.js":  "vite",
		"vite.config.ts":  "vite",
		"angular.json":    "angular",
		"svelte.config.js": "svelte",
		"astro.config.mjs": "astro",
	}

	for config, framework := range configs {
		if _, err := os.Stat(filepath.Join(repoDir, config)); err == nil {
			return framework
		}
	}

	pkgPath := filepath.Join(repoDir, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		var pkg struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if json.Unmarshal(data, &pkg) == nil {
			allDeps := make(map[string]string)
			for k, v := range pkg.Dependencies {
				allDeps[k] = v
			}
			for k, v := range pkg.DevDependencies {
				allDeps[k] = v
			}
			
			frameworkOrder := []struct{ dep, fw string }{
				{"next", "nextjs"},
				{"nuxt", "nuxt"},
				{"vite", "vite"},
				{"@angular/core", "angular"},
				{"svelte", "svelte"},
				{"astro", "astro"},
				{"react", "vite"},
				{"vue", "vite"},
			}
			for _, f := range frameworkOrder {
				if _, ok := allDeps[f.dep]; ok {
					return f.fw
				}
			}
		}
	}
	return "static"
}

func generateDockerfile(framework string) string {
	dockerfiles := map[string]string{
		"nextjs": `FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline
COPY . .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public
COPY --from=builder /app/package.json ./
COPY --from=builder /app/node_modules ./node_modules
EXPOSE 3000
CMD ["npm", "start"]`,
		"nuxt": `FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline
COPY . .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app/.output ./.output
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]`,
		"vite": `FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80`,
		"angular": `FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline
COPY . .
RUN npm run build --configuration=production

FROM nginx:alpine
COPY --from=builder /app/dist/browser /usr/share/nginx/html
EXPOSE 80`,
		"svelte": `FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/build /usr/share/nginx/html
EXPOSE 80`,
		"astro": `FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --prefer-offline
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
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

func getExposedPort(framework string) int {
	if framework == "nextjs" || framework == "nuxt" {
		return 3000
	}
	return 80
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func processJob(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool, jobJSON string) {
	var job DeployJob
	if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
		log.Printf("[error] Failed to parse job: %v", err)
		return
	}

	var logs []string
	logFn := func(msg string) {
		logs = append(logs, msg)
		log.Printf("[deploy] %s: %s", job.ProjectName, msg)
	}

	updateDB := func(status, url string) {
		logsStr := strings.Join(logs, "\n")
		if url != "" {
			db.Exec(ctx, `UPDATE deployments SET status = $1, url = $2, logs = $3, finished_at = NOW() WHERE id = $4`, status, url, logsStr, job.DeploymentID)
		} else {
			db.Exec(ctx, `UPDATE deployments SET status = $1, logs = $2, finished_at = NOW() WHERE id = $3`, status, logsStr, job.DeploymentID)
		}
	}

	logFn(fmt.Sprintf("🚀 Iniciando deploy de %s (%s)...", job.ProjectName, job.Branch))
	db.Exec(ctx, `UPDATE deployments SET status = 'building', logs = $1 WHERE id = $2`, strings.Join(logs, "\n"), job.DeploymentID)

	repoDir := filepath.Join(deploysDir, job.ProjectSlug)
	if job.RepoURL != "" {
		if _, err := os.Stat(repoDir); os.IsNotExist(err) {
			logFn("📦 Clonando repositorio...")
			if output, err := runCmd(ctx, "git", "clone", job.RepoURL, repoDir); err != nil {
				logFn(fmt.Sprintf("❌ Error clonando: %s", output))
				updateDB("failed", "")
				return
			}
		} else {
			logFn("🔄 Actualizando repositorio...")
			runCmd(ctx, "git", "fetch", "--all", "-C", repoDir)
		}

		if output, err := runCmd(ctx, "git", "-C", repoDir, "checkout", job.Branch); err != nil {
			logFn(fmt.Sprintf("❌ Error checkout: %s", output))
			updateDB("failed", "")
			return
		}
		if output, err := runCmd(ctx, "git", "-C", repoDir, "pull", "origin", job.Branch); err != nil {
			logFn(fmt.Sprintf("❌ Error pull: %s", output))
			updateDB("failed", "")
			return
		}
		logFn(fmt.Sprintf("✅ Branch %s actualizada", job.Branch))
	} else {
		logFn("⚠️ Sin repositorio configurado")
		os.MkdirAll(filepath.Join(repoDir, "src"), 0755)
		os.WriteFile(filepath.Join(repoDir, "src", "index.html"), []byte(fmt.Sprintf("<h1>%s</h1><p>Deploy #%s (%s)</p>", job.ProjectName, job.DeploymentID[:8], job.Branch)), 0644)
		logFn("📄 Proyecto creado sin repositorio")
	}

	framework := detectFramework(repoDir)
	logFn(fmt.Sprintf("🔍 Framework detectado: %s", framework))
	logFn("🐳 Build da imagen Docker...")

	dockerfile := generateDockerfile(framework)
	buildCmd := exec.CommandContext(ctx, "docker", "build", "-t", job.ImageTag, "-f-", repoDir)
	buildCmd.Stdin = strings.NewReader(dockerfile)
	if output, err := buildCmd.CombinedOutput(); err != nil {
		logFn(fmt.Sprintf("❌ Error build: %s", output))
		updateDB("failed", "")
		return
	}
	logFn("✅ Build concluido")

	logFn("🔄 Removendo container anterior...")
	runCmd(ctx, "docker", "rm", "-f", job.ContainerName)
	
	logFn("🚀 Iniciando container...")
	exposedPort := getExposedPort(framework)
	if output, err := runCmd(ctx, "docker", "run", "-d", "--name", job.ContainerName, "-p", fmt.Sprintf("0:%d", exposedPort), "--restart", "unless-stopped", job.ImageTag); err != nil {
		logFn(fmt.Sprintf("❌ Error iniciando: %s", output))
		updateDB("failed", "")
		return
	}

	portOutput, _ := runCmd(ctx, "docker", "port", job.ContainerName, fmt.Sprintf("%d", exposedPort))
	port := strings.TrimSpace(strings.Split(portOutput, ":")[1])
	
	url := job.Domain
	if url == "" || job.IsPreview {
		url = fmt.Sprintf("http://%s:%s", host, port)
	}
	logFn(fmt.Sprintf("✅ Deploy concluido em %s", url))

	updateDB("success", url)
	if !job.IsPreview {
		db.Exec(ctx, "UPDATE projects SET status = 'ACTIVE' WHERE id = $1", job.ProjectID)
	}
}

type WorkerPool struct {
	workers int
	jobs    chan string
	wg      sync.WaitGroup
}

func NewWorkerPool(workers int) *WorkerPool {
	return &WorkerPool{
		workers: workers,
		jobs:    make(chan string, 100),
	}
}

func (wp *WorkerPool) Start(ctx context.Context, rdb *redis.Client, pool *pgxpool.Pool) {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go func(id int) {
			defer wp.wg.Done()
			log.Printf("[worker-%d] Started", id)
			for {
				select {
				case <-ctx.Done():
					return
				case jobJSON, ok := <-wp.jobs:
					if !ok {
						return
					}
					log.Printf("[worker-%d] Processing job", id)
					processJob(ctx, rdb, pool, jobJSON)
				}
			}
		}(i)
	}
}

func (wp *WorkerPool) Submit(job string) {
	wp.jobs <- job
}

func (wp *WorkerPool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}
	
	log.Printf("[deploy-worker] Starting Go deploy worker with %d workers...", numWorkers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("[error] Failed to parse database config: %v", err)
	}
	poolConfig.MaxConns = 20
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("[error] Failed to connect to database: %v", err)
	}
	defer pool.Close()

	opts, _ := redis.ParseURL(redisURL)
	rdb := redis.NewClient(opts)
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[error] Failed to connect to Redis: %v", err)
	}
	log.Println("[deploy-worker] Connected to Redis and PostgreSQL")

	workerPool := NewWorkerPool(numWorkers)
	workerPool.Start(ctx, rdb, pool)

	log.Println("[deploy-worker] Waiting for jobs...")

	for {
		result, err := rdb.BRPop(ctx, 0, "deploy-queue").Result()
		if err != nil {
			log.Printf("[error] Redis BRPop error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if len(result) >= 2 {
			workerPool.Submit(result[1])
		}
	}
}
