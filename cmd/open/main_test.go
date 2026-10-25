package main

import (
	"encoding/base64"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

func TestImportURLPreservesUTF8AndUsesOnlyFragment(t *testing.T) {
	content := "BEGIN:VCALENDAR\r\nSUMMARY:Möte 🗓️ </textarea><script>alert(1)</script>&\r\nEND:VCALENDAR"
	address, err := importURL("https://example.com/mycal/", []byte("\xef\xbb\xbf"+content))
	require.NoError(t, err)
	u, err := url.Parse(address)
	require.NoError(t, err)
	assert.Equal(t, "/mycal/", u.Path)
	assert.Empty(t, u.RawQuery)
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(u.Fragment, "import="))
	require.NoError(t, err)
	assert.Equal(t, content, string(decoded))
	assert.NotContains(t, address, "<script>")
}

func TestImportURLRejectsInvalidOrOversizedContent(t *testing.T) {
	for _, data := range []string{"", "https://example.com/event.ics", "BEGIN:VCALENDAR\xff", "BEGIN:VCALENDAR" + strings.Repeat("x", maxFileSize)} {
		_, err := importURL("https://example.com/", []byte(data))
		assert.Error(t, err)
	}
	_, err := importURL("https://example.com/"+strings.Repeat("x", maxBrowserURLLength), []byte("BEGIN:VCALENDAR\nEND:VCALENDAR"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file picker")
}

func TestImportURLBudgetIncludesServerURLAndBase64Expansion(t *testing.T) {
	target := "https://example.com/mycal/"
	size := (maxBrowserURLLength - len(target) - len("#import=")) * 3 / 4
	prefix := "BEGIN:VCALENDAR\n"
	data := []byte(prefix + strings.Repeat("x", size-len(prefix)))
	address, err := importURL(target, data)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(address), maxBrowserURLLength)
	_, err = importURL(target, append(data, 'x'))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file picker")
}

func TestOversizedFileDoesNotLaunchBrowser(t *testing.T) {
	t.Setenv("MYCAL_URL", "https://example.com/")
	file := filepath.Join(t.TempDir(), "large.ics")
	require.NoError(t, os.WriteFile(file, []byte("BEGIN:VCALENDAR"+strings.Repeat("x", maxFileSize)), 0600))
	err := run([]string{file})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file picker")
}
