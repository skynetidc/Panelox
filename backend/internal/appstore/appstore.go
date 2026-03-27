package appstore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/claims"
	"github.com/jakub/panelox/internal/pkg/response"
)

const appsDir = "/opt/panelox/apps"
const portRangeStart = 10000
const portRangeEnd = 60000

type App struct {
	ID          int64           `json:"id"`
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Icon        string          `json:"icon"`
	Category    string          `json:"category"`
	Versions    json.RawMessage `json:"versions"`
	Compose     string          `json:"compose,omitempty"`
	EnvSchema   json.RawMessage `json:"env_schema"`
	CreatedAt   time.Time       `json:"created_at"`
}

type AppInstall struct {
	ID           int64           `json:"id"`
	UserID       int64           `json:"user_id"`
	AppKey       string          `json:"app_key"`
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	Status       string          `json:"status"`
	ContainerIDs []string        `json:"container_ids"`
	EnvValues    json.RawMessage `json:"env_values"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type InstallRequest struct {
	AppKey    string          `json:"app_key"`
	Name      string          `json:"name"`
	Version   string          `json:"version"`
	EnvValues json.RawMessage `json:"env_values"`
}

type Handler struct {
	pool      *pgxpool.Pool
	getClaims func(r *http.Request) *claims.Claims
}

func NewHandler(pool *pgxpool.Pool, getClaims func(r *http.Request) *claims.Claims) *Handler {
	return &Handler{pool: pool, getClaims: getClaims}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/apps", func(r chi.Router) {
		r.Get("/catalog", h.ListCatalog)
		r.Get("/installed", h.ListInstalled)
		r.Post("/install", h.Install)
		r.Delete("/installed/{id}", h.Uninstall)
		r.Post("/installed/{id}/operate", h.Operate)
	})
}

func (h *Handler) ListCatalog(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, key, name, description, icon, category, versions, '', env_schema, created_at
		 FROM app_catalog ORDER BY name`)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var apps []App
	for rows.Next() {
		var a App
		rows.Scan(&a.ID, &a.Key, &a.Name, &a.Description, &a.Icon, &a.Category, &a.Versions, &a.Compose, &a.EnvSchema, &a.CreatedAt)
		apps = append(apps, a)
	}

	if len(apps) == 0 {
		apps = seedDefaultCatalog(r.Context(), h.pool)
	}

	response.JSON(w, http.StatusOK, apps)
}

func (h *Handler) ListInstalled(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, user_id, app_key, name, version, status, container_ids, env_values, created_at, updated_at
		 FROM app_installs ORDER BY id DESC`)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var installs []AppInstall
	for rows.Next() {
		var a AppInstall
		rows.Scan(&a.ID, &a.UserID, &a.AppKey, &a.Name, &a.Version, &a.Status, &a.ContainerIDs, &a.EnvValues, &a.CreatedAt, &a.UpdatedAt)
		installs = append(installs, a)
	}
	response.JSON(w, http.StatusOK, installs)
}

func (h *Handler) Install(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var req InstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.AppKey == "" {
		response.BadRequest(w, "app_key is required")
		return
	}

	// Get app from catalog
	var compose string
	err := h.pool.QueryRow(r.Context(), "SELECT compose FROM app_catalog WHERE key = $1", req.AppKey).Scan(&compose)
	if err == pgx.ErrNoRows {
		response.NotFound(w, "app not found in catalog")
		return
	}

	// Auto-generate name if not provided
	if req.Name == "" {
		req.Name = h.generateInstanceName(r.Context(), req.AppKey)
	}

	if req.Version == "" {
		req.Version = "latest"
	}

	// Collect already-used ports from existing installs
	usedPorts := h.getUsedPorts(r.Context())

	// Auto-assign ports and generate passwords from compose template
	envValues := h.autoConfigureEnv(compose, req.EnvValues, usedPorts)

	envJSON, _ := json.Marshal(envValues)

	var install AppInstall
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO app_installs (user_id, app_key, name, version, status, env_values)
		 VALUES ($1, $2, $3, $4, 'installing', $5)
		 RETURNING id, user_id, app_key, name, version, status, container_ids, env_values, created_at, updated_at`,
		c.UserID, req.AppKey, req.Name, req.Version, envJSON,
	).Scan(&install.ID, &install.UserID, &install.AppKey, &install.Name, &install.Version, &install.Status, &install.ContainerIDs, &install.EnvValues, &install.CreatedAt, &install.UpdatedAt)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	go runInstall(h.pool, &install, compose)
	response.JSON(w, http.StatusAccepted, install)
}

func (h *Handler) Uninstall(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var name string
	h.pool.QueryRow(r.Context(), "SELECT name FROM app_installs WHERE id = $1", id).Scan(&name)

	appDir := filepath.Join(appsDir, name)
	exec.Command("docker", "compose", "-f", filepath.Join(appDir, "docker-compose.yml"), "down", "-v").Run()
	os.RemoveAll(appDir)

	h.pool.Exec(r.Context(), "DELETE FROM app_installs WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "uninstalled"})
}

func (h *Handler) Operate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct{ Action string `json:"action"` }
	json.NewDecoder(r.Body).Decode(&body)

	var name string
	h.pool.QueryRow(r.Context(), "SELECT name FROM app_installs WHERE id = $1", id).Scan(&name)
	composeFile := filepath.Join(appsDir, name, "docker-compose.yml")

	switch body.Action {
	case "start":
		exec.Command("docker", "compose", "-f", composeFile, "up", "-d").Run()
		h.pool.Exec(r.Context(), "UPDATE app_installs SET status = 'running', updated_at = NOW() WHERE id = $1", id)
	case "stop":
		exec.Command("docker", "compose", "-f", composeFile, "stop").Run()
		h.pool.Exec(r.Context(), "UPDATE app_installs SET status = 'stopped', updated_at = NOW() WHERE id = $1", id)
	case "restart":
		exec.Command("docker", "compose", "-f", composeFile, "restart").Run()
		h.pool.Exec(r.Context(), "UPDATE app_installs SET status = 'running', updated_at = NOW() WHERE id = $1", id)
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": body.Action + " completed"})
}

// ─────────────────────────────────────────────────────────────
// Auto-configuration helpers
// ─────────────────────────────────────────────────────────────

var portVarRe = regexp.MustCompile(`\$\{(\w*PORT\w*):-(\d+)\}`)
var passwordVarRe = regexp.MustCompile(`\$\{(\w*(?:PASSWORD|SECRET)\w*):-([^}]*)\}`)
var userVarRe = regexp.MustCompile(`\$\{(\w*(?:USER|USERNAME)\w*):-([^}]*)\}`)

// autoConfigureEnv parses the compose template and auto-assigns ports + generates passwords.
// User-provided env_values override auto-generated ones.
func (h *Handler) autoConfigureEnv(compose string, userEnv json.RawMessage, usedPorts map[int]bool) map[string]string {
	env := make(map[string]string)

	// Auto-assign ports
	for _, match := range portVarRe.FindAllStringSubmatch(compose, -1) {
		varName := match[1]
		defaultPort, _ := strconv.Atoi(match[2])
		port := findFreePort(defaultPort, usedPorts)
		env[varName] = strconv.Itoa(port)
		usedPorts[port] = true
	}

	// Auto-generate passwords (16 hex chars)
	for _, match := range passwordVarRe.FindAllStringSubmatch(compose, -1) {
		varName := match[1]
		env[varName] = generatePassword()
	}

	// Keep default usernames (they're usually fine)
	for _, match := range userVarRe.FindAllStringSubmatch(compose, -1) {
		varName := match[1]
		defaultVal := match[2]
		env[varName] = defaultVal
	}

	// Merge user-provided values (override auto-generated)
	if userEnv != nil {
		var userMap map[string]string
		if json.Unmarshal(userEnv, &userMap) == nil {
			for k, v := range userMap {
				if v != "" {
					env[k] = v
				}
			}
		}
	}

	return env
}

// getUsedPorts returns all ports currently used by installed apps.
func (h *Handler) getUsedPorts(ctx context.Context) map[int]bool {
	used := make(map[int]bool)
	rows, err := h.pool.Query(ctx, "SELECT env_values FROM app_installs WHERE status != 'error'")
	if err != nil {
		return used
	}
	defer rows.Close()

	for rows.Next() {
		var ev json.RawMessage
		rows.Scan(&ev)
		var m map[string]string
		if json.Unmarshal(ev, &m) == nil {
			for k, v := range m {
				if strings.Contains(strings.ToUpper(k), "PORT") {
					if p, err := strconv.Atoi(v); err == nil {
						used[p] = true
					}
				}
			}
		}
	}
	return used
}

// findFreePort finds the next available port starting from preferred.
// Checks both the usedPorts map (other app installs) and actual TCP listeners.
func findFreePort(preferred int, usedPorts map[int]bool) int {
	// Try preferred first
	if !usedPorts[preferred] && !isPortListening(preferred) {
		return preferred
	}
	// Scan from portRangeStart
	for p := portRangeStart; p < portRangeEnd; p++ {
		if !usedPorts[p] && !isPortListening(p) {
			return p
		}
	}
	// Fallback: let OS pick
	return preferred
}

func isPortListening(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return true // port is in use
	}
	ln.Close()
	return false
}

func generatePassword() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateInstanceName creates a unique name like "wordpress", "wordpress-2", etc.
func (h *Handler) generateInstanceName(ctx context.Context, appKey string) string {
	var count int
	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_installs WHERE app_key = $1", appKey).Scan(&count)
	if count == 0 {
		return appKey
	}
	return fmt.Sprintf("%s-%d", appKey, count+1)
}

// ─────────────────────────────────────────────────────────────
// Install runner
// ─────────────────────────────────────────────────────────────

func runInstall(pool *pgxpool.Pool, install *AppInstall, compose string) {
	ctx := context.Background()
	appDir := filepath.Join(appsDir, install.Name)
	os.MkdirAll(appDir, 0755)

	// Write docker-compose.yml
	composeFile := filepath.Join(appDir, "docker-compose.yml")
	os.WriteFile(composeFile, []byte(compose), 0644)

	// Write .env file from env_values
	var envMap map[string]string
	json.Unmarshal(install.EnvValues, &envMap)
	var envContent strings.Builder
	for k, v := range envMap {
		envContent.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}
	os.WriteFile(filepath.Join(appDir, ".env"), []byte(envContent.String()), 0644)

	// Docker compose up
	out, err := exec.Command("docker", "compose", "-f", composeFile, "up", "-d").CombinedOutput()
	if err != nil {
		pool.Exec(ctx, "UPDATE app_installs SET status = 'error', updated_at = NOW() WHERE id = $1", install.ID)
		log.Printf("App install %d failed: %s", install.ID, string(out))
		return
	}

	pool.Exec(ctx, "UPDATE app_installs SET status = 'running', updated_at = NOW() WHERE id = $1", install.ID)
	log.Printf("App %s installed successfully", install.Name)
}

// ─────────────────────────────────────────────────────────────
// Default catalog
// ─────────────────────────────────────────────────────────────

func seedDefaultCatalog(ctx context.Context, pool *pgxpool.Pool) []App {
	catalog := []struct {
		key, name, desc, icon, category, compose string
	}{
		{"wordpress", "WordPress", "Popular CMS for blogs and websites", "https://s.w.org/style/images/about/WordPress-logotype-wmark.png", "CMS",
			"services:\n  wordpress:\n    image: wordpress:latest\n    ports:\n      - \"${PORT:-8080}:80\"\n    environment:\n      WORDPRESS_DB_HOST: db\n      WORDPRESS_DB_USER: wordpress\n      WORDPRESS_DB_PASSWORD: ${DB_PASSWORD:-wordpress}\n      WORDPRESS_DB_NAME: wordpress\n    volumes:\n      - wp_data:/var/www/html\n    depends_on:\n      - db\n  db:\n    image: mysql:8.0\n    environment:\n      MYSQL_ROOT_PASSWORD: ${DB_PASSWORD:-wordpress}\n      MYSQL_DATABASE: wordpress\n      MYSQL_USER: wordpress\n      MYSQL_PASSWORD: ${DB_PASSWORD:-wordpress}\n    volumes:\n      - db_data:/var/lib/mysql\nvolumes:\n  wp_data:\n  db_data:"},
		{"gitea", "Gitea", "Lightweight self-hosted Git service", "https://gitea.io/images/gitea.png", "DevOps",
			"services:\n  gitea:\n    image: gitea/gitea:latest\n    ports:\n      - \"${PORT:-3000}:3000\"\n      - \"${SSH_PORT:-2222}:22\"\n    volumes:\n      - gitea_data:/data\n    environment:\n      GITEA__database__DB_TYPE: sqlite3\nvolumes:\n  gitea_data:"},
		{"redis", "Redis", "In-memory data structure store", "https://redis.io/images/redis-white.png", "Database",
			"services:\n  redis:\n    image: redis:7-alpine\n    ports:\n      - \"${PORT:-6379}:6379\"\n    volumes:\n      - redis_data:/data\n    command: redis-server --requirepass ${PASSWORD:-redis}\nvolumes:\n  redis_data:"},
		{"nginx-proxy-manager", "Nginx Proxy Manager", "Docker reverse proxy with Let's Encrypt", "https://nginxproxymanager.com/icon.png", "Networking",
			"services:\n  npm:\n    image: jc21/nginx-proxy-manager:latest\n    ports:\n      - \"${HTTP_PORT:-80}:80\"\n      - \"${HTTPS_PORT:-443}:443\"\n      - \"${ADMIN_PORT:-81}:81\"\n    volumes:\n      - npm_data:/data\n      - npm_letsencrypt:/etc/letsencrypt\nvolumes:\n  npm_data:\n  npm_letsencrypt:"},
		{"portainer", "Portainer", "Docker management UI", "https://www.portainer.io/hubfs/portainer-logo-black.svg", "DevOps",
			"services:\n  portainer:\n    image: portainer/portainer-ce:latest\n    ports:\n      - \"${PORT:-9443}:9443\"\n    volumes:\n      - /var/run/docker.sock:/var/run/docker.sock\n      - portainer_data:/data\nvolumes:\n  portainer_data:"},
		{"mariadb", "MariaDB", "Community-developed MySQL fork", "https://mariadb.com/wp-content/uploads/2019/11/mariadb-logo.png", "Database",
			"services:\n  mariadb:\n    image: mariadb:11\n    ports:\n      - \"${PORT:-3306}:3306\"\n    environment:\n      MYSQL_ROOT_PASSWORD: ${ROOT_PASSWORD:-mariadb}\n    volumes:\n      - mariadb_data:/var/lib/mysql\nvolumes:\n  mariadb_data:"},
		{"uptime-kuma", "Uptime Kuma", "Self-hosted monitoring tool", "https://uptime.kuma.pet/img/icon.svg", "Monitoring",
			"services:\n  uptime-kuma:\n    image: louislam/uptime-kuma:latest\n    ports:\n      - \"${PORT:-3001}:3001\"\n    volumes:\n      - kuma_data:/app/data\nvolumes:\n  kuma_data:"},
		{"minio", "MinIO", "S3-compatible object storage", "https://min.io/resources/img/logo.svg", "Storage",
			"services:\n  minio:\n    image: minio/minio:latest\n    ports:\n      - \"${API_PORT:-9000}:9000\"\n      - \"${CONSOLE_PORT:-9001}:9001\"\n    environment:\n      MINIO_ROOT_USER: ${ROOT_USER:-minioadmin}\n      MINIO_ROOT_PASSWORD: ${ROOT_PASSWORD:-minioadmin}\n    volumes:\n      - minio_data:/data\n    command: server /data --console-address \":9001\"\nvolumes:\n  minio_data:"},
	}

	var apps []App
	for _, c := range catalog {
		var a App
		pool.QueryRow(ctx,
			`INSERT INTO app_catalog (key, name, description, icon, category, versions, compose, env_schema)
			 VALUES ($1, $2, $3, $4, $5, '["latest"]', $6, '[]')
			 ON CONFLICT (key) DO NOTHING
			 RETURNING id, key, name, description, icon, category, versions, '', env_schema, created_at`,
			c.key, c.name, c.desc, c.icon, c.category, c.compose,
		).Scan(&a.ID, &a.Key, &a.Name, &a.Description, &a.Icon, &a.Category, &a.Versions, &a.Compose, &a.EnvSchema, &a.CreatedAt)
		if a.ID > 0 {
			apps = append(apps, a)
		}
	}
	return apps
}

var ErrNotFound = errors.New("not found")
