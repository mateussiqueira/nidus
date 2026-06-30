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
	"github.com/stackrun-dev/stackrun-api/mail"
	"io"
	"log"
	"math"
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
	"unicode"
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

	// Mail service
	mail.Init(db)
	mail.Configure(mail.Config{
		Provider:  "sendmail",
		FromName:  "StackRun",
		FromEmail: "noreply@stackrun.vercel.app",
	})

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

	jwtSecret = []byte(getEnv("JWT_SECRET", "local_stackrun_jwt_secret_change_me"))

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
	mux.HandleFunc("PUT /api/projects", func(w http.ResponseWriter, r *http.Request) {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("PUT /api/projects/", func(w http.ResponseWriter, r *http.Request) {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("PATCH /api/projects", func(w http.ResponseWriter, r *http.Request) {
		jsonError(w, "Method not allowed. Use PATCH /api/projects/{id}", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("PATCH /api/projects/", func(w http.ResponseWriter, r *http.Request) {
		jsonError(w, "Method not allowed. Use PATCH /api/projects/{id}", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("DELETE /api/projects", func(w http.ResponseWriter, r *http.Request) {
		jsonError(w, "Method not allowed. Use DELETE /api/projects/{id}", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("DELETE /api/projects/", func(w http.ResponseWriter, r *http.Request) {
		jsonError(w, "Method not allowed. Use DELETE /api/projects/{id}", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("GET /api/projects", withAuth(handleListProjects))

	// Databases — catch-all for /api/databases/*
	mux.HandleFunc("/api/databases/", withAuth(handleDatabaseRoutes))
	mux.HandleFunc("POST /api/databases", withAuth(handleCreateDatabase))
	mux.HandleFunc("GET /api/databases", withAuth(handleListDatabases))

	// Mail
	mux.HandleFunc("POST /api/mail/send", withAuth(handleSendMail))
	// DISABLED: mux.HandleFunc("GET /api/mail/templates", withAuth(handleListTemplates))
	// DISABLED: mux.HandleFunc("POST /api/mail/templates", withAuth(handleCreateTemplate))
	// DISABLED: mux.HandleFunc("/api/mail/templates/", withAuth(handleTemplateRoutes))
	mux.HandleFunc("GET /api/mail/logs", withAuth(handleMailLogs))

	// Webhook
	// DB metrics
	mux.HandleFunc("GET /api/databases/{dbId}/metrics", withAuth(handleDatabaseMetrics))

	mux.HandleFunc("POST /api/projects/{projectId}/compose", withAuth(handleDeployCompose))

	// Webhooks (outgoing)\n	mux.HandleFunc("GET /api/projects/{projectId}/webhooks", withAuth(handleListWebhooks))\n	mux.HandleFunc("POST /api/projects/{projectId}/webhooks", withAuth(handleCreateWebhook))\n	mux.HandleFunc("DELETE /api/projects/{projectId}/webhooks/{webhookId}", withAuth(handleDeleteWebhook))
	// Billing & Plans
        mux.HandleFunc("GET /api/plans", handleListPlans)
        mux.HandleFunc("GET /api/billing/usage", withAuth(handleGetUsage))
        mux.HandleFunc("POST /api/billing/subscribe", withAuth(handleSubscribe))

	mux.HandleFunc("GET /api/projects/{projectId}/cron", withAuth(handleListCronJobs))
	mux.HandleFunc("POST /api/projects/{projectId}/cron", withAuth(handleCreateCronJob))
	mux.HandleFunc("DELETE /api/projects/{projectId}/cron/{cronId}", withAuth(handleDeleteCronJob))

        mux.HandleFunc("GET /api/tokens", withAuth(handleListTokens))
        mux.HandleFunc("POST /api/tokens", withAuth(handleCreateToken))
        mux.HandleFunc("DELETE /api/tokens/{tokenId}", withAuth(handleDeleteToken))
        mux.HandleFunc("GET /api/projects/{projectId}/webhooks", withAuth(handleListWebhooks))
        mux.HandleFunc("POST /api/projects/{projectId}/webhooks", withAuth(handleCreateWebhook))
        mux.HandleFunc("DELETE /api/projects/{projectId}/webhooks/{webhookId}", withAuth(handleDeleteWebhook))

	mux.HandleFunc("POST /api/billing/checkout", withAuth(handleBillingCheckout))
	mux.HandleFunc("POST /api/billing/webhook", handleBillingWebhook)

        mux.HandleFunc("POST /api/webhook/github", handleWebhook)

	// Metrics
	mux.HandleFunc("GET /api/metrics", handleMetrics)
	mux.HandleFunc("GET /api/metrics/prometheus", handlePrometheus)
	mux.HandleFunc("GET /api/projects/{projectId}/metrics/history", withAuth(handleProjectMetricsHistory))

	// Domains
	mux.HandleFunc("GET /api/projects/{projectId}/domains", withAuth(handleListDomains))
	mux.HandleFunc("POST /api/projects/{projectId}/domains", withAuth(handleAddDomain))
	mux.HandleFunc("DELETE /api/projects/{projectId}/domains/{domainId}", withAuth(handleDeleteDomain))
	mux.HandleFunc("POST /api/projects/{projectId}/domains/{domainId}/verify", withAuth(handleVerifyDomain))

	// Volumes
	mux.HandleFunc("GET /api/projects/{projectId}/volumes", withAuth(handleListVolumes))
	mux.HandleFunc("POST /api/projects/{projectId}/volumes", withAuth(handleCreateVolume))
	mux.HandleFunc("DELETE /api/projects/{projectId}/volumes/{volumeId}", withAuth(handleDeleteVolume))

	// Env vars
	mux.HandleFunc("GET /api/projects/{projectId}/envs", withAuth(handleListEnvVars))
	mux.HandleFunc("POST /api/projects/{projectId}/envs", withAuth(handleCreateEnvVar))
	mux.HandleFunc("PATCH /api/projects/{projectId}/envs/{envID}", withAuth(handleUpdateEnvVar))
	mux.HandleFunc("DELETE /api/projects/{projectId}/envs/{envID}", withAuth(handleDeleteEnvVar))

	// Rollback
	mux.HandleFunc("POST /api/projects/{projectId}/deployments/{deploymentId}/rollback", withAuth(handleRollback))

	// WebSocket for real-time logs
	mux.HandleFunc("GET /api/ws/deployments/{id}/logs", handleWebSocketLogs)

	handler := corsMiddleware(requestIDMiddleware(loggingMiddleware(mux)))

	port := getEnv("API_PORT", "3001")
	log.Printf("StackRun API v%s starting on :%s (Go %s, %d goroutines)", Version, port, runtime.Version(), runtime.NumCPU())

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
	allowedOrigins := strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000"), ",")
	allowedMap := make(map[string]bool)
	for _, o := range allowedOrigins {
		allowedMap[strings.TrimSpace(o)] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedMap[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", strings.TrimSpace(allowedOrigins[0]))
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-request-id")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
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
		// Dashboard SSR bypass: internal token
		dashToken := getEnv("DASHBOARD_TOKEN", "")
		if dashToken != "" && r.Header.Get("X-Dashboard-Token") == dashToken {
			ctx := context.WithValue(r.Context(), "userID", "d780231d-3f40-47bb-8bf7-a2b762998325")
			next(w, r.WithContext(ctx))
			return
		}

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
		"name":         "stackrun-control-plane",
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
	if body.Email == "" || body.Password == "" {
		jsonError(w, "Email e password sao obrigatorios", http.StatusBadRequest)
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
		redirectURI = "https://api.stackrun.vercel.app/api/auth/github/callback"
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
		redirectURI = "https://api.stackrun.vercel.app/api/auth/github/callback"
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
		frontendURL = "https://app.stackrun.vercel.app"
	}
	http.Redirect(w, r, fmt.Sprintf("%s/login?token=%s", frontendURL, token), http.StatusFound)
}

// ─── Projects ───────────────────────────────────────────────────────────────

func handleListProjects(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)

	rows, err := db.QueryContext(r.Context(),
		`SELECT id, name, slug, framework, status, domain, repo_url, env_vars, port, created_at
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
		var port int
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &slug, &framework, &status, &domain, &repoURL, &envVars, &port, &createdAt); err != nil {
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
			"port":      port,
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

	re := regexp.MustCompile("<[^>]*>")
	body.Name = re.ReplaceAllString(body.Name, "")
	body.Name = strings.TrimSpace(body.Name)
	if len(body.Name) > 100 {
		body.Name = body.Name[:100]
	}

	// Auto-assign template repo if none provided
	if body.RepoURL == "" {
		tmpl := "static"
		if body.Framework == "express" || body.Framework == "nodejs" {
			tmpl = "express"
		}
		body.RepoURL = "/root/stackrun-repos/template-" + tmpl + ".git"
		if body.Framework == "" {
			body.Framework = tmpl
		}
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
	var port int
	err := db.QueryRowContext(r.Context(),
		`INSERT INTO projects (id, name, slug, user_id, framework, repo_url, status, port, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, nullif($5,''), nullif($6,''), 'ACTIVE', (SELECT COALESCE(MAX(port), 8081) + 1 FROM projects WHERE port BETWEEN 8082 AND 8181), $7, $7)
		 RETURNING id, name, slug, framework, status, domain, repo_url, env_vars, created_at, port`,
		id, body.Name, slug, userID, body.Framework, body.RepoURL, now,
	).Scan(&id, &body.Name, &slug, &framework, &status, &domain, &repoURL, &envVars, &createdAt, &port)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			jsonError(w, "Slug ja existe. Escolha outro nome.", http.StatusConflict)
		} else {
			jsonError(w, "Erro ao criar projeto", http.StatusInternalServerError)
		}
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

	// Validate UUID format for project ID
	if len(parts) > 0 && parts[0] != "" {
		if _, err := uuid.Parse(parts[0]); err != nil {
			jsonError(w, "ID de projeto invalido", http.StatusBadRequest)
			return
		}
	}

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
		if r.Method == "DELETE" {
			handleDeleteProject(w, r, parts[0])
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

	var name, slug, pStatus, branch string
	var framework, domain, repoURL, envVars sql.NullString
	var port int
	var createdAt time.Time
	err := db.QueryRowContext(r.Context(),
		`SELECT name, slug, framework, status, domain, repo_url, env_vars, branch, port, created_at
		 FROM projects WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&name, &slug, &framework, &pStatus, &domain, &repoURL, &envVars, &branch, &port, &createdAt)
	if err == sql.ErrNoRows {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
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
		"branch":    branch,
		"port":      port,
		"createdAt": createdAt.Format(time.RFC3339),
	})
}

func handleUpdateProject(w http.ResponseWriter, r *http.Request, id string) {
	userID := r.Context().Value("userID").(string)

	var body struct {
		EnvVars   string `json:"envVars"`
		Domain    string `json:"domain"`
		RepoURL   string `json:"repoUrl"`
		Framework string `json:"framework"`
		Branch    string `json:"branch"`
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
	if body.Framework != "" {
		sets = append(sets, fmt.Sprintf("framework = $%d", argIdx))
		args = append(args, body.Framework)
		argIdx++
	}
	if body.Branch != "" {
		sets = append(sets, fmt.Sprintf("branch = $%d", argIdx))
		args = append(args, body.Branch)
		argIdx++
	}

	if len(sets) == 0 {
		handleGetProject(w, r, id)
		return
	}

	// Use SQLite-compatible timestamp
	sets = append(sets, "updated_at = datetime('now')")
	args = append(args, id, userID)

	query := fmt.Sprintf("UPDATE projects SET %s WHERE id = $%d AND user_id = $%d RETURNING id, name, slug, framework, status, domain, repo_url, env_vars, branch, created_at",
		strings.Join(sets, ", "), argIdx, argIdx+1)

	var name, slug, pStatus, branch string
	var framework, domain, repoURL, envVars sql.NullString
	var createdAt time.Time
	err := db.QueryRowContext(r.Context(), query, args...).Scan(&id, &name, &slug, &framework, &pStatus, &domain, &repoURL, &envVars, &branch, &createdAt)
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
		"branch":    branch,
		"createdAt": createdAt.Format(time.RFC3339),
	})
}

func handleDeleteProject(w http.ResponseWriter, r *http.Request, id string) {
	userID := r.Context().Value("userID").(string)

	var exists bool
	db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND user_id = $2)", id, userID).Scan(&exists)
	if !exists {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	// Stop and remove Docker container
	var slug string
	db.QueryRowContext(r.Context(), "SELECT slug FROM projects WHERE id = $1", id).Scan(&slug)
	exec.Command("docker", "rm", "-f", "stackrun-"+slug).Run()

	// Delete related records
	db.ExecContext(r.Context(), "DELETE FROM domains WHERE project_id = $1", id)
	db.ExecContext(r.Context(), "DELETE FROM deployments WHERE project_id = $1", id)
	db.ExecContext(r.Context(), "DELETE FROM databases WHERE project_id = $1", id)
	db.ExecContext(r.Context(), "DELETE FROM projects WHERE id = $1 AND user_id = $2", id, userID)

	log.Printf("[project] Deleted project %s (%s)", id, slug)
	jsonResponse(w, map[string]interface{}{"success": true})
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
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
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

// ── Env Vars CRUD ─────────────────────────────────────────────────────

func handleListEnvVars(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")

	var projectUserID string
	err := db.QueryRow("SELECT user_id FROM projects WHERE id = $1", projectID).Scan(&projectUserID)
	if err != nil || projectUserID != userID {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	rows, err := db.Query("SELECT id, key, value, secret, created_at FROM project_env_vars WHERE project_id = $1 ORDER BY key", projectID)
	if err != nil {
		jsonError(w, "Erro ao listar env vars", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var envs []map[string]interface{}
	for rows.Next() {
		var id, key, value string
		var secret bool
		var createdAt time.Time
		rows.Scan(&id, &key, &value, &secret, &createdAt)
		if secret {
			value = "********"
		}
		envs = append(envs, map[string]interface{}{
			"id":        id,
			"key":       key,
			"value":     value,
			"secret":    secret,
			"createdAt": createdAt.Format(time.RFC3339),
		})
	}
	if envs == nil {
		envs = []map[string]interface{}{}
	}
	jsonResponse(w, map[string]interface{}{"envs": envs}, http.StatusOK)
}

func handleCreateEnvVar(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")

	var projectUserID string
	err := db.QueryRow("SELECT user_id FROM projects WHERE id = $1", projectID).Scan(&projectUserID)
	if err != nil || projectUserID != userID {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	var body struct {
		Key    string `json:"key"`
		Value  string `json:"value"`
		Secret bool   `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Body invalido", http.StatusBadRequest)
		return
	}
	if body.Key == "" || body.Value == "" {
		jsonError(w, "key e value sao obrigatorios", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	_, err = db.Exec(
		`INSERT INTO project_env_vars (id, project_id, key, value, secret) VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (project_id, key) DO UPDATE SET value = $4, secret = $5`,
		id, projectID, body.Key, body.Value, body.Secret,
	)
	if err != nil {
		jsonError(w, "Erro ao criar env var", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{"id": id, "key": body.Key, "secret": body.Secret}, http.StatusCreated)
}

func handleUpdateEnvVar(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")
	envID := r.PathValue("envID")

	var projectUserID string
	err := db.QueryRow("SELECT user_id FROM projects WHERE id = $1", projectID).Scan(&projectUserID)
	if err != nil || projectUserID != userID {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	var body struct {
		Value  string `json:"value"`
		Secret bool   `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Body invalido", http.StatusBadRequest)
		return
	}

	result, err := db.Exec(
		"UPDATE project_env_vars SET value = $1, secret = $2 WHERE id = $3 AND project_id = $4",
		body.Value, body.Secret, envID, projectID,
	)
	if err != nil {
		jsonError(w, "Erro ao atualizar env var", http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		jsonError(w, "Env var nao encontrada", http.StatusNotFound)
		return
	}

	jsonResponse(w, map[string]interface{}{"ok": true}, http.StatusOK)
}

func handleDeleteEnvVar(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")
	envID := r.PathValue("envID")

	var projectUserID string
	err := db.QueryRow("SELECT user_id FROM projects WHERE id = $1", projectID).Scan(&projectUserID)
	if err != nil || projectUserID != userID {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	result, err := db.Exec(
		"DELETE FROM project_env_vars WHERE id = $1 AND project_id = $2",
		envID, projectID,
	)
	if err != nil {
		jsonError(w, "Erro ao deletar env var", http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		jsonError(w, "Env var nao encontrada", http.StatusNotFound)
		return
	}

	jsonResponse(w, map[string]interface{}{"ok": true}, http.StatusOK)
}

func handleDeploy(w http.ResponseWriter, r *http.Request, projectID string) {
	userID := r.Context().Value("userID").(string)

	var name, slug, repoURL sql.NullString
	var branch string
	var projectPort int
	err := db.QueryRowContext(r.Context(),
		"SELECT name, slug, repo_url, branch, port FROM projects WHERE id = $1 AND user_id = $2", projectID, userID,
	).Scan(&name, &slug, &repoURL, &branch, &projectPort)
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

	// Validate repo URL
	if !repoURL.Valid || repoURL.String == "" {
		jsonError(w, "Configure o repositorio Git antes de fazer deploy", http.StatusBadRequest)
		return
	}

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
		imageTag := fmt.Sprintf("stackrun-%s:%s", slug.String, branch)
		if branch == "main" {
			imageTag = fmt.Sprintf("stackrun-%s:latest", slug.String)
		}
		containerName := fmt.Sprintf("stackrun-%s", slug.String)
		if branch != "main" {
			containerName = fmt.Sprintf("stackrun-%s-preview-%s", slug.String, sanitizeBranch(branch))
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
			"port":          projectPort,
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
	containerName := "stackrun-" + slug
	if branch != "" && branch != "main" {
		containerName = "stackrun-" + slug + "-preview-" + sanitizeBranch(branch)
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
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
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

	// Use unicode for cleaner rune check
	dbName := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return r
		}
		return '_'
	}, strings.ToLower(body.Name))
	dbName = "stackrun_" + dbName
	password := generatePassword(16)

	// Create database using system PostgreSQL
	dbCreate := exec.Command("sudo", "-u", "postgres", "createdb", dbName)
	dbCreate.Run()

	// Create user and grant privileges
	userCreate := exec.Command("sudo", "-u", "postgres", "psql", "-c",
		fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", dbName, password))
	userCreate.Run()

	grantDB := exec.Command("sudo", "-u", "postgres", "psql", "-c",
		fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s", dbName, dbName))
	grantDB.Run()

	// Grant schema permissions (requires connecting to the database)
	grantSchema := exec.Command("sudo", "-u", "postgres", "psql", "-d", dbName, "-c",
		fmt.Sprintf("GRANT ALL ON SCHEMA public TO %s", dbName))
	grantSchema.Run()

	grantTables := exec.Command("sudo", "-u", "postgres", "psql", "-d", dbName, "-c",
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO %s", dbName))
	grantTables.Run()

	connURL := fmt.Sprintf("postgresql://%s:%s@localhost:5432/%s", dbName, password, dbName)

	var id string
	var createdAt time.Time
	id = uuid.New().String()
	var err error
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

	// Grant remaining permissions via sudo
	exec.Command("sudo", "-u", "postgres", "psql", "-d", dbName, "-c",
		fmt.Sprintf("GRANT ALL ON ALL TABLES IN SCHEMA public TO %s", dbName)).Run()
	exec.Command("sudo", "-u", "postgres", "psql", "-d", dbName, "-c",
		fmt.Sprintf("GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO %s", dbName)).Run()

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

	dbName := "stackrun_" + name
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
		imageTag := fmt.Sprintf("stackrun-%s:%s", slug, safeBranch)
		containerName := fmt.Sprintf("stackrun-%s", slug)
		if branch != defaultBranch {
			imageTag = fmt.Sprintf("stackrun-%s:preview-%s", slug, safeBranch)
			containerName = fmt.Sprintf("stackrun-%s-preview-%s", slug, safeBranch)
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

// ─── Project Metrics History (Prometheus) ──────────────────────────────────

func handleProjectMetricsHistory(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	userID := r.Context().Value("userID").(string)

	var slug string
	err := db.QueryRowContext(r.Context(),
		"SELECT slug FROM projects WHERE id = $1 AND user_id = $2", projectID, userID,
	).Scan(&slug)
	if err != nil {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	containerName := "stackrun-" + slug
	branch := r.URL.Query().Get("branch")
	if branch != "" && branch != "main" {
		containerName = "stackrun-" + slug + "-preview-" + sanitizeBranch(branch)
	}

	range_ := r.URL.Query().Get("range")
	if range_ == "" {
		range_ = "1h"
	}

	type metricPoint struct {
		Time  float64 `json:"t"`
		Value float64 `json:"v"`
	}

	result := map[string]interface{}{
		"container": containerName,
		"cpu":       []metricPoint{},
		"memory":    []metricPoint{},
	}

	now := time.Now()
	start := now.Add(-parseDuration(range_))

	// Query Prometheus for CPU
	// Use url.Values for proper encoding
	cpuParams := url.Values{}
	cpuParams.Set("query", "sum(rate(container_cpu_usage_seconds_total{name=~\""+containerName+".*\"}[1m]))")
	cpuParams.Set("start", strconv.FormatInt(start.Unix(), 10))
	cpuParams.Set("end", strconv.FormatInt(now.Unix(), 10))
	cpuParams.Set("step", "60")
	cpuResp, err := http.Get("http://localhost:9090/api/v1/query_range?" + cpuParams.Encode())
	if err == nil && cpuResp.StatusCode == 200 {
		defer cpuResp.Body.Close()
		var cpuData map[string]interface{}
		if json.NewDecoder(cpuResp.Body).Decode(&cpuData) == nil {
			if data, ok := cpuData["data"].(map[string]interface{}); ok {
				if results, ok := data["result"].([]interface{}); ok && len(results) > 0 {
					if r0, ok := results[0].(map[string]interface{}); ok {
						if values, ok := r0["values"].([]interface{}); ok {
							cpuPoints := []metricPoint{}
							for _, v := range values {
								if pair, ok := v.([]interface{}); ok && len(pair) == 2 {
									t, _ := pair[0].(float64)
									vs := fmt.Sprintf("%%v", pair[1])
									val, _ := strconv.ParseFloat(vs, 64)
									cpuPoints = append(cpuPoints, metricPoint{Time: t, Value: val * 100})
								}
							}
							result["cpu"] = cpuPoints
						}
					}
				}
			}
		}
	}

	// Query Prometheus for Memory
	memParams := url.Values{}
	memParams.Set("query", "container_memory_usage_bytes{name=~\""+containerName+".*\"}")
	memParams.Set("start", strconv.FormatInt(start.Unix(), 10))
	memParams.Set("end", strconv.FormatInt(now.Unix(), 10))
	memParams.Set("step", "60")
	memResp, err := http.Get("http://localhost:9090/api/v1/query_range?" + memParams.Encode())
	if err == nil && memResp.StatusCode == 200 {
		defer memResp.Body.Close()
		var memData map[string]interface{}
		if json.NewDecoder(memResp.Body).Decode(&memData) == nil {
			if data, ok := memData["data"].(map[string]interface{}); ok {
				if results, ok := data["result"].([]interface{}); ok && len(results) > 0 {
					if r0, ok := results[0].(map[string]interface{}); ok {
						if values, ok := r0["values"].([]interface{}); ok {
							memPoints := []metricPoint{}
							for _, v := range values {
								if pair, ok := v.([]interface{}); ok && len(pair) == 2 {
									t, _ := pair[0].(float64)
									vs := fmt.Sprintf("%%v", pair[1])
									val, _ := strconv.ParseFloat(vs, 64)
									memPoints = append(memPoints, metricPoint{Time: t, Value: val / 1048576})
								}
							}
							result["memory"] = memPoints
						}
					}
				}
			}
		}
	}

	jsonResponse(w, result)
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Hour
	}
	return d
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
			"heapAlloc":  float64(m.HeapAlloc) / 1048576,
			"heapTotal":  float64(m.HeapSys) / 1048576,
			"heapInuse":  float64(m.HeapInuse) / 1048576,
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
	fmt.Fprintf(w, `# HELP go_info Go runtime info
# TYPE go_info gauge
go_info{version="%s"} 1

# HELP go_goroutines Number of goroutines
# TYPE go_goroutines gauge
go_goroutines %d

# HELP go_memstats_alloc_bytes Allocated memory bytes
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes %d

# HELP go_memstats_sys_bytes System memory bytes
# TYPE go_memstats_sys_bytes gauge
go_memstats_sys_bytes %d

# HELP stackrun_uptime_seconds Uptime in seconds
# TYPE stackrun_uptime_seconds gauge
stackrun_uptime_seconds %.2f

# HELP stackrun_db_open_connections Database open connections
# TYPE stackrun_db_open_connections gauge
stackrun_db_open_connections %d

# HELP stackrun_memory_heap_used_bytes Heap memory used
# TYPE stackrun_memory_heap_used_bytes gauge
stackrun_memory_heap_used_bytes %d

# HELP stackrun_memory_rss_bytes Resident set size
# TYPE stackrun_memory_rss_bytes gauge
stackrun_memory_rss_bytes %d
`, runtime.Version(), runtime.NumGoroutine(), m.Alloc, m.Sys,
		time.Since(startTime).Seconds(), db.Stats().OpenConnections, m.HeapInuse, m.Sys)
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
	nidusIP := getEnv("STACKRUN_SERVER_IP", "2.24.204.31")
	txtValue := fmt.Sprintf("stackrun-verify=%s", projectSlug)
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
	cmd := exec.Command("dig", "+short", "TXT", "_stackrun-verify."+domain)
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

	host := getEnv("STACKRUN_HOST", "localhost")
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

// ─── Mail ───────────────────────────────────────────────────────────────────

func handleSendMail(w http.ResponseWriter, r *http.Request) {
	var input mail.SendInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonError(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if input.ToEmail == "" {
		jsonError(w, "to_email is required", http.StatusBadRequest)
		return
	}

	result, err := mail.Send(input)
	if err != nil {
		jsonError(w, fmt.Sprintf("Failed to send: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := mail.GetTemplates()
	if err != nil {
		jsonError(w, "Failed to list templates", http.StatusInternalServerError)
		return
	}
	if templates == nil {
		templates = []mail.TemplateData{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

func handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	// GET /api/mail/templates/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/mail/templates/")
	if path == "" || path == r.URL.Path {
		http.NotFound(w, r)
		return
	}
	id := strings.Split(path, "/")[0]

	var t mail.TemplateData
	err := db.QueryRow("SELECT id, name, subject, html_body, text_body, created_at FROM email_templates WHERE id = $1", id).
		Scan(&t.ID, &t.Name, &t.Subject, &t.HTMLBody, &t.TextBody, &t.CreatedAt)
	if err != nil {
		jsonError(w, "Template not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var t mail.TemplateData
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		jsonError(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if t.Name == "" || t.Subject == "" {
		jsonError(w, "name and subject are required", http.StatusBadRequest)
		return
	}

	// Generate ID from name
	t.ID = strings.ToLower(strings.ReplaceAll(t.Name, " ", "-"))
	
	now := time.Now()
	_, err := db.Exec(`INSERT INTO email_templates (id, name, subject, html_body, text_body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET name=$2, subject=$3, html_body=$4, text_body=$5, updated_at=$7`,
		t.ID, t.Name, t.Subject, t.HTMLBody, t.TextBody, now, now)
	if err != nil {
		jsonError(w, fmt.Sprintf("Failed to create template: %v", err), http.StatusInternalServerError)
		return
	}

	t.CreatedAt = now.Format(time.RFC3339)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/mail/templates/")
	if path == "" || path == r.URL.Path {
		http.NotFound(w, r)
		return
	}
	id := strings.Split(path, "/")[0]

	_, err := db.Exec("DELETE FROM email_templates WHERE id = $1", id)
	if err != nil {
		jsonError(w, "Failed to delete template", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleMailLogs(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, template_id, to_email, subject, status, provider, error_message, sent_at, created_at FROM email_logs ORDER BY created_at DESC LIMIT 50")
	if err != nil {
		jsonError(w, "Failed to list logs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type LogEntry struct {
		ID           string  `json:"id"`
		TemplateID   *string `json:"template_id"`
		ToEmail      string  `json:"to"`
		Subject      string  `json:"subject"`
		Status       string  `json:"status"`
		Provider     string  `json:"provider"`
		ErrorMessage *string `json:"error,omitempty"`
		SentAt       *string `json:"sent_at"`
		CreatedAt    string  `json:"created_at"`
	}

	var logs []LogEntry
	for rows.Next() {
		var l LogEntry
		var sentAt, createdAt time.Time
		var tmplID, errMsg sql.NullString
		if err := rows.Scan(&l.ID, &tmplID, &l.ToEmail, &l.Subject, &l.Status, &l.Provider, &errMsg, &sentAt, &createdAt); err != nil {
			continue
		}
		if tmplID.Valid {
			l.TemplateID = &tmplID.String
		}
		if errMsg.Valid {
			l.ErrorMessage = &errMsg.String
		}
		s := sentAt.Format(time.RFC3339)
		l.SentAt = &s
		l.CreatedAt = createdAt.Format(time.RFC3339)
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []LogEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}


func handleTemplateRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetTemplate(w, r)
	case http.MethodDelete:
		handleDeleteTemplate(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// ─── RBAC Middleware ──────────────────────────────────────────────────────

func withAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID")
		if userID == nil {
			jsonError(w, "Nao autenticado", http.StatusUnauthorized)
			return
		}
		var role string
		db.QueryRow("SELECT role FROM users WHERE id = $1", userID).Scan(&role)
		if role != "admin" {
			jsonError(w, "Acesso restrito a administradores", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func auditLog(userID, action, resource, resourceID, ip string, details map[string]interface{}) {
	detailsJSON, _ := json.Marshal(details)
	db.Exec(
		"INSERT INTO audit_logs (user_id, action, resource, resource_id, details, ip_address) VALUES ($1, $2, $3, $4, $5, $6)",
		userID, action, resource, resourceID, string(detailsJSON), ip,
	)
}

// ─── DB Metrics ──────────────────────────────────────────────────────────

func handleDatabaseMetrics(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	id := r.PathValue("dbId")

	var exists bool
	db.QueryRow("SELECT EXISTS(SELECT 1 FROM databases d JOIN projects p ON d.project_id=p.id WHERE d.id=$1 AND p.user_id=$2)", id, userID).Scan(&exists)
	if !exists {
		jsonError(w, "Banco nao encontrado", http.StatusNotFound)
		return
	}

	metrics := map[string]interface{}{}

	// DB size
	var dbSize string
	db.QueryRow("SELECT pg_size_pretty(pg_database_size(current_database()))").Scan(&dbSize)
	metrics["size"] = dbSize

	// Active connections
	var conns int
	db.QueryRow("SELECT count(*) FROM pg_stat_activity WHERE state = active").Scan(&conns)
	metrics["activeConnections"] = conns

	// Total connections
	var totalConns int
	db.QueryRow("SELECT count(*) FROM pg_stat_activity").Scan(&totalConns)
	metrics["totalConnections"] = totalConns

	// Cache hit ratio
	var cacheHit float64
	db.QueryRow("SELECT COALESCE(sum(heap_blks_hit)*100.0 / nullif(sum(heap_blks_hit)+sum(heap_blks_read),0), 100.0) FROM pg_statio_user_tables").Scan(&cacheHit)
	metrics["cacheHitRatio"] = math.Round(cacheHit*10) / 10

	// Transaction rate (last minute)
	var tps int
	db.QueryRow("SELECT count(*) FROM pg_stat_database WHERE datname=current_database()").Scan(&tps)
	metrics["databases"] = tps

	jsonResponse(w, metrics)
}

// ─── Admin routes registration patch ────────────────────────────────────
// (registered below in existing route setup)


// ─── Docker Compose ──────────────────────────────────────────────────────

func handleDeployCompose(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")

	var body struct{ Compose string `json:"compose"` }
	json.NewDecoder(r.Body).Decode(&body)
	if body.Compose == "" { jsonError(w, "compose yaml obrigatorio", http.StatusBadRequest); return }

	var projectSlug string
	db.QueryRow("SELECT slug FROM projects WHERE id=$1 AND user_id=$2", projectID, userID).Scan(&projectSlug)
	if projectSlug == "" { jsonError(w, "Projeto nao encontrado", http.StatusNotFound); return }

	deployID := uuid.New().String()
	db.Exec("INSERT INTO deployments (id, project_id, branch, type, status) VALUES ($1,$2,compose,compose,queued)", deployID, projectID)

	jobData, _ := json.Marshal(map[string]interface{}{
		"deploymentId": deployID,
		"projectId": projectID,
		"projectSlug": projectSlug,
		"compose": body.Compose,
		"type": "compose",
	})

	if rdb != nil {
		jobID := fmt.Sprintf("%d", time.Now().UnixNano())
		rdb.LPush(context.Background(), "bull:deploy-queue:wait", jobID)
		rdb.HSet(context.Background(), "bull:deploy-queue:"+jobID, "data", string(jobData))
	}

	jsonResponse(w, map[string]interface{}{
		"deploymentId": deployID, "status": "queued", "message": "Compose deploy enfileirado",
	}, http.StatusAccepted)
}


// ─── Webhooks & Notifications ────────────────────────────────────────────

func sendWebhookNotification(projectName, projectSlug, status, url, branch string) {
	rows, err := db.Query("SELECT url, secret FROM webhooks WHERE project_id IN (SELECT id FROM projects WHERE slug=$1) AND events @> ARRAY[$2]::TEXT[] AND status=active", projectSlug, status)
	if err != nil { return }
	defer rows.Close()
	for rows.Next() {
		var webhookURL, secret string
		rows.Scan(&webhookURL, &secret)
		payload := map[string]interface{}{
			"project": projectName,
			"slug": projectSlug,
			"status": status,
			"url": url,
			"branch": branch,
			"timestamp": time.Now().Format(time.RFC3339),
		}
		body, _ := json.Marshal(payload)
		go func(url string, b []byte) {
			req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			http.DefaultClient.Do(req)
		}(webhookURL, body)
	}
}

// Discord webhook support (converts to Discord embed format)
func sendDiscordNotification(projectName, projectSlug, status, url, branch string, webhookURL string) {
	color := 0x22d3ee
	switch status {
	case "depoy.success": color = 0x22c55e
	case "depoy.failed": color = 0xef4444
	}
	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{{
			"title": fmt.Sprintf("Deploy %s — %s", status, projectName),
			"description": fmt.Sprintf("Branch: **%s**\nURL: %s", branch, url),
			"color": color,
			"timestamp": time.Now().Format(time.RFC3339),
		}},
	}
	body, _ := json.Marshal(payload)
	go func(u string, b []byte) {
		http.Post(u, "application/json", bytes.NewReader(b))
	}(webhookURL, body)
}

func handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	rows, err := db.Query("SELECT id, url, events, status, created_at FROM webhooks WHERE project_id=$1 ORDER BY created_at DESC", projectID)
	if err != nil { jsonError(w, "Erro", http.StatusInternalServerError); return }
	defer rows.Close()
	webhooks := []map[string]interface{}{}
	for rows.Next() {
		var id, url, status string
		var events []string
		var createdAt time.Time
		rows.Scan(&id, &url, &events, &status, &createdAt)
		webhooks = append(webhooks, map[string]interface{}{
			"id": id, "url": url, "events": events, "status": status,
			"createdAt": createdAt.Format(time.RFC3339),
		})
	}
	jsonResponse(w, webhooks)
}

func handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	var body struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.URL == "" { jsonError(w, "URL obrigatoria", http.StatusBadRequest); return }
	if len(body.Events) == 0 { body.Events = []string{"deploy.success", "deploy.failed"} }
	id := uuid.New().String()
	db.Exec("INSERT INTO webhooks (id, project_id, url, events) VALUES ($1,$2,$3,$4)", id, projectID, body.URL, body.Events)
	jsonResponse(w, map[string]interface{}{"id": id, "url": body.URL, "events": body.Events}, http.StatusCreated)
}

func handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectId")
	webhookID := r.PathValue("webhookId")
	db.Exec("DELETE FROM webhooks WHERE id=$1 AND project_id=$2", webhookID, projectID)
	jsonResponse(w, map[string]interface{}{"ok": true})
}


// ─── Volumes ──────────────────────────────────────────────────────────────

func handleListVolumes(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")

	var projectUserID string
	err := db.QueryRow("SELECT user_id FROM projects WHERE id = $1", projectID).Scan(&projectUserID)
	if err != nil || projectUserID != userID {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	rows, err := db.QueryContext(r.Context(), `SELECT id, name, mount_path, size_mb, created_at FROM project_volumes WHERE project_id = $1 ORDER BY created_at`, projectID)
	if err != nil {
		jsonError(w, "Erro ao listar volumes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	volumes := []map[string]interface{}{}
	for rows.Next() {
		var id, name, mountPath string
		var sizeMb int
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &mountPath, &sizeMb, &createdAt); err != nil {
			continue
		}
		volumes = append(volumes, map[string]interface{}{
			"id":        id,
			"name":      name,
			"mountPath": mountPath,
			"sizeMb":    sizeMb,
			"createdAt": createdAt.Format(time.RFC3339),
		})
	}
	if volumes == nil {
		volumes = []map[string]interface{}{}
	}
	jsonResponse(w, volumes)
}

func handleCreateVolume(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")

	var projectUserID string
	err := db.QueryRow("SELECT user_id FROM projects WHERE id = $1", projectID).Scan(&projectUserID)
	if err != nil || projectUserID != userID {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	var body struct {
		Name      string `json:"name"`
		MountPath string `json:"mountPath"`
		SizeMb    int    `json:"sizeMb"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "Body invalido", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.MountPath == "" {
		jsonError(w, "name e mountPath sao obrigatorios", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	_, err = db.Exec(
		`INSERT INTO project_volumes (id, project_id, name, mount_path, size_mb)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, projectID, body.Name, body.MountPath, body.SizeMb,
	)
	if err != nil {
		jsonError(w, "Erro ao criar volume: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"id":        id,
		"name":      body.Name,
		"mountPath": body.MountPath,
		"sizeMb":    body.SizeMb,
	}, http.StatusCreated)
}

func handleDeleteVolume(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	projectID := r.PathValue("projectId")
	volumeID := r.PathValue("volumeId")

	var projectUserID string
	err := db.QueryRow("SELECT user_id FROM projects WHERE id = $1", projectID).Scan(&projectUserID)
	if err != nil || projectUserID != userID {
		jsonError(w, "Projeto nao encontrado", http.StatusNotFound)
		return
	}

	result, err := db.Exec(
		"DELETE FROM project_volumes WHERE id = $1 AND project_id = $2",
		volumeID, projectID,
	)
	if err != nil {
		jsonError(w, "Erro ao deletar volume", http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		jsonError(w, "Volume nao encontrado", http.StatusNotFound)
		return
	}

	jsonResponse(w, map[string]interface{}{"ok": true})
}


// --- Billing ---

func handleListPlans(w http.ResponseWriter, r *http.Request) {
        rows, err := db.Query("SELECT id, name, price_cents, max_projects, max_databases, features FROM plans ORDER BY price_cents")
        if err != nil {
                log.Printf("handleListPlans query error: %v", err)
                jsonError(w, "Erro ao listar planos", http.StatusInternalServerError)
                return
        }
        defer rows.Close()
        var plans []map[string]interface{}
        for rows.Next() {
                var id, name string
                var features []byte
                var price, maxProj, maxDB int
                if err := rows.Scan(&id, &name, &price, &maxProj, &maxDB, &features); err != nil {
                        log.Printf("handleListPlans scan error: %v", err)
                        jsonError(w, "Erro ao ler planos", http.StatusInternalServerError)
                        return
                }
                plans = append(plans, map[string]interface{}{
                        "id":          id,
                        "name":        name,
                        "priceCents":  price,
                        "maxProjects": maxProj,
                        "maxDatabases": maxDB,
                        "features":    string(features),
                })
        }
        jsonResponse(w, plans)
}

func handleGetUsage(w http.ResponseWriter, r *http.Request) {
        userID := r.Context().Value("userID").(string)
        month := time.Now().Format("2006-01")
        row := db.QueryRow("SELECT deploys, projects, databases FROM usage_metrics WHERE user_id=$1 AND month=$2", userID, month)
        var deploys, projects, dbs int
        row.Scan(&deploys, &projects, &dbs)
        var planID string
        db.QueryRow("SELECT COALESCE(plan_id,'free') FROM users WHERE id=$1", userID).Scan(&planID)
        jsonResponse(w, map[string]interface{}{
                "month": month, "deploys": deploys, "projects": projects, "databases": dbs, "plan": planID,
        })
}

func handleSubscribe(w http.ResponseWriter, r *http.Request) {
        userID := r.Context().Value("userID").(string)
        var body struct {
                PlanID string `json:"planId"`
        }
        json.NewDecoder(r.Body).Decode(&body)
        if body.PlanID == "" {
                body.PlanID = "pro"
        }
        db.Exec("UPDATE users SET plan_id=$1 WHERE id=$2", body.PlanID, userID)
        db.Exec("INSERT INTO subscriptions (user_id, plan_id, current_period_start, current_period_end) VALUES ($1,$2,NOW(),NOW()+INTERVAL '30 days') ON CONFLICT DO NOTHING", userID, body.PlanID)
        jsonResponse(w, map[string]interface{}{"plan": body.PlanID, "status": "active"})
}
