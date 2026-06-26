#!/bin/bash
# ============================================
# NIDUS - Docker Security Hardening
# Execute como root no VPS
# ============================================
set -e

echo "🐳 Configurando Docker com segurança..."

# Criar grupo docker se não existir
if ! getent group docker | grep -q docker; then
    groupadd docker
fi

# Criar usuário deploy no grupo docker
usermod -aG docker deploy

# Configurações de segurança do Docker
mkdir -p /etc/docker
cat > /etc/docker/daemon.json << 'EOF'
{
    "icc": false,
    "userns-remap": "default",
    "no-new-privileges": true,
    "log-driver": "json-file",
    "log-opts": {
        "max-size": "10m",
        "max-file": "3"
    },
    "storage-driver": "overlay2",
    "live-restore": true,
    "userland-proxy": false,
    "experimental": false,
    "metrics-addr": "127.0.0.1:9323",
    "default-address-pools": [
        {"base": "172.17.0.0/12", "size": 24}
    ]
}
EOF

# Configurar limites de recursos do sistema
cat >> /etc/sysctl.conf << 'EOF'

# ============================================
# NIDUS - Docker Security
# ============================================

# Previne IP spoofing
net.ipv4.conf.all.rp_filter = 1
net.ipv4.conf.default.rp_filter = 1

# Desabilita roteamento entre interfaces
net.ipv4.ip_forward = 0
net.ipv6.conf.all.forwarding = 0

# Previne SYN flood attacks
net.ipv4.tcp_syncookies = 1
net.ipv4.tcp_max_syn_backlog = 2048
net.ipv4.tcp_synack_retries = 2

# Limita conexões ICMP
net.ipv4.icmp_echo_ignore_broadcasts = 1

# Previne IP spoofing
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.default.accept_redirects = 0
net.ipv6.conf.all.accept_redirects = 0
net.ipv6.conf.default.accept_redirects = 0

# Desabilita source routing
net.ipv4.conf.all.accept_source_route = 0
net.ipv4.conf.default.accept_source_route = 0
net.ipv6.conf.all.accept_source_route = 0
net.ipv6.conf.default.accept_source_route = 0
EOF

# Aplicar configurações do sysctl
echo "🔄 Aplicando configurações do kernel..."
sysctl -p

# Configurar limites de recursos
cat >> /etc/security/limits.conf << 'EOF'

# ============================================
# NIDUS - Resource Limits
# ============================================

# Limitar processos por usuário
deploy soft nproc 65535
deploy hard nproc 65535

# Limitar arquivos abertos
deploy soft nofile 65535
deploy hard nofile 65535
EOF

# Instalar docker-compose se não existir
if ! command -v docker-compose &>/dev/null; then
    echo "📦 Instalando Docker Compose..."
    apt-get update
    apt-get install -y docker-compose-plugin
fi

# Reiniciar Docker
echo "🔄 Reiniciando Docker..."
systemctl restart docker
systemctl enable docker

echo "✅ Docker configurado com segurança!"
echo ""
echo "📋 Configurações aplicadas:"
echo "   - User namespace remapping (rootless)"
echo "   - No new privileges"
echo "   - Inter-container communication (ICC) disabled"
echo "   - Live restore enabled"
echo "   - Log rotation configured"
echo "   - Network security hardened"
echo ""
echo "⚠️  IMPORTANTE:"
echo "   - Execute 'newgrp docker' ou faça logout/login para usar Docker"
echo "   - O usuário 'deploy' foi adicionado ao grupo 'docker'"
