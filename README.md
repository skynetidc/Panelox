<p align="center">
  <h1 align="center">Panelox</h1>
  <p align="center">
    Modern, open-source hosting control panel built with Go + React.
    <br />
    A lightweight, fast alternative to cPanel, Plesk, and CloudPanel.
  </p>
</p>

<p align="center">
  <a href="#features">Features</a> &bull;
  <a href="#comparison">Comparison</a> &bull;
  <a href="#quick-start">Quick Start</a> &bull;
  <a href="#development">Development</a> &bull;
  <a href="#architecture">Architecture</a> &bull;
  <a href="#license">License</a>
</p>

---

## Why Panelox?

Most hosting panels are either **expensive** (cPanel $15/mo+, Plesk $10/mo+), **bloated** (Webmin/Virtualmin), or **limited** (CloudPanel, HestiaCP). Panelox is different:

- **Single binary** — one Go binary + static frontend, no PHP/Python/Perl runtime needed
- **Modern stack** — Go backend (fast, low memory) + React frontend (responsive, real-time)
- **38+ modules** — more features than most paid panels
- **Multi-server ready** — Core + Agent architecture for managing multiple VPS nodes
- **Open source** — MIT license, no per-server fees, no feature gates

## Comparison

| Feature | Panelox | cPanel | Plesk | CloudPanel | HestiaCP | CyberPanel |
|---------|:-------:|:------:|:-----:|:----------:|:--------:|:----------:|
| **Price** | Free (MIT) | $15+/mo | $10+/mo | Free | Free | Free |
| **Websites + Nginx** | Yes | Yes (Apache) | Yes | Yes | Yes (Apache/Nginx) | Yes (OpenLiteSpeed) |
| **Let's Encrypt SSL** | Yes | Yes | Yes | Yes | Yes | Yes |
| **DNS management** | Yes | Yes | Yes | No | Yes | Yes |
| **Mail server** | Yes | Yes | Yes | No | Yes | Yes |
| **Multiple databases** | MySQL + PG + Redis | MySQL + PG | MySQL + PG | MySQL | MySQL + PG | MySQL + PG |
| **Docker management** | Full (Compose, images, volumes, networks) | No | Docker ext. | No | No | Docker ext. |
| **App store (1-click)** | 30+ apps | Softaculous ($) | 100+ apps | No | No | 1-click WP |
| **File manager** | Monaco editor + search + recycle bin | Yes | Yes | Yes | Yes | Yes |
| **Web terminal** | Yes (multi-tab, xterm.js) | Yes | Yes | No | No | Yes |
| **Git deploy** | Yes (push-to-deploy, webhooks) | Yes (basic) | Git ext. | No | No | Git ext. |
| **Firewall (iptables)** | Yes | CSF ($) | Yes | Yes (UFW) | Yes | Yes |
| **Backup (S3/SFTP)** | Yes + scheduling | Yes ($) | Yes | No | Yes | Yes |
| **System monitoring** | Yes (30-day, per-container) | No | No | No | No | No |
| **Web analytics** | Yes (built-in, no JS) | AWStats | No | No | No | No |
| **Fail2Ban** | Yes | Yes | Yes | No | Yes | Yes |
| **2FA / TOTP** | Yes | Yes | Yes | Yes | No | No |
| **RBAC + quotas** | Yes | Yes | Yes | No | Yes (packages) | Yes |
| **Reseller support** | Yes (packages + billing) | Yes ($) | Yes ($) | No | Yes | Yes |
| **Multi-server / nodes** | Yes (Core + Agent) | WHM/DNS clusters | Multi-server ext. | No | No | No |
| **White-label branding** | Yes | cPanel retains branding | Yes ($) | No | Yes | No |
| **API webhooks** | 17 events + HMAC | No | No | No | No | No |
| **ClamAV antivirus** | Yes | Yes ($) | No | No | Yes | No |
| **PHP versions** | 5.6–8.3 (Docker) | 5.6–8.3 | 5.6–8.3 | 7.1–8.3 | 5.6–8.3 | 7.0–8.2 |
| **HTTP/3 + HSTS** | Yes | No | Yes | No | No | No |
| **Load balancing** | Yes (round-robin, least-conn, IP hash) | No | No | No | No | No |
| **Staging environments** | Yes | No | No | No | No | No |
| **Process manager** | Yes | No | No | No | No | No |
| **System cleanup tools** | Yes | No | No | No | No | No |
| **API keys** | Yes | Yes | Yes | No | Yes | No |
| **Audit log** | Yes | Yes | Yes | No | No | No |
| **Backend** | Go | Perl/PHP | PHP | Node.js | PHP/Bash | Python |
| **RAM usage** | ~30 MB | ~500+ MB | ~300+ MB | ~100 MB | ~200+ MB | ~200+ MB |

### TL;DR

- **vs cPanel/Plesk** — same features, zero cost, fraction of the RAM, modern UI
- **vs CloudPanel** — Panelox adds DNS, mail, Docker, app store, Git deploy, monitoring, multi-server
- **vs HestiaCP** — Panelox adds Docker, app store, Git deploy, web terminal, analytics, staging, load balancing
- **vs CyberPanel** — Panelox uses Nginx (not OpenLiteSpeed), has full Docker management, webhooks, multi-server

## Features

### Core Hosting
- **Website management** — static, PHP (5.6–8.3), reverse proxy, Docker container sites
- **SSL/TLS** — Let's Encrypt auto-renewal, wildcard, SAN, manual upload, DNS challenge
- **DNS** — full zone management (A, AAAA, CNAME, MX, TXT, SRV, NS...)
- **Email** — mail domains, mailboxes with quotas, DKIM
- **Databases** — MySQL/MariaDB, PostgreSQL, Redis with user management
- **Firewall** — iptables rules (TCP/UDP/ICMP, port ranges, IP filtering)
- **FTP** — user management with quotas and home directories

### Developer Tools
- **File manager** — Monaco editor (20+ languages), drag & drop upload, compression, search, recycle bin
- **Web terminal** — multi-tab WebSocket terminal (xterm.js) with SSH key management
- **Docker** — Compose projects, images, volumes, networks, per-container metrics
- **App store** — 30+ one-click apps (WordPress, Nextcloud, Grafana, Jenkins, n8n, phpMyAdmin...)
- **Git deploy** — push-to-deploy via GitHub/GitLab webhooks, auto-build, rollback
- **Cron jobs** — scheduler with execution history and failure notifications
- **Staging** — staging subdomain creation, sync from production, promote to production

### Security
- **Authentication** — JWT + refresh tokens, 2FA/TOTP, API keys
- **Fail2Ban** — brute-force protection with configurable thresholds
- **Audit log** — all changes tracked with user, IP, timestamp
- **Login history** — successful/failed attempts with IP and user agent
- **ClamAV** — antivirus scanning, quarantine, auto-update definitions

### Monitoring & Analytics
- **System monitoring** — CPU, RAM, disk, network, I/O metrics with 30-day history
- **Container metrics** — per-container CPU/RAM/network/block I/O
- **Web analytics** — visitor stats, top pages, referrers, status codes (no external JS)
- **Alerts** — configurable rules with Telegram, Discord, email, webhook delivery

### Infrastructure
- **Multi-server** — Core + Agent architecture, manage multiple VPS nodes from one panel
- **Load balancing** — round-robin, least connections, IP hash upstream configs
- **HTTP/3 + HSTS** — modern protocol support per website
- **Backups** — S3/SFTP/local storage with scheduling and retention policies
- **Snapshots** — full system snapshots with point-in-time restore

### Business
- **RBAC** — role-based access control
- **Quotas** — per-user limits (websites, databases, disk, bandwidth)
- **Reseller packages** — quota templates with pricing for reseller hosting
- **White-label** — custom branding (name, logo, colors, favicon)
- **Webhooks** — 17 events with HMAC-SHA256 signatures

## Quick Start

### One-line install (Linux)

```bash
curl -sSL https://raw.githubusercontent.com/jakub961241/Panelox/main/install.sh | sudo bash
```

Supports Ubuntu/Debian/CentOS/RHEL/Rocky/AlmaLinux/Fedora. Installs Nginx, PostgreSQL, Docker automatically.

### Manual setup

```bash
# 1. Clone
git clone https://github.com/jakub961241/Panelox.git
cd Panelox

# 2. Start PostgreSQL + Redis
docker compose -f docker/compose.dev.yml up -d

# 3. Build backend
cd backend
go build -o panelox ./cmd/panelox
cd ..

# 4. Build frontend
cd frontend
npm install
npm run build
cd ..

# 5. Run
cd backend
./panelox
# Panel runs on http://localhost:8080
```

### First login

1. Open `http://YOUR_SERVER_IP:8080`
2. Register first account (automatically becomes admin)
3. Or use seed data: `admin@panelox.io` / `admin123`

## Development

### Prerequisites
- Go 1.21+
- Node.js 18+
- Docker + Docker Compose
- PostgreSQL 16 + Redis 7

### Using Makefile

```bash
make dev          # Start Docker services + backend + frontend
make build        # Cross-compile for Linux (amd64)
make test         # Run Go tests
make migrate      # Run database migrations
make seed         # Load test data
make lint         # Go vet analysis
make release      # Package for distribution
```

### Cross-compile from Windows

```bash
cd backend
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o panelox ./cmd/panelox
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o panelox-agent ./cmd/panelox-agent
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o panelox-core ./cmd/panelox-core
```

### Frontend dev server

```bash
cd frontend
npm install
npm run dev
# Opens on http://localhost:5173, proxies /api to :8080
```

### Docker (full stack)

```bash
docker compose -f docker/compose.fulldev.yml up -d
```

## Architecture

```
panelox/
├── backend/
│   ├── cmd/
│   │   ├── panelox/          # Single-server mode (all-in-one)
│   │   ├── panelox-agent/    # Agent (runs on each VPS)
│   │   └── panelox-core/     # Core (UI + proxy to agents)
│   ├── internal/             # 38+ feature modules
│   │   ├── auth/             #   JWT + 2FA authentication
│   │   ├── website/          #   Website management + nginx
│   │   ├── ssl/              #   Let's Encrypt ACME
│   │   ├── dns/              #   DNS zone management
│   │   ├── mail/             #   Mail domains + mailboxes
│   │   ├── database/         #   MySQL/PostgreSQL/Redis
│   │   ├── firewall/         #   iptables rules
│   │   ├── filemanager/      #   File browser + editor
│   │   ├── terminal/         #   WebSocket terminal
│   │   ├── container/        #   Docker management
│   │   ├── appstore/         #   One-click app install
│   │   ├── monitoring/       #   System metrics
│   │   ├── backup/           #   S3/SFTP/local backups
│   │   ├── gitdeploy/        #   Push-to-deploy
│   │   └── ...               #   + 24 more modules
│   ├── migrations/           # 17 SQL migrations (auto-run)
│   └── pkg/                  # Shared packages
├── frontend/                 # React 19 + TypeScript + Vite
│   └── src/pages/            # 33 pages
├── docker/                   # Dev + prod compose files
├── marketing/                # Landing page
├── install.sh                # Production installer
└── Makefile
```

### Module pattern

Each backend module follows: `handler.go` → `service.go` → `repository.go` → `model.go`

### API

~194 endpoints under `/api/v1/`. JWT Bearer auth required (except login/register).

```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@panelox.io","password":"admin123"}'

# Use token
curl http://localhost:8080/api/v1/dashboard/overview \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PANELOX_PORT` | `8080` | Panel HTTP port |
| `PANELOX_DATABASE_URL` | `postgres://...localhost:5432/panelox` | PostgreSQL connection |
| `PANELOX_JWT_SECRET` | — | JWT signing key (min 64 chars) |
| `PANELOX_ENCRYPTION_KEY` | — | Encryption key (32 bytes) |
| `PANELOX_CORS_ORIGINS` | `http://localhost:5173` | Allowed CORS origins |
| `PANELOX_LOG_LEVEL` | `debug` | Log level |

## License

[MIT](LICENSE) — free for personal and commercial use. No per-server fees.
