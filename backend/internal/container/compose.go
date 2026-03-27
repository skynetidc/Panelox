package container

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/response"
)

const composeDir = "/opt/panelox/compose"

type ComposeProject struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	ConfigDir string `json:"config_dir"`
	Services  int    `json:"services"`
}

type ComposeCreateRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"` // docker-compose.yml content
	EnvFile string `json:"env_file"` // .env content
}

// RegisterComposeRoutes adds Compose project management routes.
func RegisterComposeRoutes(r chi.Router, h *Handler) {
	r.Route("/compose", func(r chi.Router) {
		r.Get("/", h.ListCompose)
		r.Post("/", h.CreateCompose)
		r.Post("/{name}/up", h.ComposeUp)
		r.Post("/{name}/down", h.ComposeDown)
		r.Post("/{name}/restart", h.ComposeRestart)
		r.Get("/{name}/logs", h.ComposeLogs)
		r.Get("/{name}/config", h.ComposeGetConfig)
		r.Put("/{name}/config", h.ComposeUpdateConfig)
		r.Delete("/{name}", h.ComposeDelete)
	})
}

func (h *Handler) ListCompose(w http.ResponseWriter, r *http.Request) {
	os.MkdirAll(composeDir, 0755)
	entries, err := os.ReadDir(composeDir)
	if err != nil {
		response.JSON(w, http.StatusOK, []ComposeProject{})
		return
	}

	var projects []ComposeProject
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		composePath := filepath.Join(composeDir, e.Name(), "docker-compose.yml")
		if _, err := os.Stat(composePath); err != nil {
			continue
		}

		// Get status
		out, _ := exec.Command("docker", "compose", "-f", composePath, "ps", "--format", "json").Output()
		svcCount := strings.Count(string(out), "\"Name\"")
		status := "stopped"
		if strings.Contains(string(out), "running") {
			status = "running"
		}

		projects = append(projects, ComposeProject{
			Name:      e.Name(),
			Status:    status,
			ConfigDir: filepath.Join(composeDir, e.Name()),
			Services:  svcCount,
		})
	}
	response.JSON(w, http.StatusOK, projects)
}

func (h *Handler) CreateCompose(w http.ResponseWriter, r *http.Request) {
	var req ComposeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Name == "" || req.Content == "" {
		response.BadRequest(w, "name and content are required")
		return
	}

	dir := filepath.Join(composeDir, req.Name)
	os.MkdirAll(dir, 0755)

	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(req.Content), 0644)
	if req.EnvFile != "" {
		os.WriteFile(filepath.Join(dir, ".env"), []byte(req.EnvFile), 0644)
	}

	response.JSON(w, http.StatusCreated, map[string]string{"message": "project created", "path": dir})
}

func (h *Handler) ComposeUp(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	composePath := filepath.Join(composeDir, name, "docker-compose.yml")

	out, err := exec.Command("docker", "compose", "-f", composePath, "up", "-d").CombinedOutput()
	if err != nil {
		response.InternalError(w, fmt.Sprintf("up failed: %s", string(out)))
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "started", "output": string(out)})
}

func (h *Handler) ComposeDown(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	composePath := filepath.Join(composeDir, name, "docker-compose.yml")

	out, err := exec.Command("docker", "compose", "-f", composePath, "down").CombinedOutput()
	if err != nil {
		response.InternalError(w, fmt.Sprintf("down failed: %s", string(out)))
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "stopped", "output": string(out)})
}

func (h *Handler) ComposeRestart(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	composePath := filepath.Join(composeDir, name, "docker-compose.yml")
	exec.Command("docker", "compose", "-f", composePath, "restart").Run()
	response.JSON(w, http.StatusOK, map[string]string{"message": "restarted"})
}

func (h *Handler) ComposeLogs(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	composePath := filepath.Join(composeDir, name, "docker-compose.yml")

	out, _ := exec.Command("docker", "compose", "-f", composePath, "logs", "--tail", "200").CombinedOutput()
	response.JSON(w, http.StatusOK, map[string]string{"logs": string(out)})
}

func (h *Handler) ComposeGetConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	content, err := os.ReadFile(filepath.Join(composeDir, name, "docker-compose.yml"))
	if err != nil {
		response.NotFound(w, "compose file not found")
		return
	}
	envContent, _ := os.ReadFile(filepath.Join(composeDir, name, ".env"))
	response.JSON(w, http.StatusOK, map[string]string{"compose": string(content), "env": string(envContent)})
}

func (h *Handler) ComposeUpdateConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var body struct {
		Content string `json:"content"`
		EnvFile string `json:"env_file"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if body.Content != "" {
		os.WriteFile(filepath.Join(composeDir, name, "docker-compose.yml"), []byte(body.Content), 0644)
	}
	if body.EnvFile != "" {
		os.WriteFile(filepath.Join(composeDir, name, ".env"), []byte(body.EnvFile), 0644)
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "config updated"})
}

func (h *Handler) ComposeDelete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	composePath := filepath.Join(composeDir, name, "docker-compose.yml")
	exec.Command("docker", "compose", "-f", composePath, "down", "-v").Run()
	os.RemoveAll(filepath.Join(composeDir, name))
	response.JSON(w, http.StatusOK, map[string]string{"message": "project deleted"})
}
