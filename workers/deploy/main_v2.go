package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// ── Config ────────────────────────────────────────────────────────────

var (
	deploysDir          = env("NIDUS_DEPLOYS_DIR", "/tmp/nidus-deploys")
	host                = env("NIDUS_HOST", "localhost")
	redisURL            = env("REDIS_URL", "redis://localhost:6379")
	dbURL               = env("DATABASE_URL", "postgresql://broto:broto@localhost:5432/nidus")
	workerPort          = env("WORKER_PORT", "8081")
	maxConcurrentBuilds = envInt("MAX_CONCURRENT_BUILDS", 50)
	dockerTimeout       = envDuration("DOCKER_TIMEOUT_SECONDS", 120*time.Second)
	buildkitEnabled     = envBool("BUILDKIT_ENABLED", true)
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i := atoi(v); i >= 0 {
			return i
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v + "s"); err == nil {
			return d
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "true" || v == "1" || v == "yes"
	}
	return fallback
}

func atoi(s string) int {
	for i, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		if i == 0 && (c < '1' || c > '9') && len(s) > 1 {
			return -1
		}
	}
	result := 0
	for _, c := range s {
		result = result*10 + int(c-'0')
	}
	return result
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
	DeploymentID  string            `json:"deploymentId"`
	ProjectID     string            `json:"projectId"`
	ProjectName   string            `json:"projectName"`
	ProjectSlug   string            `json:"projectSlug"`
	RepoURL       string            `json:"repoUrl"`
	Domain        string            `json:"domain"`
	Branch        string            `json:"branch"`
	DeployType    string            `json:"deployType"`
	ContainerName string            `json:"containerName"`
	ImageTag      string            `json:"imageTag"`
	IsPreview     bool              `json:"isPreview"`
	SafeBranch    string            `json:"safeBranch"`
	EnvVars       map[string]string `json:"envVars,omitempty"`
}

// BuildStats tracks build performance metrics
type BuildStats struct {
	GitTime       time.Duration
	BuildTime     time.Duration
	ContainerTime time.Duration
	TotalTime     time.Duration
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
			Buckets: []float64{5, 10, 20, 30, 60, 120, 300, 600},
		},
	)
	deployActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nidus_deploy_active",
			Help: "Number of deploys currently being processed",
		},
	)
	buildDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nidus_build_duration_seconds",
			Help:    "Docker build duration in seconds",
			Buckets: []float64{5, 10, 20, 30, 60, 120, 300},
		},
	)
	gitDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nidus_git_duration_seconds",
			Help:    "Git operations duration in seconds",
			Buckets: []float64{1, 2, 5, 10, 20, 30},
		},
	)
	buildQueueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nidus_build_queue_size",
			Help: "Current size of the build queue",
		},
	)
	buildCacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nidus_build_cache_hits_total",
			Help: "Total number of Docker build cache hits",
		},
	)
	buildCacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nidus_build_cache_misses_total",
			Help: "Total number of Docker build cache misses",
		},
	)
)

func init() {
	prometheus.MustRegister(
		deploysTotal, deployDuration, deployActive,
		buildDuration, gitDuration, buildQueueSize,
		buildCacheHits, buildCacheMisses,
	)
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

// fetchEnvVars retrieves environment variables for a project from the database
func fetchEnvVars(ctx context.Context, db *pgxpool.Pool, projectID string) (map[string]string, error) {
	envVars := make(map[string]string)

	// Try to fetch from project_env_vars table
	rows, err := db.Query(ctx,
		`SELECT key, value FROM project_env_vars WHERE project_id = $1`,
		projectID)
	if err != nil {
		// Table might not exist, try alternative table name
		rows, err = db.Query(ctx,
			`SELECT key, value FROM environment_variables WHERE project_id = $1`,
			projectID)
		if err != nil {
			return envVars, nil // Return empty map if table doesn't exist
		}
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		envVars[key] = value
	}

	return envVars, nil
}

// maskEnvVar masks sensitive values for logging
func maskEnvVar(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}

func sanitizeShell(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9._\/\-:]`)
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
RUN npm ci --prefer-offline --no-audit
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
RUN npm ci --prefer-offline --no-audit
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
	db           *pgxpool.Pool
	rdb          *redis.Client
	dockerClient *client.Client
	buildQueue   chan *DeployJob
	semaphore    chan struct{}
	activeCount  atomic.Int32
}

func (dp *DeployProcessor) updateDeploymentStatus(ctx context.Context, deploymentID, status, url, logs string) {
	if dp.db != nil {
		// PostgreSQL mode - direct DB access
		if url != "" {
			dp.db.Exec(ctx, `UPDATE deployments SET status=$1, url=$2, logs=$3, finished_at=NOW() WHERE id=$4`,
				status, url, logs, deploymentID)
		} else {
			dp.db.Exec(ctx, `UPDATE deployments SET status=$1, logs=$2, finished_at=NOW() WHERE id=$3`,
				status, logs, deploymentID)
		}
	} else {
		// SQLite mode - log only
		log.Printf("[deploy] Status update: %s -> %s", deploymentID, status)
	}
}

// processGit handles git clone/fetch operations efficiently
func (dp *DeployProcessor) processGit(ctx context.Context, job *DeployJob, repoDir string, logFn func(string)) error {
	gitStart := time.Now()
	defer func() { gitDuration.Observe(time.Since(gitStart).Seconds()) }()

	if job.RepoURL == "" {
		logFn("⚠️ No repository configured")
		os.MkdirAll(filepath.Join(repoDir, "src"), 0755)
		os.WriteFile(filepath.Join(repoDir, "src", "index.html"),
			[]byte(fmt.Sprintf("<h1>%s</h1><p>Deploy #%s (%s)</p>", job.ProjectName, job.DeploymentID[:8], job.Branch)), 0644)
		logFn("📄 Project created without repository")
		return nil
	}

	// Check if directory exists and has content
	hasContent := false
	if info, err := os.Stat(repoDir); !os.IsNotExist(err) && info.IsDir() {
		entries, _ := os.ReadDir(repoDir)
		for _, entry := range entries {
			if entry.Name() != ".git" {
				hasContent = true
				break
			}
		}
	}

	if !hasContent {
		logFn("📦 Cloning repository...")
		if _, err := runCmd(ctx, "git", "clone", "--depth", "1", "--single-branch", "-b", job.Branch, job.RepoURL, repoDir); err != nil {
			logFn(fmt.Sprintf("❌ Git clone failed: %v", err))
			return err
		}
	} else {
		logFn("🔄 Updating repository...")
		if _, err := runCmd(ctx, "git", "-C", repoDir, "fetch", "--depth", "1", "origin", job.Branch); err != nil {
			logFn(fmt.Sprintf("⚠️ Git fetch failed: %v", err))
		}
		if _, err := runCmd(ctx, "git", "-C", repoDir, "checkout", job.Branch); err != nil {
			logFn(fmt.Sprintf("❌ Git checkout failed: %v", err))
			return err
		}
	}
	logFn(fmt.Sprintf("✅ Branch %s ready", job.Branch))
	return nil
}

// processBuild handles Docker build with BuildKit and caching
func (dp *DeployProcessor) processBuild(ctx context.Context, job *DeployJob, repoDir string, envVars map[string]string, logFn func(string)) error {
	buildStart := time.Now()
	defer func() { buildDuration.Observe(time.Since(buildStart).Seconds()) }()

	framework := detectFramework(repoDir)
	logFn(fmt.Sprintf("🔍 Framework detected: %s", framework))

	// Check for custom Dockerfile
	customDockerfile := filepath.Join(repoDir, "Dockerfile")
	customNidusDockerfile := filepath.Join(repoDir, "Dockerfile.nidus")
	useCustomDockerfile := false

	if _, err := os.Stat(customDockerfile); err == nil {
		logFn("📄 Using custom Dockerfile")
		useCustomDockerfile = true
	} else if _, err := os.Stat(customNidusDockerfile); err == nil {
		logFn("📄 Using custom Dockerfile.nidus")
		useCustomDockerfile = true
	}

	logFn("🐳 Building Docker image...")

	var buildArgs []string
	buildArgs = append(buildArgs, "build")
	buildArgs = append(buildArgs, "-t", job.ImageTag)

	// Enable BuildKit if available
	if buildkitEnabled {
		buildArgs = append(buildArgs, "--progress=plain")
	}

	if useCustomDockerfile {
		if _, err := os.Stat(customDockerfile); err == nil {
			buildArgs = append(buildArgs, "-f", customDockerfile)
		} else {
			buildArgs = append(buildArgs, "-f", customNidusDockerfile)
		}
	} else {
		dockerfile := generateDockerfile(framework)
		dockerfilePath := filepath.Join(repoDir, "Dockerfile.nidus")
		os.WriteFile(dockerfilePath, []byte(dockerfile), 0644)
		defer os.Remove(dockerfilePath)
		buildArgs = append(buildArgs, "-f", dockerfilePath)
	}

	// Add build args for environment variables (with limited size)
	for key, value := range envVars {
		if len(value) < 1000 { // Limit env var size
			buildArgs = append(buildArgs, "--build-arg", fmt.Sprintf("%s=%s", key, value))
		}
	}

	buildArgs = append(buildArgs, repoDir)

	logFn(fmt.Sprintf("🔨 Command: docker %s", strings.Join(buildArgs, " ")))

	buildCmd := exec.CommandContext(ctx, "docker", buildArgs...)
	buildStdout, _ := buildCmd.StdoutPipe()
	buildStderr, _ := buildCmd.StderrPipe()

	if err := buildCmd.Start(); err != nil {
		logFn(fmt.Sprintf("❌ Failed to start build: %v", err))
		return err
	}

	// Stream build output with cache hit detection
	var buildWg sync.WaitGroup
	buildWg.Add(2)

	buildCtx, buildCancel := context.WithTimeout(ctx, dockerTimeout)
	defer buildCancel()

	go func() {
		defer buildWg.Done()
		scanner := bufio.NewScanner(buildStdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB buffer
		for scanner.Scan() {
			if buildCtx.Err() != nil {
				return
			}
			line := scanner.Text()
			logFn(line)
			if strings.Contains(line, "CACHED") || strings.Contains(line, "Using cache") {
				buildCacheHits.Inc()
			}
		}
	}()

	go func() {
		defer buildWg.Done()
		scanner := bufio.NewScanner(buildStderr)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			if buildCtx.Err() != nil {
				return
			}
			logFn(scanner.Text())
		}
	}()

	buildWg.Wait()

	if buildCtx.Err() != nil {
		logFn("❌ Build timeout")
		buildCmd.Process.Kill()
		return fmt.Errorf("build timeout after %v", dockerTimeout)
	}

	if err := buildCmd.Wait(); err != nil {
		logFn(fmt.Sprintf("❌ Build failed: %v", err))
		buildCacheMisses.Inc()
		return err
	}

	logFn("✅ Build completed")
	return nil
}

// processContainer handles container creation and health checks
func (dp *DeployProcessor) processContainer(ctx context.Context, job *DeployJob, envVars map[string]string, framework string, logFn func(string)) (string, error) {
	containerStart := time.Now()
	defer func() {
		logFn(fmt.Sprintf("⏱️  Container setup took %v", time.Since(containerStart)))
	}()

	exposedPort := getExposedPort(framework)

	// Remove old container
	logFn("🔄 Removing previous container...")
	runCmd(ctx, "docker", "rm", "-f", job.ContainerName)

	// Create container
	logFn("🚀 Starting container...")
	var runArgs []string
	runArgs = append(runArgs,
		"run", "-d",
		"--name", job.ContainerName,
		"-p", fmt.Sprintf("0:%d", exposedPort),
		"--restart", "unless-stopped",
		"-m", "512m", // Memory limit
		"--cpus", "1.0", // CPU limit
	)

	// Add environment variables
	for key, value := range envVars {
		runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", key, value))
		logFn(fmt.Sprintf("🔧 ENV: %s=%s", key, maskEnvVar(value)))
	}

	runArgs = append(runArgs, job.ImageTag)

	if out, err := runCmd(ctx, "docker", runArgs...); err != nil {
		logFn(fmt.Sprintf("❌ Failed to start container: %s", out))
		return "", err
	}

	// Get mapped port
	portOutput, _ := runCmd(ctx, "docker", "port", job.ContainerName, fmt.Sprintf("%d", exposedPort))
	port := ""
	if lines := strings.Split(strings.TrimSpace(portOutput), "\n"); len(lines) > 0 {
		parts := strings.Split(lines[0], ":")
		if len(parts) > 1 {
			port = parts[len(parts)-1]
		}
	}

	// Health check
	logFn("🏥 Checking container health...")
	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Second)
		inspectOutput, _ := runCmd(ctx, "docker", "inspect", "--format", "{{.State.Running}}", job.ContainerName)
		if strings.TrimSpace(inspectOutput) == "true" {
			logFn("✅ Container is running")
			return port, nil
		}
		logFn(fmt.Sprintf("⏳ Waiting for container... (%d/15)", i+1))
	}

	logFn("❌ Container failed to start")
	containerLogs, _ := runCmd(ctx, "docker", "logs", "--tail", "50", job.ContainerName)
	if containerLogs != "" {
		logFn(fmt.Sprintf("📋 Container logs:\n%s", containerLogs))
	}
	return "", fmt.Errorf("container health check failed")
}

func (dp *DeployProcessor) Process(ctx context.Context, jobJSON string) {
	var job DeployJob
	if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
		log.Printf("[error] Failed to parse job: %v", err)
		return
	}

	start := time.Now()
	deployActive.Inc()
	dp.activeCount.Add(1)
	defer func() {
		deployActive.Dec()
		dp.activeCount.Add(-1)
	}()

	var logs []string
	logMutex := &sync.Mutex{}
	logFn := func(msg string) {
		logMutex.Lock()
		logs = append(logs, msg)
		logMutex.Unlock()
		log.Printf("[deploy] %s: %s", job.ProjectName, msg)
	}

	updateDB := func(status, url string) {
		logsStr := strings.Join(logs, "\n")
		dp.updateDeploymentStatus(ctx, job.DeploymentID, status, url, logsStr)
	}

	logFn(fmt.Sprintf("🚀 Starting deploy of %s (%s)...", job.ProjectName, job.Branch))
	if dp.db != nil {
		dp.db.Exec(ctx, `UPDATE deployments SET status='building', logs=$1 WHERE id=$2`,
			strings.Join(logs, "\n"), job.DeploymentID)
	}

	// Fetch environment variables
	envVars, err := fetchEnvVars(ctx, dp.db, job.ProjectID)
	if err != nil {
		logFn(fmt.Sprintf("⚠️ Warning: Could not fetch env vars: %v", err))
	}
	if job.EnvVars != nil {
		for k, v := range job.EnvVars {
			envVars[k] = v
		}
	}
	logFn(fmt.Sprintf("🔧 Environment variables: %d loaded", len(envVars)))

	// Process git
	repoDir := filepath.Join(deploysDir, job.ProjectSlug)
	if err := dp.processGit(ctx, &job, repoDir, logFn); err != nil {
		logFn(fmt.Sprintf("❌ Git processing failed: %v", err))
		updateDB("failed", "")
		deploysTotal.WithLabelValues("failed").Inc()
		return
	}

	framework := detectFramework(repoDir)

	// Process build
	if err := dp.processBuild(ctx, &job, repoDir, envVars, logFn); err != nil {
		logFn(fmt.Sprintf("❌ Build failed: %v", err))
		updateDB("failed", "")
		deploysTotal.WithLabelValues("failed").Inc()
		return
	}

	// Process container
	port, err := dp.processContainer(ctx, &job, envVars, framework, logFn)
	if err != nil {
		logFn(fmt.Sprintf("❌ Container startup failed: %v", err))
		updateDB("failed", "")
		deploysTotal.WithLabelValues("failed").Inc()
		return
	}

	url := job.Domain
	if url == "" || job.IsPreview {
		url = fmt.Sprintf("http://%s:%s", host, port)
	}

	logFn(fmt.Sprintf("✅ Deploy completed in %v", time.Since(start)))
	logFn(fmt.Sprintf("🔗 URL: %s", url))

	updateDB("success", url)
	if !job.IsPreview && dp.db != nil {
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

// ── Worker Pool ───────────────────────────────────────────────────────

type WorkerPool struct {
	processor    *DeployProcessor
	numWorkers   int
	jobQueue     chan *DeployJob
	redisClient  *redis.Client
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	processingMu sync.Mutex
	processing   map[string]bool
}

func NewWorkerPool(processor *DeployProcessor, numWorkers int, rdb *redis.Client) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		processor:   processor,
		numWorkers:  numWorkers,
		jobQueue:    make(chan *DeployJob, 100),
		redisClient: rdb,
		ctx:         ctx,
		cancel:      cancel,
		processing:  make(map[string]bool),
	}
}

func (wp *WorkerPool) Start() {
	log.Printf("[pool] Starting %d workers", wp.numWorkers)
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	wp.wg.Add(1)
	go wp.redisConsumer()
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	log.Printf("[worker-%d] Started", id)

	for {
		select {
		case <-wp.ctx.Done():
			log.Printf("[worker-%d] Stopping", id)
			return
		case job := <-wp.jobQueue:
			if job == nil {
				return
			}
			wp.processingMu.Lock()
			wp.processing[job.DeploymentID] = true
			wp.processingMu.Unlock()

			wp.processor.Process(wp.ctx, mustMarshalJSON(job))

			wp.processingMu.Lock()
			delete(wp.processing, job.DeploymentID)
			wp.processingMu.Unlock()
		}
	}
}

func (wp *WorkerPool) redisConsumer() {
	defer wp.wg.Done()
	if wp.redisClient == nil {
		log.Println("[pool] Redis not configured, using queue channel only")
		return
	}

	log.Println("[pool] Redis consumer started")
	for {
		select {
		case <-wp.ctx.Done():
			return
		default:
		}

		result, err := wp.redisClient.BRPop(wp.ctx, 5*time.Second, "bull:deploy-queue:wait").Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			if wp.ctx.Err() != nil {
				return
			}
			log.Printf("[pool] Redis error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if len(result) >= 2 {
			jobID := result[1]
			jobData, err := wp.redisClient.HGet(wp.ctx, "bull:deploy-queue:"+jobID, "data").Result()
			if err != nil {
				log.Printf("[pool] Failed to get job data for %s: %v", jobID, err)
				continue
			}

			var job DeployJob
			if err := json.Unmarshal([]byte(jobData), &job); err != nil {
				log.Printf("[pool] Failed to unmarshal job: %v", err)
				continue
			}

			select {
			case wp.jobQueue <- &job:
				log.Printf("[pool] Job enqueued: %s", jobID)
			case <-wp.ctx.Done():
				return
			}
		}
	}
}

func (wp *WorkerPool) Shutdown(timeout time.Duration) {
	log.Println("[pool] Shutting down worker pool...")
	wp.cancel()

	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("[pool] Worker pool stopped gracefully")
	case <-time.After(timeout):
		log.Println("[pool] Worker pool shutdown timeout")
	}
}

// ── Health Server ─────────────────────────────────────────────────────

func startHealthServer(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool, pool *WorkerPool) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		var dbOK, redisOK bool
		dbOK = db == nil || db.Ping(ctx) == nil
		redisOK = rdb == nil || rdb.Ping(ctx).Err() == nil

		status := "ok"
		code := http.StatusOK
		if !dbOK || !redisOK {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		activeCount := pool.processor.activeCount.Load()
		var processingCount int
		if pool != nil {
			pool.processingMu.Lock()
			processingCount = len(pool.processing)
			pool.processingMu.Unlock()
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		fmt.Fprintf(w, `{"status":"%s","db":%v,"redis":%v,"workers":%d,"active":%d,"cpu":%d,"maxConcurrent":%d}`,
			status, dbOK, redisOK, maxConcurrentBuilds, activeCount, runtime.NumCPU(), maxConcurrentBuilds)
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

func mustMarshalJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// ── Main ──────────────────────────────────────────────────────────────

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("═══════════════════════════════════════════")
	log.Println("  Nidus Deploy Worker v2 (Go - Optimized)")
	log.Printf("  Max Concurrent Builds: %d", maxConcurrentBuilds)
	log.Printf("  Docker Timeout: %v", dockerTimeout)
	log.Printf("  BuildKit: %v", buildkitEnabled)
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

	// PostgreSQL or SQLite
	var dbPool *pgxpool.Pool
	isSQLite := strings.HasPrefix(dbURL, "sqlite") || strings.HasPrefix(dbURL, "file:")

	if isSQLite {
		log.Println("[db] SQLite mode - worker will use API for database operations")
	} else {
		poolConfig, err := pgxpool.ParseConfig(cleanDBURL(dbURL))
		if err != nil {
			log.Fatalf("[fatal] Parse database config: %v", err)
		}
		poolConfig.MaxConns = 20
		poolConfig.MinConns = 5
		poolConfig.MaxConnLifetime = time.Hour
		poolConfig.MaxConnIdleTime = 30 * time.Minute

		dbPool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			log.Fatalf("[fatal] Connect to database: %v", err)
		}
		defer dbPool.Close()

		if err := dbPool.Ping(ctx); err != nil {
			log.Fatalf("[fatal] Ping database: %v", err)
		}
		log.Println("[db] Connected to PostgreSQL")
	}

	// Redis (optional for lite mode)
	var rdb *redis.Client
	if redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Printf("[warning] Parse Redis URL failed: %v", err)
		} else {
			opts.PoolSize = 20
			opts.MinIdleConns = 5
			opts.MaxRetries = 3
			rdb = redis.NewClient(opts)
			if err := rdb.Ping(ctx).Err(); err != nil {
				log.Printf("[warning] Redis connection failed: %v", err)
				rdb = nil
			} else {
				log.Println("[redis] Connected to Redis")
			}
		}
	} else {
		log.Println("[redis] Redis not configured, running in queue channel mode")
	}

	// Docker client (optional - for future native SDK support)
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("[warning] Docker client init failed: %v, using CLI", err)
	}

	processor := &DeployProcessor{
		db:           dbPool,
		rdb:          rdb,
		dockerClient: dockerClient,
	}

	// Worker pool with semaphore
	numWorkers := min(runtime.NumCPU(), 16)
	pool := NewWorkerPool(processor, numWorkers, rdb)
	pool.Start()

	// Health + metrics server
	go startHealthServer(ctx, rdb, dbPool, pool)

	log.Printf("[main] %d workers started, waiting for jobs...", numWorkers)

	// Keep running until context is cancelled
	<-ctx.Done()

	// Graceful shutdown
	pool.Shutdown(30 * time.Second)

	if dbPool != nil {
		dbPool.Close()
	}

	log.Println("[main] Shutdown complete")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
