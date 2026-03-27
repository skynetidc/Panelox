package loginlog

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
)

type Entry struct {
	ID        int64  `json:"id"`
	UserID    *int64 `json:"user_id,omitempty"`
	Email     string `json:"email"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	CreatedAt string `json:"created_at"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/login-log", h.List)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 { page = 1 }
	if size < 1 || size > 100 { size = 50 }
	offset := (page - 1) * size

	var total int64
	h.pool.QueryRow(r.Context(), "SELECT COUNT(*) FROM login_log").Scan(&total)

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, user_id, email, ip_address, user_agent, success, message, created_at
		 FROM login_log ORDER BY id DESC LIMIT $1 OFFSET $2`, size, offset)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		rows.Scan(&e.ID, &e.UserID, &e.Email, &e.IPAddress, &e.UserAgent, &e.Success, &e.Message, &e.CreatedAt)
		entries = append(entries, e)
	}
	response.JSONList(w, entries, page, size, total)
}

// Record logs a login attempt. Called from auth handler.
func Record(ctx context.Context, pool *pgxpool.Pool, userID *int64, email, ip, ua string, success bool, msg string) {
	pool.Exec(ctx,
		`INSERT INTO login_log (user_id, email, ip_address, user_agent, success, message)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, email, ip, ua, success, msg)
}
