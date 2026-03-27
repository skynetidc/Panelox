package sshmanage

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/response"
)

type SSHConfig struct {
	Port             string `json:"port"`
	PermitRootLogin  string `json:"permit_root_login"`
	PasswordAuth     string `json:"password_auth"`
	PubkeyAuth       string `json:"pubkey_auth"`
	MaxAuthTries     string `json:"max_auth_tries"`
}

type AuthKey struct {
	Key     string `json:"key"`
	Comment string `json:"comment"`
}

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/ssh/config", h.GetConfig)
	r.Put("/ssh/config", h.UpdateConfig)
	r.Get("/ssh/authorized-keys", h.ListKeys)
	r.Post("/ssh/authorized-keys", h.AddKey)
	r.Delete("/ssh/authorized-keys", h.RemoveKey)
}

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("/etc/ssh/sshd_config")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	cfg := SSHConfig{Port: "22", PermitRootLogin: "yes", PasswordAuth: "yes", PubkeyAuth: "yes", MaxAuthTries: "6"}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") { continue }
		parts := strings.Fields(line)
		if len(parts) < 2 { continue }
		switch parts[0] {
		case "Port": cfg.Port = parts[1]
		case "PermitRootLogin": cfg.PermitRootLogin = parts[1]
		case "PasswordAuthentication": cfg.PasswordAuth = parts[1]
		case "PubkeyAuthentication": cfg.PubkeyAuth = parts[1]
		case "MaxAuthTries": cfg.MaxAuthTries = parts[1]
		}
	}
	response.JSON(w, http.StatusOK, cfg)
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var cfg SSHConfig
	json.NewDecoder(r.Body).Decode(&cfg)

	data, err := os.ReadFile("/etc/ssh/sshd_config")
	if err != nil { response.InternalError(w, err.Error()); return }

	content := string(data)
	updates := map[string]string{
		"Port": cfg.Port, "PermitRootLogin": cfg.PermitRootLogin,
		"PasswordAuthentication": cfg.PasswordAuth, "PubkeyAuthentication": cfg.PubkeyAuth,
		"MaxAuthTries": cfg.MaxAuthTries,
	}

	for key, val := range updates {
		if val == "" { continue }
		re := regexp.MustCompile(`(?m)^#?\s*` + key + `\s+.*$`)
		replacement := key + " " + val
		if re.MatchString(content) {
			content = re.ReplaceAllString(content, replacement)
		} else {
			content += "\n" + replacement
		}
	}

	if err := os.WriteFile("/etc/ssh/sshd_config", []byte(content), 0644); err != nil {
		response.InternalError(w, err.Error())
		return
	}

	exec.Command("systemctl", "reload", "sshd").Run()
	response.JSON(w, http.StatusOK, map[string]string{"message": "SSH config updated and reloaded"})
}

func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("/root/.ssh/authorized_keys")
	if err != nil { response.JSON(w, http.StatusOK, []AuthKey{}); return }

	var keys []AuthKey
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") { continue }
		parts := strings.SplitN(line, " ", 3)
		comment := ""
		if len(parts) >= 3 { comment = parts[2] }
		keys = append(keys, AuthKey{Key: line, Comment: comment})
	}
	response.JSON(w, http.StatusOK, keys)
}

func (h *Handler) AddKey(w http.ResponseWriter, r *http.Request) {
	var body struct{ Key string `json:"key"` }
	json.NewDecoder(r.Body).Decode(&body)
	if body.Key == "" { response.BadRequest(w, "key is required"); return }

	os.MkdirAll("/root/.ssh", 0700)
	f, err := os.OpenFile("/root/.ssh/authorized_keys", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil { response.InternalError(w, err.Error()); return }
	defer f.Close()
	f.WriteString(body.Key + "\n")
	response.JSON(w, http.StatusCreated, map[string]string{"message": "key added"})
}

func (h *Handler) RemoveKey(w http.ResponseWriter, r *http.Request) {
	var body struct{ Key string `json:"key"` }
	json.NewDecoder(r.Body).Decode(&body)

	data, _ := os.ReadFile("/root/.ssh/authorized_keys")
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != strings.TrimSpace(body.Key) && strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	os.WriteFile("/root/.ssh/authorized_keys", []byte(strings.Join(lines, "\n")+"\n"), 0600)
	response.JSON(w, http.StatusOK, map[string]string{"message": "key removed"})
}
