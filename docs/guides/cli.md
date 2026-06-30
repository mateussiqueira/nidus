# CLI

## Instalação

```bash
npm install -g stackrun
```

## Comandos

### Login

```bash
stackrun login <email> <password>
```

### Deploy

```bash
cd meu-projeto
stackrun deploy
```

Opções:

```bash
stackrun deploy --branch main
stackrun deploy --branch feature-x
```

### Listar projetos

```bash
stackrun projects
```

### Status

```bash
nidus status
```

## Exemplo completo

```bash
# 1. Login
stackrun login admin@stackrun.local senha123

# 2. Criar projeto (ou conectar via GitHub)
stackrun projects create --name meu-app

# 3. Deploy
cd meu-app
stackrun deploy

# 4. Verificar
nidus status
```
