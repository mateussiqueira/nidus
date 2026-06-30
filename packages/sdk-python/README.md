# Nidus SDK

Python SDK for the Nidus PaaS API.

## Installation

```bash
pip install nidus-sdk
```

## Usage

```python
from nidus import NidusClient

nidus = NidusClient(token="nidus_xxx")
projects = nidus.projects.list()

# Deploy a project
nidus.projects.deploy(projects[0]["id"])

# Create a volume
nidus.projects.volume_create(projects[0]["id"], "data", "/app/data")
```

### Authentication

```python
nidus = NidusClient()
nidus.auth.login("user@example.com", "password")
```

Or pass a token directly:

```python
nidus = NidusClient(token="nidus_xxx")
```

### Projects

```python
# List all projects
nidus.projects.list()

# Get a single project
nidus.projects.get("project-id")

# Create a project
nidus.projects.create({"name": "my-app", "type": "nodejs"})

# Deploy a project
nidus.projects.deploy("project-id")
nidus.projects.deploy("project-id", branch="staging")

# Environment variables
nidus.projects.envs("project-id")
nidus.projects.env_set("project-id", "DATABASE_URL", "postgres://...")

# Volumes
nidus.projects.volumes("project-id")
nidus.projects.volume_create("project-id", "data", "/app/data")
```

### Deployments

```python
nidus.deployments.list("project-id")
nidus.deployments.logs("project-id", "deployment-id")
```

### Domains

```python
nidus.domains.list("project-id")
nidus.domains.add("project-id", "myapp.example.com")
nidus.domains.delete("project-id", "domain-id")
```

### Databases

```python
nidus.databases.list()
nidus.databases.create("my-db", "project-id")
```
