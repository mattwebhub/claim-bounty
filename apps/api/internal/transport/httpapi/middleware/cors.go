package middleware

import (
	"net/http"

	"github.com/mattwebhub/micro1-template/apps/api/internal/transport/httpapi/response"
)

func CORS(allowedOrigins []string) Middleware {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}
	_, wildcard := allowed["*"]
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Add("Vary", "Origin")
			_, exact := allowed[origin]
			if !wildcard && !exact {
				_ = response.WriteError(w, http.StatusForbidden, "cors_origin_denied", "request origin is not allowed", RequestIDFromContext(r.Context()), nil)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, If-Match, Idempotency-Key, X-Request-ID")
			w.Header().Set("Access-Control-Expose-Headers", "ETag, Location, X-Request-ID")
			if r.Method == http.MethodOptions {
				w.Header().Add("Vary", "Access-Control-Request-Method")
				w.Header().Add("Vary", "Access-Control-Request-Headers")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
