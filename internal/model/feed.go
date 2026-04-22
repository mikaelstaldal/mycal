package model

type Feed struct {
	ID                     int64
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
