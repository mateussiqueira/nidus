# @stackrun.sdk

JavaScript/TypeScript SDK for the StackRun PaaS API.

## Install

```bash
npm install @stackrun.sdk
```

## Usage

```js
import { StackRunClient } from "@stackrun.sdk";

const nidus = new StackRunClient({ token: process.env.STACKRUN_TOKEN });

const projects = await stackrun.projects.list();
await stackrun.projects.deploy(projects[0].id);
```

## API

### `new StackRunClient(options?)`

| Option | Default                  | Description       |
| ------ | ------------------------ | ----------------- |
| apiUrl | `https://api.stackrun.vercel.app`  | API base URL      |
| token  | `null`                   | Bearer auth token |

### `stackrun.setToken(token)`

Set or update the auth token after instantiation.

### `stackrun.projects`

| Method                    | Description            |
| ------------------------- | ---------------------- |
| `.list()`                 | List all projects      |
| `.get(id)`                | Get a project by ID    |
| `.create(data)`           | Create a new project   |
| `.deploy(id, branch?)`    | Trigger a deployment   |
| `.envs(id)`               | List environment vars  |
| `.envSet(id, key, value)` | Set an environment var |

### `stackrun.deployments`

| Method                           | Description         |
| -------------------------------- | ------------------- |
| `.list(projectId)`               | List deployments    |
| `.logs(projectId, deploymentId)` | Get deployment logs |

### `stackrun.domains`

| Method                    | Description         |
| ------------------------- | ------------------- |
| `.list(projectId)`        | List custom domains |
| `.add(projectId, domain)` | Add a custom domain |

### `stackrun.databases`

| Method                     | Description       |
| -------------------------- | ----------------- |
| `.list()`                  | List all databases |
| `.create(name, projectId)` | Create a database  |

### `stackrun.auth`

| Method                             | Description |
| ---------------------------------- | ----------- |
| `.login(email, password)`          | Login       |
| `.register(email, password, name)` | Register    |

## License

MIT
