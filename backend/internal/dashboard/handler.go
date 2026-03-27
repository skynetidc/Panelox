package dashboard

import (
	"net/http"
	"runtime"

	"github.com/go-chi/chi/v5"
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
	r.Get("/dashboard/stats", h.Stats)
	r.Get("/dashboard/overview", h.Overview)
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	response.JSON(w, http.StatusOK, map[string]any{
		"go_routines":  runtime.NumGoroutine(),
		"memory_alloc": m.Alloc,
		"memory_sys":   m.Sys,
		"num_cpu":      runtime.NumCPU(),
		"go_version":   runtime.Version(),
	})
}

func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	counts := map[string]int64{}

	tables := []struct {
		name  string
		table string
	}{
		{"users", "users"},
		{"websites", "websites"},
		{"ssl_certificates", "ssl_certificates"},
		{"dns_zones", "dns_zones"},
		{"databases", "databases"},
		{"mail_domains", "mail_domains"},
		{"firewall_rules", "firewall_rules"},
	}

	for _, t := range tables {
		var count int64
		err := h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+t.table).Scan(&count)
		if err != nil {
			count = 0
		}
		counts[t.name] = count
	}

	response.JSON(w, http.StatusOK, counts)
}
