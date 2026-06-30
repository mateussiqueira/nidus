package nidus

import "time"

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Framework string    `json:"framework"`
	Status    string    `json:"status"`
	RepoURL   string    `json:"repo_url"`
	Branch    string    `json:"branch"`
	Port      int       `json:"port"`
	Domain    string    `json:"domain"`
	CreatedAt time.Time `json:"created_at"`
}

type Deployment struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	URL       string    `json:"url"`
	Branch    string    `json:"branch"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type EnvVar struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

type Volume struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	MountPath string    `json:"mount_path"`
	SizeMb    int       `json:"size_mb"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

type DomainEntry struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
	Status string `json:"status"`
}

type Database struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	ProjectID string `json:"project_id,omitempty"`
}

type CreateProjectRequest struct {
	Name      string `json:"name"`
	RepoURL   string `json:"repo_url"`
	Branch    string `json:"branch"`
	Framework string `json:"framework,omitempty"`
	Port      int    `json:"port,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type EnvSetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type VolumeCreateRequest struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
}

type CreateDatabaseRequest struct {
	Name      string `json:"name"`
	ProjectID string `json:"project_id"`
}

type DeployRequest struct {
	Branch string `json:"branch"`
}
