package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestFlexibleDates(t *testing.T) {
	location, err := time.LoadLocation("Europe/Stockholm")
	require.NoError(t, err)
	original := time.Local
	time.Local = location
	t.Cleanup(func() { time.Local = original })
	for _, tc := range []struct{ input, want string }{
		{"2026-10-08", "2026-10-08T00:00:00+02:00"},
		{"2026-12-08", "2026-12-08T00:00:00+01:00"},
		{"2026-10-08T09:30", "2026-10-08T09:30:00+02:00"},
		{"2026-10-08T09:30:12.123", "2026-10-08T09:30:12.123+02:00"},
		{"2026-10-08 09:30:12", "2026-10-08T09:30:12+02:00"},
		{"2026-10-08T09:30:12Z", "2026-10-08T09:30:12Z"},
		{"2026-10-08T09:30:12-05:00", "2026-10-08T09:30:12-05:00"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			stamp, err := parseDateTime(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.want, stamp.Format(time.RFC3339Nano))
		})
	}
	for _, input := range []string{"2026-02-30", "2026-10-08T24:00", "09:30", "2026-10-08junk", "2026-03-29T02:30"} {
		_, err := parseDateTime(input)
		assert.Error(t, err)
	}
	for _, command := range []string{"list", "search"} {
		args := []string{"events", command, "-from", "2026-10-25", "-to", "2026-10-26"}
		if command == "search" {
			args = append(args, "-q", "meeting")
		}
		_, _, q, err := parseCommand(args)
		require.NoError(t, err)
		assert.Equal(t, "2026-10-25T00:00:00+02:00", q.Get("from"))
		assert.Equal(t, "2026-10-26T00:00:00+01:00", q.Get("to"))
	}
	for _, command := range []string{"get", "ics"} {
		for _, id := range []string{"42_2026-10-08T09:30:00+02:00", "42_2026-10-08T07:30:00Z", "42_2026-10-08T09:30:00.000+02:00"} {
			args := []string{"events", command, id}
			_, path, _, err := parseCommand(args)
			require.NoError(t, err)
			expected := "/events/" + id
			if command == "ics" {
				expected += "/ics"
			}
			assert.Equal(t, expected, path)
			assert.Equal(t, id, args[2])
		}
		_, _, _, err := parseCommand([]string{"events", command, "42_2026-10-08T09:30"})
		assert.Error(t, err, "event IDs require a zoned recurrence timestamp")
	}
	_, _, _, err = parseCommand([]string{"events", "list", "-from", "2026-10-09", "-to", "2026-10-08"})
	assert.Error(t, err)
	_, _, _, err = parseCommand([]string{"events", "list", "-from", "2026-10-08T03:00:00Z", "-to", "2026-10-08T04:00"})
	assert.Error(t, err, "compare instants after resolving the local offset")
}

func TestHelp(t *testing.T) {
	t.Setenv("MYCAL_URL", "invalid")
	t.Setenv("MYCAL_TOKEN_FILE", "/does/not/exist")
	for _, args := range [][]string{{"-help"}, {"--help"}, {"-h"}, {"events", "-help"}, {"events", "list", "-help"}, {"events", "search", "-help"}, {"events", "get", "-help"}, {"events", "ics", "-help"}, {"calendars", "list", "-help"}, {"calendars", "ics", "-help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var out bytes.Buffer
			require.NoError(t, run(args, strings.NewReader(""), &out))
			assert.Equal(t, usage, out.String())
		})
	}
	for _, args := range [][]string{{"bogus", "-help"}, {"event", "-h"}} {
		var out bytes.Buffer
		require.ErrorContains(t, run(args, strings.NewReader(""), &out), "invalid command")
		assert.Empty(t, out.String())
	}

}
