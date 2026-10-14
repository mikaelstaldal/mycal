package main

import (
	"bytes"
	_ "embed"
	"net/http"
	"strings"
	"time"
)

//go:embed openapi.yaml
var openAPIDocument []byte

const apiCatalog = `{"linkset":[{"anchor":"/api/v1/","service-desc":[{"href":"/openapi.yaml","type":"application/yaml"}]}]}`

// apiDiscovery serves public discovery before authentication, while retaining
// the deployment's Host, CSRF and security-header policy.
func apiDiscovery(next http.Handler, demo bool) http.Handler {
	mux := http.NewServeMux()
	for _, path := range []string{"/openapi.yaml", "/.well-known/api-catalog"} {
		mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
			if demo {
				http.NotFound(w, r)
				return
			}
			data := openAPIDocument
			w.Header().Set("Content-Type", "application/yaml")
			if r.URL.Path == "/.well-known/api-catalog" {
				data = []byte(apiCatalog)
				w.Header().Set("Content-Type", `application/linkset+json; profile="https://www.rfc-editor.org/info/rfc9727"`)
				w.Header().Set("Link", `</.well-known/api-catalog>; rel="api-catalog"`)
			}
			http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(data))
		})
	}
	mux.Handle("/", next)
	return mux
}

// Wrap the full deployment stack so errors from auth, CSRF and Host validation
// carry the same API description link as successful API responses.
func apiDescriptionLink(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/") || r.URL.Path == "/api/v1" {
			w.Header().Add("Link", `</openapi.yaml>; rel="service-desc"`)
		}
		next.ServeHTTP(w, r)
	})
}
