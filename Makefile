.PHONY: dev build backend frontend migrate docker-up docker-down test seed api-test

# =============================================
# Development
# =============================================

dev: docker-up
	@echo "Starting backend and frontend..."
	@cd backend && go run ./cmd/panelox &
	@cd frontend && npm run dev

# =============================================
# Build (cross-compile for Linux from any OS)
# =============================================

build: backend-all frontend

backend:
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o panelox ./cmd/panelox

backend-agent:
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o panelox-agent ./cmd/panelox-agent

backend-core:
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o panelox-core ./cmd/panelox-core

backend-all: backend backend-agent backend-core

# ARM64 builds (Raspberry Pi, AWS Graviton)
backend-arm64:
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o panelox-arm64 ./cmd/panelox

# Local build (for testing on current OS)
backend-local:
	cd backend && go build -o panelox-local ./cmd/panelox

frontend:
	cd frontend && npm install && npm run build

# =============================================
# Database
# =============================================

migrate:
	cd backend && go run ./cmd/panelox -migrate

# =============================================
# Docker
# =============================================

# Dev: just postgres + redis
docker-up:
	docker compose -f docker/compose.dev.yml up -d

docker-down:
	docker compose -f docker/compose.dev.yml down

# Full dev: postgres + redis + backend + frontend (all containers)
docker-full:
	docker compose -f docker/compose.fulldev.yml up -d --build

docker-full-down:
	docker compose -f docker/compose.fulldev.yml down

docker-full-logs:
	docker compose -f docker/compose.fulldev.yml logs -f

# Production
docker-prod:
	docker compose -f docker/compose.prod.yml up -d --build

docker-prod-down:
	docker compose -f docker/compose.prod.yml down

# =============================================
# Testing
# =============================================

test:
	cd backend && go test ./...

# Seed test data (run after backend is started)
seed:
	bash scripts/seed.sh

# Run API test suite
api-test:
	bash scripts/test-api.sh

# =============================================
# Lint
# =============================================

lint:
	cd backend && go vet ./...

# =============================================
# Clean
# =============================================

clean:
	rm -f backend/panelox backend/panelox-agent backend/panelox-core backend/panelox-arm64 backend/panelox-local
	rm -rf frontend/dist frontend/node_modules

# =============================================
# Release (build everything for distribution)
# =============================================

release: clean frontend backend-all
	mkdir -p dist
	cp backend/panelox dist/
	cp backend/panelox-agent dist/
	cp backend/panelox-core dist/
	cp -r backend/migrations dist/
	cp -r frontend/dist dist/frontend
	cp install.sh dist/
	cp README.md dist/
	tar -czf panelox-release.tar.gz -C dist .
	rm -rf dist
	@echo "Release package: panelox-release.tar.gz"
