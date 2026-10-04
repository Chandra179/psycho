package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBrowserSecurityHeaders(t *testing.T) {
	h := BrowserSecurity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(400) }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	for _, directive := range []string{"script-src 'self'", "style-src 'self'", "style-src-attr 'unsafe-inline'", "connect-src 'self'", "form-action 'self'", "object-src 'none'", "frame-ancestors 'none'"} {
		if !strings.Contains(w.Header().Get("Content-Security-Policy"), directive) {
			t.Errorf("missing %s", directive)
		}
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff")
	}
}
