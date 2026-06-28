package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all worker configuration, loaded from environment variables.
type Config struct {
	// Paths
	DeploysDir string

	// Networking
	Host     string
	APIURL   string

	// Database & Cache
	DatabaseURL string
	RedisURL   string

	// Server
	WorkerPort string

	// Build settings
	MaxConcurrentBuilds int
	DockerTimeout       time.Duration
	BuildkitEnabled     bool

	// Docker
	DockerHost string
}

// LoadConfig reads configuration from environment variables with sensible defaults.
func LoadConfig() *Config {
	return &Config{
		DeploysDir:          envOr("NIDUS_DEPLOYS_DIR", "/tmp/nidus-deploys"),
		Host:                envOr("NIDUS_HOST", "localhost"),
		APIURL:              envOr("API_URL", "http://localhost:3001"),
		DatabaseURL:         envOr("DATABASE_URL", "postgresql://broto:broto@localhost:5432/nidus"),
		RedisURL:            envOr("REDIS_URL", "redis://localhost:6379"),
		WorkerPort:          envOr("WORKER_PORT", "8081"),
		MaxConcurrentBuilds: envInt("MAX_CONCURRENT_BUILDS", 50),
		DockerTimeout:       envDuration("DOCKER_TIMEOUT_SECONDS", 120),
		BuildkitEnabled:     envBool("BUILDKIT_ENABLED", true),
		DockerHost:          envOr("DOCKER_HOST", ""),
	}
}

// cleanDBURL removes Prisma-specific query params (e.g., ?schema=public).
func cleanDBURL(url string) string {
	if idx := strings.Index(url, "?"); idx != -1 {
		return url[:idx]
	}
	return url
}

// envOr returns the value of the environment variable or a fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envInt parses an environment variable as int, returning fallback on failure.
func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil || i < 0 {
		return fallback
	}
	return i
}

// envDuration parses an environment variable as seconds → time.Duration.
func envDuration(key string, fallbackSeconds int) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return time.Duration(fallbackSeconds) * time.Second
	}
	d, err := time.ParseDuration(v + "s")
	if err != nil {
		return time.Duration(fallbackSeconds) * time.Second
	}
	return d
}

// envBool parses an environment variable as boolean.
func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1" || v == "yes"
}
