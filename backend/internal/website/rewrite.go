package website

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/nginx"
	"github.com/jakub/panelox/internal/pkg/response"
)

type RewriteRule struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // rewrite, redirect
	Flag   string `json:"flag"` // last, permanent, redirect, break
}

type LoadBalancer struct {
	Algorithm string   `json:"algorithm"` // round-robin, least_conn, ip_hash
	Servers   []string `json:"servers"`
}

func RegisterRewriteRoutes(r chi.Router, h *Handler) {
	r.Get("/websites/{id}/rewrite", h.GetRewrite)
	r.Put("/websites/{id}/rewrite", h.SetRewrite)
	r.Get("/websites/{id}/loadbalance", h.GetLoadBalance)
	r.Put("/websites/{id}/loadbalance", h.SetLoadBalance)
	r.Get("/disk/usage", h.DiskUsage)
}

func (h *Handler) GetRewrite(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	site, err := h.svc.Get(r.Context(), id)
	if err != nil { handleWebsiteError(w, err); return }

	paths := nginx.GetSitePaths(site.Alias)
	rewritePath := paths.Rewrite + "/rules.conf"
	data, _ := os.ReadFile(rewritePath)
	response.JSON(w, http.StatusOK, map[string]string{"rules": string(data), "path": rewritePath})
}

func (h *Handler) SetRewrite(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	site, err := h.svc.Get(r.Context(), id)
	if err != nil { handleWebsiteError(w, err); return }

	var body struct {
		Rules []RewriteRule `json:"rules"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	paths := nginx.GetSitePaths(site.Alias)
	os.MkdirAll(paths.Rewrite, 0755)
	rewritePath := paths.Rewrite + "/rules.conf"

	var content strings.Builder
	for _, rule := range body.Rules {
		flag := rule.Flag
		if flag == "" { flag = "last" }
		switch rule.Type {
		case "redirect":
			content.WriteString(fmt.Sprintf("rewrite %s %s redirect;\n", rule.Source, rule.Target))
		case "permanent":
			content.WriteString(fmt.Sprintf("rewrite %s %s permanent;\n", rule.Source, rule.Target))
		default:
			content.WriteString(fmt.Sprintf("rewrite %s %s %s;\n", rule.Source, rule.Target, flag))
		}
	}

	os.WriteFile(rewritePath, []byte(content.String()), 0644)
	nginx.TestAndReload()
	response.JSON(w, http.StatusOK, map[string]string{"message": "rewrite rules updated"})
}

func (h *Handler) GetLoadBalance(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	site, err := h.svc.Get(r.Context(), id)
	if err != nil { handleWebsiteError(w, err); return }

	paths := nginx.GetSitePaths(site.Alias)
	lbPath := paths.Root + "/upstream.conf"
	data, _ := os.ReadFile(lbPath)
	response.JSON(w, http.StatusOK, map[string]string{"config": string(data), "path": lbPath})
}

func (h *Handler) SetLoadBalance(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	site, err := h.svc.Get(r.Context(), id)
	if err != nil { handleWebsiteError(w, err); return }

	var lb LoadBalancer
	json.NewDecoder(r.Body).Decode(&lb)

	var content strings.Builder
	content.WriteString(fmt.Sprintf("upstream %s_backend {\n", site.Alias))
	if lb.Algorithm == "least_conn" {
		content.WriteString("    least_conn;\n")
	} else if lb.Algorithm == "ip_hash" {
		content.WriteString("    ip_hash;\n")
	}
	for _, srv := range lb.Servers {
		content.WriteString(fmt.Sprintf("    server %s;\n", srv))
	}
	content.WriteString("}\n")

	paths := nginx.GetSitePaths(site.Alias)
	os.WriteFile(paths.Root+"/upstream.conf", []byte(content.String()), 0644)
	nginx.TestAndReload()
	response.JSON(w, http.StatusOK, map[string]string{"message": "load balancer updated"})
}

func (h *Handler) DiskUsage(w http.ResponseWriter, r *http.Request) {
	type DiskInfo struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Size string `json:"size"`
	}

	var items []DiskInfo

	// Per-website disk usage
	rows, err := h.svc.repo.pool.Query(r.Context(), "SELECT alias, domain FROM websites")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var alias, domain string
			rows.Scan(&alias, &domain)
			dir := fmt.Sprintf("/www/sites/%s", alias)
			if out, err := exec.Command("du", "-sh", dir).Output(); err == nil {
				parts := strings.Fields(string(out))
				if len(parts) >= 1 {
					items = append(items, DiskInfo{Name: domain, Path: dir, Size: parts[0]})
				}
			}
		}
	}

	response.JSON(w, http.StatusOK, items)
}
