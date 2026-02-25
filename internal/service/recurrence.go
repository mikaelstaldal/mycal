package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mikaelstaldal/mycal/internal/model"
)

const maxExpansions = 1000

// byDayEntry represents a parsed BYDAY value like "2MO" (2nd Monday) or "MO" (every Monday).
type byDayEntry struct {
	Offset  int          // 0 means every occurrence, positive = nth, negative = nth from end
	Weekday time.Weekday // Go weekday value
}

var weekdayMap = map[string]time.Weekday{
	"SU": time.Sunday,
	"MO": time.Monday,
	"TU": time.Tuesday,
	"WE": time.Wednesday,
	"TH": time.Thursday,
	"FR": time.Friday,
	"SA": time.Saturday,
}

func parseByDay(s string) []byDayEntry {
	if s == "" {
		return nil
	}
	var entries []byDayEntry
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if len(part) < 2 {
			continue
		}
		dayAbbr := part[len(part)-2:]
		wd, ok := weekdayMap[dayAbbr]
		if !ok {
			continue
		}
		offset := 0
		if len(part) > 2 {
			n, err := strconv.Atoi(part[:len(part)-2])
			if err != nil {
				continue
			}
			offset = n
		}
		entries = append(entries, byDayEntry{Offset: offset, Weekday: wd})
	}
	return entries
}

func parseIntList(s string) []int {
	if s == "" {
		return nil
	}
	var result []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		result = append(result, n)
	}
	return result
}

// nthWeekdayOfMonth finds the nth occurrence of a weekday in the given month.
// n > 0: count from start (1 = first). n < 0: count from end (-1 = last).
func nthWeekdayOfMonth(year int, month time.Month, wd time.Weekday, n int) (time.Time, bool) {
	if n == 0 {
		return time.Time{}, false
	}
	if n > 0 {
		// Find first occurrence of weekday in month
		first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		diff := int(wd) - int(first.Weekday())
		if diff < 0 {
			diff += 7
		}
		day := 1 + diff + (n-1)*7
		t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		if t.Month() != month {
			return time.Time{}, false
		}
		return t, true
	}
	// n < 0: count from end
	// Find last day of month
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC)
	diff := int(lastDay.Weekday()) - int(wd)
	if diff < 0 {
		diff += 7
	}
	day := lastDay.Day() - diff + (n+1)*7
	if day < 1 {
		return time.Time{}, false
	}
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return t, true
}

func expandRecurring(event model.Event, from, to time.Time) []model.Event {
	if event.RecurrenceFreq == "" {
		return nil
	}

	startTime, err := time.Parse(time.RFC3339, event.StartTime)
	if err != nil {
		return nil
	}
	endTime, err := time.Parse(time.RFC3339, event.EndTime)
	if err != nil {
		return nil
	}
	duration := endTime.Sub(startTime)

	var untilTime time.Time
	if event.RecurrenceUntil != "" {
		if t, err := time.Parse(time.RFC3339, event.RecurrenceUntil); err == nil {
			untilTime = t
		}
	}

	interval := event.RecurrenceInterval
	if interval < 1 {
		interval = 1
	}

	byDay := parseByDay(event.RecurrenceByDay)
	byMonthDay := parseIntList(event.RecurrenceByMonthDay)
	byMonth := parseIntList(event.RecurrenceByMonth)

	// Parse EXDATE set for filtering
	exdateSet := make(map[string]bool)
	if event.ExDates != "" {
		for _, exd := range strings.Split(event.ExDates, ",") {
			exd = strings.TrimSpace(exd)
			if exd != "" {
				exdateSet[exd] = true
			}
		}
	}

	// Generate candidate start times
	var candidates []time.Time
	hasByParams := len(byDay) > 0 || len(byMonthDay) > 0 || len(byMonth) > 0

	if hasByParams {
		candidates = expandWithByParams(startTime, event.RecurrenceFreq, interval, byDay, byMonthDay, byMonth, untilTime, to, event.RecurrenceCount)
	} else {
		candidates = expandSimple(startTime, event.RecurrenceFreq, interval, untilTime, to, event.RecurrenceCount)
	}

	// Build instances from candidates, filtering by EXDATE and query window
	var instances []model.Event
	for i, instStart := range candidates {
		instEnd := instStart.Add(duration)

		// Filter by EXDATE
		if exdateSet[instStart.Format(time.RFC3339)] {
			continue
		}

		// Include if overlaps the query window
		if instEnd.After(from) && instStart.Before(to) {
			inst := event
			inst.StartTime = instStart.Format(time.RFC3339)
			inst.EndTime = instEnd.Format(time.RFC3339)
			inst.RecurrenceIndex = i
			instances = append(instances, inst)
		}
	}

	// Add RDATE instances
	if event.RDates != "" {
		for _, rd := range strings.Split(event.RDates, ",") {
			rd = strings.TrimSpace(rd)
			if rd == "" {
				continue
			}
			rdTime, err := time.Parse(time.RFC3339, rd)
			if err != nil {
				continue
			}
			// Filter by EXDATE
			if exdateSet[rdTime.Format(time.RFC3339)] {
				continue
			}
			rdEnd := rdTime.Add(duration)
			if rdEnd.After(from) && rdTime.Before(to) {
				if !untilTime.IsZero() && rdTime.After(untilTime) {
					continue
				}
				inst := event
				inst.StartTime = rdTime.Format(time.RFC3339)
				inst.EndTime = rdEnd.Format(time.RFC3339)
				inst.RecurrenceIndex = -1 // mark as RDATE instance
				instances = append(instances, inst)
			}
		}
		// Re-sort after adding RDATEs
		sort.Slice(instances, func(i, j int) bool {
			return instances[i].StartTime < instances[j].StartTime
		})
	}

	return instances
}

func expandSimple(startTime time.Time, freq string, interval int, untilTime, to time.Time, count int) []time.Time {
	var candidates []time.Time
	for i := 0; i < maxExpansions; i++ {
		if count > 0 && i >= count {
			break
		}
		instStart := addFreq(startTime, freq, i*interval)
		if !untilTime.IsZero() && instStart.After(untilTime) {
			break
		}
		if instStart.Compare(to) >= 0 {
			break
		}
		candidates = append(candidates, instStart)
	}
	return candidates
}

func expandWithByParams(startTime time.Time, freq string, interval int, byDay []byDayEntry, byMonthDay []int, byMonth []int, untilTime, to time.Time, count int) []time.Time {
	var candidates []time.Time
	generated := 0

	switch freq {
	case "WEEKLY":
		if len(byDay) > 0 {
			candidates = expandWeeklyByDay(startTime, interval, byDay, untilTime, to, count)
		}
	case "MONTHLY":
		if len(byDay) > 0 {
			candidates = expandMonthlyByDay(startTime, interval, byDay, untilTime, to, count)
		} else if len(byMonthDay) > 0 {
			candidates = expandMonthlyByMonthDay(startTime, interval, byMonthDay, untilTime, to, count)
		}
	case "YEARLY":
		if len(byMonth) > 0 {
			candidates = expandYearlyByMonth(startTime, interval, byMonth, byMonthDay, untilTime, to, count)
		} else if len(byDay) > 0 {
			// YEARLY + BYDAY (e.g. every year on the 2nd Monday of the start month)
			candidates = expandYearlyByDay(startTime, interval, byDay, untilTime, to, count)
		}
	case "DAILY":
		// DAILY with BY* params: generate daily candidates, filter by params
		for i := 0; generated < maxExpansions; i++ {
			if count > 0 && generated >= count {
				break
			}
			instStart := startTime.AddDate(0, 0, i*interval)
			if !untilTime.IsZero() && instStart.After(untilTime) {
				break
			}
			if instStart.Compare(to) >= 0 {
				break
			}
			if matchesByParams(instStart, byDay, byMonthDay, byMonth) {
				candidates = append(candidates, instStart)
				generated++
			}
			if i > maxExpansions*10 {
				break // safety limit
			}
		}
	}

	// Fallback: if no BY* matched the freq, use simple expansion
	if candidates == nil {
		return expandSimple(startTime, freq, interval, untilTime, to, count)
	}
	return candidates
}

func expandWeeklyByDay(startTime time.Time, interval int, byDay []byDayEntry, untilTime, to time.Time, count int) []time.Time {
	var candidates []time.Time
	generated := 0

	// Find the start of the week containing startTime (use Monday as week start for RFC 5545)
	weekStart := startTime
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}

	for weekIdx := 0; generated < maxExpansions; weekIdx++ {
		currentWeekStart := weekStart.AddDate(0, 0, weekIdx*interval*7)
		if currentWeekStart.AddDate(0, 0, 7).Before(startTime) && weekIdx == 0 {
			continue
		}

		for _, entry := range byDay {
			dayOffset := int(entry.Weekday) - int(time.Monday)
			if dayOffset < 0 {
				dayOffset += 7
			}
			instStart := time.Date(currentWeekStart.Year(), currentWeekStart.Month(), currentWeekStart.Day()+dayOffset,
				startTime.Hour(), startTime.Minute(), startTime.Second(), 0, startTime.Location())

			if instStart.Before(startTime) {
				continue
			}
			if !untilTime.IsZero() && instStart.After(untilTime) {
				return candidates
			}
			if instStart.Compare(to) >= 0 {
				return candidates
			}
			if count > 0 && generated >= count {
				return candidates
			}

			candidates = append(candidates, instStart)
			generated++
		}

		if weekIdx > maxExpansions {
			break
		}
	}
	return candidates
}

func expandMonthlyByDay(startTime time.Time, interval int, byDay []byDayEntry, untilTime, to time.Time, count int) []time.Time {
	var candidates []time.Time
	generated := 0

	for monthIdx := 0; generated < maxExpansions; monthIdx++ {
		// Use first-of-month arithmetic to avoid day-overflow issues
		baseMonth := time.Date(startTime.Year(), startTime.Month(), 1, 0, 0, 0, 0, startTime.Location())
		currentMonth := baseMonth.AddDate(0, monthIdx*interval, 0)
		year, month := currentMonth.Year(), currentMonth.Month()

		for _, entry := range byDay {
			if entry.Offset != 0 {
				// Nth weekday of month (e.g., 2nd Monday, last Friday)
				t, ok := nthWeekdayOfMonth(year, month, entry.Weekday, entry.Offset)
				if !ok {
					continue
				}
				instStart := time.Date(year, month, t.Day(),
					startTime.Hour(), startTime.Minute(), startTime.Second(), 0, startTime.Location())
				if instStart.Before(startTime) {
					continue
				}
				if !untilTime.IsZero() && instStart.After(untilTime) {
					return candidates
				}
				if instStart.Compare(to) >= 0 {
					return candidates
				}
				if count > 0 && generated >= count {
					return candidates
				}
				candidates = append(candidates, instStart)
				generated++
			} else {
				// Every occurrence of this weekday in the month
				first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
				diff := int(entry.Weekday) - int(first.Weekday())
				if diff < 0 {
					diff += 7
				}
				for day := 1 + diff; day <= daysInMonth(year, month); day += 7 {
					instStart := time.Date(year, month, day,
						startTime.Hour(), startTime.Minute(), startTime.Second(), 0, startTime.Location())
					if instStart.Before(startTime) {
						continue
					}
					if !untilTime.IsZero() && instStart.After(untilTime) {
						return candidates
					}
					if instStart.Compare(to) >= 0 {
						return candidates
					}
					if count > 0 && generated >= count {
						return candidates
					}
					candidates = append(candidates, instStart)
					generated++
				}
			}
		}

		if monthIdx > maxExpansions {
			break
		}
	}
	return candidates
}

func expandMonthlyByMonthDay(startTime time.Time, interval int, byMonthDay []int, untilTime, to time.Time, count int) []time.Time {
	var candidates []time.Time
	generated := 0

	for monthIdx := 0; generated < maxExpansions; monthIdx++ {
		baseMonth := time.Date(startTime.Year(), startTime.Month(), 1, 0, 0, 0, 0, startTime.Location())
		currentMonth := baseMonth.AddDate(0, monthIdx*interval, 0)
		year, month := currentMonth.Year(), currentMonth.Month()
		maxDay := daysInMonth(year, month)

		for _, day := range byMonthDay {
			actualDay := day
			if day < 0 {
				actualDay = maxDay + day + 1
			}
			if actualDay < 1 || actualDay > maxDay {
				continue
			}

			instStart := time.Date(year, month, actualDay,
				startTime.Hour(), startTime.Minute(), startTime.Second(), 0, startTime.Location())
			if instStart.Before(startTime) {
				continue
			}
			if !untilTime.IsZero() && instStart.After(untilTime) {
				return candidates
			}
			if instStart.Compare(to) >= 0 {
				return candidates
			}
			if count > 0 && generated >= count {
				return candidates
			}
			candidates = append(candidates, instStart)
			generated++
		}

		if monthIdx > maxExpansions {
			break
		}
	}
	return candidates
}

func expandYearlyByMonth(startTime time.Time, interval int, byMonth []int, byMonthDay []int, untilTime, to time.Time, count int) []time.Time {
	var candidates []time.Time
	generated := 0

	for yearIdx := 0; generated < maxExpansions; yearIdx++ {
		year := startTime.Year() + yearIdx*interval

		for _, m := range byMonth {
			if m < 1 || m > 12 {
				continue
			}
			month := time.Month(m)
			maxDay := daysInMonth(year, month)

			days := byMonthDay
			if len(days) == 0 {
				days = []int{startTime.Day()}
			}

			for _, day := range days {
				actualDay := day
				if day < 0 {
					actualDay = maxDay + day + 1
				}
				if actualDay < 1 || actualDay > maxDay {
					continue
				}

				instStart := time.Date(year, month, actualDay,
					startTime.Hour(), startTime.Minute(), startTime.Second(), 0, startTime.Location())
				if instStart.Before(startTime) {
					continue
				}
				if !untilTime.IsZero() && instStart.After(untilTime) {
					return candidates
				}
				if instStart.Compare(to) >= 0 {
					return candidates
				}
				if count > 0 && generated >= count {
					return candidates
				}
				candidates = append(candidates, instStart)
				generated++
			}
		}

		if yearIdx > maxExpansions {
			break
		}
	}
	return candidates
}

func expandYearlyByDay(startTime time.Time, interval int, byDay []byDayEntry, untilTime, to time.Time, count int) []time.Time {
	var candidates []time.Time
	generated := 0

	for yearIdx := 0; generated < maxExpansions; yearIdx++ {
		year := startTime.Year() + yearIdx*interval
		month := startTime.Month()

		for _, entry := range byDay {
			if entry.Offset != 0 {
				t, ok := nthWeekdayOfMonth(year, month, entry.Weekday, entry.Offset)
				if !ok {
					continue
				}
				instStart := time.Date(year, month, t.Day(),
					startTime.Hour(), startTime.Minute(), startTime.Second(), 0, startTime.Location())
				if instStart.Before(startTime) {
					continue
				}
				if !untilTime.IsZero() && instStart.After(untilTime) {
					return candidates
				}
				if instStart.Compare(to) >= 0 {
					return candidates
				}
				if count > 0 && generated >= count {
					return candidates
				}
				candidates = append(candidates, instStart)
				generated++
			}
		}

		if yearIdx > maxExpansions {
			break
		}
	}
	return candidates
}

func matchesByParams(t time.Time, byDay []byDayEntry, byMonthDay []int, byMonth []int) bool {
	if len(byMonth) > 0 {
		found := false
		for _, m := range byMonth {
			if int(t.Month()) == m {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(byMonthDay) > 0 {
		found := false
		for _, d := range byMonthDay {
			if t.Day() == d {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(byDay) > 0 {
		found := false
		for _, entry := range byDay {
			if entry.Offset == 0 && t.Weekday() == entry.Weekday {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func addFreq(t time.Time, freq string, n int) time.Time {
	switch freq {
	case "DAILY":
		return t.AddDate(0, 0, n)
	case "WEEKLY":
		return t.AddDate(0, 0, n*7)
	case "MONTHLY":
		return t.AddDate(0, n, 0)
	case "YEARLY":
		return t.AddDate(n, 0, 0)
	default:
		return t
	}
}

func mergeEvents(a, b []model.Event) []model.Event {
	result := append(a, b...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartTime < result[j].StartTime
	})
	return result
}

// FormatByDay returns a human-readable description of BYDAY value.
func FormatByDay(byDay string) string {
	entries := parseByDay(byDay)
	if len(entries) == 0 {
		return ""
	}
	dayNames := map[time.Weekday]string{
		time.Sunday: "Sun", time.Monday: "Mon", time.Tuesday: "Tue",
		time.Wednesday: "Wed", time.Thursday: "Thu", time.Friday: "Fri", time.Saturday: "Sat",
	}
	var parts []string
	for _, e := range entries {
		name := dayNames[e.Weekday]
		if e.Offset != 0 {
			parts = append(parts, fmt.Sprintf("%s %s", ordinal(e.Offset), name))
		} else {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, ", ")
}

func ordinal(n int) string {
	if n < 0 {
		return "last"
	}
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		return fmt.Sprintf("%dth", n)
	}
}
