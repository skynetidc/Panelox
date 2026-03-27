package container

import (
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
	r.Route("/containers", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/{id}/start", h.Start)
		r.Post("/{id}/stop", h.Stop)
		r.Post("/{id}/restart", h.Restart)
		r.Delete("/{id}", h.Remove)
		r.Get("/{id}/logs", h.Logs)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	containers, err := h.svc.List()
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, containers)
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Start(chi.URLParam(r, "id")); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "started"})
}

func (h *Handler) Stop(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Stop(chi.URLParam(r, "id")); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "stopped"})
}

func (h *Handler) Restart(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Restart(chi.URLParam(r, "id")); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "restarted"})
}

func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Remove(chi.URLParam(r, "id")); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "removed"})
}

func (h *Handler) Logs(w http.ResponseWriter, r *http.Request) {
	tail, _ := strconv.Atoi(r.URL.Query().Get("tail"))
	logs, err := h.svc.Logs(chi.URLParam(r, "id"), tail)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"logs": logs})
}
