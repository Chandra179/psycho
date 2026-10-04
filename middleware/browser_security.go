package middleware

import "net/http"

// BrowserSecurity permits local assets and the existing numeric bar-width
// attributes. HTMX never evaluates expressions or executes swapped scripts.
func BrowserSecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; connect-src 'self'; form-action 'self'; style-src 'self'; style-src-attr 'unsafe-inline'; object-src 'none'; frame-ancestors 'none'; base-uri 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
