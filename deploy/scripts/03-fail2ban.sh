#!/bin/bash
# ============================================
# STACKRUN - Fail2Ban Configuration
# Execute como root no VPS
# ============================================
set -e

echo "🛡️  Configurando Fail2Ban..."

# Instalar fail2ban se não existir
if ! command -v fail2ban-client &>/dev/null; then
    echo "📦 Instalando Fail2Ban..."
    apt-get update
    apt-get install -y fail2ban
fi

# Criar configuração do StackRun
cat > /etc/fail2ban/jail.d/nidus.conf << 'EOF'
[DEFAULT]
# Banir por 1 hora
bantime = 3600

# Janela de tempo de 10 minutos
findtime = 600

# Maximo de tentativas
maxretry = 3

# Banir permanentemente após 3 bans
bantime.increment = true
bantime.factor = 2
bantime.maxtime = 604800

# Ação padrão
action = %(action_mwl)s

[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
bantime = 3600

[stackrun-api]
enabled = true
port = 3001
filter = stackrun-api
logpath = /var/log/caddy/api.log
maxretry = 10
bantime = 600

[stackrun-dashboard]
enabled = true
port = 3000
filter = stackrun-dashboard
logpath = /var/log/caddy/dashboard.log
maxretry = 10
bantime = 600

[stackrun.vercel.apps]
enabled = true
port = 3080
filter = stackrun.vercel.apps
logpath = /var/log/caddy/apps.log
maxretry = 20
bantime = 300
EOF

# Criar filtro para API
cat > /etc/fail2ban/filter.d/stackrun-api.conf << 'EOF'
[Definition]
failregex = ^.*"remote_ip":"<HOST>".*"status":(401|403|429).*$
ignoreregex =
EOF

# Criar filtro para Dashboard
cat > /etc/fail2ban/filter.d/stackrun-dashboard.conf << 'EOF'
[Definition]
failregex = ^.*"remote_ip":"<HOST>".*"status":(401|403|429).*$
ignoreregex =
EOF

# Criar filtro para Apps
cat > /etc/fail2ban/filter.d/stackrun.vercel.apps.conf << 'EOF'
[Definition]
failregex = ^.*"remote_ip":"<HOST>".*"status":(401|403|429).*$
ignoreregex =
EOF

# Criar diretório de logs se não existir
mkdir -p /var/log/caddy
touch /var/log/caddy/access.log
touch /var/log/caddy/api.log
touch /var/log/caddy/dashboard.log
touch /var/log/caddy/apps.log

# Reiniciar fail2ban
echo "🔄 Reiniciando Fail2Ban..."
systemctl restart fail2ban
systemctl enable fail2ban

echo "✅ Fail2Ban configurado!"
echo ""
fail2ban-client status
echo ""
echo "📍 Jails ativos:"
fail2ban-client status sshd
echo ""
echo "⚠️  Para ver bans: fail2ban-client status sshd"
echo "⚠️  Para desbanir IP: fail2ban-client set sshd unbanip <IP>"
