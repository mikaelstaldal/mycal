package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	require.Len(t, instances, 3)
	assert.Equal(t, "2026-02-01T10:00:00Z", instances[0].StartTime)
	assert.Equal(t, "2026-02-03T10:00:00Z", instances[2].StartTime)
}

func TestExpandWithInterval(t *testing.T) {
	e := makeEvent("WEEKLY", "2026-02-02T10:00:00Z", "2026-02-02T11:00:00Z", func(e *model.Event) {
		e.RecurrenceInterval = 2
	})
	from := parseTime("2026-02-01T00:00:00Z")
	to := parseTime("2026-03-16T00:00:00Z")

	instances := expandRecurring(e, from, to)
	require.Len(t, instances, 3)
	// Feb 2, Feb 16, Mar 2
	assert.Equal(t, "2026-02-02T10:00:00Z", instances[0].StartTime)
	assert.Equal(t, "2026-02-16T10:00:00Z", instances[1].StartTime)
	assert.Equal(t, "2026-03-02T10:00:00Z", instances[2].StartTime)
}

func TestExpandWeeklyByDay(t *testing.T) {
	// Weekly on Mon, Wed, Fri starting from Monday Feb 2
	e := makeEvent("WEEKLY", "2026-02-02T10:00:00Z", "2026-02-02T11:00:00Z", func(e *model.Event) {
		e.RecurrenceByDay = "MO,WE,FR"
	})
	from := parseTime("2026-02-02T00:00:00Z")
	to := parseTime("2026-02-09T00:00:00Z")

	instances := expandRecurring(e, from, to)
	require.Len(t, instances, 3)
	// Mon Feb 2, Wed Feb 4, Fri Feb 6
	assert.Equal(t, "2026-02-02T10:00:00Z", instances[0].StartTime, "expected Mon Feb 2")
	assert.Equal(t, "2026-02-04T10:00:00Z", instances[1].StartTime, "expected Wed Feb 4")
	assert.Equal(t, "2026-02-06T10:00:00Z", instances[2].StartTime, "expected Fri Feb 6")
}

func TestExpandMonthlyByMonthDay(t *testing.T) {
	// Monthly on the 15th
	e := makeEvent("MONTHLY", "2026-01-15T10:00:00Z", "2026-01-15T11:00:00Z", func(e *model.Event) {
		e.RecurrenceByMonthDay = "15"
	})
	from := parseTime("2026-01-01T00:00:00Z")
	to := parseTime("2026-04-01T00:00:00Z")

	instances := expandRecurring(e, from, to)
	require.Len(t, instances, 3)
	assert.Equal(t, "2026-01-15T10:00:00Z", instances[0].StartTime)
	assert.Equal(t, "2026-02-15T10:00:00Z", instances[1].StartTime)
	assert.Equal(t, "2026-03-15T10:00:00Z", instances[2].StartTime)
}

func TestExpandMonthlyByDayWithOffset(t *testing.T) {
	// Monthly on the 2nd Monday
	e := makeEvent("MONTHLY", "2026-01-12T10:00:00Z", "2026-01-12T11:00:00Z", func(e *model.Event) {
		e.RecurrenceByDay = "2MO"
	})
	from := parseTime("2026-01-01T00:00:00Z")
	to := parseTime("2026-04-01T00:00:00Z")

	instances := expandRecurring(e, from, to)
	require.Len(t, instances, 3)
	// 2nd Monday: Jan 12, Feb 9, Mar 9
	assert.Equal(t, "2026-01-12T10:00:00Z", instances[0].StartTime)
	assert.Equal(t, "2026-02-09T10:00:00Z", instances[1].StartTime)
	assert.Equal(t, "2026-03-09T10:00:00Z", instances[2].StartTime)
}

func TestExpandMonthlyLastFriday(t *testing.T) {
	// Monthly on the last Friday
	e := makeEvent("MONTHLY", "2026-01-30T10:00:00Z", "2026-01-30T11:00:00Z", func(e *model.Event) {
		e.RecurrenceByDay = "-1FR"
	})
	from := parseTime("2026-01-01T00:00:00Z")
	to := parseTime("2026-04-01T00:00:00Z")

	instances := expandRecurring(e, from, to)
	require.Len(t, instances, 3)
	// Last Fridays: Jan 30, Feb 27, Mar 27
	expected := []string{"2026-01-30T10:00:00Z", "2026-02-27T10:00:00Z", "2026-03-27T10:00:00Z"}
	for i, exp := range expected {
		assert.Equal(t, exp, instances[i].StartTime, "instance %d", i)
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
	require.Len(t, instances, 4)
	// Jan 15 2026, Jun 15 2026, Jan 15 2027, Jun 15 2027
	assert.Equal(t, "2026-01-15T10:00:00Z", instances[0].StartTime)
	assert.Equal(t, "2026-06-15T10:00:00Z", instances[1].StartTime)
}

func TestExpandWithCount(t *testing.T) {
	e := makeEvent("DAILY", "2026-02-01T10:00:00Z", "2026-02-01T11:00:00Z", func(e *model.Event) {
		e.RecurrenceCount = 3
	})
	from := parseTime("2026-02-01T00:00:00Z")
	to := parseTime("2026-12-31T00:00:00Z")

	instances := expandRecurring(e, from, to)
	assert.Len(t, instances, 3)
}

func TestExpandWithUntil(t *testing.T) {
	e := makeEvent("DAILY", "2026-02-01T10:00:00Z", "2026-02-01T11:00:00Z", func(e *model.Event) {
		e.RecurrenceUntil = "2026-02-03T23:59:59Z"
	})
	from := parseTime("2026-02-01T00:00:00Z")
	to := parseTime("2026-12-31T00:00:00Z")

	instances := expandRecurring(e, from, to)
	assert.Len(t, instances, 3)
}

func TestExpandWithExdate(t *testing.T) {
	e := makeEvent("DAILY", "2026-02-01T10:00:00Z", "2026-02-01T11:00:00Z", func(e *model.Event) {
		e.ExDates = "2026-02-02T10:00:00Z"
		e.RecurrenceCount = 5
	})
	from := parseTime("2026-02-01T00:00:00Z")
	to := parseTime("2026-02-10T00:00:00Z")

	instances := expandRecurring(e, from, to)
	require.Len(t, instances, 4)
	// Should skip Feb 2
	for _, inst := range instances {
		assert.NotEqual(t, "2026-02-02T10:00:00Z", inst.StartTime, "EXDATE instance should be excluded")
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
	assert.Len(t, instances, 3)
}

func TestExpandIntervalWithByDay(t *testing.T) {
	// Every 2 weeks on MO, FR
	e := makeEvent("WEEKLY", "2026-02-02T10:00:00Z", "2026-02-02T11:00:00Z", func(e *model.Event) {
		e.RecurrenceInterval = 2
		e.RecurrenceByDay = "MO,FR"
		e.RecurrenceCount = 4
	})
	from := parseTime("2026-02-01T00:00:00Z")
	to := parseTime("2026-03-31T00:00:00Z")

	instances := expandRecurring(e, from, to)
	require.Len(t, instances, 4)
	// Week 1 (Feb 2): Mon Feb 2, Fri Feb 6
	// Skip week 2
	// Week 3 (Feb 16): Mon Feb 16, Fri Feb 20
	assert.Equal(t, "2026-02-02T10:00:00Z", instances[0].StartTime)
	assert.Equal(t, "2026-02-06T10:00:00Z", instances[1].StartTime)
	assert.Equal(t, "2026-02-16T10:00:00Z", instances[2].StartTime)
	assert.Equal(t, "2026-02-20T10:00:00Z", instances[3].StartTime)
}

func TestNthWeekdayOfMonth(t *testing.T) {
	// 2nd Monday of February 2026
	d, ok := nthWeekdayOfMonth(2026, time.February, time.Monday, 2)
	assert.True(t, ok)
	assert.Equal(t, 9, d.Day())

	// Last Friday of January 2026
	d, ok = nthWeekdayOfMonth(2026, time.January, time.Friday, -1)
	assert.True(t, ok)
	assert.Equal(t, 30, d.Day())

	// 5th Monday of February 2026 (doesn't exist)
	_, ok = nthWeekdayOfMonth(2026, time.February, time.Monday, 5)
	assert.False(t, ok)
}

func TestParseByDay(t *testing.T) {
	entries := parseByDay("MO,WE,FR")
	require.Len(t, entries, 3)
	assert.Equal(t, time.Monday, entries[0].Weekday)
	assert.Equal(t, 0, entries[0].Offset)
	assert.Equal(t, time.Wednesday, entries[1].Weekday)
	assert.Equal(t, 0, entries[1].Offset)

	// Ordinal
	entries = parseByDay("2MO,-1FR")
	require.Len(t, entries, 2)
	assert.Equal(t, 2, entries[0].Offset)
	assert.Equal(t, time.Monday, entries[0].Weekday)
	assert.Equal(t, -1, entries[1].Offset)
	assert.Equal(t, time.Friday, entries[1].Weekday)
}

func TestParseIntList(t *testing.T) {
	result := parseIntList("15,30")
	assert.Equal(t, []int{15, 30}, result)

	result = parseIntList("")
	assert.Nil(t, result)
}
