package ical

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mikaelstaldal/mycal/internal/model"
)

// Encode writes events as an iCalendar (RFC 5545) document to w.
func Encode(w io.Writer, events []model.Event) error {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//mycal//mycal//EN\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("METHOD:PUBLISH\r\n")
	b.WriteString("X-WR-CALNAME:mycal\r\n")

	for _, e := range events {
		start, err := time.Parse(time.RFC3339, e.StartTime)
		if err != nil {
			continue
		}
		end, err := time.Parse(time.RFC3339, e.EndTime)
		if err != nil {
			continue
		}

		b.WriteString("BEGIN:VEVENT\r\n")
		b.WriteString(fmt.Sprintf("UID:event-%d@mycal\r\n", e.ID))
		if e.AllDay {
			b.WriteString(fmt.Sprintf("DTSTART;VALUE=DATE:%s\r\n", start.UTC().Format("20060102")))
			b.WriteString(fmt.Sprintf("DTEND;VALUE=DATE:%s\r\n", end.UTC().Format("20060102")))
		} else {
			b.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatICalTime(start)))
			b.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatICalTime(end)))
		}
		b.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeText(e.Title)))
		if e.Description != "" {
			b.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeText(e.Description)))
		}
		if e.RecurrenceFreq != "" {
			rrule := "RRULE:FREQ=" + e.RecurrenceFreq
			if e.RecurrenceCount > 0 {
				rrule += fmt.Sprintf(";COUNT=%d", e.RecurrenceCount)
			}
			b.WriteString(rrule + "\r\n")
		}
		if e.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, e.CreatedAt); err == nil {
				b.WriteString(fmt.Sprintf("CREATED:%s\r\n", formatICalTime(t)))
			}
		}
		if e.UpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339, e.UpdatedAt); err == nil {
				b.WriteString(fmt.Sprintf("LAST-MODIFIED:%s\r\n", formatICalTime(t)))
				b.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICalTime(t)))
			}
		}
		b.WriteString("END:VEVENT\r\n")
	}

	b.WriteString("END:VCALENDAR\r\n")
	_, err := io.WriteString(w, b.String())
	return err
}

func formatICalTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// Decode parses an iCalendar document and returns the events found.
func Decode(r io.Reader) ([]model.Event, error) {
	lines, err := unfoldLines(r)
	if err != nil {
		return nil, fmt.Errorf("reading ical: %w", err)
	}

	var events []model.Event
	var inEvent bool
	var props []string

	for _, line := range lines {
		upper := strings.ToUpper(strings.TrimSpace(line))
		if upper == "BEGIN:VEVENT" {
			inEvent = true
			props = nil
			continue
		}
		if upper == "END:VEVENT" {
			inEvent = false
			if ev, ok := parseEvent(props); ok {
				events = append(events, ev)
			}
			continue
		}
		if inEvent {
			props = append(props, line)
		}
	}

	return events, nil
}

// unfoldLines reads iCal content and handles line unfolding per RFC 5545:
// lines that start with a space or tab are continuations of the previous line.
func unfoldLines(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		// Remove trailing \r if present
		line = strings.TrimRight(line, "\r")
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			// Continuation line: append to previous
			if len(lines) > 0 {
				lines[len(lines)-1] += line[1:]
			}
		} else {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

func parseEvent(props []string) (model.Event, bool) {
	var summary, description, dtstart, dtend string
	var recurrenceFreq string
	var recurrenceCount int
	allDay := false

	for _, prop := range props {
		name, params, value := parsePropLine(prop)
		switch strings.ToUpper(name) {
		case "SUMMARY":
			summary = unescapeText(value)
		case "DESCRIPTION":
			description = unescapeText(value)
		case "DTSTART":
			upperParams := strings.ToUpper(params)
			if strings.Contains(upperParams, "VALUE=DATE") {
				allDay = true
			}
			dtstart = parseICalTime(value, params)
		case "DTEND":
			dtend = parseICalTime(value, params)
		case "RRULE":
			recurrenceFreq, recurrenceCount = parseRRule(value)
		}
	}

	if summary == "" || dtstart == "" || dtend == "" {
		return model.Event{}, false
	}

	return model.Event{
		Title:           summary,
		Description:     description,
		StartTime:       dtstart,
		EndTime:         dtend,
		AllDay:          allDay,
		RecurrenceFreq:  recurrenceFreq,
		RecurrenceCount: recurrenceCount,
	}, true
}

func parseRRule(value string) (freq string, count int) {
	for _, part := range strings.Split(value, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.ToUpper(kv[0]) {
		case "FREQ":
			freq = strings.ToUpper(kv[1])
		case "COUNT":
			fmt.Sscanf(kv[1], "%d", &count)
		}
	}
	return
}

// parsePropLine splits "DTSTART;TZID=Europe/Stockholm:20060102T150405" into
// name="DTSTART", params="TZID=Europe/Stockholm", value="20060102T150405".
func parsePropLine(line string) (name, params, value string) {
	// Split at first colon that's not inside params
	colonIdx := strings.Index(line, ":")
	semiIdx := strings.Index(line, ";")

	if colonIdx < 0 {
		return line, "", ""
	}

	nameAndParams := line[:colonIdx]
	value = line[colonIdx+1:]

	if semiIdx >= 0 && semiIdx < colonIdx {
		name = line[:semiIdx]
		params = line[semiIdx+1 : colonIdx]
	} else {
		name = nameAndParams
	}
	return name, params, value
}

func parseICalTime(value, params string) string {
	// Check for VALUE=DATE (all-day event)
	upperParams := strings.ToUpper(params)
	if strings.Contains(upperParams, "VALUE=DATE") {
		t, err := time.Parse("20060102", value)
		if err != nil {
			return ""
		}
		return t.UTC().Format(time.RFC3339)
	}

	// Check for TZID
	var loc *time.Location
	for _, part := range strings.Split(params, ";") {
		if strings.HasPrefix(strings.ToUpper(part), "TZID=") {
			tzName := part[5:]
			if l, err := time.LoadLocation(tzName); err == nil {
				loc = l
			}
		}
	}

	// Try UTC format: 20060102T150405Z
	if t, err := time.Parse("20060102T150405Z", value); err == nil {
		return t.UTC().Format(time.RFC3339)
	}

	// Try local format with TZID: 20060102T150405
	if t, err := time.Parse("20060102T150405", value); err == nil {
		if loc != nil {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
		}
		return t.UTC().Format(time.RFC3339)
	}

	// Try date-only without VALUE=DATE param
	if t, err := time.Parse("20060102", value); err == nil {
		return t.UTC().Format(time.RFC3339)
	}

	return ""
}

func unescapeText(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\N", "\n")
	s = strings.ReplaceAll(s, "\\,", ",")
	s = strings.ReplaceAll(s, "\\;", ";")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

func escapeText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
