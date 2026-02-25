package service

import (
	"testing"
	"time"

	"github.com/mikaelstaldal/mycal/internal/model"
)

func makeEvent(freq string, start, end string, opts ...func(*model.Event)) model.Event {
	e := model.Event{
		ID:             1,
		Title:          "Test",
		RecurrenceFreq: freq,
		StartTime:      start,
		EndTime:        end,
	}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func TestExpandSimpleDaily(t *testing.T) {
	e := makeEvent("DAILY", "2026-02-01T10:00:00Z", "2026-02-01T11:00:00Z")
	from := parseTime("2026-02-01T00:00:00Z")
	to := parseTime("2026-02-04T00:00:00Z")

	instances := expandRecurring(e, from, to)
	if len(instances) != 3 {
		t.Fatalf("expected 3 instances, got %d", len(instances))
	}
	if instances[0].StartTime != "2026-02-01T10:00:00Z" {
		t.Errorf("first instance start = %s", instances[0].StartTime)
	}
	if instances[2].StartTime != "2026-02-03T10:00:00Z" {
		t.Errorf("third instance start = %s", instances[2].StartTime)
	}
}

func TestExpandWithInterval(t *testing.T) {
	e := makeEvent("WEEKLY", "2026-02-02T10:00:00Z", "2026-02-02T11:00:00Z", func(e *model.Event) {
		e.RecurrenceInterval = 2
	})
	from := parseTime("2026-02-01T00:00:00Z")
	to := parseTime("2026-03-16T00:00:00Z")

	instances := expandRecurring(e, from, to)
	if len(instances) != 3 {
		t.Fatalf("expected 3 instances (every 2 weeks), got %d", len(instances))
	}
	// Feb 2, Feb 16, Mar 2
	if instances[0].StartTime != "2026-02-02T10:00:00Z" {
		t.Errorf("instance 0: %s", instances[0].StartTime)
	}
	if instances[1].StartTime != "2026-02-16T10:00:00Z" {
		t.Errorf("instance 1: %s", instances[1].StartTime)
	}
	if instances[2].StartTime != "2026-03-02T10:00:00Z" {
		t.Errorf("instance 2: %s", instances[2].StartTime)
	}
}

func TestExpandWeeklyByDay(t *testing.T) {
	// Weekly on Mon, Wed, Fri starting from Monday Feb 2
	e := makeEvent("WEEKLY", "2026-02-02T10:00:00Z", "2026-02-02T11:00:00Z", func(e *model.Event) {
		e.RecurrenceByDay = "MO,WE,FR"
	})
	from := parseTime("2026-02-02T00:00:00Z")
	to := parseTime("2026-02-09T00:00:00Z")

	instances := expandRecurring(e, from, to)
	if len(instances) != 3 {
		t.Fatalf("expected 3 instances (MO,WE,FR), got %d", len(instances))
	}
	// Mon Feb 2, Wed Feb 4, Fri Feb 6
	if instances[0].StartTime != "2026-02-02T10:00:00Z" {
		t.Errorf("instance 0: %s (expected Mon Feb 2)", instances[0].StartTime)
	}
	if instances[1].StartTime != "2026-02-04T10:00:00Z" {
		t.Errorf("instance 1: %s (expected Wed Feb 4)", instances[1].StartTime)
	}
	if instances[2].StartTime != "2026-02-06T10:00:00Z" {
		t.Errorf("instance 2: %s (expected Fri Feb 6)", instances[2].StartTime)
	}
}

func TestExpandMonthlyByMonthDay(t *testing.T) {
	// Monthly on the 15th
	e := makeEvent("MONTHLY", "2026-01-15T10:00:00Z", "2026-01-15T11:00:00Z", func(e *model.Event) {
		e.RecurrenceByMonthDay = "15"
	})
	from := parseTime("2026-01-01T00:00:00Z")
	to := parseTime("2026-04-01T00:00:00Z")

	instances := expandRecurring(e, from, to)
	if len(instances) != 3 {
		t.Fatalf("expected 3 instances, got %d", len(instances))
	}
	if instances[0].StartTime != "2026-01-15T10:00:00Z" {
		t.Errorf("instance 0: %s", instances[0].StartTime)
	}
	if instances[1].StartTime != "2026-02-15T10:00:00Z" {
		t.Errorf("instance 1: %s", instances[1].StartTime)
	}
	if instances[2].StartTime != "2026-03-15T10:00:00Z" {
		t.Errorf("instance 2: %s", instances[2].StartTime)
	}
}

func TestExpandMonthlyByDayWithOffset(t *testing.T) {
	// Monthly on the 2nd Monday
	e := makeEvent("MONTHLY", "2026-01-12T10:00:00Z", "2026-01-12T11:00:00Z", func(e *model.Event) {
		e.RecurrenceByDay = "2MO"
	})
	from := parseTime("2026-01-01T00:00:00Z")
	to := parseTime("2026-04-01T00:00:00Z")

	instances := expandRecurring(e, from, to)
	if len(instances) != 3 {
		t.Fatalf("expected 3 instances, got %d", len(instances))
	}
	// 2nd Monday: Jan 12, Feb 9, Mar 9
	if instances[0].StartTime != "2026-01-12T10:00:00Z" {
		t.Errorf("instance 0: %s", instances[0].StartTime)
	}
	if instances[1].StartTime != "2026-02-09T10:00:00Z" {
		t.Errorf("instance 1: %s", instances[1].StartTime)
	}
	if instances[2].StartTime != "2026-03-09T10:00:00Z" {
		t.Errorf("instance 2: %s", instances[2].StartTime)
	}
}

func TestExpandMonthlyLastFriday(t *testing.T) {
	// Monthly on the last Friday
	e := makeEvent("MONTHLY", "2026-01-30T10:00:00Z", "2026-01-30T11:00:00Z", func(e *model.Event) {
		e.RecurrenceByDay = "-1FR"
	})
	from := parseTime("2026-01-01T00:00:00Z")
	to := parseTime("2026-04-01T00:00:00Z")

	instances := expandRecurring(e, from, to)
	if len(instances) != 3 {
		t.Fatalf("expected 3 instances, got %d", len(instances))
	}
	// Last Fridays: Jan 30, Feb 27, Mar 27
	expected := []string{"2026-01-30T10:00:00Z", "2026-02-27T10:00:00Z", "2026-03-27T10:00:00Z"}
	for i, exp := range expected {
		if instances[i].StartTime != exp {
			t.Errorf("instance %d: got %s, want %s", i, instances[i].StartTime, exp)
		}
	}
}

func TestExpandYearlyByMonth(t *testing.T) {
	// Yearly in January and June on the 15th
	e := makeEvent("YEARLY", "2026-01-15T10:00:00Z", "2026-01-15T11:00:00Z", func(e *model.Event) {
		e.RecurrenceByMonth = "1,6"
	})
	from := parseTime("2026-01-01T00:00:00Z")
	to := parseTime("2027-07-01T00:00:00Z")

	instances := expandRecurring(e, from, to)
	if len(instances) != 4 {
		t.Fatalf("expected 4 instances, got %d", len(instances))
	}
	// Jan 15 2026, Jun 15 2026, Jan 15 2027, Jun 15 2027
	if instances[0].StartTime != "2026-01-15T10:00:00Z" {
		t.Errorf("instance 0: %s", instances[0].StartTime)
	}
	if instances[1].StartTime != "2026-06-15T10:00:00Z" {
		t.Errorf("instance 1: %s", instances[1].StartTime)
	}
}

func TestExpandWithCount(t *testing.T) {
	e := makeEvent("DAILY", "2026-02-01T10:00:00Z", "2026-02-01T11:00:00Z", func(e *model.Event) {
		e.RecurrenceCount = 3
	})
	from := parseTime("2026-02-01T00:00:00Z")
	to := parseTime("2026-12-31T00:00:00Z")

	instances := expandRecurring(e, from, to)
	if len(instances) != 3 {
		t.Fatalf("expected 3 instances (COUNT=3), got %d", len(instances))
	}
}

func TestExpandWithUntil(t *testing.T) {
	e := makeEvent("DAILY", "2026-02-01T10:00:00Z", "2026-02-01T11:00:00Z", func(e *model.Event) {
		e.RecurrenceUntil = "2026-02-03T23:59:59Z"
	})
	from := parseTime("2026-02-01T00:00:00Z")
	to := parseTime("2026-12-31T00:00:00Z")

	instances := expandRecurring(e, from, to)
	if len(instances) != 3 {
		t.Fatalf("expected 3 instances (UNTIL Feb 3), got %d", len(instances))
	}
}

func TestExpandWithExdate(t *testing.T) {
	e := makeEvent("DAILY", "2026-02-01T10:00:00Z", "2026-02-01T11:00:00Z", func(e *model.Event) {
		e.ExDates = "2026-02-02T10:00:00Z"
		e.RecurrenceCount = 5
	})
	from := parseTime("2026-02-01T00:00:00Z")
	to := parseTime("2026-02-10T00:00:00Z")

	instances := expandRecurring(e, from, to)
	if len(instances) != 4 {
		t.Fatalf("expected 4 instances (5 - 1 exdate), got %d", len(instances))
	}
	// Should skip Feb 2
	for _, inst := range instances {
		if inst.StartTime == "2026-02-02T10:00:00Z" {
			t.Error("EXDATE instance should be excluded")
		}
	}
}

func TestExpandWithRdate(t *testing.T) {
	e := makeEvent("WEEKLY", "2026-02-02T10:00:00Z", "2026-02-02T11:00:00Z", func(e *model.Event) {
		e.RDates = "2026-02-05T10:00:00Z"
		e.RecurrenceCount = 2
	})
	from := parseTime("2026-02-01T00:00:00Z")
	to := parseTime("2026-02-15T00:00:00Z")

	instances := expandRecurring(e, from, to)
	if len(instances) != 3 {
		t.Fatalf("expected 3 instances (2 weekly + 1 RDATE), got %d", len(instances))
	}
}

func TestExpandIntervalWithByDay(t *testing.T) {
	// Every 2 weeks on MO,FR
	e := makeEvent("WEEKLY", "2026-02-02T10:00:00Z", "2026-02-02T11:00:00Z", func(e *model.Event) {
		e.RecurrenceInterval = 2
		e.RecurrenceByDay = "MO,FR"
		e.RecurrenceCount = 4
	})
	from := parseTime("2026-02-01T00:00:00Z")
	to := parseTime("2026-03-31T00:00:00Z")

	instances := expandRecurring(e, from, to)
	if len(instances) != 4 {
		t.Fatalf("expected 4 instances, got %d", len(instances))
	}
	// Week 1 (Feb 2): Mon Feb 2, Fri Feb 6
	// Skip week 2
	// Week 3 (Feb 16): Mon Feb 16, Fri Feb 20
	if instances[0].StartTime != "2026-02-02T10:00:00Z" {
		t.Errorf("instance 0: %s", instances[0].StartTime)
	}
	if instances[1].StartTime != "2026-02-06T10:00:00Z" {
		t.Errorf("instance 1: %s", instances[1].StartTime)
	}
	if instances[2].StartTime != "2026-02-16T10:00:00Z" {
		t.Errorf("instance 2: %s", instances[2].StartTime)
	}
	if instances[3].StartTime != "2026-02-20T10:00:00Z" {
		t.Errorf("instance 3: %s", instances[3].StartTime)
	}
}

func TestNthWeekdayOfMonth(t *testing.T) {
	// 2nd Monday of February 2026
	d, ok := nthWeekdayOfMonth(2026, time.February, time.Monday, 2)
	if !ok {
		t.Fatal("expected ok")
	}
	if d.Day() != 9 {
		t.Errorf("expected day 9, got %d", d.Day())
	}

	// Last Friday of January 2026
	d, ok = nthWeekdayOfMonth(2026, time.January, time.Friday, -1)
	if !ok {
		t.Fatal("expected ok")
	}
	if d.Day() != 30 {
		t.Errorf("expected day 30, got %d", d.Day())
	}

	// 5th Monday of February 2026 (doesn't exist)
	_, ok = nthWeekdayOfMonth(2026, time.February, time.Monday, 5)
	if ok {
		t.Error("expected not ok for 5th Monday of Feb")
	}
}

func TestParseByDay(t *testing.T) {
	entries := parseByDay("MO,WE,FR")
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Weekday != time.Monday || entries[0].Offset != 0 {
		t.Errorf("entry 0: %+v", entries[0])
	}
	if entries[1].Weekday != time.Wednesday || entries[1].Offset != 0 {
		t.Errorf("entry 1: %+v", entries[1])
	}

	// Ordinal
	entries = parseByDay("2MO,-1FR")
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Offset != 2 || entries[0].Weekday != time.Monday {
		t.Errorf("entry 0: %+v", entries[0])
	}
	if entries[1].Offset != -1 || entries[1].Weekday != time.Friday {
		t.Errorf("entry 1: %+v", entries[1])
	}
}

func TestParseIntList(t *testing.T) {
	result := parseIntList("15,30")
	if len(result) != 2 || result[0] != 15 || result[1] != 30 {
		t.Errorf("got %v", result)
	}

	result = parseIntList("")
	if result != nil {
		t.Errorf("expected nil for empty, got %v", result)
	}
}
