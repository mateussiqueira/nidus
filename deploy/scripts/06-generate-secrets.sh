#!/bin/bash
# ============================================
# STACKRUN - Generate Secure Secrets
# Execute uma única vez no VPS
# ============================================
set -e

echo "🔑 Gerando segredos seguros..."

# Criar diretório de secrets
mkdir -p /opt/stackrun/secrets
chmod 700 /opt/stackrun/secrets

# Gerar senhas seguras (32 caracteres)
POSTGRES_PASSWORD=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
REDIS_PASSWORD=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
JWT_SECRET=$(openssl rand -base64 64 | tr -dc 'a-zA-Z0-9' | head -c 64)
NEXTAUTH_SECRET=$(openssl rand -base64 64 | tr -dc 'a-zA-Z0-9' | head -c 64)

# Salvar secrets em arquivos
echo -n "$POSTGRES_PASSWORD" > /opt/stackrun/secrets/postgres_password.txt
echo -n "$REDIS_PASSWORD" > /opt/stackrun/secrets/redis_password.txt
echo -n "$JWT_SECRET" > /opt/stackrun/secrets/jwt_secret.txt
echo -n "$NEXTAUTH_SECRET" > /opt/stackrun/secrets/nextauth_secret.txt

# Gerar .env.production
cat > /opt/stackrun/.env.production << EOF
# ============================================
# STACKRUN - Production Environment Variables
# Gerado em: $(date)
# ============================================

# PostgreSQL
POSTGRES_USER=nidus
POSTGRES_PASSWORD=$POSTGRES_PASSWORD

# Redis
REDIS_PASSWORD=$REDIS_PASSWORD

# JWT
JWT_SECRET=$JWT_SECRET
JWT_EXPIRES_IN=7d

# NextAuth
NEXTAUTH_SECRET=$NEXTAUTH_SECRET
NEXTAUTH_URL=https://stackrun.vercel.app

# CORS
CORS_ORIGINS=https://stackrun.vercel.app,https://www.stackrun.vercel.app

# Worker
WORKER_CONCURRENCY=10

# Rate Limiting
RATE_LIMIT_PER_MINUTE=100

# Logging
LOG_LEVEL=info

# Domain
DOMAIN=stackrun.vercel.app
API_DOMAIN=api.stackrun.vercel.app
APP_DOMAIN=*.stackrun.vercel.app
EOF

# Proteger arquivos
chmod 600 /opt/stackrun/secrets/*
chmod 600 /opt/stackrun/.env.production

echo ""
echo "✅ Segredos gerados com sucesso!"
echo ""
echo "📁 Arquivos criados:"
echo "   /opt/stackrun/secrets/postgres_password.txt"
echo "   /opt/stackrun/secrets/redis_password.txt"
echo "   /opt/stackrun/secrets/jwt_secret.txt"
echo "   /opt/stackrun/secrets/nextauth_secret.txt"
echo "   /opt/stackrun/.env.production"
echo ""
echo "🔒 Permissões: 600 (somente root)"
echo ""
echo "⚠️  IMPORTANTE:"
echo "   - Guarde essas senhas em local seguro (ex: gerenciador de senhas)"
echo "   - NUNCA commite esses arquivos no Git"
echo "   - Esses arquivos são necessários para rodar o sistema"
echo ""
echo "📋 Para ver o .env.production:"
echo "   cat /opt/stackrun/.env.production"
