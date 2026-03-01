package service

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/mikaelstaldal/mycal/internal/model"
)

// mockRepo implements repository.EventRepository with configurable behavior per test.
type mockRepo struct {
	listFn            func(from, to string) ([]model.Event, error)
	listAllFn         func() ([]model.Event, error)
	listRecurringFn   func(to string) ([]model.Event, error)
	searchFn          func(query, from, to string) ([]model.Event, error)
	getByIDFn         func(id int64) (*model.Event, error)
	createFn          func(event *model.Event) error
	updateFn          func(event *model.Event) error
	deleteFn          func(id int64) error
	listOverridesFn   func(parentIDs []int64) ([]model.Event, error)
	getOverrideFn     func(parentID int64, originalStart string) (*model.Event, error)
	deleteByParentIDFn func(parentID int64) error
}

func (m *mockRepo) List(from, to string) ([]model.Event, error) {
	if m.listFn != nil {
		return m.listFn(from, to)
	}
	return nil, nil
}

func (m *mockRepo) ListAll() ([]model.Event, error) {
	if m.listAllFn != nil {
		return m.listAllFn()
	}
	return nil, nil
}

func (m *mockRepo) ListRecurring(to string) ([]model.Event, error) {
	if m.listRecurringFn != nil {
		return m.listRecurringFn(to)
	}
	return nil, nil
}

func (m *mockRepo) Search(query, from, to string) ([]model.Event, error) {
	if m.searchFn != nil {
		return m.searchFn(query, from, to)
	}
	return nil, nil
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

func (m *mockRepo) ListOverrides(parentIDs []int64) ([]model.Event, error) {
	if m.listOverridesFn != nil {
		return m.listOverridesFn(parentIDs)
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
func strPtr(s string) *string    { return &s }
func intPtr(i int) *int          { return &i }
func boolPtr(b bool) *bool       { return &b }
func float64Ptr(f float64) *float64 { return &f }

var errRepo = errors.New("repo error")

func TestNewEventService(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.repo != repo {
		t.Fatal("expected repo to be set")
	}
}

// --- ListAll ---

func TestListAll_ReturnsEvents(t *testing.T) {
	repo := &mockRepo{
		listAllFn: func() ([]model.Event, error) {
			return []model.Event{{ID: 1, Title: "A"}, {ID: 2, Title: "B"}}, nil
		},
	}
	svc := NewEventService(repo)
	events, err := svc.ListAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestListAll_NilNormalizesToEmptySlice(t *testing.T) {
	repo := &mockRepo{
		listAllFn: func() ([]model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo)
	events, err := svc.ListAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestListAll_RepoError(t *testing.T) {
	repo := &mockRepo{
		listAllFn: func() ([]model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo)
	_, err := svc.ListAll()
	if !errors.Is(err, errRepo) {
		t.Fatalf("expected repo error, got: %v", err)
	}
}

// --- List ---

func TestList_BasicList(t *testing.T) {
	repo := &mockRepo{
		listFn: func(from, to string) ([]model.Event, error) {
			return []model.Event{{ID: 1, Title: "Meeting"}}, nil
		},
		listRecurringFn: func(to string) ([]model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo)
	events, err := svc.List("2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestList_NilNormalizesToEmptySlice(t *testing.T) {
	repo := &mockRepo{
		listFn: func(from, to string) ([]model.Event, error) {
			return nil, nil
		},
		listRecurringFn: func(to string) ([]model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo)
	events, err := svc.List("2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events == nil {
		t.Fatal("expected non-nil empty slice")
	}
}

func TestList_RepoListError(t *testing.T) {
	repo := &mockRepo{
		listFn: func(from, to string) ([]model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo)
	_, err := svc.List("2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z")
	if !errors.Is(err, errRepo) {
		t.Fatalf("expected repo error, got: %v", err)
	}
}

func TestList_RepoListRecurringError(t *testing.T) {
	repo := &mockRepo{
		listFn: func(from, to string) ([]model.Event, error) {
			return []model.Event{}, nil
		},
		listRecurringFn: func(to string) ([]model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo)
	_, err := svc.List("2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z")
	if !errors.Is(err, errRepo) {
		t.Fatalf("expected repo error, got: %v", err)
	}
}

func TestList_WithRecurringAndOverrides(t *testing.T) {
	parentID := int64(10)
	repo := &mockRepo{
		listFn: func(from, to string) ([]model.Event, error) {
			return []model.Event{}, nil
		},
		listRecurringFn: func(to string) ([]model.Event, error) {
			return []model.Event{{
				ID:             parentID,
				Title:          "Daily",
				StartTime:      "2026-02-01T10:00:00Z",
				EndTime:        "2026-02-01T11:00:00Z",
				RecurrenceFreq: "DAILY",
			}}, nil
		},
		listOverridesFn: func(parentIDs []int64) ([]model.Event, error) {
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
	svc := NewEventService(repo)
	events, err := svc.List("2026-02-01T00:00:00Z", "2026-02-04T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have expanded instances with override applied
	if len(events) == 0 {
		t.Fatal("expected some events from recurring expansion")
	}
	// Check that the override replaced the Feb 2 instance
	foundOverride := false
	for _, e := range events {
		if e.Title == "Daily (modified)" {
			foundOverride = true
		}
	}
	if !foundOverride {
		t.Fatal("expected override to replace Feb 2 instance")
	}
}

func TestList_ListOverridesError(t *testing.T) {
	repo := &mockRepo{
		listFn: func(from, to string) ([]model.Event, error) {
			return []model.Event{}, nil
		},
		listRecurringFn: func(to string) ([]model.Event, error) {
			return []model.Event{{
				ID:             1,
				Title:          "Daily",
				StartTime:      "2026-02-01T10:00:00Z",
				EndTime:        "2026-02-01T11:00:00Z",
				RecurrenceFreq: "DAILY",
			}}, nil
		},
		listOverridesFn: func(parentIDs []int64) ([]model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo)
	_, err := svc.List("2026-02-01T00:00:00Z", "2026-02-04T00:00:00Z")
	if !errors.Is(err, errRepo) {
		t.Fatalf("expected repo error, got: %v", err)
	}
}

// --- Search ---

func TestSearch_ReturnsResults(t *testing.T) {
	repo := &mockRepo{
		searchFn: func(query, from, to string) ([]model.Event, error) {
			return []model.Event{{ID: 1, Title: "Meeting"}}, nil
		},
	}
	svc := NewEventService(repo)
	events, err := svc.Search("meet", "2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestSearch_NilNormalizesToEmptySlice(t *testing.T) {
	repo := &mockRepo{
		searchFn: func(query, from, to string) ([]model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo)
	events, err := svc.Search("nothing", "2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events == nil {
		t.Fatal("expected non-nil empty slice")
	}
}

func TestSearch_RepoError(t *testing.T) {
	repo := &mockRepo{
		searchFn: func(query, from, to string) ([]model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo)
	_, err := svc.Search("test", "2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z")
	if !errors.Is(err, errRepo) {
		t.Fatalf("expected repo error, got: %v", err)
	}
}

// --- GetByID ---

func TestGetByID_Found(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{ID: id, Title: "Found"}, nil
		},
	}
	svc := NewEventService(repo)
	e, err := svc.GetByID(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.ID != 42 || e.Title != "Found" {
		t.Fatalf("unexpected event: %+v", e)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo)
	_, err := svc.GetByID(42)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetByID_RepoError(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return nil, errRepo
		},
	}
	svc := NewEventService(repo)
	_, err := svc.GetByID(42)
	if !errors.Is(err, errRepo) {
		t.Fatalf("expected repo error, got: %v", err)
	}
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
	svc := NewEventService(repo)
	req := &model.CreateEventRequest{
		Title:     "New Event",
		StartTime: "2026-02-15T10:00:00Z",
		EndTime:   "2026-02-15T11:00:00Z",
	}
	e, err := svc.Create(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.ID != 1 {
		t.Fatalf("expected ID 1, got %d", e.ID)
	}
	if created.Title != "New Event" {
		t.Fatalf("expected title 'New Event', got %q", created.Title)
	}
}

func TestCreate_ValidationFailure(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo)
	req := &model.CreateEventRequest{
		Title: "", // required
	}
	_, err := svc.Create(req)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}
}

func TestCreate_RepoError(t *testing.T) {
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			return errRepo
		},
	}
	svc := NewEventService(repo)
	req := &model.CreateEventRequest{
		Title:     "Test",
		StartTime: "2026-02-15T10:00:00Z",
		EndTime:   "2026-02-15T11:00:00Z",
	}
	_, err := svc.Create(req)
	if !errors.Is(err, errRepo) {
		t.Fatalf("expected repo error, got: %v", err)
	}
}

func TestCreate_HTMLSanitization(t *testing.T) {
	var created *model.Event
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			created = event
			return nil
		},
	}
	svc := NewEventService(repo)
	req := &model.CreateEventRequest{
		Title:       "Test",
		Description: `<b>bold</b><script>alert('xss')</script>`,
		StartTime:   "2026-02-15T10:00:00Z",
		EndTime:     "2026-02-15T11:00:00Z",
	}
	_, err := svc.Create(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Description != "<b>bold</b>" {
		t.Fatalf("expected sanitized description, got: %q", created.Description)
	}
}

// --- Update ---

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
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{
		Title: strPtr("Updated"),
	}
	e, err := svc.Update(1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Title != "Updated" {
		t.Fatalf("expected title 'Updated', got %q", e.Title)
	}
	// StartTime should remain unchanged
	if e.StartTime != "2026-02-15T10:00:00Z" {
		t.Fatalf("expected unchanged start_time, got %q", e.StartTime)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{Title: strPtr("X")}
	_, err := svc.Update(1, req)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestUpdate_ValidationFailure(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{Title: strPtr("")} // empty title not allowed
	_, err := svc.Update(1, req)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}
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
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{
		EndTime: strPtr("2026-02-15T09:00:00Z"),
	}
	_, err := svc.Update(1, req)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}
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
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{
		Duration: strPtr("PT2H"),
	}
	e, err := svc.Update(1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.EndTime != "2026-02-15T12:00:00Z" {
		t.Fatalf("expected end_time recomputed to 12:00, got %q", e.EndTime)
	}
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
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{
		StartTime: strPtr("2026-02-20"),
		EndTime:   strPtr("2026-02-22"),
	}
	e, err := svc.Update(1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.StartTime != "2026-02-20T00:00:00Z" {
		t.Fatalf("expected start normalized to midnight UTC, got %q", e.StartTime)
	}
	if e.EndTime != "2026-02-22T00:00:00Z" {
		t.Fatalf("expected end normalized to midnight UTC, got %q", e.EndTime)
	}
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
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{
		AllDay: boolPtr(true),
	}
	e, err := svc.Update(1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should normalize start to midnight and set end to next day
	if e.StartTime != "2026-02-15T00:00:00Z" {
		t.Fatalf("expected start normalized to midnight, got %q", e.StartTime)
	}
	if e.EndTime != "2026-02-16T00:00:00Z" {
		t.Fatalf("expected end set to next day, got %q", e.EndTime)
	}
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
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{
		Color:              strPtr("blue"),
		RecurrenceFreq:     strPtr("WEEKLY"),
		RecurrenceCount:    intPtr(10),
		RecurrenceUntil:    strPtr("2026-12-31T00:00:00Z"),
		RecurrenceInterval: intPtr(2),
		RecurrenceByDay:    strPtr("MO,WE"),
		RecurrenceByMonthDay: strPtr("1,15"),
		RecurrenceByMonth:  strPtr("1,6"),
		ExDates:            strPtr("2026-02-22T10:00:00Z"),
		RDates:             strPtr("2026-03-01T10:00:00Z"),
		Categories:         strPtr("work"),
		URL:                strPtr("https://example.com"),
		ReminderMinutes:    intPtr(30),
		Location:           strPtr("Room A"),
		Latitude:           float64Ptr(59.33),
		Longitude:          float64Ptr(18.07),
	}
	e, err := svc.Update(1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Color != "blue" {
		t.Fatalf("expected color 'blue', got %q", e.Color)
	}
	if e.RecurrenceFreq != "WEEKLY" {
		t.Fatalf("expected freq 'WEEKLY', got %q", e.RecurrenceFreq)
	}
	if e.RecurrenceCount != 10 {
		t.Fatalf("expected count 10, got %d", e.RecurrenceCount)
	}
	if e.RecurrenceInterval != 2 {
		t.Fatalf("expected interval 2, got %d", e.RecurrenceInterval)
	}
	if e.RecurrenceByDay != "MO,WE" {
		t.Fatalf("expected by_day 'MO,WE', got %q", e.RecurrenceByDay)
	}
	if e.RecurrenceByMonthDay != "1,15" {
		t.Fatalf("expected by_monthday '1,15', got %q", e.RecurrenceByMonthDay)
	}
	if e.RecurrenceByMonth != "1,6" {
		t.Fatalf("expected by_month '1,6', got %q", e.RecurrenceByMonth)
	}
	if e.Location != "Room A" {
		t.Fatalf("expected location 'Room A', got %q", e.Location)
	}
	if e.Latitude == nil || *e.Latitude != 59.33 {
		t.Fatalf("expected latitude 59.33, got %v", e.Latitude)
	}
	if e.Longitude == nil || *e.Longitude != 18.07 {
		t.Fatalf("expected longitude 18.07, got %v", e.Longitude)
	}
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
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{
		Description: strPtr(`<em>hi</em><script>bad</script>`),
	}
	e, err := svc.Update(1, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Description != "<em>hi</em>" {
		t.Fatalf("expected sanitized description, got %q", e.Description)
	}
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
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{Title: strPtr("New")}
	_, err := svc.Update(1, req)
	if !errors.Is(err, errRepo) {
		t.Fatalf("expected repo error, got: %v", err)
	}
}

// --- CreateOrUpdateOverride ---

func TestCreateOrUpdateOverride_NewOverride(t *testing.T) {
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
			return nil, nil // no existing override
		},
		createFn: func(event *model.Event) error {
			event.ID = 100
			return nil
		},
	}
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{Title: strPtr("Modified Instance")}
	e, err := svc.CreateOrUpdateOverride(parentID, "2026-02-08T09:00:00Z", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Title != "Modified Instance" {
		t.Fatalf("expected title 'Modified Instance', got %q", e.Title)
	}
	if e.RecurrenceParentID == nil || *e.RecurrenceParentID != parentID {
		t.Fatal("expected recurrence_parent_id to be set")
	}
	if e.RecurrenceOriginalStart != "2026-02-08T09:00:00Z" {
		t.Fatalf("expected original_start '2026-02-08T09:00:00Z', got %q", e.RecurrenceOriginalStart)
	}
	// EndTime should be computed from parent's duration (1h)
	if e.EndTime != "2026-02-08T10:00:00Z" {
		t.Fatalf("expected end_time '2026-02-08T10:00:00Z', got %q", e.EndTime)
	}
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
			// Return existing override for Update call
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
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{Title: strPtr("Updated Override")}
	e, err := svc.CreateOrUpdateOverride(parentID, "2026-02-08T09:00:00Z", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Title != "Updated Override" {
		t.Fatalf("expected title 'Updated Override', got %q", e.Title)
	}
}

func TestCreateOrUpdateOverride_ParentNotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{Title: strPtr("X")}
	_, err := svc.CreateOrUpdateOverride(999, "2026-02-08T09:00:00Z", req)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
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
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{Title: strPtr("X")}
	_, err := svc.CreateOrUpdateOverride(1, "2026-02-08T09:00:00Z", req)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}
}

func TestCreateOrUpdateOverride_InvalidInstanceStart(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{Title: strPtr("X")}
	_, err := svc.CreateOrUpdateOverride(1, "not-a-date", req)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}
}

func TestCreateOrUpdateOverride_NewOverrideWithAllFields(t *testing.T) {
	parentID := int64(10)
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return &model.Event{
				ID:             parentID,
				Title:          "Weekly",
				Description:    "Desc",
				StartTime:      "2026-02-01T09:00:00Z",
				EndTime:        "2026-02-01T10:00:00Z",
				RecurrenceFreq: "WEEKLY",
				Location:       "Office",
				Categories:     "work",
				URL:            "https://example.com",
				ReminderMinutes: 15,
				Latitude:       float64Ptr(59.0),
				Longitude:      float64Ptr(18.0),
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
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{
		Title:           strPtr("New Title"),
		Description:     strPtr("<b>bold</b><script>bad</script>"),
		StartTime:       strPtr("2026-02-08T10:00:00Z"),
		EndTime:         strPtr("2026-02-08T12:00:00Z"),
		AllDay:          boolPtr(false),
		Color:           strPtr("red"),
		Duration:        strPtr("PT3H"),
		Categories:      strPtr("meeting"),
		URL:             strPtr("https://new.example.com"),
		ReminderMinutes: intPtr(30),
		Location:        strPtr("Home"),
		Latitude:        float64Ptr(60.0),
		Longitude:       float64Ptr(19.0),
	}
	e, err := svc.CreateOrUpdateOverride(parentID, "2026-02-08T09:00:00Z", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Title != "New Title" {
		t.Fatalf("expected title 'New Title', got %q", e.Title)
	}
	if e.Description != "<b>bold</b>" {
		t.Fatalf("expected sanitized description, got %q", e.Description)
	}
	if e.Color != "red" {
		t.Fatalf("expected color 'red', got %q", e.Color)
	}
	if !e.AllDay {
		// AllDay was set to false explicitly
	}
	if e.Categories != "meeting" {
		t.Fatalf("expected categories 'meeting', got %q", e.Categories)
	}
	if e.URL != "https://new.example.com" {
		t.Fatalf("expected URL updated, got %q", e.URL)
	}
	if e.ReminderMinutes != 30 {
		t.Fatalf("expected reminder_minutes 30, got %d", e.ReminderMinutes)
	}
	if e.Location != "Home" {
		t.Fatalf("expected location 'Home', got %q", e.Location)
	}
	if e.Latitude == nil || *e.Latitude != 60.0 {
		t.Fatalf("expected latitude 60.0, got %v", e.Latitude)
	}
	if e.Longitude == nil || *e.Longitude != 19.0 {
		t.Fatalf("expected longitude 19.0, got %v", e.Longitude)
	}
	// Duration was set, so EndTime should be recomputed from StartTime + Duration
	if e.Duration != "PT3H" {
		t.Fatalf("expected duration 'PT3H', got %q", e.Duration)
	}
	if e.EndTime != "2026-02-08T13:00:00Z" {
		t.Fatalf("expected end_time recomputed from start+duration, got %q", e.EndTime)
	}
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
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{Title: strPtr("X")}
	_, err := svc.CreateOrUpdateOverride(parentID, "2026-02-08T09:00:00Z", req)
	if !errors.Is(err, errRepo) {
		t.Fatalf("expected repo error, got: %v", err)
	}
}

func TestCreateOrUpdateOverride_ValidationFailure(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo)
	req := &model.UpdateEventRequest{Title: strPtr("")} // empty title not allowed
	_, err := svc.CreateOrUpdateOverride(1, "2026-02-08T09:00:00Z", req)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}
}

// --- ImportSingle ---

func TestImportSingle_SingleEvent(t *testing.T) {
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			event.ID = 1
			return nil
		},
	}
	svc := NewEventService(repo)
	events := []model.Event{{
		Title:     "Imported",
		StartTime: "2026-02-15T10:00:00Z",
		EndTime:   "2026-02-15T11:00:00Z",
	}}
	e, err := svc.ImportSingle(events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Title != "Imported" {
		t.Fatalf("expected title 'Imported', got %q", e.Title)
	}
}

func TestImportSingle_NoEvents(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo)
	_, err := svc.ImportSingle(nil)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}
}

func TestImportSingle_MultipleParents(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo)
	events := []model.Event{
		{Title: "A", StartTime: "2026-02-15T10:00:00Z", EndTime: "2026-02-15T11:00:00Z"},
		{Title: "B", StartTime: "2026-02-16T10:00:00Z", EndTime: "2026-02-16T11:00:00Z"},
	}
	_, err := svc.ImportSingle(events)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for multiple events, got: %v", err)
	}
}

func TestImportSingle_RejectsOverrides(t *testing.T) {
	parentID := int64(5)
	repo := &mockRepo{}
	svc := NewEventService(repo)

	t.Run("parent with override", func(t *testing.T) {
		events := []model.Event{
			{Title: "Parent", StartTime: "2026-02-15T10:00:00Z", EndTime: "2026-02-15T11:00:00Z"},
			{Title: "Override", StartTime: "2026-02-22T10:00:00Z", EndTime: "2026-02-22T11:00:00Z",
				RecurrenceParentID: &parentID, RecurrenceOriginalStart: "2026-02-22T10:00:00Z"},
		}
		_, err := svc.ImportSingle(events)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("expected ErrValidation, got: %v", err)
		}
	})

	t.Run("only override", func(t *testing.T) {
		events := []model.Event{
			{Title: "Override Only", StartTime: "2026-02-22T10:00:00Z", EndTime: "2026-02-22T11:00:00Z",
				RecurrenceParentID: &parentID, RecurrenceOriginalStart: "2026-02-22T10:00:00Z"},
		}
		_, err := svc.ImportSingle(events)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("expected ErrValidation, got: %v", err)
		}
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
	svc := NewEventService(repo)
	events := []model.Event{{
		Title:     "All Day",
		StartTime: "2026-02-15T00:00:00Z",
		EndTime:   "2026-02-16T00:00:00Z",
		AllDay:    true,
	}}
	_, err := svc.ImportSingle(events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// AllDay events should have their times normalized via Validate
	if created.StartTime != "2026-02-15T00:00:00Z" {
		t.Fatalf("expected start_time '2026-02-15T00:00:00Z', got %q", created.StartTime)
	}
}

func TestImportSingle_ValidationFailure(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo)
	events := []model.Event{{
		Title:     "", // empty title
		StartTime: "2026-02-15T10:00:00Z",
		EndTime:   "2026-02-15T11:00:00Z",
	}}
	_, err := svc.ImportSingle(events)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}
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
	svc := NewEventService(repo)
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
			StartTime:               "2026-02-08T10:00:00Z",
			EndTime:                 "2026-02-08T11:30:00Z",
			RecurrenceOriginalStart: "2026-02-08T10:00:00Z",
			ImportUID:               "uid-123",
		},
	}
	count, err := svc.Import(events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 imported, got %d", count)
	}
	if len(createdEvents) != 2 {
		t.Fatalf("expected 2 created events, got %d", len(createdEvents))
	}
	// Second should be override linked to first
	override := createdEvents[1]
	if override.RecurrenceParentID == nil || *override.RecurrenceParentID != 1 {
		t.Fatal("expected override to reference parent ID 1")
	}
}

func TestImport_SkipsInvalidEvents(t *testing.T) {
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			event.ID = 1
			return nil
		},
	}
	svc := NewEventService(repo)
	events := []model.Event{
		{Title: "", StartTime: "2026-02-15T10:00:00Z", EndTime: "2026-02-15T11:00:00Z"}, // invalid: no title
		{Title: "Valid", StartTime: "2026-02-15T10:00:00Z", EndTime: "2026-02-15T11:00:00Z"},
	}
	count, err := svc.Import(events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 imported (invalid skipped), got %d", count)
	}
}

func TestImport_OverrideWithoutParent(t *testing.T) {
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			event.ID = 1
			return nil
		},
	}
	svc := NewEventService(repo)
	events := []model.Event{
		{
			Title:                   "Orphan Override",
			StartTime:               "2026-02-08T10:00:00Z",
			EndTime:                 "2026-02-08T11:00:00Z",
			RecurrenceOriginalStart: "2026-02-08T10:00:00Z",
			ImportUID:               "uid-no-parent",
		},
	}
	count, err := svc.Import(events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 imported (orphan override skipped), got %d", count)
	}
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
	svc := NewEventService(repo)
	events := []model.Event{{
		Title:     "All Day Import",
		StartTime: "2026-03-10T00:00:00Z",
		EndTime:   "2026-03-11T00:00:00Z",
		AllDay:    true,
	}}
	count, err := svc.Import(events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 imported, got %d", count)
	}
	if created.StartTime != "2026-03-10T00:00:00Z" {
		t.Fatalf("expected normalized start_time, got %q", created.StartTime)
	}
}

func TestImport_RepoCreateError(t *testing.T) {
	repo := &mockRepo{
		createFn: func(event *model.Event) error {
			return errRepo
		},
	}
	svc := NewEventService(repo)
	events := []model.Event{
		{Title: "Test", StartTime: "2026-02-15T10:00:00Z", EndTime: "2026-02-15T11:00:00Z"},
	}
	count, err := svc.Import(events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err) // Import continues on create errors
	}
	if count != 0 {
		t.Fatalf("expected 0 imported (create failed), got %d", count)
	}
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
	svc := NewEventService(repo)
	e, err := svc.AddExDate(1, "2026-02-15T09:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "2026-02-08T09:00:00Z,2026-02-15T09:00:00Z"
	if updated.ExDates != expected {
		t.Fatalf("expected exdates %q, got %q", expected, updated.ExDates)
	}
	if e.ExDates != expected {
		t.Fatalf("expected exdates %q, got %q", expected, e.ExDates)
	}
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
	svc := NewEventService(repo)
	_, err := svc.AddExDate(1, "2026-02-08T09:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ExDates != "2026-02-08T09:00:00Z" {
		t.Fatalf("expected single exdate, got %q", updated.ExDates)
	}
}

func TestAddExDate_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo)
	_, err := svc.AddExDate(999, "2026-02-08T09:00:00Z")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
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
	svc := NewEventService(repo)
	_, err := svc.AddExDate(1, "2026-02-08T09:00:00Z")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}
}

func TestAddExDate_InvalidFormat(t *testing.T) {
	repo := &mockRepo{}
	svc := NewEventService(repo)
	_, err := svc.AddExDate(1, "not-a-date")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}
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
	svc := NewEventService(repo)
	_, err := svc.AddExDate(1, "2026-02-08T09:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != overrideID {
		t.Fatalf("expected override %d to be deleted, deleted %d", overrideID, deletedID)
	}
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
	svc := NewEventService(repo)
	_, err := svc.RemoveExDate(1, "2026-02-15T09:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "2026-02-08T09:00:00Z,2026-02-22T09:00:00Z"
	if updated.ExDates != expected {
		t.Fatalf("expected exdates %q, got %q", expected, updated.ExDates)
	}
}

func TestRemoveExDate_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(id int64) (*model.Event, error) {
			return nil, nil
		},
	}
	svc := NewEventService(repo)
	_, err := svc.RemoveExDate(999, "2026-02-08T09:00:00Z")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
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
	svc := NewEventService(repo)
	err := svc.Delete(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteByParentCalled {
		t.Fatal("expected DeleteByParentID to be called")
	}
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
	svc := NewEventService(repo)
	err := svc.Delete(999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
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
	svc := NewEventService(repo)
	err := svc.Delete(1)
	if !errors.Is(err, errRepo) {
		t.Fatalf("expected repo error, got: %v", err)
	}
}
