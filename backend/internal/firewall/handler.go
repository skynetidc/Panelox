package firewall

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/firewall", func(r chi.Router) {
		r.Get("/status", h.Status)
		r.Get("/rules", h.ListRules)
		r.Post("/rules", h.CreateRule)
		r.Delete("/rules/{id}", h.DeleteRule)
		r.Post("/reapply", h.ReapplyAll)
	})
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, h.svc.Status())
}

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.svc.ListRules(r.Context())
	if err != nil {
		response.InternalError(w, "failed to list rules")
		return
	}
	response.JSON(w, http.StatusOK, rules)
}

func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Protocol == "" || req.DstPort == "" {
		response.BadRequest(w, "protocol and dst_port are required")
		return
	}

	rule, err := h.svc.CreateRule(r.Context(), req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, rule)
}

func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid rule id")
		return
	}
	if err := h.svc.DeleteRule(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(w, "rule not found")
			return
		}
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "rule deleted"})
}

func (h *Handler) ReapplyAll(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ReapplyAll(r.Context()); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "rules reapplied"})
}
