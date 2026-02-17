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
