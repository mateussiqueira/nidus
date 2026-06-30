# @nidus/sdk

JavaScript/TypeScript SDK for the Nidus PaaS API.

## Install

```bash
npm install @nidus/sdk
```

## Usage

```js
import { NidusClient } from "@nidus/sdk";

const nidus = new NidusClient({ token: process.env.NIDUS_TOKEN });

const projects = await nidus.projects.list();
await nidus.projects.deploy(projects[0].id);
```

## API

### `new NidusClient(options?)`

| Option | Default                  | Description       |
| ------ | ------------------------ | ----------------- |
| apiUrl | `https://api.nidus.app`  | API base URL      |
| token  | `null`                   | Bearer auth token |

### `nidus.setToken(token)`

Set or update the auth token after instantiation.

### `nidus.projects`

| Method                    | Description            |
| ------------------------- | ---------------------- |
| `.list()`                 | List all projects      |
| `.get(id)`                | Get a project by ID    |
| `.create(data)`           | Create a new project   |
| `.deploy(id, branch?)`    | Trigger a deployment   |
| `.envs(id)`               | List environment vars  |
| `.envSet(id, key, value)` | Set an environment var |

### `nidus.deployments`

| Method                           | Description         |
| -------------------------------- | ------------------- |
| `.list(projectId)`               | List deployments    |
| `.logs(projectId, deploymentId)` | Get deployment logs |

### `nidus.domains`

| Method                    | Description         |
| ------------------------- | ------------------- |
| `.list(projectId)`        | List custom domains |
| `.add(projectId, domain)` | Add a custom domain |

### `nidus.databases`

| Method                     | Description       |
| -------------------------- | ----------------- |
| `.list()`                  | List all databases |
| `.create(name, projectId)` | Create a database  |

### `nidus.auth`

| Method                             | Description |
| ---------------------------------- | ----------- |
| `.login(email, password)`          | Login       |
| `.register(email, password, name)` | Register    |

## License

MIT
