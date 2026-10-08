package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommands(t *testing.T) {
	for _, tc := range []struct {
		args        []string
		path, query string
	}{
		{[]string{"calendars", "list"}, "/calendars", ""},
		{[]string{"calendars", "ics", "-calendars", "0,2"}, "/events.ics", "calendar_id=0&calendar_id=2"},
		{[]string{"events", "get", "42"}, "/events/42", ""},
		{[]string{"events", "ics", "42_2026-10-01T10:00:00Z"}, "/events/42_2026-10-01T10:00:00Z/ics", ""},
		{[]string{"events", "search", "-q", "meeting", "-calendars", "0"}, "/events", "calendar_id=0&q=meeting"},
		{[]string{"events", "list", "-from", "2026-10-01T00:00:00Z", "-to", "2026-11-01T00:00:00Z"}, "/events", "from=2026-10-01T00%3A00%3A00Z&to=2026-11-01T00%3A00%3A00Z"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			method, path, q, err := parseCommand(tc.args)
			require.NoError(t, err)
			assert.Equal(t, http.MethodGet, method)
			assert.Equal(t, tc.path, path)
			assert.Equal(t, tc.query, q.Encode())
		})
	}
	for _, args := range [][]string{{}, {"events"}, {"events", "get", "0"}, {"events", "get", "-1"}, {"events", "get", "1_bad"}, {"events", "get", "1/extra"}, {"events", "list"}, {"events", "search"}, {"events", "search", "-q", strings.Repeat("x", 201)}, {"events", "search", "-q", "x", "-calendars", "0,0"}, {"events", "search", "-q", "x", "-calendars", "-1"}, {"events", "search", "-q", "x", "-from", "bad"}, {"events", "search", "-q", "x", "-from", "2026-10-01T00:00:00Z"}, {"events", "search", "-q", "x", "-to", "2026-11-01T00:00:00Z"}, {"calendars", "list", "extra"}, {"calendars", "ics", "-q", "x"}, {"events", "delete", "1"}} {
		_, _, _, err := parseCommand(args)
		assert.Error(t, err, args)
	}
}

func TestReadCLI(t *testing.T) {
	t.Setenv("MYCAL_TOKEN_FILE", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/prefix/api/v1/events", r.URL.Path)
		assert.Equal(t, "0", r.URL.Query().Get("calendar_id"))
		assert.Equal(t, "Bearer mycal_test", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`[{"title":"meeting"}]`))
	}))
	defer srv.Close()
	var out bytes.Buffer
	require.NoError(t, run([]string{"-url", srv.URL + "/prefix", "-token-stdin", "events", "search", "-q", "meeting", "-calendars", "0"}, strings.NewReader("mycal_test\n"), &out))
	assert.Equal(t, `[{"title":"meeting"}]`, out.String())
	file := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(file, []byte("mycal_test\n"), 0600))
	t.Setenv("MYCAL_URL", srv.URL+"/prefix")
	t.Setenv("MYCAL_TOKEN_FILE", file)
	out.Reset()
	require.NoError(t, run([]string{"events", "search", "-q", "meeting", "-calendars", "0"}, strings.NewReader(""), &out))
	require.Error(t, run([]string{"-token-file", file, "-token-stdin", "calendars", "list"}, strings.NewReader("token"), &out))
	require.Error(t, run([]string{"-token-stdin", "calendars", "list"}, strings.NewReader("two tokens"), &out))
	require.Error(t, run([]string{"-token-stdin", "calendars", "list"}, strings.NewReader(strings.Repeat("x", 4097)), &out))
}
