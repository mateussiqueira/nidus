# Contribuindo para o Nimbus

Obrigado por interesse em contribuir com o Nimbus! Este documento explica como começar.

## Como Contribuir

1. **Fork** o repositório
2. **Clone** seu fork:
   ```bash
   git clone https://github.com/seu-usuario/nidus.git
   cd nidus
   ```
3. **Crie uma branch** para sua feature:
   ```bash
   git checkout -b feature/nome-da-feature
   ```
4. **Instale dependências**:
   ```bash
   pnpm install
   ```
5. **Configure o ambiente**:
   ```bash
   cp .env.example .env
   docker compose up -d
   ```
6. **Faça suas mudanças** e teste localmente
7. **Commit** com mensagens convencionais
8. **Push** para seu fork
9. **Abra um Pull Request**

## Estrutura do Projeto

```
stackrun/
├── apps/control-plane/    # API NestJS (TypeScript)
├── apps/dashboard/        # Frontend Next.js
├── apps/api/              # API alternativa (Go)
├── apps/proxy/            # Reverse proxy (Rust)
├── workers/deploy/        # Deploy worker (Go)
├── cli/                   # CLI (Node.js)
├── packages/shared/       # Tipos compartilhdos
├── packages/runtime/      # Engine de deploy
└── docs-site/             # Site da documentação
```

## Configuração de Desenvolvimento

### Pré-requisitos

- Node.js 22+
- Go 1.23+
- Rust 1.70+
- Docker + Docker Compose
- pnpm

### Setup

```bash
# Instalar dependências
pnpm install

# Iniciar banco e cache
docker compose up -d postgres redis

# Rodar migration
cd apps/control-plane
pnpm db:push

# Iniciar em modo dev
pnpm dev
```

## Convenções de Código

### Commits

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: adicionar nova feature
fix: corrigir bug
docs: atualizar documentação
test: adicionar testes
refactor: refatorar código
perf: melhorar performance
chore: tarefas de manutenção
```

### TypeScript

- Use tipos explícitos em todas as funções
- Evite `any` quando possível
- Use interfaces para objetos complexos

### Go

- Siga o [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` para formatação
- Escreva testes para funções públicas

### Rust

- Siga o [Rust Style Guide](https://doc.rust-lang.org/style-guide/)
- Use `cargo fmt` para formatação
- Trate erros com `Result` em vez de `panic!`

## Estrutura de Pull Requests

1. **Título claro** descrevendo a mudança
2. **Descrição** do que foi feito e por quê
3. **Screenshots** se aplicável
4. **Testes** passando localmente
5. **Docs** atualizadas se necessário

## Reportando Bugs

Abra uma issue com:

1. **Descrição clara** do problema
2. **Passos para reproduzir**
3. **Comportamento esperado vs atual**
4. **Ambiente** (OS, versão do Nimbus, etc.)
5. **Logs** se disponível

## Sugerindo Features

Abra uma issue com label `enhancement`:

1. **Problema** que a feature resolve
2. **Solução** proposta
3. **Alternativas** consideradas
4. **Contexto** adicional

## Code Review

Todas as PRs precisam de review antes de serem merged. O reviewer vai verificar:

- [ ] Código funciona como esperado
- [ ] Testes passam
- [ ] Não quebra funcionalidade existente
- [ ] Código segue as convenções
- [ ] Documentação está atualizada

## Perguntas?

Abra uma issue com label `question` ou entre no Discord do projeto.
