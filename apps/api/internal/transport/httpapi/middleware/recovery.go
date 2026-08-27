package middleware

import (
	"log/slog"
	"net/http"

	"github.com/mattwebhub/micro1-template/apps/api/internal/transport/httpapi/response"
)

func Recovery(logger *slog.Logger) Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					requestID := RequestIDFromContext(r.Context())
					logger.ErrorContext(r.Context(), "panic recovered",
						"request_id", requestID,
						"method", r.Method,
						"route", r.Pattern,
						"panic_type", slog.AnyValue(recovered).Kind().String(),
					)
					_ = response.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred", requestID, nil)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
