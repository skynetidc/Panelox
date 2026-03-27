package database

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
	r.Route("/databases", func(r chi.Router) {
		// Servers
		r.Get("/servers", h.ListServers)
		r.Post("/servers", h.CreateServer)
		r.Get("/servers/{id}", h.GetServer)
		r.Delete("/servers/{id}", h.DeleteServer)
		r.Post("/servers/{id}/operate", h.OperateServer)

		// Databases within a server
		r.Get("/servers/{id}/databases", h.ListDatabases)
		r.Post("/servers/{id}/databases", h.CreateDatabase)
		r.Delete("/{id}", h.DeleteDatabase)

		// Users within a server
		r.Get("/servers/{id}/users", h.ListDBUsers)
		r.Post("/servers/{id}/users", h.CreateDBUser)
		r.Delete("/users/{id}", h.DeleteDBUser)
	})
}

func (h *Handler) CreateServer(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var req CreateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Name == "" || req.Type == "" || req.RootPassword == "" {
		response.BadRequest(w, "name, type and root_password are required")
		return
	}

	srv, err := h.svc.CreateServer(r.Context(), c.UserID, req)
	if err != nil {
		handleDBError(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, srv)
}

func (h *Handler) GetServer(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	srv, err := h.svc.GetServer(r.Context(), id)
	if err != nil {
		handleDBError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, srv)
}

func (h *Handler) ListServers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	servers, total, err := h.svc.ListServers(r.Context(), page, size)
	if err != nil {
		response.InternalError(w, "failed to list servers")
		return
	}

	response.JSONList(w, servers, page, size, total)
}

func (h *Handler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	if err := h.svc.DeleteServer(r.Context(), id); err != nil {
		handleDBError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "server deleted"})
}

func (h *Handler) OperateServer(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.svc.OperateServer(r.Context(), id, body.Action); err != nil {
		handleDBError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "operation completed"})
}

func (h *Handler) CreateDatabase(w http.ResponseWriter, r *http.Request) {
	serverID, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid server id")
		return
	}

	var req CreateDatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Name == "" {
		response.BadRequest(w, "name is required")
		return
	}

	db, err := h.svc.CreateDatabase(r.Context(), serverID, req)
	if err != nil {
		handleDBError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, db)
}

func (h *Handler) ListDatabases(w http.ResponseWriter, r *http.Request) {
	serverID, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid server id")
		return
	}

	dbs, err := h.svc.ListDatabases(r.Context(), serverID)
	if err != nil {
		response.InternalError(w, "failed to list databases")
		return
	}

	response.JSON(w, http.StatusOK, dbs)
}

func (h *Handler) DeleteDatabase(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	if err := h.svc.DeleteDatabase(r.Context(), id); err != nil {
		handleDBError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "database deleted"})
}

func (h *Handler) CreateDBUser(w http.ResponseWriter, r *http.Request) {
	serverID, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid server id")
		return
	}

	var req CreateDBUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		response.BadRequest(w, "username and password are required")
		return
	}

	user, err := h.svc.CreateDBUser(r.Context(), serverID, req)
	if err != nil {
		handleDBError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, user)
}

func (h *Handler) ListDBUsers(w http.ResponseWriter, r *http.Request) {
	serverID, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid server id")
		return
	}

	users, err := h.svc.ListDBUsers(r.Context(), serverID)
	if err != nil {
		response.InternalError(w, "failed to list users")
		return
	}

	response.JSON(w, http.StatusOK, users)
}

func (h *Handler) DeleteDBUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		response.BadRequest(w, "invalid id")
		return
	}

	if err := h.svc.DeleteDBUser(r.Context(), id); err != nil {
		handleDBError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

func parseID(r *http.Request, param string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, param), 10, 64)
}

func handleDBError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.NotFound(w, "not found")
	case errors.Is(err, ErrInvalidType):
		response.BadRequest(w, err.Error())
	default:
		response.InternalError(w, err.Error())
	}
}
