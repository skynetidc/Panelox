package website

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/nginx"
	"github.com/jakub/panelox/internal/pkg/response"
)

// Advanced features handler methods

// RegisterAdvancedRoutes adds per-website feature routes
func RegisterAdvancedRoutes(r chi.Router, h *Handler) {
	// Logs
	r.Get("/websites/{id}/logs/access", h.AccessLog)
	r.Get("/websites/{id}/logs/error", h.ErrorLog)

	// Redirects
	r.Get("/websites/{id}/redirects", h.ListRedirects)
	r.Post("/websites/{id}/redirects", h.CreateRedirect)
	r.Delete("/websites/{id}/redirects/{redirectId}", h.DeleteRedirect)

	// Advanced config
	r.Put("/websites/{id}/security", h.UpdateSecurity)
	r.Put("/websites/{id}/proxy-config", h.UpdateProxyConfig)
}

// --- Log Viewer ---

func (h *Handler) AccessLog(w http.ResponseWriter, r *http.Request) {
	h.serveLog(w, r, "access")
}

func (h *Handler) ErrorLog(w http.ResponseWriter, r *http.Request) {
	h.serveLog(w, r, "error")
}

func (h *Handler) serveLog(w http.ResponseWriter, r *http.Request, logType string) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	site, err := h.svc.Get(r.Context(), id)
	if err != nil {
		handleWebsiteError(w, err)
		return
	}

	paths := nginx.GetSitePaths(site.Alias)
	var logPath string
	if logType == "access" {
		logPath = paths.AccessLog
	} else {
		logPath = paths.ErrorLog
	}

	lines, _ := strconv.Atoi(r.URL.Query().Get("lines"))
	if lines <= 0 || lines > 1000 {
		lines = 100
	}

	// Use tail to get last N lines
	out, err := exec.Command("tail", "-n", strconv.Itoa(lines), logPath).CombinedOutput()
	if err != nil {
		// Try reading file directly
		content, readErr := readLastLines(logPath, lines)
		if readErr != nil {
			response.JSON(w, http.StatusOK, map[string]string{"logs": "", "path": logPath})
			return
		}
		response.JSON(w, http.StatusOK, map[string]string{"logs": content, "path": logPath})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"logs": string(out), "path": logPath})
}

func readLastLines(path string, n int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	return strings.Join(lines, "\n"), nil
}

// --- Redirect Rules ---

type Redirect struct {
	ID        int64  `json:"id"`
	WebsiteID int64  `json:"website_id"`
	Type      string `json:"type"` // 301, 302
	Source    string `json:"source"`
	Target    string `json:"target"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

type CreateRedirectRequest struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func (h *Handler) ListRedirects(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	rows, err := h.svc.repo.pool.Query(r.Context(),
		"SELECT id, website_id, type, source, target, enabled, created_at FROM website_redirects WHERE website_id = $1 ORDER BY id", id)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var redirects []Redirect
	for rows.Next() {
		var rd Redirect
		rows.Scan(&rd.ID, &rd.WebsiteID, &rd.Type, &rd.Source, &rd.Target, &rd.Enabled, &rd.CreatedAt)
		redirects = append(redirects, rd)
	}
	response.JSON(w, http.StatusOK, redirects)
}

func (h *Handler) CreateRedirect(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var req CreateRedirectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Source == "" || req.Target == "" {
		response.BadRequest(w, "source and target are required")
		return
	}
	if req.Type == "" {
		req.Type = "301"
	}

	var rd Redirect
	err := h.svc.repo.pool.QueryRow(r.Context(),
		`INSERT INTO website_redirects (website_id, type, source, target) VALUES ($1, $2, $3, $4)
		 RETURNING id, website_id, type, source, target, enabled, created_at`,
		id, req.Type, req.Source, req.Target,
	).Scan(&rd.ID, &rd.WebsiteID, &rd.Type, &rd.Source, &rd.Target, &rd.Enabled, &rd.CreatedAt)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, rd)
}

func (h *Handler) DeleteRedirect(w http.ResponseWriter, r *http.Request) {
	redirectID, _ := strconv.ParseInt(chi.URLParam(r, "redirectId"), 10, 64)
	h.svc.repo.pool.Exec(r.Context(), "DELETE FROM website_redirects WHERE id = $1", redirectID)
	response.JSON(w, http.StatusOK, map[string]string{"message": "redirect deleted"})
}

// --- Security Settings (HSTS, Basic Auth, Anti-Hotlink) ---

type SecurityConfig struct {
	HSTSEnabled       *bool   `json:"hsts_enabled,omitempty"`
	HSTSMaxAge        *int    `json:"hsts_max_age,omitempty"`
	HSTSSubdomains    *bool   `json:"hsts_subdomains,omitempty"`
	HTTP3Enabled      *bool   `json:"http3_enabled,omitempty"`
	BasicAuthEnabled  *bool   `json:"basic_auth_enabled,omitempty"`
	BasicAuthUser     *string `json:"basic_auth_user,omitempty"`
	BasicAuthPass     *string `json:"basic_auth_pass,omitempty"`
	AntiHotlink       *bool   `json:"anti_hotlink,omitempty"`
	HotlinkExtensions *string `json:"hotlink_extensions,omitempty"`
	HotlinkAllowed    *string `json:"hotlink_allowed_domains,omitempty"`
}

func (h *Handler) UpdateSecurity(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var cfg SecurityConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	sets := []string{"updated_at = NOW()"}
	args := []any{}
	idx := 1

	addField := func(col string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, val)
		idx++
	}

	if cfg.HSTSEnabled != nil { addField("hsts_enabled", *cfg.HSTSEnabled) }
	if cfg.HSTSMaxAge != nil { addField("hsts_max_age", *cfg.HSTSMaxAge) }
	if cfg.HSTSSubdomains != nil { addField("hsts_subdomains", *cfg.HSTSSubdomains) }
	if cfg.HTTP3Enabled != nil { addField("http3_enabled", *cfg.HTTP3Enabled) }
	if cfg.BasicAuthEnabled != nil { addField("basic_auth_enabled", *cfg.BasicAuthEnabled) }
	if cfg.BasicAuthUser != nil { addField("basic_auth_user", *cfg.BasicAuthUser) }
	if cfg.BasicAuthPass != nil { addField("basic_auth_pass", *cfg.BasicAuthPass) }
	if cfg.AntiHotlink != nil { addField("anti_hotlink", *cfg.AntiHotlink) }
	if cfg.HotlinkExtensions != nil { addField("hotlink_extensions", *cfg.HotlinkExtensions) }
	if cfg.HotlinkAllowed != nil { addField("hotlink_allowed_domains", *cfg.HotlinkAllowed) }

	if len(args) == 0 {
		response.BadRequest(w, "no fields to update")
		return
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE websites SET %s WHERE id = $%d", strings.Join(sets, ", "), idx)
	_, err := h.svc.repo.pool.Exec(r.Context(), query, args...)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "security settings updated"})
}

// --- Proxy Config (cache, WebSocket, custom headers) ---

type ProxyConfig struct {
	Cache        *bool              `json:"proxy_cache,omitempty"`
	CacheTime    *string            `json:"proxy_cache_time,omitempty"`
	WebSocket    *bool              `json:"proxy_websocket,omitempty"`
	Headers      map[string]string  `json:"proxy_headers,omitempty"`
}

func (h *Handler) UpdateProxyConfig(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var cfg ProxyConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	sets := []string{"updated_at = NOW()"}
	args := []any{}
	idx := 1

	if cfg.Cache != nil {
		sets = append(sets, fmt.Sprintf("proxy_cache = $%d", idx)); args = append(args, *cfg.Cache); idx++
	}
	if cfg.CacheTime != nil {
		sets = append(sets, fmt.Sprintf("proxy_cache_time = $%d", idx)); args = append(args, *cfg.CacheTime); idx++
	}
	if cfg.WebSocket != nil {
		sets = append(sets, fmt.Sprintf("proxy_websocket = $%d", idx)); args = append(args, *cfg.WebSocket); idx++
	}
	if cfg.Headers != nil {
		headersJSON, _ := json.Marshal(cfg.Headers)
		sets = append(sets, fmt.Sprintf("proxy_headers = $%d", idx)); args = append(args, string(headersJSON)); idx++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE websites SET %s WHERE id = $%d", strings.Join(sets, ", "), idx)
	if _, err := h.svc.repo.pool.Exec(r.Context(), query, args...); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(w, "website not found")
			return
		}
		response.InternalError(w, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "proxy config updated"})
}
