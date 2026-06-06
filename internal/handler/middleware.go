package handler

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/mikaelstaldal/go-server-common/httputil"
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
		defer func() { _ = gz.Close() }()
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, writer: gz}, r)
	})
}

// SecurityHeadersMiddleware adds security headers to all responses.
// importMapHash is the 'sha256-…' CSP token for the inline importmap in
// index.html (see web.ImportMapCSPHash); it is added to script-src so the
// importmap is allowed without 'unsafe-inline'. Pass httpsMode=true when the
// app is served over HTTPS (directly or via TLS-terminating proxy) to also emit
// Strict-Transport-Security.
func SecurityHeadersMiddleware(importMapHash string, httpsMode bool) func(http.Handler) http.Handler {
	csp := "default-src 'self';" +
		" script-src 'self' " + importMapHash + " https://maps.googleapis.com;" +
		" style-src-elem 'self'; style-src-attr 'unsafe-inline';" +
		" img-src 'self' data: https://*.tile.openstreetmap.org https://maps.googleapis.com https://maps.gstatic.com;" +
		" connect-src 'self' https://maps.googleapis.com https://*.tile.openstreetmap.org;" +
		" font-src 'self';" +
		" frame-src 'none';" +
		" object-src 'none';" +
		" frame-ancestors 'none'"
	hsts := ""
	if httpsMode {
		hsts = "max-age=31536000; includeSubDomains"
	}
	return httputil.SecurityHeaders(httputil.SecurityHeadersOptions{
		CSP:            csp,
		ReferrerPolicy: "strict-origin-when-cross-origin",
		HSTS:           hsts,
	})
}
