package ical

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			Color:       "dodgerblue",
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
	err := Encode(&buf, events)
	require.NoError(t, err)

	output := buf.String()

	// Verify iCal structure
	assert.Contains(t, output, "BEGIN:VCALENDAR")
	assert.Contains(t, output, "END:VCALENDAR")
	assert.Contains(t, output, "VERSION:2.0")

	// Verify first event
	assert.Contains(t, output, "UID:event-1@mycal")
	assert.Contains(t, output, "SUMMARY:Team Meeting")
	assert.Contains(t, output, "DTSTART:20260217T140000Z")
	assert.Contains(t, output, "DESCRIPTION:Weekly sync")

	// Verify comma escaping in second event
	assert.Contains(t, output, "SUMMARY:Lunch\\, with friends")

	// Count VEVENT blocks
	assert.Equal(t, 2, strings.Count(output, "BEGIN:VEVENT"))
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
	err := Encode(&buf, events)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "LOCATION:Stockholm Office")
	assert.Contains(t, output, "GEO:59.329300;18.068600")
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
	require.NoError(t, err)
	require.Len(t, events, 1)
	ev := events[0]
	assert.Equal(t, "Stockholm Office", ev.Location)
	require.NotNil(t, ev.Latitude)
	require.NotNil(t, ev.Longitude)
	assert.InDelta(t, 59.3293, *ev.Latitude, 0.001)
	assert.InDelta(t, 18.0686, *ev.Longitude, 0.001)
}

func TestEncodeDecodeRRuleInterval(t *testing.T) {
	events := []model.Event{
		{
			ID:                 1,
			Title:              "Biweekly",
			StartTime:          "2026-02-02T10:00:00Z",
			EndTime:            "2026-02-02T11:00:00Z",
			RecurrenceFreq:     "WEEKLY",
			RecurrenceInterval: 2,
			CreatedAt:          "2026-02-01T00:00:00Z",
			UpdatedAt:          "2026-02-01T00:00:00Z",
		},
	}

	var buf bytes.Buffer
	err := Encode(&buf, events)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "RRULE:FREQ=WEEKLY;INTERVAL=2")

	decoded, err := Decode(strings.NewReader(output))
	require.NoError(t, err)
	require.Len(t, decoded, 1)
	assert.Equal(t, "WEEKLY", decoded[0].RecurrenceFreq)
	assert.Equal(t, 2, decoded[0].RecurrenceInterval)
}

func TestEncodeDecodeRRuleByDay(t *testing.T) {
	events := []model.Event{
		{
			ID:              1,
			Title:           "MWF Meeting",
			StartTime:       "2026-02-02T10:00:00Z",
			EndTime:         "2026-02-02T11:00:00Z",
			RecurrenceFreq:  "WEEKLY",
			RecurrenceByDay: "MO,WE,FR",
			CreatedAt:       "2026-02-01T00:00:00Z",
			UpdatedAt:       "2026-02-01T00:00:00Z",
		},
	}

	var buf bytes.Buffer
	err := Encode(&buf, events)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, ";BYDAY=MO,WE,FR")

	decoded, err := Decode(strings.NewReader(output))
	require.NoError(t, err)
	assert.Equal(t, "MO,WE,FR", decoded[0].RecurrenceByDay)
}

func TestEncodeDecodeExdate(t *testing.T) {
	events := []model.Event{
		{
			ID:             1,
			Title:          "Daily",
			StartTime:      "2026-02-02T10:00:00Z",
			EndTime:        "2026-02-02T11:00:00Z",
			RecurrenceFreq: "DAILY",
			ExDates:        "2026-02-05T10:00:00Z,2026-02-10T10:00:00Z",
			CreatedAt:      "2026-02-01T00:00:00Z",
			UpdatedAt:      "2026-02-01T00:00:00Z",
		},
	}

	var buf bytes.Buffer
	err := Encode(&buf, events)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "EXDATE:20260205T100000Z")
	assert.Contains(t, output, "EXDATE:20260210T100000Z")

	decoded, err := Decode(strings.NewReader(output))
	require.NoError(t, err)
	assert.Equal(t, "2026-02-05T10:00:00Z,2026-02-10T10:00:00Z", decoded[0].ExDates)
}

func TestEncodeDecodeRdate(t *testing.T) {
	events := []model.Event{
		{
			ID:             1,
			Title:          "Weekly",
			StartTime:      "2026-02-02T10:00:00Z",
			EndTime:        "2026-02-02T11:00:00Z",
			RecurrenceFreq: "WEEKLY",
			RDates:         "2026-02-05T10:00:00Z",
			CreatedAt:      "2026-02-01T00:00:00Z",
			UpdatedAt:      "2026-02-01T00:00:00Z",
		},
	}

	var buf bytes.Buffer
	err := Encode(&buf, events)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "RDATE:20260205T100000Z")

	decoded, err := Decode(strings.NewReader(output))
	require.NoError(t, err)
	assert.Equal(t, "2026-02-05T10:00:00Z", decoded[0].RDates)
}

func TestDecodeRRuleWithByParams(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"SUMMARY:Monthly\r\n" +
		"DTSTART:20260115T100000Z\r\n" +
		"DTEND:20260115T110000Z\r\n" +
		"RRULE:FREQ=MONTHLY;INTERVAL=2;BYDAY=2MO;COUNT=6\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	e := events[0]
	assert.Equal(t, "MONTHLY", e.RecurrenceFreq)
	assert.Equal(t, 2, e.RecurrenceInterval)
	assert.Equal(t, "2MO", e.RecurrenceByDay)
	assert.Equal(t, 6, e.RecurrenceCount)
}

func TestDecodeValidColor(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"SUMMARY:Golden\r\n" +
		"DTSTART:20260115T100000Z\r\n" +
		"DTEND:20260115T110000Z\r\n" +
		"COLOR:gold\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "gold", events[0].Color)
}

func TestDecodeInvalidColor(t *testing.T) {
	input := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"SUMMARY:Golden\r\n" +
		"DTSTART:20260115T100000Z\r\n" +
		"DTEND:20260115T110000Z\r\n" +
		"COLOR:bogus\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	events, err := Decode(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Empty(t, events[0].Color)
}

func TestEncodeEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := Encode(&buf, []model.Event{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "BEGIN:VCALENDAR")
	assert.NotContains(t, output, "BEGIN:VEVENT")
}
