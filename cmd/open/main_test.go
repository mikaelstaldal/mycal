package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLDefaultPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name       string
		env        string
		flags      []string
		invalidURL bool
	}{
		{name: "local fallback"},
		{name: "environment", env: "https://example.com/mycal/"},
		{name: "invalid environment", env: "invalid", invalidURL: true},
		{name: "flag overrides environment", env: "invalid", flags: []string{"-url", "https://example.com/mycal/"}},
		{name: "empty flag overrides environment", env: "https://example.com/mycal/", flags: []string{"-url", ""}, invalidURL: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MYCAL_URL", tc.env)
			// Stop before browser launching; a valid URL reaches file opening.
			missing := filepath.Join(t.TempDir(), "missing.ics")
			err := run(append(tc.flags, missing))
			require.Error(t, err)
			if tc.invalidURL {
				assert.Contains(t, err.Error(), "-url must be")
			} else {
				assert.ErrorIs(t, err, os.ErrNotExist)
			}
		})
	}
}

func TestTargetURL(t *testing.T) {
	for _, raw := range []string{"https://example.com/mycal", "https://example.com/mycal/"} {
		got, err := targetURL(raw)
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/mycal/", got)
	}
	for _, raw := range []string{"", "file:///tmp/test", "http://example.com/", "https://user:password@example.com", "https://example.com/?token=secret", "https://example.com/#fragment"} {
		_, err := targetURL(raw)
		assert.Error(t, err, raw)
	}
	_, err := targetURL("http://127.0.0.1:8080/")
	require.NoError(t, err)
}

func TestBridgeEscapesFileContents(t *testing.T) {
	var output bytes.Buffer
	err := bridgePage.Execute(&output, bridgeData{
		"https://example.com/mycal/",
		"BEGIN:VCALENDAR\r\nSUMMARY:</textarea><script>alert(1)</script>&\r\nEND:VCALENDAR",
		"test-nonce",
	})
	require.NoError(t, err)
	assert.NotContains(t, output.String(), "<script>alert(1)</script>")
	assert.NotContains(t, output.String(), "</textarea>")
	assert.Contains(t, output.String(), `"https://example.com/mycal/"`)
}
