package handler

import (
	"net/http"

	"github.com/mikaelstaldal/go-server-common/httputil"
	"github.com/mikaelstaldal/go-server-common/recovery"
)

func withMiddleware(h http.Handler) http.Handler {
	return recovery.Middleware(httputil.Gzip(apiCacheMiddleware(h)))
}

// apiCacheMiddleware prevents caching of dynamic API responses.
func apiCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware adds security headers to all responses.
// importMapHash is the 'sha256-…' CSP token for the inline importmap in
// index.html (see web.ImportMapCSPHash); it is added to script-src so the
// importmap is allowed without 'unsafe-inline'. configScriptHash is the
// 'sha256-…' token for the injected server-config inline script (empty if
// no script was injected). Pass httpsMode=true when the app is served over
// HTTPS (directly or via TLS-terminating proxy) to also emit
// Strict-Transport-Security.
func SecurityHeadersMiddleware(importMapHash, configScriptHash string, httpsMode bool) func(http.Handler) http.Handler {
	scriptSrc := "'self' " + importMapHash
	if configScriptHash != "" {
		scriptSrc += " " + configScriptHash
	}
	csp := "default-src 'self';" +
		" script-src " + scriptSrc + " https://maps.googleapis.com;" +
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
