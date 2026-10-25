package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsQuote(t *testing.T) {
	for _, tc := range []struct{ input, expected string }{
		{`C:\Program Files\MyCal\mycal-open.exe`, `"C:\Program Files\MyCal\mycal-open.exe"`},
		{`C:\directory\`, `"C:\directory\\"`},
		{`a"b`, `"a\"b"`},
		{`a\"b`, `"a\\\"b"`},
		{"", `""`},
		{"https://example.com/cal/?x=%20&y=$value", `"https://example.com/cal/?x=%20&y=$value"`},
	} {
		assert.Equal(t, tc.expected, windowsQuote(tc.input), tc.input)
	}
}

func TestWindowsRegistrationPreservesExistingDefaults(t *testing.T) {
	commands := windowsRegistration(`C:\Users\Name With Spaces\mycal-open.exe`, "https://example.com/cal/")
	var foundCommand, foundOpenWith, foundCapabilities bool
	for _, args := range commands {
		require.True(t, strings.HasPrefix(args[1], `HKCU\`), args)
		assert.NotContains(t, strings.Join(args, " "), "UserChoice")
		if strings.HasSuffix(args[1], `\shell\open\command`) {
			foundCommand = true
			assert.Contains(t, args, `"C:\Users\Name With Spaces\mycal-open.exe" -url "https://example.com/cal/" "%1"`)
		}
		if args[1] == `HKCU\Software\Classes\.ics\OpenWithProgids` {
			foundOpenWith = true
			assert.Contains(t, args, "MyCal.iCalendar")
		}
		if args[1] == `HKCU\Software\MyCal\Capabilities\FileAssociations` {
			foundCapabilities = true
		}
	}
	assert.True(t, foundCommand)
	assert.True(t, foundOpenWith)
	assert.True(t, foundCapabilities)
}

func TestMacScriptEscapesURLAndHandlesFinderDocuments(t *testing.T) {
	assert.Equal(t, `"a\"b\\c\nd"`, appleScriptQuote("a\"b\\c\nd"))
	script := macOpenScript("https://example.com/mycal/")
	assert.Contains(t, script, "on open theFiles")
	assert.Contains(t, script, `path to resource "mycal-open"`)
	assert.Contains(t, script, `quoted form of helperPath`)
	assert.Contains(t, script, `quoted form of "https://example.com/mycal/"`)
	assert.Contains(t, script, `quoted form of filePath`)
	assert.Contains(t, script, `on error errorMessage`)
	assert.Contains(t, script, `display alert "Could not open iCalendar file"`)
	var types []map[string]any
	require.NoError(t, json.Unmarshal([]byte(macDocumentTypes), &types))
	require.Len(t, types, 1)
	assert.Equal(t, "Viewer", types[0]["CFBundleTypeRole"])
	assert.Equal(t, []any{"ics", "ical"}, types[0]["CFBundleTypeExtensions"])
}
