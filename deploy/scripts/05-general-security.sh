#!/bin/bash
# ============================================
# NIDUS - General Security Hardening
# Execute como root no VPS
# ============================================
set -e

echo "🔐 Configurando segurança geral..."

# Atualizar sistema
echo "📦 Atualizando sistema..."
apt-get update
apt-get upgrade -y

# Instalar dependências de segurança
echo "📦 Instalando dependências..."
apt-get install -y \
    ufw \
    fail2ban \
    unattended-upgrades \
    apt-listchanges \
    needrestart \
    rkhunter \
    chkrootkit \
    lynis \
    curl \
    git \
    jq \
    htop \
    net-tools \
    dnsutils \
    traceroute

# Configurar atualizações automáticas
echo "🔄 Configurando atualizações automáticas..."
cat > /etc/apt/apt.conf.d/50unattended-upgrades << 'EOF'
Unattended-Upgrade::Allowed-Origins {
    "${distro_id}:${distro_codename}";
    "${distro_id}:${distro_codename}-security";
    "${distro_id}ESMApps:${distro_codename}-apps-security";
    "${distro_id}ESM:${distro_codename}-infra-security";
};

Unattended-Upgrade::Package-Blacklist {
};

Unattended-Upgrade::AutoFixInterruptedDpkg "true";
Unattended-Upgrade::MinimalSteps "true";
Unattended-Upgrade::Remove-Unused-Kernel-Packages "true";
Unattended-Upgrade::Remove-New-Unused-Dependencies "true";
Unattended-Upgrade::Remove-Unused-Dependencies "true";
Unattended-Upgrade::Automatic-Reboot "false";
Unattended-Upgrade::Automatic-Reboot-Time "03:00";
EOF

cat > /etc/apt/apt.conf.d/20auto-upgrades << 'EOF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::AutocleanInterval "7";
EOF

# Configurar sudoers para deploy
echo "👤 Configurando sudoers..."
cat > /etc/sudoers.d/nidus << 'EOF'
# NIDUS Security - Deploy user
deploy ALL=(ALL) NOPASSWD: /usr/bin/docker, /usr/bin/docker-compose, /usr/bin/systemctl restart docker, /usr/bin/systemctl restart caddy
deploy ALL=(ALL) NOPASSWD: /usr/bin/journalctl -u nidus*, /usr/bin/tail -f /var/log/nidus/*
EOF
chmod 0440 /etc/sudoers.d/nidus

# Configurar logs de auditoria
echo "📝 Configurando auditoria..."
apt-get install -y auditd audispd-plugins

cat > /etc/audit/rules.d/nidus.rules << 'EOF'
# NIDUS Audit Rules

# Monitor alterações em arquivos sensíveis
-w /etc/passwd -p wa -k identity
-w /etc/group -p wa -k identity
-w /etc/shadow -p wa -k identity
-w /etc/gshadow -p wa -k identity
-w /etc/sudoers -p wa -k sudoers
-w /etc/sudoers.d/ -p wa -k sudoers

# Monitor comandos Docker
-w /usr/bin/docker -p x -k docker
-w /usr/bin/docker-compose -p x -k docker

# Monitor SSH
-w /etc/ssh/sshd_config -p wa -k sshd_config

# Monitor logs
-w /var/log/auth.log -p wa -k auth_log
-w /var/log/syslog -p wa -k syslog

# Monitor Nidus
-w /opt/nidus/ -p wa -k nidus
EOF

# Reiniciar auditd
systemctl restart auditd
systemctl enable auditd

# Configurar rkhunter
echo "🔍 Configurando RKHunter..."
rkhunter --update
rkhunter --propupd

# Configurarlynis para auditoria
echo "📊 Instalando Lynis para auditoria..."
apt-get install -y lynis

# Criar script de auditoria
cat > /usr/local/bin/nidus-audit.sh << 'EOF'
#!/bin/bash
echo "🔍 Executando auditoria de segurança..."
echo ""
lynis audit system --no-colors
echo ""
echo "📊 Verificação completa!"
echo "📁 Logs em: /var/log/lynis.log"
EOF
chmod +x /usr/local/bin/nidus-audit.sh

# Criar script de verificação de segurança
cat > /usr/local/bin/nidus-security-check.sh << 'EOF'
#!/bin/bash
echo "🔍 Verificação de Segurança Nidus"
echo "=================================="
echo ""

echo "1. Verificando SSH..."
sshd -T | grep -E "(permitrootlogin|passwordauthentication|protocol)" || true
echo ""

echo "2. Verificando Firewall..."
ufw status verbose
echo ""

echo "3. Verificando Fail2Ban..."
fail2ban-client status
echo ""

echo "4. Verificando Docker..."
docker info | grep -E "(Security Options|Root Dir|Logging Driver)"
echo ""

echo "5. Verificando Portas Abertas..."
netstat -tlnp | grep -E "(LISTEN)" || true
echo ""

echo "6. Verificando Usuários com Shell..."
grep -E "bash$|sh$" /etc/passwd || true
echo ""

echo "7. Verificando Ultimos Logins..."
last -10 || true
echo ""

echo "8. Verificando Tentativas de Login..."
journalctl -u sshd --since "1 hour ago" | grep -E "Failed|Invalid" || true
echo ""

echo "✅ Verificação completa!"
EOF
chmod +x /usr/local/bin/nidus-security-check.sh

# Criar script de backup de segurança
cat > /usr/local/bin/nidus-security-backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/opt/nidus/backups/security/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "📦 Criando backup de segurança..."

# Backup configs
cp -r /etc/ssh "$BACKUP_DIR/ssh"
cp -r /etc/fail2ban "$BACKUP_DIR/fail2ban"
cp -r /etc/ufw "$BACKUP_DIR/ufw"
cp -r /etc/docker "$BACKUP_DIR/docker"
cp -r /etc/audit "$BACKUP_DIR/audit"
cp /etc/passwd "$BACKUP_DIR/passwd"
cp /etc/shadow "$BACKUP_DIR/shadow"
cp /etc/group "$BACKUP_DIR/group"
cp /etc/sudoers "$BACKUP_DIR/sudoers"

# Backup logs
cp -r /var/log/auth.log "$BACKUP_DIR/auth.log"
cp -r /var/log/syslog "$BACKUP_DIR/syslog"

# Compactar
tar -czf "$BACKUP_DIR.tar.gz" "$BACKUP_DIR"
rm -rf "$BACKUP_DIR"

echo "✅ Backup criado: $BACKUP_DIR.tar.gz"
echo "📁 Mantendo apenas os últimos 7 backups"
ls -t /opt/nidus/backups/security/*.tar.gz | tail -n +8 | xargs rm -f
EOF
chmod +x /usr/local/bin/nidus-security-backup.sh

# Criar cron para backup diário
cat > /etc/cron.d/nidus-security << 'EOF'
# Backup de segurança diário às 2:00
0 2 * * * root /usr/local/bin/nidus-security-backup.sh
EOF

echo ""
echo "✅ Segurança geral configurada!"
echo ""
echo "📋 Scripts criados:"
echo "   /usr/local/bin/nidus-audit.sh - Auditoria completa"
echo "   /usr/local/bin/nidus-security-check.sh - Verificação rápida"
echo "   /usr/local/bin/nidus-security-backup.sh - Backup de configs"
echo ""
echo "📅 Cron configurado:"
echo "   Backup de segurança diário às 2:00"
echo ""
echo "⚠️  Execute agora: /usr/local/bin/nidus-security-check.sh"
