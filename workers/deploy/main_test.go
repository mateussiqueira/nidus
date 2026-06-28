package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectFramework(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		files    map[string]string
		expected string
	}{
		{
			"nextjs config",
			map[string]string{"next.config.js": ""},
			"nextjs",
		},
		{
			"nuxt config",
			map[string]string{"nuxt.config.ts": ""},
			"nuxt",
		},
		{
			"vite config",
			map[string]string{"vite.config.js": ""},
			"vite",
		},
		{
			"angular",
			map[string]string{"angular.json": "{}"},
			"angular",
		},
		{
			"astro",
			map[string]string{"astro.config.mjs": ""},
			"astro",
		},
		{
			"static (no config)",
			map[string]string{},
			"static",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanDir(t, tmpDir)
			for path, content := range tt.files {
				dir := filepath.Dir(filepath.Join(tmpDir, path))
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(tmpDir, path), []byte(content), 0644)
			}
			result := detectFramework(tmpDir)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDetectFrameworkDart(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("vaden", func(t *testing.T) {
		os.WriteFile(filepath.Join(tmpDir, "pubspec.yaml"), []byte("name: myapp\ndependencies:\n  vaden: ^1.0.0\n"), 0644)
		if result := detectFramework(tmpDir); result != "vaden" {
			t.Errorf("Expected vaden, got %q", result)
		}
	})

	t.Run("dart shelf", func(t *testing.T) {
		cleanDir(t, tmpDir)
		os.WriteFile(filepath.Join(tmpDir, "pubspec.yaml"), []byte("name: myapp\ndependencies:\n  shelf: ^1.4.0\n"), 0644)
		if result := detectFramework(tmpDir); result != "vaden" {
			t.Errorf("Expected vaden for shelf, got %q", result)
		}
	})

	t.Run("dart generic", func(t *testing.T) {
		cleanDir(t, tmpDir)
		os.WriteFile(filepath.Join(tmpDir, "pubspec.yaml"), []byte("name: myapp\n"), 0644)
		if result := detectFramework(tmpDir); result != "dart" {
			t.Errorf("Expected dart, got %q", result)
		}
	})

	t.Run("flutter", func(t *testing.T) {
		cleanDir(t, tmpDir)
		os.WriteFile(filepath.Join(tmpDir, "pubspec.yaml"), []byte("name: myapp\nenvironment:\n  flutter: '>=3.0.0'\n"), 0644)
		if result := detectFramework(tmpDir); !strings.Contains(result, "flutter") {
			t.Errorf("Expected flutter, got %q", result)
		}
	})
}

func TestDetectFrameworkFromPackageJSON(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"next from deps", `{"dependencies":{"next":"14.0.0"}}`, "nextjs"},
		{"nuxt from deps", `{"dependencies":{"nuxt":"3.0.0"}}`, "nuxt"},
		{"react with vite", `{"dependencies":{"react":"18.0.0"},"devDependencies":{"vite":"5.0.0"}}`, "vite"},
		{"svelte", `{"dependencies":{"svelte":"4.0.0"}}`, "svelte"},
		{"no deps", `{}`, "static"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanDir(t, tmpDir)
			os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.content), 0644)
			result := detectFramework(tmpDir)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateDockerfile(t *testing.T) {
	tests := []struct {
		framework string
		contains  string
	}{
		{"nextjs", "FROM node:22-alpine"},
		{"nuxt", ".output/server/index.mjs"},
		{"vite", "FROM nginx:alpine"},
		{"angular", "FROM nginx:alpine"},
		{"svelte", "FROM nginx:alpine"},
		{"astro", "FROM nginx:alpine"},
		{"vaden", "FROM dart:stable"},
		{"dart", "FROM dart:stable"},
		{"flutter", "FROM ubuntu:22.04"},
		{"static", "FROM nginx:alpine"},
		{"unknown", "FROM nginx:alpine"},
	}

	for _, tt := range tests {
		t.Run(tt.framework, func(t *testing.T) {
			result := generateDockerfile(tt.framework)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Expected Dockerfile for %q to contain %q", tt.framework, tt.contains)
			}
		})
	}
}

func TestGetExposedPort(t *testing.T) {
	tests := []struct {
		framework string
		expected  int
	}{
		{"nextjs", 3000},
		{"nuxt", 3000},
		{"vaden", 8080},
		{"dart", 8080},
		{"flutter", 80},
		{"vite", 80},
		{"angular", 80},
		{"svelte", 80},
		{"static", 80},
	}

	for _, tt := range tests {
		t.Run(tt.framework, func(t *testing.T) {
			result := getExposedPort(tt.framework)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestSanitizeBranch(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"main", "main"},
		{"feature/my-app", "feature-my-app"},
		{"FEATURE-1", "feature-1"},
		{"fix!@#$%^", "fix------"},
		{"a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeBranch(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizeShell(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"safe-path", "safe-path"},
		{"https://github.com/user/repo.git", "https://github.com/user/repo.git"},
		{"rm -rf /", "rm-rf/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeShell(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDeployJobStruct(t *testing.T) {
	job := DeployJob{
		DeploymentID: "dep-123",
		ProjectID:    "proj-456",
		ProjectName:  "Test App",
		ProjectSlug:  "test-app",
		Branch:       "main",
		DeployType:   "production",
		IsPreview:    false,
	}

	if job.DeploymentID != "dep-123" {
		t.Errorf("Expected dep-123, got %s", job.DeploymentID)
	}
	if job.ProjectName != "Test App" {
		t.Errorf("Expected Test App, got %s", job.ProjectName)
	}
}

func cleanDir(t *testing.T, dir string) {
	t.Helper()
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		os.RemoveAll(filepath.Join(dir, e.Name()))
	}
}
