#!/bin/bash
set -e

# Panelox Quick Start Script
# Starts PostgreSQL (Docker) + builds + runs panel

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${GREEN}[Panelox]${NC} $1"; }

echo -e "${BLUE}"
echo "  ____                  _            "
echo " |  _ \ __ _ _ __   ___| | _____  __ "
echo " | |_) / _\` | '_ \ / _ \ |/ _ \ \/ / "
echo " |  __/ (_| | | | |  __/ | (_) >  <  "
echo " |_|   \__,_|_| |_|\___|_|\___/_/\_\ "
echo ""
echo -e "${NC}"

# Check prerequisites
command -v docker >/dev/null 2>&1 || { echo "Docker not found. Install: curl -fsSL https://get.docker.com | sh"; exit 1; }
command -v go >/dev/null 2>&1 || { echo "Go not found. Install: https://go.dev/dl/"; exit 1; }
command -v node >/dev/null 2>&1 || { echo "Node.js not found. Install: https://nodejs.org"; exit 1; }

# Start PostgreSQL
log "Starting PostgreSQL..."
docker compose -f docker/compose.dev.yml up -d

# Wait for PostgreSQL
log "Waiting for PostgreSQL..."
for i in $(seq 1 30); do
  if docker compose -f docker/compose.dev.yml exec -T postgres pg_isready -U panelox >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

# Build backend
log "Building backend..."
cd backend
go build -o panelox ./cmd/panelox
cd ..

# Build frontend
log "Building frontend..."
cd frontend
npm install --silent
npm run build --silent
cd ..

# Copy frontend dist for serving
mkdir -p /opt/panelox/frontend 2>/dev/null || true
cp -r frontend/dist/* /opt/panelox/frontend/ 2>/dev/null || true

# Source env
set -a
source .env
set +a

# Run
log "Starting Panelox on port $PANELOX_PORT..."
cd backend
./panelox &
PANELOX_PID=$!
cd ..

sleep 2

# Seed data
log "Seeding test data..."
bash scripts/seed.sh http://localhost:$PANELOX_PORT/api/v1 2>/dev/null

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN} Panelox is running!${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""
echo -e " Panel:    ${BLUE}http://localhost:$PANELOX_PORT${NC}"
echo -e " Admin:    admin@panelox.io / admin123"
echo -e " User:     user@panelox.io / user123"
echo ""
echo -e " API Test: ${BLUE}make api-test${NC}"
echo -e " Seed:     ${BLUE}make seed${NC}"
echo -e " Stop:     ${BLUE}kill $PANELOX_PID && make docker-down${NC}"
echo ""

# Wait for process
wait $PANELOX_PID
