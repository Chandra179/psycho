package middleware

import "net/http"

// Middleware is the standard stdlib adapter type.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares in order: first argument = outermost wrapper.
// The server applies: Chain(handler, d.Recovery(), RequestID, Timeout(...)).
// Logger, Auth, and RateLimit are placeholders for future middlewares.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
