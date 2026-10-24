package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorDiagnosticsEscapeTerminalControls(t *testing.T) {
	var stderr bytes.Buffer
	printError(&stderr, errors.New("bad\x1b]52;c;payload\a\r\u009b\xff\t\ntext"))
	assert.Equal(t, "mycal-token: bad\\x1b]52;c;payload\\a\\r\\u009b\\xff\t\ntext\n", stderr.String())
}

func TestTokenHTTPErrorEscaping(t *testing.T) {
	t.Setenv("MYCAL_USER", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("bad\x1b]52;c;payload\a\r\u009b\xff!"))
	}))
	defer server.Close()
	for _, command := range [][]string{
		{"create", "-name", "test", "-lifetime", "1d", "-calendars", "0"},
		{"revoke", "test"},
	} {
		t.Run(command[0], func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(append([]string{"-url", server.URL}, command...), strings.NewReader(""), &stdout, &stderr)
			require.Error(t, err)
			assert.Equal(t, "HTTP 500: bad\\x1b]52;c;payload\\a\\r\\u009b\\xff!", err.Error())
			printError(&stderr, err)
			assert.Equal(t, "mycal-token: "+err.Error()+"\n", stderr.String())
			assert.Empty(t, stdout.String())
		})
	}
}

func TestDeviceLoginErrorEscaping(t *testing.T) {
	for _, phase := range []string{"device_authorization", "token"} {
		t.Run(phase, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if phase == "token" && strings.HasSuffix(r.URL.Path, "/device_authorization") {
					_ = json.NewEncoder(w).Encode(map[string]any{"device_code": "device", "user_code": "BCDF-GHJK", "verification_uri": "https://example.org/approve", "expires_in": 10, "interval": 1})
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "bad\x1b]52;c;payload\a\r\u009b"})
			}))
			defer server.Close()
			var stdout, stderr bytes.Buffer
			err := run([]string{"-url", server.URL, "login", "-name", "test"}, unreadableInput{}, &stdout, &stderr)
			require.Error(t, err)
			printError(&stderr, err)
			assert.Contains(t, stderr.String(), "mycal-token: device login: bad\\x1b]52;c;payload\\a\\r\\u009b\n")
			assert.NotContains(t, stderr.String(), "\x1b")
			assert.Empty(t, stdout.String())
		})
	}
}
