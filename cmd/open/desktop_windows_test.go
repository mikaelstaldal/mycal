package main

import (
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
