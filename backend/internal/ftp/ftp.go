package ftp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
	"golang.org/x/crypto/bcrypt"
)

type FTPUser struct {
	ID        int64  `json:"id"`
	WebsiteID *int64 `json:"website_id,omitempty"`
	Username  string `json:"username"`
	HomeDir   string `json:"home_dir"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type CreateRequest struct {
	WebsiteID *int64 `json:"website_id,omitempty"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	HomeDir   string `json:"home_dir"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/ftp", func(r chi.Router) {
		r.Get("/users", h.List)
		r.Post("/users", h.Create)
		r.Delete("/users/{id}", h.Delete)
		r.Post("/users/{id}/toggle", h.Toggle)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, website_id, username, home_dir, status, created_at FROM ftp_users ORDER BY id")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var users []FTPUser
	for rows.Next() {
		var u FTPUser
		rows.Scan(&u.ID, &u.WebsiteID, &u.Username, &u.HomeDir, &u.Status, &u.CreatedAt)
		users = append(users, u)
	}
	response.JSON(w, http.StatusOK, users)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		response.BadRequest(w, "username and password are required")
		return
	}
	if req.HomeDir == "" {
		req.HomeDir = fmt.Sprintf("/www/sites/%s", req.Username)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		response.InternalError(w, "hash error")
		return
	}

	var u FTPUser
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO ftp_users (website_id, username, password, home_dir) VALUES ($1, $2, $3, $4)
		 RETURNING id, website_id, username, home_dir, status, created_at`,
		req.WebsiteID, req.Username, string(hash), req.HomeDir,
	).Scan(&u.ID, &u.WebsiteID, &u.Username, &u.HomeDir, &u.Status, &u.CreatedAt)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	// Create system FTP user (best effort - requires vsftpd/proftpd)
	exec.Command("useradd", "-d", req.HomeDir, "-s", "/sbin/nologin", req.Username).Run()
	exec.Command("bash", "-c", fmt.Sprintf("echo '%s:%s' | chpasswd", req.Username, req.Password)).Run()

	response.JSON(w, http.StatusCreated, u)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var username string
	err := h.pool.QueryRow(r.Context(), "SELECT username FROM ftp_users WHERE id = $1", id).Scan(&username)
	if err == pgx.ErrNoRows {
		response.NotFound(w, "user not found")
		return
	}

	h.pool.Exec(r.Context(), "DELETE FROM ftp_users WHERE id = $1", id)
	exec.Command("userdel", username).Run()

	response.JSON(w, http.StatusOK, map[string]string{"message": "FTP user deleted"})
}

func (h *Handler) Toggle(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(),
		"UPDATE ftp_users SET status = CASE WHEN status = 'active' THEN 'disabled' ELSE 'active' END WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "toggled"})
}
