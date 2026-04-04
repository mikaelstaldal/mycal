package handler

import (
	"net/http"

	"github.com/mikaelstaldal/go-server-common/recovery"
)

func withMiddleware(h http.Handler) http.Handler {
	return recovery.Middleware(h)
}

// SecurityHeadersMiddleware adds security headers to all responses.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self';"+
				" script-src 'self' 'sha256-W7jEZMnlsPnNaTGwLEPNi7ZrjFTfDSyONcSF5PDuAcE=' https://maps.googleapis.com;"+
				" style-src 'self' 'unsafe-inline';"+
				" img-src 'self' data: https://*.tile.openstreetmap.org https://maps.googleapis.com https://maps.gstatic.com;"+
				" connect-src 'self' https://maps.googleapis.com;"+
				" font-src 'self';"+
				" frame-src 'none';"+
				" object-src 'none'")
		next.ServeHTTP(w, r)
	})
}
