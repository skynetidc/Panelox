package dns

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
	r.Route("/dns", func(r chi.Router) {
		r.Get("/zones", h.ListZones)
		r.Post("/zones", h.CreateZone)
		r.Get("/zones/{id}", h.GetZone)
		r.Delete("/zones/{id}", h.DeleteZone)
		r.Get("/zones/{id}/records", h.ListRecords)
		r.Post("/zones/{id}/records", h.CreateRecord)
		r.Get("/zones/{id}/export", h.ExportZoneFile)
		r.Put("/records/{id}", h.UpdateRecord)
		r.Delete("/records/{id}", h.DeleteRecord)
	})
}

func (h *Handler) CreateZone(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var req CreateZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Name == "" {
		response.BadRequest(w, "name is required")
		return
	}

	zone, err := h.svc.CreateZone(r.Context(), c.UserID, req)
	if err != nil {
		handleDNSError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, zone)
}

func (h *Handler) GetZone(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid zone id")
		return
	}

	zone, err := h.svc.GetZone(r.Context(), id)
	if err != nil {
		handleDNSError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, zone)
}

func (h *Handler) ListZones(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	zones, total, err := h.svc.ListZones(r.Context(), page, size)
	if err != nil {
		response.InternalError(w, "failed to list zones")
		return
	}

	response.JSONList(w, zones, page, size, total)
}

func (h *Handler) DeleteZone(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid zone id")
		return
	}

	if err := h.svc.DeleteZone(r.Context(), id); err != nil {
		handleDNSError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "zone deleted"})
}

func (h *Handler) CreateRecord(w http.ResponseWriter, r *http.Request) {
	zoneID, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid zone id")
		return
	}

	var req CreateRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Name == "" || req.Type == "" || req.Content == "" {
		response.BadRequest(w, "name, type and content are required")
		return
	}

	rec, err := h.svc.CreateRecord(r.Context(), zoneID, req)
	if err != nil {
		handleDNSError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, rec)
}

func (h *Handler) ListRecords(w http.ResponseWriter, r *http.Request) {
	zoneID, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid zone id")
		return
	}

	records, err := h.svc.ListRecords(r.Context(), zoneID)
	if err != nil {
		response.InternalError(w, "failed to list records")
		return
	}

	response.JSON(w, http.StatusOK, records)
}

func (h *Handler) UpdateRecord(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid record id")
		return
	}

	var req UpdateRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.svc.UpdateRecord(r.Context(), id, req); err != nil {
		handleDNSError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "record updated"})
}

func (h *Handler) DeleteRecord(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid record id")
		return
	}

	if err := h.svc.DeleteRecord(r.Context(), id); err != nil {
		handleDNSError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "record deleted"})
}

func (h *Handler) ExportZoneFile(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid zone id")
		return
	}

	content, err := h.svc.GenerateZoneFile(r.Context(), id)
	if err != nil {
		handleDNSError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename=zone.txt")
	w.Write([]byte(content))
}

func parseID(r *http.Request, param string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, param), 10, 64)
}

func handleDNSError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.NotFound(w, "not found")
	case errors.Is(err, ErrZoneExists):
		response.BadRequest(w, "zone already exists")
	case errors.Is(err, ErrInvalidDomain):
		response.BadRequest(w, "invalid domain name")
	case errors.Is(err, ErrInvalidType):
		response.BadRequest(w, "invalid record type")
	default:
		response.InternalError(w, err.Error())
	}
}
