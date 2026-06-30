#!/bin/bash
# ============================================
# STACKRUN - Firewall Configuration (UFW)
# Execute como root no VPS
# ============================================
set -e

echo "🛡️  Configurando firewall..."

# Instalar UFW se não existir
if ! command -v ufw &>/dev/null; then
    echo "📦 Instalando UFW..."
    apt-get update
    apt-get install -y ufw
fi

# Resetar regras
echo "🔄 Resetando regras..."
ufw --force reset

# Políticas padrão
echo "📋 Configurando políticas padrão..."
ufw default deny incoming
ufw default allow outgoing

# SSH (essential - manter porta padrão ou usar porta custom)
echo "🔑 Configurando SSH..."
ufw allow 22/tcp comment "SSH"

# HTTP/HTTPS (Caddy)
echo "🌐 Configurando HTTP/HTTPS..."
ufw allow 80/tcp comment "HTTP"
ufw allow 443/tcp comment "HTTPS"

# API Interna (apenas localhost)
echo "🔌 Configurando API interna..."
ufw allow from 127.0.0.1 to any port 3001 comment "API Local"

# Dashboard (apenas localhost)
echo "📊 Configurando Dashboard interno..."
ufw allow from 127.0.0.1 to any port 3000 comment "Dashboard Local"

# Apps dos usuários (wildcard *.stackrun.vercel.app)
echo "🚀 Configurando Apps dos usuários..."
ufw allow 3080/tcp comment "User Apps Proxy"

# PostgreSQL (apenas Docker network)
echo "🗄️  Configurando PostgreSQL..."
ufw allow from 172.16.0.0/12 to any port 5432 comment "PostgreSQL Docker"

# Redis (apenas Docker network)
echo "📦 Configurando Redis..."
ufw allow from 172.16.0.0/12 to any port 6379 comment "Redis Docker"

# Desabilitar ICMP (ping) - opcional
echo "🔇 Desabilitando ICMP..."
ufw deny proto icmp from any to any

# Logging ativo
echo "📝 Configurando logging..."
ufw logging on
ufw logging medium

# Ativar firewall
echo "⚡ Ativando firewall..."
ufw --force enable

# Mostrar status
echo ""
echo "✅ Firewall configurado!"
echo ""
ufw status verbose
echo ""
echo "⚠️  IMPORTANTE:"
echo "   - SSH está aberto (porta 22) - configure chave pública"
echo "   - HTTP/HTTPS aberto para Caddy"
echo "   - PostgreSQL e Redis bloqueados externamente"
echo "   - ICMP (ping) bloqueado"
