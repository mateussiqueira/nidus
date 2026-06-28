package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	db             *sql.DB
	rdb            *redis.Client
	jwtSecret      []byte
	deployQueue    string

	// Request metrics (atomic for thread safety)
	reqTotal       atomic.Int64
	reqSuccess     atomic.Int64
	reqError       atomic.Int64
	reqDurationSum atomic.Int64
	reqCount       atomic.Int64
	durationBuckets [4]atomic.Int64 // <100ms, <500ms, <1s, >=1s
)

const Version = "0.2.0"

func main() {
	loadEnv()

	var err error
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	// Detect database driver based on URL
	var dbInit func() error
	
	if strings.HasPrefix(dbURL, "sqlite") || strings.HasPrefix(dbURL, "file:") {
		// Extract file path from URL
		dbPath := strings.TrimPrefix(dbURL, "sqlite://")
		dbPath = strings.TrimPrefix(dbPath, "file:")
		if dbPath == "" {
			dbPath = "./nidus.db"
		}
		log.Printf("Using SQLite database: %s", dbPath)
		db, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
		dbInit = func() error {
			return initSQLite(db)
		}
	} else {
		// PostgreSQL
		dbURL = strings.Split(dbURL, "?")[0]
		db, err = sql.Open("pgx", dbURL)
		dbInit = func() error {
			// Run migrations for domains and deployment columns
			migrations := []string{
				`CREATE TABLE IF NOT EXISTS domains (
					id TEXT PRIMARY KEY,
					project_id TEXT NOT NULL REFERENCES projects(id),
					domain TEXT NOT NULL UNIQUE,
					verified BOOLEAN DEFAULT FALSE,
					ssl_status TEXT DEFAULT 'pending',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
				`ALTER TABLE deployments ADD COLUMN IF NOT EXISTS container_name TEXT`,
				`ALTER TABLE deployments ADD COLUMN IF NOT EXISTS image_tag TEXT`,
			}
			for _, m := range migrations {
				if _, err := db.Exec(m); err != nil {
					log.Printf("Warning: Migration failed (may be ok): %v", err)
				}
			}
			return nil
		}
	}
	
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

	// Initialize database schema if needed
	if err := dbInit(); err != nil {
		log.Printf("Warning: Database init failed: %v", err)
	}

	// Redis (optional for lite mode)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		opts, _ := redis.ParseURL(redisURL)
		rdb = redis.NewClient(opts)
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("Warning: Redis connection failed: %v", err)
			rdb = nil
		} else {
			deployQueue = "bull:deploy-queue"
		}
	} else {
		log.Println("Redis not configured, running without cache/queue")
	}

	jwtSecret = []byte(getEnv("JWT_SECRET", "local_nidus_jwt_secret_change_me"))

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", handleHealth)

	// Auth
	mux.HandleFunc("POST /api/auth/register", handleRegister)
	mux.HandleFunc("POST /api/auth/login", handleLogin)
	mux.HandleFunc("GET /api/auth/me", withAuth(handleMe))
	mux.HandleFunc("GET /api/auth/github/login", handleGitHubLogin)
	mux.HandleFunc("GET /api/auth/github/callback", handleGitHubCallback)

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

	// Domains
	mux.HandleFunc("GET /api/projects/{projectId}/domains", withAuth(handleListDomains))
	mux.HandleFunc("POST /api/projects/{projectId}/domains", withAuth(handleAddDomain))
	mux.HandleFunc("DELETE /api/projects/{projectId}/domains/{domainId}", withAuth(handleDeleteDomain))
	mux.HandleFunc("POST /api/projects/{projectId}/domains/{domainId}/verify", withAuth(handleVerifyDomain))

	// Rollback
	mux.HandleFunc("POST /api/projects/{projectId}/deployments/{deploymentId}/rollback", withAuth(handleRollback))

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

		duration := time.Since(start)
		reqTotal.Add(1)
		reqDurationSum.Add(duration.Microseconds())
		reqCount.Add(1)

		if sw.status >= 400 {
			reqError.Add(1)
		} else {
			reqSuccess.Add(1)
		}

		d := duration.Milliseconds()
		switch {
		case d < 100:
			durationBuckets[0].Add(1)
		case d < 500:
			durationBuckets[1].Add(1)
		case d < 1000:
			durationBuckets[2].Add(1)
		default:
			durationBuckets[3].Add(1)
		}

		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, duration.Round(time.Millisecond))
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

// ─── GitHub OAuth ────────────────────────────────────────────────────────

var githubOAuthState = uuid.New().String()

func handleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	if clientID == "" {
		jsonError(w, "GitHub OAuth not configured", http.StatusBadRequest)
		return
	}
	redirectURI := os.Getenv("GITHUB_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "https://api.nidus.app/api/auth/github/callback"
	}

	url := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user:email&state=%s",
		clientID, url.QueryEscape(redirectURI), githubOAuthState)
	http.Redirect(w, r, url, http.StatusFound)
}

func handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		jsonError(w, "GitHub OAuth not configured", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state != githubOAuthState {
		jsonError(w, "Invalid state", http.StatusUnauthorized)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		jsonError(w, "Missing code", http.StatusBadRequest)
		return
	}

	// Exchange code for access token
	redirectURI := os.Getenv("GITHUB_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "https://api.nidus.app/api/auth/github/callback"
	}

	tokenURL := fmt.Sprintf("https://github.com/login/oauth/access_token?client_id=%s&client_secret=%s&code=%s&redirect_uri=%s",
		clientID, clientSecret, code, url.QueryEscape(redirectURI))

	req, _ := http.NewRequest("POST", tokenURL, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		jsonError(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error_description"`
	}
	json.NewDecoder(resp.Body).Decode(&tokenResp)

	if tokenResp.Error != "" {
		jsonError(w, "OAuth error: "+tokenResp.Error, http.StatusUnauthorized)
		return
	}

	// Get user info from GitHub
	userReq, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		jsonError(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer userResp.Body.Close()

	var ghUser struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	json.NewDecoder(userResp.Body).Decode(&ghUser)

	if ghUser.Email == "" {
		// Get emails from GitHub if primary email is hidden
		emailReq, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
		emailReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		emailResp, _ := http.DefaultClient.Do(emailReq)
		if emailResp != nil {
			defer emailResp.Body.Close()
			var emails []struct {
				Email   string `json:"email"`
				Primary bool   `json:"primary"`
			}
			json.NewDecoder(emailResp.Body).Decode(&emails)
			for _, e := range emails {
				if e.Primary {
					ghUser.Email = e.Email
					break
				}
			}
			if ghUser.Email == "" && len(emails) > 0 {
				ghUser.Email = emails[0].Email
			}
		}
	}

	if ghUser.Email == "" {
		ghUser.Email = fmt.Sprintf("%s@github.user", ghUser.Login)
	}
	if ghUser.Name == "" {
		ghUser.Name = ghUser.Login
	}

	// Find or create user
	ghID := fmt.Sprintf("gh_%d", ghUser.ID)
	var userID, userName, userEmail string
	err = db.QueryRowContext(r.Context(),
		"SELECT id, name, email FROM users WHERE id = $1", ghID).Scan(&userID, &userName, &userEmail)
	if err == sql.ErrNoRows {
		// Create new user
		now := time.Now().UTC()
		_, err = db.ExecContext(r.Context(),
			`INSERT INTO users (id, email, name, password, avatar, created_at, updated_at)
			 VALUES ($1, $2, $3, '', $4, $5, $6)`,
			ghID, ghUser.Email, ghUser.Name, ghUser.AvatarURL, now, now)
		if err != nil {
			jsonError(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
		userID = ghID
		userName = ghUser.Name
		userEmail = ghUser.Email
	} else if err != nil {
		jsonError(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Generate JWT
	token := generateToken(userID, userEmail)
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://app.nidus.app"
	}
	http.Redirect(w, r, fmt.Sprintf("%s/login?token=%s", frontendURL, token), http.StatusFound)
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

	// Use SQLite-compatible timestamp
	sets = append(sets, "updated_at = datetime('now')")
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

	// If Redis is available, enqueue BullMQ job
	if rdb != nil {
		imageTag := fmt.Sprintf("nidus-%s:%s", slug.String, branch)
		if branch == "main" {
			imageTag = fmt.Sprintf("nidus-%s:latest", slug.String)
		}
		containerName := fmt.Sprintf("nidus-%s", slug.String)
		if branch != "main" {
			containerName = fmt.Sprintf("nidus-%s-preview-%s", slug.String, sanitizeBranch(branch))
		}

		jobData, _ := json.Marshal(map[string]interface{}{
			"deploymentId":  deployID,
			"projectId":     projectID,
			"projectName":   name.String,
			"projectSlug":   slug.String,
			"repoUrl":       repoURL.String,
			"branch":        branch,
			"deployType":    deployType,
			"containerName": containerName,
			"imageTag":      imageTag,
			"isPreview":     branch != "main",
			"safeBranch":    sanitizeBranch(branch),
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
	} else {
		// SQLite mode - deploy directly via API
		log.Printf("[deploy] Triggering deploy for %s (branch: %s)", name.String, branch)
		
		// Update status to building
		db.ExecContext(r.Context(), "UPDATE deployments SET status = 'building' WHERE id = $1", deployID)
		
		// TODO: Execute deploy synchronously or via background goroutine
		// For now, mark as queued and let the user know
		jsonResponse(w, map[string]interface{}{
			"id":      deployID,
			"status":  "queued",
			"jobId":   deployID,
			"branch":  branch,
			"type":    deployType,
			"message": "Deploy queued (Redis not available, manual execution needed)",
		}, http.StatusCreated)
	}
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

	// Verify webhook secret if configured
	webhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if webhookSecret != "" {
		sigHeader := r.Header.Get("x-hub-signature-256")
		if sigHeader == "" {
			jsonError(w, "Missing signature", http.StatusUnauthorized)
			return
		}
		// Read body for signature verification
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			jsonError(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write(bodyBytes)
		expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(sigHeader), []byte(expectedSig)) {
			jsonError(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
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

		safeBranch := sanitizeBranch(branch)
		imageTag := fmt.Sprintf("nidus-%s:%s", slug, safeBranch)
		containerName := fmt.Sprintf("nidus-%s", slug)
		if branch != defaultBranch {
			imageTag = fmt.Sprintf("nidus-%s:preview-%s", slug, safeBranch)
			containerName = fmt.Sprintf("nidus-%s-preview-%s", slug, safeBranch)
		}

		jobData, _ := json.Marshal(map[string]interface{}{
			"deploymentId":  deployID,
			"projectId":     id,
			"projectName":   name,
			"projectSlug":   slug,
			"repoUrl":       body.Repository.CloneURL,
			"branch":        branch,
			"deployType":    deployType,
			"containerName": containerName,
			"imageTag":      imageTag,
			"isPreview":     branch != defaultBranch,
			"safeBranch":    safeBranch,
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

	// Count deployments by status
	var totalDeploys, successDeploys, failedDeploys, activeDeploys int
	db.QueryRow("SELECT COUNT(*) FROM deployments").Scan(&totalDeploys)
	db.QueryRow("SELECT COUNT(*) FROM deployments WHERE status = 'success'").Scan(&successDeploys)
	db.QueryRow("SELECT COUNT(*) FROM deployments WHERE status = 'failed'").Scan(&failedDeploys)
	db.QueryRow("SELECT COUNT(*) FROM deployments WHERE status IN ('building','pending','queued')").Scan(&activeDeploys)

	// Get container info from Docker
	containerRunning, containerTotal := 0, 0
	out, err := exec.Command("docker", "ps", "-q").Output()
	if err == nil {
		containerRunning = len(strings.Fields(string(out)))
	}
	out, err = exec.Command("docker", "ps", "-aq").Output()
	if err == nil {
		containerTotal = len(strings.Fields(string(out)))
	}

	// Disk usage
	diskTotal, diskUsed, diskAvail := float64(0), float64(0), float64(0)
	dfOut, err := exec.Command("df", "-B1", "/").Output()
	if err == nil {
		lines := strings.Split(string(dfOut), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				diskTotal, _ = strconv.ParseFloat(fields[1], 64)
				diskUsed, _ = strconv.ParseFloat(fields[2], 64)
				diskAvail, _ = strconv.ParseFloat(fields[3], 64)
			}
		}
	}

	// Memory info from /proc/meminfo
	memTotal, memAvail := float64(0), float64(0)
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					memTotal, _ = strconv.ParseFloat(fields[1], 64)
					memTotal *= 1024 // kB to bytes
				}
			}
			if strings.HasPrefix(line, "MemAvailable:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					memAvail, _ = strconv.ParseFloat(fields[1], 64)
					memAvail *= 1024
				}
			}
		}
	}
	memUsed := memTotal - memAvail
	memPercent := float64(0)
	if memTotal > 0 {
		memPercent = (memUsed / memTotal) * 100
	}

	uptime := time.Since(startTime).Seconds()

	// Calculate request metrics
	total := reqTotal.Load()
	success := reqSuccess.Load()
	errors := reqError.Load()
	count := reqCount.Load()
	durSum := reqDurationSum.Load()
	b0 := durationBuckets[0].Load()
	b1 := durationBuckets[1].Load()
	b2 := durationBuckets[2].Load()
	b3 := durationBuckets[3].Load()

	avgDuration := float64(0)
	p50 := float64(0)
	p95 := float64(0)
	p99 := float64(0)
	if count > 0 {
		avgDuration = float64(durSum) / float64(count) / 1000 // convert to ms
		// Approximate percentiles from buckets
		target50 := count * 50 / 100
		target95 := count * 95 / 100
		target99 := count * 99 / 100
		soFar := int64(0)
		bucketVals := []struct{ limit int64; val float64 }{
			{b0, 50}, {b1, 300}, {b2, 750}, {b3, 1500},
		}
		for _, bv := range bucketVals {
			soFar += bv.limit
			if soFar >= target50 && p50 == 0 {
				p50 = bv.val
			}
			if soFar >= target95 && p95 == 0 {
				p95 = bv.val
			}
			if soFar >= target99 && p99 == 0 {
				p99 = bv.val
			}
		}
	}

	jsonResponse(w, map[string]interface{}{
		"requests": map[string]interface{}{
			"total":       total,
			"success":     success,
			"error":       errors,
			"avgDuration": avgDuration,
			"p50":         p50,
			"p95":         p95,
			"p99":         p99,
		},
		"memory": map[string]interface{}{
			"total":   memTotal,
			"used":    memUsed,
			"free":    memAvail,
			"percent": memPercent,
			"heapUsed":  float64(m.HeapInuse) / 1048576,
			"heapTotal": float64(m.HeapAlloc) / 1048576,
			"rss":       float64(m.Sys) / 1048576,
		},
		"disk": map[string]interface{}{
			"total":   diskTotal,
			"used":    diskUsed,
			"free":    diskAvail,
			"percent": func() float64 {
				if diskTotal > 0 { return (diskUsed / diskTotal) * 100 }
				return 0
			}(),
		},
		"uptime": uptime,
		"containers": map[string]interface{}{
			"running": containerRunning,
			"total":   containerTotal,
		},
		"deploys": map[string]interface{}{
			"total":   totalDeploys,
			"success": successDeploys,
			"failed":  failedDeploys,
			"active":  activeDeploys,
		},
		"cache": map[string]interface{}{
			"hits": 0, "misses": 0, "hitRate": 0, "size": 0,
		},
		"database": map[string]interface{}{
			"connections": db.Stats().OpenConnections,
			"queries":     0, "avgQueryTime": 0,
		},
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

// ─── Domains ─────────────────────────────────────────────────────────────

func handleListDomains(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")
	if projectID == "" {
		jsonError(w, "Project ID required", http.StatusBadRequest)
		return
	}

	var exists bool
	db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND user_id = $2)", projectID, userID).Scan(&exists)
	if !exists {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	rows, err := db.QueryContext(r.Context(),
		`SELECT id, domain, verified, ssl_status, created_at FROM domains WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	domains := []map[string]interface{}{}
	for rows.Next() {
		var id, domain, sslStatus string
		var verified bool
		var createdAt time.Time
		if err := rows.Scan(&id, &domain, &verified, &sslStatus, &createdAt); err != nil {
			continue
		}
		domains = append(domains, map[string]interface{}{
			"id":         id,
			"domain":     domain,
			"verified":   verified,
			"sslStatus":  sslStatus,
			"createdAt":  createdAt.Format(time.RFC3339),
		})
	}
	jsonResponse(w, domains)
}

func handleAddDomain(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")
	if projectID == "" {
		jsonError(w, "Project ID required", http.StatusBadRequest)
		return
	}

	var exists bool
	db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND user_id = $2)", projectID, userID).Scan(&exists)
	if !exists {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	var body struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Domain == "" {
		jsonError(w, "Domain é obrigatório", http.StatusBadRequest)
		return
	}

	// Check if domain already exists
	var count int
	db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM domains WHERE domain = $1", body.Domain).Scan(&count)
	if count > 0 {
		jsonError(w, "Domínio já cadastrado", http.StatusConflict)
		return
	}

	id := uuid.New().String()
	now := time.Now().UTC()

	_, err := db.ExecContext(r.Context(),
		`INSERT INTO domains (id, project_id, domain, verified, ssl_status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, projectID, body.Domain, false, "pending", now)
	if err != nil {
		jsonError(w, "Erro ao adicionar domínio", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"id":        id,
		"domain":    body.Domain,
		"verified":  false,
		"sslStatus": "pending",
		"createdAt": now.Format(time.RFC3339),
	}, http.StatusCreated)
}

func handleDeleteDomain(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")
	domainID := r.PathValue("domainId")
	if projectID == "" || domainID == "" {
		jsonError(w, "Project ID and Domain ID required", http.StatusBadRequest)
		return
	}

	var exists bool
	db.QueryRowContext(r.Context(),
		"SELECT EXISTS(SELECT 1 FROM domains d JOIN projects p ON d.project_id = p.id WHERE d.id = $1 AND d.project_id = $2 AND p.user_id = $3)",
		domainID, projectID, userID).Scan(&exists)
	if !exists {
		jsonError(w, "Domínio não encontrado", http.StatusNotFound)
		return
	}

	db.ExecContext(r.Context(), "DELETE FROM domains WHERE id = $1", domainID)
	jsonResponse(w, map[string]interface{}{"success": true})
}

func handleVerifyDomain(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")
	domainID := r.PathValue("domainId")
	if projectID == "" || domainID == "" {
		jsonError(w, "Project ID and Domain ID required", http.StatusBadRequest)
		return
	}

	var domain, projectSlug string
	err := db.QueryRowContext(r.Context(),
		`SELECT d.domain, p.slug FROM domains d JOIN projects p ON d.project_id = p.id
		 WHERE d.id = $1 AND d.project_id = $2 AND p.user_id = $3`,
		domainID, projectID, userID).Scan(&domain, &projectSlug)
	if err != nil {
		jsonError(w, "Domínio não encontrado", http.StatusNotFound)
		return
	}

	// Verify domain via DNS TXT record
	nidusIP := getEnv("NIDUS_SERVER_IP", "2.24.204.31")
	txtValue := fmt.Sprintf("nidus-verify=%s", projectSlug)
	verified := verifyDNSTXT(domain, txtValue)

	if verified {
		db.ExecContext(r.Context(),
			`UPDATE domains SET verified = 1, ssl_status = 'verified' WHERE id = $1`, domainID)
		db.ExecContext(r.Context(),
			`UPDATE projects SET domain = $1 WHERE id = $2`, domain, projectID)

		// Trigger Caddy SSL provisioning via API
		go provisionSSLCaddy(domain, r.Context())
	}

	jsonResponse(w, map[string]interface{}{
		"id":       domainID,
		"domain":   domain,
		"verified": verified,
		"sslStatus": map[bool]string{true: "verified", false: "pending"}[verified],
		"expectedTxt": txtValue,
		"ip":          nidusIP,
	})
}

func verifyDNSTXT(domain, expected string) bool {
	// Attempt DNS TXT lookup for verification
	// Fallback: if lookup fails, we still allow manual verification
	cmd := exec.Command("dig", "+short", "TXT", "_nidus-verify."+domain)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[domain] DNS lookup failed for %s: %v", domain, err)
		return false
	}
	txtRecords := strings.TrimSpace(string(out))
	return strings.Contains(txtRecords, expected)
}

func provisionSSLCaddy(domain string, ctx context.Context) {
	// Call Caddy admin API to provision cert
	caddyAdmin := getEnv("CADDY_ADMIN_URL", "http://localhost:2019")
	config := map[string]interface{}{
		"@id": domain,
		"match": []map[string]interface{}{{"host": []string{domain}}},
		"handle": []map[string]interface{}{
			{
				"handler": "subroute",
				"routes": []map[string]interface{}{
					{
						"handle": []map[string]interface{}{
							{"handler": "reverse_proxy", "upstreams": []map[string]interface{}{{"dial": "localhost:3080"}},
							"headers": map[string]interface{}{
								"request": map[string]interface{}{
									"set": map[string]string{"X-Forwarded-Host": "{hostport}"},
								},
							}},
						},
					},
				},
			},
		},
		"tls": map[string]interface{}{},
	}

	body, _ := json.Marshal(config)
	resp, err := http.Post(fmt.Sprintf("%s/config/apps/http/servers/srv0/routes/", caddyAdmin),
		"application/json", strings.NewReader(string(body)))
	if err != nil {
		log.Printf("[domain] Caddy SSL provisioning failed for %s: %v", domain, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("[domain] Caddy SSL provisioned for %s (status: %d)", domain, resp.StatusCode)
}

// ─── Rollback ───────────────────────────────────────────────────────────

func handleRollback(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")
	deploymentID := r.PathValue("deploymentId")
	if projectID == "" || deploymentID == "" {
		jsonError(w, "Project ID and Deployment ID required", http.StatusBadRequest)
		return
	}

	// Verify project ownership and get deployment info
	var depBranch, depType string
	var depURL, containerName, imageTag sql.NullString
	err := db.QueryRowContext(r.Context(),
		`SELECT d.branch, d.type, d.url, d.container_name, d.image_tag
		 FROM deployments d JOIN projects p ON d.project_id = p.id
		 WHERE d.id = $1 AND d.project_id = $2 AND p.user_id = $3`,
		deploymentID, projectID, userID,
	).Scan(&depBranch, &depType, &depURL, &containerName, &imageTag)
	if err == sql.ErrNoRows {
		jsonError(w, "Deployment não encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		jsonError(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	if !imageTag.Valid || imageTag.String == "" {
		jsonError(w, "Este deployment não possui imagem para rollback", http.StatusBadRequest)
		return
	}

	var slug string
	db.QueryRowContext(r.Context(),
		"SELECT slug FROM projects WHERE id = $1", projectID).Scan(&slug)

	newContainerName := containerName.String
	newDeployID := uuid.New().String()

	log.Printf("[rollback] Rolling back project %s to deployment %s (image: %s)", projectID, deploymentID, imageTag.String)

	// Stop and remove current container
	exec.Command("docker", "rm", "-f", newContainerName).Run()

	// Start container with the old image
	exposedPort := getExposedPortFromImage(imageTag.String)
	runArgs := []string{"-d", "--name", newContainerName,
		"-p", fmt.Sprintf("0:%d", exposedPort),
		"--restart", "unless-stopped", imageTag.String}

	if out, err := exec.Command("docker", append([]string{"run"}, runArgs...)...).CombinedOutput(); err != nil {
		log.Printf("[rollback] Docker run failed: %s", out)
		jsonError(w, "Erro ao iniciar container do rollback", http.StatusInternalServerError)
		return
	}

	// Get new port
	portOutput, _ := exec.Command("docker", "port", newContainerName, fmt.Sprintf("%d", exposedPort)).CombinedOutput()
	port := ""
	if lines := strings.Split(strings.TrimSpace(string(portOutput)), "\n"); len(lines) > 0 {
		parts := strings.Split(lines[0], ":")
		if len(parts) > 1 {
			port = parts[len(parts)-1]
		}
	}

	host := getEnv("NIDUS_HOST", "localhost")
	newURL := fmt.Sprintf("http://%s:%s", host, port)
	if depURL.Valid {
		newURL = depURL.String
	}

	// Create rollback deployment record
	now := time.Now().UTC()
	db.ExecContext(r.Context(),
		`INSERT INTO deployments (id, project_id, branch, type, status, url, container_name, image_tag, created_at, finished_at)
		 VALUES ($1, $2, $3, 'rollback', 'success', $4, $5, $6, $7, $7)`,
		newDeployID, projectID, depBranch, newURL, newContainerName, imageTag.String, now)

	db.ExecContext(r.Context(),
		`UPDATE deployments SET status = 'rolled_back' WHERE id = $1`, deploymentID)

	log.Printf("[rollback] Rollback complete: %s -> %s (image: %s)", projectID, newDeployID, imageTag.String)

	jsonResponse(w, map[string]interface{}{
		"id":             newDeployID,
		"rollbackFrom":   deploymentID,
		"status":         "success",
		"url":            newURL,
		"imageTag":       imageTag.String,
		"createdAt":      now.Format(time.RFC3339),
	})
}

func getExposedPortFromImage(imageTag string) int {
	// Try to inspect image for exposed port
	out, err := exec.Command("docker", "inspect", "--format", "{{range $p, $v := .Config.ExposedPorts}}{{$p}} {{end}}", imageTag).Output()
	if err == nil {
		parts := strings.Fields(string(out))
		for _, p := range parts {
			if strings.Contains(p, "/tcp") {
				portStr := strings.Split(p, "/")[0]
				if port, err := strconv.Atoi(portStr); err == nil {
					return port
				}
			}
		}
	}
	return 3000
}

func initSQLite(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		password TEXT NOT NULL,
		avatar TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		slug TEXT UNIQUE NOT NULL,
		user_id TEXT NOT NULL,
		repo_url TEXT,
		branch TEXT DEFAULT 'main',
		framework TEXT,
		status TEXT DEFAULT 'ACTIVE',
		domain TEXT,
		env_vars TEXT,
		database_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		branch TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT NOT NULL,
		url TEXT,
		logs TEXT,
		container_name TEXT,
		image_tag TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		finished_at DATETIME,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);

	CREATE TABLE IF NOT EXISTS databases (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);
	CREATE TABLE IF NOT EXISTS domains (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		domain TEXT NOT NULL UNIQUE,
		verified INTEGER DEFAULT 0,
		ssl_status TEXT DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);
	`
	_, err := db.Exec(schema)
	return err
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
