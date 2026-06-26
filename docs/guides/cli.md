# CLI

## Instalação

```bash
npm install -g nidus
```

## Comandos

### Login

```bash
nidus login <email> <password>
```

### Deploy

```bash
cd meu-projeto
nidus deploy
```

Opções:

```bash
nidus deploy --branch main
nidus deploy --branch feature-x
```

### Listar projetos

```bash
nidus projects
```

### Status

```bash
nidus status
```

## Exemplo completo

```bash
# 1. Login
nidus login admin@nidus.local senha123

# 2. Criar projeto (ou conectar via GitHub)
nidus projects create --name meu-app

# 3. Deploy
cd meu-app
nidus deploy

# 4. Verificar
nidus status
```
