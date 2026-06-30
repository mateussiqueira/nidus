#!/bin/bash
# ============================================
# STACKRUN - SSH Hardening Script
# Execute como root no VPS
# ============================================
set -e

echo "🔒 Configurando SSH seguro..."

# Backup da configuração original
cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak.$(date +%Y%m%d)

# Desabilitar login como root
sed -i 's/^PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config
sed -i 's/^#PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config

# Desabilitar autenticação por senha
sed -i 's/^PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
sed -i 's/^#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config

# Desabilitar SSH v1
sed -i 's/^Protocol 1/Protocol 2/' /etc/ssh/sshd_config

# Configurações adicionais de segurança
cat >> /etc/ssh/sshd_config << 'EOF'

# ============================================
# STACKRUN Security Hardening
# ============================================

# Porta customizada (opcional, descomente se quiser)
# Port 2222

# Limitar tentativas de login
MaxAuthTries 3

# Timeout de conexão
ClientAliveInterval 300
ClientAliveCountMax 2

# Desabilitar X11Forwarding
X11Forwarding no

# Desabilitar TCP Forwarding
AllowTcpForwarding no

# Desabilitar GatewayPorts
GatewayPorts no

# Usar apenas SSH v2
Protocol 2

# Limitar usuários
AllowUsers deploy

# Algoritmos de criptografia seguros
KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org,diffie-hellman-group16-sha512,diffie-hellman-group18-sha512
Ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com,aes128-gcm@openssh.com,aes256-ctr,aes192-ctr,aes128-ctr
MACs hmac-sha2-512-etm@openssh.com,hmac-sha2-256-etm@openssh.com,hmac-sha2-512,hmac-sha2-256

# Log level
LogLevel VERBOSE
EOF

# Criar usuário deploy se não existir
if ! id -u deploy &>/dev/null; then
    echo "👤 Criando usuário deploy..."
    useradd -m -s /bin/bash deploy
    usermod -aG sudo deploy
    echo "deploy ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/deploy
    chmod 0440 /etc/sudoers.d/deploy
fi

# Criar diretório .ssh para deploy
mkdir -p /home/deploy/.ssh
chmod 700 /home/deploy/.ssh
touch /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys
chown -R deploy:deploy /home/deploy/.ssh

# Reiniciar SSH
echo "🔄 Reiniciando SSH..."
systemctl restart sshd

echo "✅ SSH configurado!"
echo ""
echo "⚠️  IMPORTANTE: Copie sua chave pública para /home/deploy/.ssh/authorized_keys"
echo "   Antes de desconectar a sessão atual!"
echo ""
echo "   cat sua-chave-publique >> /home/deploy/.ssh/authorized_keys"
