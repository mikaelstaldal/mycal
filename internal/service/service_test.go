package service

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mikaelstaldal/mycal/internal/api"
	"github.com/mikaelstaldal/mycal/internal/model"
)

// mockRepo implements repository.EventRepository with configurable behavior per test.
type mockRepo struct {
	listFn             func(from, to string, calendarIDs []int64) ([]model.Event, error)
	listAllFn          func(calendarIDs []int64) ([]model.Event, error)
	listRecurringFn    func(to string, calendarIDs []int64) ([]model.Event, error)
	searchFn           func(query, from, to string, calendarIDs []int64) ([]model.Event, error)
	getByIDFn          func(id int64) (*model.Event, error)
	createFn           func(event *model.Event) error
	updateFn           func(event *model.Event) error
	deleteFn           func(id int64) error
	listOverridesFn    func(parentIDs []int64, from, to string) ([]model.Event, error)
	getOverrideFn      func(parentID int64, originalStart string) (*model.Event, error)
	deleteByParentIDFn func(parentID int64) error
}

func (m *mockRepo) List(from, to string, calendarIDs []int64) ([]model.Event, error) {
	if m.listFn != nil {
		return m.listFn(from, to, calendarIDs)
	}
	return nil, nil
}

func (m *mockRepo) ListAll(calendarIDs []int64) ([]model.Event, error) {
	if m.listAllFn != nil {
		return m.listAllFn(calendarIDs)
	}
	return nil, nil
}

func (m *mockRepo) ListRecurring(to string, calendarIDs []int64) ([]model.Event, error) {
	if m.listRecurringFn != nil {
		return m.listRecurringFn(to, calendarIDs)
	}
	return nil, nil
}

func (m *mockRepo) Search(query, from, to string, calendarIDs []int64) ([]model.Event, error) {
	if m.searchFn != nil {
		return m.searchFn(query, from, to, calendarIDs)
	}
	return nil, nil
}

// mockCalRepo implements repository.CalendarRepository
type mockCalRepo struct{}

func (m *mockCalRepo) ListCalendars() ([]model.Calendar, error) {
	return []model.Calendar{{ID: 0, Name: "Default", Color: "dodgerblue"}}, nil
}
func (m *mockCalRepo) GetCalendarByID(id int64) (*model.Calendar, error) {
	return nil, nil
}
func (m *mockCalRepo) GetCalendarByName(name string) (*model.Calendar, error) {
	return nil, nil
}
func (m *mockCalRepo) CreateCalendar(cal *model.Calendar) error {
	return nil
}
func (m *mockCalRepo) UpdateCalendar(cal *model.Calendar) error {
	return nil
}
func (m *mockCalRepo) DeleteCalendar(id int64) error { return nil }

func (m *mockCalRepo) DeleteCalendarIfUnused(id int64) error {
	return nil
}

func (m *mockRepo) GetByID(id int64) (*model.Event, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, nil
}

func (m *mockRepo) Create(event *model.Event) error {
	if m.createFn != nil {
		return m.createFn(event)
	}
	return nil
}

func (m *mockRepo) Update(event *model.Event) error {
	if m.updateFn != nil {
		return m.updateFn(event)
	}
	return nil
}

func (m *mockRepo) Delete(id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}

func (m *mockRepo) ListOverrides(parentIDs []int64, from, to string) ([]model.Event, error) {
	if m.listOverridesFn != nil {
		return m.listOverridesFn(parentIDs, from, to)
	}
	return nil, nil
}

func (m *mockRepo) GetOverride(parentID int64, originalStart string) (*model.Event, error) {
	if m.getOverrideFn != nil {
		return m.getOverrideFn(parentID, originalStart)
	}
	return nil, nil
}

func (m *mockRepo) DeleteByParentID(parentID int64) error {
	if m.deleteByParentIDFn != nil {
		return m.deleteByParentIDFn(parentID)
	}
	return nil
}

// helpers
func float64Ptr(f float64) *float64 { return &f }

func optString(s string) api.OptString     { return api.NewOptString(s) }
func optInt(i int) api.OptInt              { return api.NewOptInt(i) }
func optBool(b bool) api.OptBool           { return api.NewOptBool(b) }
func optFloat(f float64) api.OptNilFloat64 { return api.NewOptNilFloat64(f) }
func optDateTime(s string) api.OptDateTime {
	t, _ := time.Parse(time.RFC3339, s)
	return api.NewOptDateTime(t)
}
func optDate(s string) api.OptDate {
	t, _ := time.Parse("2006-01-02", s)
	return api.NewOptDate(t)
}
func optFreqUpdate(s string) api.OptUpdateEventRequestRecurrenceFreq {
	return api.NewOptUpdateEventRequestRecurrenceFreq(api.UpdateEventRequestRecurrenceFreq(s))
}

var errRepo = errors.New("repo error")

func TestNewEventService(t *testing.T) {
	repo := &mockRepo{}
	calRepo := &mockCalRepo{}
	svc := NewEventService(repo, calRepo)
	assert.NotNil(t, svc)
}

// --- ListAll ---

func TestListAll_ReturnsEvents(t *testing.T) {
	repo := &mockRepo{
		listAllFn: func(calendarIDs []int64) ([]model.Event, error) {
			return []model.Event{{ID: 1, Title: "A"}, {ID: 2, Title: "B"}}, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events, err := svc.ListAll(nil)
	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestListAll_NilNormalizesToEmptySlice(t *testing.T) {
	repo := &mockRepo{
		listAllFn: func(calendarIDs []int64) ([]model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events, err := svc.ListAll(nil)
	require.NoError(t, err)
	assert.NotNil(t, events)
	assert.Empty(t, events)
}

func TestListAll_RepoError(t *testing.T) {
	repo := &mockRepo{
		listAllFn: func(calendarIDs []int64) ([]model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.ListAll(nil)
	assert.ErrorIs(t, err, errRepo)
}

// --- List ---

func TestList_BasicList(t *testing.T) {
	repo := &mockRepo{
		listFn: func(from, to string, calendarIDs []int64) ([]model.Event, error) {
			return []model.Event{{ID: 1, Title: "Meeting"}}, nil
		},
		listRecurringFn: func(to string, calendarIDs []int64) ([]model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events, err := svc.List("2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z", nil)
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestList_NilNormalizesToEmptySlice(t *testing.T) {
	repo := &mockRepo{
		listFn: func(from, to string, calendarIDs []int64) ([]model.Event, error) {
			return nil, nil
		},
		listRecurringFn: func(to string, calendarIDs []int64) ([]model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events, err := svc.List("2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z", nil)
	require.NoError(t, err)
	assert.NotNil(t, events)
}

func TestList_RepoListError(t *testing.T) {
	repo := &mockRepo{
		listFn: func(from, to string, calendarIDs []int64) ([]model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.List("2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z", nil)
	assert.ErrorIs(t, err, errRepo)
}

func TestList_RepoListRecurringError(t *testing.T) {
	repo := &mockRepo{
		listFn: func(from, to string, calendarIDs []int64) ([]model.Event, error) {
			return []model.Event{}, nil
		},
		listRecurringFn: func(to string, calendarIDs []int64) ([]model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.List("2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z", nil)
	assert.ErrorIs(t, err, errRepo)
}

func TestList_WithRecurringAndOverrides(t *testing.T) {
	parentID := int64(10)
	repo := &mockRepo{
		listFn: func(from, to string, calendarIDs []int64) ([]model.Event, error) {
			return []model.Event{}, nil
		},
		listRecurringFn: func(to string, calendarIDs []int64) ([]model.Event, error) {
			return []model.Event{{
				ID:             parentID,
				Title:          "Daily",
				StartTime:      "2026-02-01T10:00:00Z",
				EndTime:        "2026-02-01T11:00:00Z",
				RecurrenceFreq: "DAILY",
			}}, nil
		},
		listOverridesFn: func(parentIDs []int64, from, to string) ([]model.Event, error) {
			return []model.Event{{
				ID:                      20,
				Title:                   "Daily (modified)",
				StartTime:               "2026-02-02T10:00:00Z",
				EndTime:                 "2026-02-02T11:00:00Z",
				RecurrenceParentID:      &parentID,
				RecurrenceOriginalStart: "2026-02-02T10:00:00Z",
			}}, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events, err := svc.List("2026-02-01T00:00:00Z", "2026-02-04T00:00:00Z", nil)
	require.NoError(t, err)
	// Should have expanded instances with override applied
	assert.NotEmpty(t, events)
	// Check that the override replaced the Feb 2 instance
	foundOverride := false
	for _, e := range events {
		if e.Title == "Daily (modified)" {
			foundOverride = true
		}
	}
	assert.True(t, foundOverride, "expected override to replace Feb 2 instance")
}

func TestList_ListOverridesError(t *testing.T) {
	repo := &mockRepo{
		listFn: func(from, to string, calendarIDs []int64) ([]model.Event, error) {
			return []model.Event{}, nil
		},
		listRecurringFn: func(to string, calendarIDs []int64) ([]model.Event, error) {
			return []model.Event{{
				ID:             1,
				Title:          "Daily",
				StartTime:      "2026-02-01T10:00:00Z",
				EndTime:        "2026-02-01T11:00:00Z",
				RecurrenceFreq: "DAILY",
			}}, nil
		},
		listOverridesFn: func(parentIDs []int64, from, to string) ([]model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.List("2026-02-01T00:00:00Z", "2026-02-04T00:00:00Z", nil)
	assert.ErrorIs(t, err, errRepo)
}

// --- Search ---

func TestSearch_ReturnsResults(t *testing.T) {
	repo := &mockRepo{
		searchFn: func(query, from, to string, calendarIDs []int64) ([]model.Event, error) {
			return []model.Event{{ID: 1, Title: "Meeting"}}, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events, err := svc.Search("meet", "2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z", nil)
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestSearch_NilNormalizesToEmptySlice(t *testing.T) {
	repo := &mockRepo{
		searchFn: func(query, from, to string, calendarIDs []int64) ([]model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events, err := svc.Search("nothing", "2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z", nil)
	require.NoError(t, err)
	assert.NotNil(t, events)
}

func TestSearch_RepoError(t *testing.T) {
	repo := &mockRepo{
		searchFn: func(query, from, to string, calendarIDs []int64) ([]model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.Search("test", "2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z", nil)
	assert.ErrorIs(t, err, errRepo)
}

// --- GetByID ---

func TestGetByID_Found(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{ID: id, Title: "Found"}, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	e, err := svc.GetByID(42)
	require.NoError(t, err)
	assert.Equal(t, int64(42), e.ID)
	assert.Equal(t, "Found", e.Title)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.GetByID(42)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetByID_RepoError(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.GetByID(42)
	assert.ErrorIs(t, err, errRepo)
}

// --- Create ---

func TestCreate_Valid(t *testing.T) {
	var created *model.Event
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			event.ID = 1
			created = event
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.CreateEventRequest{
		Title:     "New Event",
		StartTime: optDateTime("2026-02-15T10:00:00Z"),
		EndTime:   optDateTime("2026-02-15T11:00:00Z"),
	}
	e, err := svc.Create(req)
	require.NoError(t, err)
	assert.Equal(t, int64(1), e.ID)
	assert.Equal(t, "New Event", created.Title)
}

func TestCreate_ValidationFailure(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.CreateEventRequest{
		Title: "", // required
	}
	_, err := svc.Create(req)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestCreate_RepoError(t *testing.T) {
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			return errRepo
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.CreateEventRequest{
		Title:     "Test",
		StartTime: optDateTime("2026-02-15T10:00:00Z"),
		EndTime:   optDateTime("2026-02-15T11:00:00Z"),
	}
	_, err := svc.Create(req)
	assert.ErrorIs(t, err, errRepo)
}

func TestCreate_HTMLSanitization(t *testing.T) {
	var created *model.Event
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			created = event
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.CreateEventRequest{
		Title:       "A & B: 3 < 5 > 2",
		Description: optString(`<b>bold</b><script>alert('xss')</script>`),
		StartTime:   optDateTime("2026-02-15T10:00:00Z"),
		EndTime:     optDateTime("2026-02-15T11:00:00Z"),
	}
	_, err := svc.Create(req)
	require.NoError(t, err)
	assert.Equal(t, "<b>bold</b>", created.Description)
	assert.Equal(t, "A & B: 3 < 5 > 2", created.Title)
}

func TestCreate_NoteSlug(t *testing.T) {
	var created *model.Event
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			created = event
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.CreateEventRequest{
		Title:     "Test",
		NoteSlug:  optString("project-kickoff"),
		StartTime: optDateTime("2026-02-15T10:00:00Z"),
		EndTime:   optDateTime("2026-02-15T11:00:00Z"),
	}
	_, err := svc.Create(req)
	require.NoError(t, err)
	assert.Equal(t, "project-kickoff", created.NoteSlug)
}

func TestCreate_InvalidNoteSlug(t *testing.T) {
	svc := NewEventService(&mockRepo{}, &mockCalRepo{})
	req := &api.CreateEventRequest{
		Title:     "Test",
		NoteSlug:  optString("Not A Slug"),
		StartTime: optDateTime("2026-02-15T10:00:00Z"),
		EndTime:   optDateTime("2026-02-15T11:00:00Z"),
	}
	_, err := svc.Create(req)
	assert.ErrorIs(t, err, ErrValidation)
}

// --- Update ---

func TestUpdate_NoteSlugCleared(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:        id,
				Title:     "Original",
				NoteSlug:  "project-kickoff",
				StartTime: "2026-02-15T10:00:00Z",
				EndTime:   "2026-02-15T11:00:00Z",
			}, nil
		},
		updateFn: func(event *model.Event) error { return nil },
	}
	svc := NewEventService(repo, &mockCalRepo{})

	// An omitted note_slug leaves the link alone; the empty string unlinks.
	e, err := svc.Update(1, &api.UpdateEventRequest{Title: optString("Updated")})
	require.NoError(t, err)
	assert.Equal(t, "project-kickoff", e.NoteSlug)

	e, err = svc.Update(1, &api.UpdateEventRequest{NoteSlug: optString("")})
	require.NoError(t, err)
	assert.Empty(t, e.NoteSlug)
}

func TestUpdate_PartialUpdate(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:        id,
				Title:     "Original",
				StartTime: "2026-02-15T10:00:00Z",
				EndTime:   "2026-02-15T11:00:00Z",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{
		Title: optString("Updated"),
	}
	e, err := svc.Update(1, req)
	require.NoError(t, err)
	assert.Equal(t, "Updated", e.Title)
	// StartTime should remain unchanged
	assert.Equal(t, "2026-02-15T10:00:00Z", e.StartTime)
}

func TestUpdate_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{Title: optString("X")}
	_, err := svc.Update(1, req)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUpdate_ValidationFailure(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{Title: optString("")} // an empty title isn't allowed
	_, err := svc.Update(1, req)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestUpdate_EndBeforeStart(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:        id,
				Title:     "Test",
				StartTime: "2026-02-15T10:00:00Z",
				EndTime:   "2026-02-15T11:00:00Z",
			}, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{
		EndTime: optDateTime("2026-02-15T09:00:00Z"),
	}
	_, err := svc.Update(1, req)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestUpdate_DurationRecomputesEndTime(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:        id,
				Title:     "Test",
				StartTime: "2026-02-15T10:00:00Z",
				EndTime:   "2026-02-15T11:00:00Z",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{
		Duration: optString("PT2H"),
	}
	e, err := svc.Update(1, req)
	require.NoError(t, err)
	assert.Equal(t, "2026-02-15T12:00:00Z", e.EndTime)
}

func TestUpdate_AllDayDateHandling(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:        id,
				Title:     "All Day",
				AllDay:    true,
				StartTime: "2026-02-15T00:00:00Z",
				EndTime:   "2026-02-16T00:00:00Z",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{
		StartDate: optDate("2026-02-20"),
		EndDate:   optDate("2026-02-22"),
	}
	e, err := svc.Update(1, req)
	require.NoError(t, err)
	assert.Equal(t, "2026-02-20T00:00:00Z", e.StartTime)
	assert.Equal(t, "2026-02-22T00:00:00Z", e.EndTime)
}

func TestUpdate_ToggleToAllDay(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:        id,
				Title:     "Test",
				AllDay:    false,
				StartTime: "2026-02-15T14:30:00Z",
				EndTime:   "2026-02-15T15:30:00Z",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{
		AllDay: optBool(true),
	}
	e, err := svc.Update(1, req)
	require.NoError(t, err)
	// Should normalize start to midnight and set the end to the next day
	assert.Equal(t, "2026-02-15T00:00:00Z", e.StartTime)
	assert.Equal(t, "2026-02-16T00:00:00Z", e.EndTime)
}

func TestUpdate_AllFieldUpdates(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:        id,
				Title:     "Original",
				StartTime: "2026-02-15T10:00:00Z",
				EndTime:   "2026-02-15T11:00:00Z",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{
		Color:                optString("blue"),
		RecurrenceFreq:       optFreqUpdate("WEEKLY"),
		RecurrenceCount:      optInt(10),
		RecurrenceUntil:      optString("2026-12-31T00:00:00Z"),
		RecurrenceInterval:   optInt(2),
		RecurrenceByDay:      optString("MO,WE"),
		RecurrenceByMonthday: optString("1,15"),
		RecurrenceByMonth:    optString("1,6"),
		Exdates:              optString("2026-02-22T10:00:00Z"),
		Rdates:               optString("2026-03-01T10:00:00Z"),
		Categories:           optString("work"),
		URL:                  optString("https://example.com"),
		ReminderMinutes:      optInt(30),
		Location:             optString("Room A"),
		Latitude:             optFloat(59.33),
		Longitude:            optFloat(18.07),
	}
	e, err := svc.Update(1, req)
	require.NoError(t, err)
	assert.Equal(t, "blue", e.Color)
	assert.Equal(t, "WEEKLY", e.RecurrenceFreq)
	assert.Equal(t, 10, e.RecurrenceCount)
	assert.Equal(t, 2, e.RecurrenceInterval)
	assert.Equal(t, "MO,WE", e.RecurrenceByDay)
	assert.Equal(t, "1,15", e.RecurrenceByMonthDay)
	assert.Equal(t, "1,6", e.RecurrenceByMonth)
	assert.Equal(t, "Room A", e.Location)
	require.NotNil(t, e.Latitude)
	assert.Equal(t, 59.33, *e.Latitude)
	require.NotNil(t, e.Longitude)
	assert.Equal(t, 18.07, *e.Longitude)
}

func TestUpdate_HTMLSanitization(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:        id,
				Title:     "Test",
				StartTime: "2026-02-15T10:00:00Z",
				EndTime:   "2026-02-15T11:00:00Z",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{
		Title:       optString("A & B: 3 < 5 > 2"),
		Description: optString(`<em>hi</em><script>bad</script>`),
	}
	e, err := svc.Update(1, req)
	require.NoError(t, err)
	assert.Equal(t, "<em>hi</em>", e.Description)
	assert.Equal(t, "A & B: 3 < 5 > 2", e.Title)
}

func TestUpdate_RepoError(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:        id,
				Title:     "Test",
				StartTime: "2026-02-15T10:00:00Z",
				EndTime:   "2026-02-15T11:00:00Z",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			return errRepo
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{Title: optString("New")}
	_, err := svc.Update(1, req)
	assert.ErrorIs(t, err, errRepo)
}

// --- CreateOrUpdateOverride ---

func TestCreateOrUpdateOverride_NewOverride(t *testing.T) {
	parentID := int64(10)
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:             parentID,
				CalendarID:     7,
				Title:          "Weekly",
				StartTime:      "2026-02-01T09:00:00Z",
				EndTime:        "2026-02-01T10:00:00Z",
				RecurrenceFreq: "WEEKLY",
			}, nil
		},
		getOverrideFn: func(pid int64, origStart string) (*model.Event, error) {
			return nil, nil // no existing override
		},
		createFn: func(event *model.Event) error {
			assert.Equal(t, int64(7), event.CalendarID)
			event.ID = 100
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{Title: optString("Modified Instance")}
	e, err := svc.CreateOrUpdateOverride(parentID, "2026-02-08T09:00:00Z", req)
	require.NoError(t, err)
	assert.Equal(t, "Modified Instance", e.Title)
	assert.Equal(t, int64(7), e.CalendarID)
	require.NotNil(t, e.RecurrenceParentID)
	assert.Equal(t, parentID, *e.RecurrenceParentID)
	assert.Equal(t, "2026-02-08T09:00:00Z", e.RecurrenceOriginalStart)
	// EndTime should be computed from the parent's duration (1h)
	assert.Equal(t, "2026-02-08T10:00:00Z", e.EndTime)
}

func TestCreateOrUpdateOverride_ExistingOverride(t *testing.T) {
	parentID := int64(10)
	overrideID := int64(20)
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			if id == parentID {
				return &model.Event{
					ID:             parentID,
					Title:          "Weekly",
					StartTime:      "2026-02-01T09:00:00Z",
					EndTime:        "2026-02-01T10:00:00Z",
					RecurrenceFreq: "WEEKLY",
				}, nil
			}
			// Return the existing override for Update call
			return &model.Event{
				ID:                      overrideID,
				Title:                   "Old Override",
				StartTime:               "2026-02-08T09:00:00Z",
				EndTime:                 "2026-02-08T10:00:00Z",
				RecurrenceParentID:      &parentID,
				RecurrenceOriginalStart: "2026-02-08T09:00:00Z",
			}, nil
		},
		getOverrideFn: func(pid int64, origStart string) (*model.Event, error) {
			return &model.Event{
				ID:                      overrideID,
				Title:                   "Old Override",
				StartTime:               "2026-02-08T09:00:00Z",
				EndTime:                 "2026-02-08T10:00:00Z",
				RecurrenceParentID:      &parentID,
				RecurrenceOriginalStart: "2026-02-08T09:00:00Z",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{Title: optString("Updated Override")}
	e, err := svc.CreateOrUpdateOverride(parentID, "2026-02-08T09:00:00Z", req)
	require.NoError(t, err)
	assert.Equal(t, "Updated Override", e.Title)
}

func TestCreateOrUpdateOverride_ParentNotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{Title: optString("X")}
	_, err := svc.CreateOrUpdateOverride(999, "2026-02-08T09:00:00Z", req)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCreateOrUpdateOverride_ParentNotRecurring(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:        1,
				Title:     "Single",
				StartTime: "2026-02-01T09:00:00Z",
				EndTime:   "2026-02-01T10:00:00Z",
				// No RecurrenceFreq
			}, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{Title: optString("X")}
	_, err := svc.CreateOrUpdateOverride(1, "2026-02-08T09:00:00Z", req)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestCreateOrUpdateOverride_InvalidInstanceStart(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{Title: optString("X")}
	_, err := svc.CreateOrUpdateOverride(1, "not-a-date", req)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestCreateOrUpdateOverride_NewOverrideWithAllFields(t *testing.T) {
	parentID := int64(10)
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:              parentID,
				Title:           "Weekly",
				Description:     "Desc",
				StartTime:       "2026-02-01T09:00:00Z",
				EndTime:         "2026-02-01T10:00:00Z",
				RecurrenceFreq:  "WEEKLY",
				Location:        "Office",
				Categories:      "work",
				URL:             "https://example.com",
				ReminderMinutes: 15,
				Latitude:        float64Ptr(59.0),
				Longitude:       float64Ptr(18.0),
			}, nil
		},
		getOverrideFn: func(pid int64, origStart string) (*model.Event, error) {
			return nil, nil
		},
		createFn: func(event *model.Event) error {
			event.ID = 100
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{
		Title:           optString("New Title & 3 < 5 > 2"),
		Description:     optString("<b>bold</b><script>bad</script>"),
		StartTime:       optDateTime("2026-02-08T10:00:00Z"),
		EndTime:         optDateTime("2026-02-08T12:00:00Z"),
		AllDay:          optBool(false),
		Color:           optString("red"),
		Duration:        optString("PT3H"),
		Categories:      optString("meeting"),
		URL:             optString("https://new.example.com"),
		ReminderMinutes: optInt(30),
		Location:        optString("Home"),
		Latitude:        optFloat(60.0),
		Longitude:       optFloat(19.0),
	}
	e, err := svc.CreateOrUpdateOverride(parentID, "2026-02-08T09:00:00Z", req)
	require.NoError(t, err)
	assert.Equal(t, "New Title & 3 < 5 > 2", e.Title)
	assert.Equal(t, "<b>bold</b>", e.Description)
	assert.Equal(t, "red", e.Color)
	assert.Equal(t, "meeting", e.Categories)
	assert.Equal(t, "https://new.example.com", e.URL)
	assert.Equal(t, 30, e.ReminderMinutes)
	assert.Equal(t, "Home", e.Location)
	require.NotNil(t, e.Latitude)
	assert.Equal(t, 60.0, *e.Latitude)
	require.NotNil(t, e.Longitude)
	assert.Equal(t, 19.0, *e.Longitude)
	// Duration was set, so EndTime should be recomputed from StartTime + Duration
	assert.Equal(t, "PT3H", e.Duration)
	assert.Equal(t, "2026-02-08T13:00:00Z", e.EndTime)
}

func TestCreateOrUpdateOverride_GetOverrideError(t *testing.T) {
	parentID := int64(10)
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:             parentID,
				Title:          "Weekly",
				StartTime:      "2026-02-01T09:00:00Z",
				EndTime:        "2026-02-01T10:00:00Z",
				RecurrenceFreq: "WEEKLY",
			}, nil
		},
		getOverrideFn: func(pid int64, origStart string) (*model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{Title: optString("X")}
	_, err := svc.CreateOrUpdateOverride(parentID, "2026-02-08T09:00:00Z", req)
	assert.ErrorIs(t, err, errRepo)
}

func TestCreateOrUpdateOverride_ValidationFailure(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo, &mockCalRepo{})
	req := &api.UpdateEventRequest{Title: optString("")} // an empty title isn't allowed
	_, err := svc.CreateOrUpdateOverride(1, "2026-02-08T09:00:00Z", req)
	assert.ErrorIs(t, err, ErrValidation)
}

// --- ImportSingle ---

func TestImportSingle_SingleEvent(t *testing.T) {
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			event.ID = 1
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events := []model.Event{{
		Title:     "Imported",
		StartTime: "2026-02-15T10:00:00Z",
		EndTime:   "2026-02-15T11:00:00Z",
	}}
	e, err := svc.ImportSingle(events, "")
	require.NoError(t, err)
	assert.Equal(t, "Imported", e.Title)
}

func TestImportSingle_NoEvents(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.ImportSingle(nil, "")
	assert.ErrorIs(t, err, ErrValidation)
}

func TestImportSingle_MultipleParents(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo, &mockCalRepo{})
	events := []model.Event{
		{Title: "A", StartTime: "2026-02-15T10:00:00Z", EndTime: "2026-02-15T11:00:00Z"},
		{Title: "B", StartTime: "2026-02-16T10:00:00Z", EndTime: "2026-02-16T11:00:00Z"},
	}
	_, err := svc.ImportSingle(events, "")
	assert.ErrorIs(t, err, ErrValidation)
}

func TestImportSingle_RejectsOverrides(t *testing.T) {
	parentID := int64(5)
	repo := &mockRepo{}
	svc := NewEventService(repo, &mockCalRepo{})

	t.Run("parent with override", func(t *testing.T) {
		events := []model.Event{
			{Title: "Parent", StartTime: "2026-02-15T10:00:00Z", EndTime: "2026-02-15T11:00:00Z"},
			{Title: "Override", StartTime: "2026-02-22T10:00:00Z", EndTime: "2026-02-22T11:00:00Z",
				RecurrenceParentID: &parentID, RecurrenceOriginalStart: "2026-02-22T10:00:00Z"},
		}
		_, err := svc.ImportSingle(events, "")
		assert.ErrorIs(t, err, ErrValidation)
	})

	t.Run("only override", func(t *testing.T) {
		events := []model.Event{
			{Title: "Override Only", StartTime: "2026-02-22T10:00:00Z", EndTime: "2026-02-22T11:00:00Z",
				RecurrenceParentID: &parentID, RecurrenceOriginalStart: "2026-02-22T10:00:00Z"},
		}
		_, err := svc.ImportSingle(events, "")
		assert.ErrorIs(t, err, ErrValidation)
	})
}

func TestImportSingle_AllDayFormatConversion(t *testing.T) {
	var created *model.Event
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			created = event
			event.ID = 1
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events := []model.Event{{
		Title:     "All Day",
		StartTime: "2026-02-15T00:00:00Z",
		EndTime:   "2026-02-16T00:00:00Z",
		AllDay:    true,
	}}
	_, err := svc.ImportSingle(events, "")
	require.NoError(t, err)
	// AllDay events should have their times normalized via Validate
	assert.Equal(t, "2026-02-15T00:00:00Z", created.StartTime)
}

func TestImportSingle_ValidationFailure(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo, &mockCalRepo{})
	events := []model.Event{{
		Title:     "", // empty title
		StartTime: "2026-02-15T10:00:00Z",
		EndTime:   "2026-02-15T11:00:00Z",
	}}
	_, err := svc.ImportSingle(events, "")
	assert.ErrorIs(t, err, ErrValidation)
}

// --- Import (batch) ---

func TestImport_ParentsAndOverrides(t *testing.T) {
	var createdEvents []*model.Event
	nextID := int64(1)
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			event.ID = nextID
			nextID++
			createdEvents = append(createdEvents, event)
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events := []model.Event{
		{
			Title:          "Weekly Meeting",
			StartTime:      "2026-02-01T10:00:00Z",
			EndTime:        "2026-02-01T11:00:00Z",
			RecurrenceFreq: "WEEKLY",
			ImportUID:      "uid-123",
		},
		{
			Title:                   "Modified Instance",
			RecurrenceFreq:          "DAILY",
			RecurrenceCount:         3,
			RecurrenceUntil:         "invalid",
			RecurrenceInterval:      2,
			RecurrenceByDay:         "MO",
			RecurrenceByMonthDay:    "1",
			RecurrenceByMonth:       "1",
			ExDates:                 "invalid",
			RDates:                  "invalid",
			StartTime:               "2026-02-08T10:00:00Z",
			EndTime:                 "2026-02-08T11:30:00Z",
			RecurrenceOriginalStart: "2026-02-08T10:00:00Z",
			ImportUID:               "uid-123",
		},
	}
	count, err := svc.Import(events, "")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Len(t, createdEvents, 2)
	// The second should be override linked to the first
	override := createdEvents[1]
	require.NotNil(t, override.RecurrenceParentID)
	assert.Equal(t, int64(1), *override.RecurrenceParentID)
	assert.Empty(t, override.RecurrenceFreq)
	assert.Zero(t, override.RecurrenceCount)
	assert.Empty(t, override.RecurrenceUntil)
	assert.Zero(t, override.RecurrenceInterval)
	assert.Empty(t, override.RecurrenceByDay)
	assert.Empty(t, override.RecurrenceByMonthDay)
	assert.Empty(t, override.RecurrenceByMonth)
	assert.Empty(t, override.ExDates)
	assert.Empty(t, override.RDates)
	assert.Empty(t, override.IcsUID)

}

func TestImport_SkipsInvalidEvents(t *testing.T) {
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			event.ID = 1
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events := []model.Event{
		{Title: "", StartTime: "2026-02-15T10:00:00Z", EndTime: "2026-02-15T11:00:00Z"}, // invalid: no title
		{Title: "Valid", StartTime: "2026-02-15T10:00:00Z", EndTime: "2026-02-15T11:00:00Z"},
	}
	count, err := svc.Import(events, "")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestImport_OverrideWithoutParent(t *testing.T) {
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			event.ID = 1
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events := []model.Event{
		{
			Title:                   "Orphan Override",
			StartTime:               "2026-02-08T10:00:00Z",
			EndTime:                 "2026-02-08T11:00:00Z",
			RecurrenceOriginalStart: "2026-02-08T10:00:00Z",
			ImportUID:               "uid-no-parent",
		},
	}
	count, err := svc.Import(events, "")
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestImport_AllDayFormatConversion(t *testing.T) {
	var created *model.Event
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			created = event
			event.ID = 1
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events := []model.Event{{
		Title:     "All Day Import",
		StartTime: "2026-03-10T00:00:00Z",
		EndTime:   "2026-03-11T00:00:00Z",
		AllDay:    true,
	}}
	count, err := svc.Import(events, "")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, "2026-03-10T00:00:00Z", created.StartTime)
}

func TestImport_RepoCreateError(t *testing.T) {
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			return errRepo
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	events := []model.Event{
		{Title: "Test", StartTime: "2026-02-15T10:00:00Z", EndTime: "2026-02-15T11:00:00Z"},
	}
	count, err := svc.Import(events, "")
	require.NoError(t, err) // Import continues on create errors
	assert.Equal(t, 0, count)
}

// --- AddExDate ---

func TestAddExDate_Appends(t *testing.T) {
	var updated *model.Event
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:             id,
				Title:          "Weekly",
				StartTime:      "2026-02-01T09:00:00Z",
				EndTime:        "2026-02-01T10:00:00Z",
				RecurrenceFreq: "WEEKLY",
				ExDates:        "2026-02-08T09:00:00Z",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			updated = event
			return nil
		},
		getOverrideFn: func(parentID int64, originalStart string) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	e, err := svc.AddExDate(1, "2026-02-15T09:00:00Z")
	require.NoError(t, err)
	expected := "2026-02-08T09:00:00Z,2026-02-15T09:00:00Z"
	assert.Equal(t, expected, updated.ExDates)
	assert.Equal(t, expected, e.ExDates)
}

func TestAddExDate_FirstExDate(t *testing.T) {
	var updated *model.Event
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:             id,
				Title:          "Weekly",
				StartTime:      "2026-02-01T09:00:00Z",
				EndTime:        "2026-02-01T10:00:00Z",
				RecurrenceFreq: "WEEKLY",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			updated = event
			return nil
		},
		getOverrideFn: func(parentID int64, originalStart string) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.AddExDate(1, "2026-02-08T09:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, "2026-02-08T09:00:00Z", updated.ExDates)
}

func TestAddExDate_FirstOccurrenceMovesStart(t *testing.T) {
	var updated *model.Event
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:             id,
				Title:          "Weekly",
				StartTime:      "2026-02-01T09:00:00Z",
				EndTime:        "2026-02-01T10:00:00Z",
				RecurrenceFreq: "WEEKLY",
				ExDates:        "2026-02-01T09:00:00Z,2026-02-08T09:00:00Z",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			updated = event
			return nil
		},
		getOverrideFn: func(parentID int64, originalStart string) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	e, err := svc.AddExDate(1, "2026-02-15T09:00:00Z")
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "2026-02-22T09:00:00Z", updated.StartTime)
	assert.Equal(t, "2026-02-22T10:00:00Z", updated.EndTime)
	assert.Equal(t, "", updated.ExDates)
	assert.Equal(t, updated, e)
}

func TestAddExDate_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.AddExDate(999, "2026-02-08T09:00:00Z")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestAddExDate_NotRecurring(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:        id,
				Title:     "Single",
				StartTime: "2026-02-01T09:00:00Z",
				EndTime:   "2026-02-01T10:00:00Z",
			}, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.AddExDate(1, "2026-02-08T09:00:00Z")
	assert.ErrorIs(t, err, ErrValidation)
}

func TestAddExDate_InvalidFormat(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.AddExDate(1, "not-a-date")
	assert.ErrorIs(t, err, ErrValidation)
}

func TestAddExDate_DeletesAssociatedOverride(t *testing.T) {
	overrideID := int64(20)
	deletedID := int64(0)
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:             id,
				Title:          "Weekly",
				StartTime:      "2026-02-01T09:00:00Z",
				EndTime:        "2026-02-01T10:00:00Z",
				RecurrenceFreq: "WEEKLY",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			return nil
		},
		getOverrideFn: func(parentID int64, originalStart string) (*model.Event, error) {
			return &model.Event{ID: overrideID}, nil
		},
		deleteFn: func(id int64) error {
			deletedID = id
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.AddExDate(1, "2026-02-08T09:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, overrideID, deletedID)
}

// --- RemoveExDate ---

func TestRemoveExDate_RemovesSpecific(t *testing.T) {
	var updated *model.Event
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:             id,
				Title:          "Weekly",
				StartTime:      "2026-02-01T09:00:00Z",
				EndTime:        "2026-02-01T10:00:00Z",
				RecurrenceFreq: "WEEKLY",
				ExDates:        "2026-02-08T09:00:00Z,2026-02-15T09:00:00Z,2026-02-22T09:00:00Z",
			}, nil
		},
		updateFn: func(event *model.Event) error {
			updated = event
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.RemoveExDate(1, "2026-02-15T09:00:00Z")
	require.NoError(t, err)
	expected := "2026-02-08T09:00:00Z,2026-02-22T09:00:00Z"
	assert.Equal(t, expected, updated.ExDates)
}

func TestRemoveExDate_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	_, err := svc.RemoveExDate(999, "2026-02-08T09:00:00Z")
	assert.ErrorIs(t, err, ErrNotFound)
}

// --- Delete ---

func TestDelete_Success(t *testing.T) {
	deleteByParentCalled := false
	repo := &mockRepo{
		deleteByParentIDFn: func(parentID int64) error {
			deleteByParentCalled = true
			return nil
		},
		deleteFn: func(id int64) error {
			return nil
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	err := svc.Delete(1)
	require.NoError(t, err)
	assert.True(t, deleteByParentCalled, "expected DeleteByParentID to be called")
}

func TestDelete_NotFound(t *testing.T) {
	repo := &mockRepo{
		deleteByParentIDFn: func(parentID int64) error {
			return nil
		},
		deleteFn: func(id int64) error {
			return sql.ErrNoRows
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	err := svc.Delete(999)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDelete_RepoError(t *testing.T) {
	repo := &mockRepo{
		deleteByParentIDFn: func(parentID int64) error {
			return nil
		},
		deleteFn: func(id int64) error {
			return errRepo
		},
	}
	svc := NewEventService(repo, &mockCalRepo{})
	err := svc.Delete(1)
	assert.ErrorIs(t, err, errRepo)
}

func TestImportDurationValidation(t *testing.T) {
	for _, duration := range []string{"", "PT1H", "garbage", "PT1H\rATTENDEE:mailto:evil@example.com"} {
		t.Run(duration, func(t *testing.T) {
			valid := duration == "" || duration == "PT1H"
			var created []*model.Event
			svc := NewEventService(&mockRepo{createFn: func(e *model.Event) error {
				e.ID = int64(len(created) + 1)
				created = append(created, e)
				return nil
			}}, &mockCalRepo{})
			event := model.Event{Title: "Test", StartTime: "2026-01-01T10:00:00Z",
				EndTime: "2026-01-01T11:00:00Z", Duration: duration, ImportUID: "test"}
			_, err := svc.ImportSingle([]model.Event{event}, "")
			if valid {
				require.NoError(t, err)
				require.Len(t, created, 1)
				assert.Equal(t, duration, created[0].Duration)
			} else {
				require.ErrorIs(t, err, ErrValidation)
				assert.Empty(t, created)
			}
			created = nil
			parent := event
			parent.Duration = ""
			parent.RecurrenceFreq = "DAILY"
			event.RecurrenceOriginalStart = event.StartTime
			count, err := svc.Import([]model.Event{parent, event}, "")
			require.NoError(t, err)
			if valid {
				assert.Equal(t, 2, count)
				require.Len(t, created, 2)
				assert.Equal(t, duration, created[1].Duration)
			} else {
				assert.Equal(t, 1, count)
				assert.Len(t, created, 1)
			}
		})
	}
}

func TestImportRejectsOversizedOverride(t *testing.T) {
	created := 0
	svc := NewEventService(&mockRepo{createFn: func(e *model.Event) error {
		created++
		e.ID = int64(created)
		return nil
	}}, &mockCalRepo{})
	parent := model.Event{Title: "Parent", StartTime: "2026-01-01T10:00:00Z",
		EndTime: "2026-01-01T11:00:00Z", ImportUID: "test", RecurrenceFreq: "DAILY"}
	override := parent
	override.RecurrenceOriginalStart = parent.StartTime
	override.EndTime = "2054-10-01T10:00:00Z"
	override.Duration = "P1500W"
	count, err := svc.Import([]model.Event{parent, override}, "")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, 1, created)
}

func TestImportOverrideValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*model.Event)
	}{
		{"empty title", func(e *model.Event) { e.Title = "" }},
		{"long title", func(e *model.Event) { e.Title = strings.Repeat("x", maxTitleLength+1) }},
		{"long description", func(e *model.Event) { e.Description = strings.Repeat("x", maxDescriptionLength+1) }},
		{"long location", func(e *model.Event) { e.Location = strings.Repeat("x", maxLocationLength+1) }},
		{"long categories", func(e *model.Event) { e.Categories = strings.Repeat("x", maxCategoriesLength+1) }},
		{"invalid start", func(e *model.Event) { e.StartTime = "invalid" }},
		{"reversed times", func(e *model.Event) { e.EndTime = "2026-01-01T09:00:00Z" }},
		{"past date", func(e *model.Event) { e.StartTime = "1000-01-01T10:00:00Z" }},
		{"future date", func(e *model.Event) { e.EndTime = "3000-01-01T10:00:00Z" }},
		{"long span", func(e *model.Event) { e.EndTime = "2028-01-01T10:00:00Z" }},
		{"duration", func(e *model.Event) { e.Duration = "invalid" }},
		{"latitude", func(e *model.Event) { v := 91.0; e.Latitude = &v }},
		{"longitude", func(e *model.Event) { v := -181.0; e.Longitude = &v }},
		{"recurrence ID", func(e *model.Event) { e.RecurrenceOriginalStart = "invalid" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var created []*model.Event
			svc := NewEventService(&mockRepo{createFn: func(e *model.Event) error {
				e.ID = int64(len(created) + 1)
				created = append(created, e)
				return nil
			}}, &mockCalRepo{})
			parent := model.Event{Title: "Parent", StartTime: "2026-01-01T10:00:00Z",
				EndTime: "2026-01-01T11:00:00Z", ImportUID: "test", RecurrenceFreq: "DAILY"}
			valid := model.Event{Title: "Valid override", StartTime: parent.StartTime, EndTime: parent.EndTime,
				ImportUID: "test", RecurrenceOriginalStart: parent.StartTime}
			invalid := valid
			tt.mutate(&invalid)
			count, err := svc.Import([]model.Event{invalid, parent, valid}, "")
			require.NoError(t, err)
			assert.Equal(t, 2, count)
			require.Len(t, created, 2)
			assert.Equal(t, valid.Title, created[1].Title)
			require.NotNil(t, created[1].RecurrenceParentID)
			assert.Equal(t, created[0].ID, *created[1].RecurrenceParentID)
			assert.Equal(t, valid.RecurrenceOriginalStart, created[1].RecurrenceOriginalStart)
		})
	}
}

func TestAddExDate_ListLimits(t *testing.T) {
	const instance = "2026-02-15T09:00:00Z"
	for _, tc := range []struct {
		name       string
		exdates    string
		wantError  bool
		wantUpdate bool
	}{
		{"duplicate", "  " + instance + "  ", false, false},
		{"duplicate in oversized legacy list", strings.Repeat(instance+",", 250) + instance, false, false},
		{"at limit", strings.Repeat("2026-02-08T09:00:00Z,", 236) + "2026-02-08T09:00:00.00Z", false, true},
		{"over limit", strings.Repeat("2026-02-08T09:00:00Z,", 236) + "2026-02-08T09:00:00.000Z", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			event := &model.Event{ID: 1, StartTime: "2026-02-01T09:00:00Z",
				EndTime: "2026-02-01T10:00:00Z", RecurrenceFreq: "WEEKLY", ExDates: tc.exdates}
			updated := false
			repo := &mockRepo{
				getByIDFn:     func(int64) (*model.Event, error) { return event, nil },
				updateFn:      func(*model.Event) error { updated = true; return nil },
				getOverrideFn: func(int64, string) (*model.Event, error) { return nil, nil },
			}
			result, err := NewEventService(repo, &mockCalRepo{}).AddExDate(1, instance)
			if tc.wantError {
				require.ErrorIs(t, err, ErrValidation)
				assert.Equal(t, tc.exdates, event.ExDates)
			} else {
				require.NoError(t, err)
				require.Same(t, event, result)
				if tc.wantUpdate {
					assert.Len(t, result.ExDates, maxRecurrenceListLen)
				} else {
					assert.Equal(t, tc.exdates, result.ExDates)
				}
			}
			assert.Equal(t, tc.wantUpdate, updated)
		})
	}
}
