package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	db             *sql.DB
	rdb            *redis.Client
	jwtSecret      []byte
	deployQueue    string
)

const Version = "0.2.0"

func main() {
	loadEnv()

	var err error
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	// Prisma appends ?schema=public — pgx doesn't need it
	dbURL = strings.Split(dbURL, "?")[0]

	db, err = sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(30 * time.Second)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	opts, _ := redis.ParseURL(redisURL)
	rdb = redis.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	deployQueue = "bull:deploy-queue"

	jwtSecret = []byte(getEnv("JWT_SECRET", "local_nidus_jwt_secret_change_me"))

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", handleHealth)

	// Auth
	mux.HandleFunc("POST /api/auth/register", handleRegister)
	mux.HandleFunc("POST /api/auth/login", handleLogin)
	mux.HandleFunc("GET /api/auth/me", withAuth(handleMe))

	// Projects — catch-all for /api/projects/*
	mux.HandleFunc("/api/projects/", withAuth(handleProjectRoutes))
	mux.HandleFunc("POST /api/projects", withAuth(handleCreateProject))
	mux.HandleFunc("GET /api/projects", withAuth(handleListProjects))

	// Databases — catch-all for /api/databases/*
	mux.HandleFunc("/api/databases/", withAuth(handleDatabaseRoutes))
	mux.HandleFunc("POST /api/databases", withAuth(handleCreateDatabase))
	mux.HandleFunc("GET /api/databases", withAuth(handleListDatabases))

	// Webhook
	mux.HandleFunc("POST /api/webhook/github", handleWebhook)

	// Metrics
	mux.HandleFunc("GET /api/metrics", handleMetrics)
	mux.HandleFunc("GET /api/metrics/prometheus", handlePrometheus)

	// WebSocket for real-time logs
	mux.HandleFunc("GET /api/ws/deployments/{id}/logs", handleWebSocketLogs)

	handler := corsMiddleware(requestIDMiddleware(loggingMiddleware(mux)))

	port := getEnv("API_PORT", "3001")
	log.Printf("Nidus API v%s starting on :%s (Go %s, %d goroutines)", Version, port, runtime.Version(), runtime.NumCPU())

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)
}

// ─── Middleware ──────────────────────────────────────────────────────────────

func corsMiddleware(next http.Handler) http.Handler {
	origins := getEnv("CORS_ORIGINS", "http://localhost:3000")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origins)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-request-id")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("x-request-id")
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set("x-request-id", id)
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tokenStr == "" {
			jsonError(w, "Token ausente", http.StatusUnauthorized)
			return
		}
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			jsonError(w, "Token invalido", http.StatusUnauthorized)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			jsonError(w, "Token invalido", http.StatusUnauthorized)
			return
		}
		sub, _ := claims["sub"].(string)
		ctx := context.WithValue(r.Context(), "userID", sub)
		next(w, r.WithContext(ctx))
	}
}

// ─── Handlers ───────────────────────────────────────────────────────────────

func handleHealth(w http.ResponseWriter, r *http.Request) {
	err := db.PingContext(r.Context())
	status := "ok"
	dbConnected := err == nil
	if !dbConnected {
		status = "error"
	}
	jsonResponse(w, map[string]interface{}{
		"status":       status,
		"name":         "nidus-control-plane",
		"version":      Version,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"dbConnected":  dbConnected,
	})
}

// ─── Auth ───────────────────────────────────────────────────────────────────

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Body invalido", http.StatusBadRequest)
		return
	}
	if body.Email == "" || body.Name == "" || body.Password == "" {
		jsonError(w, "Email, name e password sao obrigatorios", http.StatusBadRequest)
		return
	}

	var exists bool
	db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", body.Email).Scan(&exists)
	if exists {
		jsonError(w, "Email ja cadastrado", http.StatusConflict)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	var id, name, email string
	id = uuid.New().String()
	now := time.Now().UTC()
	err = db.QueryRowContext(r.Context(),
		"INSERT INTO users (id, email, name, password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, name, email",
		id, body.Email, body.Name, string(hash), now, now,
	).Scan(&id, &name, &email)
	if err != nil {
		log.Printf("Register DB error: %v (email=%s)", err, body.Email)
		jsonError(w, "Erro ao criar usuario", http.StatusInternalServerError)
		return
	}

	token := generateToken(id, email)
	jsonResponse(w, map[string]interface{}{
		"token": token,
		"user":  map[string]string{"id": id, "name": name, "email": email},
	}, http.StatusCreated)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Body invalido", http.StatusBadRequest)
		return
	}

	var id, name, email, password string
	err := db.QueryRowContext(r.Context(),
		"SELECT id, name, email, password FROM users WHERE email = $1", body.Email,
	).Scan(&id, &name, &email, &password)
	if err == sql.ErrNoRows {
		jsonError(w, "Credenciais invalidas", http.StatusUnauthorized)
		return
	}
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(body.Password)); err != nil {
		jsonError(w, "Credenciais invalidas", http.StatusUnauthorized)
		return
	}

	token := generateToken(id, email)
	jsonResponse(w, map[string]interface{}{
		"token": token,
		"user":  map[string]string{"id": id, "name": name, "email": email},
	})
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)

	var name, email string
	var avatar sql.NullString
	var createdAt time.Time
	err := db.QueryRowContext(r.Context(),
		"SELECT name, email, avatar, created_at FROM users WHERE id = $1", userID,
	).Scan(&name, &email, &avatar, &createdAt)
	if err == sql.ErrNoRows {
		jsonError(w, "Usuario nao encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"id":         userID,
		"name":       name,
		"email":      email,
		"avatar":     nullString(avatar),
		"created_at": createdAt.Format(time.RFC3339),
	})
}

// ─── Projects ───────────────────────────────────────────────────────────────

func handleListProjects(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)

	rows, err := db.QueryContext(r.Context(),
		`SELECT id, name, slug, framework, status, domain, repo_url, env_vars, created_at
		 FROM projects WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	projects := []map[string]interface{}{}
	for rows.Next() {
		var id, name, slug, status string
		var framework, domain, repoURL, envVars sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &slug, &framework, &status, &domain, &repoURL, &envVars, &createdAt); err != nil {
			continue
		}
		projects = append(projects, map[string]interface{}{
			"id":        id,
			"name":      name,
			"slug":      slug,
			"framework": nullString(framework),
			"status":    status,
			"domain":    nullString(domain),
			"repoUrl":   nullString(repoURL),
			"envVars":   nullString(envVars),
			"createdAt": createdAt.Format(time.RFC3339),
		})
	}
	jsonResponse(w, projects)
}

func handleCreateProject(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)

	var body struct {
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		RepoURL   string `json:"repoUrl"`
		Framework string `json:"framework"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Body invalido", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		jsonError(w, "Name e obrigatorio", http.StatusBadRequest)
		return
	}

	slug := body.Slug
	if slug == "" {
		slug = generateSlug(body.Name)
	}

	var id, status string
	var framework, domain, repoURL, envVars sql.NullString
	var createdAt time.Time
	id = uuid.New().String()
	now := time.Now().UTC()
	err := db.QueryRowContext(r.Context(),
		`INSERT INTO projects (id, name, slug, user_id, framework, repo_url, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, nullif($5,''), nullif($6,''), 'ACTIVE', $7, $7)
		 RETURNING id, name, slug, framework, status, domain, repo_url, env_vars, created_at`,
		id, body.Name, slug, userID, body.Framework, body.RepoURL, now,
	).Scan(&id, &body.Name, &slug, &framework, &status, &domain, &repoURL, &envVars, &createdAt)
	if err != nil {
		jsonError(w, "Erro ao criar projeto", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"id":        id,
		"name":      body.Name,
		"slug":      slug,
		"framework": nullString(framework),
		"status":    status,
		"domain":    nullString(domain),
		"repoUrl":   nullString(repoURL),
		"envVars":   nullString(envVars),
		"createdAt": createdAt.Format(time.RFC3339),
	}, http.StatusCreated)
}

func handleProjectRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.Split(path, "/")

	// GET /api/projects/:id
	if len(parts) == 1 || (len(parts) == 2 && parts[1] == "") {
		if r.Method == "GET" {
			handleGetProject(w, r, parts[0])
			return
		}
		if r.Method == "PATCH" {
			handleUpdateProject(w, r, parts[0])
			return
		}
	}

	// GET /api/projects/:id/deployments
	if len(parts) == 2 && parts[1] == "deployments" {
		handleListDeployments(w, r, parts[0])
		return
	}

	// POST /api/projects/:id/deploy
	if len(parts) == 2 && parts[1] == "deploy" {
		handleDeploy(w, r, parts[0])
		return
	}

	// GET /api/projects/:id/metrics
	if len(parts) == 2 && parts[1] == "metrics" {
		handleProjectMetrics(w, r, parts[0])
		return
	}

	// Nested routes with projectId
	if len(parts) >= 2 {
		projectID := parts[0]
		sub := parts[1]

		switch sub {
		case "deployments":
			if len(parts) == 2 {
				handleListDeployments(w, r, projectID)
			} else if len(parts) == 3 {
				handleGetDeployment(w, r, projectID, parts[2])
			} else if len(parts) == 4 && parts[3] == "logs" {
				handleDeploymentLogs(w, r, projectID, parts[2])
			}
			return
		case "previews":
			handleListPreviews(w, r, projectID)
			return
		case "metrics":
			handleProjectMetrics(w, r, projectID)
			return
		case "deploy":
			handleDeploy(w, r, projectID)
			return
		}
	}

	jsonError(w, "Not found", http.StatusNotFound)
}

func handleGetProject(w http.ResponseWriter, r *http.Request, id string) {
	userID := r.Context().Value("userID").(string)

	var name, slug, pStatus string
	var framework, domain, repoURL, envVars sql.NullString
	var createdAt time.Time
	err := db.QueryRowContext(r.Context(),
		`SELECT name, slug, framework, status, domain, repo_url, env_vars, created_at
		 FROM projects WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&name, &slug, &framework, &pStatus, &domain, &repoURL, &envVars, &createdAt)
	if err == sql.ErrNoRows {
		jsonResponse(w, nil)
		return
	}
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"id":        id,
		"name":      name,
		"slug":      slug,
		"framework": nullString(framework),
		"status":    pStatus,
		"domain":    nullString(domain),
		"repoUrl":   nullString(repoURL),
		"envVars":   nullString(envVars),
		"createdAt": createdAt.Format(time.RFC3339),
	})
}

func handleUpdateProject(w http.ResponseWriter, r *http.Request, id string) {
	userID := r.Context().Value("userID").(string)

	var body struct {
		EnvVars string `json:"envVars"`
		Domain  string `json:"domain"`
		RepoURL string `json:"repoUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Body invalido", http.StatusBadRequest)
		return
	}

	sets := []string{}
	args := []interface{}{}
	argIdx := 1

	if body.EnvVars != "" {
		sets = append(sets, fmt.Sprintf("env_vars = $%d", argIdx))
		args = append(args, body.EnvVars)
		argIdx++
	}
	if body.Domain != "" {
		sets = append(sets, fmt.Sprintf("domain = $%d", argIdx))
		args = append(args, body.Domain)
		argIdx++
	}
	if body.RepoURL != "" {
		sets = append(sets, fmt.Sprintf("repo_url = $%d", argIdx))
		args = append(args, body.RepoURL)
		argIdx++
	}

	if len(sets) == 0 {
		handleGetProject(w, r, id)
		return
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id, userID)

	query := fmt.Sprintf("UPDATE projects SET %s WHERE id = $%d AND user_id = $%d RETURNING id, name, slug, framework, status, domain, repo_url, env_vars, created_at",
		strings.Join(sets, ", "), argIdx, argIdx+1)

	var name, slug, pStatus string
	var framework, domain, repoURL, envVars sql.NullString
	var createdAt time.Time
	err := db.QueryRowContext(r.Context(), query, args...).Scan(&id, &name, &slug, &framework, &pStatus, &domain, &repoURL, &envVars, &createdAt)
	if err != nil {
		jsonError(w, "Erro ao atualizar projeto", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"id":        id,
		"name":      name,
		"slug":      slug,
		"framework": nullString(framework),
		"status":    pStatus,
		"domain":    nullString(domain),
		"repoUrl":   nullString(repoURL),
		"envVars":   nullString(envVars),
		"createdAt": createdAt.Format(time.RFC3339),
	})
}

// ─── Deployments ────────────────────────────────────────────────────────────

func handleListDeployments(w http.ResponseWriter, r *http.Request, projectID string) {
	userID := r.Context().Value("userID").(string)

	// Verify ownership
	var exists bool
	db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND user_id = $2)", projectID, userID).Scan(&exists)
	if !exists {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	rows, err := db.QueryContext(r.Context(),
		`SELECT id, status, url, branch, type, created_at, finished_at
		 FROM deployments WHERE project_id = $1 ORDER BY created_at DESC LIMIT 50`, projectID)
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	deployments := []map[string]interface{}{}
	for rows.Next() {
		var id, status, branch, deployType string
		var url sql.NullString
		var createdAt time.Time
		var finishedAt sql.NullTime
		if err := rows.Scan(&id, &status, &url, &branch, &deployType, &createdAt, &finishedAt); err != nil {
			continue
		}
		deployments = append(deployments, map[string]interface{}{
			"id":        id,
			"status":    status,
			"url":       nullString(url),
			"branch":    branch,
			"type":      deployType,
			"createdAt": createdAt.Format(time.RFC3339),
			"finishedAt": nullTime(finishedAt),
		})
	}
	jsonResponse(w, deployments)
}

func handleListPreviews(w http.ResponseWriter, r *http.Request, projectID string) {
	userID := r.Context().Value("userID").(string)

	var exists bool
	db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND user_id = $2)", projectID, userID).Scan(&exists)
	if !exists {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	rows, err := db.QueryContext(r.Context(),
		`SELECT id, status, url, branch, type, created_at, finished_at
		 FROM deployments WHERE project_id = $1 AND type = 'preview' ORDER BY created_at DESC LIMIT 20`, projectID)
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	deployments := []map[string]interface{}{}
	for rows.Next() {
		var id, status, branch, deployType string
		var url sql.NullString
		var createdAt time.Time
		var finishedAt sql.NullTime
		if err := rows.Scan(&id, &status, &url, &branch, &deployType, &createdAt, &finishedAt); err != nil {
			continue
		}
		deployments = append(deployments, map[string]interface{}{
			"id":        id,
			"status":    status,
			"url":       nullString(url),
			"branch":    branch,
			"type":      deployType,
			"createdAt": createdAt.Format(time.RFC3339),
			"finishedAt": nullTime(finishedAt),
		})
	}
	jsonResponse(w, deployments)
}

func handleGetDeployment(w http.ResponseWriter, r *http.Request, projectID, deploymentID string) {
	userID := r.Context().Value("userID").(string)

	var exists bool
	db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND user_id = $2)", projectID, userID).Scan(&exists)
	if !exists {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	var status, branch, deployType string
	var url sql.NullString
	var createdAt time.Time
	var finishedAt sql.NullTime
	err := db.QueryRowContext(r.Context(),
		`SELECT status, url, branch, type, created_at, finished_at
		 FROM deployments WHERE id = $1 AND project_id = $2`, deploymentID, projectID,
	).Scan(&status, &url, &branch, &deployType, &createdAt, &finishedAt)
	if err == sql.ErrNoRows {
		jsonResponse(w, nil)
		return
	}
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"id":        deploymentID,
		"status":    status,
		"url":       nullString(url),
		"branch":    branch,
		"type":      deployType,
		"createdAt": createdAt.Format(time.RFC3339),
		"finishedAt": nullTime(finishedAt),
	})
}

func handleDeploymentLogs(w http.ResponseWriter, r *http.Request, projectID, deploymentID string) {
	userID := r.Context().Value("userID").(string)

	var exists bool
	db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND user_id = $2)", projectID, userID).Scan(&exists)
	if !exists {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	var logs sql.NullString
	err := db.QueryRowContext(r.Context(),
		"SELECT logs FROM deployments WHERE id = $1 AND project_id = $2", deploymentID, projectID,
	).Scan(&logs)
	if err == sql.ErrNoRows {
		jsonResponse(w, "")
		return
	}
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if logs.Valid {
		w.Write([]byte(logs.String))
	}
}

func handleDeploy(w http.ResponseWriter, r *http.Request, projectID string) {
	userID := r.Context().Value("userID").(string)

	var name, slug, repoURL sql.NullString
	var branch string
	err := db.QueryRowContext(r.Context(),
		"SELECT name, slug, repo_url, branch FROM projects WHERE id = $1 AND user_id = $2", projectID, userID,
	).Scan(&name, &slug, &repoURL, &branch)
	if err == sql.ErrNoRows {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	// Query param branch
	if b := r.URL.Query().Get("branch"); b != "" {
		branch = sanitizeShell(b)
	}
	if branch == "" {
		branch = "main"
	}

	deployType := "production"
	if branch != "main" {
		deployType = "preview"
	}

	deployID := uuid.New().String()

	// Create deployment record
	_, err = db.ExecContext(r.Context(),
		`INSERT INTO deployments (id, project_id, branch, type, status)
		 VALUES ($1, $2, $3, $4, 'queued')`, deployID, projectID, branch, deployType)
	if err != nil {
		jsonError(w, "Erro ao criar deploy", http.StatusInternalServerError)
		return
	}

	// Enqueue BullMQ job
	jobData, _ := json.Marshal(map[string]interface{}{
		"deploymentId": deployID,
		"projectId":    projectID,
		"projectName":  name.String,
		"projectSlug":  slug.String,
		"repoUrl":      repoURL.String,
		"branch":       branch,
	})

	ctx := r.Context()
	jobID := fmt.Sprintf("%d", time.Now().UnixNano())

	// BullMQ protocol: LPUSH to wait set, HSET job data
	rdb.LPush(ctx, deployQueue+":wait", jobID)
	rdb.HSet(ctx, deployQueue+":"+jobID, map[string]interface{}{
		"data": string(jobData),
		"id":   jobID,
		"name": "deploy",
	})
	rdb.ZAdd(ctx, deployQueue+":delayed", redis.Z{Score: 0, Member: jobID})

	jsonResponse(w, map[string]interface{}{
		"id":      deployID,
		"status":  "queued",
		"jobId":   jobID,
		"branch":  branch,
		"type":    deployType,
	}, http.StatusCreated)
}

func handleProjectMetrics(w http.ResponseWriter, r *http.Request, projectID string) {
	userID := r.Context().Value("userID").(string)

	var slug string
	err := db.QueryRowContext(r.Context(),
		"SELECT slug FROM projects WHERE id = $1 AND user_id = $2", projectID, userID,
	).Scan(&slug)
	if err != nil {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	branch := r.URL.Query().Get("branch")
	containerName := "nidus-" + slug
	if branch != "" && branch != "main" {
		containerName = "nidus-" + slug + "-preview-" + sanitizeBranch(branch)
	}

	// Docker inspect
	out, err := exec.Command("docker", "inspect", "--format", "{{.State.StartedAt}} {{.State.Running}} {{.RestartCount}} {{.State.ExitCode}}", containerName).Output()
	if err != nil {
		jsonResponse(w, map[string]interface{}{
			"status": "stopped", "running": false, "cpu": 0, "memory": 0,
		})
		return
	}

	fields := strings.Fields(string(out))
	startedAt := fields[0]
	running := fields[1] == "true"
	restartCount, _ := strconv.Atoi(fields[2])
	exitCode, _ := strconv.Atoi(fields[3])

	// Docker stats (one-shot)
	statsCtx, statsCancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer statsCancel()
	statsCmd := exec.CommandContext(statsCtx, "docker", "stats", containerName, "--no-stream", "--format", "{{.CPUPerc}} {{.MemUsage}} {{.MemPerc}} {{.NetIO}}")
	statsOut, err := statsCmd.Output()
	cpu, memUsage, memPercent, netIO := "0", "0MB", "0", "0/0"
	if err == nil {
		statsFields := strings.Fields(string(statsOut))
		if len(statsFields) >= 4 {
			cpu = strings.TrimSuffix(statsFields[0], "%")
			memUsage = statsFields[1]
			memPercent = strings.TrimSuffix(statsFields[3], "%")
			netIO = statsFields[4]
		}
	}

	jsonResponse(w, map[string]interface{}{
		"status":       "running",
		"running":      running,
		"startedAt":    startedAt,
		"cpu":          cpu,
		"memory":       map[string]string{"usage": memUsage, "percent": memPercent},
		"network":      netIO,
		"restartCount": restartCount,
		"exitCode":     exitCode,
	})
}

// ─── Databases ──────────────────────────────────────────────────────────────

func handleListDatabases(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)

	rows, err := db.QueryContext(r.Context(),
		`SELECT d.id, d.name, d.url, d.project_id, d.created_at
		 FROM databases d JOIN projects p ON d.project_id = p.id
		 WHERE p.user_id = $1 ORDER BY d.created_at DESC`, userID)
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	databases := []map[string]interface{}{}
	for rows.Next() {
		var id, name string
		var url, projectID sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &url, &projectID, &createdAt); err != nil {
			continue
		}
		databases = append(databases, map[string]interface{}{
			"id":        id,
			"name":      name,
			"url":       nullString(url),
			"projectId": nullString(projectID),
			"createdAt": createdAt.Format(time.RFC3339),
		})
	}
	jsonResponse(w, databases)
}

func handleDatabaseRoutes(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/databases/")
	if id == "" {
		jsonError(w, "Not found", http.StatusNotFound)
		return
	}
	switch r.Method {
	case "GET":
		handleGetDatabase(w, r, id)
	case "DELETE":
		handleDeleteDatabase(w, r, id)
	default:
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetDatabase(w http.ResponseWriter, r *http.Request, id string) {
	userID := r.Context().Value("userID").(string)

	var name string
	var url, projectID sql.NullString
	var createdAt time.Time
	err := db.QueryRowContext(r.Context(),
		`SELECT d.name, d.url, d.project_id, d.created_at
		 FROM databases d JOIN projects p ON d.project_id = p.id
		 WHERE d.id = $1 AND p.user_id = $2`, id, userID,
	).Scan(&name, &url, &projectID, &createdAt)
	if err == sql.ErrNoRows {
		jsonResponse(w, nil)
		return
	}
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"id":        id,
		"name":      name,
		"url":       nullString(url),
		"projectId": nullString(projectID),
		"createdAt": createdAt.Format(time.RFC3339),
	})
}

func handleCreateDatabase(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)

	var body struct {
		ProjectID string `json:"projectId"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Body invalido", http.StatusBadRequest)
		return
	}
	if body.ProjectID == "" || body.Name == "" {
		jsonError(w, "projectId e name sao obrigatorios", http.StatusBadRequest)
		return
	}

	// Verify project ownership
	var exists bool
	db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND user_id = $2)", body.ProjectID, userID).Scan(&exists)
	if !exists {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	dbName := "nidus_" + body.Name
	password := generatePassword(16)

	psqlPath := "/opt/homebrew/bin/psql"
	createdbPath := "/opt/homebrew/bin/createdb"

	// Create database
	exec.Command(createdbPath, "-U", "broto", dbName).Run()

	// Create user + grant
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://broto@localhost:5432/nidus"
	}
	dbURL = strings.Split(dbURL, "?")[0]

	sqlConn, err := sql.Open("pgx", dbURL)
	if err == nil {
		sqlConn.Exec(fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", dbName, password))
		sqlConn.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s", dbName, dbName))
		sqlConn.Exec(fmt.Sprintf("GRANT ALL ON SCHEMA public TO %s", dbName))
		sqlConn.Close()
	}

	connURL := fmt.Sprintf("postgresql://%s:%s@localhost:5432/%s", dbName, password, dbName)

	var id string
	var createdAt time.Time
	id = uuid.New().String()
	err = db.QueryRowContext(r.Context(),
		`INSERT INTO databases (id, project_id, name, url) VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`, id, body.ProjectID, body.Name, connURL,
	).Scan(&id, &createdAt)
	if err != nil {
		jsonError(w, "Erro ao criar database", http.StatusInternalServerError)
		return
	}

	// Link to project
	db.ExecContext(r.Context(), "UPDATE projects SET database_id = $1 WHERE id = $2", id, body.ProjectID)

	// Update psql permissions
	exec.Command(psqlPath, "-U", "broto", "-d", dbName, "-c",
		fmt.Sprintf("GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO %s; GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO %s", dbName, dbName),
	).Run()

	jsonResponse(w, map[string]interface{}{
		"id":        id,
		"name":      body.Name,
		"url":       connURL,
		"projectId": body.ProjectID,
		"createdAt": createdAt.Format(time.RFC3339),
	}, http.StatusCreated)
}

func handleDeleteDatabase(w http.ResponseWriter, r *http.Request, id string) {
	userID := r.Context().Value("userID").(string)

	var name string
	err := db.QueryRowContext(r.Context(),
		`SELECT d.name FROM databases d JOIN projects p ON d.project_id = p.id
		 WHERE d.id = $1 AND p.user_id = $2`, id, userID,
	).Scan(&name)
	if err == sql.ErrNoRows {
		jsonError(w, "Database nao encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	dbName := "nidus_" + name
	dropdbPath := "/opt/homebrew/bin/dropdb"

	exec.Command(dropdbPath, "-U", "broto", "--if-exists", dbName).Run()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://broto@localhost:5432/nidus"
	}
	dbURL = strings.Split(dbURL, "?")[0]
	if sqlConn, err := sql.Open("pgx", dbURL); err == nil {
		sqlConn.Exec(fmt.Sprintf("DROP USER IF EXISTS %s", dbName))
		sqlConn.Close()
	}

	db.ExecContext(r.Context(), "DELETE FROM databases WHERE id = $1", id)

	jsonResponse(w, map[string]interface{}{"success": true})
}

// ─── Webhook ────────────────────────────────────────────────────────────────

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	event := r.Header.Get("x-github-event")

	if event == "ping" {
		jsonResponse(w, map[string]interface{}{"ok": true, "msg": "pong"})
		return
	}

	if event != "push" {
		jsonResponse(w, map[string]interface{}{"ok": false, "msg": "event " + event + " ignored"})
		return
	}

	var body struct {
		Ref        string `json:"ref"`
		Repository struct {
			CloneURL string `json:"clone_url"`
		} `json:"repository"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Body invalido", http.StatusBadRequest)
		return
	}

	if body.Repository.CloneURL == "" || body.Ref == "" {
		jsonResponse(w, map[string]interface{}{"ok": false, "msg": "missing repo_url or branch"})
		return
	}

	branch := strings.TrimPrefix(body.Ref, "refs/heads/")

	rows, err := db.QueryContext(r.Context(),
		"SELECT id, name, slug, branch FROM projects WHERE repo_url = $1", body.Repository.CloneURL)
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type result struct {
		Project string `json:"project"`
		Slug    string `json:"slug"`
		Branch  string `json:"branch"`
		Type    string `json:"type"`
		Status  string `json:"status"`
		JobID   string `json:"jobId"`
	}

	results := []result{}

	for rows.Next() {
		var id, name, slug, defaultBranch string
		if err := rows.Scan(&id, &name, &slug, &defaultBranch); err != nil {
			continue
		}

		deployType := "preview"
		if branch == defaultBranch {
			deployType = "production"
		}

		deployID := uuid.New().String()
		db.ExecContext(r.Context(),
			"INSERT INTO deployments (id, project_id, branch, type, status) VALUES ($1, $2, $3, $4, 'queued')",
			deployID, id, branch, deployType)

		jobData, _ := json.Marshal(map[string]interface{}{
			"deploymentId": deployID,
			"projectId":    id,
			"projectName":  name,
			"projectSlug":  slug,
			"repoUrl":      body.Repository.CloneURL,
			"branch":       branch,
		})

		ctx := r.Context()
		jobID := fmt.Sprintf("%d", time.Now().UnixNano())
		rdb.LPush(ctx, deployQueue+":wait", jobID)
		rdb.HSet(ctx, deployQueue+":"+jobID, map[string]interface{}{
			"data": string(jobData),
			"id":   jobID,
			"name": "deploy",
		})

		results = append(results, result{
			Project: name, Slug: slug, Branch: branch,
			Type: deployType, Status: "queued", JobID: jobID,
		})
	}

	jsonResponse(w, map[string]interface{}{
		"ok":       true,
		"deployed": len(results),
		"results":  results,
	})
}

// ─── Metrics ────────────────────────────────────────────────────────────────

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	jsonResponse(w, map[string]interface{}{
		"requests": map[string]interface{}{
			"total": 0, "success": 0, "error": 0, "avgDuration": 0,
			"p50": 0, "p95": 0, "p99": 0,
		},
		"cache": map[string]interface{}{
			"hits": 0, "misses": 0, "hitRate": 0, "size": 0,
		},
		"database": map[string]interface{}{
			"connections": db.Stats().OpenConnections,
			"queries":     0, "avgQueryTime": 0,
		},
		"memory": map[string]interface{}{
			"heapUsed":  float64(m.HeapInuse) / 1048576,
			"heapTotal": float64(m.HeapAlloc) / 1048576,
			"rss":       float64(m.Sys) / 1048576,
			"external":  float64(m.Mallocs) / 1048576,
		},
		"uptime": time.Since(startTime).Seconds(),
	})
}

func handlePrometheus(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, `# HELP nidus_uptime_seconds Uptime in seconds
# TYPE nidus_uptime_seconds gauge
nidus_uptime_seconds %.2f

# HELP nidus_db_open_connections Database open connections
# TYPE nidus_db_open_connections gauge
nidus_db_open_connections %d

# HELP nidus_memory_heap_used_bytes Heap memory used
# TYPE nidus_memory_heap_used_bytes gauge
nidus_memory_heap_used_bytes %d

# HELP nidus_memory_rss_bytes Resident set size
# TYPE nidus_memory_rss_bytes gauge
nidus_memory_rss_bytes %d
`, time.Since(startTime).Seconds(), db.Stats().OpenConnections, m.HeapInuse, m.Sys)
}

var startTime = time.Now()

// ─── Helpers ────────────────────────────────────────────────────────────────

func generateToken(userID, email string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})
	s, _ := token.SignedString(jwtSecret)
	return s
}

func generateSlug(name string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := re.ReplaceAllString(strings.ToLower(name), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = uuid.New().String()[:8]
	}
	return slug
}

func sanitizeShell(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9._\/-]`)
	return re.ReplaceAllString(s, "")
}

func sanitizeBranch(branch string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)
	sanitized := re.ReplaceAllString(strings.ToLower(branch), "-")
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}
	return sanitized
}

func generatePassword(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[n.Int64()]
	}
	return string(result)
}

func nullString(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func nullTime(nt sql.NullTime) interface{} {
	if nt.Valid {
		return nt.Time.Format(time.RFC3339)
	}
	return nil
}

func jsonResponse(w http.ResponseWriter, data interface{}, status ...int) {
	code := 200
	if len(status) > 0 {
		code = status[0]
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	jsonResponse(w, map[string]string{"message": msg}, status)
}

// ─── WebSocket for Real-time Deployment Logs ────────────────────────────────

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for WebSocket
	},
}

func handleWebSocketLogs(w http.ResponseWriter, r *http.Request) {
	// Extract deployment ID from path
	deploymentID := r.PathValue("id")
	if deploymentID == "" {
		http.Error(w, "Missing deployment ID", http.StatusBadRequest)
		return
	}

	// Verify deployment exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM deployments WHERE id=$1)", deploymentID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Deployment not found", http.StatusNotFound)
		return
	}

	// Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("WebSocket client connected for deployment %s", deploymentID)

	// Send initial logs
	var logs sql.NullString
	err = db.QueryRow("SELECT logs FROM deployments WHERE id=$1", deploymentID).Scan(&logs)
	if err == nil && logs.Valid {
		conn.WriteMessage(websocket.TextMessage, []byte(logs.String))
	}

	// Poll for new logs every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastLogLen int
	if logs.Valid {
		lastLogLen = len(logs.String)
	}

	for {
		select {
		case <-ticker.C:
			// Check if deployment is still building
			var status string
			err := db.QueryRow("SELECT status FROM deployments WHERE id=$1", deploymentID).Scan(&status)
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte("Error checking deployment status"))
				return
			}

			// Get current logs
			err = db.QueryRow("SELECT logs FROM deployments WHERE id=$1", deploymentID).Scan(&logs)
			if err == nil && logs.Valid && len(logs.String) > lastLogLen {
				// Send only new logs
				newLogs := logs.String[lastLogLen:]
				conn.WriteMessage(websocket.TextMessage, []byte(newLogs))
				lastLogLen = len(logs.String)
			}

			// If deployment is done, send final message and close
			if status == "success" || status == "failed" {
				conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\n[Deploy %s]", status)))
				return
			}

		case <-r.Context().Done():
			return
		}
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, "\"'")
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}
