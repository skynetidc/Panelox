package website

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/claims"
	"github.com/jakub/panelox/internal/pkg/response"
)

type Handler struct {
	svc       *Service
	getClaims func(r *http.Request) *claims.Claims
}

func NewHandler(svc *Service, getClaims func(r *http.Request) *claims.Claims) *Handler {
	return &Handler{svc: svc, getClaims: getClaims}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/websites", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/operate", h.Operate)
		r.Get("/{id}/nginx", h.GetNginxConfig)
		r.Put("/{id}/nginx", h.UpdateNginxConfig)
		r.Get("/{id}/domains", h.ListDomains)
		r.Post("/{id}/domains", h.AddDomain)
		r.Delete("/{id}/domains/{domainId}", h.DeleteDomain)
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Domain == "" {
		response.BadRequest(w, "domain is required")
		return
	}
	if req.Type == "" {
		req.Type = TypeStatic
	}

	site, err := h.svc.Create(r.Context(), c.UserID, req)
	if err != nil {
		handleWebsiteError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, site)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid website id")
		return
	}

	site, err := h.svc.Get(r.Context(), id)
	if err != nil {
		handleWebsiteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, site)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := ListQuery{
		Search: r.URL.Query().Get("search"),
		Type:   r.URL.Query().Get("type"),
		Status: r.URL.Query().Get("status"),
	}
	q.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	q.Size, _ = strconv.Atoi(r.URL.Query().Get("size"))

	websites, total, err := h.svc.List(r.Context(), q)
	if err != nil {
		response.InternalError(w, "failed to list websites")
		return
	}

	response.JSONList(w, websites, q.Page, q.Size, total)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid website id")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.svc.Update(r.Context(), id, req); err != nil {
		handleWebsiteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "website updated"})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid website id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		handleWebsiteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "website deleted"})
}

func (h *Handler) Operate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid website id")
		return
	}

	var req OperateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.svc.Operate(r.Context(), id, req.Action); err != nil {
		handleWebsiteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "operation completed"})
}

func (h *Handler) GetNginxConfig(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid website id")
		return
	}

	config, err := h.svc.GetNginxConfig(r.Context(), id)
	if err != nil {
		handleWebsiteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"config": config})
}

func (h *Handler) UpdateNginxConfig(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid website id")
		return
	}

	var req NginxConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.svc.UpdateNginxConfig(r.Context(), id, req.Config); err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "nginx config updated"})
}

func (h *Handler) ListDomains(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid website id")
		return
	}

	domains, err := h.svc.ListDomains(r.Context(), id)
	if err != nil {
		response.InternalError(w, "failed to list domains")
		return
	}

	response.JSON(w, http.StatusOK, domains)
}

func (h *Handler) AddDomain(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid website id")
		return
	}

	var req DomainCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Domain == "" {
		response.BadRequest(w, "domain is required")
		return
	}

	domain, err := h.svc.AddDomain(r.Context(), id, req)
	if err != nil {
		handleWebsiteError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, domain)
}

func (h *Handler) DeleteDomain(w http.ResponseWriter, r *http.Request) {
	websiteID, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid website id")
		return
	}
	domainID, err := parseID(r, "domainId")
	if err != nil {
		response.BadRequest(w, "invalid domain id")
		return
	}

	if err := h.svc.DeleteDomain(r.Context(), websiteID, domainID); err != nil {
		handleWebsiteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "domain deleted"})
}

// Helpers

func parseID(r *http.Request, param string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, param), 10, 64)
}

func handleWebsiteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.NotFound(w, "website not found")
	case errors.Is(err, ErrDomainTaken):
		response.BadRequest(w, "domain already in use")
	case errors.Is(err, ErrAliasTaken):
		response.BadRequest(w, "alias already in use")
	case errors.Is(err, ErrInvalidType):
		response.BadRequest(w, "invalid website type (use: static, proxy, php, node)")
	case errors.Is(err, ErrInvalidDomain):
		response.BadRequest(w, "invalid domain name")
	default:
		response.InternalError(w, err.Error())
	}
}
