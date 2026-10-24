package service

import (
	"strings"
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

// Existing stored rules bypass write validation. Skipped candidates must still
// consume the work budget, including invalid dates and missing nth weekdays.
func TestExpansionAttemptBudget(t *testing.T) {
	start := parseTime("2026-01-01T10:00:00Z")
	to := start.AddDate(100, 0, 0)
	invalidDays := make([]int, maxExpansionAttempts)
	for i := range invalidDays {
		invalidDays[i] = 32
	}
	invalidDays = append(invalidDays, 1)
	assert.Empty(t, expandMonthlyByMonthDay(start, 1, invalidDays, time.Time{}, to, 0))
	assert.Empty(t, expandYearlyByMonth(start, 1, []int{2}, invalidDays, time.Time{}, to, 0))

	invalidWeekdays := make([]byDayEntry, maxExpansionAttempts)
	for i := range invalidWeekdays {
		invalidWeekdays[i] = byDayEntry{Offset: 6, Weekday: time.Monday}
	}
	invalidWeekdays = append(invalidWeekdays, byDayEntry{Offset: 1, Weekday: time.Monday})
	assert.Empty(t, expandMonthlyByDay(start, 1, invalidWeekdays, time.Time{}, to, 0))
	assert.Empty(t, expandYearlyByDay(start, 1, invalidWeekdays, time.Time{}, to, 0))
}

func TestExpansionGeneratedLimitWithinPeriod(t *testing.T) {
	start := parseTime("2026-01-01T10:00:00Z")
	days := make([]int, maxExpansions+1)
	for i := range days {
		days[i] = 1
	}
	assert.Len(t, expandYearlyByMonth(start, 1, []int{1}, days, time.Time{}, start.AddDate(1, 0, 0), 0), maxExpansions)
}

func TestRecurrenceListValidationLimits(t *testing.T) {
	for _, tc := range []struct {
		name      string
		validate  func(string) error
		valid     string
		duplicate string
		limit     int
	}{
		{"month", validateByMonth, "1,12", "2,+02", 12},
		{"monthday", validateByMonthDay, "-31,31", "30,+030", 62},
		{"weekday", validateByDay, "MO,1MO,-53SU", "1MO,+01MO", 749},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.validate(tc.valid))
			require.ErrorContains(t, tc.validate(tc.duplicate), "duplicate")
			require.ErrorContains(t, tc.validate(strings.Repeat("1,", tc.limit)+"1"), "at most")
		})
	}
}

func TestExpansionLegacyRepetitiveInvalidRule(t *testing.T) {
	e := makeEvent("YEARLY", "2026-01-01T10:00:00Z", "2026-01-01T11:00:00Z")
	e.RecurrenceByMonth = strings.TrimSuffix(strings.Repeat("2,", 2500), ",")
	e.RecurrenceByMonthDay = strings.TrimSuffix(strings.Repeat("30,", 1666), ",")
	// Preserve the existing simple-expansion fallback while bounding its work.
	assert.Len(t, expandRecurring(e, parseTime("2026-01-01T00:00:00Z"), parseTime("2026-02-01T00:00:00Z")), 1)
}

// startTimes lists the start times of the instances of e in [from, to).
func startTimes(e model.Event, from, to string) []string {
	var starts []string
	for _, inst := range expandRecurring(e, parseTime(from), parseTime(to)) {
		starts = append(starts, inst.StartTime)
	}
	return starts
}

func TestAdvanceSeriesStart_PreservesRemainingOccurrences(t *testing.T) {
	tests := []struct {
		name    string
		event   model.Event
		cancel  string
		newFrom string
	}{
		{
			name:    "weekly forever",
			event:   makeEvent("WEEKLY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z"),
			cancel:  "2026-02-01T09:00:00Z",
			newFrom: "2026-02-08T09:00:00Z",
		},
		{
			name: "weekly with earlier occurrences already excluded",
			event: makeEvent("WEEKLY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z", func(e *model.Event) {
				e.ExDates = "2026-02-01T09:00:00Z,2026-02-08T09:00:00Z"
			}),
			cancel:  "2026-02-15T09:00:00Z",
			newFrom: "2026-02-22T09:00:00Z",
		},
		{
			name: "next occurrence already excluded",
			event: makeEvent("WEEKLY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z", func(e *model.Event) {
				e.ExDates = "2026-02-08T09:00:00Z,2026-03-01T09:00:00Z"
			}),
			cancel:  "2026-02-01T09:00:00Z",
			newFrom: "2026-02-15T09:00:00Z",
		},
		{
			name: "biweekly by day",
			event: makeEvent("WEEKLY", "2026-02-02T09:00:00+01:00", "2026-02-02T10:00:00+01:00", func(e *model.Event) {
				e.RecurrenceInterval = 2
				e.RecurrenceByDay = "MO,WE"
			}),
			cancel:  "2026-02-02T09:00:00+01:00",
			newFrom: "2026-02-04T09:00:00+01:00",
		},
		{
			name: "by day not in chronological order",
			event: makeEvent("WEEKLY", "2026-02-02T09:00:00Z", "2026-02-02T10:00:00Z", func(e *model.Event) {
				e.RecurrenceByDay = "WE,MO"
				e.RecurrenceCount = 6
				e.ExDates = "2026-02-02T09:00:00Z"
			}),
			cancel:  "2026-02-04T09:00:00Z",
			newFrom: "2026-02-09T09:00:00Z",
		},
		{
			name: "start not matching by day",
			event: makeEvent("WEEKLY", "2026-02-02T09:00:00Z", "2026-02-02T10:00:00Z", func(e *model.Event) {
				e.RecurrenceByDay = "TU"
			}),
			cancel:  "2026-02-03T09:00:00Z",
			newFrom: "2026-02-10T09:00:00Z",
		},
		{
			name: "every third day",
			event: makeEvent("DAILY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z", func(e *model.Event) {
				e.RecurrenceInterval = 3
			}),
			cancel:  "2026-02-01T09:00:00Z",
			newFrom: "2026-02-04T09:00:00Z",
		},
		{
			name: "count",
			event: makeEvent("WEEKLY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z", func(e *model.Event) {
				e.RecurrenceCount = 5
				e.ExDates = "2026-02-08T09:00:00Z"
			}),
			cancel:  "2026-02-01T09:00:00Z",
			newFrom: "2026-02-15T09:00:00Z",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := tt.event
			if before.ExDates == "" {
				before.ExDates = tt.cancel
			} else {
				before.ExDates += "," + tt.cancel
			}
			expected := startTimes(before, "2026-01-01T00:00:00Z", "2028-01-01T00:00:00Z")

			after := tt.event
			require.True(t, advanceSeriesStart(&after, parseTime(tt.cancel), nil))
			assert.Equal(t, tt.newFrom, after.StartTime)
			assert.Equal(t, parseTime(tt.newFrom).Add(time.Hour), parseTime(after.EndTime))
			for _, exd := range strings.Split(after.ExDates, ",") {
				if exd != "" {
					assert.False(t, parseTime(exd).Before(parseTime(after.StartTime)), "stale EXDATE %s kept", exd)
				}
			}
			assert.Equal(t, expected, startTimes(after, "2026-01-01T00:00:00Z", "2028-01-01T00:00:00Z"))
		})
	}
}

func TestAdvanceSeriesStart_ReducesCount(t *testing.T) {
	e := makeEvent("WEEKLY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z", func(e *model.Event) {
		e.RecurrenceCount = 5
		e.ExDates = "2026-02-08T09:00:00Z"
	})
	require.True(t, advanceSeriesStart(&e, parseTime("2026-02-01T09:00:00Z"), nil))
	assert.Equal(t, 3, e.RecurrenceCount)
	assert.Equal(t, "", e.ExDates)
}

func TestAdvanceSeriesStart_KeepsExDateOfEarlierRDate(t *testing.T) {
	e := makeEvent("WEEKLY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z", func(e *model.Event) {
		e.RDates = "2026-02-03T09:00:00Z"
		e.ExDates = "2026-02-03T09:00:00Z"
	})
	require.True(t, advanceSeriesStart(&e, parseTime("2026-02-01T09:00:00Z"), nil))
	assert.Equal(t, "2026-02-08T09:00:00Z", e.StartTime)
	assert.Equal(t, "2026-02-03T09:00:00Z", e.ExDates)
}

func TestAdvanceSeriesStart_Declines(t *testing.T) {
	tests := []struct {
		name   string
		event  model.Event
		cancel string
	}{
		{
			name:   "not the first occurrence",
			event:  makeEvent("WEEKLY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z"),
			cancel: "2026-02-08T09:00:00Z",
		},
		{
			name: "last remaining occurrence",
			event: makeEvent("WEEKLY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z", func(e *model.Event) {
				e.RecurrenceCount = 2
				e.ExDates = "2026-02-08T09:00:00Z"
			}),
			cancel: "2026-02-01T09:00:00Z",
		},
		{
			name: "occurrence after until",
			event: makeEvent("DAILY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z", func(e *model.Event) {
				e.RecurrenceUntil = "2026-02-01T09:00:00Z"
			}),
			cancel: "2026-02-01T09:00:00Z",
		},
		{
			name: "cancelled occurrence is also an RDATE",
			event: makeEvent("WEEKLY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z", func(e *model.Event) {
				e.RDates = "2026-02-01T09:00:00Z"
			}),
			cancel: "2026-02-01T09:00:00Z",
		},
		{
			name: "RDATE before the new start",
			event: makeEvent("WEEKLY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z", func(e *model.Event) {
				e.RDates = "2026-02-03T09:00:00Z"
			}),
			cancel: "2026-02-01T09:00:00Z",
		},
		{
			name:   "monthly",
			event:  makeEvent("MONTHLY", "2026-01-31T09:00:00Z", "2026-01-31T10:00:00Z"),
			cancel: "2026-01-31T09:00:00Z",
		},
		{
			name:   "yearly",
			event:  makeEvent("YEARLY", "2028-02-29T09:00:00Z", "2028-02-29T10:00:00Z"),
			cancel: "2028-02-29T09:00:00Z",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.event
			assert.False(t, advanceSeriesStart(&e, parseTime(tt.cancel), nil))
			assert.Equal(t, tt.event, e)
		})
	}
}

func TestExpandRecurring_FirstInstanceIsAnInstance(t *testing.T) {
	e := makeEvent("WEEKLY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z")
	instances := expandRecurring(e, parseTime("2026-02-01T00:00:00Z"), parseTime("2026-02-09T00:00:00Z"))
	require.Len(t, instances, 2)
	for _, inst := range instances {
		inst.SetStringID()
		assert.Equal(t, "1_"+inst.StartTime, inst.StringID)
		assert.Equal(t, "1", inst.ParentID)
	}
}

func TestAdvanceSeriesStart_Overrides(t *testing.T) {
	parentID := int64(1)
	override := func(originalStart, start string) model.Event {
		return model.Event{ID: 2, RecurrenceParentID: &parentID, RecurrenceOriginalStart: originalStart, StartTime: start}
	}
	tests := []struct {
		name      string
		overrides []model.Event
		advanced  bool
	}{
		{"override of the cancelled occurrence", []model.Event{override("2026-02-01T09:00:00Z", "2026-02-02T09:00:00Z")}, true},
		{"override at or after the new start", []model.Event{override("2026-02-15T09:00:00Z", "2026-02-08T09:00:00Z")}, true},
		{"override moved before the new start", []model.Event{override("2026-02-15T09:00:00Z", "2026-02-03T09:00:00Z")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := makeEvent("WEEKLY", "2026-02-01T09:00:00Z", "2026-02-01T10:00:00Z")
			assert.Equal(t, tt.advanced, advanceSeriesStart(&e, parseTime("2026-02-01T09:00:00Z"), tt.overrides))
		})
	}
}

func TestExpandWeeklyByDay_ChronologicalWithinWeek(t *testing.T) {
	e := makeEvent("WEEKLY", "2026-02-02T09:00:00Z", "2026-02-02T10:00:00Z", func(e *model.Event) {
		e.RecurrenceByDay = "WE,MO"
		e.RecurrenceCount = 3
	})
	assert.Equal(t, []string{"2026-02-02T09:00:00Z", "2026-02-04T09:00:00Z", "2026-02-09T09:00:00Z"},
		startTimes(e, "2026-01-01T00:00:00Z", "2027-01-01T00:00:00Z"))
}
