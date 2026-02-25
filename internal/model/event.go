package model

import (
	"fmt"
	"time"
)

type Event struct {
	ID              int64    `json:"id"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	StartTime       string   `json:"start_time"`
	EndTime         string   `json:"end_time"`
	AllDay          bool     `json:"all_day"`
	Color           string   `json:"color"`
	RecurrenceFreq  string   `json:"recurrence_freq"`
	RecurrenceCount    int      `json:"recurrence_count"`
	RecurrenceUntil    string   `json:"recurrence_until"`
	RecurrenceInterval int      `json:"recurrence_interval"`
	RecurrenceByDay    string   `json:"recurrence_by_day"`
	RecurrenceByMonthDay string `json:"recurrence_by_monthday"`
	RecurrenceByMonth  string   `json:"recurrence_by_month"`
	ExDates            string   `json:"exdates"`
	RDates             string   `json:"rdates"`
	RecurrenceIndex    int      `json:"recurrence_index,omitempty"`
	ReminderMinutes int      `json:"reminder_minutes"`
	Location        string   `json:"location"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
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
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	StartTime       string   `json:"start_time"`
	EndTime         string   `json:"end_time"`
	AllDay          bool     `json:"all_day"`
	Color           string   `json:"color"`
	RecurrenceFreq     string   `json:"recurrence_freq"`
	RecurrenceCount    int      `json:"recurrence_count"`
	RecurrenceUntil    string   `json:"recurrence_until"`
	RecurrenceInterval int      `json:"recurrence_interval"`
	RecurrenceByDay    string   `json:"recurrence_by_day"`
	RecurrenceByMonthDay string `json:"recurrence_by_monthday"`
	RecurrenceByMonth  string   `json:"recurrence_by_month"`
	ExDates            string   `json:"exdates"`
	RDates             string   `json:"rdates"`
	ReminderMinutes    int      `json:"reminder_minutes"`
	Location           string   `json:"location"`
	Latitude           *float64 `json:"latitude"`
	Longitude          *float64 `json:"longitude"`
}

const dateOnly = "2006-01-02"

const (
	maxTitleLength       = 500
	maxDescriptionLength = 10000
	maxLocationLength    = 500
	maxReminderMinutes   = 40320 // 4 weeks
	maxRecurrenceCount   = 1000
	maxEventDuration     = 366 * 24 * time.Hour // 366 days
	minYear              = 1900
	maxYear              = 2200
)

func validateDateRange(t time.Time) error {
	if t.Year() < minYear || t.Year() > maxYear {
		return fmt.Errorf("date must be between year %d and %d", minYear, maxYear)
	}
	return nil
}

func (r *CreateEventRequest) Validate() error {
	if r.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(r.Title) > maxTitleLength {
		return fmt.Errorf("title must be at most %d characters", maxTitleLength)
	}
	if len(r.Description) > maxDescriptionLength {
		return fmt.Errorf("description must be at most %d characters", maxDescriptionLength)
	}
	if len(r.Location) > maxLocationLength {
		return fmt.Errorf("location must be at most %d characters", maxLocationLength)
	}
	if r.StartTime == "" {
		return fmt.Errorf("start_time is required")
	}

	if r.AllDay {
		start, err := time.Parse(dateOnly, r.StartTime)
		if err != nil {
			return fmt.Errorf("start_time must be YYYY-MM-DD format for all-day events")
		}
		if err := validateDateRange(start); err != nil {
			return err
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
			if err := validateDateRange(end); err != nil {
				return err
			}
			if end.Sub(start) > maxEventDuration {
				return fmt.Errorf("event duration must not exceed 366 days")
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
		if r.RecurrenceCount > maxRecurrenceCount {
			return fmt.Errorf("recurrence_count must be at most %d", maxRecurrenceCount)
		}
		if r.RecurrenceUntil != "" {
			if _, err := time.Parse(time.RFC3339, r.RecurrenceUntil); err != nil {
				return fmt.Errorf("recurrence_until must be RFC 3339 format")
			}
		}
		if r.ReminderMinutes < 0 {
			return fmt.Errorf("reminder_minutes must be >= 0")
		}
		if r.ReminderMinutes > maxReminderMinutes {
			return fmt.Errorf("reminder_minutes must be at most %d", maxReminderMinutes)
		}
		return r.validateCoordinates()
	}

	if r.EndTime == "" {
		return fmt.Errorf("end_time is required")
	}
	start, err := time.Parse(time.RFC3339, r.StartTime)
	if err != nil {
		return fmt.Errorf("start_time must be RFC 3339 format")
	}
	if err := validateDateRange(start); err != nil {
		return err
	}
	end, err := time.Parse(time.RFC3339, r.EndTime)
	if err != nil {
		return fmt.Errorf("end_time must be RFC 3339 format")
	}
	if err := validateDateRange(end); err != nil {
		return err
	}
	if !end.After(start) {
		return fmt.Errorf("end_time must be after start_time")
	}
	if end.Sub(start) > maxEventDuration {
		return fmt.Errorf("event duration must not exceed 366 days")
	}
	if !validFreqs[r.RecurrenceFreq] {
		return fmt.Errorf("recurrence_freq must be one of: DAILY, WEEKLY, MONTHLY, YEARLY")
	}
	if r.RecurrenceCount < 0 {
		return fmt.Errorf("recurrence_count must be >= 0")
	}
	if r.RecurrenceCount > maxRecurrenceCount {
		return fmt.Errorf("recurrence_count must be at most %d", maxRecurrenceCount)
	}
	if r.RecurrenceUntil != "" {
		if _, err := time.Parse(time.RFC3339, r.RecurrenceUntil); err != nil {
			return fmt.Errorf("recurrence_until must be RFC 3339 format")
		}
	}
	if r.ReminderMinutes < 0 {
		return fmt.Errorf("reminder_minutes must be >= 0")
	}
	if r.ReminderMinutes > maxReminderMinutes {
		return fmt.Errorf("reminder_minutes must be at most %d", maxReminderMinutes)
	}
	return r.validateCoordinates()
}

func (r *CreateEventRequest) validateCoordinates() error {
	if r.Latitude != nil && (*r.Latitude < -90 || *r.Latitude > 90) {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	if r.Longitude != nil && (*r.Longitude < -180 || *r.Longitude > 180) {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	return nil
}

type UpdateEventRequest struct {
	Title           *string  `json:"title"`
	Description     *string  `json:"description"`
	StartTime       *string  `json:"start_time"`
	EndTime         *string  `json:"end_time"`
	AllDay          *bool    `json:"all_day"`
	Color           *string  `json:"color"`
	RecurrenceFreq     *string  `json:"recurrence_freq"`
	RecurrenceCount    *int     `json:"recurrence_count"`
	RecurrenceUntil    *string  `json:"recurrence_until"`
	RecurrenceInterval *int     `json:"recurrence_interval"`
	RecurrenceByDay    *string  `json:"recurrence_by_day"`
	RecurrenceByMonthDay *string `json:"recurrence_by_monthday"`
	RecurrenceByMonth  *string  `json:"recurrence_by_month"`
	ExDates            *string  `json:"exdates"`
	RDates             *string  `json:"rdates"`
	ReminderMinutes    *int     `json:"reminder_minutes"`
	Location           *string  `json:"location"`
	Latitude           *float64 `json:"latitude"`
	Longitude          *float64 `json:"longitude"`
}

func (r *UpdateEventRequest) Validate() error {
	if r.Title != nil && *r.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if r.Title != nil && len(*r.Title) > maxTitleLength {
		return fmt.Errorf("title must be at most %d characters", maxTitleLength)
	}
	if r.Description != nil && len(*r.Description) > maxDescriptionLength {
		return fmt.Errorf("description must be at most %d characters", maxDescriptionLength)
	}
	if r.Location != nil && len(*r.Location) > maxLocationLength {
		return fmt.Errorf("location must be at most %d characters", maxLocationLength)
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
		if err1 == nil && err2 == nil && e.Sub(s) > maxEventDuration {
			return fmt.Errorf("event duration must not exceed 366 days")
		}
	}
	if r.RecurrenceFreq != nil && !validFreqs[*r.RecurrenceFreq] {
		return fmt.Errorf("recurrence_freq must be one of: DAILY, WEEKLY, MONTHLY, YEARLY")
	}
	if r.RecurrenceCount != nil && *r.RecurrenceCount < 0 {
		return fmt.Errorf("recurrence_count must be >= 0")
	}
	if r.RecurrenceCount != nil && *r.RecurrenceCount > maxRecurrenceCount {
		return fmt.Errorf("recurrence_count must be at most %d", maxRecurrenceCount)
	}
	if r.ReminderMinutes != nil && *r.ReminderMinutes < 0 {
		return fmt.Errorf("reminder_minutes must be >= 0")
	}
	if r.ReminderMinutes != nil && *r.ReminderMinutes > maxReminderMinutes {
		return fmt.Errorf("reminder_minutes must be at most %d", maxReminderMinutes)
	}
	if r.Latitude != nil && (*r.Latitude < -90 || *r.Latitude > 90) {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	if r.Longitude != nil && (*r.Longitude < -180 || *r.Longitude > 180) {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	return nil
}
