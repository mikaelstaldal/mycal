package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

func TestAPIDiscovery(t *testing.T) {
	db, err := repository.OpenDB(filepath.Join(t.TempDir(), "calendar.db"), 5000)
	require.NoError(t, err)
	defer db.Close()
	document, err := os.ReadFile("openapi.yaml")
	require.NoError(t, err)
	var spec struct {
		OpenAPI string         `yaml:"openapi"`
		Paths   map[string]any `yaml:"paths"`
		Servers []struct {
			URL string `yaml:"url"`
		} `yaml:"servers"`
	}
	require.NoError(t, yaml.Unmarshal(document, &spec))
	assert.Equal(t, "3.0.3", spec.OpenAPI)
	require.Len(t, spec.Servers, 1)
	serverURL, err := url.Parse(spec.Servers[0].URL)
	require.NoError(t, err)
	assert.False(t, serverURL.IsAbs())
	assert.Empty(t, serverURL.Host)
	// OpenAPI paths already include /api/v1; the server URL must not double it.
	assert.Equal(t, "/", serverURL.Path)
	require.Contains(t, spec.Paths, "/api/v1/calendars")
	origin, err := url.Parse("http://127.0.0.1:8080/openapi.yaml")
	require.NoError(t, err)
	operationURL := origin.ResolveReference(serverURL)
	operationURL.Path += "api/v1/calendars"
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	require.NoError(t, err)
	authFile := filepath.Join(t.TempDir(), "htpasswd")
	require.NoError(t, os.WriteFile(authFile, append([]byte("user:"), hash...), 0600))
	for _, auth := range []bool{false, true} {
		for _, demo := range []bool{false, true} {
			t.Run(fmtName(auth, demo), func(t *testing.T) {
				opts := httpServerOptions{addr: "127.0.0.1", port: 8080, demo: demo, tokenStore: &token.Store{DB: db}}
				if auth {
					opts.basicAuthFile = authFile
					opts.basicAuthRealm = "test"
				}
				mux := http.NewServeMux()
				mux.HandleFunc("GET /api/v1/calendars", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
				h, err := buildHTTPHandler(mux, opts)
				require.NoError(t, err)
				request := func(method, path, authorization string) *httptest.ResponseRecorder {
					r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, nil)
					r.Header.Set("Authorization", authorization)
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					return w
				}
				w := request(http.MethodGet, operationURL.Path, "Bearer invalid")
				assert.Equal(t, http.StatusUnauthorized, w.Code)
				if demo {
					assert.Empty(t, w.Header().Get("Link"))
				} else {
					assert.Equal(t, `</openapi.yaml>; rel="service-desc"`, w.Header().Get("Link"))
				}
				for _, method := range []string{http.MethodGet, http.MethodHead} {
					for _, path := range []string{"/openapi.yaml", "/.well-known/api-catalog"} {
						// Even invalid bearer credentials cannot interfere with public discovery.
						for _, authorization := range []string{"", "Bearer invalid"} {
							w := request(method, path, authorization)
							if demo {
								assert.Equal(t, http.StatusNotFound, w.Code)
								continue
							}
							require.Equal(t, http.StatusOK, w.Code)
							assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"))
							if path == "/openapi.yaml" {
								assert.Equal(t, "application/yaml", w.Header().Get("Content-Type"))
								if method == http.MethodGet {
									assert.Equal(t, document, w.Body.Bytes())
								}
							} else {
								assert.Equal(t, `application/linkset+json; profile="https://www.rfc-editor.org/info/rfc9727"`, w.Header().Get("Content-Type"))
								assert.Equal(t, `</.well-known/api-catalog>; rel="api-catalog"`, w.Header().Get("Link"))
								if method == http.MethodGet {
									assert.JSONEq(t, `{"linkset":[{"anchor":"/api/v1/","service-desc":[{"href":"/openapi.yaml","type":"application/yaml"}]}]}`, w.Body.String())
								}
							}
							if method == http.MethodHead {
								assert.Empty(t, w.Body.String())
							}
						}
					}
				}
				for _, path := range []string{operationURL.Path, "/api/v1/missing"} {
					r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080"+path, nil)
					r.SetBasicAuth("user", "password")
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if path == operationURL.Path {
						assert.Equal(t, http.StatusNoContent, w.Code)
					} else {
						assert.Equal(t, http.StatusNotFound, w.Code)
					}
					if demo {
						assert.Empty(t, w.Header().Get("Link"))
					} else {
						assert.Equal(t, `</openapi.yaml>; rel="service-desc"`, w.Header().Get("Link"))
					}
				}
				if auth {
					w := request(http.MethodGet, operationURL.Path, "")
					assert.Equal(t, http.StatusUnauthorized, w.Code)
					if demo {
						assert.Empty(t, w.Header().Get("Link"))
					} else {
						assert.Equal(t, `</openapi.yaml>; rel="service-desc"`, w.Header().Get("Link"))
					}
				}
			})
		}
	}
}

func TestDemoDiscoveryRoutes(t *testing.T) {
	mux, opts, err := buildDemoRoutes(httpServerOptions{addr: "127.0.0.1", port: 8080})
	require.NoError(t, err)
	h, err := buildHTTPHandler(mux, opts)
	require.NoError(t, err)
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		for _, path := range []string{"/openapi.yaml", "/.well-known/api-catalog", "/api/v1/calendars"} {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(method, "http://127.0.0.1:8080"+path, nil))
			if path == "/api/v1/calendars" {
				assert.Equal(t, http.StatusServiceUnavailable, w.Code)
			} else {
				assert.Equal(t, http.StatusNotFound, w.Code)
			}
			assert.Empty(t, w.Header().Get("Link"))
		}
	}
}

func fmtName(auth, demo bool) string {
	if auth {
		if demo {
			return "auth/demo"
		}
		return "auth/real"
	}
	if demo {
		return "public/demo"
	}
	return "public/real"
}
