package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()
	os.Setenv("DATABASE_URL", "sqlite://:memory:")
	os.Setenv("JWT_SECRET", "test_secret_stackrun_2026")
	os.Setenv("REDIS_URL", "")

	loadEnv()
	jwtSecret = []byte("test_secret_stackrun_2026")

	var err error
	db, err = sql.Open("sqlite3", ":memory:?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	if err := initSQLite(db); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}
	return func() {
		db.Close()
	}
}

func setupTestRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /api/auth/register", handleRegister)
	mux.HandleFunc("POST /api/auth/login", handleLogin)
	mux.HandleFunc("GET /api/auth/me", withAuth(handleMe))
	mux.HandleFunc("/api/projects/", withAuth(handleProjectRoutes))
	mux.HandleFunc("POST /api/projects", withAuth(handleCreateProject))
	mux.HandleFunc("GET /api/projects", withAuth(handleListProjects))
	mux.HandleFunc("/api/databases/", withAuth(handleDatabaseRoutes))
	mux.HandleFunc("POST /api/databases", withAuth(handleCreateDatabase))
	mux.HandleFunc("GET /api/databases", withAuth(handleListDatabases))
	mux.HandleFunc("POST /api/webhook/github", handleWebhook)
	mux.HandleFunc("GET /api/metrics", handleMetrics)
	mux.HandleFunc("GET /api/metrics/prometheus", handlePrometheus)
	mux.HandleFunc("GET /api/projects/{projectId}/domains", withAuth(handleListDomains))
	mux.HandleFunc("POST /api/projects/{projectId}/domains", withAuth(handleAddDomain))
	mux.HandleFunc("DELETE /api/projects/{projectId}/domains/{domainId}", withAuth(handleDeleteDomain))
	mux.HandleFunc("POST /api/projects/{projectId}/domains/{domainId}/verify", withAuth(handleVerifyDomain))
	mux.HandleFunc("POST /api/projects/{projectId}/deployments/{deploymentId}/rollback", withAuth(handleRollback))

	return corsMiddleware(requestIDMiddleware(loggingMiddleware(mux)))
}

func jsonBody(v interface{}) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

func parseJSON(t *testing.T, body *bytes.Buffer) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	return result
}

func parseJSONArray(t *testing.T, body *bytes.Buffer) []interface{} {
	t.Helper()
	var result []interface{}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse JSON array: %v", err)
	}
	return result
}

func TestHealthEndpoint(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupTestRouter()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	result := parseJSON(t, w.Body)
	if result["status"] != "ok" {
		t.Errorf("Expected status ok, got %v", result["status"])
	}
	if result["name"] != "stackrun-control-plane" {
		t.Errorf("Expected stackrun-control-plane, got %v", result["name"])
	}
	if v, ok := result["version"]; !ok || v == "" {
		t.Error("Version should not be empty")
	}
}

func TestAuthFlow(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupTestRouter()

	// Register
	regBody := map[string]string{"email": "test@stackrun.dev", "name": "Test User", "password": "test123456"}
	req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(regBody))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Register: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w.Body)
	token, ok := result["token"].(string)
	if !ok || token == "" {
		t.Fatal("Register: token not returned")
	}
	if user, ok := result["user"].(map[string]interface{}); ok {
		if user["email"] != "test@stackrun.dev" {
			t.Errorf("Register: expected test@stackrun.dev, got %v", user["email"])
		}
	} else {
		t.Fatal("Register: user not returned")
	}

	// Login
	loginBody := map[string]string{"email": "test@stackrun.dev", "password": "test123456"}
	req = httptest.NewRequest("POST", "/api/auth/login", jsonBody(loginBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Login: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	loginResult := parseJSON(t, w.Body)
	loginToken, ok := loginResult["token"].(string)
	if !ok || loginToken == "" {
		t.Fatal("Login: token not returned")
	}

	// Me
	req = httptest.NewRequest("GET", "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+loginToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Me: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	meResult := parseJSON(t, w.Body)
	if meResult["email"] != "test@stackrun.dev" {
		t.Errorf("Me: expected test@stackrun.dev, got %v", meResult["email"])
	}

	// Me without token
	req = httptest.NewRequest("GET", "/api/auth/me", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Me without token: expected 401, got %d", w.Code)
	}

	// Duplicate register
	req = httptest.NewRequest("POST", "/api/auth/register", jsonBody(regBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("Duplicate register: expected 409, got %d", w.Code)
	}

	// Invalid login
	badLogin := map[string]string{"email": "test@stackrun.dev", "password": "wrongpass"}
	req = httptest.NewRequest("POST", "/api/auth/login", jsonBody(badLogin))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Bad login: expected 401, got %d", w.Code)
	}
}

func TestProjectsCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupTestRouter()

	// Register and get token
	regBody := map[string]string{"email": "proj@stackrun.dev", "name": "Proj User", "password": "test123456"}
	req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(regBody))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	result := parseJSON(t, w.Body)
	token := result["token"].(string)

	authHeader := "Bearer " + token

	// Create project
	projBody := map[string]string{"name": "My App", "slug": "my-app", "framework": "nextjs"}
	req = httptest.NewRequest("POST", "/api/projects", jsonBody(projBody))
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Create project: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	proj := parseJSON(t, w.Body)
	if proj["name"] != "My App" {
		t.Errorf("Expected 'My App', got %v", proj["name"])
	}
	projectID := proj["id"].(string)

	// List projects
	req = httptest.NewRequest("GET", "/api/projects", nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("List projects: expected 200, got %d", w.Code)
	}
	projects := parseJSONArray(t, w.Body)
	if len(projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(projects))
	}

	// Get single project
	req = httptest.NewRequest("GET", "/api/projects/"+projectID, nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Get project: expected 200, got %d", w.Code)
	}

	// Update project
	updateBody := map[string]string{"envVars": "DATABASE_URL=postgresql://localhost/db"}
	req = httptest.NewRequest("PATCH", "/api/projects/"+projectID, jsonBody(updateBody))
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Update project: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Create project without slug (auto-generate)
	autoBody := map[string]string{"name": "Another Project"}
	req = httptest.NewRequest("POST", "/api/projects", jsonBody(autoBody))
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("Auto-slug project: expected 201, got %d", w.Code)
	}
}

func TestDeployments(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupTestRouter()

	// Register and create project
	regBody := map[string]string{"email": "dep@stackrun.dev", "name": "Dep User", "password": "test123456"}
	req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(regBody))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	result := parseJSON(t, w.Body)
	token := result["token"].(string)
	authHeader := "Bearer " + token

	projBody := map[string]string{"name": "Deploy App", "slug": "deploy-app"}
	req = httptest.NewRequest("POST", "/api/projects", jsonBody(projBody))
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	proj := parseJSON(t, w.Body)
	projectID := proj["id"].(string)

	// Trigger deploy
	req = httptest.NewRequest("POST", "/api/projects/"+projectID+"/deploy", nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("Deploy: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List deployments
	req = httptest.NewRequest("GET", "/api/projects/"+projectID+"/deployments", nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("List deployments: expected 200, got %d", w.Code)
	}
	deps := parseJSONArray(t, w.Body)
	if len(deps) != 1 {
		t.Errorf("Expected 1 deployment, got %d", len(deps))
	}
}

func TestDomains(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupTestRouter()

	// Register and create project
	regBody := map[string]string{"email": "dom@stackrun.dev", "name": "Dom User", "password": "test123456"}
	req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(regBody))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	result := parseJSON(t, w.Body)
	token := result["token"].(string)
	authHeader := "Bearer " + token

	projBody := map[string]string{"name": "Domain App", "slug": "domain-app"}
	req = httptest.NewRequest("POST", "/api/projects", jsonBody(projBody))
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	proj := parseJSON(t, w.Body)
	projectID := proj["id"].(string)

	// Add domain
	domBody := map[string]string{"domain": "myapp.stackrun.dev"}
	req = httptest.NewRequest("POST", "/api/projects/"+projectID+"/domains", jsonBody(domBody))
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("Add domain: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	domResult := parseJSON(t, w.Body)
	domainID := domResult["id"].(string)
	if domResult["domain"] != "myapp.stackrun.dev" {
		t.Errorf("Expected myapp.stackrun.dev, got %v", domResult["domain"])
	}

	// List domains
	req = httptest.NewRequest("GET", "/api/projects/"+projectID+"/domains", nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("List domains: expected 200, got %d", w.Code)
	}
	domains := parseJSONArray(t, w.Body)
	if len(domains) != 1 {
		t.Errorf("Expected 1 domain, got %d", len(domains))
	}

	// Duplicate domain
	req = httptest.NewRequest("POST", "/api/projects/"+projectID+"/domains", jsonBody(domBody))
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("Duplicate domain: expected 409, got %d", w.Code)
	}

	// Delete domain
	req = httptest.NewRequest("DELETE", "/api/projects/"+projectID+"/domains/"+domainID, nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Delete domain: expected 200, got %d", w.Code)
	}

	// List domains (should be empty)
	req = httptest.NewRequest("GET", "/api/projects/"+projectID+"/domains", nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	domains = parseJSONArray(t, w.Body)
	if len(domains) != 0 {
		t.Errorf("Expected 0 domains after delete, got %d", len(domains))
	}
}

func TestDatabases(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupTestRouter()

	regBody := map[string]string{"email": "db@stackrun.dev", "name": "DB User", "password": "test123456"}
	req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(regBody))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	result := parseJSON(t, w.Body)
	token := result["token"].(string)
	authHeader := "Bearer " + token

	projBody := map[string]string{"name": "DB App", "slug": "db-app"}
	req = httptest.NewRequest("POST", "/api/projects", jsonBody(projBody))
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	proj := parseJSON(t, w.Body)
	projectID := proj["id"].(string)

	// Create database (this may fail in SQLite mode without psql, which is expected)
	dbBody := map[string]string{"projectId": projectID, "name": "testdb"}
	req = httptest.NewRequest("POST", "/api/databases", jsonBody(dbBody))
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Logf("Create database returned %d (may be expected in SQLite): %s", w.Code, w.Body.String())
	}
}

func TestWebhook(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupTestRouter()

	// Ping event
	whBody := map[string]interface{}{
		"repository": map[string]string{"clone_url": "https://github.com/user/repo.git"},
		"ref":        "refs/heads/main",
	}
	req := httptest.NewRequest("POST", "/api/webhook/github", jsonBody(whBody))
	req.Header.Set("x-github-event", "ping")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Webhook ping: expected 200, got %d", w.Code)
	}

	// Push event (no project with this repo, should still return ok with 0 deployed)
	req = httptest.NewRequest("POST", "/api/webhook/github", jsonBody(whBody))
	req.Header.Set("x-github-event", "push")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Webhook push: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w.Body)
	if result["ok"] != true {
		t.Errorf("Webhook: expected ok=true, got %v", result["ok"])
	}
}

func TestMetrics(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupTestRouter()

	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Metrics: expected 200, got %d", w.Code)
	}
	result := parseJSON(t, w.Body)
	if _, ok := result["uptime"]; !ok {
		t.Error("Metrics: uptime not found")
	}
	if _, ok := result["memory"]; !ok {
		t.Error("Metrics: memory not found")
	}

	// Prometheus metrics
	req = httptest.NewRequest("GET", "/api/metrics/prometheus", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Prometheus: expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "stackrun_uptime_seconds") {
		t.Error("Prometheus: expected stackrun_uptime_seconds")
	}
}

func TestAuthValidation(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupTestRouter()

	tests := []struct {
		name string
		body map[string]string
		code int
	}{
		{"Empty fields", map[string]string{"email": "", "name": "", "password": ""}, http.StatusBadRequest},
		{"Missing password", map[string]string{"email": "a@b.com", "name": "A"}, http.StatusBadRequest},
		{"Invalid JSON body", map[string]string{}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(tt.body))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tt.code {
				t.Errorf("Expected %d, got %d: %s", tt.code, w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthorization(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupTestRouter()

	protectedEndpoints := []string{
		"GET /api/projects",
		"POST /api/projects",
		"GET /api/auth/me",
		"GET /api/databases",
		"POST /api/databases",
	}

	for _, ep := range protectedEndpoints {
		t.Run(ep, func(t *testing.T) {
			parts := strings.SplitN(ep, " ", 2)
			method, path := parts[0], parts[1]
			body := strings.NewReader("{}")
			req := httptest.NewRequest(method, path, body)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s %s: expected 401, got %d", method, path, w.Code)
			}
		})
	}
}

func TestProjectNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupTestRouter()

	regBody := map[string]string{"email": "notfound@stackrun.dev", "name": "NF", "password": "test123456"}
	req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(regBody))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	result := parseJSON(t, w.Body)
	token := result["token"].(string)
	authHeader := "Bearer " + token

	// Accessing non-existent project should return null, not error
	req = httptest.NewRequest("GET", "/api/projects/nonexistent-id", nil)
	req.Header.Set("Authorization", authHeader)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for nonexistent project, got %d", w.Code)
	}
}
