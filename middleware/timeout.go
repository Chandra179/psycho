package middleware

import (
	"net/http"
	"time"
)

type TimeoutConfig struct {
	Duration time.Duration
}

// Timeout bounds each request. http.TimeoutHandler (rather than a bare
// context timeout) so an overdue request gets an explicit 503 and late
// handler writes are suppressed instead of hanging until the server's
// write timeout. It also installs the deadline on the request context,
// which the pipeline checks between stages.
func Timeout(cfg TimeoutConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, cfg.Duration, "request timed out\n")
	}
}
