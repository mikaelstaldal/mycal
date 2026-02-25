package ical

import (
	"strings"
	"testing"
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Title != "Team Meeting" {
		t.Errorf("title = %q, want %q", e.Title, "Team Meeting")
	}
	if e.Description != "Weekly sync" {
		t.Errorf("description = %q, want %q", e.Description, "Weekly sync")
	}
	if e.StartTime != "2025-03-15T10:00:00Z" {
		t.Errorf("start = %q, want %q", e.StartTime, "2025-03-15T10:00:00Z")
	}
	if e.EndTime != "2025-03-15T11:00:00Z" {
		t.Errorf("end = %q, want %q", e.EndTime, "2025-03-15T11:00:00Z")
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Title != "Event One" {
		t.Errorf("event 0 title = %q", events[0].Title)
	}
	if events[1].Title != "Event Two" {
		t.Errorf("event 1 title = %q", events[1].Title)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	want := "This is a very long summary that spans multiple lines"
	if events[0].Title != want {
		t.Errorf("title = %q, want %q", events[0].Title, want)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	// Stockholm is CET (UTC+1) in March
	if events[0].StartTime != "2025-03-15T09:00:00Z" {
		t.Errorf("start = %q, want %q", events[0].StartTime, "2025-03-15T09:00:00Z")
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].StartTime != "2025-03-15T00:00:00Z" {
		t.Errorf("start = %q, want %q", events[0].StartTime, "2025-03-15T00:00:00Z")
	}
	if events[0].EndTime != "2025-03-16T00:00:00Z" {
		t.Errorf("end = %q, want %q", events[0].EndTime, "2025-03-16T00:00:00Z")
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "Hello, World" {
		t.Errorf("title = %q, want %q", events[0].Title, "Hello, World")
	}
	wantDesc := "Line one\nLine two;semicolon\\backslash"
	if events[0].Description != wantDesc {
		t.Errorf("description = %q, want %q", events[0].Description, wantDesc)
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "Valid Event" {
		t.Errorf("title = %q, want %q", events[0].Title, "Valid Event")
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events (parent + override), got %d", len(events))
	}
	// First event is the parent
	if events[0].Title != "Weekly Meeting" {
		t.Errorf("parent title = %q, want %q", events[0].Title, "Weekly Meeting")
	}
	if events[0].RecurrenceFreq != "WEEKLY" {
		t.Errorf("parent freq = %q, want WEEKLY", events[0].RecurrenceFreq)
	}
	if events[0].RecurrenceOriginalStart != "" {
		t.Errorf("parent should not have RecurrenceOriginalStart, got %q", events[0].RecurrenceOriginalStart)
	}
	// Second event is the override
	if events[1].Title != "Weekly Meeting (moved)" {
		t.Errorf("override title = %q, want %q", events[1].Title, "Weekly Meeting (moved)")
	}
	if events[1].RecurrenceOriginalStart != "2026-03-09T10:00:00Z" {
		t.Errorf("override RecurrenceOriginalStart = %q, want %q", events[1].RecurrenceOriginalStart, "2026-03-09T10:00:00Z")
	}
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
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	// June 15 is CEST (UTC+2): 10:00 local = 08:00 UTC
	if events[0].StartTime != "2025-06-15T08:00:00Z" {
		t.Errorf("start = %q, want %q", events[0].StartTime, "2025-06-15T08:00:00Z")
	}
	if events[0].EndTime != "2025-06-15T09:00:00Z" {
		t.Errorf("end = %q, want %q", events[0].EndTime, "2025-06-15T09:00:00Z")
	}
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
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	// +0530: 14:00 local = 08:30 UTC
	if events[0].StartTime != "2025-01-15T08:30:00Z" {
		t.Errorf("start = %q, want %q", events[0].StartTime, "2025-01-15T08:30:00Z")
	}
	if events[0].EndTime != "2025-01-15T09:30:00Z" {
		t.Errorf("end = %q, want %q", events[0].EndTime, "2025-01-15T09:30:00Z")
	}
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
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	// January EST (UTC-5): 09:00 local = 14:00 UTC
	if events[0].StartTime != "2025-01-15T14:00:00Z" {
		t.Errorf("start = %q, want %q", events[0].StartTime, "2025-01-15T14:00:00Z")
	}
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
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	// No timezone info available, parsed as-is (no offset applied)
	if events[0].StartTime != "2025-01-15T09:00:00Z" {
		t.Errorf("start = %q, want %q", events[0].StartTime, "2025-01-15T09:00:00Z")
	}
}
