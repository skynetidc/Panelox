package appstore

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/response"
)

type AppRepo struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Enabled   bool   `json:"enabled"`
	LastSync  *string `json:"last_sync,omitempty"`
	CreatedAt string `json:"created_at"`
}

// RegisterAdvancedRoutes adds version management, health checks, repos.
func RegisterAdvancedRoutes(r chi.Router, h *Handler) {
	r.Post("/apps/installed/{id}/upgrade", h.Upgrade)
	r.Get("/apps/installed/{id}/health", h.HealthCheck)
	r.Post("/apps/health-check-all", h.HealthCheckAll)
	r.Get("/apps/repos", h.ListRepos)
	r.Post("/apps/repos", h.AddRepo)
	r.Delete("/apps/repos/{id}", h.DeleteRepo)
}

// --- Version Management ---

func (h *Handler) Upgrade(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Version string `json:"version"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Version == "" {
		body.Version = "latest"
	}

	var name, appKey string
	h.pool.QueryRow(r.Context(), "SELECT name, app_key FROM app_installs WHERE id = $1", id).Scan(&name, &appKey)
	if name == "" {
		response.NotFound(w, "install not found")
		return
	}

	// Get compose from catalog
	var compose string
	h.pool.QueryRow(r.Context(), "SELECT compose FROM app_catalog WHERE key = $1", appKey).Scan(&compose)

	// Update version in compose
	composeFile := filepath.Join(appsDir, name, "docker-compose.yml")

	// Pull new image and restart
	go func() {
		log.Printf("Upgrading %s to version %s", name, body.Version)

		// Docker compose pull
		exec.Command("docker", "compose", "-f", composeFile, "pull").Run()

		// Restart with new image
		exec.Command("docker", "compose", "-f", composeFile, "up", "-d", "--force-recreate").Run()

		// Update DB
		h.pool.Exec(r.Context(), "UPDATE app_installs SET version = $1, updated_at = NOW() WHERE id = $2", body.Version, id)

		log.Printf("Upgrade of %s to %s completed", name, body.Version)
	}()

	response.JSON(w, http.StatusAccepted, map[string]string{"message": fmt.Sprintf("upgrading to %s", body.Version)})
}

// --- Health Checks ---

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var name string
	var containerIDs []string
	h.pool.QueryRow(r.Context(), "SELECT name, container_ids FROM app_installs WHERE id = $1", id).Scan(&name, &containerIDs)
	if name == "" {
		response.NotFound(w, "install not found")
		return
	}

	health := checkAppHealth(name)

	h.pool.Exec(r.Context(),
		"UPDATE app_installs SET health_status = $1, health_checked = NOW() WHERE id = $2",
		health.Status, id)

	response.JSON(w, http.StatusOK, health)
}

func (h *Handler) HealthCheckAll(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), "SELECT id, name FROM app_installs WHERE status = 'running'")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	results := map[string]string{}
	for rows.Next() {
		var id int64
		var name string
		rows.Scan(&id, &name)

		health := checkAppHealth(name)
		results[name] = health.Status

		h.pool.Exec(r.Context(),
			"UPDATE app_installs SET health_status = $1, health_checked = NOW() WHERE id = $2",
			health.Status, id)
	}

	response.JSON(w, http.StatusOK, results)
}

type AppHealth struct {
	Status     string   `json:"status"` // healthy, unhealthy, unknown
	Containers []ContainerHealth `json:"containers"`
}

type ContainerHealth struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Health string `json:"health"`
}

func checkAppHealth(name string) *AppHealth {
	composeFile := filepath.Join(appsDir, name, "docker-compose.yml")

	out, err := exec.Command("docker", "compose", "-f", composeFile, "ps", "--format", "json").Output()
	if err != nil {
		return &AppHealth{Status: "unknown"}
	}

	health := &AppHealth{Status: "healthy"}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var container struct {
			Name  string `json:"Name"`
			State string `json:"State"`
			Health string `json:"Health"`
		}
		json.Unmarshal([]byte(line), &container)

		ch := ContainerHealth{
			Name:   container.Name,
			State:  container.State,
			Health: container.Health,
		}

		if container.State != "running" {
			health.Status = "unhealthy"
		}

		health.Containers = append(health.Containers, ch)
	}

	if len(health.Containers) == 0 {
		health.Status = "unknown"
	}

	return health
}

// --- Custom Repos ---

func (h *Handler) ListRepos(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, name, url, enabled, last_sync, created_at FROM app_repos ORDER BY id")
	if err != nil {
		response.JSON(w, http.StatusOK, []AppRepo{})
		return
	}
	defer rows.Close()

	var repos []AppRepo
	for rows.Next() {
		var repo AppRepo
		rows.Scan(&repo.ID, &repo.Name, &repo.URL, &repo.Enabled, &repo.LastSync, &repo.CreatedAt)
		repos = append(repos, repo)
	}
	response.JSON(w, http.StatusOK, repos)
}

func (h *Handler) AddRepo(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" || body.URL == "" {
		response.BadRequest(w, "name and url are required")
		return
	}

	var repo AppRepo
	h.pool.QueryRow(r.Context(),
		`INSERT INTO app_repos (name, url) VALUES ($1, $2)
		 RETURNING id, name, url, enabled, last_sync, created_at`,
		body.Name, body.URL,
	).Scan(&repo.ID, &repo.Name, &repo.URL, &repo.Enabled, &repo.LastSync, &repo.CreatedAt)

	response.JSON(w, http.StatusCreated, repo)
}

func (h *Handler) DeleteRepo(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "DELETE FROM app_repos WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "repo deleted"})
}

// Background health checker
type HealthChecker struct {
	h    *Handler
	stop chan struct{}
}

func NewHealthChecker(h *Handler) *HealthChecker {
	return &HealthChecker{h: h, stop: make(chan struct{})}
}

func (hc *HealthChecker) Start() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				hc.checkAll()
			case <-hc.stop:
				return
			}
		}
	}()
	log.Println("App health checker started")
}

func (hc *HealthChecker) Stop() { close(hc.stop) }

func (hc *HealthChecker) checkAll() {
	rows, _ := hc.h.pool.Query(nil, "SELECT id, name FROM app_installs WHERE status = 'running'")
	if rows == nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var name string
		rows.Scan(&id, &name)
		health := checkAppHealth(name)
		hc.h.pool.Exec(nil, "UPDATE app_installs SET health_status = $1, health_checked = NOW() WHERE id = $2", health.Status, id)
	}
}
