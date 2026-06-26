-- Nidus Control Plane Schema

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
    env_vars TEXT,
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
