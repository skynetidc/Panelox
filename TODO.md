# Panelox - TODO

## ~~Vlna 1: Bezpečnost + Stabilita~~ ✅ HOTOVO

- [x] **Fail2Ban** - Ochrana proti brute-force (SSH, panel login), ban IP po X pokusech
- [x] **Notifications** - Telegram/Discord/Email/Webhook alerty
- [x] **Alert Rules** - Konfigurovatelná pravidla: "když CPU > 90% po 5 min → pošli alert"
- [x] **Login Log** - Historie přihlášení (IP, čas, user agent, úspěch/neúspěch)
- [x] **2FA/TOTP** - Dvoufaktorová autentizace (Google Authenticator, Authy)
- [x] **Snapshots + Rollback** - Celý systém snapshot (soubory + DB + nginx configs) → restore

## ~~Vlna 2: Docker plný support~~ ✅ HOTOVO

- [x] **Docker Compose** - Správa compose projektů, up/down/restart/logs/config edit/delete
- [x] **Docker Networks** - Vytváření/správa/mazání Docker sítí
- [x] **Docker Volumes** - Správa Docker volumes, prune unused
- [x] **Docker Images** - Pull/delete images, prune unused
- [x] **Redis management** - Redis servery (Docker), INFO, CLI execute, start/stop

## ~~Vlna 3: Konkurenční výhoda~~ ✅ HOTOVO

- [x] **Git Deploy** - Push-to-deploy, GitHub/GitLab webhook, auto-build, rollback, deploy history
- [x] **API Webhooks** - Event webhooks (17 events), HMAC signature, test endpoint
- [x] **White-label** - Custom branding přes settings (brand_name, brand_logo, brand_primary_color...)
- [x] **Resource Billing** - Usage per user (disk, websites, databases), history, collect endpoint
- [x] **Staging/Preview** - Staging subdomain, sync from production, promote to production
- [x] **Web Analytics** - Nginx access log parser (visitors, top pages, referers, status codes, hourly hits)
- [ ] **Plugin System** - Go pluginy pro rozšíření panelu (custom moduly bez forku)

## ~~Vlna 4: Vylepšení existujících features~~ ✅ VĚTŠINA HOTOVO

### ~~Backup~~ ✅
- [x] Backup do S3 (AWS, MinIO, Backblaze B2) - přes AWS CLI
- [x] Backup do SFTP / remote server - přes scp/sshpass
- [ ] Backup do Google Drive / OneDrive (TODO: OAuth integration)
- [x] Scheduled backups - hourly/daily/weekly s účtem storage
- [x] Backup retention policy (max days + max count + auto cleanup)

### ~~File Manager~~ ✅
- [x] Drag & drop upload
- [x] Compress/decompress (ZIP, tar.gz, tar.bz2)
- [x] Search v souborech (grep) s file type filtrem
- [x] Multi-select (bulk delete, move, chmod, compress)
- [x] Monaco Editor (syntax highlighting 20+ jazyků, dark theme)
- [x] Recycle Bin (smazané soubory do koše, restore, empty)

### ~~App Store~~ ✅ (catalog)
- [x] 30 apps v katalogu (WordPress, Nextcloud, Ghost, Grafana, Prometheus, Jenkins, n8n, NocoDB, Adminer, phpMyAdmin, pgAdmin, Vaultwarden, Traefik, Caddy, RabbitMQ, MongoDB, Metabase, Plausible, Duplicati, FileBrowser, MailHog, Watchtower, Elasticsearch, Drone CI + originals)
- [ ] Version management (upgrade/downgrade apps)
- [ ] Custom app repository URL
- [ ] App health checks

### ~~Monitoring~~ ✅
- [x] Per-container CPU/RAM/Net/Block I/O (docker stats)
- [x] Network traffic per interface (/proc/net/dev)
- [x] Disk I/O metriky (/proc/diskstats) per device
- [x] Bandwidth per website (nginx access log parsing) + bar chart
- [x] 30-day history (retention extended, time range selector 6h/24h/3d/7d/30d)

### ~~Terminal~~ ✅
- [x] xterm.js (plný terminal emulator, Tokyo Night theme, barvy, resize)
- [x] Multiple terminal sessions (taby, +/close, přepínání)
- [x] SSH key management (Ed25519 generate, list, delete)

### ~~Cron Jobs~~ ✅
- [x] Standard cron syntax (`*/5 * * * *`) parser (minute, hour, day, month, weekday)
- [x] Výstup archiv (posledních N runů) - cronjob_history tabulka + API
- [x] Email/notification na selhání (přes notification modul)

### ~~Node Management~~ ✅
- [x] SSH key generate (Ed25519) + list + delete
- [x] Node health dashboard (CPU/RAM/disk per node, SSH collect)
- [x] Migrate website between nodes (SSH rsync)
- [x] Node groups (regions: EU, US, Asia)

### ~~RBAC + Multi-tenancy~~ ✅
- [x] Resource quotas per user (max weby, DB, FTP, mail, disk, bandwidth)
- [x] Reseller packages (quota templates s cenou, assign to users)
- [x] Per-user billing dashboard (usage bars, quota vs used)
- [x] User suspension (disable all websites) + unsuspend

## ~~Vlna 5: Ostatní features~~ ✅ HOTOVO

- [x] **Process Manager** - ps aux seznam, kill s signal (TERM/KILL/HUP)
- [x] **ClamAV Antivirus** - status, scan path, update DB
- [x] **System Cleanup** - logs/docker/journal/cache/all cleanup
- [x] **SSH Management** - config (port, root login, password auth), authorized_keys CRUD
- [x] **Disk Management** - du per website
- [x] **URL Rewrite rules** - nginx rewrite per web (redirect, permanent, last, break)
- [x] **Load Balancing** - upstream config (round-robin, least_conn, ip_hash) per web
- [ ] **Website CA** - Self-signed CA (TODO)
- [x] **Compose Templates** - LAMP, LEMP, MEAN, MERN stack templates v App Store (34 apps total)
- [ ] **AI Deployment** - Ollama (TODO)

## Hotovo ✅

- [x] Foundation (auth, config, DB, migrations, middleware)
- [x] Websites + SSL + DNS + Mail + Databases + Firewall
- [x] File Manager (Monaco Editor, drag&drop, compress, search, multi-select, recycle bin)
- [x] Terminal (WebSocket)
- [x] Docker (Containers + Compose + Networks + Volumes + Images)
- [x] App Store (8 apps, one-click install)
- [x] Cron Jobs + Backups (S3/SFTP/local, scheduled, retention)
- [x] Monitoring + Dashboard grafy (recharts, CPU/RAM/Disk)
- [x] PHP Runtime management (multi-version Docker)
- [x] FTP uživatelé
- [x] RBAC + Audit log + Settings + Auto-update
- [x] Security (Fail2Ban, 2FA/TOTP, Login Log, Notifications, Alerts, Snapshots)
- [x] HSTS / HTTP3 / Basic Auth / Anti-Hotlink / CORS / Redirects / Log viewer
- [x] Instalační skript + Systemd service
- [x] Core + Agent architektura (multi-VPS, Unix socket + SSH proxy)
- [x] Node management (SSH, install agent, ping all)
- [x] open_basedir izolace per-web
- [x] Git Deploy (webhook receiver, auto-build, rollback, history)
- [x] API Webhooks (17 events, HMAC signature)
- [x] White-label branding
- [x] Resource Billing (usage per user)
- [x] Staging/Preview environments
- [x] Web Analytics (nginx log parser)
- [x] Redis management (Docker, CLI, INFO)
- [x] i18n (EN + CZ)

## Statistiky

| Metrika | Hodnota |
|---------|---------|
| Backend moduly | 38+ |
| API endpointy | ~194 |
| DB migrace | 10 |
| Frontend stránky | 25+ |
| Sidebar položky | 20 |
| Frontend build | 936kB (263kB gzip) |
| Go binaries | panelox, panelox-core, panelox-agent |
