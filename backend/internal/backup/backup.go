package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/claims"
	"github.com/jakub/panelox/internal/pkg/response"
)

const defaultBackupDir = "/opt/panelox/backups"

type Backup struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // website, database, full
	SourceID  *int64    `json:"source_id,omitempty"`
	FilePath  string    `json:"file_path"`
	FileSize  int64     `json:"file_size"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateRequest struct {
	Type     string `json:"type"`
	SourceID *int64 `json:"source_id,omitempty"`
	Name     string `json:"name"`
}

type Handler struct {
	pool      *pgxpool.Pool
	getClaims func(r *http.Request) *claims.Claims
}

func NewHandler(pool *pgxpool.Pool, getClaims func(r *http.Request) *claims.Claims) *Handler {
	return &Handler{pool: pool, getClaims: getClaims}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/backups", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/restore", h.Restore)
		r.Get("/{id}/download", h.Download)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 { page = 1 }
	if size < 1 || size > 100 { size = 20 }
	offset := (page - 1) * size

	var total int64
	h.pool.QueryRow(r.Context(), "SELECT COUNT(*) FROM backups").Scan(&total)

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, user_id, name, type, source_id, file_path, file_size, status, message, created_at
		 FROM backups ORDER BY id DESC LIMIT $1 OFFSET $2`, size, offset)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var backups []Backup
	for rows.Next() {
		var b Backup
		rows.Scan(&b.ID, &b.UserID, &b.Name, &b.Type, &b.SourceID, &b.FilePath, &b.FileSize, &b.Status, &b.Message, &b.CreatedAt)
		backups = append(backups, b)
	}
	response.JSONList(w, backups, page, size, total)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Type == "" {
		response.BadRequest(w, "type is required (website, database, full)")
		return
	}

	timestamp := time.Now().Format("20060102-150405")
	name := req.Name
	if name == "" {
		name = fmt.Sprintf("backup-%s-%s", req.Type, timestamp)
	}
	fileName := fmt.Sprintf("%s.tar.gz", name)
	filePath := filepath.Join(defaultBackupDir, fileName)

	var b Backup
	err := h.pool.QueryRow(r.Context(),
		`INSERT INTO backups (user_id, name, type, source_id, file_path, status)
		 VALUES ($1, $2, $3, $4, $5, 'running') RETURNING id, user_id, name, type, source_id, file_path, file_size, status, message, created_at`,
		c.UserID, name, req.Type, req.SourceID, filePath,
	).Scan(&b.ID, &b.UserID, &b.Name, &b.Type, &b.SourceID, &b.FilePath, &b.FileSize, &b.Status, &b.Message, &b.CreatedAt)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	go runBackup(h.pool, &b)
	response.JSON(w, http.StatusAccepted, b)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var filePath string
	h.pool.QueryRow(r.Context(), "SELECT file_path FROM backups WHERE id = $1", id).Scan(&filePath)
	if filePath != "" {
		os.Remove(filePath)
	}

	h.pool.Exec(r.Context(), "DELETE FROM backups WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "backup deleted"})
}

func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var b Backup
	err := h.pool.QueryRow(r.Context(),
		"SELECT id, type, source_id, file_path, status FROM backups WHERE id = $1", id,
	).Scan(&b.ID, &b.Type, &b.SourceID, &b.FilePath, &b.Status)
	if err != nil {
		response.NotFound(w, "backup not found")
		return
	}

	if b.Status != "completed" {
		response.BadRequest(w, "only completed backups can be restored")
		return
	}

	go runRestore(h.pool, &b)
	response.JSON(w, http.StatusAccepted, map[string]string{"message": "restore started"})
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var filePath, name string
	h.pool.QueryRow(r.Context(), "SELECT file_path, name FROM backups WHERE id = $1", id).Scan(&filePath, &name)
	if filePath == "" {
		response.NotFound(w, "backup not found")
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.tar.gz", name))
	w.Header().Set("Content-Type", "application/gzip")
	http.ServeFile(w, r, filePath)
}

func runBackup(pool *pgxpool.Pool, b *Backup) {
	ctx := context.Background()
	os.MkdirAll(defaultBackupDir, 0755)

	var sourcePath string
	switch b.Type {
	case "website":
		if b.SourceID != nil {
			var alias string
			pool.QueryRow(ctx, "SELECT alias FROM websites WHERE id = $1", *b.SourceID).Scan(&alias)
			sourcePath = fmt.Sprintf("/www/sites/%s", alias)
		} else {
			sourcePath = "/www/sites"
		}
	case "database":
		// Dump all databases
		sourcePath = "/tmp/panelox-db-dump"
		os.MkdirAll(sourcePath, 0755)
		rows, _ := pool.Query(ctx, "SELECT container_id, type, root_password, name FROM db_servers WHERE status = 'running'")
		if rows != nil {
			for rows.Next() {
				var containerID, dbType, rootPass, srvName string
				rows.Scan(&containerID, &dbType, &rootPass, &srvName)
				dumpFile := filepath.Join(sourcePath, srvName+".sql")
				switch dbType {
				case "mysql":
					out, _ := exec.Command("docker", "exec", containerID, "mysqldump", "-uroot", "-p"+rootPass, "--all-databases").Output()
					os.WriteFile(dumpFile, out, 0644)
				case "postgresql":
					out, _ := exec.Command("docker", "exec", containerID, "pg_dumpall", "-U", "postgres").Output()
					os.WriteFile(dumpFile, out, 0644)
				}
			}
			rows.Close()
		}
	case "full":
		sourcePath = "/www"
	default:
		pool.Exec(ctx, "UPDATE backups SET status = 'failed', message = 'unknown type' WHERE id = $1", b.ID)
		return
	}

	// Create tar.gz
	out, err := exec.Command("tar", "-czf", b.FilePath, "-C", filepath.Dir(sourcePath), filepath.Base(sourcePath)).CombinedOutput()
	if err != nil {
		pool.Exec(ctx, "UPDATE backups SET status = 'failed', message = $1 WHERE id = $2", string(out), b.ID)
		return
	}

	// Get file size
	info, _ := os.Stat(b.FilePath)
	var fileSize int64
	if info != nil {
		fileSize = info.Size()
	}

	pool.Exec(ctx, "UPDATE backups SET status = 'completed', file_size = $1 WHERE id = $2", fileSize, b.ID)
	log.Printf("Backup %d completed: %s (%d bytes)", b.ID, b.FilePath, fileSize)

	// Cleanup temp db dump dir
	if b.Type == "database" {
		os.RemoveAll("/tmp/panelox-db-dump")
	}
}

func runRestore(pool *pgxpool.Pool, b *Backup) {
	ctx := context.Background()

	var destPath string
	switch b.Type {
	case "website":
		destPath = "/www/sites"
	case "full":
		destPath = "/"
	default:
		return
	}

	out, err := exec.Command("tar", "-xzf", b.FilePath, "-C", destPath).CombinedOutput()
	if err != nil {
		log.Printf("Restore %d failed: %s", b.ID, string(out))
		return
	}

	log.Printf("Restore %d completed from %s", b.ID, b.FilePath)
	_ = ctx
}

var ErrNotFound = errors.New("not found")
