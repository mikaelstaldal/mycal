package repository

import "github.com/mikaelstaldal/mycal/internal/model"

type EventRepository interface {
	List(from, to string, calendarIDs []int64) ([]model.Event, error)
	ListAll(calendarIDs []int64) ([]model.Event, error)
	ListRecurring(to string, calendarIDs []int64) ([]model.Event, error)
	Search(query, from, to string, calendarIDs []int64) ([]model.Event, error)
	GetByID(id int64) (*model.Event, error)
	Create(event *model.Event) error
	Update(event *model.Event) error
	Delete(id int64) error
	ListOverrides(parentIDs []int64) ([]model.Event, error)
	GetOverride(parentID int64, originalStart string) (*model.Event, error)
	DeleteByParentID(parentID int64) error
	FilterExistingIcsUIDs(uids []string) (map[string]bool, error)
}

type FeedRepository interface {
	CreateFeed(feed *model.Feed) error
	GetFeedByID(id int64) (*model.Feed, error)
	ListFeeds() ([]model.Feed, error)
	UpdateFeed(feed *model.Feed) error
	DeleteFeed(id int64) error
}

type PreferencesRepository interface {
	GetAllPreferences() (map[string]string, error)
	GetPreference(key string) (string, bool, error)
	SetPreference(key, value string) error
	DeletePreference(key string) error
}

type CalendarRepository interface {
	ListCalendars() ([]model.Calendar, error)
	GetCalendarByID(id int64) (*model.Calendar, error)
	GetCalendarByName(name string) (*model.Calendar, error)
	CreateCalendar(cal *model.Calendar) error
	UpdateCalendar(cal *model.Calendar) error
	DeleteCalendarIfUnused(id int64) error
}
