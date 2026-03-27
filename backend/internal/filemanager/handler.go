package filemanager

import (
	"encoding/json"
	"errors"
	"net/http"

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
	r.Route("/files", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/content", h.ReadFile)
		r.Put("/content", h.WriteFile)
		r.Post("/upload", h.Upload)
		r.Delete("/", h.Delete)
		r.Post("/mkdir", h.Mkdir)
		r.Post("/rename", h.Rename)
		r.Post("/chmod", h.Chmod)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/www"
	}

	entries, err := h.svc.List(path)
	if err != nil {
		handleFileError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, entries)
}

func (h *Handler) ReadFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		response.BadRequest(w, "path is required")
		return
	}

	result, err := h.svc.ReadFile(path)
	if err != nil {
		handleFileError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) WriteFile(w http.ResponseWriter, r *http.Request) {
	var req WriteFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Path == "" {
		response.BadRequest(w, "path is required")
		return
	}

	if err := h.svc.WriteFile(req.Path, req.Content); err != nil {
		handleFileError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "file saved"})
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	// Max 100MB upload
	r.ParseMultipartForm(100 << 20)

	path := r.FormValue("path")
	if path == "" {
		path = "/www"
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	if err := h.svc.Upload(path, file, header.Filename); err != nil {
		handleFileError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "file uploaded"})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		response.BadRequest(w, "path is required")
		return
	}

	if err := h.svc.Delete(path); err != nil {
		handleFileError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *Handler) Mkdir(w http.ResponseWriter, r *http.Request) {
	var req MkdirRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.svc.Mkdir(req.Path); err != nil {
		handleFileError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "directory created"})
}

func (h *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	var req RenameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.svc.Rename(req.OldPath, req.NewPath); err != nil {
		handleFileError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "renamed"})
}

func (h *Handler) Chmod(w http.ResponseWriter, r *http.Request) {
	var req ChmodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.svc.Chmod(req.Path, req.Mode); err != nil {
		handleFileError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "permissions changed"})
}

func handleFileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrPathTraversal):
		response.Forbidden(w, "path traversal not allowed")
	case errors.Is(err, ErrNotFound):
		response.NotFound(w, "file or directory not found")
	case errors.Is(err, ErrPermission):
		response.Forbidden(w, "permission denied")
	default:
		response.InternalError(w, err.Error())
	}
}
