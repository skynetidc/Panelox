package appstore

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/license"
	"github.com/jakub/panelox/internal/pkg/response"
)

type AutoUpdateConfig struct {
	Enabled  bool   `json:"enabled"`
	Schedule string `json:"schedule"` // daily, weekly
}

func RegisterAutoUpdateRoutes(r chi.Router, h *Handler) {
	r.Get("/apps/auto-update", h.GetAutoUpdateConfig)
	r.Put("/apps/auto-update", h.SetAutoUpdateConfig)
	r.Post("/apps/auto-update/run", h.RunAutoUpdateNow)
}

func (h *Handler) GetAutoUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var val string
	h.pool.QueryRow(r.Context(), "SELECT value FROM settings WHERE key = 'app_auto_update'").Scan(&val)
	var cfg AutoUpdateConfig
	if val != "" {
		json.Unmarshal([]byte(val), &cfg)
	}
	response.JSON(w, http.StatusOK, cfg)
}

func (h *Handler) SetAutoUpdateConfig(w http.ResponseWriter, r *http.Request) {
	tier := license.GetCurrentTier(r.Context(), h.pool)
	if tier == license.TierFree {
		response.JSON(w, http.StatusForbidden, map[string]interface{}{
			"error": map[string]interface{}{"message": "Auto-update requires a Pro license.", "code": "LICENSE_REQUIRED"},
		})
		return
	}

	var cfg AutoUpdateConfig
	json.NewDecoder(r.Body).Decode(&cfg)
	data, _ := json.Marshal(cfg)

	h.pool.Exec(r.Context(),
		"INSERT INTO settings (key, value) VALUES ('app_auto_update', $1) ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = NOW()",
		string(data))

	response.JSON(w, http.StatusOK, cfg)
}

func (h *Handler) RunAutoUpdateNow(w http.ResponseWriter, r *http.Request) {
	tier := license.GetCurrentTier(r.Context(), h.pool)
	if tier == license.TierFree {
		response.JSON(w, http.StatusForbidden, map[string]interface{}{
			"error": map[string]interface{}{"message": "Auto-update requires a Pro license.", "code": "LICENSE_REQUIRED"},
		})
		return
	}

	results := h.updateAllApps(r.Context())
	response.JSON(w, http.StatusOK, results)
}

// AutoUpdateRunner runs periodic Docker image updates
type AutoUpdateRunner struct {
	h    *Handler
	stop chan struct{}
}

func NewAutoUpdateRunner(h *Handler) *AutoUpdateRunner {
	return &AutoUpdateRunner{h: h, stop: make(chan struct{})}
}

func (r *AutoUpdateRunner) Start() {
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				r.check()
			case <-r.stop:
				return
			}
		}
	}()
	log.Println("App auto-update runner started (checks every 6h)")
}

func (r *AutoUpdateRunner) check() {
	ctx := context.Background()

	// Check if PRO
	tier := license.GetCurrentTier(ctx, r.h.pool)
	if tier == license.TierFree {
		return
	}

	// Check if enabled
	var val string
	r.h.pool.QueryRow(ctx, "SELECT value FROM settings WHERE key = 'app_auto_update'").Scan(&val)
	var cfg AutoUpdateConfig
	if val != "" {
		json.Unmarshal([]byte(val), &cfg)
	}
	if !cfg.Enabled {
		return
	}

	log.Println("Running auto-update for all apps...")
	results := r.h.updateAllApps(ctx)
	for name, status := range results {
		log.Printf("Auto-update %s: %s", name, status)
	}
}

func (h *Handler) updateAllApps(ctx context.Context) map[string]string {
	results := map[string]string{}

	rows, err := h.pool.Query(ctx, "SELECT id, name, app_key FROM app_installs WHERE status = 'running'")
	if err != nil {
		return results
	}
	defer rows.Close()

	type app struct {
		id   int64
		name string
		key  string
	}
	var apps []app
	for rows.Next() {
		var a app
		rows.Scan(&a.id, &a.name, &a.key)
		apps = append(apps, a)
	}

	for _, a := range apps {
		composeFile := filepath.Join(appsDir, a.name, "docker-compose.yml")

		// Pull latest images
		out, err := exec.Command("docker", "compose", "-f", composeFile, "pull").CombinedOutput()
		if err != nil {
			results[a.name] = "pull failed: " + string(out)
			continue
		}

		// Check if images changed (docker compose will recreate only if needed)
		out, err = exec.Command("docker", "compose", "-f", composeFile, "up", "-d").CombinedOutput()
		if err != nil {
			results[a.name] = "restart failed: " + string(out)
			continue
		}

		results[a.name] = "updated"
		h.pool.Exec(ctx, "UPDATE app_installs SET updated_at = NOW() WHERE id = $1", a.id)
	}

	return results
}
