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
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// ── Config ────────────────────────────────────────────────────────────

var (
	deploysDir = env("NIDUS_DEPLOYS_DIR", "/tmp/nidus-deploys")
	host       = env("NIDUS_HOST", "localhost")
	redisURL   = env("REDIS_URL", "redis://localhost:6379")
	dbURL      = env("DATABASE_URL", "postgresql://broto:broto@localhost:5432/nidus")
	workerPort = env("WORKER_PORT", "8081")
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
	Port          int               `json:"port"`
	EnvVars       map[string]string `json:"envVars,omitempty"`
}

// EnvVar represents a project environment variable
type EnvVar struct {
	Key   string
	Value string
	Secret bool
}

// fetchEnvVars retrieves environment variables for a project from the database
func fetchEnvVars(ctx context.Context, db *pgxpool.Pool, projectID string) (map[string]string, error) {
	envVars := make(map[string]string)

	// Try to fetch from project_env_vars table
	rows, err := db.Query(ctx,
		`SELECT key, value, secret FROM project_env_vars WHERE project_id = $1`,
		projectID)
	if err != nil {
		// Table might not exist, try alternative table name
		rows, err = db.Query(ctx,
			`SELECT key, value, secret FROM environment_variables WHERE project_id = $1`,
			projectID)
		if err != nil {
			return envVars, nil // Return empty map if table doesn't exist
		}
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		var secret bool
		if err := rows.Scan(&key, &value, &secret); err != nil {
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

// ── Deploy Processor ──────────────────────────────────────────────────

type DeployProcessor struct {
	db  *pgxpool.Pool
	rdb *redis.Client
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
		// SQLite mode - log only (API will handle status updates)
		log.Printf("[deploy] Status update: %s -> %s", deploymentID, status)
	}
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
		dp.updateDeploymentStatus(ctx, job.DeploymentID, status, url, logsStr)
		if status == "failed" {
			emailCfg := loadEmailConfig()
			if emailCfg.Enabled {
				userEmail := fetchUserEmail(job.ProjectID, dp.db)
				if userEmail != "" {
					go emailCfg.sendDeployNotification(userEmail, job.ProjectName, "failed", "", job.Branch, time.Since(start))
				}
			}
		}
	}

	logFn(fmt.Sprintf("🚀 Iniciando deploy de %s (%s)...", job.ProjectName, job.Branch))
	dp.db.Exec(ctx, `UPDATE deployments SET status='building', logs=$1 WHERE id=$2`,
		strings.Join(logs, "\n"), job.DeploymentID)

	// ── Fetch environment variables ──
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

	// ── Git clone / fetch ──
	repoDir := filepath.Join(deploysDir, job.ProjectSlug)
	if job.RepoURL != "" {
		// Check if directory exists and has content
		hasContent := false
		if info, err := os.Stat(repoDir); !os.IsNotExist(err) && info.IsDir() {
			// Check if directory has actual files (not just .git)
			entries, _ := os.ReadDir(repoDir)
			for _, entry := range entries {
				if entry.Name() != ".git" {
					hasContent = true
					break
				}
			}
		}

		if !hasContent {
			// Remove old directory if it exists
			if _, err := os.Stat(repoDir); !os.IsNotExist(err) {
				os.RemoveAll(repoDir)
			}
			
			logFn("📦 Clonando repositorio...")
			safeURL := sanitizeShell(job.RepoURL)
			safeDir := sanitizeShell(repoDir)
			
			// Try git clone first
			if out, err := runCmd(ctx, "git", "clone", "--single-branch", safeURL, safeDir); err != nil {
				logFn(fmt.Sprintf("⚠️ Git clone falhou, tentando copia direta: %s", out))
				
				// Fallback: copy files directly if it's a local path
				if _, err := os.Stat(safeURL); err == nil {
					os.MkdirAll(safeDir, 0755)
					runCmd(ctx, "cp", "-r", safeURL+"/.", safeDir)
				} else {
					logFn(fmt.Sprintf("❌ Error clonando: %s", out))
					updateDB("failed", "")
					deploysTotal.WithLabelValues("failed").Inc()
					return
				}
			} else {
				// Checkout files after clone
				runCmd(ctx, "git", "-C", safeDir, "checkout", "HEAD", "--", ".")
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
			runCmd(ctx, "git", "-C", repoDir, "checkout", "HEAD", "--", ".")
		}
		logFn(fmt.Sprintf("✅ Branch %s atualizada", job.Branch))
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

	// ── Check for custom Dockerfile ──
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

	// ── Docker build with streaming (via CLI for real-time logs) ──
	logFn("🐳 Build da imagen Docker...")

	var buildArgs []string
	buildArgs = append(buildArgs, "build")
	buildArgs = append(buildArgs, "-t", job.ImageTag)

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

	// Add build args for environment variables
	for key, value := range envVars {
		buildArgs = append(buildArgs, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	buildArgs = append(buildArgs, "--progress=plain", repoDir)

	logFn(fmt.Sprintf("🔨 Comando: docker %s", strings.Join(buildArgs, " ")))
	buildCmd := exec.CommandContext(ctx, "docker", buildArgs...)

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

	// Build timeout (10 minutes)
	buildCtx, buildCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer buildCancel()

	go func() {
		defer buildWg.Done()
		scanner := bufio.NewScanner(buildStdout)
		for scanner.Scan() {
			if buildCtx.Err() != nil {
				return
			}
			logFn(scanner.Text())
		}
	}()
	go func() {
		defer buildWg.Done()
		scanner := bufio.NewScanner(buildStderr)
		for scanner.Scan() {
			if buildCtx.Err() != nil {
				return
			}
			logFn(scanner.Text())
		}
	}()
	buildWg.Wait()

	if buildCtx.Err() != nil {
		logFn("❌ Build timeout (10 minutes)")
		buildCmd.Process.Kill()
		updateDB("failed", "")
		deploysTotal.WithLabelValues("failed").Inc()
		return
	}

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

	// ── Start new container with env vars ──
	logFn("🚀 Iniciando container...")
	var runArgs []string
	runArgs = append(runArgs, "-d",
		"--name", job.ContainerName,
		"-p", fmt.Sprintf("127.0.0.1:%d:%d", job.Port, exposedPort),
		"--restart", "unless-stopped")

	// Add environment variables
	for key, value := range envVars {
		runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", key, value))
		logFn(fmt.Sprintf("🔧 ENV: %s=%s", key, maskEnvVar(value)))
	}

	runArgs = append(runArgs, job.ImageTag)

	// Build docker run command with all arguments
	dockerArgs := append([]string{"run"}, runArgs...)
	if out, err := runCmd(ctx, "docker", dockerArgs...); err != nil {
		logFn(fmt.Sprintf("❌ Error iniciando: %s", out))
		updateDB("failed", "")
		deploysTotal.WithLabelValues("failed").Inc()
		return
	}

	// ── Container health check ──
	logFn("🏥 Verificando saúde do container...")
	healthOK := false
	for i := 0; i < 10; i++ {
		time.Sleep(2 * time.Second)
		inspectOutput, _ := runCmd(ctx, "docker", "inspect", "--format", "{{.State.Running}}", job.ContainerName)
		if strings.TrimSpace(inspectOutput) == "true" {
			healthOK = true
			break
		}
		logFn(fmt.Sprintf("⏳ Aguardando container... (%d/10)", i+1))
	}

	if !healthOK {
		logFn("❌ Container não está rodando após 20 segundos")
		// Get container logs for debugging
		containerLogs, _ := runCmd(ctx, "docker", "logs", "--tail", "50", job.ContainerName)
		if containerLogs != "" {
			logFn(fmt.Sprintf("📋 Container logs:\n%s", containerLogs))
		}
		updateDB("failed", "")
		deploysTotal.WithLabelValues("failed").Inc()
		return
	}

	url := job.Domain
	if url == "" || job.IsPreview {
		url = fmt.Sprintf("http://%s:%d", host, job.Port)
	}
	logFn(fmt.Sprintf("✅ Deploy concluido em %s", url))

	updateDB("success", url)
	if !job.IsPreview {
		dp.db.Exec(ctx, "UPDATE projects SET status='ACTIVE' WHERE id=$1", job.ProjectID)
	}

	deploysTotal.WithLabelValues("success").Inc()
	deployDuration.Observe(time.Since(start).Seconds())

	// Email notification
	emailCfg := loadEmailConfig()
	if emailCfg.Enabled {
		userEmail := fetchUserEmail(job.ProjectID, dp.db)
		if userEmail != "" {
			go emailCfg.sendDeployNotification(userEmail, job.ProjectName, "success", url, job.Branch, time.Since(start))
		}
	}
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
		dbOK := db == nil || db.Ping(ctx) == nil
		redisOK := rdb == nil || rdb.Ping(ctx).Err() == nil

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

	// PostgreSQL or SQLite
	var dbPool *pgxpool.Pool
	isSQLite := strings.HasPrefix(dbURL, "sqlite") || strings.HasPrefix(dbURL, "file:")
	
	if isSQLite {
		log.Println("[db] SQLite mode - worker will use API for database operations")
		// For SQLite, worker communicates via API instead of direct DB
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
	redisURL := os.Getenv("REDIS_URL")
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
		log.Println("[redis] Redis not configured, running without cache/queue")
	}

	processor := &DeployProcessor{db: dbPool, rdb: rdb}

	// Health + metrics server
	go startHealthServer(ctx, rdb, dbPool)

	// Worker pool
	var wg sync.WaitGroup
	
	if rdb == nil {
		log.Println("[worker] Redis not configured, worker pool disabled (API-only mode)")
		log.Println("[worker] Deploys must be triggered via API")
		// Keep the worker alive
		<-ctx.Done()
	} else {
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
	}

	log.Printf("[main] %d workers started, waiting for jobs...", numWorkers)
	wg.Wait()
	log.Println("[main] Shutdown complete")
}
