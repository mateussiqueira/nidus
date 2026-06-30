package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (dp *DeployProcessor) deployCompose(deploymentID, projectID, projectSlug, composeYAML string, logFn func(string)) error {
	composeDir := filepath.Join("/tmp", "nidus-compose", projectSlug)
	os.MkdirAll(composeDir, 0755)
	defer os.RemoveAll(composeDir)

	composeFile := filepath.Join(composeDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(composeYAML), 0644); err != nil {
		return fmt.Errorf("write compose file: %w", err)
	}

	// Set project name to isolate services per deployment
	projectName := fmt.Sprintf("nidus-%s", projectSlug)
	projectName = strings.ReplaceAll(projectName, "-", "")

	logFn(fmt.Sprintf("📦 Reading compose services from %s...", composeFile))

	// Parse services from yaml (basic)
	configCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", projectName, "config", "--services")
	configCmd.Dir = composeDir
	servicesOut, err := configCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose config: %s", string(servicesOut))
	}
	services := strings.Fields(string(servicesOut))
	logFn(fmt.Sprintf("📋 Services detected: %v", services))

	// Pull images
	logFn("📥 Pulling images...")
	pullCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", projectName, "pull")
	pullCmd.Dir = composeDir
	if out, err := pullCmd.CombinedOutput(); err != nil {
		logFn(fmt.Sprintf("⚠ Pull warnings: %s", string(out)))
	}

	// Up
	logFn("🚀 Starting compose stack...")
	upCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", projectName, "up", "-d", "--remove-orphans")
	upCmd.Dir = composeDir
	upOut, err := upCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose up: %s", string(upOut))
	}
	logFn(string(upOut))

	// Get ports for each service
	for _, svc := range services {
		portCmd := exec.Command("docker", "compose", "-f", composeFile, "-p", projectName, "port", svc, "80")
		portCmd.Dir = composeDir
		portOut, _ := portCmd.CombinedOutput()
		logFn(fmt.Sprintf("  %s → %s", svc, strings.TrimSpace(string(portOut))))
	}

	logFn("✅ Compose stack deployed!")

	// Record services in DB
	for _, svc := range services {
		inspectCmd := exec.Command("docker", "inspect", "--format", "{{.State.Status}}|{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", fmt.Sprintf("%s-%s-1", projectName, svc))
		statusOut, _ := inspectCmd.CombinedOutput()
		parts := strings.SplitN(strings.TrimSpace(string(statusOut)), "|", 2)
		status := "running"
		ip := ""
		if len(parts) > 0 { status = parts[0] }
		if len(parts) > 1 { ip = parts[1] }

		dp.db.Exec(context.Background(),
			"INSERT INTO project_services (project_id, service_name, container_name, status, ip) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (project_id, service_name) DO UPDATE SET status=$4, ip=$5",
			projectID, svc, fmt.Sprintf("%s-%s-1", projectName, svc), status, ip)
	}

	return nil
}
