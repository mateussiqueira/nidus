# SDKs

StackRun tem SDKs oficiais para JavaScript, Python e Go.

## JavaScript

```bash
npm install @stackrun.sdk
```

```js
import { StackRun } from '@stackrun.sdk'

const nidus = new StackRun({ token: process.env.STACKRUN_TOKEN })

const projects = await stackrun.projects.list()
const deploy = await nidus.deploy.create({
  projectId: 'my-app',
  compose: dockerComposeYaml
})

console.log(deploy.url)
```

## Python

```bash
pip install stackrun-sdk
```

```python
from nidus import StackRun

nidus = StackRun(token="seu-token")

projects = stackrun.projects.list()
deploy = nidus.deploy.create(
    project_id="my-app",
    compose=docker_compose_yaml
)

print(deploy.url)
```

## Go

```bash
go get github.com/mateussiqueira/stackrun/packages/sdk-go
```

```go
import "github.com/mateussiqueira/stackrun/packages/sdk-go"

client := stackrun.NewClient("seu-token")

projects, _ := client.Projects.List()
deploy, _ := client.Deploy.Create(&nidus.DeployInput{
    ProjectID: "my-app",
    Compose:   dockerComposeYaml,
})

fmt.Println(deploy.URL)
```
