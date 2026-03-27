package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/claims"
	"github.com/jakub/panelox/internal/pkg/response"
)

const keyPrefix = "pnx_"

type APIKey struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	Name        string     `json:"name"`
	KeyPrefix   string     `json:"key_prefix"`
	Permissions string     `json:"permissions"` // full, readonly
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type APIKeyWithSecret struct {
	APIKey
	Key string `json:"key"` // Only returned on creation
}

type CreateRequest struct {
	Name        string `json:"name"`
	Permissions string `json:"permissions"` // full, readonly
	ExpiresIn   int    `json:"expires_in"`  // days, 0 = never
}

type Handler struct {
	pool      *pgxpool.Pool
	getClaims func(r *http.Request) *claims.Claims
}

func NewHandler(pool *pgxpool.Pool, getClaims func(r *http.Request) *claims.Claims) *Handler {
	return &Handler{pool: pool, getClaims: getClaims}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/api-keys", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Delete("/{id}", h.Delete)
	})
}

// generateKey creates a random API key like: pnx_a1b2c3d4e5f6...
func generateKey() (string, string) {
	b := make([]byte, 32)
	rand.Read(b)
	key := keyPrefix + hex.EncodeToString(b)
	hash := hashKey(key)
	return key, hash
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, user_id, name, key_prefix, permissions, expires_at, last_used, created_at
		 FROM api_keys WHERE user_id = $1 ORDER BY id DESC`, c.UserID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.Permissions, &k.ExpiresAt, &k.LastUsed, &k.CreatedAt)
		keys = append(keys, k)
	}
	response.JSON(w, http.StatusOK, keys)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var req CreateRequest
	json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" {
		response.BadRequest(w, "name is required")
		return
	}
	if req.Permissions == "" {
		req.Permissions = "full"
	}

	key, hash := generateKey()
	prefix := key[:len(keyPrefix)+8] + "..."

	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		t := time.Now().AddDate(0, 0, req.ExpiresIn)
		expiresAt = &t
	}

	var k APIKey
	err := h.pool.QueryRow(r.Context(),
		`INSERT INTO api_keys (user_id, name, key_hash, key_prefix, permissions, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, name, key_prefix, permissions, expires_at, last_used, created_at`,
		c.UserID, req.Name, hash, prefix, req.Permissions, expiresAt,
	).Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.Permissions, &k.ExpiresAt, &k.LastUsed, &k.CreatedAt)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	// Return full key only on creation (never stored, only hash)
	result := APIKeyWithSecret{APIKey: k, Key: key}
	response.JSON(w, http.StatusCreated, result)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	h.pool.Exec(r.Context(), "DELETE FROM api_keys WHERE id = $1 AND user_id = $2", id, c.UserID)
	response.JSON(w, http.StatusOK, map[string]string{"message": "API key deleted"})
}

// ValidateAPIKey checks if an API key is valid and returns the user claims.
// Called from auth middleware as fallback when JWT fails.
func ValidateAPIKey(ctx context.Context, pool *pgxpool.Pool, key string) (*claims.Claims, error) {
	if !strings.HasPrefix(key, keyPrefix) {
		return nil, fmt.Errorf("invalid key format")
	}

	hash := hashKey(key)

	var userID int64
	var permissions string
	var expiresAt *time.Time
	err := pool.QueryRow(ctx,
		`SELECT ak.user_id, ak.permissions, ak.expires_at
		 FROM api_keys ak WHERE ak.key_hash = $1`, hash,
	).Scan(&userID, &permissions, &expiresAt)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("invalid API key")
	}
	if err != nil {
		return nil, err
	}

	// Check expiration
	if expiresAt != nil && time.Now().After(*expiresAt) {
		return nil, fmt.Errorf("API key expired")
	}

	// Update last_used
	pool.Exec(ctx, "UPDATE api_keys SET last_used = NOW() WHERE key_hash = $1", hash)

	// Get user info
	var email, role string
	pool.QueryRow(ctx, "SELECT email, role FROM users WHERE id = $1", userID).Scan(&email, &role)

	// For readonly keys, override role
	if permissions == "readonly" {
		role = "readonly"
	}

	return &claims.Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
	}, nil
}
