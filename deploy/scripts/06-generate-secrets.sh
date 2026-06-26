#!/bin/bash
# ============================================
# NIDUS - Generate Secure Secrets
# Execute uma única vez no VPS
# ============================================
set -e

echo "🔑 Gerando segredos seguros..."

# Criar diretório de secrets
mkdir -p /opt/nidus/secrets
chmod 700 /opt/nidus/secrets

# Gerar senhas seguras (32 caracteres)
POSTGRES_PASSWORD=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
REDIS_PASSWORD=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
JWT_SECRET=$(openssl rand -base64 64 | tr -dc 'a-zA-Z0-9' | head -c 64)
NEXTAUTH_SECRET=$(openssl rand -base64 64 | tr -dc 'a-zA-Z0-9' | head -c 64)

# Salvar secrets em arquivos
echo -n "$POSTGRES_PASSWORD" > /opt/nidus/secrets/postgres_password.txt
echo -n "$REDIS_PASSWORD" > /opt/nidus/secrets/redis_password.txt
echo -n "$JWT_SECRET" > /opt/nidus/secrets/jwt_secret.txt
echo -n "$NEXTAUTH_SECRET" > /opt/nidus/secrets/nextauth_secret.txt

# Gerar .env.production
cat > /opt/nidus/.env.production << EOF
# ============================================
# NIDUS - Production Environment Variables
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
NEXTAUTH_URL=https://nidus.com

# CORS
CORS_ORIGINS=https://nidus.com,https://www.nidus.com

# Worker
WORKER_CONCURRENCY=10

# Rate Limiting
RATE_LIMIT_PER_MINUTE=100

# Logging
LOG_LEVEL=info

# Domain
DOMAIN=nidus.com
API_DOMAIN=api.nidus.com
APP_DOMAIN=*.nidus.com
EOF

# Proteger arquivos
chmod 600 /opt/nidus/secrets/*
chmod 600 /opt/nidus/.env.production

echo ""
echo "✅ Segredos gerados com sucesso!"
echo ""
echo "📁 Arquivos criados:"
echo "   /opt/nidus/secrets/postgres_password.txt"
echo "   /opt/nidus/secrets/redis_password.txt"
echo "   /opt/nidus/secrets/jwt_secret.txt"
echo "   /opt/nidus/secrets/nextauth_secret.txt"
echo "   /opt/nidus/.env.production"
echo ""
echo "🔒 Permissões: 600 (somente root)"
echo ""
echo "⚠️  IMPORTANTE:"
echo "   - Guarde essas senhas em local seguro (ex: gerenciador de senhas)"
echo "   - NUNCA commite esses arquivos no Git"
echo "   - Esses arquivos são necessários para rodar o sistema"
echo ""
echo "📋 Para ver o .env.production:"
echo "   cat /opt/nidus/.env.production"
