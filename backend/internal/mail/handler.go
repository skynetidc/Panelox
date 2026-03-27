package mail

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
	r.Route("/mail", func(r chi.Router) {
		r.Get("/domains", h.ListDomains)
		r.Post("/domains", h.CreateDomain)
		r.Delete("/domains/{id}", h.DeleteDomain)
		r.Get("/domains/{id}/mailboxes", h.ListMailboxes)
		r.Post("/domains/{id}/mailboxes", h.CreateMailbox)
		r.Delete("/mailboxes/{id}", h.DeleteMailbox)
	})
}

func (h *Handler) CreateDomain(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}
	var req CreateDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Domain == "" {
		response.BadRequest(w, "domain is required")
		return
	}
	d, err := h.svc.CreateDomain(r.Context(), c.UserID, req)
	if err != nil {
		if errors.Is(err, ErrDomainTaken) {
			response.BadRequest(w, "domain already exists")
			return
		}
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, d)
}

func (h *Handler) ListDomains(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	domains, total, err := h.svc.ListDomains(r.Context(), page, size)
	if err != nil {
		response.InternalError(w, "failed to list domains")
		return
	}
	response.JSONList(w, domains, page, size, total)
}

func (h *Handler) DeleteDomain(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.svc.DeleteDomain(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(w, "domain not found")
			return
		}
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "domain deleted"})
}

func (h *Handler) CreateMailbox(w http.ResponseWriter, r *http.Request) {
	domainID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req CreateMailboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Address == "" || req.Password == "" {
		response.BadRequest(w, "address and password are required")
		return
	}
	m, err := h.svc.CreateMailbox(r.Context(), domainID, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(w, "domain not found")
			return
		}
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, m)
}

func (h *Handler) ListMailboxes(w http.ResponseWriter, r *http.Request) {
	domainID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	mailboxes, err := h.svc.ListMailboxes(r.Context(), domainID)
	if err != nil {
		response.InternalError(w, "failed to list mailboxes")
		return
	}
	response.JSON(w, http.StatusOK, mailboxes)
}

func (h *Handler) DeleteMailbox(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.svc.DeleteMailbox(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(w, "mailbox not found")
			return
		}
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "mailbox deleted"})
}
