package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// ── Config ────────────────────────────────────────────────────────────

var (
	deploysDir = env("NIDUS_DEPLOYS_DIR", "/tmp/nidus-deploys")
	host       = env("NIDUS_HOST", "localhost")
	redisURL   = env("REDIS_URL", "redis://localhost:6379")
	dbURL      = env("DATABASE_URL", "postgresql://broto:broto@localhost:5432/nidus")
	workerPort = env("WORKER_PORT", "8080")
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// cleanDBURL removes Prisma-specific query params (like ?schema=public)
func cleanDBURL(url string) string {
	if idx := strings.Index(url, "?"); idx != -1 {
		return url[:idx]
	}
	return url
}

// ── Models ────────────────────────────────────────────────────────────

type DeployJob struct {
	DeploymentID  string `json:"deploymentId"`
	ProjectID     string `json:"projectId"`
	ProjectName   string `json:"projectName"`
	ProjectSlug   string `json:"projectSlug"`
	RepoURL       string `json:"repoUrl"`
	Domain        string `json:"domain"`
	Branch        string `json:"branch"`
	DeployType    string `json:"deployType"`
	ContainerName string `json:"containerName"`
	ImageTag      string `json:"imageTag"`
	IsPreview     bool   `json:"isPreview"`
	SafeBranch    string `json:"safeBranch"`
}

// ── Metrics ───────────────────────────────────────────────────────────

var (
	deploysTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nidus_deploys_total",
			Help: "Total number of deploys processed",
		},
		[]string{"status"},
	)
	deployDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nidus_deploy_duration_seconds",
			Help:    "Deploy duration in seconds",
			Buckets: []float64{10, 30, 60, 120, 300, 600},
		},
	)
	deployActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nidus_deploy_active",
			Help: "Number of deploys currently being processed",
		},
	)
)

func init() {
	prometheus.MustRegister(deploysTotal, deployDuration, deployActive)
}

// ── Helpers ───────────────────────────────────────────────────────────

func sanitizeBranch(branch string) string {
	reg := regexp.MustCompile(`[^a-z0-9\-_.]`)
	s := reg.ReplaceAllString(strings.ToLower(branch), "-")
	if len(s) > 50 {
		s = s[:50]
	}
	return s
}

func sanitizeShell(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9._\/\-]`)
	return reg.ReplaceAllString(s, "")
}

func detectFramework(repoDir string) string {
	configs := map[string]string{
		"next.config.js":   "nextjs",
		"next.config.ts":   "nextjs",
		"nuxt.config.js":   "nuxt",
		"nuxt.config.ts":   "nuxt",
		"vite.config.js":   "vite",
		"vite.config.ts":   "vite",
		"angular.json":     "angular",
		"svelte.config.js": "svelte",
		"astro.config.mjs": "astro",
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
			all := make(map[string]string)
			for k, v := range pkg.Dependencies {
				all[k] = v
			}
			for k, v := range pkg.DevDependencies {
				all[k] = v
			}
			for _, f := range []struct{ dep, fw string }{
				{"next", "nextjs"}, {"nuxt", "nuxt"}, {"vite", "vite"},
				{"@angular/core", "angular"}, {"svelte", "svelte"},
				{"astro", "astro"}, {"react", "vite"}, {"vue", "vite"},
			} {
				if _, ok := all[f.dep]; ok {
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

// ── Deploy Processor ──────────────────────────────────────────────────

type DeployProcessor struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func (dp *DeployProcessor) Process(ctx context.Context, jobJSON string) {
	var job DeployJob
	if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
		log.Printf("[error] failed to parse job: %v", err)
		return
	}

	start := time.Now()
	deployActive.Inc()
	defer deployActive.Dec()

	var logs []string
	logFn := func(msg string) {
		logs = append(logs, msg)
		log.Printf("[deploy] %s: %s", job.ProjectName, msg)
	}

	updateDB := func(status, url string) {
		logsStr := strings.Join(logs, "\n")
		if url != "" {
			dp.db.Exec(ctx, `UPDATE deployments SET status=$1, url=$2, logs=$3, finished_at=NOW() WHERE id=$4`,
				status, url, logsStr, job.DeploymentID)
		} else {
			dp.db.Exec(ctx, `UPDATE deployments SET status=$1, logs=$2, finished_at=NOW() WHERE id=$3`,
				status, logsStr, job.DeploymentID)
		}
	}

	logFn(fmt.Sprintf("🚀 Iniciando deploy de %s (%s)...", job.ProjectName, job.Branch))
	dp.db.Exec(ctx, `UPDATE deployments SET status='building', logs=$1 WHERE id=$2`,
		strings.Join(logs, "\n"), job.DeploymentID)

	// ── Git clone / fetch ──
	repoDir := filepath.Join(deploysDir, job.ProjectSlug)
	if job.RepoURL != "" {
		if _, err := os.Stat(repoDir); os.IsNotExist(err) {
			logFn("📦 Clonando repositorio (depth=1)...")
			safeURL := sanitizeShell(job.RepoURL)
			safeDir := sanitizeShell(repoDir)
			if out, err := runCmd(ctx, "git", "clone", "--depth", "1", "--single-branch", safeURL, safeDir); err != nil {
				logFn(fmt.Sprintf("❌ Error clonando: %s", out))
				updateDB("failed", "")
				deploysTotal.WithLabelValues("failed").Inc()
				return
			}
		} else {
			logFn("🔄 Actualizando repositorio...")
			runCmd(ctx, "git", "-C", repoDir, "fetch", "--all")
			safeBranch := sanitizeShell(job.Branch)
			if out, err := runCmd(ctx, "git", "-C", repoDir, "checkout", safeBranch); err != nil {
				logFn(fmt.Sprintf("❌ Error checkout: %s", out))
				updateDB("failed", "")
				deploysTotal.WithLabelValues("failed").Inc()
				return
			}
			runCmd(ctx, "git", "-C", repoDir, "pull", "origin", safeBranch)
		}
		logFn(fmt.Sprintf("✅ Branch %s actualizada", job.Branch))
	} else {
		logFn("⚠️ Sin repositorio configurado")
		os.MkdirAll(filepath.Join(repoDir, "src"), 0755)
		os.WriteFile(filepath.Join(repoDir, "src", "index.html"),
			[]byte(fmt.Sprintf("<h1>%s</h1><p>Deploy #%s (%s)</p>", job.ProjectName, job.DeploymentID[:8], job.Branch)), 0644)
		logFn("📄 Proyecto creado sin repositorio")
	}

	// ── Framework detection ──
	framework := detectFramework(repoDir)
	logFn(fmt.Sprintf("🔍 Framework detectado: %s", framework))
	exposedPort := getExposedPort(framework)

	// ── Docker build with streaming (via CLI for real-time logs) ──
	logFn("🐳 Build da imagen Docker...")
	dockerfile := generateDockerfile(framework)
	dockerfilePath := filepath.Join(repoDir, "Dockerfile.nidus")
	os.WriteFile(dockerfilePath, []byte(dockerfile), 0644)
	defer os.Remove(dockerfilePath)

	buildCmd := exec.CommandContext(ctx, "docker", "build",
		"-t", job.ImageTag,
		"-f", dockerfilePath,
		"--progress=plain",
		repoDir)

	buildStdout, _ := buildCmd.StdoutPipe()
	buildStderr, _ := buildCmd.StderrPipe()

	if err := buildCmd.Start(); err != nil {
		logFn(fmt.Sprintf("❌ Error starting build: %v", err))
		updateDB("failed", "")
		deploysTotal.WithLabelValues("failed").Inc()
		return
	}

	// Stream build output in real-time
	var buildWg sync.WaitGroup
	buildWg.Add(2)
	go func() {
		defer buildWg.Done()
		scanner := bufio.NewScanner(buildStdout)
		for scanner.Scan() {
			logFn(scanner.Text())
		}
	}()
	go func() {
		defer buildWg.Done()
		scanner := bufio.NewScanner(buildStderr)
		for scanner.Scan() {
			logFn(scanner.Text())
		}
	}()
	buildWg.Wait()

	if err := buildCmd.Wait(); err != nil {
		logFn(fmt.Sprintf("❌ Error build: %v", err))
		updateDB("failed", "")
		deploysTotal.WithLabelValues("failed").Inc()
		return
	}
	logFn("✅ Build concluido")

	// ── Remove old container ──
	logFn("🔄 Removendo container anterior...")
	runCmd(ctx, "docker", "rm", "-f", job.ContainerName)

	// ── Start new container ──
	logFn("🚀 Iniciando container...")
	if out, err := runCmd(ctx, "docker", "run", "-d",
		"--name", job.ContainerName,
		"-p", fmt.Sprintf("0:%d", exposedPort),
		"--restart", "unless-stopped",
		job.ImageTag); err != nil {
		logFn(fmt.Sprintf("❌ Error iniciando: %s", out))
		updateDB("failed", "")
		deploysTotal.WithLabelValues("failed").Inc()
		return
	}

	// ── Get mapped port ──
	portOutput, _ := runCmd(ctx, "docker", "port", job.ContainerName, fmt.Sprintf("%d", exposedPort))
	port := ""
	if lines := strings.Split(strings.TrimSpace(portOutput), "\n"); len(lines) > 0 {
		parts := strings.Split(lines[0], ":")
		if len(parts) > 1 {
			port = parts[len(parts)-1]
		}
	}

	url := job.Domain
	if url == "" || job.IsPreview {
		url = fmt.Sprintf("http://%s:%s", host, port)
	}
	logFn(fmt.Sprintf("✅ Deploy concluido em %s", url))

	updateDB("success", url)
	if !job.IsPreview {
		dp.db.Exec(ctx, "UPDATE projects SET status='ACTIVE' WHERE id=$1", job.ProjectID)
	}

	deploysTotal.WithLabelValues("success").Inc()
	deployDuration.Observe(time.Since(start).Seconds())
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ── Health Server ─────────────────────────────────────────────────────

func startHealthServer(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		dbOK := db.Ping(ctx) == nil
		redisOK := rdb.Ping(ctx).Err() == nil

		status := "ok"
		code := http.StatusOK
		if !dbOK || !redisOK {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		fmt.Fprintf(w, `{"status":"%s","db":%v,"redis":%v,"workers":%d}`,
			status, dbOK, redisOK, runtime.NumCPU())
	})

	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:         ":" + workerPort,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("[health] Listening on :%s", workerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[health] Server error: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
}

// ── Main ──────────────────────────────────────────────────────────────

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("═══════════════════════════════════════════")
	log.Println("  Nidus Deploy Worker (Go)")
	numWorkers := min(runtime.NumCPU(), 16)
	log.Printf("  Workers: %d", numWorkers)
	log.Println("═══════════════════════════════════════════")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("[shutdown] Received %s, shutting down...", sig)
		cancel()
	}()

	// PostgreSQL
	poolConfig, err := pgxpool.ParseConfig(cleanDBURL(dbURL))
	if err != nil {
		log.Fatalf("[fatal] Parse database config: %v", err)
	}
	poolConfig.MaxConns = 20
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("[fatal] Connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("[fatal] Ping database: %v", err)
	}
	log.Println("[db] Connected to PostgreSQL")

	// Redis
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("[fatal] Parse Redis URL: %v", err)
	}
	opts.PoolSize = 20
	opts.MinIdleConns = 5
	opts.MaxRetries = 3

	rdb := redis.NewClient(opts)
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[fatal] Connect to Redis: %v", err)
	}
	log.Println("[redis] Connected to Redis")

	processor := &DeployProcessor{db: db, rdb: rdb}

	// Health + metrics server
	go startHealthServer(ctx, rdb, db)

	// Worker pool
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			log.Printf("[worker-%d] Started", id)
			for {
				select {
				case <-ctx.Done():
					log.Printf("[worker-%d] Stopping", id)
					return
				default:
				}
				// BullMQ uses "bull:deploy-queue:wait" as the wait list
				result, err := rdb.BRPop(ctx, 5*time.Second, "bull:deploy-queue:wait").Result()
				if err != nil {
					if err == redis.Nil {
						continue
					}
					if ctx.Err() != nil {
						return
					}
					log.Printf("[worker-%d] Redis error: %v", id, err)
					time.Sleep(time.Second)
					continue
				}
				if len(result) >= 2 {
					jobID := result[1]
					// Get job data from BullMQ hash
					jobData, err := rdb.HGet(ctx, "bull:deploy-queue:"+jobID, "data").Result()
					if err != nil {
						log.Printf("[worker-%d] Failed to get job data for %s: %v", id, jobID, err)
						continue
					}
					log.Printf("[worker-%d] Processing job %s...", id, jobID)
					processor.Process(ctx, jobData)
					// Mark as completed in BullMQ
					rdb.HSet(ctx, "bull:deploy-queue:"+jobID, "finishedOn", time.Now().UnixMilli())
					rdb.LRem(ctx, "bull:deploy-queue:active", 0, jobID)
					rdb.ZAdd(ctx, "bull:deploy-queue:completed", redis.Z{
						Score:  float64(time.Now().UnixMilli()),
						Member: jobID,
					})
				}
			}
		}(i)
	}

	log.Printf("[main] %d workers started, waiting for jobs...", numWorkers)
	wg.Wait()
	log.Println("[main] Shutdown complete")
}
