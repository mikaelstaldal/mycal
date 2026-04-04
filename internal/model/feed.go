package model

import (
	"fmt"
	"strconv"
)

type Feed struct {
	ID                     int64
	StringID               string
	URL                    string
	CalendarID             int64
	CalendarName           string
	RefreshIntervalMinutes int
	LastRefreshedAt        string
	LastError              string
	Enabled                bool
	CreatedAt              string
	UpdatedAt              string
}

func (f *Feed) SetStringID() {
	f.StringID = strconv.FormatInt(f.ID, 10)
}

func FormatFeedID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func ParseFeedID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

const (
	DefaultRefreshIntervalMinutes = 60
	maxRefreshIntervalMinutes     = 10080 // 1 week
	minRefreshIntervalMinutes     = 5
	maxFeedURLLength              = 2000
)

type CreateFeedRequest struct {
	URL                    string
	CalendarName           string
	CalendarColor          string
	RefreshIntervalMinutes int
}

func (r *CreateFeedRequest) Validate() error {
	if r.URL == "" {
		return fmt.Errorf("url is required")
	}
	if len(r.URL) > maxFeedURLLength {
		return fmt.Errorf("url must be at most %d characters", maxFeedURLLength)
	}
	if len(r.CalendarName) > MaxCalendarNameLength {
		return fmt.Errorf("calendar_name must be at most %d characters", MaxCalendarNameLength)
	}
	if err := ValidateColor(r.CalendarColor); err != nil {
		return err
	}
	if r.RefreshIntervalMinutes == 0 {
		r.RefreshIntervalMinutes = DefaultRefreshIntervalMinutes
	}
	if r.RefreshIntervalMinutes < minRefreshIntervalMinutes {
		return fmt.Errorf("refresh_interval_minutes must be at least %d", minRefreshIntervalMinutes)
	}
	if r.RefreshIntervalMinutes > maxRefreshIntervalMinutes {
		return fmt.Errorf("refresh_interval_minutes must be at most %d", maxRefreshIntervalMinutes)
	}
	return nil
}

type UpdateFeedRequest struct {
	URL                    *string
	CalendarName           *string
	RefreshIntervalMinutes *int
	Enabled                *bool
}

func (r *UpdateFeedRequest) Validate() error {
	if r.URL != nil {
		if *r.URL == "" {
			return fmt.Errorf("url cannot be empty")
		}
		if len(*r.URL) > maxFeedURLLength {
			return fmt.Errorf("url must be at most %d characters", maxFeedURLLength)
		}
	}
	if r.CalendarName != nil && len(*r.CalendarName) > MaxCalendarNameLength {
		return fmt.Errorf("calendar_name must be at most %d characters", MaxCalendarNameLength)
	}
	if r.RefreshIntervalMinutes != nil {
		if *r.RefreshIntervalMinutes < minRefreshIntervalMinutes {
			return fmt.Errorf("refresh_interval_minutes must be at least %d", minRefreshIntervalMinutes)
		}
		if *r.RefreshIntervalMinutes > maxRefreshIntervalMinutes {
			return fmt.Errorf("refresh_interval_minutes must be at most %d", maxRefreshIntervalMinutes)
		}
	}
	return nil
}
