package license

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
)

// KeyAPI URL - reads from env var or .env file
func getAPIURL() string {
	if url := os.Getenv("PANELOX_LICENSE_API"); url != "" {
		return url
	}
	// Fallback: read from .env file in working directory
	if data, err := os.ReadFile(".env"); err == nil {
		for _, line := range bytes.Split(data, []byte("\n")) {
			line = bytes.TrimSpace(line)
			if bytes.HasPrefix(line, []byte("PANELOX_LICENSE_API=")) {
				return string(bytes.TrimPrefix(line, []byte("PANELOX_LICENSE_API=")))
			}
		}
	}
	return "https://api.panelox.com/api/panelox"
}

const (
	TierFree = "free"
	TierPro  = "pro"
)

const (
	FreeMaxWebsites  = 3
	FreeMaxNodes     = 1
	FreeMaxDatabases = 5
	FreeMaxApps      = 3
)

type License struct {
	ID            int64      `json:"id"`
	LicenseKey    string     `json:"license_key"`
	Tier          string     `json:"tier"`
	CustomerName  string     `json:"customer_name"`
	CustomerEmail string     `json:"customer_email"`
	MaxWebsites   int        `json:"max_websites"`
	MaxNodes      int        `json:"max_nodes"`
	MaxDatabases  int        `json:"max_databases"`
	MaxApps       int        `json:"max_apps"`
	Features      json.RawMessage `json:"features"`
	Status        string     `json:"status"`
	ActivatedAt   time.Time  `json:"activated_at"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	LastVerified  *time.Time `json:"last_verified,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type LicenseStatus struct {
	License      *License `json:"license"`
	Tier         string   `json:"tier"`
	UsedWebsites  int     `json:"used_websites"`
	UsedNodes     int     `json:"used_nodes"`
	UsedDatabases int     `json:"used_databases"`
	UsedApps      int     `json:"used_apps"`
	MaxWebsites   int     `json:"max_websites"`
	MaxNodes      int     `json:"max_nodes"`
	MaxDatabases  int     `json:"max_databases"`
	MaxApps       int     `json:"max_apps"`
	Features     []string `json:"features"`
	IsExpired    bool     `json:"is_expired"`
}

// VerifyResponse is what KeyAPI returns
type VerifyResponse struct {
	Valid         bool   `json:"valid"`
	Error         string `json:"error,omitempty"`
	Tier          string `json:"tier"`
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
	MaxWebsites   int    `json:"max_websites"`
	MaxNodes      int    `json:"max_nodes"`
	MaxDatabases  int    `json:"max_databases"`
	MaxApps       int    `json:"max_apps"`
	ActivatedAt   string `json:"activated_at"`
}

type Handler struct {
	pool   *pgxpool.Pool
	client *http.Client
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{
		pool:   pool,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/license", h.GetStatus)
	r.Post("/license/activate", h.Activate)
	r.Delete("/license", h.Deactivate)
	r.Get("/license/check/{resource}", h.CheckLimit)
}

// ─────────────────────────────────────────────────────────────
// Handlers
// ─────────────────────────────────────────────────────────────

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.getLicenseStatus(r.Context())
	response.JSON(w, http.StatusOK, status)
}

func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LicenseKey string `json:"license_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.LicenseKey == "" {
		response.BadRequest(w, "license_key is required")
		return
	}

	// Verify against KeyAPI
	hostname, _ := os.Hostname()
	verifyResp, err := h.verifyWithAPI(req.LicenseKey, hostname)
	if err != nil {
		response.InternalError(w, "Failed to contact license server: "+err.Error())
		return
	}
	if !verifyResp.Valid {
		response.BadRequest(w, verifyResp.Error)
		return
	}

	// Remove any existing license
	h.pool.Exec(r.Context(), "DELETE FROM licenses")

	featuresJSON := json.RawMessage("[]")
	now := time.Now()

	var lic License
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO licenses (license_key, tier, customer_name, customer_email, max_websites, max_nodes, max_databases, max_apps, features, status, expires_at, last_verified)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active', NULL, $10)
		 RETURNING id, license_key, tier, customer_name, customer_email, max_websites, max_nodes, max_databases, max_apps, features, status, activated_at, expires_at, last_verified, created_at`,
		req.LicenseKey, verifyResp.Tier, verifyResp.CustomerName, verifyResp.CustomerEmail,
		verifyResp.MaxWebsites, verifyResp.MaxNodes, verifyResp.MaxDatabases, verifyResp.MaxApps,
		featuresJSON, now,
	).Scan(&lic.ID, &lic.LicenseKey, &lic.Tier, &lic.CustomerName, &lic.CustomerEmail,
		&lic.MaxWebsites, &lic.MaxNodes, &lic.MaxDatabases, &lic.MaxApps,
		&lic.Features, &lic.Status, &lic.ActivatedAt, &lic.ExpiresAt, &lic.LastVerified, &lic.CreatedAt)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	// Mask key for response
	masked := lic.LicenseKey
	if len(masked) > 12 {
		masked = masked[:6] + "..." + masked[len(masked)-6:]
	}
	lic.LicenseKey = masked

	response.JSON(w, http.StatusOK, lic)
}

func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	// Get the full key before deleting
	var key string
	h.pool.QueryRow(r.Context(), "SELECT license_key FROM licenses WHERE status = 'active' LIMIT 1").Scan(&key)

	// Notify KeyAPI to unbind
	if key != "" {
		h.deactivateWithAPI(key)
	}

	h.pool.Exec(r.Context(), "DELETE FROM licenses")
	response.JSON(w, http.StatusOK, map[string]string{"message": "License deactivated."})
}

func (h *Handler) CheckLimit(w http.ResponseWriter, r *http.Request) {
	resource := chi.URLParam(r, "resource")
	status := h.getLicenseStatus(r.Context())

	var allowed bool
	var current, max int

	switch resource {
	case "website", "websites":
		current, max = status.UsedWebsites, status.MaxWebsites
	case "node", "nodes":
		current, max = status.UsedNodes, status.MaxNodes
	case "database", "databases":
		current, max = status.UsedDatabases, status.MaxDatabases
	case "app", "apps":
		current, max = status.UsedApps, status.MaxApps
	default:
		response.BadRequest(w, "unknown resource: "+resource)
		return
	}

	if max == 0 {
		allowed = true
	} else {
		allowed = current < max
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"allowed":  allowed,
		"current":  current,
		"max":      max,
		"tier":     status.Tier,
		"resource": resource,
	})
}

// ─────────────────────────────────────────────────────────────
// KeyAPI communication
// ─────────────────────────────────────────────────────────────

func (h *Handler) verifyWithAPI(licenseKey, hostname string) (*VerifyResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"license_key": licenseKey,
		"hostname":    hostname,
	})

	resp, err := h.client.Post(getAPIURL()+"/verify", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var result VerifyResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid response from license server")
	}

	if resp.StatusCode != 200 {
		if result.Error == "" {
			result.Error = "License verification failed"
		}
		result.Valid = false
	}

	return &result, nil
}

func (h *Handler) deactivateWithAPI(licenseKey string) {
	body, _ := json.Marshal(map[string]string{"license_key": licenseKey})
	resp, err := h.client.Post(getAPIURL()+"/deactivate", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("Failed to deactivate license with API: %v", err)
		return
	}
	resp.Body.Close()
}

// ─────────────────────────────────────────────────────────────
// Periodic re-verification (call from background goroutine)
// ─────────────────────────────────────────────────────────────

func (h *Handler) StartPeriodicVerification() {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			h.reverify()
		}
	}()
	log.Println("License periodic verification started (every 24h)")
}

func (h *Handler) reverify() {
	ctx := context.Background()
	var key string
	err := h.pool.QueryRow(ctx, "SELECT license_key FROM licenses WHERE status = 'active' LIMIT 1").Scan(&key)
	if err != nil || key == "" {
		return
	}

	hostname, _ := os.Hostname()
	resp, err := h.verifyWithAPI(key, hostname)
	if err != nil {
		log.Printf("License re-verification failed (network): %v", err)
		return // Don't revoke on network errors - be graceful
	}

	if !resp.Valid {
		log.Printf("License re-verification failed: %s - revoking", resp.Error)
		h.pool.Exec(ctx, "UPDATE licenses SET status = 'revoked' WHERE license_key = $1", key)
		return
	}

	// Update last_verified + sync limits from server
	h.pool.Exec(ctx, `UPDATE licenses SET last_verified = NOW(), tier = $1, max_websites = $2, max_nodes = $3, max_databases = $4, max_apps = $5 WHERE license_key = $6`,
		resp.Tier, resp.MaxWebsites, resp.MaxNodes, resp.MaxDatabases, resp.MaxApps, key)
}

// ─────────────────────────────────────────────────────────────
// Internal
// ─────────────────────────────────────────────────────────────

func (h *Handler) getLicenseStatus(ctx context.Context) *LicenseStatus {
	status := &LicenseStatus{
		Tier:         TierFree,
		MaxWebsites:  FreeMaxWebsites,
		MaxNodes:     FreeMaxNodes,
		MaxDatabases: FreeMaxDatabases,
		MaxApps:      FreeMaxApps,
		Features:     []string{},
	}

	var lic License
	err := h.pool.QueryRow(ctx,
		`SELECT id, license_key, tier, customer_name, customer_email, max_websites, max_nodes, max_databases, max_apps, features, status, activated_at, expires_at, last_verified, created_at
		 FROM licenses WHERE status = 'active' ORDER BY id DESC LIMIT 1`,
	).Scan(&lic.ID, &lic.LicenseKey, &lic.Tier, &lic.CustomerName, &lic.CustomerEmail,
		&lic.MaxWebsites, &lic.MaxNodes, &lic.MaxDatabases, &lic.MaxApps,
		&lic.Features, &lic.Status, &lic.ActivatedAt, &lic.ExpiresAt, &lic.LastVerified, &lic.CreatedAt)

	if err == nil {
		masked := lic.LicenseKey
		if len(masked) > 12 {
			masked = masked[:6] + "..." + masked[len(masked)-6:]
		}
		lic.LicenseKey = masked
		status.License = &lic
		status.Tier = lic.Tier
		status.MaxWebsites = lic.MaxWebsites
		status.MaxNodes = lic.MaxNodes
		status.MaxDatabases = lic.MaxDatabases
		status.MaxApps = lic.MaxApps
		var features []string
		json.Unmarshal(lic.Features, &features)
		status.Features = features
	}

	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM websites").Scan(&status.UsedWebsites)
	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM nodes").Scan(&status.UsedNodes)
	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM databases").Scan(&status.UsedDatabases)
	h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_installs").Scan(&status.UsedApps)

	return status
}

// Helpers for other modules

func GetCurrentTier(ctx context.Context, pool *pgxpool.Pool) string {
	var tier string
	err := pool.QueryRow(ctx, "SELECT tier FROM licenses WHERE status = 'active' ORDER BY id DESC LIMIT 1").Scan(&tier)
	if err != nil {
		return TierFree
	}
	return tier
}

func CheckResourceLimit(ctx context.Context, pool *pgxpool.Pool, resource string) bool {
	var maxVal int
	err := pool.QueryRow(ctx,
		`SELECT CASE
			WHEN $1 = 'websites' THEN max_websites
			WHEN $1 = 'nodes' THEN max_nodes
			WHEN $1 = 'databases' THEN max_databases
			WHEN $1 = 'apps' THEN max_apps
			ELSE 0
		 END FROM licenses WHERE status = 'active' ORDER BY id DESC LIMIT 1`, resource).Scan(&maxVal)
	if err != nil {
		switch resource {
		case "websites":
			maxVal = FreeMaxWebsites
		case "nodes":
			maxVal = FreeMaxNodes
		case "databases":
			maxVal = FreeMaxDatabases
		case "apps":
			maxVal = FreeMaxApps
		}
	}
	if maxVal == 0 {
		return true
	}
	var currentVal int
	switch resource {
	case "websites":
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM websites").Scan(&currentVal)
	case "nodes":
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM nodes").Scan(&currentVal)
	case "databases":
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM databases").Scan(&currentVal)
	case "apps":
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_installs").Scan(&currentVal)
	}
	return currentVal < maxVal
}
