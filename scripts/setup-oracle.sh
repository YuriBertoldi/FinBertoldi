#!/bin/bash
# ============================================================
# setup-oracle.sh — Instala Docker e prepara o servidor Oracle
# Execute como: bash setup-oracle.sh
# Testado em: Oracle Linux 8/9 e Ubuntu 22.04 (ARM e x86)
# ============================================================
set -e

echo "==> Detectando sistema operacional..."
if [ -f /etc/oracle-release ] || [ -f /etc/redhat-release ]; then
  OS="oracle"
elif [ -f /etc/lsb-release ]; then
  OS="ubuntu"
else
  OS="unknown"
fi

echo "==> Sistema: $OS"

# ---- Instalar Docker ----
if command -v docker &>/dev/null; then
  echo "==> Docker já instalado: $(docker --version)"
else
  echo "==> Instalando Docker..."
  if [ "$OS" = "ubuntu" ]; then
    apt-get update -qq
    apt-get install -y ca-certificates curl gnupg
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
      https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
      > /etc/apt/sources.list.d/docker.list
    apt-get update -qq
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
  elif [ "$OS" = "oracle" ]; then
    dnf install -y dnf-plugins-core
    dnf config-manager --add-repo https://download.docker.com/linux/rhel/docker-ce.repo
    dnf install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
  else
    # Fallback: script oficial
    curl -fsSL https://get.docker.com | sh
  fi
  systemctl enable --now docker
  usermod -aG docker "$USER"
  echo "==> Docker instalado com sucesso!"
fi

# ---- Abrir portas no firewall do sistema ----
echo "==> Configurando firewall do sistema operacional..."
if command -v firewall-cmd &>/dev/null; then
  # Oracle Linux usa firewalld
  firewall-cmd --permanent --add-service=http
  firewall-cmd --permanent --add-service=https
  firewall-cmd --permanent --add-port=8080/tcp
  firewall-cmd --reload
  echo "==> Portas 80, 443 e 8080 abertas via firewalld"
elif command -v ufw &>/dev/null; then
  # Ubuntu usa ufw
  ufw allow 80/tcp
  ufw allow 443/tcp
  ufw allow 8080/tcp
  ufw allow OpenSSH
  ufw --force enable
  echo "==> Portas 80, 443 e 8080 abertas via ufw"
fi

# ---- Criar pasta do app ----
APP_DIR="/opt/finbertoldi"
echo "==> Criando pasta do app em $APP_DIR..."
mkdir -p "$APP_DIR"
chown "$USER":"$USER" "$APP_DIR"

echo ""
echo "======================================================"
echo " Servidor pronto!"
echo ""
echo " Próximos passos:"
echo "   1. Copie os arquivos do app para $APP_DIR"
echo "   2. Configure: cp .env.prod.example .env.prod && nano .env.prod"
echo "   3. Suba o app:  cd $APP_DIR && docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build"
echo "   4. Acesse: http://SEU_IP_PUBLICO"
echo "======================================================"
