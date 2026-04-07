package handler

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/mikaelstaldal/go-server-common/recovery"
)

func withMiddleware(h http.Handler) http.Handler {
	return recovery.Middleware(gzipMiddleware(apiCacheMiddleware(h)))
}

// apiCacheMiddleware prevents caching of dynamic API responses.
func apiCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer io.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.writer.Write(b)
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.DefaultCompression)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		defer gz.Close()
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, writer: gz}, r)
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
				" script-src 'self' 'sha256-l4YleaHsC1MWhnC491PTrqrnc9YJbIKzgYkX6jf35As=' https://maps.googleapis.com;"+
				" style-src 'self' 'unsafe-inline';"+
				" img-src 'self' data: https://*.tile.openstreetmap.org https://maps.googleapis.com https://maps.gstatic.com;"+
				" connect-src 'self' https://maps.googleapis.com;"+
				" font-src 'self';"+
				" frame-src 'none';"+
				" object-src 'none'")
		next.ServeHTTP(w, r)
	})
}
