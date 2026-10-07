package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestHTTPHostAndCSRFPolicy(t *testing.T) {
	for _, deployment := range []struct {
		name, public, addr string
		port               int
		origins            []string
	}{
		{"concrete bind", "http://public.example:80/mycal", "192.0.2.1", 8089, []string{"http://public.example", "http://192.0.2.1:8089"}},
		{"IPv6 bind", "", "::1", 8089, []string{"http://[::1]:8089"}},
		{"local", "", "127.0.0.1", 8089, []string{"http://localhost:8089", "http://127.0.0.1:8089", "http://[::1]:8089"}},
		{"proxy path and custom port", "https://PUBLIC.example:8443/mycal", "0.0.0.0", 8089, []string{"https://public.example:8443", "http://localhost:8089"}},
		{"proxy default port", "https://public.example:443/mycal", "::", 8089, []string{"https://public.example", "http://[::1]:8089"}},
	} {
		t.Run(deployment.name, func(t *testing.T) {
			called := false
			h, err := buildHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true; w.WriteHeader(http.StatusNoContent) }), httpServerOptions{addr: deployment.addr, port: deployment.port, publicURL: deployment.public})
			require.NoError(t, err)
			for _, path := range []string{"/", "/api/v1/events", "/calendar.ics", "/api/v1/events.ics", "/demo-sw.js"} {
				for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPost} {
					for _, host := range []string{"attacker.example:8089", "localhost:8090", "localhost:", "public.example:8089"} {
						r := httptest.NewRequest(method, "http://localhost:8089"+path, nil)
						r.Host = host
						r.Header.Set("Origin", "https://attacker.example")
						r.Header.Set("X-Forwarded-Host", "public.example:8443")
						r.Header.Set("Forwarded", "host=public.example:8443;proto=https")
						w := httptest.NewRecorder()
						called = false
						h.ServeHTTP(w, r)
						assert.Equal(t, http.StatusMisdirectedRequest, w.Code, "%s %s %s", method, path, host)
						assert.False(t, called)
					}
				}
			}
			for _, origin := range deployment.origins {
				for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPost} {
					r := httptest.NewRequest(method, origin+"/api/v1/events", nil)
					r.Header.Set("Origin", origin)
					w := httptest.NewRecorder()
					called = false
					h.ServeHTTP(w, r)
					assert.Equal(t, http.StatusNoContent, w.Code, origin)
					assert.True(t, called)
					r.Header.Set("Origin", "https://attacker.example")
					w = httptest.NewRecorder()
					called = false
					h.ServeHTTP(w, r)
					if method == http.MethodPost {
						assert.Equal(t, http.StatusForbidden, w.Code)
						assert.False(t, called)
					} else {
						assert.Equal(t, http.StatusNoContent, w.Code)
						assert.True(t, called)
					}
				}
			}
		})
	}
}

func TestHTTPHostValidationBeforeAuthentication(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	require.NoError(t, err)
	file := filepath.Join(t.TempDir(), "htpasswd")
	require.NoError(t, os.WriteFile(file, append([]byte("user:"), hash...), 0600))
	h, err := buildHTTPHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("route reached") }), httpServerOptions{addr: "127.0.0.1", port: 8089, basicAuthFile: file, basicAuthRealm: "mycal"})
	require.NoError(t, err)
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPost} {
		for _, host := range []string{"attacker.example:8089", "localhost:8089"} {
			r := httptest.NewRequest(method, "http://"+host+"/", nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if host == "localhost:8089" {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			} else {
				assert.Equal(t, http.StatusMisdirectedRequest, w.Code)
				assert.Empty(t, w.Header().Get("WWW-Authenticate"))
			}
		}
	}
}

func TestHTTPHostConfigurationErrors(t *testing.T) {
	for _, opts := range []httpServerOptions{
		{addr: "0.0.0.0", port: 8089}, {addr: "::", port: 8089}, {addr: "", port: 8089},
		{addr: "127.0.0.1", port: 8089, publicURL: "ftp://example.com/mycal"},
		{addr: "127.0.0.1", port: 8089, publicURL: "https://example.com:0/mycal"},
	} {
		h, err := buildHTTPHandler(http.NotFoundHandler(), opts)
		require.Error(t, err)
		assert.Nil(t, h)
	}
}
