# Contributing to Nidus

## Getting started

1. Fork the repo
2. `git clone https://github.com/your-user/nidus.git`
3. `cd nidus && npm install`
4. `cp .env.example .env`
5. `docker compose up -d`
6. `npm run dev`

## Structure

- `apps/control-plane/` — NestJS API
- `apps/dashboard/` — Next.js frontend
- `workers/deploy/` — Go worker
- `packages/` — Shared code

## Commits

Use conventional commits:

```
feat: add X
fix: fix Y
refactor: refactor Z
perf: improve performance of W
docs: update documentation
test: add tests
```

## Pull Requests

1. Create a branch from `main`
2. Make changes
3. Run tests: `npm test`
4. Open PR with clear description of what changed

## Code

- No unnecessary comments
- Clear, descriptive names
- Small functions (< 50 lines)
- Types everywhere in TypeScript

## Bugs

Open an issue with:
- Steps to reproduce
- Expected vs actual behavior
- Node/OS version
