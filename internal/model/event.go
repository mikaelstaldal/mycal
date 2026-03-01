package model

import (
	"fmt"
	"strconv"
	"strings"
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
	RecurrenceParentID    *int64 `json:"recurrence_parent_id,omitempty"`
	RecurrenceOriginalStart string `json:"recurrence_original_start,omitempty"`
	Duration        string   `json:"duration,omitempty"`
	Categories      string   `json:"categories,omitempty"`
	URL             string   `json:"url,omitempty"`
	ReminderMinutes int      `json:"reminder_minutes"`
	Location        string   `json:"location"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
	ImportUID       string   `json:"-"` // transient field for iCal import UID matching
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
	Duration           string   `json:"duration"`
	Categories         string   `json:"categories"`
	URL                string   `json:"url"`
	ReminderMinutes    int      `json:"reminder_minutes"`
	Location           string   `json:"location"`
	Latitude           *float64 `json:"latitude"`
	Longitude          *float64 `json:"longitude"`
}

const dateOnly = "2006-01-02"

const (
	maxTitleLength         = 500
	maxDescriptionLength   = 10000
	maxLocationLength      = 500
	maxCategoriesLength    = 500
	maxURLLength           = 2000
	maxReminderMinutes     = 40320 // 4 weeks
	maxRecurrenceCount     = 1000
	maxRecurrenceInterval  = 999
	maxRecurrenceListLen   = 5000 // max length for comma-separated recurrence fields
	maxEventDuration       = 366 * 24 * time.Hour // 366 days
	minYear                = 1970
	maxYearOffset          = 100 // max year is current year + this offset
)

var validWeekdays = map[string]bool{
	"SU": true, "MO": true, "TU": true, "WE": true,
	"TH": true, "FR": true, "SA": true,
}

func validateByDay(s string) error {
	if s == "" {
		return nil
	}
	if len(s) > maxRecurrenceListLen {
		return fmt.Errorf("recurrence_by_day must be at most %d characters", maxRecurrenceListLen)
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if len(part) < 2 {
			return fmt.Errorf("recurrence_by_day contains invalid entry: %q", part)
		}
		dayAbbr := part[len(part)-2:]
		if !validWeekdays[dayAbbr] {
			return fmt.Errorf("recurrence_by_day contains invalid weekday: %q", dayAbbr)
		}
		if len(part) > 2 {
			offsetStr := part[:len(part)-2]
			offset, err := strconv.Atoi(offsetStr)
			if err != nil {
				return fmt.Errorf("recurrence_by_day contains invalid offset: %q", offsetStr)
			}
			if offset == 0 || offset < -53 || offset > 53 {
				return fmt.Errorf("recurrence_by_day offset must be between -53 and 53, not zero")
			}
		}
	}
	return nil
}

func validateByMonthDay(s string) error {
	if s == "" {
		return nil
	}
	if len(s) > maxRecurrenceListLen {
		return fmt.Errorf("recurrence_by_monthday must be at most %d characters", maxRecurrenceListLen)
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err != nil {
			return fmt.Errorf("recurrence_by_monthday contains invalid number: %q", part)
		}
		if n == 0 || n < -31 || n > 31 {
			return fmt.Errorf("recurrence_by_monthday values must be between -31 and 31, not zero")
		}
	}
	return nil
}

func validateByMonth(s string) error {
	if s == "" {
		return nil
	}
	if len(s) > maxRecurrenceListLen {
		return fmt.Errorf("recurrence_by_month must be at most %d characters", maxRecurrenceListLen)
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err != nil {
			return fmt.Errorf("recurrence_by_month contains invalid number: %q", part)
		}
		if n < 1 || n > 12 {
			return fmt.Errorf("recurrence_by_month values must be between 1 and 12")
		}
	}
	return nil
}

func validateDateList(s string, fieldName string) error {
	if s == "" {
		return nil
	}
	if len(s) > maxRecurrenceListLen {
		return fmt.Errorf("%s must be at most %d characters", fieldName, maxRecurrenceListLen)
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, err := time.Parse(time.RFC3339, part); err != nil {
			return fmt.Errorf("%s contains invalid RFC 3339 datetime: %q", fieldName, part)
		}
	}
	return nil
}

func validateRecurrenceFields(freq string, count int, until string, interval int, byDay, byMonthDay, byMonth, exDates, rDates string) error {
	if interval < 0 {
		return fmt.Errorf("recurrence_interval must be >= 0")
	}
	if interval > maxRecurrenceInterval {
		return fmt.Errorf("recurrence_interval must be at most %d", maxRecurrenceInterval)
	}
	if count > 0 && until != "" {
		return fmt.Errorf("recurrence_count and recurrence_until are mutually exclusive")
	}
	if freq == "" {
		if count > 0 || until != "" || interval > 0 || byDay != "" || byMonthDay != "" || byMonth != "" || exDates != "" || rDates != "" {
			return fmt.Errorf("recurrence fields require recurrence_freq to be set")
		}
		return nil
	}
	if err := validateByDay(byDay); err != nil {
		return err
	}
	if err := validateByMonthDay(byMonthDay); err != nil {
		return err
	}
	if err := validateByMonth(byMonth); err != nil {
		return err
	}
	if err := validateDateList(exDates, "exdates"); err != nil {
		return err
	}
	if err := validateDateList(rDates, "rdates"); err != nil {
		return err
	}
	return nil
}

func validateDateRange(t time.Time) error {
	maxYear := time.Now().Year() + maxYearOffset
	if t.Year() < minYear || t.Year() > maxYear {
		return fmt.Errorf("date must be between year %d and %d", minYear, maxYear)
	}
	return nil
}

// ParseDuration parses an ISO 8601 duration string like PT1H, PT30M, P1D, P1DT2H30M.
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	s = strings.ToUpper(s)
	if !strings.HasPrefix(s, "P") {
		return 0, fmt.Errorf("duration must start with P")
	}
	s = s[1:] // strip "P"

	var total time.Duration
	inTime := false

	num := ""
	for _, c := range s {
		if c == 'T' {
			inTime = true
			continue
		}
		if c >= '0' && c <= '9' {
			num += string(c)
			continue
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			return 0, fmt.Errorf("invalid duration number: %s", num)
		}
		num = ""

		if inTime {
			switch c {
			case 'H':
				total += time.Duration(n) * time.Hour
			case 'M':
				total += time.Duration(n) * time.Minute
			case 'S':
				total += time.Duration(n) * time.Second
			default:
				return 0, fmt.Errorf("unknown time unit: %c", c)
			}
		} else {
			switch c {
			case 'D':
				total += time.Duration(n) * 24 * time.Hour
			case 'W':
				total += time.Duration(n) * 7 * 24 * time.Hour
			default:
				return 0, fmt.Errorf("unknown date unit: %c", c)
			}
		}
	}

	if total <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}
	return total, nil
}

func validateURL(u string) error {
	if u == "" {
		return nil
	}
	if len(u) > maxURLLength {
		return fmt.Errorf("url must be at most %d characters", maxURLLength)
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return fmt.Errorf("url must start with http:// or https://")
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
	if len(r.Categories) > maxCategoriesLength {
		return fmt.Errorf("categories must be at most %d characters", maxCategoriesLength)
	}
	if err := validateURL(r.URL); err != nil {
		return err
	}
	if r.StartTime == "" {
		return fmt.Errorf("start_time is required")
	}

	// Duration and EndTime conflict
	if r.Duration != "" && r.EndTime != "" {
		return fmt.Errorf("cannot specify both duration and end_time")
	}

	// If Duration is set, compute EndTime
	if r.Duration != "" {
		dur, err := ParseDuration(r.Duration)
		if err != nil {
			return fmt.Errorf("invalid duration: %s", err.Error())
		}
		if r.AllDay {
			start, err := time.Parse(dateOnly, r.StartTime)
			if err != nil {
				return fmt.Errorf("start_time must be YYYY-MM-DD format for all-day events")
			}
			r.EndTime = start.Add(dur).Format(dateOnly)
		} else {
			start, err := time.Parse(time.RFC3339, r.StartTime)
			if err != nil {
				return fmt.Errorf("start_time must be RFC 3339 format")
			}
			r.EndTime = start.Add(dur).Format(time.RFC3339)
		}
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
		if err := r.validateRecurrence(); err != nil {
			return err
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
	if err := r.validateRecurrence(); err != nil {
		return err
	}
	if r.ReminderMinutes < 0 {
		return fmt.Errorf("reminder_minutes must be >= 0")
	}
	if r.ReminderMinutes > maxReminderMinutes {
		return fmt.Errorf("reminder_minutes must be at most %d", maxReminderMinutes)
	}
	return r.validateCoordinates()
}

func (r *CreateEventRequest) validateRecurrence() error {
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
	return validateRecurrenceFields(r.RecurrenceFreq, r.RecurrenceCount, r.RecurrenceUntil, r.RecurrenceInterval,
		r.RecurrenceByDay, r.RecurrenceByMonthDay, r.RecurrenceByMonth, r.ExDates, r.RDates)
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
	Duration           *string  `json:"duration"`
	Categories         *string  `json:"categories"`
	URL                *string  `json:"url"`
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
	if r.Categories != nil && len(*r.Categories) > maxCategoriesLength {
		return fmt.Errorf("categories must be at most %d characters", maxCategoriesLength)
	}
	if r.URL != nil {
		if err := validateURL(*r.URL); err != nil {
			return err
		}
	}
	if r.Duration != nil && *r.Duration != "" {
		if _, err := ParseDuration(*r.Duration); err != nil {
			return fmt.Errorf("invalid duration: %s", err.Error())
		}
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
	if r.RecurrenceUntil != nil && *r.RecurrenceUntil != "" {
		if _, err := time.Parse(time.RFC3339, *r.RecurrenceUntil); err != nil {
			return fmt.Errorf("recurrence_until must be RFC 3339 format")
		}
	}
	if r.RecurrenceInterval != nil {
		if *r.RecurrenceInterval < 0 {
			return fmt.Errorf("recurrence_interval must be >= 0")
		}
		if *r.RecurrenceInterval > maxRecurrenceInterval {
			return fmt.Errorf("recurrence_interval must be at most %d", maxRecurrenceInterval)
		}
	}
	if r.RecurrenceByDay != nil {
		if err := validateByDay(*r.RecurrenceByDay); err != nil {
			return err
		}
	}
	if r.RecurrenceByMonthDay != nil {
		if err := validateByMonthDay(*r.RecurrenceByMonthDay); err != nil {
			return err
		}
	}
	if r.RecurrenceByMonth != nil {
		if err := validateByMonth(*r.RecurrenceByMonth); err != nil {
			return err
		}
	}
	if r.ExDates != nil {
		if err := validateDateList(*r.ExDates, "exdates"); err != nil {
			return err
		}
	}
	if r.RDates != nil {
		if err := validateDateList(*r.RDates, "rdates"); err != nil {
			return err
		}
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
