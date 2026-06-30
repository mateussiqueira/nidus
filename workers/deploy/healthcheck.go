package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	healthCheckInterval    = 30 * time.Second
	maxConsecutiveFailures = 3
	restartBackoffBase     = 30 * time.Second
	maxRestartBackoff      = 5 * time.Minute
)

type HealthChecker struct {
	db       *pgxpool.Pool
	rdb      *redis.Client
	mu       sync.Mutex
	failures map[string]int
	backoffs map[string]time.Time
}

func startHealthChecker(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client) {
	hc := &HealthChecker{
		db:       db,
		rdb:      rdb,
		failures: make(map[string]int),
		backoffs: make(map[string]time.Time),
	}
	go hc.run(ctx)
}

func (hc *HealthChecker) run(ctx context.Context) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	log.Println("[health-checker] Started (interval: 30s, max failures: 3)")

	time.Sleep(5 * time.Second)
	hc.checkAll(ctx)

	for {
		select {
		case <-ticker.C:
			hc.checkAll(ctx)
		case <-ctx.Done():
			log.Println("[health-checker] Stopped")
			return
		}
	}
}

func (hc *HealthChecker) checkAll(ctx context.Context) {
	rows, err := hc.db.Query(ctx, `SELECT id, slug, name, port FROM projects WHERE status = $1 AND port > 0 AND port IS NOT NULL`, "ACTIVE")
	if err != nil {
		log.Printf("[health-checker] Query error: %v", err)
		return
	}
	defer rows.Close()

	var wg sync.WaitGroup
	for rows.Next() {
		var id, slug, name string
		var port int
		if err := rows.Scan(&id, &slug, &name, &port); err != nil {
			continue
		}
		wg.Add(1)
		go func(pid, slug, name string, port int) {
			defer wg.Done()
			hc.checkProject(ctx, pid, slug, name, port)
		}(id, slug, name, port)
	}
	wg.Wait()
}

func (hc *HealthChecker) checkProject(ctx context.Context, projectID, slug, name string, port int) {
	start := time.Now()
	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)

	resp, httpErr := http.Get(url)
	elapsed := time.Since(start).Milliseconds()

	var status string
	var httpCode int
	var errMsg string

	if httpErr != nil {
		status = "down"
		errMsg = httpErr.Error()
	} else {
		defer resp.Body.Close()
		httpCode = resp.StatusCode
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			status = "up"
		} else {
			status = "down"
			errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}

	// Record to DB using simple syntax
	_, dbErr := hc.db.Exec(ctx,
		`INSERT INTO health_checks (project_id, status, response_time_ms, http_code, error, checked_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())`,
		projectID, status, int(elapsed), httpCode, errMsg)
	if dbErr != nil {
		log.Printf("[health-checker] DB insert error for %s: %v", slug, dbErr)
	}

	hc.mu.Lock()
	if status == "down" {
		hc.failures[projectID]++
	} else {
		hc.failures[projectID] = 0
		hc.backoffs[projectID] = time.Time{}
	}

	failures := hc.failures[projectID]
	lastRestart := hc.backoffs[projectID]
	hc.mu.Unlock()

	if failures >= maxConsecutiveFailures && time.Since(lastRestart) > restartBackoffBase {
		backoff := restartBackoffBase * time.Duration(1<<minInt(failures-maxConsecutiveFailures, 4))
		if backoff > maxRestartBackoff {
			backoff = maxRestartBackoff
		}

		if time.Since(lastRestart) > backoff {
			log.Printf("[health-checker] ⚠ %s (%s) down for %d checks — restarting (backoff: %s)", slug, name, failures, backoff)
			hc.restartContainer(projectID, slug, port)
			hc.mu.Lock()
			hc.backoffs[projectID] = time.Now()
			hc.failures[projectID] = 0
			hc.mu.Unlock()
		}
	}

	if failures > 0 {
		log.Printf("[health-checker] %s:%d %s ❌ (fail #%d) %dms", slug, port, errMsg, failures, elapsed)
	} else if status == "up" {
		log.Printf("[health-checker] %s:%d ✅ %dms", slug, port, elapsed)
	}
}

func (hc *HealthChecker) restartContainer(projectID, slug string, port int) {
	containerName := fmt.Sprintf("nidus-%s-%s", slug, projectID[:8])

	cmd := exec.Command("docker", "stop", containerName)
	cmd.CombinedOutput()

	cmd = exec.Command("docker", "start", containerName)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[health-checker] Start container %s error: %v — %s", containerName, err, strings.TrimSpace(string(out)))
		return
	}

	log.Printf("[health-checker] ✅ Restarted container %s", containerName)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
