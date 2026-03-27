package clamav

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/response"
)

type ScanResult struct {
	Path        string `json:"path"`
	Infected    int    `json:"infected"`
	Scanned     int    `json:"scanned"`
	Threats     []string `json:"threats"`
	Output      string `json:"output"`
}

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/clamav/status", h.Status)
	r.Post("/clamav/scan", h.Scan)
	r.Post("/clamav/update", h.UpdateDB)
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	installed := false
	version := ""
	if out, err := exec.Command("clamscan", "--version").Output(); err == nil {
		installed = true
		version = strings.TrimSpace(string(out))
	}
	response.JSON(w, http.StatusOK, map[string]any{"installed": installed, "version": version})
}

func (h *Handler) Scan(w http.ResponseWriter, r *http.Request) {
	var body struct{ Path string `json:"path"` }
	json.NewDecoder(r.Body).Decode(&body)
	if body.Path == "" {
		body.Path = "/www/sites"
	}

	out, _ := exec.Command("clamscan", "-r", "--no-summary", body.Path).CombinedOutput()
	output := string(out)

	result := ScanResult{Path: body.Path, Output: output}
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "FOUND") {
			result.Infected++
			result.Threats = append(result.Threats, strings.TrimSpace(line))
		}
		if strings.Contains(line, ": OK") {
			result.Scanned++
		}
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) UpdateDB(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("freshclam").CombinedOutput()
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]string{"status": "error", "output": string(out)})
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok", "output": string(out)})
}
