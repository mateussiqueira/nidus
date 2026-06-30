#!/bin/bash
# ============================================
# STACKRUN - Main Deploy Script for VPS
# Execute como root no VPS
# ============================================
set -e

echo "🚀 STACKRUN - Deploy Script para Produção"
echo "========================================"
echo ""

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Funções auxiliais
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Verificar se está rodando como root
if [ "$EUID" -ne 0 ]; then
    log_error "Execute como root: sudo ./deploy.sh"
    exit 1
fi

# Verificar sistema operacional
if ! grep -q "Ubuntu\|Debian" /etc/os-release; then
    log_error "Este script é apenas para Ubuntu/Debian"
    exit 1
fi

echo "📋 Etapas do deploy:"
echo "   1. Atualizar sistema"
echo "   2. Configurar SSH seguro"
echo "   3. Configurar Firewall (UFW)"
echo "   4. Configurar Fail2Ban"
echo "   5. Configurar Docker com segurança"
echo "   6. Aplicar segurança geral"
echo "   7. Gerar segredos"
echo "   8. Instalar Docker"
echo "   9. Clonar repositório"
echo "  10. Configurar volumes"
echo "  11. Iniciar serviços"
echo ""
read -p "Continuar? (y/N): " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_warn "Deploy cancelado"
    exit 1
fi

# Etapa 1: Atualizar sistema
echo ""
log_info "Etapa 1/11: Atualizando sistema..."
apt-get update
apt-get upgrade -y

# Etapa 2: Configurar SSH
echo ""
log_info "Etapa 2/11: Configurando SSH seguro..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
bash "$SCRIPT_DIR/01-ssh-hardening.sh"

# Etapa 3: Configurar Firewall
echo ""
log_info "Etapa 3/11: Configurando Firewall..."
bash "$SCRIPT_DIR/02-firewall.sh"

# Etapa 4: Configurar Fail2Ban
echo ""
log_info "Etapa 4/11: Configurando Fail2Ban..."
bash "$SCRIPT_DIR/03-fail2ban.sh"

# Etapa 5: Configurar Docker
echo ""
log_info "Etapa 5/11: Configurando Docker com segurança..."
bash "$SCRIPT_DIR/04-docker-security.sh"

# Etapa 6: Segurança geral
echo ""
log_info "Etapa 6/11: Aplicando segurança geral..."
bash "$SCRIPT_DIR/05-general-security.sh"

# Etapa 7: Gerar segredos
echo ""
log_info "Etapa 7/11: Gerando segredos..."
bash "$SCRIPT_DIR/06-generate-secrets.sh"

# Etapa 8: Instalar Docker
echo ""
log_info "Etapa 8/11: Verificando Docker..."
if ! command -v docker &>/dev/null; then
    log_error "Docker não encontrado. Execute o script 04-docker-security.sh primeiro"
    exit 1
fi

# Etapa 9: Clonar repositório
echo ""
log_info "Etapa 9/11: Clonando repositório StackRun..."
mkdir -p /opt
if [ -d "/opt/nidus" ]; then
    log_warn "Diretório /opt/nidus já existe. Atualizando..."
    cd /opt/nidus
    git pull origin main
else
    git clone https://github.com/seu-usuario/nidus.git /opt/nidus
    cd /opt/nidus
fi

# Etapa 10: Configurar volumes
echo ""
log_info "Etapa 10/11: Configurando volumes..."
mkdir -p /opt/stackrun/data/{postgres,redis,deploys}
mkdir -p /opt/stackrun/logs/{caddy,api,worker,dashboard,proxy}
chown -R deploy:deploy /opt/nidus

# Etapa 11: Iniciar serviços
echo ""
log_info "Etapa 11/11: Iniciando serviços..."
cd /opt/nidus

# Copiar .env.production para .env
if [ ! -f ".env" ]; then
    cp .env.production .env
fi

# Iniciar com docker-compose
docker-compose -f docker-compose.prod.yml up -d --build

echo ""
echo "========================================"
echo "✅ DEPLOY CONCLUÍDO!"
echo "========================================"
echo ""
echo "📍 Serviços:"
echo "   - API: https://api.stackrun.vercel.app"
echo "   - Dashboard: https://stackrun.vercel.app"
echo "   - Apps: https://*.stackrun.vercel.app"
echo ""
echo "🔐 Credenciais (salvas em /opt/stackrun/.env.production):"
echo "   PostgreSQL: nidus / [ver .env.production]"
echo "   Redis: [ver .env.production]"
echo "   JWT Secret: [ver .env.production]"
echo ""
echo "📊 Comandos úteis:"
echo "   docker-compose -f docker-compose.prod.yml ps      # Ver status"
echo "   docker-compose -f docker-compose.prod.yml logs -f # Ver logs"
echo "   docker-compose -f docker-compose.prod.yml restart # Reiniciar"
echo "   docker-compose -f docker-compose.prod.yml down    # Parar"
echo ""
echo "🔍 Verificação de segurança:"
echo "   /usr/local/bin/stackrun-security-check.sh"
echo ""
echo "📝 Próximos passos:"
echo "   1. Configure seu domínio (stackrun.vercel.app) para apontar para este IP"
echo "   2. Configure DNS: api.stackrun.vercel.app → IP do VPS"
echo "   3. Configure DNS: *.stackrun.vercel.app → IP do VPS"
echo "   4. Acesse https://stackrun.vercel.app para criar sua conta"
echo "   5. Execute /usr/local/bin/stackrun-audit.sh para auditoria"
echo ""
echo "⚠️  IMPORTANTE:"
echo "   - SSH está configurado para chave pública apenas"
echo "   - Firewall está ativo (UFW)"
echo "   - Fail2Ban está protegendo contra brute force"
echo "   - Docker está configurado com segurança"
echo "   - TLS automático via Let's Encrypt (Caddy)"
