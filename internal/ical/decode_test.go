package ical

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeBasicEvent(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"VERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20250315T100000Z\r\n" +
		"DTEND:20250315T110000Z\r\n" +
		"SUMMARY:Team Meeting\r\n" +
		"DESCRIPTION:Weekly sync\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	e := events[0]
	assert.Equal(t, "Team Meeting", e.Title)
	assert.Equal(t, "Weekly sync", e.Description)
	assert.Equal(t, "2025-03-15T10:00:00Z", e.StartTime)
	assert.Equal(t, "2025-03-15T11:00:00Z", e.EndTime)
}

func TestDecodeMultipleEvents(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20250315T100000Z\r\n" +
		"DTEND:20250315T110000Z\r\n" +
		"SUMMARY:Event One\r\n" +
		"END:VEVENT\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20250316T140000Z\r\n" +
		"DTEND:20250316T150000Z\r\n" +
		"SUMMARY:Event Two\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, "Event One", events[0].Title)
	assert.Equal(t, "Event Two", events[1].Title)
}

func TestDecodeLineUnfolding(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20250315T100000Z\r\n" +
		"DTEND:20250315T110000Z\r\n" +
		"SUMMARY:This is a very long \r\n" +
		" summary that spans \r\n" +
		" multiple lines\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	want := "This is a very long summary that spans multiple lines"
	assert.Equal(t, want, events[0].Title)
}

func TestDecodeTZID(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART;TZID=Europe/Stockholm:20250315T100000\r\n" +
		"DTEND;TZID=Europe/Stockholm:20250315T110000\r\n" +
		"SUMMARY:Stockholm Meeting\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	// Stockholm is CET (UTC+1) in March
	assert.Equal(t, "2025-03-15T09:00:00Z", events[0].StartTime)
}

func TestDecodeAllDay(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART;VALUE=DATE:20250315\r\n" +
		"DTEND;VALUE=DATE:20250316\r\n" +
		"SUMMARY:All Day Event\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "2025-03-15T00:00:00Z", events[0].StartTime)
	assert.Equal(t, "2025-03-16T00:00:00Z", events[0].EndTime)
}

func TestDecodeTextUnescaping(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20250315T100000Z\r\n" +
		"DTEND:20250315T110000Z\r\n" +
		"SUMMARY:Hello\\, World\r\n" +
		"DESCRIPTION:Line one\\nLine two\\;semicolon\\\\backslash\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "Hello, World", events[0].Title)
	wantDesc := "Line one\nLine two;semicolon\\backslash"
	assert.Equal(t, wantDesc, events[0].Description)
}

func TestDecodeSkipsMalformedEvents(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"SUMMARY:No times\r\n" +
		"END:VEVENT\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20250315T100000Z\r\n" +
		"SUMMARY:No end time\r\n" +
		"END:VEVENT\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20250315T100000Z\r\n" +
		"DTEND:20250315T110000Z\r\n" +
		"SUMMARY:Valid Event\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "Valid Event", events[0].Title)
}

func TestDecodeRecurrenceID(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20260302T100000Z\r\n" +
		"DTEND:20260302T110000Z\r\n" +
		"RRULE:FREQ=WEEKLY\r\n" +
		"SUMMARY:Weekly Meeting\r\n" +
		"END:VEVENT\r\n" +
		"BEGIN:VEVENT\r\n" +
		"RECURRENCE-ID:20260309T100000Z\r\n" +
		"DTSTART:20260309T140000Z\r\n" +
		"DTEND:20260309T150000Z\r\n" +
		"SUMMARY:Weekly Meeting (moved)\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 2)
	// First event is the parent
	assert.Equal(t, "Weekly Meeting", events[0].Title)
	assert.Equal(t, "WEEKLY", events[0].RecurrenceFreq)
	assert.Empty(t, events[0].RecurrenceOriginalStart, "parent should not have RecurrenceOriginalStart")
	// Second event is the override
	assert.Equal(t, "Weekly Meeting (moved)", events[1].Title)
	assert.Equal(t, "2026-03-09T10:00:00Z", events[1].RecurrenceOriginalStart)
}

func TestDecodePathStyleTZIDWithIANA(t *testing.T) {
	ics := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VTIMEZONE\r\n" +
		"TZID:/citadel.org/20250101_1/Europe/Stockholm\r\n" +
		"BEGIN:STANDARD\r\n" +
		"DTSTART:19701025T030000\r\n" +
		"TZOFFSETTO:+0100\r\n" +
		"TZOFFSETFROM:+0200\r\n" +
		"END:STANDARD\r\n" +
		"BEGIN:DAYLIGHT\r\n" +
		"DTSTART:19700329T020000\r\n" +
		"TZOFFSETTO:+0200\r\n" +
		"TZOFFSETFROM:+0100\r\n" +
		"END:DAYLIGHT\r\n" +
		"END:VTIMEZONE\r\n" +
		"BEGIN:VEVENT\r\n" +
		"SUMMARY:Test Event\r\n" +
		"DTSTART;TZID=/citadel.org/20250101_1/Europe/Stockholm:20250615T100000\r\n" +
		"DTEND;TZID=/citadel.org/20250101_1/Europe/Stockholm:20250615T110000\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(ics))
	require.NoError(t, err)
	require.Len(t, events, 1)
	// June 15 is CEST (UTC+2): 10:00 local = 08:00 UTC
	assert.Equal(t, "2025-06-15T08:00:00Z", events[0].StartTime)
	assert.Equal(t, "2025-06-15T09:00:00Z", events[0].EndTime)
}

func TestDecodeNonIANATZIDWithOffset(t *testing.T) {
	ics := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VTIMEZONE\r\n" +
		"TZID:CustomTZ-XYZ\r\n" +
		"BEGIN:STANDARD\r\n" +
		"DTSTART:19701025T030000\r\n" +
		"TZOFFSETTO:+0530\r\n" +
		"TZOFFSETFROM:+0530\r\n" +
		"END:STANDARD\r\n" +
		"END:VTIMEZONE\r\n" +
		"BEGIN:VEVENT\r\n" +
		"SUMMARY:Offset Event\r\n" +
		"DTSTART;TZID=CustomTZ-XYZ:20250115T140000\r\n" +
		"DTEND;TZID=CustomTZ-XYZ:20250115T150000\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(ics))
	require.NoError(t, err)
	require.Len(t, events, 1)
	// +0530: 14:00 local = 08:30 UTC
	assert.Equal(t, "2025-01-15T08:30:00Z", events[0].StartTime)
	assert.Equal(t, "2025-01-15T09:30:00Z", events[0].EndTime)
}

func TestDecodeStandardIANATZIDWithVTimezone(t *testing.T) {
	ics := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VTIMEZONE\r\n" +
		"TZID:America/New_York\r\n" +
		"BEGIN:STANDARD\r\n" +
		"DTSTART:19701101T020000\r\n" +
		"TZOFFSETTO:-0500\r\n" +
		"TZOFFSETFROM:-0400\r\n" +
		"END:STANDARD\r\n" +
		"END:VTIMEZONE\r\n" +
		"BEGIN:VEVENT\r\n" +
		"SUMMARY:NY Event\r\n" +
		"DTSTART;TZID=America/New_York:20250115T090000\r\n" +
		"DTEND;TZID=America/New_York:20250115T100000\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(ics))
	require.NoError(t, err)
	require.Len(t, events, 1)
	// January EST (UTC-5): 09:00 local = 14:00 UTC
	assert.Equal(t, "2025-01-15T14:00:00Z", events[0].StartTime)
}

func TestDecodeGoogleConferenceAsURL(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20250315T100000Z\r\n" +
		"DTEND:20250315T110000Z\r\n" +
		"SUMMARY:Google Meet\r\n" +
		"X-GOOGLE-CONFERENCE:https://meet.google.com/abc-defg-hij\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "https://meet.google.com/abc-defg-hij", events[0].URL)
}

func TestDecodeURLTakesPrecedenceOverGoogleConference(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"DTSTART:20250315T100000Z\r\n" +
		"DTEND:20250315T110000Z\r\n" +
		"SUMMARY:Meeting with both\r\n" +
		"URL:https://example.com/meeting\r\n" +
		"X-GOOGLE-CONFERENCE:https://meet.google.com/abc-defg-hij\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "https://example.com/meeting", events[0].URL)
}

func TestDecodeUnknownTZIDNoVTimezone(t *testing.T) {
	ics := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"SUMMARY:Unknown TZ Event\r\n" +
		"DTSTART;TZID=Unknown/Nowhere:20250115T090000\r\n" +
		"DTEND;TZID=Unknown/Nowhere:20250115T100000\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(ics))
	require.NoError(t, err)
	require.Len(t, events, 1)
	// No timezone info available, parsed as-is (no offset applied)
	assert.Equal(t, "2025-01-15T09:00:00Z", events[0].StartTime)
}
