#!/bin/bash
set -e

# Panelox Hosting Panel - Installation Script
# Usage: curl -sSL https://get.panelox.io/install.sh | bash

PANELOX_VERSION="${PANELOX_VERSION:-latest}"
PANELOX_PORT="${PANELOX_PORT:-8080}"
PANELOX_DIR="/opt/panelox"
PANELOX_USER="panelox"
PANELOX_DB_PASSWORD=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 24)
PANELOX_JWT_SECRET=$(head -c 64 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 64)
PANELOX_ENC_KEY=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${GREEN}[Panelox]${NC} $1"; }
warn() { echo -e "${YELLOW}[Warning]${NC} $1"; }
error() { echo -e "${RED}[Error]${NC} $1"; exit 1; }

# Check root
if [ "$EUID" -ne 0 ]; then
    error "Please run as root: sudo bash install.sh"
fi

# Detect OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
    OS_VERSION=$VERSION_ID
else
    error "Unsupported operating system"
fi

echo -e "${BLUE}"
echo "  ____                  _            "
echo " |  _ \ __ _ _ __   ___| | _____  __ "
echo " | |_) / _\` | '_ \ / _ \ |/ _ \ \/ / "
echo " |  __/ (_| | | | |  __/ | (_) >  <  "
echo " |_|   \__,_|_| |_|\___|_|\___/_/\_\ "
echo ""
echo -e "${NC}"
log "Installing Panelox Hosting Panel..."
log "OS: $OS $OS_VERSION"

# Install dependencies
log "Installing dependencies..."
case $OS in
    ubuntu|debian)
        apt-get update -qq
        apt-get install -y -qq curl wget tar nginx postgresql docker.io docker-compose-plugin >/dev/null 2>&1
        systemctl enable --now docker postgresql
        ;;
    centos|rhel|rocky|alma|fedora)
        dnf install -y -q curl wget tar nginx postgresql-server docker docker-compose-plugin >/dev/null 2>&1
        postgresql-setup --initdb 2>/dev/null || true
        systemctl enable --now docker postgresql
        ;;
    *)
        warn "Unsupported OS: $OS. Trying generic install..."
        ;;
esac

# Create user
log "Creating panelox user..."
id -u $PANELOX_USER &>/dev/null || useradd -r -s /bin/false -d $PANELOX_DIR $PANELOX_USER

# Create directories
log "Creating directories..."
mkdir -p $PANELOX_DIR/{bin,migrations,frontend}
mkdir -p /www/{sites,conf.d,acme-challenge}
mkdir -p /opt/panelox/{backups,apps}
mkdir -p /var/log/panelox

# Setup PostgreSQL
log "Configuring PostgreSQL..."
su - postgres -c "psql -c \"CREATE USER panelox WITH PASSWORD '$PANELOX_DB_PASSWORD';\"" 2>/dev/null || true
su - postgres -c "psql -c \"CREATE DATABASE panelox OWNER panelox;\"" 2>/dev/null || true

# Download Panelox binary
log "Downloading Panelox $PANELOX_VERSION..."
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
esac

# For now, build from source or use pre-built binary
if [ -f "$PANELOX_DIR/bin/panelox" ]; then
    log "Binary already exists, skipping download"
else
    warn "Pre-built binary not available yet. Build from source:"
    warn "  cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=$ARCH go build -ldflags='-s -w' -o $PANELOX_DIR/bin/panelox ./cmd/panelox"
fi

# Create .env file
log "Creating configuration..."
cat > $PANELOX_DIR/.env <<EOF
PANELOX_PORT=$PANELOX_PORT
PANELOX_DATABASE_URL=postgres://panelox:$PANELOX_DB_PASSWORD@localhost:5432/panelox?sslmode=disable
PANELOX_JWT_SECRET=$PANELOX_JWT_SECRET
PANELOX_ENCRYPTION_KEY=$PANELOX_ENC_KEY
PANELOX_CORS_ORIGINS=http://localhost:$PANELOX_PORT
PANELOX_LOG_LEVEL=info
EOF
chmod 600 $PANELOX_DIR/.env

# Create systemd service
log "Creating systemd service..."
cat > /etc/systemd/system/panelox.service <<EOF
[Unit]
Description=Panelox Hosting Panel
After=network.target postgresql.service docker.service
Requires=postgresql.service docker.service

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=$PANELOX_DIR
EnvironmentFile=$PANELOX_DIR/.env
ExecStart=$PANELOX_DIR/bin/panelox
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# Configure nginx for panel
log "Configuring nginx..."
cat > /etc/nginx/conf.d/panelox-panel.conf <<EOF
server {
    listen 80 default_server;
    server_name _;

    # ACME challenge
    location /.well-known/acme-challenge/ {
        root /www/acme-challenge;
    }

    # Managed website configs
    include /www/conf.d/*.conf;
}
EOF

# Include managed configs in nginx
if ! grep -q "include /www/conf.d" /etc/nginx/nginx.conf 2>/dev/null; then
    log "Note: Ensure /www/conf.d/*.conf is included in your nginx config"
fi

# Set permissions
chown -R root:root $PANELOX_DIR
chown -R www-data:www-data /www 2>/dev/null || chown -R nginx:nginx /www 2>/dev/null || true

# Enable and start services
log "Starting services..."
systemctl daemon-reload
systemctl enable panelox
nginx -t && systemctl restart nginx

if [ -f "$PANELOX_DIR/bin/panelox" ]; then
    systemctl start panelox
    log "Panelox started!"
else
    warn "Binary not found. Place it at $PANELOX_DIR/bin/panelox and run: systemctl start panelox"
fi

# Firewall
if command -v ufw &>/dev/null; then
    ufw allow $PANELOX_PORT/tcp 2>/dev/null || true
    ufw allow 80/tcp 2>/dev/null || true
    ufw allow 443/tcp 2>/dev/null || true
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN} Panelox Installation Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e " Panel URL:    ${BLUE}http://$(hostname -I | awk '{print $1}'):$PANELOX_PORT${NC}"
echo -e " Config:       $PANELOX_DIR/.env"
echo -e " Service:      systemctl status panelox"
echo -e " Logs:         journalctl -u panelox -f"
echo ""
echo -e " ${YELLOW}First visit: Register your admin account${NC}"
echo ""
echo -e " DB Password:  $PANELOX_DB_PASSWORD"
echo -e " JWT Secret:   (stored in .env)"
echo ""
