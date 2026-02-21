package model

import (
	"fmt"
	"time"
)

type Event struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	AllDay          bool   `json:"all_day"`
	Color           string `json:"color"`
	RecurrenceFreq  string `json:"recurrence_freq"`
	RecurrenceCount int    `json:"recurrence_count"`
	RecurrenceIndex int    `json:"recurrence_index,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

func (e *Event) IsRecurring() bool {
	return e.RecurrenceFreq != ""
}

var validFreqs = map[string]bool{
	"":        true,
	"DAILY":   true,
	"WEEKLY":  true,
	"MONTHLY": true,
	"YEARLY":  true,
}

type CreateEventRequest struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	AllDay          bool   `json:"all_day"`
	Color           string `json:"color"`
	RecurrenceFreq  string `json:"recurrence_freq"`
	RecurrenceCount int    `json:"recurrence_count"`
}

const dateOnly = "2006-01-02"

func (r *CreateEventRequest) Validate() error {
	if r.Title == "" {
		return fmt.Errorf("title is required")
	}
	if r.StartTime == "" {
		return fmt.Errorf("start_time is required")
	}

	if r.AllDay {
		start, err := time.Parse(dateOnly, r.StartTime)
		if err != nil {
			return fmt.Errorf("start_time must be YYYY-MM-DD format for all-day events")
		}
		if r.EndTime == "" || r.EndTime == r.StartTime {
			// Default to single-day event (next day exclusive end)
			r.EndTime = start.AddDate(0, 0, 1).Format(dateOnly)
		} else {
			end, err := time.Parse(dateOnly, r.EndTime)
			if err != nil {
				return fmt.Errorf("end_time must be YYYY-MM-DD format for all-day events")
			}
			if end.Before(start) {
				return fmt.Errorf("end_time must not be before start_time")
			}
		}
		// Normalize to midnight UTC
		endDate, _ := time.Parse(dateOnly, r.EndTime)
		r.StartTime = start.UTC().Format(time.RFC3339)
		r.EndTime = endDate.UTC().Format(time.RFC3339)
		if !validFreqs[r.RecurrenceFreq] {
			return fmt.Errorf("recurrence_freq must be one of: DAILY, WEEKLY, MONTHLY, YEARLY")
		}
		if r.RecurrenceCount < 0 {
			return fmt.Errorf("recurrence_count must be >= 0")
		}
		return nil
	}

	if r.EndTime == "" {
		return fmt.Errorf("end_time is required")
	}
	start, err := time.Parse(time.RFC3339, r.StartTime)
	if err != nil {
		return fmt.Errorf("start_time must be RFC 3339 format")
	}
	end, err := time.Parse(time.RFC3339, r.EndTime)
	if err != nil {
		return fmt.Errorf("end_time must be RFC 3339 format")
	}
	if !end.After(start) {
		return fmt.Errorf("end_time must be after start_time")
	}
	if !validFreqs[r.RecurrenceFreq] {
		return fmt.Errorf("recurrence_freq must be one of: DAILY, WEEKLY, MONTHLY, YEARLY")
	}
	if r.RecurrenceCount < 0 {
		return fmt.Errorf("recurrence_count must be >= 0")
	}
	return nil
}

type UpdateEventRequest struct {
	Title           *string `json:"title"`
	Description     *string `json:"description"`
	StartTime       *string `json:"start_time"`
	EndTime         *string `json:"end_time"`
	AllDay          *bool   `json:"all_day"`
	Color           *string `json:"color"`
	RecurrenceFreq  *string `json:"recurrence_freq"`
	RecurrenceCount *int    `json:"recurrence_count"`
}

func (r *UpdateEventRequest) Validate() error {
	if r.Title != nil && *r.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}

	allDay := r.AllDay != nil && *r.AllDay

	if allDay {
		if r.StartTime != nil {
			if _, err := time.Parse(dateOnly, *r.StartTime); err != nil {
				return fmt.Errorf("start_time must be YYYY-MM-DD format for all-day events")
			}
		}
		if r.EndTime != nil {
			if _, err := time.Parse(dateOnly, *r.EndTime); err != nil {
				return fmt.Errorf("end_time must be YYYY-MM-DD format for all-day events")
			}
		}
		return nil
	}

	// If not setting all_day, and all_day pointer is nil, we don't know yet
	// whether event is all-day; service layer will do final validation
	start := ""
	end := ""
	if r.StartTime != nil {
		if _, err := time.Parse(time.RFC3339, *r.StartTime); err != nil {
			// Might be date-only if toggling to all-day
			if _, err2 := time.Parse(dateOnly, *r.StartTime); err2 != nil {
				return fmt.Errorf("start_time must be RFC 3339 format")
			}
		}
		start = *r.StartTime
	}
	if r.EndTime != nil {
		if _, err := time.Parse(time.RFC3339, *r.EndTime); err != nil {
			if _, err2 := time.Parse(dateOnly, *r.EndTime); err2 != nil {
				return fmt.Errorf("end_time must be RFC 3339 format")
			}
		}
		end = *r.EndTime
	}
	if start != "" && end != "" {
		s, err1 := time.Parse(time.RFC3339, start)
		e, err2 := time.Parse(time.RFC3339, end)
		if err1 == nil && err2 == nil && !e.After(s) {
			return fmt.Errorf("end_time must be after start_time")
		}
	}
	if r.RecurrenceFreq != nil && !validFreqs[*r.RecurrenceFreq] {
		return fmt.Errorf("recurrence_freq must be one of: DAILY, WEEKLY, MONTHLY, YEARLY")
	}
	if r.RecurrenceCount != nil && *r.RecurrenceCount < 0 {
		return fmt.Errorf("recurrence_count must be >= 0")
	}
	return nil
}
