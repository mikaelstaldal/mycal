package main

import (
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
			script := serverConfigScript(deriveMymailURL(publicURL), deriveMynotesURL(publicURL))
			assert.NotContains(t, script, "<")
			assert.NotContains(t, script, ">")
			assert.True(t, strings.HasPrefix(script, "window.__serverConfig={mymailUrl:"),
				"unexpected script shape: %s", script)
			assert.Contains(t, script, ",mynotesUrl:")
		})
	}
}

func TestBuildIndexHTMLInjectsConfigScript(t *testing.T) {
	fsys := fstest.MapFS{
		"static/index.html": &fstest.MapFile{Data: []byte("<html><head><title>t</title></head><body></body></html>")},
	}

	unchanged, err := buildIndexHTML(fsys, "")
	require.NoError(t, err)
	assert.NotContains(t, string(unchanged), "<script>")

	script := serverConfigScript("https://example.com/mymail", "https://example.com/mynotes")
	injected, err := buildIndexHTML(fsys, script)
	require.NoError(t, err)
	assert.Contains(t, string(injected), "<script>"+script+"</script>")
	assert.Contains(t, string(injected), "</head>")
}
