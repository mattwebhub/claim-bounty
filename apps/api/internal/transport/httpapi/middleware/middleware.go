package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler

// Chain applies middleware in declaration order; the first item is outermost.
func Chain(handler http.Handler, middleware ...Middleware) http.Handler {
	for index := len(middleware) - 1; index >= 0; index-- {
		handler = middleware[index](handler)
	}
	return handler
}
