package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"brook/zlogger"
)

func (d *Dependencies) Recovery() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
				d.logger.Error(r.Context(), "panic recovered",
					zlogger.Field{Key: "panic", Value: fmt.Sprintf("%v", rec)},
					zlogger.Field{Key: "stack", Value: string(debug.Stack())},
				)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
