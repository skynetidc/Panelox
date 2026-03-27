package billing

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
)

type UsageReport struct {
	UserID    int64   `json:"user_id"`
	UserName  string  `json:"user_name"`
	Email     string  `json:"email"`
	Period    string  `json:"period"`
	DiskBytes int64   `json:"disk_bytes"`
	Bandwidth int64   `json:"bandwidth"`
	CPUMin    float64 `json:"cpu_minutes"`
	Websites  int     `json:"websites"`
	Databases int     `json:"databases"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/billing", func(r chi.Router) {
		r.Get("/usage", h.CurrentUsage)
		r.Get("/history", h.UsageHistory)
		r.Post("/collect", h.CollectUsage)
	})
}

func (h *Handler) CurrentUsage(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT u.id, u.name, u.email,
			COALESCE((SELECT COUNT(*) FROM websites WHERE user_id = u.id), 0),
			COALESCE((SELECT COUNT(*) FROM databases d JOIN db_servers s ON d.server_id = s.id WHERE s.user_id = u.id), 0)
		 FROM users u ORDER BY u.id`)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var reports []UsageReport
	for rows.Next() {
		var r UsageReport
		rows.Scan(&r.UserID, &r.UserName, &r.Email, &r.Websites, &r.Databases)
		r.Period = time.Now().Format("2006-01")

		// Calculate disk usage per user (from their websites)
		r.DiskBytes = getUserDiskUsage(h.pool, r.UserID)

		reports = append(reports, r)
	}
	response.JSON(w, http.StatusOK, reports)
}

func (h *Handler) UsageHistory(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)

	query := `SELECT user_id, '', '', period, disk_bytes, bandwidth, cpu_minutes, websites, databases
		 FROM resource_usage`
	args := []any{}
	if userID > 0 {
		query += " WHERE user_id = $1"
		args = append(args, userID)
	}
	query += " ORDER BY period DESC LIMIT 12"

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var reports []UsageReport
	for rows.Next() {
		var rr UsageReport
		rows.Scan(&rr.UserID, &rr.UserName, &rr.Email, &rr.Period, &rr.DiskBytes, &rr.Bandwidth, &rr.CPUMin, &rr.Websites, &rr.Databases)
		reports = append(reports, rr)
	}
	response.JSON(w, http.StatusOK, reports)
}

func (h *Handler) CollectUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	period := time.Now().Format("2006-01")

	rows, err := h.pool.Query(ctx, "SELECT id FROM users")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var userID int64
		rows.Scan(&userID)

		disk := getUserDiskUsage(h.pool, userID)
		var websites, databases int
		h.pool.QueryRow(ctx, "SELECT COUNT(*) FROM websites WHERE user_id = $1", userID).Scan(&websites)

		h.pool.Exec(ctx,
			`INSERT INTO resource_usage (user_id, period, disk_bytes, websites, databases)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (user_id, period) DO UPDATE SET disk_bytes = $3, websites = $4, databases = $5`,
			userID, period, disk, websites, databases)
		count++
	}

	response.JSON(w, http.StatusOK, map[string]any{"message": "usage collected", "users": count})
}

func getUserDiskUsage(pool *pgxpool.Pool, userID int64) int64 {
	rows, _ := pool.Query(context.Background(), "SELECT alias FROM websites WHERE user_id = $1", userID)
	if rows == nil {
		return 0
	}
	defer rows.Close()

	var total int64
	for rows.Next() {
		var alias string
		rows.Scan(&alias)
		dir := fmt.Sprintf("/www/sites/%s", alias)
		out, err := exec.Command("du", "-sb", dir).Output()
		if err == nil {
			parts := strings.Fields(string(out))
			if len(parts) >= 1 {
				size, _ := strconv.ParseInt(parts[0], 10, 64)
				total += size
			}
		}
	}
	return total
}
