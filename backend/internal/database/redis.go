package database

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/response"
)

type RedisInfo struct {
	Version       string `json:"version"`
	Mode          string `json:"mode"`
	ConnectedClients int `json:"connected_clients"`
	UsedMemory    string `json:"used_memory"`
	UsedMemoryPeak string `json:"used_memory_peak"`
	TotalKeys     int    `json:"total_keys"`
	Uptime        int    `json:"uptime_seconds"`
}

type RedisCLIRequest struct {
	Command string `json:"command"`
}

// RegisterRedisRoutes adds Redis management routes.
func RegisterRedisRoutes(r chi.Router, h *Handler) {
	r.Get("/databases/redis/servers", h.ListRedisServers)
	r.Post("/databases/redis/servers", h.CreateRedisServer)
	r.Delete("/databases/redis/servers/{id}", h.DeleteRedisServer)
	r.Post("/databases/redis/servers/{id}/operate", h.OperateRedisServer)
	r.Get("/databases/redis/servers/{id}/info", h.RedisInfo)
	r.Post("/databases/redis/servers/{id}/cli", h.RedisCLI)
}

func (h *Handler) ListRedisServers(w http.ResponseWriter, r *http.Request) {
	servers, total, err := h.svc.ListServers(r.Context(), 1, 100)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	// Filter Redis servers
	var redisServers []DBServer
	for _, s := range servers {
		if s.Type == "redis" {
			redisServers = append(redisServers, s)
		}
	}
	response.JSONList(w, redisServers, 1, 100, total)
}

func (h *Handler) CreateRedisServer(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var body struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Port     int    `json:"port"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if body.Name == "" {
		response.BadRequest(w, "name is required")
		return
	}
	if body.Port == 0 {
		body.Port = 6379
	}
	if body.Password == "" {
		body.Password = "panelox-redis"
	}

	containerName := fmt.Sprintf("panelox-redis-%s", strings.ToLower(body.Name))

	srv := &DBServer{
		UserID:       c.UserID,
		Name:         body.Name,
		Type:         "redis",
		Version:      "7",
		Host:         "127.0.0.1",
		Port:         body.Port,
		RootPassword: body.Password,
		Status:       StatusCreating,
	}

	if err := h.svc.repo.CreateServer(r.Context(), srv); err != nil {
		response.InternalError(w, err.Error())
		return
	}

	go func() {
		args := []string{
			"run", "-d",
			"--name", containerName,
			"--restart", "unless-stopped",
			"-p", fmt.Sprintf("%d:6379", body.Port),
			"--label", "panelox.managed=true",
			"--label", "panelox.type=redis",
			"-v", fmt.Sprintf("panelox-redis-%s-data:/data", body.Name),
			"redis:7-alpine",
			"redis-server", "--requirepass", body.Password, "--appendonly", "yes",
		}

		out, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			h.svc.repo.UpdateServerStatus(r.Context(), srv.ID, StatusError, "")
			fmt.Printf("Redis create failed: %s\n", string(out))
			return
		}

		containerID := strings.TrimSpace(string(out))
		if len(containerID) > 12 {
			containerID = containerID[:12]
		}
		h.svc.repo.UpdateServerStatus(r.Context(), srv.ID, StatusRunning, containerID)
	}()

	response.JSON(w, http.StatusAccepted, srv)
}

func (h *Handler) DeleteRedisServer(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.svc.DeleteServer(r.Context(), id); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "redis server deleted"})
}

func (h *Handler) OperateRedisServer(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct{ Action string `json:"action"` }
	json.NewDecoder(r.Body).Decode(&body)

	if err := h.svc.OperateServer(r.Context(), id, body.Action); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": body.Action + " completed"})
}

func (h *Handler) RedisInfo(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	srv, err := h.svc.GetServer(r.Context(), id)
	if err != nil {
		response.NotFound(w, "server not found")
		return
	}

	out, err := exec.Command("docker", "exec", srv.ContainerID,
		"redis-cli", "-a", srv.RootPassword, "INFO").CombinedOutput()
	if err != nil {
		response.InternalError(w, string(out))
		return
	}

	info := parseRedisInfo(string(out))
	response.JSON(w, http.StatusOK, info)
}

func (h *Handler) RedisCLI(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	srv, err := h.svc.GetServer(r.Context(), id)
	if err != nil {
		response.NotFound(w, "server not found")
		return
	}

	var req RedisCLIRequest
	json.NewDecoder(r.Body).Decode(&req)
	if req.Command == "" {
		response.BadRequest(w, "command is required")
		return
	}

	args := []string{"exec", srv.ContainerID, "redis-cli", "-a", srv.RootPassword}
	args = append(args, strings.Fields(req.Command)...)

	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]string{"output": string(out), "error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"output": string(out)})
}

func parseRedisInfo(raw string) *RedisInfo {
	info := &RedisInfo{}
	for _, line := range strings.Split(raw, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "redis_version":
			info.Version = parts[1]
		case "redis_mode":
			info.Mode = parts[1]
		case "connected_clients":
			info.ConnectedClients, _ = strconv.Atoi(parts[1])
		case "used_memory_human":
			info.UsedMemory = parts[1]
		case "used_memory_peak_human":
			info.UsedMemoryPeak = parts[1]
		case "uptime_in_seconds":
			info.Uptime, _ = strconv.Atoi(parts[1])
		}
	}
	return info
}
