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
