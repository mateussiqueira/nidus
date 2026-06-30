package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB
var dashboardURL = "http://localhost:3000"

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://nidus:nidus_dev_2026@localhost:5432/nidus?sslmode=disable"
	}

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping DB: %v", err)
	}

	if v := os.Getenv("DASHBOARD_URL"); v != "" {
		dashboardURL = v
	}

	port := os.Getenv("PROXY_PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleProxy)

	log.Printf("StackRun Proxy starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	host := strings.Split(r.Host, ":")[0]
	log.Printf("Request: host=%s path=%s", host, r.URL.Path)

	// Check if this is a project subdomain (e.g., my-project.stackrun.vercel.app)
	var projectSlug string
	if strings.HasSuffix(host, ".stackrun.vercel.app") {
		idx := strings.Index(host, ".stackrun.vercel.app")
		projectSlug = host[:idx]
	} else if strings.HasSuffix(host, ".localhost") {
		idx := strings.Index(host, ".localhost")
		projectSlug = host[:idx]
	}

	log.Printf("Extracted slug: %q", projectSlug)

	// Skip known system subdomains
	if projectSlug == "app" || projectSlug == "api" || projectSlug == "docs" || projectSlug == "metrics" || projectSlug == "stackrun" || projectSlug == "" {
		log.Printf("Routing to dashboard (system subdomain or empty)")
		proxyTo(w, r, dashboardURL)
		return
	}

	// Look up project by slug
	var containerPort int
	err := db.QueryRow(
		`SELECT COALESCE(port, 0) FROM projects WHERE slug = $1`,
		projectSlug,
	).Scan(&containerPort)

	if err == sql.ErrNoRows || containerPort == 0 {
		log.Printf("No project found for slug %q, routing to dashboard", projectSlug)
		proxyTo(w, r, dashboardURL)
		return
	}
	if err != nil {
		log.Printf("DB error for slug %q: %v", projectSlug, err)
		proxyTo(w, r, dashboardURL)
		return
	}

	targetURL := fmt.Sprintf("http://localhost:%d", containerPort)
	log.Printf("Routing to project container: %s", targetURL)
	proxyTo(w, r, targetURL)
}

func proxyTo(w http.ResponseWriter, r *http.Request, target string) {
	targetURL, err := url.Parse(target)
	if err != nil {
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ServeHTTP(w, r)
}
