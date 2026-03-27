package license

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
)

// EnforceLimits returns a middleware that blocks resource creation when limits are exceeded.
// It checks the HTTP method + path to determine which resource is being created.
func EnforceLimits(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check on create operations
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			path := r.URL.Path

			var resource string
			switch {
			case strings.HasSuffix(path, "/websites") || strings.HasSuffix(path, "/websites/"):
				resource = "websites"
			case strings.HasSuffix(path, "/nodes") || strings.HasSuffix(path, "/nodes/"):
				resource = "nodes"
			case strings.HasSuffix(path, "/apps/install"):
				resource = "apps"
			case strings.Contains(path, "/databases") && !strings.Contains(path, "/databases/"):
				// POST to /websites/{id}/databases or /databases/{id}/databases
				resource = "databases"
			}

			if resource == "" {
				next.ServeHTTP(w, r)
				return
			}

			if !CheckResourceLimit(r.Context(), pool, resource) {
				tier := GetCurrentTier(r.Context(), pool)
				var msg string
				switch resource {
				case "websites":
					msg = "Website limit reached"
				case "nodes":
					msg = "Server node limit reached"
				case "databases":
					msg = "Database limit reached"
				case "apps":
					msg = "App install limit reached"
				}
				if tier == TierFree {
					msg += ". Upgrade to Pro for unlimited resources."
				}
				response.JSON(w, http.StatusForbidden, map[string]interface{}{
					"error": map[string]interface{}{
						"message":  msg,
						"code":     "LICENSE_LIMIT",
						"resource": resource,
						"tier":     tier,
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
