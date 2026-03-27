package middleware

import (
	"net/http"
	"strings"
)

func CORS(origins string) func(http.Handler) http.Handler {
	allowedOrigins := strings.Split(origins, ",")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				allowed := false
				for _, o := range allowedOrigins {
					if strings.TrimSpace(o) == origin || strings.TrimSpace(o) == "*" {
						allowed = true
						break
					}
				}
				if allowed || origin == "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					// Allow anyway if coming from same server (nginx proxy)
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
