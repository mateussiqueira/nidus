# SDKs

Nidus tem SDKs oficiais para JavaScript, Python e Go.

## JavaScript

```bash
npm install @nidus/sdk
```

```js
import { Nidus } from '@nidus/sdk'

const nidus = new Nidus({ token: process.env.NIDUS_TOKEN })

const projects = await nidus.projects.list()
const deploy = await nidus.deploy.create({
  projectId: 'my-app',
  compose: dockerComposeYaml
})

console.log(deploy.url)
```

## Python

```bash
pip install nidus-sdk
```

```python
from nidus import Nidus

nidus = Nidus(token="seu-token")

projects = nidus.projects.list()
deploy = nidus.deploy.create(
    project_id="my-app",
    compose=docker_compose_yaml
)

print(deploy.url)
```

## Go

```bash
go get github.com/mateussiqueira/nidus/packages/sdk-go
```

```go
import "github.com/mateussiqueira/nidus/packages/sdk-go"

client := nidus.NewClient("seu-token")

projects, _ := client.Projects.List()
deploy, _ := client.Deploy.Create(&nidus.DeployInput{
    ProjectID: "my-app",
    Compose:   dockerComposeYaml,
})

fmt.Println(deploy.URL)
```
