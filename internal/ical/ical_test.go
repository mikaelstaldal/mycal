package ical

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mikaelstaldal/mycal/internal/model"
)

func TestEncode(t *testing.T) {
	events := []model.Event{
		{
			ID:          1,
			Title:       "Team Meeting",
			Description: "Weekly sync",
			StartTime:   "2026-02-17T14:00:00Z",
			EndTime:     "2026-02-17T15:00:00Z",
			Color:       "#4285f4",
			CreatedAt:   "2026-02-17T10:00:00Z",
			UpdatedAt:   "2026-02-17T10:00:00Z",
		},
		{
			ID:        2,
			Title:     "Lunch, with friends",
			StartTime: "2026-02-18T12:00:00Z",
			EndTime:   "2026-02-18T13:00:00Z",
			CreatedAt: "2026-02-17T10:00:00Z",
			UpdatedAt: "2026-02-17T10:00:00Z",
		},
	}

	var buf bytes.Buffer
	if err := Encode(&buf, events); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	output := buf.String()

	// Verify iCal structure
	if !strings.Contains(output, "BEGIN:VCALENDAR") {
		t.Error("missing VCALENDAR begin")
	}
	if !strings.Contains(output, "END:VCALENDAR") {
		t.Error("missing VCALENDAR end")
	}
	if !strings.Contains(output, "VERSION:2.0") {
		t.Error("missing VERSION")
	}

	// Verify first event
	if !strings.Contains(output, "UID:event-1@mycal") {
		t.Error("missing UID for event 1")
	}
	if !strings.Contains(output, "SUMMARY:Team Meeting") {
		t.Error("missing SUMMARY for event 1")
	}
	if !strings.Contains(output, "DTSTART:20260217T140000Z") {
		t.Error("missing DTSTART for event 1")
	}
	if !strings.Contains(output, "DESCRIPTION:Weekly sync") {
		t.Error("missing DESCRIPTION for event 1")
	}

	// Verify comma escaping in second event
	if !strings.Contains(output, "SUMMARY:Lunch\\, with friends") {
		t.Error("comma not escaped in SUMMARY")
	}

	// Count VEVENT blocks
	if strings.Count(output, "BEGIN:VEVENT") != 2 {
		t.Errorf("expected 2 VEVENT blocks, got %d", strings.Count(output, "BEGIN:VEVENT"))
	}
}

func TestEncodeWithLocation(t *testing.T) {
	lat := 59.3293
	lon := 18.0686
	events := []model.Event{
		{
			ID:        1,
			Title:     "Office Meeting",
			StartTime: "2026-02-17T14:00:00Z",
			EndTime:   "2026-02-17T15:00:00Z",
			Location:  "Stockholm Office",
			Latitude:  &lat,
			Longitude: &lon,
			CreatedAt: "2026-02-17T10:00:00Z",
			UpdatedAt: "2026-02-17T10:00:00Z",
		},
	}

	var buf bytes.Buffer
	if err := Encode(&buf, events); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "LOCATION:Stockholm Office") {
		t.Error("missing LOCATION property")
	}
	if !strings.Contains(output, "GEO:59.329300;18.068600") {
		t.Errorf("missing or incorrect GEO property, output:\n%s", output)
	}
}

func TestDecodeWithLocation(t *testing.T) {
	input := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:Office Meeting
DTSTART:20260217T140000Z
DTEND:20260217T150000Z
LOCATION:Stockholm Office
GEO:59.3293;18.0686
END:VEVENT
END:VCALENDAR`

	events, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	ev := events[0]
	if ev.Location != "Stockholm Office" {
		t.Errorf("got location %q, want %q", ev.Location, "Stockholm Office")
	}
	if ev.Latitude == nil || ev.Longitude == nil {
		t.Fatal("expected non-nil coordinates")
	}
	if *ev.Latitude < 59.329 || *ev.Latitude > 59.330 {
		t.Errorf("got latitude %f, want ~59.3293", *ev.Latitude)
	}
	if *ev.Longitude < 18.068 || *ev.Longitude > 18.069 {
		t.Errorf("got longitude %f, want ~18.0686", *ev.Longitude)
	}
}

func TestEncodeEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := Encode(&buf, []model.Event{}); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "BEGIN:VCALENDAR") {
		t.Error("missing VCALENDAR")
	}
	if strings.Contains(output, "BEGIN:VEVENT") {
		t.Error("should have no VEVENT blocks for empty list")
	}
}
