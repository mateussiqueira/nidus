# Nidus Go SDK

Go client for the [Nidus](https://nidus.app) PaaS API.

## Installation

```bash
go get github.com/mateussiqueira/nidus/packages/sdk-go
```

## Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/mateussiqueira/nidus/packages/sdk-go/nidus"
)

func main() {
    client := nidus.NewClient("https://api.nidus.app")
    client.SetToken("nidus_xxx")

    projects, err := client.Projects.List()
    if err != nil {
        log.Fatal(err)
    }

    for _, p := range projects {
        fmt.Println(p.Name)

        deployment, err := client.Projects.Deploy(p.ID, "main")
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("Deploying: %s\n", deployment.Status)
    }
}
```

## Services

### Projects

```go
projects, _ := client.Projects.List()
project, _ := client.Projects.Get("id")
created, _ := client.Projects.Create(nidus.CreateProjectRequest{...})
deployment, _ := client.Projects.Deploy("id", "main")
envs, _ := client.Projects.Envs("id")
client.Projects.EnvSet("id", "KEY", "value")
volumes, _ := client.Projects.Volumes("id")
volume, _ := client.Projects.VolumeCreate("id", "data", "/data")
```

### Deployments

```go
deployments, _ := client.Deployments.List("project-id")
deployment, _ := client.Deployments.Get("project-id", "deployment-id")
logs, _ := client.Deployments.Logs("project-id", "deployment-id")
```

### Domains

```go
domains, _ := client.Domains.List("project-id")
entry, _ := client.Domains.Add("project-id", "app.example.com")
client.Domains.Delete("project-id", "domain-id")
```

### Databases

```go
databases, _ := client.Databases.List()
db, _ := client.Databases.Create("my-db", "project-id")
```

### Auth

```go
user, _ := client.Auth.Login("email@example.com", "password")
client.SetToken(user.Token)

user, _ := client.Auth.Register("email@example.com", "password", "Name")
```
