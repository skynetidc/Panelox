package container

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/response"
)

// Docker Networks, Volumes, Images management

type Network struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Scope  string `json:"scope"`
}

type Volume struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
	Size       string `json:"size"`
}

type Image struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
	Created    string `json:"created"`
}

// RegisterResourceRoutes adds Docker resources routes.
func RegisterResourceRoutes(r chi.Router, h *Handler) {
	// Networks
	r.Route("/docker/networks", func(r chi.Router) {
		r.Get("/", h.ListNetworks)
		r.Post("/", h.CreateNetwork)
		r.Delete("/{id}", h.DeleteNetwork)
	})

	// Volumes
	r.Route("/docker/volumes", func(r chi.Router) {
		r.Get("/", h.ListVolumes)
		r.Post("/", h.CreateVolume)
		r.Delete("/{name}", h.DeleteVolume)
		r.Post("/prune", h.PruneVolumes)
	})

	// Images
	r.Route("/docker/images", func(r chi.Router) {
		r.Get("/", h.ListImages)
		r.Post("/pull", h.PullImage)
		r.Delete("/{id}", h.DeleteImage)
		r.Post("/prune", h.PruneImages)
	})
}

// --- Networks ---

func (h *Handler) ListNetworks(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("docker", "network", "ls", "--format",
		`{"id":"{{.ID}}","name":"{{.Name}}","driver":"{{.Driver}}","scope":"{{.Scope}}"}`).Output()
	if err != nil {
		response.InternalError(w, "docker network ls failed")
		return
	}

	var networks []Network
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var n Network
		json.Unmarshal([]byte(line), &n)
		networks = append(networks, n)
	}
	response.JSON(w, http.StatusOK, networks)
}

func (h *Handler) CreateNetwork(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Driver string `json:"driver"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" {
		response.BadRequest(w, "name is required")
		return
	}
	if body.Driver == "" {
		body.Driver = "bridge"
	}

	out, err := exec.Command("docker", "network", "create", "--driver", body.Driver, body.Name).CombinedOutput()
	if err != nil {
		response.InternalError(w, fmt.Sprintf("create failed: %s", string(out)))
		return
	}
	response.JSON(w, http.StatusCreated, map[string]string{"message": "network created", "id": strings.TrimSpace(string(out))})
}

func (h *Handler) DeleteNetwork(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	out, err := exec.Command("docker", "network", "rm", id).CombinedOutput()
	if err != nil {
		response.InternalError(w, fmt.Sprintf("delete failed: %s", string(out)))
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "network deleted"})
}

// --- Volumes ---

func (h *Handler) ListVolumes(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("docker", "volume", "ls", "--format",
		`{"name":"{{.Name}}","driver":"{{.Driver}}","mountpoint":"{{.Mountpoint}}"}`).Output()
	if err != nil {
		response.InternalError(w, "docker volume ls failed")
		return
	}

	var volumes []Volume
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var v Volume
		json.Unmarshal([]byte(line), &v)

		// Get size
		sizeOut, _ := exec.Command("docker", "system", "df", "-v", "--format", "{{.Name}}\t{{.Size}}").Output()
		for _, sl := range strings.Split(string(sizeOut), "\n") {
			parts := strings.SplitN(sl, "\t", 2)
			if len(parts) == 2 && parts[0] == v.Name {
				v.Size = parts[1]
			}
		}

		volumes = append(volumes, v)
	}
	response.JSON(w, http.StatusOK, volumes)
}

func (h *Handler) CreateVolume(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Driver string `json:"driver"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" {
		response.BadRequest(w, "name is required")
		return
	}

	args := []string{"volume", "create"}
	if body.Driver != "" {
		args = append(args, "--driver", body.Driver)
	}
	args = append(args, body.Name)

	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		response.InternalError(w, fmt.Sprintf("create failed: %s", string(out)))
		return
	}
	response.JSON(w, http.StatusCreated, map[string]string{"message": "volume created"})
}

func (h *Handler) DeleteVolume(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	out, err := exec.Command("docker", "volume", "rm", name).CombinedOutput()
	if err != nil {
		response.InternalError(w, fmt.Sprintf("delete failed: %s", string(out)))
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "volume deleted"})
}

func (h *Handler) PruneVolumes(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("docker", "volume", "prune", "-f").CombinedOutput()
	if err != nil {
		response.InternalError(w, string(out))
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "unused volumes pruned", "output": string(out)})
}

// --- Images ---

func (h *Handler) ListImages(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("docker", "images", "--format",
		`{"id":"{{.ID}}","repository":"{{.Repository}}","tag":"{{.Tag}}","size":"{{.Size}}","created":"{{.CreatedSince}}"}`).Output()
	if err != nil {
		response.InternalError(w, "docker images failed")
		return
	}

	var images []Image
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var img Image
		json.Unmarshal([]byte(line), &img)
		images = append(images, img)
	}
	response.JSON(w, http.StatusOK, images)
}

func (h *Handler) PullImage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Image string `json:"image"` // e.g. "nginx:latest"
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Image == "" {
		response.BadRequest(w, "image is required")
		return
	}

	out, err := exec.Command("docker", "pull", body.Image).CombinedOutput()
	if err != nil {
		response.InternalError(w, fmt.Sprintf("pull failed: %s", string(out)))
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "image pulled", "output": string(out)})
}

func (h *Handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	out, err := exec.Command("docker", "rmi", id).CombinedOutput()
	if err != nil {
		response.InternalError(w, fmt.Sprintf("delete failed: %s", string(out)))
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "image deleted"})
}

func (h *Handler) PruneImages(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("docker", "image", "prune", "-a", "-f").CombinedOutput()
	if err != nil {
		response.InternalError(w, string(out))
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "unused images pruned", "output": string(out)})
}
