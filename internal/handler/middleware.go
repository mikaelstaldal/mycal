package handler

import (
	"log"
	"net/http"
)

func withMiddleware(h http.Handler) http.Handler {
	return recoveryMiddleware(h)
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware adds security headers to all responses.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self';"+
				" script-src 'self' 'sha256-q/j/gpKYBbsWntS1ygYOG/Yr7waDrXX8Y7UQWf44lL0=' https://esm.sh https://cdn.jsdelivr.net https://unpkg.com https://maps.googleapis.com;"+
				" style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://unpkg.com;"+
				" img-src 'self' data: https://*.tile.openstreetmap.org https://maps.googleapis.com https://maps.gstatic.com;"+
				" connect-src 'self' https://maps.googleapis.com;"+
				" font-src 'self' https://cdn.jsdelivr.net;"+
				" frame-src 'none';"+
				" object-src 'none'")
		next.ServeHTTP(w, r)
	})
}
