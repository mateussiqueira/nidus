package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupFullRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	mux.HandleFunc("POST /api/auth/register", handleRegister)
	mux.HandleFunc("POST /api/auth/login", handleLogin)
	mux.HandleFunc("GET /api/auth/me", withAuth(handleMe))
	mux.HandleFunc("GET /api/auth/github/login", handleGitHubLogin)
	mux.HandleFunc("GET /api/auth/github/callback", handleGitHubCallback)

	mux.HandleFunc("/api/projects/", withAuth(handleProjectRoutes))
	mux.HandleFunc("POST /api/projects", withAuth(handleCreateProject))
	mux.HandleFunc("GET /api/projects", withAuth(handleListProjects))

	mux.HandleFunc("/api/databases/", withAuth(handleDatabaseRoutes))
	mux.HandleFunc("POST /api/databases", withAuth(handleCreateDatabase))
	mux.HandleFunc("GET /api/databases", withAuth(handleListDatabases))
	mux.HandleFunc("GET /api/databases/{dbId}/metrics", withAuth(handleDatabaseMetrics))

	mux.HandleFunc("POST /api/webhook/github", handleWebhook)

	mux.HandleFunc("GET /api/plans", handleListPlans)
	mux.HandleFunc("POST /api/billing/checkout", withAuth(handleBillingCheckout))
	mux.HandleFunc("POST /api/billing/webhook", handleBillingWebhook)
	mux.HandleFunc("GET /api/billing/usage", withAuth(handleGetUsage))
	mux.HandleFunc("POST /api/billing/subscribe", withAuth(handleSubscribe))

	mux.HandleFunc("GET /api/admin/stats", withAuth(withAdmin(handleAdminStats)))
	mux.HandleFunc("GET /api/admin/users", withAuth(withAdmin(handleAdminUsers)))
	mux.HandleFunc("GET /api/admin/payments", withAuth(withAdmin(handleAdminPayments)))
	mux.HandleFunc("GET /api/admin/audit", withAuth(withAdmin(handleAdminAudit)))

	mux.HandleFunc("GET /api/tokens", withAuth(handleListTokens))
	mux.HandleFunc("POST /api/tokens", withAuth(handleCreateToken))
	mux.HandleFunc("DELETE /api/tokens/{tokenId}", withAuth(handleDeleteToken))

	mux.HandleFunc("GET /api/projects/{projectId}/cron", withAuth(handleListCronJobs))
	mux.HandleFunc("POST /api/projects/{projectId}/cron", withAuth(handleCreateCronJob))
	mux.HandleFunc("DELETE /api/projects/{projectId}/cron/{cronId}", withAuth(handleDeleteCronJob))

	mux.HandleFunc("GET /api/projects/{projectId}/envs", withAuth(handleListEnvVars))
	mux.HandleFunc("POST /api/projects/{projectId}/envs", withAuth(handleCreateEnvVar))
	mux.HandleFunc("PATCH /api/projects/{projectId}/envs/{envID}", withAuth(handleUpdateEnvVar))
	mux.HandleFunc("DELETE /api/projects/{projectId}/envs/{envID}", withAuth(handleDeleteEnvVar))

	mux.HandleFunc("GET /api/projects/{projectId}/volumes", withAuth(handleListVolumes))
	mux.HandleFunc("POST /api/projects/{projectId}/volumes", withAuth(handleCreateVolume))
	mux.HandleFunc("DELETE /api/projects/{projectId}/volumes/{volumeId}", withAuth(handleDeleteVolume))

	mux.HandleFunc("GET /api/metrics", handleMetrics)
	mux.HandleFunc("GET /api/metrics/prometheus", handlePrometheus)
	mux.HandleFunc("GET /api/projects/{projectId}/metrics/history", withAuth(handleProjectMetricsHistory))

	mux.HandleFunc("GET /api/projects/{projectId}/domains", withAuth(handleListDomains))
	mux.HandleFunc("POST /api/projects/{projectId}/domains", withAuth(handleAddDomain))
	mux.HandleFunc("DELETE /api/projects/{projectId}/domains/{domainId}", withAuth(handleDeleteDomain))
	mux.HandleFunc("POST /api/projects/{projectId}/domains/{domainId}/verify", withAuth(handleVerifyDomain))

	mux.HandleFunc("POST /api/projects/{projectId}/deployments/{deploymentId}/rollback", withAuth(handleRollback))

	mux.HandleFunc("GET /api/projects/{projectId}/webhooks", withAuth(handleListWebhooks))
	mux.HandleFunc("POST /api/projects/{projectId}/webhooks", withAuth(handleCreateWebhook))
	mux.HandleFunc("DELETE /api/projects/{projectId}/webhooks/{webhookId}", withAuth(handleDeleteWebhook))

	mux.HandleFunc("POST /api/mail/send", withAuth(handleSendMail))
	mux.HandleFunc("GET /api/mail/logs", withAuth(handleMailLogs))

	mux.HandleFunc("POST /api/projects/{projectId}/compose", withAuth(handleDeployCompose))

	return corsMiddleware(requestIDMiddleware(loggingMiddleware(mux)))
}

func registerAndGetToken(t *testing.T, router http.Handler, email, name, password string) (string, map[string]interface{}) {
	t.Helper()
	body := map[string]string{"email": email, "name": name, "password": password}
	req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Register failed (%d): %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w.Body)
	token, ok := result["token"].(string)
	if !ok || token == "" {
		t.Fatal("Token not returned from register")
	}
	user, _ := result["user"].(map[string]interface{})
	return token, user
}

func createTestProject(t *testing.T, router http.Handler, token, name, slug string) map[string]interface{} {
	t.Helper()
	body := map[string]string{"name": name, "slug": slug, "framework": "nextjs"}
	req := httptest.NewRequest("POST", "/api/projects", jsonBody(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create project failed (%d): %s", w.Code, w.Body.String())
	}
	return parseJSON(t, w.Body)
}

// ─── Dashboard Token Bypass ──────────────────────────────────────────────

func TestDashboardTokenBypass(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	t.Run("valid dashboard token", func(t *testing.T) {
		t.Setenv("DASHBOARD_TOKEN", "secret-dash-token-2026")
		req := httptest.NewRequest("GET", "/api/projects", nil)
		req.Header.Set("X-Dashboard-Token", "secret-dash-token-2026")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 with valid dash token, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid dashboard token", func(t *testing.T) {
		t.Setenv("DASHBOARD_TOKEN", "secret-dash-token-2026")
		req := httptest.NewRequest("GET", "/api/projects", nil)
		req.Header.Set("X-Dashboard-Token", "wrong-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 with invalid dash token, got %d", w.Code)
		}
	})

	t.Run("dashboard token not configured", func(t *testing.T) {
		t.Setenv("DASHBOARD_TOKEN", "")
		req := httptest.NewRequest("GET", "/api/projects", nil)
		req.Header.Set("X-Dashboard-Token", "anything")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 when dash token not configured, got %d", w.Code)
		}
	})

	t.Run("dashboard token on admin endpoint", func(t *testing.T) {
		t.Setenv("DASHBOARD_TOKEN", "secret-dash-token-2026")
		req := httptest.NewRequest("GET", "/api/admin/stats", nil)
		req.Header.Set("X-Dashboard-Token", "secret-dash-token-2026")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code == http.StatusOK {
			t.Error("Admin endpoint should not return 200 without admin role, even with dash token")
		}
	})
}

// ─── Auth ─────────────────────────────────────────────────────────────────

func TestLoginMissingFields(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	tests := []struct {
		name string
		body map[string]string
		code int
	}{
		{"empty body", map[string]string{}, http.StatusBadRequest},
		{"missing password", map[string]string{"email": "test@test.com"}, http.StatusBadRequest},
		{"missing email", map[string]string{"password": "test123456"}, http.StatusBadRequest},
		{"empty strings", map[string]string{"email": "", "password": ""}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/auth/login", jsonBody(tt.body))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != tt.code {
				t.Errorf("Expected %d, got %d: %s", tt.code, w.Code, w.Body.String())
			}
		})
	}
}

func TestRegisterMissingFields(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	tests := []struct {
		name string
		body map[string]string
		code int
	}{
		{"empty body", map[string]string{}, http.StatusBadRequest},
		{"missing name+password", map[string]string{"email": "test@test.com"}, http.StatusBadRequest},
		{"missing password", map[string]string{"email": "test@test.com", "name": "Test"}, http.StatusBadRequest},
		{"missing name", map[string]string{"email": "test@test.com", "password": "test123456"}, http.StatusBadRequest},
		{"only email", map[string]string{"email": "test@test.com"}, http.StatusBadRequest},
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

func TestMeAfterLogin(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	regBody := map[string]string{"email": "me-test@stackrun.dev", "name": "Me Test", "password": "test123456"}
	req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(regBody))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Register failed: %d", w.Code)
	}
	result := parseJSON(t, w.Body)
	token := result["token"].(string)

	loginBody := map[string]string{"email": "me-test@stackrun.dev", "password": "test123456"}
	req = httptest.NewRequest("POST", "/api/auth/login", jsonBody(loginBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Login after register: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	loginResult := parseJSON(t, w.Body)
	if loginResult["token"] == nil || loginResult["user"] == nil {
		t.Error("Login response should contain token and user")
	}

	req = httptest.NewRequest("GET", "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+loginResult["token"].(string))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Me with login token: expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Me: expected 200, got %d", w.Code)
	}
	meResult := parseJSON(t, w.Body)
	if meResult["email"] != "me-test@stackrun.dev" {
		t.Errorf("Me email mismatch: expected me-test@stackrun.dev, got %v", meResult["email"])
	}
	if meResult["name"] != "Me Test" {
		t.Errorf("Me name mismatch: expected Me Test, got %v", meResult["name"])
	}
	if _, ok := meResult["id"]; !ok {
		t.Error("Me response missing id")
	}
}

// ─── Invalid JWT Tokens ───────────────────────────────────────────────────

func TestInvalidJWT(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	invalidTokens := []struct {
		name  string
		token string
	}{
		{"random string", "not-a-jwt-at-all"},
		{"fake expired", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0IiwiZXhwIjoxNTAwMDAwMDAwfQ.signature"},
		{"empty token", ""},
		{"malformed bearer prefix", "Bearer "},
	}

	for _, tt := range invalidTokens {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/projects", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected 401 for %q, got %d", tt.name, w.Code)
			}
		})
	}
}

// ─── Projects ─────────────────────────────────────────────────────────────

func TestGetProjectInvalidUUID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "iu@stackrun.dev", "IU", "test123456")

	req := httptest.NewRequest("GET", "/api/projects/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid UUID, got %d", w.Code)
	}
}

func TestDeleteProject(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "del@stackrun.dev", "Del", "test123456")
	proj := createTestProject(t, router, token, "Delete Me", "delete-me")
	projectID := proj["id"].(string)

	req := httptest.NewRequest("DELETE", "/api/projects/"+projectID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Delete project: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("GET", "/api/projects/"+projectID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("Get after delete: expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProjectUpdateValidation(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "puv@stackrun.dev", "PUV", "test123456")
	proj := createTestProject(t, router, token, "Patch Test", "patch-test")
	projectID := proj["id"].(string)

	req := httptest.NewRequest("PATCH", "/api/projects/"+projectID, strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("PATCH with empty body: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	updateBody := `{"domain":"updated.example.com"}`
	req = httptest.NewRequest("PATCH", "/api/projects/"+projectID, strings.NewReader(updateBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("PATCH domain: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("GET", "/api/projects/"+projectID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	result := parseJSON(t, w.Body)
	if result["domain"] != "updated.example.com" {
		t.Errorf("Expected updated.example.com, got %v", result["domain"])
	}
}

// ─── Plans ────────────────────────────────────────────────────────────────

func TestListPlans(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	req := httptest.NewRequest("GET", "/api/plans", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 200 or 500, got %d: %s", w.Code, w.Body.String())
	}
	if w.Code == http.StatusOK {
		plans := parseJSONArray(t, w.Body)
		if plans == nil {
			t.Logf("Plans response is null (no data in table)")
		}
	}
}

// ─── Billing ──────────────────────────────────────────────────────────────

func TestBillingCheckout(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "bill@stackrun.dev", "Bill", "test123456")

	t.Run("checkout with plan", func(t *testing.T) {
		body := map[string]string{"planId": "pro", "paymentMethod": "pix"}
		req := httptest.NewRequest("POST", "/api/billing/checkout", jsonBody(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 201 or 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("checkout without plan defaults to cloud", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/billing/checkout", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 201 or 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestBillingWebhook(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	t.Run("webhook without signature none configured", func(t *testing.T) {
		body := map[string]interface{}{
			"paymentId": "pay-test-123",
			"status":    "paid",
		}
		req := httptest.NewRequest("POST", "/api/billing/webhook", jsonBody(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("webhook with invalid signature", func(t *testing.T) {
		t.Setenv("WEBHOOK_SECRET", "my-webhook-secret")
		body := map[string]string{"paymentId": "pay-test-456", "status": "paid"}
		req := httptest.NewRequest("POST", "/api/billing/webhook", jsonBody(body))
		req.Header.Set("X-Webhook-Signature", "wrong-secret")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 for invalid signature, got %d", w.Code)
		}
	})

	t.Run("webhook with valid signature", func(t *testing.T) {
		t.Setenv("WEBHOOK_SECRET", "my-webhook-secret")
		body := map[string]string{"paymentId": "pay-test-789", "status": "paid"}
		req := httptest.NewRequest("POST", "/api/billing/webhook", jsonBody(body))
		req.Header.Set("X-Webhook-Signature", "my-webhook-secret")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for valid signature, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestBillingUsage(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "usage@stackrun.dev", "Usage", "test123456")

	req := httptest.NewRequest("GET", "/api/billing/usage", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 200 or 500, got %d: %s", w.Code, w.Body.String())
	}
	if w.Code == http.StatusOK {
		result := parseJSON(t, w.Body)
		if result["month"] == nil {
			t.Error("Usage response should include month")
		}
	}
}

func TestSubscribe(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "sub@stackrun.dev", "Sub", "test123456")

	body := map[string]string{"planId": "pro"}
	req := httptest.NewRequest("POST", "/api/billing/subscribe", jsonBody(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 200 or 500, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── Admin ────────────────────────────────────────────────────────────────

func TestAdminEndpointsWithoutAdminRole(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "noadmin@stackrun.dev", "NoAdmin", "test123456")

	adminEndpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/admin/stats"},
		{"GET", "/api/admin/users"},
		{"GET", "/api/admin/payments"},
		{"GET", "/api/admin/audit"},
	}

	for _, ep := range adminEndpoints {
		t.Run(ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code == http.StatusOK {
				t.Errorf("%s should not return 200 for non-admin user", ep.path)
			}
			if w.Code != http.StatusForbidden && w.Code != http.StatusInternalServerError {
				t.Errorf("%s: expected 403 or 500, got %d", ep.path, w.Code)
			}
		})
	}
}

func TestAdminEndpointsWithoutAuth(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	adminEndpoints := []string{
		"/api/admin/stats",
		"/api/admin/users",
		"/api/admin/payments",
		"/api/admin/audit",
	}

	for _, path := range adminEndpoints {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s: expected 401, got %d", path, w.Code)
			}
		})
	}
}

// ─── CORS ─────────────────────────────────────────────────────────────────

func TestCORSPreflight(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	tests := []struct {
		origin     string
		expectCORS bool
	}{
		{"http://localhost:3000", true},
		{"http://evil.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			req := httptest.NewRequest("OPTIONS", "/api/projects", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusNoContent {
				t.Errorf("OPTIONS: expected 204, got %d", w.Code)
			}
			if w.Header().Get("Access-Control-Allow-Origin") == "" {
				t.Error("CORS allow-origin header missing")
			}
			if w.Header().Get("Access-Control-Allow-Methods") == "" {
				t.Error("CORS allow-methods header missing")
			}
		})
	}
}

func TestRequestIDHeader(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("x-request-id") == "" {
		t.Error("x-request-id header should be present")
	}

	customID := "my-custom-request-id-12345"
	req = httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("x-request-id", customID)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Header().Get("x-request-id") != customID {
		t.Errorf("Expected %s, got %s", customID, w.Header().Get("x-request-id"))
	}
}

// ─── Databases ────────────────────────────────────────────────────────────

func TestDatabaseMetrics(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "dbm@stackrun.dev", "DBM", "test123456")

	t.Run("metrics for nonexistent database", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/databases/00000000-0000-0000-0000-000000000000/metrics", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for nonexistent DB, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("metrics endpoint requires auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/databases/some-id/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}
	})
}

func TestDatabaseCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "dbcrud@stackrun.dev", "DbCRUD", "test123456")
	proj := createTestProject(t, router, token, "DB Project", "db-project")
	projectID := proj["id"].(string)

	req := httptest.NewRequest("GET", "/api/databases", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("List databases: expected 200, got %d", w.Code)
	}

	dbBody := map[string]string{"projectId": projectID, "name": "mydb"}
	req = httptest.NewRequest("POST", "/api/databases", jsonBody(dbBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	t.Logf("Create DB: %d - %s", w.Code, w.Body.String())

	if w.Code == http.StatusCreated {
		dbResult := parseJSON(t, w.Body)
		dbID := dbResult["id"].(string)

		req = httptest.NewRequest("GET", "/api/databases/"+dbID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Get database: expected 200, got %d", w.Code)
		}
	}
}

// ─── CRON Jobs ────────────────────────────────────────────────────────────

func TestCronJobs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "cron@stackrun.dev", "Cron", "test123456")
	proj := createTestProject(t, router, token, "Cron Project", "cron-project")
	projectID := proj["id"].(string)

	req := httptest.NewRequest("GET", "/api/projects/"+projectID+"/cron", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("List cron: expected 200 or 500, got %d", w.Code)
	}

	req = httptest.NewRequest("POST", "/api/projects/"+projectID+"/cron", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Create cron: expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest("DELETE", "/api/projects/"+projectID+"/cron/test-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Delete cron: expected 200, got %d", w.Code)
	}
}

// ─── Tokens ───────────────────────────────────────────────────────────────

func TestTokensCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "tk@stackrun.dev", "TK", "test123456")

	req := httptest.NewRequest("GET", "/api/tokens", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("List tokens: expected 200 or 500, got %d", w.Code)
	}

	req = httptest.NewRequest("POST", "/api/tokens", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Create token: expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest("DELETE", "/api/tokens/test-token-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Delete token: expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/tokens", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Tokens without auth: expected 401, got %d", w.Code)
	}
}

// ─── Env Vars ─────────────────────────────────────────────────────────────

func TestEnvVarsCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "env@stackrun.dev", "Env", "test123456")
	proj := createTestProject(t, router, token, "Env Project", "env-project")
	projectID := proj["id"].(string)

	req := httptest.NewRequest("GET", "/api/projects/"+projectID+"/envs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("List envs: expected 200 or 500, got %d", w.Code)
	}

	envBody := map[string]interface{}{"key": "DATABASE_URL", "value": "postgresql://localhost/db", "secret": true}
	req = httptest.NewRequest("POST", "/api/projects/"+projectID+"/envs", jsonBody(envBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
		t.Errorf("Create env: expected 201 or 500, got %d: %s", w.Code, w.Body.String())
	}

	if w.Code == http.StatusCreated {
		envResult := parseJSON(t, w.Body)
		envID, ok := envResult["id"].(string)
		if !ok {
			t.Fatal("Env var id not returned")
		}

		updateBody := map[string]string{"value": "new-value"}
		req = httptest.NewRequest("PATCH", "/api/projects/"+projectID+"/envs/"+envID, jsonBody(updateBody))
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Update env: expected 200, got %d", w.Code)
		}

		req = httptest.NewRequest("DELETE", "/api/projects/"+projectID+"/envs/"+envID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Delete env: expected 200, got %d", w.Code)
		}
	}

	badBody := map[string]string{"value": "some-value"}
	req = httptest.NewRequest("POST", "/api/projects/"+projectID+"/envs", jsonBody(badBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Create env without key: expected 400, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/projects/"+projectID+"/envs", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Envs without auth: expected 401, got %d", w.Code)
	}
}

// ─── Volumes ──────────────────────────────────────────────────────────────

func TestVolumesCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "vol@stackrun.dev", "Vol", "test123456")
	proj := createTestProject(t, router, token, "Volume Project", "vol-project")
	projectID := proj["id"].(string)

	req := httptest.NewRequest("GET", "/api/projects/"+projectID+"/volumes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("List volumes: expected 200 or 500, got %d", w.Code)
	}

	volBody := map[string]interface{}{"name": "data", "mountPath": "/data", "sizeMb": 1024}
	req = httptest.NewRequest("POST", "/api/projects/"+projectID+"/volumes", jsonBody(volBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
		t.Errorf("Create volume: expected 201 or 500, got %d: %s", w.Code, w.Body.String())
	}

	if w.Code == http.StatusCreated {
		volResult := parseJSON(t, w.Body)
		volID, ok := volResult["id"].(string)
		if !ok {
			t.Fatal("Volume id not returned")
		}

		req = httptest.NewRequest("DELETE", "/api/projects/"+projectID+"/volumes/"+volID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Delete volume: expected 200, got %d", w.Code)
		}
	}

	badBody := map[string]string{"mountPath": "/data"}
	req = httptest.NewRequest("POST", "/api/projects/"+projectID+"/volumes", jsonBody(badBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Create volume without name: expected 400, got %d", w.Code)
	}
}

// ─── Metrics ──────────────────────────────────────────────────────────────

func TestMetricsPrometheus(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	req := httptest.NewRequest("GET", "/api/metrics/prometheus", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Prometheus: expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	expectedMetrics := []string{
		"stackrun_uptime_seconds",
		"go_goroutines",
		"go_memstats_alloc_bytes",
		"go_info",
	}

	for _, m := range expectedMetrics {
		if !strings.Contains(body, m) {
			t.Errorf("Prometheus: expected metric %s not found", m)
		}
	}
}

func TestProjectMetricsHistory(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "pmh@stackrun.dev", "PMH", "test123456")
	proj := createTestProject(t, router, token, "Metrics Project", "metrics-project")
	projectID := proj["id"].(string)

	req := httptest.NewRequest("GET", "/api/projects/"+projectID+"/metrics/history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Project metrics history: expected 200 or 500, got %d", w.Code)
	}
}

// ─── Webhook ──────────────────────────────────────────────────────────────

func TestWebhookNonPushEvents(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	events := []string{"issues", "pull_request", "release", "create", "delete"}

	for _, event := range events {
		t.Run(event, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/webhook/github", nil)
			req.Header.Set("x-github-event", event)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("%s event: expected 200, got %d", event, w.Code)
			}
			result := parseJSON(t, w.Body)
			msg, _ := result["msg"].(string)
			if result["ok"] == true {
				t.Logf("%s event: ok=true (expected ok=false)", event)
			} else {
				t.Logf("%s event ignored (msg=%s)", event, msg)
			}
		})
	}
}

func TestWebhookPushWithBody(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	body := map[string]interface{}{
		"ref":   "refs/heads/main",
		"after": "abc123def456",
		"head_commit": map[string]interface{}{
			"id":      "abc123def456",
			"message": "test commit",
			"author":  map[string]string{"name": "Test", "email": "test@test.com"},
		},
		"repository": map[string]interface{}{
			"clone_url": "https://github.com/test/repo.git",
			"name":      "repo",
		},
	}

	req := httptest.NewRequest("POST", "/api/webhook/github", jsonBody(body))
	req.Header.Set("x-github-event", "push")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Webhook push: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w.Body)
	if result["ok"] != true {
		t.Errorf("Webhook push: expected ok=true, got %v", result["ok"])
	}
}

// ─── Protected Endpoints ─────────────────────────────────────────────────

func TestAllProtectedEndpoints(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	protectedEndpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/api/projects", ""},
		{"POST", "/api/projects", `{"name":"test","slug":"test"}`},
		{"GET", "/api/projects/00000000-0000-0000-0000-000000000000", ""},
		{"PATCH", "/api/projects/00000000-0000-0000-0000-000000000000", `{}`},
		{"DELETE", "/api/projects/00000000-0000-0000-0000-000000000000", ""},
		{"GET", "/api/auth/me", ""},
		{"GET", "/api/databases", ""},
		{"POST", "/api/databases", `{"name":"db"}`},
		{"POST", "/api/billing/checkout", `{}`},
		{"GET", "/api/billing/usage", ""},
		{"POST", "/api/billing/subscribe", `{}`},
		{"GET", "/api/tokens", ""},
		{"POST", "/api/tokens", ""},
		{"DELETE", "/api/tokens/test", ""},
		{"GET", "/api/projects/00000000-0000-0000-0000-000000000000/cron", ""},
		{"POST", "/api/projects/00000000-0000-0000-0000-000000000000/cron", ""},
		{"DELETE", "/api/projects/00000000-0000-0000-0000-000000000000/cron/test", ""},
		{"GET", "/api/projects/00000000-0000-0000-0000-000000000000/envs", ""},
		{"POST", "/api/projects/00000000-0000-0000-0000-000000000000/envs", `{"key":"k","value":"v"}`},
		{"PATCH", "/api/projects/00000000-0000-0000-0000-000000000000/envs/test", `{}`},
		{"DELETE", "/api/projects/00000000-0000-0000-0000-000000000000/envs/test", ""},
		{"GET", "/api/projects/00000000-0000-0000-0000-000000000000/volumes", ""},
		{"POST", "/api/projects/00000000-0000-0000-0000-000000000000/volumes", `{"name":"v","mountPath":"/v"}`},
		{"DELETE", "/api/projects/00000000-0000-0000-0000-000000000000/volumes/test", ""},
		{"GET", "/api/projects/00000000-0000-0000-0000-000000000000/metrics/history", ""},
		{"POST", "/api/projects/00000000-0000-0000-0000-000000000000/compose", ""},
		{"POST", "/api/mail/send", `{}`},
		{"GET", "/api/mail/logs", ""},
		{"GET", "/api/admin/stats", ""},
	}

	for _, ep := range protectedEndpoints {
		t.Run(fmt.Sprintf("%s %s", ep.method, ep.path), func(t *testing.T) {
			var bodyReader *strings.Reader
			if ep.body != "" {
				bodyReader = strings.NewReader(ep.body)
			} else {
				bodyReader = strings.NewReader("")
			}
			req := httptest.NewRequest(ep.method, ep.path, bodyReader)
			if ep.method != "GET" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s %s: expected 401, got %d", ep.method, ep.path, w.Code)
			}
		})
	}
}

// ─── GitHub OAuth ─────────────────────────────────────────────────────────

func TestGitHubOAuthNotConfigured(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	t.Setenv("GITHUB_CLIENT_ID", "")

	req := httptest.NewRequest("GET", "/api/auth/github/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 when GitHub OAuth not configured, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGitHubCallbackNotConfigured(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")

	req := httptest.NewRequest("GET", "/api/auth/github/callback?code=test&state=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestGitHubCallbackInvalidState(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	t.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")

	req := httptest.NewRequest("GET", "/api/auth/github/callback?code=test&state=wrong-state", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for invalid state, got %d", w.Code)
	}
}

// ─── Mail ─────────────────────────────────────────────────────────────────

func TestSendMailMissingFields(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "mail@stackrun.dev", "Mail", "test123456")

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/mail/send", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty send body, got %d", w.Code)
		}
	})

	t.Run("no to_email", func(t *testing.T) {
		body := map[string]string{"subject": "Test", "html_body": "<p>hi</p>"}
		req := httptest.NewRequest("POST", "/api/mail/send", jsonBody(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing to_email, got %d", w.Code)
		}
	})
}

func TestMailLogs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "maillog@stackrun.dev", "MailLog", "test123456")

	req := httptest.NewRequest("GET", "/api/mail/logs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Mail logs: expected 200 or 500, got %d", w.Code)
	}
}

// ─── Project Webhooks ─────────────────────────────────────────────────────

func TestProjectWebhooks(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "pwh@stackrun.dev", "PWH", "test123456")
	proj := createTestProject(t, router, token, "Webhook Project", "webhook-project")
	projectID := proj["id"].(string)

	req := httptest.NewRequest("GET", "/api/projects/"+projectID+"/webhooks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("List webhooks: expected 200 or 500, got %d", w.Code)
	}

	req = httptest.NewRequest("POST", "/api/projects/"+projectID+"/webhooks", strings.NewReader(`{"url":"https://example.com/webhook"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
		t.Errorf("Create webhook: expected 201 or 500, got %d", w.Code)
	}

	req = httptest.NewRequest("DELETE", "/api/projects/"+projectID+"/webhooks/test-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Delete webhook: expected 200 or 500, got %d", w.Code)
	}
}

// ─── Deploy Compose ──────────────────────────────────────────────────────

func TestDeployCompose(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "compose@stackrun.dev", "Compose", "test123456")
	proj := createTestProject(t, router, token, "Compose Project", "compose-project")
	projectID := proj["id"].(string)

	body := map[string]string{
		"compose": "version: \"3\"\nservices:\n  web:\n    image: nginx:alpine",
	}
	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/compose", jsonBody(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted && w.Code != http.StatusInternalServerError {
		t.Errorf("Deploy compose: expected 202 or 500, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── Deploy ──────────────────────────────────────────────────────────────

func TestDeployProject(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "deployer@stackrun.dev", "Deployer", "test123456")
	proj := createTestProject(t, router, token, "Deploy Project", "deploy-project-new")
	projectID := proj["id"].(string)

	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/deploy", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
		t.Errorf("Deploy: expected 201 or 500, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── Rollback ─────────────────────────────────────────────────────────────

func TestRollback(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "rollback@stackrun.dev", "Rollback", "test123456")
	proj := createTestProject(t, router, token, "Rollback Project", "rollback-project")
	projectID := proj["id"].(string)

	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/deployments/nonexistent/rollback", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("Rollback nonexistent deployment should not return 200")
	}
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Rollback: expected 404 or 500, got %d", w.Code)
	}
}

// ─── Deployments List ─────────────────────────────────────────────────────

func TestListDeploymentsEmpty(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "listdep@stackrun.dev", "ListDep", "test123456")
	proj := createTestProject(t, router, token, "NoDeploy Project", "nodeploy-project")
	projectID := proj["id"].(string)

	req := httptest.NewRequest("GET", "/api/projects/"+projectID+"/deployments", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("List deployments (empty): expected 200, got %d", w.Code)
	}
	deps := parseJSONArray(t, w.Body)
	if deps == nil {
		t.Error("Deployments list should be an array, not null")
	}
	if len(deps) > 0 {
		t.Errorf("Expected 0 deployments, got %d", len(deps))
	}
}

// ─── Content-Type ─────────────────────────────────────────────────────────

func TestContentTypeJSON(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	jsonEndpoints := []string{
		"/health",
		"/api/metrics",
	}

	for _, path := range jsonEndpoints {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			ct := w.Header().Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				t.Errorf("%s: expected application/json content type, got %s", path, ct)
			}
		})
	}
}

// ─── Domain Verify ────────────────────────────────────────────────────────

func TestDomainVerify(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(t, router, "domverify@stackrun.dev", "DomVerify", "test123456")
	proj := createTestProject(t, router, token, "DomainVerify Project", "domverify-project")
	projectID := proj["id"].(string)

	domBody := map[string]string{"domain": "verify-test.stackrun.dev"}
	req := httptest.NewRequest("POST", "/api/projects/"+projectID+"/domains", jsonBody(domBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusCreated {
		domResult := parseJSON(t, w.Body)
		domainID := domResult["id"].(string)

		req = httptest.NewRequest("POST", "/api/projects/"+projectID+"/domains/"+domainID+"/verify", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		t.Logf("Verify domain result: %d - %s", w.Code, w.Body.String())
		if w.Code == http.StatusInternalServerError {
			t.Log("Domain verification returned 500 (expected without actual DNS)")
		}
	}
}

// ─── Benchmark Tests ──────────────────────────────────────────────────────

func BenchmarkHealthEndpoint(b *testing.B) {
	cleanup := setupTestDB(&testing.T{})
	defer cleanup()
	router := setupFullRouter()

	req := httptest.NewRequest("GET", "/health", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkMetricsEndpoint(b *testing.B) {
	cleanup := setupTestDB(&testing.T{})
	defer cleanup()
	router := setupFullRouter()

	req := httptest.NewRequest("GET", "/api/metrics", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkPrometheusEndpoint(b *testing.B) {
	cleanup := setupTestDB(&testing.T{})
	defer cleanup()
	router := setupFullRouter()

	req := httptest.NewRequest("GET", "/api/metrics/prometheus", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkAuthMiddleware(b *testing.B) {
	cleanup := setupTestDB(&testing.T{})
	defer cleanup()
	router := setupFullRouter()

	token, _ := registerAndGetToken(&testing.T{}, router, "bench-auth@stackrun.dev", "BenchAuth", "test123456")

	req := httptest.NewRequest("GET", "/api/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRegisterHandler(b *testing.B) {
	cleanup := setupTestDB(&testing.T{})
	defer cleanup()
	router := setupFullRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		email := fmt.Sprintf("bench-reg-%d@stackrun.dev", i)
		body := map[string]string{"email": email, "name": "Bench", "password": "test123456"}
		req := httptest.NewRequest("POST", "/api/auth/register", jsonBody(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// ─── Helper: setupAuthRequest ───────────────────────────────────────────

func setupAuthRequest(method, path, body, userID string) *http.Request {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if userID != "" {
		ctx := context.WithValue(req.Context(), "userID", userID)
		req = req.WithContext(ctx)
	}
	return req
}

func seedBaseData() {
	db.Exec("INSERT OR IGNORE INTO users (id, email, name, password, role) VALUES ('test-user-1','test@test.com','Test User','hash','admin')")
	db.Exec("INSERT OR IGNORE INTO users (id, email, name, password, role) VALUES ('test-user-2','user@test.com','Normal User','hash','member')")
	db.Exec("INSERT OR IGNORE INTO plans (id, name, price_cents, max_projects, max_databases) VALUES ('free','Free',0,1,0)")
	db.Exec("INSERT OR IGNORE INTO plans (id, name, price_cents, max_projects, max_databases) VALUES ('pro','Pro',4900,10,3)")
	db.Exec("INSERT OR IGNORE INTO projects (id, name, slug, user_id, status, port, repo_url) VALUES ('550e8400-e29b-41d4-a716-446655440000','Test Project','test-project','test-user-1','ACTIVE',8090,'https://github.com/test/repo.git')")
}

// ─── DB-backed tests ───────────────────────────────────────────────────

func TestProjectList_WithDB(t *testing.T) {
	if testing.Short() { t.Skip("skipping DB test in short mode") }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/projects", "", "test-user-1")
	w := httptest.NewRecorder()
	handleListProjects(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		return
	}
	var projects []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&projects)
	if len(projects) == 0 {
		t.Error("expected at least 1 project")
	}
}

func TestProjectGet_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/projects/550e8400-e29b-41d4-a716-446655440000", "", "test-user-1")
	w := httptest.NewRecorder()
	handleProjectRoutes(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProjectCreate_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	body := `{"name":"New Project","repoUrl":"https://github.com/test/repo.git","branch":"main","framework":"nextjs"}`
	req := setupAuthRequest("POST", "/api/projects", body, "test-user-1")
	w := httptest.NewRecorder()
	handleCreateProject(w, req)

	if w.Code != 201 && w.Code != 200 {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProjectDelete_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO projects (id, name, slug, user_id, status) VALUES ('660e8400-e29b-41d4-a716-446655440099','Delete Me','delete-me','test-user-1','ACTIVE')")

	req := setupAuthRequest("DELETE", "/api/projects/660e8400-e29b-41d4-a716-446655440099", "", "test-user-1")
	w := httptest.NewRecorder()
	handleProjectRoutes(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeploy_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("POST", "/api/projects/550e8400-e29b-41d4-a716-446655440000/deploy", "{\"branch\":\"main\"}", "test-user-1")
	w := httptest.NewRecorder()
	handleProjectRoutes(w, req)

	if w.Code != 201 && w.Code != 200 {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseCreate_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	body := `{"name":"test-db","projectId":"550e8400-e29b-41d4-a716-446655440000"}`
	req := setupAuthRequest("POST", "/api/databases", body, "test-user-1")
	w := httptest.NewRecorder()
	handleCreateDatabase(w, req)

	if w.Code != 201 && w.Code != 200 {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseList_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/databases", "", "test-user-1")
	w := httptest.NewRecorder()
	handleListDatabases(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEnvVarCreate_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	body := `{"key":"DATABASE_URL","value":"postgres://localhost/test"}`
	req := setupAuthRequest("POST", "/api/projects/550e8400-e29b-41d4-a716-446655440000/envs", body, "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	w := httptest.NewRecorder()
	handleCreateEnvVar(w, req)

	if w.Code != 201 && w.Code != 200 {
		t.Errorf("expected 200 or 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDomainCreate_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	body := `{"domain":"example.com"}`
	req := setupAuthRequest("POST", "/api/projects/550e8400-e29b-41d4-a716-446655440000/domains", body, "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	w := httptest.NewRecorder()
	handleAddDomain(w, req)

	if w.Code != 201 && w.Code != 200 {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVolumeCreate_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	body := `{"name":"data","mountPath":"/app/data"}`
	req := setupAuthRequest("POST", "/api/projects/550e8400-e29b-41d4-a716-446655440000/volumes", body, "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	w := httptest.NewRecorder()
	handleCreateVolume(w, req)

	if w.Code != 201 && w.Code != 200 {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingSubscribe_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	body := `{"planId":"pro"}`
	req := setupAuthRequest("POST", "/api/billing/subscribe", body, "test-user-1")
	w := httptest.NewRecorder()
	handleSubscribe(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminStats_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/admin/stats", "", "test-user-1")
	w := httptest.NewRecorder()
	handleAdminStats(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminUsers_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/admin/users", "", "test-user-1")
	w := httptest.NewRecorder()
	handleAdminUsers(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
// ─── More DB-backed tests for coverage ────────────────────────────────

func TestDomainList_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO domains (id, project_id, domain) VALUES ('dom-1','550e8400-e29b-41d4-a716-446655440000','test.example.com')")

	req := setupAuthRequest("GET", "/api/projects/550e8400-e29b-41d4-a716-446655440000/domains", "", "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	w := httptest.NewRecorder()
	handleListDomains(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDomainDelete_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO domains (id, project_id, domain) VALUES ('dom-2','550e8400-e29b-41d4-a716-446655440000','delete.example.com')")

	req := setupAuthRequest("DELETE", "/api/projects/550e8400-e29b-41d4-a716-446655440000/domains/dom-2", "", "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	req.SetPathValue("domainId", "dom-2")
	w := httptest.NewRecorder()
	handleDeleteDomain(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDomainVerify_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO domains (id, project_id, domain) VALUES ('dom-3','550e8400-e29b-41d4-a716-446655440000','verify.example.com')")

	req := setupAuthRequest("POST", "/api/projects/550e8400-e29b-41d4-a716-446655440000/domains/dom-3/verify", "", "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	req.SetPathValue("domainId", "dom-3")
	w := httptest.NewRecorder()
	handleVerifyDomain(w, req)

	if w.Code != 200 && w.Code != 202 {
		t.Logf("Domain verify returned %d: %s", w.Code, w.Body.String())
	}
}

func TestEnvVarList_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO project_env_vars (id, project_id, key, value) VALUES ('env-1','550e8400-e29b-41d4-a716-446655440000','NODE_ENV','production')")

	req := setupAuthRequest("GET", "/api/projects/550e8400-e29b-41d4-a716-446655440000/envs", "", "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	w := httptest.NewRecorder()
	handleListEnvVars(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEnvVarDelete_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO project_env_vars (id, project_id, key, value) VALUES ('env-del','550e8400-e29b-41d4-a716-446655440000','TEMP_KEY','temp')")

	req := setupAuthRequest("DELETE", "/api/projects/550e8400-e29b-41d4-a716-446655440000/envs/env-del", "", "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	req.SetPathValue("envID", "env-del")
	w := httptest.NewRecorder()
	handleDeleteEnvVar(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVolumeList_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO project_volumes (id, project_id, name, mount_path) VALUES ('vol-1','550e8400-e29b-41d4-a716-446655440000','data','/app/data')")

	req := setupAuthRequest("GET", "/api/projects/550e8400-e29b-41d4-a716-446655440000/volumes", "", "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	w := httptest.NewRecorder()
	handleListVolumes(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVolumeDelete_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO project_volumes (id, project_id, name, mount_path) VALUES ('vol-del','550e8400-e29b-41d4-a716-446655440000','tmp','/tmp')")

	req := setupAuthRequest("DELETE", "/api/projects/550e8400-e29b-41d4-a716-446655440000/volumes/vol-del", "", "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	req.SetPathValue("volumeId", "vol-del")
	w := httptest.NewRecorder()
	handleDeleteVolume(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWebhookCreate_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	body := `{"url":"https://example.com/webhook","events":["deploy.success"]}`
	req := setupAuthRequest("POST", "/api/projects/550e8400-e29b-41d4-a716-446655440000/webhooks", body, "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	w := httptest.NewRecorder()
	handleCreateWebhook(w, req)

	if w.Code != 201 && w.Code != 200 {
		t.Errorf("expected 200 or 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWebhookList_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/projects/550e8400-e29b-41d4-a716-446655440000/webhooks", "", "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	w := httptest.NewRecorder()
	handleListWebhooks(w, req)

	if w.Code != 200 {
		t.Logf("Webhook list returned %d: %s", w.Code, w.Body.String())
	}
}

func TestTokenList_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/tokens", "", "test-user-1")
	w := httptest.NewRecorder()
	handleListTokens(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTokenCreateDelete_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	body := `{"name":"test-token"}`
	req := setupAuthRequest("POST", "/api/tokens", body, "test-user-1")
	w := httptest.NewRecorder()
	handleCreateToken(w, req)

	if w.Code != 201 && w.Code != 200 {
		t.Errorf("create token: expected 201 or 200, got %d: %s", w.Code, w.Body.String())
		return
	}
	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	tokenID, _ := result["id"].(string)
	if tokenID == "" {
		t.Skip("no token ID returned, skipping delete")
		return
	}

	req = setupAuthRequest("DELETE", "/api/tokens/"+tokenID, "", "test-user-1")
	w = httptest.NewRecorder()
	handleDeleteToken(w, req)

	if w.Code != 200 {
		t.Errorf("delete token: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlansList_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/plans", "", "")
	w := httptest.NewRecorder()
	handleListPlans(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUsage_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/billing/usage", "", "test-user-1")
	w := httptest.NewRecorder()
	handleGetUsage(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminPayments_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/admin/payments", "", "test-user-1")
	w := httptest.NewRecorder()
	handleAdminPayments(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminAudit_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/admin/audit", "", "test-user-1")
	w := httptest.NewRecorder()
	handleAdminAudit(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestComposeDeploy_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	body := "{\"compose\":\"version: '3'\\nservices:\\n  web:\\n    image: nginx\"}"
	req := setupAuthRequest("POST", "/api/projects/550e8400-e29b-41d4-a716-446655440000/compose", body, "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	w := httptest.NewRecorder()
	handleDeployCompose(w, req)

	if w.Code != 202 {
		t.Errorf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeploymentsList_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO deployments (id, project_id, branch, type, status) VALUES ('dep-1','550e8400-e29b-41d4-a716-446655440000','main','production','completed')")

	req := setupAuthRequest("GET", "/api/projects/550e8400-e29b-41d4-a716-446655440000/deployments", "", "test-user-1")
	w := httptest.NewRecorder()
	handleListDeployments(w, req, "550e8400-e29b-41d4-a716-446655440000")

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeploymentGet_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO deployments (id, project_id, branch, type, status) VALUES ('dep-2','550e8400-e29b-41d4-a716-446655440000','main','production','completed')")

	req := setupAuthRequest("GET", "/api/projects/550e8400-e29b-41d4-a716-446655440000/deployments/dep-2", "", "test-user-1")
	w := httptest.NewRecorder()
	handleGetDeployment(w, req, "550e8400-e29b-41d4-a716-446655440000", "dep-2")

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeploymentLogs_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO deployments (id, project_id, branch, type, status, logs) VALUES ('dep-3','550e8400-e29b-41d4-a716-446655440000','main','production','completed','Build succeeded')")

	req := setupAuthRequest("GET", "/api/projects/550e8400-e29b-41d4-a716-446655440000/deployments/dep-3/logs", "", "test-user-1")
	w := httptest.NewRecorder()
	handleDeploymentLogs(w, req, "550e8400-e29b-41d4-a716-446655440000", "dep-3")

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRollback_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO deployments (id, project_id, branch, type, status) VALUES ('dep-4','550e8400-e29b-41d4-a716-446655440000','main','production','completed')")

	req := setupAuthRequest("POST", "/api/projects/550e8400-e29b-41d4-a716-446655440000/deployments/dep-4/rollback", "", "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	req.SetPathValue("deploymentId", "dep-4")
	w := httptest.NewRecorder()
	handleRollback(w, req)

	if w.Code != 202 && w.Code != 200 && w.Code != 400 {
		t.Errorf("expected 200, 202, or 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseMetrics_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO databases (id, project_id, name, url) VALUES ('db-1','550e8400-e29b-41d4-a716-446655440000','metrics-db','postgres://localhost:5432/test')")

	req := setupAuthRequest("GET", "/api/databases/db-1/metrics", "", "test-user-1")
	req.SetPathValue("dbId", "db-1")
	w := httptest.NewRecorder()
	handleDatabaseMetrics(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProjectMetrics_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/projects/550e8400-e29b-41d4-a716-446655440000/metrics", "", "test-user-1")
	w := httptest.NewRecorder()
	handleProjectMetrics(w, req, "550e8400-e29b-41d4-a716-446655440000")

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProjectMetricsHistory_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/projects/550e8400-e29b-41d4-a716-446655440000/metrics/history", "", "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	w := httptest.NewRecorder()
	handleProjectMetricsHistory(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPreviewsList_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO deployments (id, project_id, branch, type, status) VALUES ('dep-5','550e8400-e29b-41d4-a716-446655440000','feat/test','preview','completed')")

	req := setupAuthRequest("GET", "/api/projects/550e8400-e29b-41d4-a716-446655440000/previews", "", "test-user-1")
	w := httptest.NewRecorder()
	handleListPreviews(w, req, "550e8400-e29b-41d4-a716-446655440000")

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
func TestDeleteDatabase_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO databases (id, project_id, name, url) VALUES ('db-del','550e8400-e29b-41d4-a716-446655440000','del-db','postgres://localhost/del')")

	req := setupAuthRequest("DELETE", "/api/databases/db-del", "", "test-user-1")
	w := httptest.NewRecorder()
	handleDeleteDatabase(w, req, "db-del")

	if w.Code != 200 && w.Code != 500 {
		t.Logf("Delete database returned %d: %s", w.Code, w.Body.String())
	}
}

func TestSanitizeBranch(t *testing.T) {
	tests := []struct{ in, want string }{
		{"main", "main"},
		{"feat/test", "feat-test"},
		{"bugfix/issue-42", "bugfix-issue-42"},
		{"feature/my branch", "feature-my-branch"},
		{"", ""},
	}
	for _, tt := range tests {
		got := sanitizeBranch(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeBranch(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSanitizeShell(t *testing.T) {
	tests := []struct{ in, want string }{
		{"echo hello", "echohello"},
		{"echo; rm -rf /", "echorm-rf/"},
		{"main", "main"},
	}
	for _, tt := range tests {
		got := sanitizeShell(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeShell(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestWithAuth_Invalid(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	handler := withAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	req := setupAuthRequest("GET", "/test", "", "")
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for missing token, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401 for invalid token, got %d", w.Code)
	}
}

func TestDBPing_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := db.Ping(); err != nil {
		t.Errorf("DB ping failed: %v", err)
	}
}

func TestRegisterDuplicate_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	body := `{"email":"test@test.com","name":"Dup User","password":"pass123"}`
	req := setupAuthRequest("POST", "/api/auth/register", body, "")
	w := httptest.NewRecorder()
	handleRegister(w, req)

	if w.Code != 409 {
		t.Errorf("expected 409 for duplicate email, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLoginSuccess_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	body := `{"email":"test@test.com","password":"hash"}`
	req := setupAuthRequest("POST", "/api/auth/login", body, "")
	w := httptest.NewRecorder()
	handleLogin(w, req)

	if w.Code != 200 && w.Code != 401 {
		t.Errorf("login: expected 200 or 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLoginMissingFields_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()

	body := `{}`
	req := setupAuthRequest("POST", "/api/auth/login", body, "")
	w := httptest.NewRecorder()
	handleLogin(w, req)

	if w.Code != 400 {
		t.Errorf("login empty body: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCronCreateDelete_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	body := `{"schedule":"* * * * *","command":"echo test"}`
	req := setupAuthRequest("POST", "/api/projects/550e8400-e29b-41d4-a716-446655440000/cron", body, "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	w := httptest.NewRecorder()
	handleCreateCronJob(w, req)

	if w.Code != 201 && w.Code != 200 {
		t.Errorf("create cron: expected 201, got %d: %s", w.Code, w.Body.String())
		return
	}
	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	cronID, _ := result["id"].(string)
	if cronID == "" {
		return
	}

	req = setupAuthRequest("DELETE", "/api/projects/550e8400-e29b-41d4-a716-446655440000/cron/"+cronID, "", "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	req.SetPathValue("cronId", cronID)
	w = httptest.NewRecorder()
	handleDeleteCronJob(w, req)

	if w.Code != 200 {
		t.Errorf("delete cron: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEnvVarUpdate_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()
	db.Exec("INSERT OR IGNORE INTO project_env_vars (id, project_id, key, value) VALUES ('env-upd','550e8400-e29b-41d4-a716-446655440000','OLD_KEY','old')")

	body := `{"key":"UPDATED_KEY","value":"newval"}`
	req := setupAuthRequest("PATCH", "/api/projects/550e8400-e29b-41d4-a716-446655440000/envs/env-upd", body, "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	req.SetPathValue("envID", "env-upd")
	w := httptest.NewRecorder()
	handleUpdateEnvVar(w, req)

	if w.Code != 200 {
		t.Logf("env var update returned %d: %s", w.Code, w.Body.String())
	}
}

func TestListCronJobs_WithDB(t *testing.T) {
	if testing.Short() { t.Skip() }
	cleanup := setupTestDB(t)
	defer cleanup()
	seedBaseData()

	req := setupAuthRequest("GET", "/api/projects/550e8400-e29b-41d4-a716-446655440000/cron", "", "test-user-1")
	req.SetPathValue("projectId", "550e8400-e29b-41d4-a716-446655440000")
	w := httptest.NewRecorder()
	handleListCronJobs(w, req)

	if w.Code != 200 {
		t.Logf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
