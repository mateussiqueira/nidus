# StackRun SDK

Python SDK for the StackRun PaaS API.

## Installation

```bash
pip install stackrun-sdk
```

## Usage

```python
from nidus import StackRunClient

nidus = StackRunClient(token="stackrun_xxx")
projects = stackrun.projects.list()

# Deploy a project
stackrun.projects.deploy(projects[0]["id"])

# Create a volume
stackrun.projects.volume_create(projects[0]["id"], "data", "/app/data")
```

### Authentication

```python
nidus = StackRunClient()
stackrun.auth.login("user@example.com", "password")
```

Or pass a token directly:

```python
nidus = StackRunClient(token="stackrun_xxx")
```

### Projects

```python
# List all projects
stackrun.projects.list()

# Get a single project
stackrun.projects.get("project-id")

# Create a project
stackrun.projects.create({"name": "my-app", "type": "nodejs"})

# Deploy a project
stackrun.projects.deploy("project-id")
stackrun.projects.deploy("project-id", branch="staging")

# Environment variables
stackrun.projects.envs("project-id")
stackrun.projects.env_set("project-id", "DATABASE_URL", "postgres://...")

# Volumes
stackrun.projects.volumes("project-id")
stackrun.projects.volume_create("project-id", "data", "/app/data")
```

### Deployments

```python
stackrun.deployments.list("project-id")
stackrun.deployments.logs("project-id", "deployment-id")
```

### Domains

```python
stackrun.domains.list("project-id")
stackrun.domains.add("project-id", "myapp.example.com")
stackrun.domains.delete("project-id", "domain-id")
```

### Databases

```python
stackrun.databases.list()
stackrun.databases.create("my-db", "project-id")
```
