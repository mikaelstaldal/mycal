package ical

import (
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
		b.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatICalTime(start)))
		b.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatICalTime(end)))
		b.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeText(e.Title)))
		if e.Description != "" {
			b.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeText(e.Description)))
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

func escapeText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
