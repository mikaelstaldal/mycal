package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
