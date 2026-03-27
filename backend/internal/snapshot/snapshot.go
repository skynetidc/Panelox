package snapshot

import (
	"context"
	"encoding/json"
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
	"github.com/jakub/panelox/internal/pkg/response"
)

const snapshotDir = "/opt/panelox/snapshots"

type Snapshot struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	FilePath    string   `json:"file_path"`
	FileSize    int64    `json:"file_size"`
	Includes    []string `json:"includes"`
	Status      string   `json:"status"`
	Message     string   `json:"message,omitempty"`
	CreatedAt   string   `json:"created_at"`
}

type CreateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Includes    []string `json:"includes"` // websites, databases, nginx, panel
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/snapshots", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/rollback", h.Rollback)
		r.Get("/{id}/download", h.Download)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, name, description, file_path, file_size, includes, status, message, created_at::text FROM snapshots ORDER BY id DESC")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var snapshots []Snapshot
	for rows.Next() {
		var s Snapshot
		rows.Scan(&s.ID, &s.Name, &s.Description, &s.FilePath, &s.FileSize, &s.Includes, &s.Status, &s.Message, &s.CreatedAt)
		snapshots = append(snapshots, s)
	}
	response.JSON(w, http.StatusOK, snapshots)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.Name == "" {
		req.Name = fmt.Sprintf("snapshot-%s", time.Now().Format("20060102-150405"))
	}
	if len(req.Includes) == 0 {
		req.Includes = []string{"websites", "databases", "nginx", "panel"}
	}

	filePath := filepath.Join(snapshotDir, req.Name+".tar.gz")

	var s Snapshot
	err := h.pool.QueryRow(r.Context(),
		`INSERT INTO snapshots (name, description, file_path, includes, status)
		 VALUES ($1, $2, $3, $4, 'creating')
		 RETURNING id, name, description, file_path, file_size, includes, status, message, created_at::text`,
		req.Name, req.Description, filePath, req.Includes,
	).Scan(&s.ID, &s.Name, &s.Description, &s.FilePath, &s.FileSize, &s.Includes, &s.Status, &s.Message, &s.CreatedAt)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	go createSnapshot(h.pool, s.ID, filePath, req.Includes)
	response.JSON(w, http.StatusAccepted, s)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var filePath string
	h.pool.QueryRow(r.Context(), "SELECT file_path FROM snapshots WHERE id = $1", id).Scan(&filePath)
	if filePath != "" {
		os.Remove(filePath)
	}
	h.pool.Exec(r.Context(), "DELETE FROM snapshots WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "snapshot deleted"})
}

func (h *Handler) Rollback(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var filePath, status string
	h.pool.QueryRow(r.Context(), "SELECT file_path, status FROM snapshots WHERE id = $1", id).Scan(&filePath, &status)
	if status != "completed" {
		response.BadRequest(w, "snapshot not ready")
		return
	}

	go rollbackSnapshot(h.pool, id, filePath)
	response.JSON(w, http.StatusAccepted, map[string]string{"message": "rollback started"})
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var filePath, name string
	h.pool.QueryRow(r.Context(), "SELECT file_path, name FROM snapshots WHERE id = $1", id).Scan(&filePath, &name)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.tar.gz", name))
	http.ServeFile(w, r, filePath)
}

func createSnapshot(pool *pgxpool.Pool, id int64, filePath string, includes []string) {
	ctx := context.Background()
	os.MkdirAll(snapshotDir, 0755)

	tmpDir := filepath.Join(snapshotDir, fmt.Sprintf("tmp-%d", id))
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	for _, inc := range includes {
		switch inc {
		case "websites":
			exec.Command("cp", "-a", "/www/sites", filepath.Join(tmpDir, "sites")).Run()
		case "nginx":
			exec.Command("cp", "-a", "/www/conf.d", filepath.Join(tmpDir, "conf.d")).Run()
			exec.Command("cp", "-a", "/etc/nginx", filepath.Join(tmpDir, "nginx")).Run()
		case "databases":
			dbDir := filepath.Join(tmpDir, "databases")
			os.MkdirAll(dbDir, 0755)
			// Dump databases from running containers
			rows, _ := pool.Query(ctx, "SELECT container_id, type, root_password, name FROM db_servers WHERE status = 'running'")
			if rows != nil {
				for rows.Next() {
					var cid, dbType, pass, name string
					rows.Scan(&cid, &dbType, &pass, &name)
					switch dbType {
					case "mysql":
						out, _ := exec.Command("docker", "exec", cid, "mysqldump", "-uroot", "-p"+pass, "--all-databases").Output()
						os.WriteFile(filepath.Join(dbDir, name+".sql"), out, 0644)
					case "postgresql":
						out, _ := exec.Command("docker", "exec", cid, "pg_dumpall", "-U", "postgres").Output()
						os.WriteFile(filepath.Join(dbDir, name+".sql"), out, 0644)
					}
				}
				rows.Close()
			}
		case "panel":
			// Dump panel database
			out, _ := exec.Command("pg_dump", "panelox").Output()
			os.WriteFile(filepath.Join(tmpDir, "panelox.sql"), out, 0644)
		}
	}

	// Create tar.gz
	out, err := exec.Command("tar", "-czf", filePath, "-C", tmpDir, ".").CombinedOutput()
	if err != nil {
		pool.Exec(ctx, "UPDATE snapshots SET status = 'failed', message = $1 WHERE id = $2", string(out), id)
		return
	}

	info, _ := os.Stat(filePath)
	var size int64
	if info != nil {
		size = info.Size()
	}

	pool.Exec(ctx, "UPDATE snapshots SET status = 'completed', file_size = $1 WHERE id = $2", size, id)
	log.Printf("Snapshot %d created: %s (%d bytes)", id, filePath, size)
}

func rollbackSnapshot(pool *pgxpool.Pool, id int64, filePath string) {
	ctx := context.Background()

	tmpDir := filepath.Join(snapshotDir, fmt.Sprintf("rollback-%d", id))
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// Extract
	if out, err := exec.Command("tar", "-xzf", filePath, "-C", tmpDir).CombinedOutput(); err != nil {
		log.Printf("Rollback extract failed: %s", string(out))
		return
	}

	// Restore websites
	if _, err := os.Stat(filepath.Join(tmpDir, "sites")); err == nil {
		exec.Command("rm", "-rf", "/www/sites").Run()
		exec.Command("cp", "-a", filepath.Join(tmpDir, "sites"), "/www/sites").Run()
	}

	// Restore nginx configs
	if _, err := os.Stat(filepath.Join(tmpDir, "conf.d")); err == nil {
		exec.Command("rm", "-rf", "/www/conf.d").Run()
		exec.Command("cp", "-a", filepath.Join(tmpDir, "conf.d"), "/www/conf.d").Run()
		exec.Command("nginx", "-s", "reload").Run()
	}

	// Restore panel DB
	if _, err := os.Stat(filepath.Join(tmpDir, "panelox.sql")); err == nil {
		exec.Command("psql", "panelox", "-f", filepath.Join(tmpDir, "panelox.sql")).Run()
	}

	log.Printf("Rollback %d completed from %s", id, filePath)
	_ = ctx
}
