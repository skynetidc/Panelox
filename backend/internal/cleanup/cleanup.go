package cleanup

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/response"
)

type CleanupResult struct {
	Action  string `json:"action"`
	Freed   string `json:"freed"`
	Details string `json:"details"`
}

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/cleanup/info", h.Info)
	r.Post("/cleanup/logs", h.CleanLogs)
	r.Post("/cleanup/docker", h.CleanDocker)
	r.Post("/cleanup/journal", h.CleanJournal)
	r.Post("/cleanup/cache", h.CleanCache)
	r.Post("/cleanup/all", h.CleanAll)
}

func (h *Handler) Info(w http.ResponseWriter, r *http.Request) {
	info := map[string]string{}
	if out, err := exec.Command("du", "-sh", "/var/log").Output(); err == nil { info["logs"] = strings.Fields(string(out))[0] }
	if out, err := exec.Command("du", "-sh", "/tmp").Output(); err == nil { info["tmp"] = strings.Fields(string(out))[0] }
	if out, err := exec.Command("docker", "system", "df").Output(); err == nil { info["docker"] = string(out) }
	if out, err := exec.Command("journalctl", "--disk-usage").Output(); err == nil { info["journal"] = strings.TrimSpace(string(out)) }
	response.JSON(w, http.StatusOK, info)
}

func (h *Handler) CleanLogs(w http.ResponseWriter, r *http.Request) {
	exec.Command("find", "/var/log", "-name", "*.gz", "-delete").Run()
	exec.Command("find", "/var/log", "-name", "*.old", "-delete").Run()
	exec.Command("find", "/var/log", "-name", "*.1", "-delete").Run()
	out, _ := exec.Command("du", "-sh", "/var/log").Output()
	response.JSON(w, http.StatusOK, CleanupResult{Action: "logs", Freed: "cleaned", Details: strings.TrimSpace(string(out))})
}

func (h *Handler) CleanDocker(w http.ResponseWriter, r *http.Request) {
	out1, _ := exec.Command("docker", "image", "prune", "-a", "-f").CombinedOutput()
	out2, _ := exec.Command("docker", "volume", "prune", "-f").CombinedOutput()
	out3, _ := exec.Command("docker", "builder", "prune", "-f").CombinedOutput()
	response.JSON(w, http.StatusOK, CleanupResult{Action: "docker", Details: fmt.Sprintf("%s\n%s\n%s", out1, out2, out3)})
}

func (h *Handler) CleanJournal(w http.ResponseWriter, r *http.Request) {
	out, _ := exec.Command("journalctl", "--vacuum-time=3d").CombinedOutput()
	response.JSON(w, http.StatusOK, CleanupResult{Action: "journal", Details: string(out)})
}

func (h *Handler) CleanCache(w http.ResponseWriter, r *http.Request) {
	exec.Command("rm", "-rf", "/tmp/*").Run()
	exec.Command("apt", "clean").Run()
	response.JSON(w, http.StatusOK, CleanupResult{Action: "cache", Details: "tmp + apt cache cleaned"})
}

func (h *Handler) CleanAll(w http.ResponseWriter, r *http.Request) {
	exec.Command("find", "/var/log", "-name", "*.gz", "-delete").Run()
	exec.Command("journalctl", "--vacuum-time=3d").Run()
	exec.Command("docker", "system", "prune", "-a", "-f").Run()
	exec.Command("rm", "-rf", "/tmp/*").Run()
	exec.Command("apt", "clean").Run()
	response.JSON(w, http.StatusOK, CleanupResult{Action: "all", Details: "logs + journal + docker + cache cleaned"})
}
