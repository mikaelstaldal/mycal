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

func TestDecodeSkipsRecurrenceID(t *testing.T) {
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
	if len(events) != 1 {
		t.Fatalf("expected 1 event (override skipped), got %d", len(events))
	}
	if events[0].Title != "Weekly Meeting" {
		t.Errorf("title = %q, want %q", events[0].Title, "Weekly Meeting")
	}
	if events[0].RecurrenceFreq != "WEEKLY" {
		t.Errorf("freq = %q, want WEEKLY", events[0].RecurrenceFreq)
	}
}
