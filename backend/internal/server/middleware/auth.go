package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/apikey"
	"github.com/jakub/panelox/internal/pkg/claims"
	"github.com/jakub/panelox/internal/pkg/response"
)

type contextKey string

const claimsKey contextKey = "claims"

type TokenValidator func(tokenStr string) (*claims.Claims, error)

// JWTAuthWithAPIKey validates JWT tokens AND API keys (pnx_xxx format).
func JWTAuthWithAPIKey(validate TokenValidator, pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenStr string

			// Try Authorization header first
			header := r.Header.Get("Authorization")
			if header != "" {
				parts := strings.SplitN(header, " ", 2)
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenStr = parts[1]
				}
			}

			// Fallback: query parameter ?token= (for WebSocket)
			if tokenStr == "" {
				tokenStr = r.URL.Query().Get("token")
			}

			if tokenStr == "" {
				response.Unauthorized(w, "missing authorization")
				return
			}

			var c *claims.Claims
			var err error

			// Check if it's an API key (starts with pnx_)
			if strings.HasPrefix(tokenStr, "pnx_") {
				c, err = apikey.ValidateAPIKey(r.Context(), pool, tokenStr)
			} else {
				// JWT token
				c, err = validate(tokenStr)
			}

			if err != nil {
				response.Unauthorized(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, c)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// JWTAuth validates only JWT tokens (kept for backward compatibility).
func JWTAuth(validate TokenValidator) func(http.Handler) http.Handler {
	return JWTAuthWithAPIKey(validate, nil)
}

func GetClaims(ctx context.Context) *claims.Claims {
	c, _ := ctx.Value(claimsKey).(*claims.Claims)
	return c
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := GetClaims(r.Context())
			if c == nil {
				response.Unauthorized(w, "not authenticated")
				return
			}

			for _, role := range roles {
				if c.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Forbidden(w, "insufficient permissions")
		})
	}
}
