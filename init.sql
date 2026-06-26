-- Nidus Control Plane Schema
-- Version: 0.3.0

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enums
CREATE TYPE "ProjectStatus" AS ENUM ('ACTIVE', 'BUILDING', 'DEPLOYING', 'FAILED', 'PAUSED');

-- Users
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    password TEXT NOT NULL,
    avatar TEXT,
    created_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Projects
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    repo_url TEXT,
    branch TEXT DEFAULT 'main' NOT NULL,
    framework TEXT,
    status "ProjectStatus" DEFAULT 'ACTIVE' NOT NULL,
    domain TEXT,
    database_id TEXT,
    created_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects(user_id);

-- Deployments
CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id TEXT NOT NULL REFERENCES projects(id) ON UPDATE CASCADE ON DELETE CASCADE,
    commit_sha TEXT,
    branch TEXT DEFAULT 'main' NOT NULL,
    type TEXT DEFAULT 'production' NOT NULL,
    status TEXT DEFAULT 'pending' NOT NULL,
    logs TEXT,
    url TEXT,
    created_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL,
    finished_at TIMESTAMP(3)
);

CREATE INDEX IF NOT EXISTS idx_deployments_project_id ON deployments(project_id, created_at DESC);

-- Databases
CREATE TABLE IF NOT EXISTS databases (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id TEXT REFERENCES projects(id) ON UPDATE CASCADE ON DELETE SET NULL,
    name TEXT UNIQUE NOT NULL,
    url TEXT,
    created_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Environment Variables (per project)
CREATE TABLE IF NOT EXISTS environment_variables (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id TEXT NOT NULL REFERENCES projects(id) ON UPDATE CASCADE ON DELETE CASCADE,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    secret BOOLEAN DEFAULT false NOT NULL,
    created_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE(project_id, key)
);

CREATE INDEX IF NOT EXISTS idx_env_vars_project_id ON environment_variables(project_id);

-- SSH Keys (for private repos)
CREATE TABLE IF NOT EXISTS ssh_keys (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    name TEXT NOT NULL,
    public_key TEXT NOT NULL,
    fingerprint TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ssh_keys_user_id ON ssh_keys(user_id);

-- Webhooks (for GitHub integration)
CREATE TABLE IF NOT EXISTS webhooks (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id TEXT NOT NULL REFERENCES projects(id) ON UPDATE CASCADE ON DELETE CASCADE,
    url TEXT NOT NULL,
    secret TEXT,
    active BOOLEAN DEFAULT true NOT NULL,
    events TEXT[] DEFAULT ARRAY['push'] NOT NULL,
    created_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_webhooks_project_id ON webhooks(project_id);

-- Deployment Preview URLs
CREATE TABLE IF NOT EXISTS preview_deployments (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    deployment_id TEXT NOT NULL REFERENCES deployments(id) ON UPDATE CASCADE ON DELETE CASCADE,
    pr_number INTEGER,
    pr_url TEXT,
    subdomain TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at TIMESTAMP(3)
);

CREATE INDEX IF NOT EXISTS idx_preview_deployments_deployment_id ON preview_deployments(deployment_id);

-- Audit Log
CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    metadata JSONB,
    ip_address TEXT,
    created_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action, created_at DESC);

-- Rate Limiting (per user tier)
CREATE TABLE IF NOT EXISTS rate_limits (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    tier TEXT DEFAULT 'free' NOT NULL,
    requests_per_minute INTEGER DEFAULT 60 NOT NULL,
    requests_per_day INTEGER DEFAULT 1000 NOT NULL,
    created_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE(user_id)
);

-- Metrics (aggregated)
CREATE TABLE IF NOT EXISTS metrics (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id TEXT NOT NULL REFERENCES projects(id) ON UPDATE CASCADE ON DELETE CASCADE,
    metric_type TEXT NOT NULL,
    value NUMERIC NOT NULL,
    labels JSONB,
    recorded_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_metrics_project_id ON metrics(project_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_type ON metrics(metric_type, recorded_at DESC);
