package staging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/nginx"
	"github.com/jakub/panelox/internal/pkg/response"
)

type StagingSite struct {
	ID          int64  `json:"id"`
	WebsiteID   int64  `json:"website_id"`
	Subdomain   string `json:"subdomain"`
	ContainerID string `json:"container_id,omitempty"`
	Status      string `json:"status"`
	URL         string `json:"url"`
	CreatedAt   string `json:"created_at"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/staging", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/sync", h.SyncFromProduction)
		r.Post("/{id}/promote", h.PromoteToProduction)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	websiteID, _ := strconv.ParseInt(r.URL.Query().Get("website_id"), 10, 64)
	query := "SELECT id, website_id, subdomain, container_id, status, created_at FROM staging_sites"
	args := []any{}
	if websiteID > 0 {
		query += " WHERE website_id = $1"
		args = append(args, websiteID)
	}
	query += " ORDER BY id DESC"

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var sites []StagingSite
	for rows.Next() {
		var s StagingSite
		rows.Scan(&s.ID, &s.WebsiteID, &s.Subdomain, &s.ContainerID, &s.Status, &s.CreatedAt)
		s.URL = fmt.Sprintf("http://%s", s.Subdomain)
		sites = append(sites, s)
	}
	response.JSON(w, http.StatusOK, sites)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WebsiteID int64 `json:"website_id"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.WebsiteID == 0 {
		response.BadRequest(w, "website_id is required")
		return
	}

	// Get website info
	var domain, alias string
	h.pool.QueryRow(r.Context(), "SELECT domain, alias FROM websites WHERE id = $1", body.WebsiteID).Scan(&domain, &alias)
	if domain == "" {
		response.NotFound(w, "website not found")
		return
	}

	subdomain := fmt.Sprintf("staging-%s.%s", alias, domain)
	stagingAlias := fmt.Sprintf("staging-%s", alias)
	stagingRoot := fmt.Sprintf("/www/sites/%s/index", stagingAlias)

	// Copy production files to staging
	prodRoot := fmt.Sprintf("/www/sites/%s/index", alias)
	os.MkdirAll(stagingRoot, 0755)
	exec.Command("cp", "-a", prodRoot+"/.", stagingRoot+"/").Run()

	// Generate nginx config for staging
	paths := nginx.GetSitePaths(stagingAlias)
	for _, dir := range paths.AllDirs() {
		os.MkdirAll(dir, 0755)
	}

	cfg := nginx.Config{
		Alias:     stagingAlias,
		Domains:   []nginx.Domain{{Name: subdomain, Port: 80}},
		Type:      "static",
		AccessLog: true,
		ErrorLog:  true,
	}
	config := nginx.GenerateServerBlock(cfg)
	os.WriteFile(paths.ConfFile, []byte(config), 0644)
	nginx.TestAndReload()

	var s StagingSite
	h.pool.QueryRow(r.Context(),
		`INSERT INTO staging_sites (website_id, subdomain, status)
		 VALUES ($1, $2, 'active') RETURNING id, website_id, subdomain, container_id, status, created_at`,
		body.WebsiteID, subdomain,
	).Scan(&s.ID, &s.WebsiteID, &s.Subdomain, &s.ContainerID, &s.Status, &s.CreatedAt)
	s.URL = fmt.Sprintf("http://%s", subdomain)

	response.JSON(w, http.StatusCreated, s)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var subdomain string
	h.pool.QueryRow(r.Context(), "SELECT subdomain FROM staging_sites WHERE id = $1", id).Scan(&subdomain)

	// Remove nginx config and files
	stagingAlias := strings.Replace(subdomain, ".", "-", -1)
	paths := nginx.GetSitePaths(stagingAlias)
	os.Remove(paths.ConfFile)
	os.RemoveAll(paths.Root)
	nginx.TestAndReload()

	h.pool.Exec(r.Context(), "DELETE FROM staging_sites WHERE id = $1", id)
	response.JSON(w, http.StatusOK, map[string]string{"message": "staging site deleted"})
}

func (h *Handler) SyncFromProduction(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var websiteID int64
	h.pool.QueryRow(r.Context(), "SELECT website_id FROM staging_sites WHERE id = $1", id).Scan(&websiteID)

	var alias string
	h.pool.QueryRow(r.Context(), "SELECT alias FROM websites WHERE id = $1", websiteID).Scan(&alias)

	stagingAlias := fmt.Sprintf("staging-%s", alias)
	exec.Command("rsync", "-a", "--delete",
		fmt.Sprintf("/www/sites/%s/index/", alias),
		fmt.Sprintf("/www/sites/%s/index/", stagingAlias)).Run()

	response.JSON(w, http.StatusOK, map[string]string{"message": "synced from production"})
}

func (h *Handler) PromoteToProduction(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var websiteID int64
	h.pool.QueryRow(r.Context(), "SELECT website_id FROM staging_sites WHERE id = $1", id).Scan(&websiteID)

	var alias string
	h.pool.QueryRow(r.Context(), "SELECT alias FROM websites WHERE id = $1", websiteID).Scan(&alias)

	stagingAlias := fmt.Sprintf("staging-%s", alias)

	// Backup production first
	exec.Command("cp", "-a",
		fmt.Sprintf("/www/sites/%s/index", alias),
		fmt.Sprintf("/www/sites/%s/index.bak", alias)).Run()

	// Promote staging to production
	exec.Command("rsync", "-a", "--delete",
		fmt.Sprintf("/www/sites/%s/index/", stagingAlias),
		fmt.Sprintf("/www/sites/%s/index/", alias)).Run()

	response.JSON(w, http.StatusOK, map[string]string{"message": "staging promoted to production"})
}
