package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveSiblingURLs(t *testing.T) {
	tests := []struct {
		name        string
		publicURL   string
		wantMymail  string
		wantMynotes string
	}{
		{"empty", "", "", ""},
		{"no path", "https://example.com", "", ""},
		{"root path only", "https://example.com/", "", ""},
		{"with path", "https://example.com/cal", "https://example.com/mymail", "https://example.com/mynotes"},
		{"query and fragment dropped", "https://example.com/cal?a=b#c", "https://example.com/mymail", "https://example.com/mynotes"},
		{"port preserved", "http://localhost:8089/cal", "http://localhost:8089/mymail", "http://localhost:8089/mynotes"},
		{"unparsable", "://nope", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMymail, deriveMymailURL(tt.publicURL))
			assert.Equal(t, tt.wantMynotes, deriveMynotesURL(tt.publicURL))
		})
	}
}

// The derived sibling URLs originate from the operator-supplied -public-url flag
// and are spliced into index.html inside an inline <script>. Nothing in them may
// be able to close that element or otherwise inject markup.
func TestServerConfigScriptCannotEscapeScriptElement(t *testing.T) {
	hostile := []string{
		`https://example.com/</script><img src=x onerror=alert(1)>`,
		`https://example.com/"><script>alert(1)</script>`,
		`https://ex<script>ample.com/cal`,
		`https://ex"ample.com/cal`,
		`https://example.com/cal?x=</script>#</script>`,
	}
	for _, publicURL := range hostile {
		t.Run(publicURL, func(t *testing.T) {
			script := serverConfig{
				MymailURL:  deriveMymailURL(publicURL),
				MynotesURL: deriveMynotesURL(publicURL),
			}.script()
			assert.NotContains(t, script, "<")
			assert.NotContains(t, script, ">")
			assert.True(t, strings.HasPrefix(script, "window.__serverConfig={"),
				"unexpected script shape: %s", script)
		})
	}
}

// An unconfigured deployment injects nothing at all; every set field shows up
// under the name web/ts/util/serverconfig.ts reads.
func TestServerConfigScript(t *testing.T) {
	assert.Empty(t, serverConfig{}.script())
	assert.Equal(t, `window.__serverConfig={"demo":true};`, serverConfig{Demo: true}.script())
	assert.Equal(t,
		`window.__serverConfig={"mymailUrl":"https://example.com/mymail","mynotesUrl":"https://example.com/mynotes"};`,
		serverConfig{
			MymailURL:  "https://example.com/mymail",
			MynotesURL: "https://example.com/mynotes",
		}.script())
}

func TestBuildIndexHTMLInjectsConfigScript(t *testing.T) {
	fsys := fstest.MapFS{
		"static/index.html": &fstest.MapFile{Data: []byte("<html><head><title>t</title></head><body></body></html>")},
	}

	unchanged, hash, err := buildIndexHTML(fsys, "/", serverConfig{})
	require.NoError(t, err)
	assert.NotContains(t, string(unchanged), "<script>")
	assert.Empty(t, hash)

	cfg := serverConfig{MymailURL: "https://example.com/mymail", MynotesURL: "https://example.com/mynotes"}
	injected, hash, err := buildIndexHTML(fsys, "/", cfg)
	require.NoError(t, err)
	assert.Contains(t, string(injected), "<script>"+cfg.script()+"</script>")
	assert.Contains(t, string(injected), "</head>")
	assert.Equal(t, inlineScriptCSPHash(cfg.script()), hash)
}

// The <base href> is only for a deployment served under a path — the embedded
// index.html has none, and every URL in it is relative.
func TestBuildIndexHTMLBaseHref(t *testing.T) {
	fsys := fstest.MapFS{
		"static/index.html": &fstest.MapFile{Data: []byte("<html><head><title>t</title></head><body></body></html>")},
	}

	root, _, err := buildIndexHTML(fsys, "/", serverConfig{})
	require.NoError(t, err)
	assert.NotContains(t, string(root), "<base")

	sub, _, err := buildIndexHTML(fsys, "/mycal/", serverConfig{Demo: true})
	require.NoError(t, err)
	assert.Contains(t, string(sub), `<base href="/mycal/">`)
}

// The base path goes into <base href> in the static demo bundle, so anything
// that could close the attribute, re-point the page at another host, or smuggle
// a second URL component must be refused rather than escaped.
func TestBasePathFromPublicURL(t *testing.T) {
	tests := []struct {
		name      string
		publicURL string
		want      string
	}{
		{"empty", "", "/"},
		{"no path", "https://example.com", "/"},
		{"root path", "https://example.com/", "/"},
		{"path gains trailing slash", "https://example.com/mycal", "/mycal/"},
		{"trailing slash kept", "https://example.com/mycal/", "/mycal/"},
		{"nested path", "https://user.github.io/mycal/demo", "/mycal/demo/"},
		{"port ignored", "http://localhost:8089/cal", "/cal/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := basePathFromPublicURL(tt.publicURL)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	hostile := []string{
		`https://example.com/"><script>alert(1)</script>`,
		`https://example.com/cal"`,
		`https://example.com//evil.example.net/`,
		`https://example.com/a b`,
		"://nope",
	}
	for _, publicURL := range hostile {
		t.Run("rejected: "+publicURL, func(t *testing.T) {
			_, err := basePathFromPublicURL(publicURL)
			assert.Error(t, err)
		})
	}
}

// A bundle is hosted by someone else's web server, so its policy has to survive
// in a <meta> element: identical to the header the demo server sends, minus the
// frame-ancestors directive browsers ignore there.
func TestMetaContentSecurityPolicy(t *testing.T) {
	meta := metaContentSecurityPolicy("'self'")
	assert.NotContains(t, meta, "frame-ancestors")
	assert.Equal(t, meta+"; frame-ancestors 'none'", contentSecurityPolicy("'self'"))
	assert.Contains(t, meta, "script-src 'self' ")
}

func TestInjectMetaCSP(t *testing.T) {
	html := []byte("<html><head><meta charset=\"UTF-8\">\n    <title>t</title></head><body></body></html>")
	got := string(injectMetaCSP(html, "default-src 'self'"))
	assert.Contains(t, got, `<meta http-equiv="Content-Security-Policy" content="default-src 'self'">`)
	// Before anything the page loads, and the charset declaration stays first.
	assert.Less(t, strings.Index(got, "Content-Security-Policy"), strings.Index(got, "<title>"))
	assert.Less(t, strings.Index(got, "charset"), strings.Index(got, "Content-Security-Policy"))
}

// The bundle is never merged into a directory that already holds something, so
// no stale file from an earlier version can survive into it.
func TestRequireEmptyDir(t *testing.T) {
	assert.NoError(t, requireEmptyDir(filepath.Join(t.TempDir(), "does-not-exist")))

	empty := t.TempDir()
	assert.NoError(t, requireEmptyDir(empty))

	populated := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(populated, "stale.html"), []byte("x"), 0o644))
	assert.Error(t, requireEmptyDir(populated))
}

// The bundle has to be self-contained: every embedded asset, a rewritten
// index.html carrying both the demo marker and the policy, and nothing pointing
// back at a server.
func TestWriteDemoBundle(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "site")
	require.NoError(t, writeDemoBundle(outDir, "https://user.github.io/mycal"))

	index, err := os.ReadFile(filepath.Join(outDir, "index.html"))
	require.NoError(t, err)
	assert.Contains(t, string(index), `<base href="/mycal/">`)
	assert.Contains(t, string(index), `window.__serverConfig={"demo":true};`)
	assert.Contains(t, string(index), `<meta http-equiv="Content-Security-Policy"`)
	assert.NotContains(t, string(index), "frame-ancestors")

	// The service worker is the bundle's whole backend; without it there is no API.
	assert.FileExists(t, filepath.Join(outDir, "demo-sw.js"))
	assert.FileExists(t, filepath.Join(outDir, "app.js"))
	assert.FileExists(t, filepath.Join(outDir, "app.css"))

	// Refuses to write into a directory that already holds the previous build.
	assert.Error(t, writeDemoBundle(outDir, "https://user.github.io/mycal"))
}
