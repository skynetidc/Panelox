package auth

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

func RegisterPublicRoutes(r chi.Router, h *Handler) {
	r.Post("/auth/login", h.Login)
	r.Post("/auth/register", h.Register)
	r.Post("/auth/refresh", h.Refresh)
}

func RegisterRoutes(r chi.Router, h *Handler, requireAdmin func(http.Handler) http.Handler) {
	r.Post("/auth/logout", h.Logout)
	r.Get("/auth/me", h.Me)

	r.Route("/users", func(r chi.Router) {
		r.Use(requireAdmin)
		r.Get("/", h.ListUsers)
		r.Get("/{id}", h.GetUser)
		r.Put("/{id}", h.UpdateUser)
		r.Delete("/{id}", h.DeleteUser)
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		response.BadRequest(w, "email and password are required")
		return
	}

	user, tokens, err := h.svc.Login(r.Context(), req, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Unauthorized(w, "invalid email or password")
			return
		}
		response.InternalError(w, "login failed")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"user":   user,
		"tokens": tokens,
	})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		response.BadRequest(w, "email, password and name are required")
		return
	}

	user, err := h.svc.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			response.BadRequest(w, "email already registered")
			return
		}
		response.InternalError(w, "registration failed")
		return
	}

	response.JSON(w, http.StatusCreated, user)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	tokens, err := h.svc.RefreshToken(r.Context(), req.RefreshToken, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		if errors.Is(err, ErrSessionExpired) {
			response.Unauthorized(w, "session expired")
			return
		}
		response.InternalError(w, "refresh failed")
		return
	}

	response.JSON(w, http.StatusOK, tokens)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		response.InternalError(w, "logout failed")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	user, err := h.svc.GetUser(r.Context(), c.UserID)
	if err != nil {
		response.InternalError(w, "failed to get user")
		return
	}

	response.JSON(w, http.StatusOK, user)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	users, total, err := h.svc.ListUsers(r.Context(), page, size)
	if err != nil {
		response.InternalError(w, "failed to list users")
		return
	}

	response.JSONList(w, users, page, size, total)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid user id")
		return
	}

	user, err := h.svc.GetUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.NotFound(w, "user not found")
			return
		}
		response.InternalError(w, "failed to get user")
		return
	}

	response.JSON(w, http.StatusOK, user)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid user id")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.svc.UpdateUser(r.Context(), id, req); err != nil {
		response.InternalError(w, "failed to update user")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "user updated"})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid user id")
		return
	}

	if err := h.svc.DeleteUser(r.Context(), id); err != nil {
		response.InternalError(w, "failed to delete user")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
