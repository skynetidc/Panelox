package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Audit logs all write operations (POST, PUT, DELETE) to the audit_log table.
func Audit(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only audit write operations
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth endpoints (login/register/refresh)
			if strings.Contains(r.URL.Path, "/auth/") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip WebSocket endpoints (terminal, etc.) - they need http.Hijacker
			if strings.Contains(r.URL.Path, "/terminal/") || r.Header.Get("Upgrade") == "websocket" {
				next.ServeHTTP(w, r)
				return
			}

			// Capture response status
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			// Only log successful mutations
			if sw.status >= 200 && sw.status < 400 {
				go logAudit(pool, r, sw.status)
			}
		})
	}
}

func logAudit(pool *pgxpool.Pool, r *http.Request, status int) {
	claims := GetClaims(r.Context())
	var userID *int64
	if claims != nil {
		userID = &claims.UserID
	}

	// Extract resource from path: /api/v1/{resource}/...
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/"), "/")
	resource := ""
	if len(parts) > 0 {
		resource = parts[0]
	}

	action := r.Method + " " + r.URL.Path

	// Read a snippet of body for details (max 1KB)
	var details string
	if r.Body != nil {
		bodyBytes, _ := io.ReadAll(io.LimitReader(r.Body, 1024))
		if len(bodyBytes) > 0 {
			details = string(bodyBytes)
		}
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	pool.Exec(r.Context(),
		`INSERT INTO audit_log (user_id, action, resource, details, ip_address)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, action, resource, details, r.RemoteAddr)
}
