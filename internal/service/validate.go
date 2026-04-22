package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mikaelstaldal/mycal/internal/api"
	"github.com/mikaelstaldal/mycal/internal/model"
)

const (
	maxTitleLength        = 500
	maxDescriptionLength  = 10000
	maxLocationLength     = 500
	maxCategoriesLength   = 500
	maxURLLength          = 2000
	maxReminderMinutes    = 40320 // 4 weeks
	maxRecurrenceCount    = 1000
	maxRecurrenceInterval = 999
	maxRecurrenceListLen  = 5000
	maxEventDuration      = 366 * 24 * time.Hour
	minYear               = 1970
	maxYearOffset         = 100

	defaultRefreshIntervalMinutes = 60
	minRefreshIntervalMinutes     = 5
	maxRefreshIntervalMinutes     = 10080 // 1 week
	maxFeedURLLength              = 2000
)

var validFreqs = map[string]bool{
	"":        true,
	"DAILY":   true,
	"WEEKLY":  true,
	"MONTHLY": true,
	"YEARLY":  true,
}

// ValidateCreateEventRequest validates a create event request.
// Returns normalized RFC 3339 start and end times on success.
func ValidateCreateEventRequest(req *api.CreateEventRequest) (startTime, endTime string, err error) {
	if req.Title == "" {
		return "", "", fmt.Errorf("title is required")
	}
	if len(req.Title) > maxTitleLength {
		return "", "", fmt.Errorf("title must be at most %d characters", maxTitleLength)
	}
	if req.Description.Set && len(req.Description.Value) > maxDescriptionLength {
		return "", "", fmt.Errorf("description must be at most %d characters", maxDescriptionLength)
	}
	if req.Location.Set && len(req.Location.Value) > maxLocationLength {
		return "", "", fmt.Errorf("location must be at most %d characters", maxLocationLength)
	}
	if req.Categories.Set && len(req.Categories.Value) > maxCategoriesLength {
		return "", "", fmt.Errorf("categories must be at most %d characters", maxCategoriesLength)
	}
	if req.URL.Set {
		if err := validateURL(req.URL.Value.String()); err != nil {
			return "", "", err
		}
	}
	if req.Color.Set {
		if err := model.ValidateColor(req.Color.Value); err != nil {
			return "", "", err
		}
	}

	if req.AllDay {
		if !req.StartDate.Set {
			return "", "", fmt.Errorf("start_date is required for all-day events")
		}
		start := req.StartDate.Value
		if err := validateDateRange(start); err != nil {
			return "", "", err
		}
		startDate := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)

		var end time.Time
		if req.Duration.Set && req.Duration.Value != "" {
			if req.EndDate.Set {
				return "", "", fmt.Errorf("cannot specify both duration and end_date")
			}
			dur, err := model.ParseDuration(req.Duration.Value)
			if err != nil {
				return "", "", fmt.Errorf("invalid duration: %s", err.Error())
			}
			end = startDate.Add(dur)
		} else if !req.EndDate.Set {
			end = startDate.AddDate(0, 0, 1)
		} else {
			endVal := req.EndDate.Value
			endDate := time.Date(endVal.Year(), endVal.Month(), endVal.Day(), 0, 0, 0, 0, time.UTC)
			if endDate.Before(startDate) {
				return "", "", fmt.Errorf("end_date must not be before start_date")
			} else if endDate.Equal(startDate) {
				end = startDate.AddDate(0, 0, 1)
			} else {
				if err := validateDateRange(endDate); err != nil {
					return "", "", err
				}
				if endDate.Sub(startDate) > maxEventDuration {
					return "", "", fmt.Errorf("event duration must not exceed 366 days")
				}
				end = endDate
			}
		}

		startTime = startDate.Format(time.RFC3339)
		endTime = end.Format(time.RFC3339)
	} else {
		if !req.StartTime.Set {
			return "", "", fmt.Errorf("start_time is required")
		}
		start := req.StartTime.Value
		if err := validateDateRange(start); err != nil {
			return "", "", err
		}

		if req.Duration.Set && req.Duration.Value != "" {
			if req.EndTime.Set {
				return "", "", fmt.Errorf("cannot specify both duration and end_time")
			}
			dur, err := model.ParseDuration(req.Duration.Value)
			if err != nil {
				return "", "", fmt.Errorf("invalid duration: %s", err.Error())
			}
			endTime = start.Add(dur).Format(time.RFC3339)
		} else if !req.EndTime.Set {
			return "", "", fmt.Errorf("end_time is required")
		} else {
			end := req.EndTime.Value
			if err := validateDateRange(end); err != nil {
				return "", "", err
			}
			if !end.After(start) {
				return "", "", fmt.Errorf("end_time must be after start_time")
			}
			if end.Sub(start) > maxEventDuration {
				return "", "", fmt.Errorf("event duration must not exceed 366 days")
			}
			endTime = end.Format(time.RFC3339)
		}
		startTime = start.Format(time.RFC3339)
	}

	freq := ""
	if req.RecurrenceFreq.Set {
		freq = string(req.RecurrenceFreq.Value)
	}
	count := 0
	if req.RecurrenceCount.Set {
		count = req.RecurrenceCount.Value
	}
	until := ""
	if req.RecurrenceUntil.Set {
		until = req.RecurrenceUntil.Value
	}
	interval := 0
	if req.RecurrenceInterval.Set {
		interval = req.RecurrenceInterval.Value
	}
	byDay := req.RecurrenceByDay.Or("")
	byMonthDay := req.RecurrenceByMonthday.Or("")
	byMonth := req.RecurrenceByMonth.Or("")
	exDates := req.Exdates.Or("")
	rDates := req.Rdates.Or("")

	if !validFreqs[freq] {
		return "", "", fmt.Errorf("recurrence_freq must be one of: DAILY, WEEKLY, MONTHLY, YEARLY")
	}
	if count < 0 {
		return "", "", fmt.Errorf("recurrence_count must be >= 0")
	}
	if count > maxRecurrenceCount {
		return "", "", fmt.Errorf("recurrence_count must be at most %d", maxRecurrenceCount)
	}
	if until != "" {
		if _, err := time.Parse(time.RFC3339, until); err != nil {
			return "", "", fmt.Errorf("recurrence_until must be RFC 3339 format")
		}
	}
	if err := validateRecurrenceFields(freq, count, until, interval, byDay, byMonthDay, byMonth, exDates, rDates); err != nil {
		return "", "", err
	}

	if req.ReminderMinutes.Set {
		if req.ReminderMinutes.Value < 0 {
			return "", "", fmt.Errorf("reminder_minutes must be >= 0")
		}
		if req.ReminderMinutes.Value > maxReminderMinutes {
			return "", "", fmt.Errorf("reminder_minutes must be at most %d", maxReminderMinutes)
		}
	}

	var lat, lon *float64
	if req.Latitude.Set && !req.Latitude.Null {
		v := req.Latitude.Value
		lat = &v
	}
	if req.Longitude.Set && !req.Longitude.Null {
		v := req.Longitude.Value
		lon = &v
	}
	if err := validateCoordinates(lat, lon); err != nil {
		return "", "", err
	}

	return startTime, endTime, nil
}

// ValidateUpdateEventRequest validates an update event request.
func ValidateUpdateEventRequest(req *api.UpdateEventRequest) error {
	if req.Title.Set {
		if req.Title.Value == "" {
			return fmt.Errorf("title cannot be empty")
		}
		if len(req.Title.Value) > maxTitleLength {
			return fmt.Errorf("title must be at most %d characters", maxTitleLength)
		}
	}
	if req.Description.Set && len(req.Description.Value) > maxDescriptionLength {
		return fmt.Errorf("description must be at most %d characters", maxDescriptionLength)
	}
	if req.Location.Set && len(req.Location.Value) > maxLocationLength {
		return fmt.Errorf("location must be at most %d characters", maxLocationLength)
	}
	if req.Categories.Set && len(req.Categories.Value) > maxCategoriesLength {
		return fmt.Errorf("categories must be at most %d characters", maxCategoriesLength)
	}
	if req.URL.Set {
		if err := validateURL(req.URL.Value.String()); err != nil {
			return err
		}
	}
	if req.Color.Set {
		if err := model.ValidateColor(req.Color.Value); err != nil {
			return err
		}
	}
	if req.Duration.Set && req.Duration.Value != "" {
		if _, err := model.ParseDuration(req.Duration.Value); err != nil {
			return fmt.Errorf("invalid duration: %s", err.Error())
		}
	}

	if req.StartTime.Set && req.EndTime.Set {
		if !req.EndTime.Value.After(req.StartTime.Value) {
			return fmt.Errorf("end_time must be after start_time")
		}
		if req.EndTime.Value.Sub(req.StartTime.Value) > maxEventDuration {
			return fmt.Errorf("event duration must not exceed 366 days")
		}
	}

	if req.RecurrenceFreq.Set && !validFreqs[string(req.RecurrenceFreq.Value)] {
		return fmt.Errorf("recurrence_freq must be one of: DAILY, WEEKLY, MONTHLY, YEARLY")
	}
	if req.RecurrenceCount.Set {
		if req.RecurrenceCount.Value < 0 {
			return fmt.Errorf("recurrence_count must be >= 0")
		}
		if req.RecurrenceCount.Value > maxRecurrenceCount {
			return fmt.Errorf("recurrence_count must be at most %d", maxRecurrenceCount)
		}
	}
	if req.RecurrenceUntil.Set && req.RecurrenceUntil.Value != "" {
		if _, err := time.Parse(time.RFC3339, req.RecurrenceUntil.Value); err != nil {
			return fmt.Errorf("recurrence_until must be RFC 3339 format")
		}
	}
	if req.RecurrenceInterval.Set {
		if req.RecurrenceInterval.Value < 0 {
			return fmt.Errorf("recurrence_interval must be >= 0")
		}
		if req.RecurrenceInterval.Value > maxRecurrenceInterval {
			return fmt.Errorf("recurrence_interval must be at most %d", maxRecurrenceInterval)
		}
	}
	if req.RecurrenceByDay.Set {
		if err := validateByDay(req.RecurrenceByDay.Value); err != nil {
			return err
		}
	}
	if req.RecurrenceByMonthday.Set {
		if err := validateByMonthDay(req.RecurrenceByMonthday.Value); err != nil {
			return err
		}
	}
	if req.RecurrenceByMonth.Set {
		if err := validateByMonth(req.RecurrenceByMonth.Value); err != nil {
			return err
		}
	}
	if req.Exdates.Set {
		if err := validateDateList(req.Exdates.Value, "exdates"); err != nil {
			return err
		}
	}
	if req.Rdates.Set {
		if err := validateDateList(req.Rdates.Value, "rdates"); err != nil {
			return err
		}
	}
	if req.ReminderMinutes.Set {
		if req.ReminderMinutes.Value < 0 {
			return fmt.Errorf("reminder_minutes must be >= 0")
		}
		if req.ReminderMinutes.Value > maxReminderMinutes {
			return fmt.Errorf("reminder_minutes must be at most %d", maxReminderMinutes)
		}
	}

	var lat, lon *float64
	if req.Latitude.Set && !req.Latitude.Null {
		v := req.Latitude.Value
		lat = &v
	}
	if req.Longitude.Set && !req.Longitude.Null {
		v := req.Longitude.Value
		lon = &v
	}
	return validateCoordinates(lat, lon)
}

// ValidateCreateFeedRequest validates a create feed request.
// Returns the effective refresh interval on success.
func ValidateCreateFeedRequest(req *api.CreateFeedRequest) (refreshInterval int, err error) {
	rawURL := req.URL.String()
	if rawURL == "" || rawURL == "<nil>" {
		return 0, fmt.Errorf("url is required")
	}
	if len(rawURL) > maxFeedURLLength {
		return 0, fmt.Errorf("url must be at most %d characters", maxFeedURLLength)
	}
	if req.CalendarName.Set && len(req.CalendarName.Value) > model.MaxCalendarNameLength {
		return 0, fmt.Errorf("calendar_name must be at most %d characters", model.MaxCalendarNameLength)
	}
	if req.CalendarColor.Set {
		if err := model.ValidateColor(req.CalendarColor.Value); err != nil {
			return 0, err
		}
	}

	refreshInterval = defaultRefreshIntervalMinutes
	if req.RefreshIntervalMinutes.Set {
		refreshInterval = req.RefreshIntervalMinutes.Value
	}
	if refreshInterval < minRefreshIntervalMinutes {
		return 0, fmt.Errorf("refresh_interval_minutes must be at least %d", minRefreshIntervalMinutes)
	}
	if refreshInterval > maxRefreshIntervalMinutes {
		return 0, fmt.Errorf("refresh_interval_minutes must be at most %d", maxRefreshIntervalMinutes)
	}
	return refreshInterval, nil
}

// ValidateUpdateFeedRequest validates an update feed request.
func ValidateUpdateFeedRequest(req *api.UpdateFeedRequest) error {
	if req.URL.Set {
		rawURL := req.URL.Value.String()
		if rawURL == "" {
			return fmt.Errorf("url cannot be empty")
		}
		if len(rawURL) > maxFeedURLLength {
			return fmt.Errorf("url must be at most %d characters", maxFeedURLLength)
		}
	}
	if req.CalendarName.Set && len(req.CalendarName.Value) > model.MaxCalendarNameLength {
		return fmt.Errorf("calendar_name must be at most %d characters", model.MaxCalendarNameLength)
	}
	if req.RefreshIntervalMinutes.Set {
		if req.RefreshIntervalMinutes.Value < minRefreshIntervalMinutes {
			return fmt.Errorf("refresh_interval_minutes must be at least %d", minRefreshIntervalMinutes)
		}
		if req.RefreshIntervalMinutes.Value > maxRefreshIntervalMinutes {
			return fmt.Errorf("refresh_interval_minutes must be at most %d", maxRefreshIntervalMinutes)
		}
	}
	return nil
}

// ---- private validation helpers ----

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

func validateDateList(s, fieldName string) error {
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

func validateCoordinates(latitude, longitude *float64) error {
	if latitude != nil && (*latitude < -90 || *latitude > 90) {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	if longitude != nil && (*longitude < -180 || *longitude > 180) {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	return nil
}
