# Contribuindo com o Nidus

## Como começar

1. Fork o repo
2. `git clone https://github.com/seu-usuario/nidus.git`
3. `cd nidus && npm install`
4. `cp .env.example .env`
5. `docker compose up -d`
6. `npm run dev`

## Estrutura

- `apps/control-plane/` — API NestJS
- `apps/dashboard/` — Frontend Next.js
- `workers/deploy/` — Worker Go
- `packages/` — Código compartilhado

## Commits

Usa conventional commits:

```
feat: adiciona X
fix: corrige Y
refactor: refatora Z
perf: melhora performance de W
docs: atualiza documentação
test: adiciona testes
```

## Pull Requests

1. Cria uma branch da `main`
2. Faz as mudanças
3. Roda os testes: `npm test`
4. Abre o PR com descrição clara do que mudou

## Código

- Sem comentários desnecessários
- Nomes claros e descritivos
- Funções pequenas (< 50 linhas)
- Tipos em tudo que for TypeScript

## Bugs

Abre uma issue com:
- Passos pra reproduzir
- Comportamento esperado vs atual
- Versão do Node/OS
