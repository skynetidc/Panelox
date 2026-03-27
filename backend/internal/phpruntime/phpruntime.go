package phpruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
)

var ErrNotFound = errors.New("runtime not found")

// Available PHP versions with their Docker images
var AvailableVersions = []string{"8.3", "8.2", "8.1", "8.0", "7.4"}

type Runtime struct {
	ID            int64     `json:"id"`
	Version       string    `json:"version"`
	ContainerID   string    `json:"container_id,omitempty"`
	ContainerName string    `json:"container_name"`
	Status        string    `json:"status"`
	Port          int       `json:"port"`
	Extensions    []string  `json:"extensions"`
	CreatedAt     time.Time `json:"created_at"`
}

type InstallRequest struct {
	Version    string   `json:"version"`
	Extensions []string `json:"extensions"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/php", func(r chi.Router) {
		r.Get("/versions", h.ListAvailable)
		r.Get("/runtimes", h.ListInstalled)
		r.Post("/runtimes", h.Install)
		r.Delete("/runtimes/{id}", h.Remove)
		r.Post("/runtimes/{id}/operate", h.Operate)
	})
}

func (h *Handler) ListAvailable(w http.ResponseWriter, r *http.Request) {
	// Return all versions with installed status
	installed := map[string]bool{}
	rows, _ := h.pool.Query(r.Context(), "SELECT version FROM php_runtimes")
	if rows != nil {
		for rows.Next() {
			var v string
			rows.Scan(&v)
			installed[v] = true
		}
		rows.Close()
	}

	type VersionInfo struct {
		Version   string `json:"version"`
		Image     string `json:"image"`
		Installed bool   `json:"installed"`
	}

	var versions []VersionInfo
	for _, v := range AvailableVersions {
		versions = append(versions, VersionInfo{
			Version:   v,
			Image:     fmt.Sprintf("php:%s-fpm-alpine", v),
			Installed: installed[v],
		})
	}
	response.JSON(w, http.StatusOK, versions)
}

func (h *Handler) ListInstalled(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, version, COALESCE(container_id,''), container_name, status, port, extensions, created_at FROM php_runtimes ORDER BY version DESC")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var runtimes []Runtime
	for rows.Next() {
		var rt Runtime
		rows.Scan(&rt.ID, &rt.Version, &rt.ContainerID, &rt.ContainerName, &rt.Status, &rt.Port, &rt.Extensions, &rt.CreatedAt)
		runtimes = append(runtimes, rt)
	}
	response.JSON(w, http.StatusOK, runtimes)
}

func (h *Handler) Install(w http.ResponseWriter, r *http.Request) {
	var req InstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Version == "" {
		response.BadRequest(w, "version is required")
		return
	}

	// Check not already installed
	var exists bool
	h.pool.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM php_runtimes WHERE version = $1)", req.Version).Scan(&exists)
	if exists {
		response.BadRequest(w, fmt.Sprintf("PHP %s is already installed", req.Version))
		return
	}

	// Find available port
	port := 9000
	var maxPort int
	h.pool.QueryRow(r.Context(), "SELECT COALESCE(MAX(port), 8999) FROM php_runtimes").Scan(&maxPort)
	if maxPort >= 9000 {
		port = maxPort + 1
	}

	containerName := fmt.Sprintf("panelox-php-%s", strings.ReplaceAll(req.Version, ".", ""))

	if req.Extensions == nil {
		req.Extensions = []string{}
	}

	var rt Runtime
	err := h.pool.QueryRow(r.Context(),
		`INSERT INTO php_runtimes (version, container_name, status, port, extensions)
		 VALUES ($1, $2, 'installing', $3, $4) RETURNING id, version, container_id, container_name, status, port, extensions, created_at`,
		req.Version, containerName, port, req.Extensions,
	).Scan(&rt.ID, &rt.Version, &rt.ContainerID, &rt.ContainerName, &rt.Status, &rt.Port, &rt.Extensions, &rt.CreatedAt)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	go installPHP(h.pool, rt.ID, req.Version, containerName, port, req.Extensions)
	response.JSON(w, http.StatusAccepted, rt)
}

func installPHP(pool *pgxpool.Pool, id int64, version, containerName string, port int, extensions []string) {
	ctx := context.Background()
	image := fmt.Sprintf("php:%s-fpm-alpine", version)

	// Build custom Dockerfile if extensions needed
	args := []string{
		"run", "-d",
		"--name", containerName,
		"--restart", "unless-stopped",
		"-p", fmt.Sprintf("127.0.0.1:%d:9000", port),
		"-v", "/www/sites:/www/sites",
		"--label", "panelox.managed=true",
		"--label", "panelox.type=php-fpm",
		"--label", fmt.Sprintf("panelox.php-version=%s", version),
	}

	// Install extensions via docker-php-ext-install if needed
	if len(extensions) > 0 {
		// For simplicity, use the base image and install extensions after
		args = append(args, image)
	} else {
		args = append(args, image)
	}

	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		log.Printf("PHP %s install failed: %v, output: %s", version, err, string(out))
		pool.Exec(ctx, "UPDATE php_runtimes SET status = 'error' WHERE id = $1", id)
		return
	}

	containerID := strings.TrimSpace(string(out))
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}

	// Install extensions if any
	for _, ext := range extensions {
		exec.Command("docker", "exec", containerName, "docker-php-ext-install", ext).Run()
	}

	pool.Exec(ctx, "UPDATE php_runtimes SET status = 'running', container_id = $1 WHERE id = $2", containerID, id)
	log.Printf("PHP %s installed: container %s on port %d", version, containerID, port)
}

func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var containerName string
	err := h.pool.QueryRow(r.Context(), "SELECT container_name FROM php_runtimes WHERE id = $1", id).Scan(&containerName)
	if err == pgx.ErrNoRows {
		response.NotFound(w, "runtime not found")
		return
	}

	exec.Command("docker", "stop", containerName).Run()
	exec.Command("docker", "rm", containerName).Run()
	h.pool.Exec(r.Context(), "DELETE FROM php_runtimes WHERE id = $1", id)

	response.JSON(w, http.StatusOK, map[string]string{"message": "PHP runtime removed"})
}

func (h *Handler) Operate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct{ Action string `json:"action"` }
	json.NewDecoder(r.Body).Decode(&body)

	var containerName string
	h.pool.QueryRow(r.Context(), "SELECT container_name FROM php_runtimes WHERE id = $1", id).Scan(&containerName)
	if containerName == "" {
		response.NotFound(w, "runtime not found")
		return
	}

	switch body.Action {
	case "start":
		exec.Command("docker", "start", containerName).Run()
		h.pool.Exec(r.Context(), "UPDATE php_runtimes SET status = 'running' WHERE id = $1", id)
	case "stop":
		exec.Command("docker", "stop", containerName).Run()
		h.pool.Exec(r.Context(), "UPDATE php_runtimes SET status = 'stopped' WHERE id = $1", id)
	case "restart":
		exec.Command("docker", "restart", containerName).Run()
		h.pool.Exec(r.Context(), "UPDATE php_runtimes SET status = 'running' WHERE id = $1", id)
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": body.Action + " completed"})
}
