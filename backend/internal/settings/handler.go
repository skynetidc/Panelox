package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
)

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/settings", func(r chi.Router) {
		r.Get("/", h.GetAll)
		r.Put("/", h.Update)
	})

	r.Route("/audit", func(r chi.Router) {
		r.Get("/", h.ListAuditLog)
	})

	r.Route("/update", func(r chi.Router) {
		r.Get("/check", h.CheckUpdate)
		r.Post("/apply", h.ApplyUpdate)
	})

	r.Get("/branding", h.GetBranding)
	r.Put("/branding", h.UpdateBranding)
}

// Settings key-value store

type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), "SELECT key, value FROM settings ORDER BY key")
	if err != nil {
		response.InternalError(w, "failed to load settings")
		return
	}
	defer rows.Close()

	settings := map[string]string{}
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		settings[k] = v
	}

	response.JSON(w, http.StatusOK, settings)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	for key, value := range body {
		h.pool.Exec(r.Context(),
			`INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, NOW())
			 ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`,
			key, value)
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "settings updated"})
}

// Get a single setting with default
func GetSetting(ctx context.Context, pool *pgxpool.Pool, key, defaultValue string) string {
	var value string
	err := pool.QueryRow(ctx, "SELECT value FROM settings WHERE key = $1", key).Scan(&value)
	if err == pgx.ErrNoRows {
		return defaultValue
	}
	if err != nil {
		return defaultValue
	}
	return value
}

// Audit log

type AuditEntry struct {
	ID        int64  `json:"id"`
	UserID    *int64 `json:"user_id"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Details   string `json:"details,omitempty"`
	IPAddress string `json:"ip_address"`
	CreatedAt string `json:"created_at"`
}

func (h *Handler) ListAuditLog(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 { page = 1 }
	if size < 1 || size > 100 { size = 50 }
	offset := (page - 1) * size

	var total int64
	h.pool.QueryRow(r.Context(), "SELECT COUNT(*) FROM audit_log").Scan(&total)

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, user_id, action, resource, COALESCE(details::text, ''), ip_address, created_at
		 FROM audit_log ORDER BY id DESC LIMIT $1 OFFSET $2`, size, offset)
	if err != nil {
		response.InternalError(w, "failed to load audit log")
		return
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		rows.Scan(&e.ID, &e.UserID, &e.Action, &e.Resource, &e.Details, &e.IPAddress, &e.CreatedAt)
		entries = append(entries, e)
	}

	response.JSONList(w, entries, page, size, total)
}

// Auto-update

const currentVersion = "1.0.0"

func (h *Handler) CheckUpdate(w http.ResponseWriter, r *http.Request) {
	// In production, this would check a remote API for the latest version
	response.JSON(w, http.StatusOK, map[string]any{
		"current_version": currentVersion,
		"latest_version":  currentVersion,
		"update_available": false,
		"changelog":       "No updates available",
	})
}

func (h *Handler) ApplyUpdate(w http.ResponseWriter, r *http.Request) {
	// In production: download new binary, replace, restart service
	log.Println("Update requested - would download and apply update")

	// Example update flow:
	// 1. Download new binary to /tmp/panelox-new
	// 2. Stop service
	// 3. Replace binary
	// 4. Start service
	go func() {
		out, err := exec.Command("systemctl", "restart", "panelox").CombinedOutput()
		if err != nil {
			log.Printf("Update restart failed: %s", string(out))
		}
		fmt.Println("Panel restarted after update")
	}()

	response.JSON(w, http.StatusAccepted, map[string]string{"message": "update started, panel will restart"})
}
