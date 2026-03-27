package process

import (
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/response"
)

type Process struct {
	PID     int     `json:"pid"`
	User    string  `json:"user"`
	CPU     float64 `json:"cpu"`
	MEM     float64 `json:"mem"`
	VSZ     string  `json:"vsz"`
	RSS     string  `json:"rss"`
	Stat    string  `json:"stat"`
	Command string  `json:"command"`
}

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/processes", h.List)
	r.Post("/processes/{pid}/kill", h.Kill)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("ps", "aux", "--sort=-pcpu").Output()
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	var procs []Process
	for i, line := range strings.Split(string(out), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		pid, _ := strconv.Atoi(fields[1])
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		cmd := strings.Join(fields[10:], " ")
		procs = append(procs, Process{
			PID: pid, User: fields[0], CPU: cpu, MEM: mem,
			VSZ: fields[4], RSS: fields[5], Stat: fields[7], Command: cmd,
		})
	}
	response.JSON(w, http.StatusOK, procs)
}

func (h *Handler) Kill(w http.ResponseWriter, r *http.Request) {
	pid := chi.URLParam(r, "pid")
	signal := r.URL.Query().Get("signal")
	if signal == "" {
		signal = "TERM"
	}

	out, err := exec.Command("kill", fmt.Sprintf("-%s", signal), pid).CombinedOutput()
	if err != nil {
		response.InternalError(w, fmt.Sprintf("kill failed: %s", string(out)))
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("signal %s sent to PID %s", signal, pid)})
}
