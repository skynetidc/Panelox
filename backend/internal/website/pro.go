package website

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/license"
	"github.com/jakub/panelox/internal/pkg/nginx"
	"github.com/jakub/panelox/internal/pkg/response"
)

// ─────────────────────────────────────────────────────────────
// Nginx Snippet Presets (PRO)
// ─────────────────────────────────────────────────────────────

type NginxSnippet struct {
	ID        int64  `json:"id"`
	WebsiteID int64  `json:"website_id"`
	PresetKey string `json:"preset_key"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	Content   string `json:"content"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

// Built-in snippet presets library
var SnippetPresets = []struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Content  string `json:"content"`
}{
	// Security
	{"security-headers", "Security Headers (Full)", "security",
		"add_header Strict-Transport-Security \"max-age=31536000; includeSubDomains; preload\" always;\nadd_header Referrer-Policy \"strict-origin-when-cross-origin\" always;\nadd_header Permissions-Policy \"camera=(), microphone=(), geolocation=()\" always;\nadd_header Content-Security-Policy \"default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:;\" always;"},
	{"block-xmlrpc", "Block XML-RPC (WordPress)", "security",
		"location = /xmlrpc.php {\n    deny all;\n    return 444;\n}"},
	{"block-hidden-files", "Block Hidden Files", "security",
		"location ~ /\\. {\n    deny all;\n    return 404;\n}"},
	{"block-php-uploads", "Block PHP in Uploads (WordPress)", "security",
		"location ~* /(?:uploads|files)/.*\\.php$ {\n    deny all;\n    return 404;\n}"},

	// Caching
	{"wp-super-cache", "WordPress Super Cache", "cache",
		"set $cache_uri $request_uri;\nif ($request_method = POST) { set $cache_uri 'null cache'; }\nif ($query_string != \"\") { set $cache_uri 'null cache'; }\nif ($request_uri ~* \"/wp-admin/|/xmlrpc.php|wp-.*.php|/feed/|sitemap(_index)?.xml\") { set $cache_uri 'null cache'; }\nif ($http_cookie ~* \"comment_author|wordpress_[a-f0-9]+|wp-postpass|wordpress_no_cache|wordpress_logged_in\") { set $cache_uri 'null cache'; }"},
	{"browser-cache-aggressive", "Aggressive Browser Cache (1 year)", "cache",
		"location ~* \\.(jpg|jpeg|png|gif|ico|css|js|woff2?|ttf|eot|svg|webp|avif)$ {\n    expires 365d;\n    add_header Cache-Control \"public, immutable\";\n    access_log off;\n}"},
	{"no-cache-html", "No Cache for HTML", "cache",
		"location ~* \\.html?$ {\n    expires -1;\n    add_header Cache-Control \"no-store, no-cache, must-revalidate\";\n}"},

	// Performance
	{"gzip-compression", "Gzip Compression", "performance",
		"gzip on;\ngzip_vary on;\ngzip_proxied any;\ngzip_comp_level 6;\ngzip_types text/plain text/css text/xml application/json application/javascript application/xml+rss application/atom+xml image/svg+xml;"},
	{"brotli-compression", "Brotli Compression", "performance",
		"brotli on;\nbrotli_comp_level 6;\nbrotli_types text/plain text/css text/xml application/json application/javascript application/xml+rss application/atom+xml image/svg+xml;"},
	{"buffer-tuning", "Buffer Tuning (Large uploads)", "performance",
		"client_max_body_size 256m;\nclient_body_buffer_size 128k;\nproxy_buffer_size 128k;\nproxy_buffers 4 256k;\nproxy_busy_buffers_size 256k;"},

	// Rate limiting
	{"rate-limit-login", "Rate Limit Login Pages", "ratelimit",
		"location ~* /(wp-login\\.php|login|signin|admin/login) {\n    limit_req zone=login burst=5 nodelay;\n    limit_req_status 429;\n    try_files $uri $uri/ /index.php$is_args$args;\n}"},
	{"rate-limit-api", "Rate Limit API Endpoints", "ratelimit",
		"location /api/ {\n    limit_req zone=api burst=20 nodelay;\n    limit_req_status 429;\n    try_files $uri $uri/ /index.php$is_args$args;\n}"},

	// WordPress
	{"wp-permalink", "WordPress Permalinks", "wordpress",
		"location / {\n    try_files $uri $uri/ /index.php$is_args$args;\n}"},
	{"wp-security-full", "WordPress Full Security", "wordpress",
		"location = /xmlrpc.php { deny all; return 444; }\nlocation ~* /(?:uploads|files)/.*\\.php$ { deny all; }\nlocation ~ /\\.(ht|git|svn|env) { deny all; }\nlocation ~* wp-config.php { deny all; }\nlocation ~* readme\\.html { deny all; }"},
	{"wp-fastcgi-cache", "WordPress FastCGI Cache", "wordpress",
		"fastcgi_cache_path /tmp/nginx-cache levels=1:2 keys_zone=wpcache:100m inactive=60m;\nfastcgi_cache_key \"$scheme$request_method$host$request_uri\";\nfastcgi_cache wpcache;\nfastcgi_cache_valid 200 60m;\nfastcgi_cache_valid 404 1m;\nfastcgi_cache_bypass $http_cookie;\nfastcgi_no_cache $http_cookie;"},

	// CORS
	{"cors-permissive", "CORS - Allow All Origins", "cors",
		"add_header Access-Control-Allow-Origin * always;\nadd_header Access-Control-Allow-Methods \"GET, POST, PUT, DELETE, OPTIONS\" always;\nadd_header Access-Control-Allow-Headers \"Authorization, Content-Type, Accept\" always;\nif ($request_method = OPTIONS) {\n    return 204;\n}"},
	{"cors-specific", "CORS - Specific Origin (edit URL)", "cors",
		"set $cors_origin \"\";\nif ($http_origin ~* \"^https://(example\\.com|app\\.example\\.com)$\") {\n    set $cors_origin $http_origin;\n}\nadd_header Access-Control-Allow-Origin $cors_origin always;\nadd_header Access-Control-Allow-Credentials true always;\nadd_header Access-Control-Allow-Methods \"GET, POST, PUT, DELETE, OPTIONS\" always;\nadd_header Access-Control-Allow-Headers \"Authorization, Content-Type\" always;"},
}

func RegisterProRoutes(r chi.Router, h *Handler) {
	// Nginx snippet presets
	r.Get("/websites/snippet-presets", h.ListSnippetPresets)
	r.Get("/websites/{id}/snippets", h.ListSnippets)
	r.Post("/websites/{id}/snippets", h.AddSnippet)
	r.Delete("/websites/{id}/snippets/{snippetId}", h.DeleteSnippet)
	r.Put("/websites/{id}/snippets/{snippetId}/toggle", h.ToggleSnippet)

	// Website cloning
	r.Post("/websites/{id}/clone", h.CloneWebsite)

	// Per-website rate limiting
	r.Get("/websites/{id}/rate-limit", h.GetRateLimit)
	r.Put("/websites/{id}/rate-limit", h.UpdateRateLimit)

	// Per-website IP access control
	r.Get("/websites/{id}/ip-rules", h.ListIPRules)
	r.Post("/websites/{id}/ip-rules", h.AddIPRule)
	r.Delete("/websites/{id}/ip-rules/{ruleId}", h.DeleteIPRule)
	r.Put("/websites/{id}/ip-access-mode", h.SetIPAccessMode)
}

// ── Snippet Handlers ──────────────────────────────────────────

func (h *Handler) ListSnippetPresets(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, SnippetPresets)
}

func (h *Handler) ListSnippets(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	rows, err := h.svc.repo.pool.Query(r.Context(),
		"SELECT id, website_id, preset_key, name, category, content, enabled, created_at FROM nginx_snippets WHERE website_id = $1 ORDER BY id", id)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var snippets []NginxSnippet
	for rows.Next() {
		var s NginxSnippet
		rows.Scan(&s.ID, &s.WebsiteID, &s.PresetKey, &s.Name, &s.Category, &s.Content, &s.Enabled, &s.CreatedAt)
		snippets = append(snippets, s)
	}
	response.JSON(w, http.StatusOK, snippets)
}

func (h *Handler) AddSnippet(w http.ResponseWriter, r *http.Request) {
	if !h.requirePro(w, r) {
		return
	}
	id, _ := parseID(r, "id")

	var body struct {
		PresetKey string `json:"preset_key"`
		Name      string `json:"name"`
		Category  string `json:"category"`
		Content   string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	// If preset_key provided, fill from library
	if body.PresetKey != "" {
		for _, p := range SnippetPresets {
			if p.Key == body.PresetKey {
				if body.Name == "" {
					body.Name = p.Name
				}
				if body.Category == "" {
					body.Category = p.Category
				}
				if body.Content == "" {
					body.Content = p.Content
				}
				break
			}
		}
	}

	if body.Content == "" {
		response.BadRequest(w, "content is required")
		return
	}

	var s NginxSnippet
	h.svc.repo.pool.QueryRow(r.Context(),
		`INSERT INTO nginx_snippets (website_id, preset_key, name, category, content) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, website_id, preset_key, name, category, content, enabled, created_at`,
		id, body.PresetKey, body.Name, body.Category, body.Content,
	).Scan(&s.ID, &s.WebsiteID, &s.PresetKey, &s.Name, &s.Category, &s.Content, &s.Enabled, &s.CreatedAt)

	// Regenerate nginx config
	h.svc.regenerateConfig(r.Context(), id)

	response.JSON(w, http.StatusCreated, s)
}

func (h *Handler) DeleteSnippet(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	snippetID, _ := strconv.ParseInt(chi.URLParam(r, "snippetId"), 10, 64)

	h.svc.repo.pool.Exec(r.Context(), "DELETE FROM nginx_snippets WHERE id = $1 AND website_id = $2", snippetID, id)
	h.svc.regenerateConfig(r.Context(), id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "snippet removed"})
}

func (h *Handler) ToggleSnippet(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	snippetID, _ := strconv.ParseInt(chi.URLParam(r, "snippetId"), 10, 64)

	h.svc.repo.pool.Exec(r.Context(), "UPDATE nginx_snippets SET enabled = NOT enabled WHERE id = $1 AND website_id = $2", snippetID, id)
	h.svc.regenerateConfig(r.Context(), id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "toggled"})
}

// ── Clone Handler ──────────────────────────────────────────

func (h *Handler) CloneWebsite(w http.ResponseWriter, r *http.Request) {
	if !h.requirePro(w, r) {
		return
	}
	sourceID, _ := parseID(r, "id")

	var body struct {
		Domain string `json:"domain"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Domain == "" {
		response.BadRequest(w, "domain is required for the clone")
		return
	}

	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	// Get source website
	source, err := h.svc.Get(r.Context(), sourceID)
	if err != nil {
		response.NotFound(w, "source website not found")
		return
	}

	// Create clone via existing Create flow
	cloneReq := CreateRequest{
		Domain:      body.Domain,
		Type:        source.Type,
		ProxyTarget: source.ProxyTarget,
		PHPVersion:  source.PHPVersion,
		IPV6:        source.IPV6,
		Remark:      fmt.Sprintf("Cloned from %s", source.Domain),
	}

	clone, err := h.svc.Create(r.Context(), c.UserID, cloneReq)
	if err != nil {
		handleWebsiteError(w, err)
		return
	}

	// Copy files from source to clone
	sourcePaths := nginx.GetSitePaths(source.Alias)
	clonePaths := nginx.GetSitePaths(clone.Alias)
	exec.Command("cp", "-a", sourcePaths.Index+"/.", clonePaths.Index+"/").Run()

	// Copy snippets
	h.svc.repo.pool.Exec(r.Context(),
		`INSERT INTO nginx_snippets (website_id, preset_key, name, category, content, enabled)
		 SELECT $1, preset_key, name, category, content, enabled FROM nginx_snippets WHERE website_id = $2`,
		clone.ID, sourceID)

	// Copy advanced settings
	h.svc.repo.pool.Exec(r.Context(),
		`UPDATE websites SET
			hsts_enabled = src.hsts_enabled, hsts_max_age = src.hsts_max_age, hsts_subdomains = src.hsts_subdomains,
			http3_enabled = src.http3_enabled,
			basic_auth_enabled = src.basic_auth_enabled, basic_auth_user = src.basic_auth_user, basic_auth_pass = src.basic_auth_pass,
			anti_hotlink = src.anti_hotlink, hotlink_extensions = src.hotlink_extensions, hotlink_allowed_domains = src.hotlink_allowed_domains,
			proxy_cache = src.proxy_cache, proxy_cache_time = src.proxy_cache_time, proxy_websocket = src.proxy_websocket, proxy_headers = src.proxy_headers,
			rate_limit_enabled = src.rate_limit_enabled, rate_limit_rps = src.rate_limit_rps, rate_limit_burst = src.rate_limit_burst, rate_limit_nodelay = src.rate_limit_nodelay,
			nginx_custom = src.nginx_custom
		 FROM websites src WHERE websites.id = $1 AND src.id = $2`,
		clone.ID, sourceID)

	// Copy redirects
	h.svc.repo.pool.Exec(r.Context(),
		`INSERT INTO website_redirects (website_id, type, source, target, enabled)
		 SELECT $1, type, source, target, enabled FROM website_redirects WHERE website_id = $2`,
		clone.ID, sourceID)

	// Regenerate config with all copied settings
	h.svc.regenerateConfig(r.Context(), clone.ID)

	response.JSON(w, http.StatusCreated, clone)
}

// ── Rate Limit Handlers ──────────────────────────────────────

type RateLimitConfig struct {
	Enabled bool `json:"enabled"`
	RPS     int  `json:"rps"`
	Burst   int  `json:"burst"`
	Nodelay bool `json:"nodelay"`
}

func (h *Handler) GetRateLimit(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var cfg RateLimitConfig
	h.svc.repo.pool.QueryRow(r.Context(),
		"SELECT rate_limit_enabled, rate_limit_rps, rate_limit_burst, rate_limit_nodelay FROM websites WHERE id = $1", id,
	).Scan(&cfg.Enabled, &cfg.RPS, &cfg.Burst, &cfg.Nodelay)
	response.JSON(w, http.StatusOK, cfg)
}

func (h *Handler) UpdateRateLimit(w http.ResponseWriter, r *http.Request) {
	if !h.requirePro(w, r) {
		return
	}
	id, _ := parseID(r, "id")
	var cfg RateLimitConfig
	json.NewDecoder(r.Body).Decode(&cfg)

	h.svc.repo.pool.Exec(r.Context(),
		"UPDATE websites SET rate_limit_enabled = $1, rate_limit_rps = $2, rate_limit_burst = $3, rate_limit_nodelay = $4, updated_at = NOW() WHERE id = $5",
		cfg.Enabled, cfg.RPS, cfg.Burst, cfg.Nodelay, id)

	h.svc.regenerateConfig(r.Context(), id)
	response.JSON(w, http.StatusOK, cfg)
}

// ── IP Access Control Handlers ──────────────────────────────

type IPRule struct {
	ID          int64  `json:"id"`
	WebsiteID   int64  `json:"website_id"`
	Mode        string `json:"mode"` // allow or deny
	IPAddress   string `json:"ip_address"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

func (h *Handler) ListIPRules(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	rows, err := h.svc.repo.pool.Query(r.Context(),
		"SELECT id, website_id, mode, ip_address, description, created_at FROM website_ip_rules WHERE website_id = $1 ORDER BY id", id)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()
	var rules []IPRule
	for rows.Next() {
		var rule IPRule
		rows.Scan(&rule.ID, &rule.WebsiteID, &rule.Mode, &rule.IPAddress, &rule.Description, &rule.CreatedAt)
		rules = append(rules, rule)
	}

	var mode string
	h.svc.repo.pool.QueryRow(r.Context(), "SELECT ip_access_mode FROM websites WHERE id = $1", id).Scan(&mode)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"mode":  mode,
		"rules": rules,
	})
}

func (h *Handler) AddIPRule(w http.ResponseWriter, r *http.Request) {
	if !h.requirePro(w, r) {
		return
	}
	id, _ := parseID(r, "id")
	var body struct {
		Mode        string `json:"mode"`
		IPAddress   string `json:"ip_address"`
		Description string `json:"description"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.IPAddress == "" {
		response.BadRequest(w, "ip_address is required")
		return
	}
	if body.Mode == "" {
		body.Mode = "deny"
	}

	var rule IPRule
	h.svc.repo.pool.QueryRow(r.Context(),
		`INSERT INTO website_ip_rules (website_id, mode, ip_address, description) VALUES ($1, $2, $3, $4)
		 RETURNING id, website_id, mode, ip_address, description, created_at`,
		id, body.Mode, body.IPAddress, body.Description,
	).Scan(&rule.ID, &rule.WebsiteID, &rule.Mode, &rule.IPAddress, &rule.Description, &rule.CreatedAt)

	h.svc.regenerateConfig(r.Context(), id)
	response.JSON(w, http.StatusCreated, rule)
}

func (h *Handler) DeleteIPRule(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	ruleID, _ := strconv.ParseInt(chi.URLParam(r, "ruleId"), 10, 64)
	h.svc.repo.pool.Exec(r.Context(), "DELETE FROM website_ip_rules WHERE id = $1 AND website_id = $2", ruleID, id)
	h.svc.regenerateConfig(r.Context(), id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "rule removed"})
}

func (h *Handler) SetIPAccessMode(w http.ResponseWriter, r *http.Request) {
	if !h.requirePro(w, r) {
		return
	}
	id, _ := parseID(r, "id")
	var body struct {
		Mode string `json:"mode"` // off, whitelist, blacklist
	}
	json.NewDecoder(r.Body).Decode(&body)
	h.svc.repo.pool.Exec(r.Context(), "UPDATE websites SET ip_access_mode = $1, updated_at = NOW() WHERE id = $2", body.Mode, id)
	h.svc.regenerateConfig(r.Context(), id)
	response.JSON(w, http.StatusOK, map[string]string{"mode": body.Mode})
}

// ── Helpers ──────────────────────────────────────────────────

func (h *Handler) requirePro(w http.ResponseWriter, r *http.Request) bool {
	tier := license.GetCurrentTier(r.Context(), h.svc.repo.pool)
	if tier == license.TierFree {
		response.JSON(w, http.StatusForbidden, map[string]interface{}{
			"error": map[string]interface{}{
				"message": "This feature requires a Pro license.",
				"code":    "LICENSE_REQUIRED",
				"tier":    tier,
			},
		})
		return false
	}
	return true
}

// regenerateConfig rebuilds nginx config including snippets + rate limiting
func (s *Service) regenerateConfig(ctx context.Context, websiteID int64) {
	site, err := s.Get(ctx, websiteID)
	if err != nil {
		return
	}

	// Build base config using existing method
	baseConfig, err := s.buildNginxConfig(ctx, site)
	if err != nil {
		return
	}

	// Collect enabled snippets
	rows, _ := s.repo.pool.Query(ctx,
		"SELECT content FROM nginx_snippets WHERE website_id = $1 AND enabled = true ORDER BY id", websiteID)
	if rows != nil {
		defer rows.Close()
		var snippetParts []string
		for rows.Next() {
			var content string
			rows.Scan(&content)
			snippetParts = append(snippetParts, content)
		}
		if len(snippetParts) > 0 {
			snippetBlock := "\n    # ── Snippet Presets ──\n"
			for _, sp := range snippetParts {
				for _, line := range strings.Split(sp, "\n") {
					if strings.TrimSpace(line) != "" {
						snippetBlock += "    " + line + "\n"
					}
				}
				snippetBlock += "\n"
			}
			// Insert before closing }
			baseConfig = strings.TrimSuffix(strings.TrimSpace(baseConfig), "}")
			baseConfig += snippetBlock + "}\n"
		}
	}

	// Add rate limiting directives
	var rlEnabled bool
	var rlRPS, rlBurst int
	var rlNodelay bool
	s.repo.pool.QueryRow(ctx,
		"SELECT rate_limit_enabled, rate_limit_rps, rate_limit_burst, rate_limit_nodelay FROM websites WHERE id = $1", websiteID,
	).Scan(&rlEnabled, &rlRPS, &rlBurst, &rlNodelay)

	// Add IP access rules
	var ipMode string
	s.repo.pool.QueryRow(ctx, "SELECT ip_access_mode FROM websites WHERE id = $1", websiteID).Scan(&ipMode)
	if ipMode == "whitelist" || ipMode == "blacklist" {
		ipRows, _ := s.repo.pool.Query(ctx,
			"SELECT mode, ip_address FROM website_ip_rules WHERE website_id = $1 ORDER BY id", websiteID)
		if ipRows != nil {
			defer ipRows.Close()
			var ipBlock string
			if ipMode == "whitelist" {
				ipBlock = "\n    # IP Whitelist (allow only listed IPs)\n"
				for ipRows.Next() {
					var mode, ip string
					ipRows.Scan(&mode, &ip)
					ipBlock += fmt.Sprintf("    allow %s;\n", ip)
				}
				ipBlock += "    deny all;\n"
			} else {
				ipBlock = "\n    # IP Blacklist (block listed IPs)\n"
				for ipRows.Next() {
					var mode, ip string
					ipRows.Scan(&mode, &ip)
					ipBlock += fmt.Sprintf("    deny %s;\n", ip)
				}
			}
			baseConfig = strings.TrimSuffix(strings.TrimSpace(baseConfig), "}")
			baseConfig += ipBlock + "}\n"
		}
	}

	if rlEnabled && rlRPS > 0 {
		zoneName := fmt.Sprintf("rl_%s", site.Alias)
		zoneDirective := fmt.Sprintf("limit_req_zone $binary_remote_addr zone=%s:10m rate=%dr/s;\n\n", zoneName, rlRPS)

		nodelayStr := ""
		if rlNodelay {
			nodelayStr = " nodelay"
		}
		limitDirective := fmt.Sprintf("    limit_req zone=%s burst=%d%s;\n    limit_req_status 429;\n", zoneName, rlBurst, nodelayStr)

		// Prepend zone directive before server block
		baseConfig = zoneDirective + baseConfig

		// Insert limit_req after first location { line
		baseConfig = strings.Replace(baseConfig, "    location / {\n", "    location / {\n"+limitDirective, 1)
	}

	paths := nginx.GetSitePaths(site.Alias)
	os.WriteFile(paths.ConfFile, []byte(baseConfig), 0644)
	nginx.TestAndReload()
}
