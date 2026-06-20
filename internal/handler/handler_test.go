package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mikaelstaldal/mycal/internal/api"
	"github.com/mikaelstaldal/mycal/internal/handler"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/service"
)

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := repository.OpenDB(":memory:", 0)
	require.NoError(t, err, "open db")
	repo, err := repository.NewSQLiteRepository(db)
	require.NoError(t, err, "init repo")
	calSvc := service.NewCalendarService(repo)
	svc := service.NewEventService(repo, repo)
	prefSvc := service.NewPreferencesService(repo)
	feedSvc := service.NewFeedService(repo, repo, repo)
	router := handler.NewRouter(svc, prefSvc, feedSvc, calSvc)
	ts := httptest.NewServer(router)
	t.Cleanup(func() {
		ts.Close()
		db.Close()
	})
	return ts
}

// marshalBody ensures pointer-receiver MarshalJSON methods are reachable.
func marshalBody(body any) ([]byte, error) {
	v := reflect.ValueOf(body)
	if v.Kind() != reflect.Pointer {
		ptr := reflect.New(v.Type())
		ptr.Elem().Set(v)
		return json.Marshal(ptr.Interface())
	}
	return json.Marshal(body)
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	data, err := marshalBody(body)
	require.NoError(t, err, "marshal")
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	require.NoError(t, err, "post")
	return resp
}

func postICS(t *testing.T, url string, icsContent string) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "text/calendar", strings.NewReader(icsContent))
	require.NoError(t, err, "post")
	return resp
}

func patchJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	data, err := marshalBody(body)
	require.NoError(t, err, "marshal")
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
	require.NoError(t, err, "new request")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "put")
	return resp
}

func doDelete(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err, "new request")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "delete")
	return resp
}

func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	err := json.NewDecoder(resp.Body).Decode(&v)
	require.NoError(t, err, "decode")
	return v
}

func createTestEvent(t *testing.T, ts *httptest.Server) api.Event {
	t.Helper()
	body := api.CreateEventRequest{
		Title:     "Test Event",
		AllDay:    false,
		StartTime: api.NewOptDateTime(mustTime("2026-03-15T10:00:00Z")),
		EndTime:   api.NewOptDateTime(mustTime("2026-03-15T11:00:00Z")),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "create event")
	return decodeJSON[api.Event](t, resp)
}

// --- CRUD tests ---

func TestCreateEvent(t *testing.T) {
	ts := setupTestServer(t)
	body := api.CreateEventRequest{
		Title:       "Meeting",
		Description: api.NewOptString("Team sync"),
		AllDay:      false,
		StartTime:   api.NewOptDateTime(mustTime("2026-03-15T10:00:00Z")),
		EndTime:     api.NewOptDateTime(mustTime("2026-03-15T11:00:00Z")),
		Location:    api.NewOptString("Room 42"),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	event := decodeJSON[api.Event](t, resp)
	assert.NotEmpty(t, event.ID)
	assert.Equal(t, "Meeting", event.Title)
	assert.Equal(t, "Team sync", event.Description.Value)
	assert.Equal(t, "Room 42", event.Location.Value)
	assert.True(t, event.CreatedAt.Set, "expected non-empty created_at")
	assert.True(t, event.UpdatedAt.Set, "expected non-empty updated_at")
}

func TestCreateEvent_ValidationErrors(t *testing.T) {
	ts := setupTestServer(t)

	tests := []struct {
		name string
		body api.CreateEventRequest
	}{
		{
			name: "missing title",
			body: api.CreateEventRequest{
				AllDay:    false,
				StartTime: api.NewOptDateTime(mustTime("2026-03-15T10:00:00Z")),
				EndTime:   api.NewOptDateTime(mustTime("2026-03-15T11:00:00Z")),
			},
		},
		{
			name: "missing start_time",
			body: api.CreateEventRequest{
				Title:   "Event",
				AllDay:  false,
				EndTime: api.NewOptDateTime(mustTime("2026-03-15T11:00:00Z")),
			},
		},
		{
			name: "end before start",
			body: api.CreateEventRequest{
				Title:     "Event",
				AllDay:    false,
				StartTime: api.NewOptDateTime(mustTime("2026-03-15T11:00:00Z")),
				EndTime:   api.NewOptDateTime(mustTime("2026-03-15T10:00:00Z")),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := postJSON(t, ts.URL+"/api/v1/events", tc.body)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestGetEvent(t *testing.T) {
	ts := setupTestServer(t)
	created := createTestEvent(t, ts)

	resp, err := http.Get(ts.URL + "/api/v1/events/" + created.ID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	event := decodeJSON[api.Event](t, resp)
	assert.Equal(t, created.ID, event.ID)
	assert.Equal(t, "Test Event", event.Title)
}

func TestGetEvent_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/events/99999")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetEvent_InvalidID(t *testing.T) {
	ts := setupTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/events/abc")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateEvent(t *testing.T) {
	ts := setupTestServer(t)
	created := createTestEvent(t, ts)

	resp := patchJSON(t, ts.URL+"/api/v1/events/"+created.ID, api.UpdateEventRequest{
		Title: api.NewOptString("Updated Title"),
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	updated := decodeJSON[api.Event](t, resp)
	assert.Equal(t, "Updated Title", updated.Title)
	// Unchanged fields should be preserved
	assert.True(t, updated.StartTime.Value.Equal(created.StartTime.Value))
}

func TestUpdateEvent_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	resp := patchJSON(t, ts.URL+"/api/v1/events/99999", api.UpdateEventRequest{
		Title: api.NewOptString("X"),
	})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteEvent(t *testing.T) {
	ts := setupTestServer(t)
	created := createTestEvent(t, ts)

	resp := doDelete(t, ts.URL+"/api/v1/events/"+created.ID)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Confirm it's gone
	resp2, err := http.Get(ts.URL + "/api/v1/events/" + created.ID)
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

func TestDeleteEvent_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	resp := doDelete(t, ts.URL+"/api/v1/events/99999")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// --- List and search tests ---

func TestListEvents(t *testing.T) {
	ts := setupTestServer(t)

	// Create events at different times
	for _, start := range []string{"2026-03-10T10:00:00Z", "2026-03-15T10:00:00Z", "2026-03-20T10:00:00Z"} {
		startT := mustTime(start)
		postJSON(t, ts.URL+"/api/v1/events", api.CreateEventRequest{
			Title:     "Event at " + start,
			AllDay:    false,
			StartTime: api.NewOptDateTime(startT),
			EndTime:   api.NewOptDateTime(startT.Add(time.Hour)),
		}).Body.Close()
	}

	// Query a range that includes only the middle event
	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-03-14T00:00:00Z&to=2026-03-16T00:00:00Z")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	events := decodeJSON[[]api.Event](t, resp)
	assert.Len(t, events, 1)
	assert.Equal(t, "Event at 2026-03-15T10:00:00Z", events[0].Title)
}

func TestListEvents_MissingParams(t *testing.T) {
	ts := setupTestServer(t)

	tests := []string{
		"/api/v1/events",
		"/api/v1/events?from=2026-03-01T00:00:00Z",
		"/api/v1/events?to=2026-03-31T00:00:00Z",
	}
	for _, path := range tests {
		resp, err := http.Get(ts.URL + path)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "GET %s", path)
	}
}

func TestSearchEvents(t *testing.T) {
	ts := setupTestServer(t)

	postJSON(t, ts.URL+"/api/v1/events", api.CreateEventRequest{
		Title:     "Go Conference",
		AllDay:    false,
		StartTime: api.NewOptDateTime(mustTime("2026-03-15T10:00:00Z")),
		EndTime:   api.NewOptDateTime(mustTime("2026-03-15T18:00:00Z")),
	}).Body.Close()
	postJSON(t, ts.URL+"/api/v1/events", api.CreateEventRequest{
		Title:     "Lunch Break",
		AllDay:    false,
		StartTime: api.NewOptDateTime(mustTime("2026-03-15T12:00:00Z")),
		EndTime:   api.NewOptDateTime(mustTime("2026-03-15T13:00:00Z")),
	}).Body.Close()

	resp, err := http.Get(ts.URL + "/api/v1/events?q=Conference")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	events := decodeJSON[[]api.Event](t, resp)
	assert.Len(t, events, 1)
	assert.Equal(t, "Go Conference", events[0].Title)
}

// --- All-day events ---

func TestCreateAllDayEvent(t *testing.T) {
	ts := setupTestServer(t)
	body := api.CreateEventRequest{
		Title:     "Holiday",
		AllDay:    true,
		StartDate: api.NewOptDate(mustDate("2026-06-15")),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	event := decodeJSON[api.Event](t, resp)
	assert.True(t, event.AllDay.Value, "expected all_day to be true")
	assert.Equal(t, "Holiday", event.Title)
}

// --- Recurring events ---

func TestRecurringEventExpansion(t *testing.T) {
	ts := setupTestServer(t)

	// Create a weekly recurring event
	body := api.CreateEventRequest{
		Title:           "Weekly Standup",
		AllDay:          false,
		StartTime:       api.NewOptDateTime(mustTime("2026-03-02T09:00:00Z")),
		EndTime:         api.NewOptDateTime(mustTime("2026-03-02T09:30:00Z")),
		RecurrenceFreq:  api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqWEEKLY),
		RecurrenceCount: api.NewOptInt(10),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// List over a 3-week range — should get 3 instances
	listResp, err := http.Get(ts.URL + "/api/v1/events?from=2026-03-01T00:00:00Z&to=2026-03-22T00:00:00Z")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	events := decodeJSON[[]api.Event](t, listResp)
	assert.GreaterOrEqual(t, len(events), 3, "got %d events, want at least 3 recurring instances", len(events))
}

// --- Delete with EXDATE ---

func TestDeleteWithInstanceStart(t *testing.T) {
	ts := setupTestServer(t)

	body := api.CreateEventRequest{
		Title:           "Daily Standup",
		AllDay:          false,
		StartTime:       api.NewOptDateTime(mustTime("2026-03-01T09:00:00Z")),
		EndTime:         api.NewOptDateTime(mustTime("2026-03-01T09:30:00Z")),
		RecurrenceFreq:  api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqDAILY),
		RecurrenceCount: api.NewOptInt(30),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	created := decodeJSON[api.Event](t, resp)

	// Delete a single instance via composite ID
	compositeID := created.ID + "_2026-03-05T09:00:00Z"
	delResp := doDelete(t, ts.URL+"/api/v1/events/"+url.PathEscape(compositeID))
	require.Equal(t, http.StatusOK, delResp.StatusCode)
	updated := decodeJSON[api.Event](t, delResp)
	assert.Contains(t, updated.Exdates.Value, "2026-03-05T09:00:00Z")
}

// --- iCal export ---

func TestExportICal(t *testing.T) {
	ts := setupTestServer(t)
	createTestEvent(t, ts)

	resp, err := http.Get(ts.URL + "/api/v1/events.ics")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	ct := resp.Header.Get("Content-Type")
	assert.True(t, strings.HasPrefix(ct, "text/calendar"), "content-type = %q", ct)
}

func TestExportSingleEventICal(t *testing.T) {
	ts := setupTestServer(t)
	event := createTestEvent(t, ts)

	resp, err := http.Get(ts.URL + "/api/v1/events/" + event.ID + "/ics")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	ct := resp.Header.Get("Content-Type")
	assert.True(t, strings.HasPrefix(ct, "text/calendar"), "content-type = %q", ct)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "BEGIN:VCALENDAR")
	assert.Contains(t, string(body), "SUMMARY:Test Event")
}

func TestExportSingleEventICal_NotFound(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/events/99999/ics")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// --- iCal import ---

func TestImportEvents(t *testing.T) {
	ts := setupTestServer(t)
	ics := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
DTSTART:20260401T100000Z
DTEND:20260401T110000Z
SUMMARY:Imported Event 1
END:VEVENT
BEGIN:VEVENT
DTSTART:20260402T100000Z
DTEND:20260402T110000Z
SUMMARY:Imported Event 2
END:VEVENT
END:VCALENDAR`

	resp := postICS(t, ts.URL+"/api/v1/import", ics)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	result := decodeJSON[map[string]int](t, resp)
	assert.Equal(t, 2, result["imported"])
}

func TestImportSingleEvent(t *testing.T) {
	ts := setupTestServer(t)
	ics := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
DTSTART:20260501T140000Z
DTEND:20260501T150000Z
SUMMARY:Single Import
LOCATION:Office
END:VEVENT
END:VCALENDAR`

	resp := postICS(t, ts.URL+"/api/v1/import-single", ics)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	event := decodeJSON[api.Event](t, resp)
	assert.Equal(t, "Single Import", event.Title)
	assert.Equal(t, "Office", event.Location.Value)
}

// --- RECURRENCE-ID override tests ---

func TestOverrideInstance(t *testing.T) {
	ts := setupTestServer(t)

	// Create a weekly recurring event
	body := api.CreateEventRequest{
		Title:           "Weekly Standup",
		AllDay:          false,
		StartTime:       api.NewOptDateTime(mustTime("2026-03-02T09:00:00Z")),
		EndTime:         api.NewOptDateTime(mustTime("2026-03-02T09:30:00Z")),
		RecurrenceFreq:  api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqWEEKLY),
		RecurrenceCount: api.NewOptInt(10),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	created := decodeJSON[api.Event](t, resp)

	// Override the 2nd instance (2026-03-09) using composite ID
	compositeID := created.ID + "_2026-03-09T09:00:00Z"
	resp = patchJSON(t, ts.URL+"/api/v1/events/"+url.PathEscape(compositeID),
		api.UpdateEventRequest{Title: api.NewOptString("Modified Standup")})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	override := decodeJSON[api.Event](t, resp)
	assert.Equal(t, "Modified Standup", override.Title)
	assert.True(t, override.RecurrenceParentID.Set, "expected non-nil recurrence_parent_id on override")

	// List events and verify the override replaces the original instance
	listResp, err := http.Get(ts.URL + "/api/v1/events?from=2026-03-01T00:00:00Z&to=2026-03-22T00:00:00Z")
	require.NoError(t, err)
	events := decodeJSON[[]api.Event](t, listResp)
	found := false
	for _, e := range events {
		if e.Title == "Modified Standup" {
			found = true
		}
		// The original "Weekly Standup" at 2026-03-09 should be replaced
		if e.Title == "Weekly Standup" && e.StartTime.Value.Format(time.RFC3339) == "2026-03-09T09:00:00Z" {
			t.Error("original instance at 2026-03-09 should have been replaced by override")
		}
	}
	assert.True(t, found, "expected to find 'Modified Standup' in list")
}

func TestDeleteParentDeletesOverrides(t *testing.T) {
	ts := setupTestServer(t)

	body := api.CreateEventRequest{
		Title:           "Series",
		AllDay:          false,
		StartTime:       api.NewOptDateTime(mustTime("2026-04-01T10:00:00Z")),
		EndTime:         api.NewOptDateTime(mustTime("2026-04-01T11:00:00Z")),
		RecurrenceFreq:  api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqDAILY),
		RecurrenceCount: api.NewOptInt(5),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	created := decodeJSON[api.Event](t, resp)

	// Create an override using composite ID
	compositeID := created.ID + "_2026-04-02T10:00:00Z"
	patchJSON(t, ts.URL+"/api/v1/events/"+url.PathEscape(compositeID),
		api.UpdateEventRequest{Title: api.NewOptString("Override")})

	// Delete the parent
	delResp := doDelete(t, ts.URL+"/api/v1/events/"+created.ID)
	require.Equal(t, http.StatusNoContent, delResp.StatusCode)
	delResp.Body.Close()

	// Verify override is also gone
	listResp, _ := http.Get(ts.URL + "/api/v1/events?from=2026-04-01T00:00:00Z&to=2026-04-10T00:00:00Z")
	events := decodeJSON[[]api.Event](t, listResp)
	for _, e := range events {
		assert.NotEqual(t, "Override", e.Title, "override should have been deleted with parent")
	}
}

func TestDeleteInstanceWithOverride(t *testing.T) {
	ts := setupTestServer(t)

	body := api.CreateEventRequest{
		Title:           "Series",
		AllDay:          false,
		StartTime:       api.NewOptDateTime(mustTime("2026-04-01T10:00:00Z")),
		EndTime:         api.NewOptDateTime(mustTime("2026-04-01T11:00:00Z")),
		RecurrenceFreq:  api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqDAILY),
		RecurrenceCount: api.NewOptInt(5),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	created := decodeJSON[api.Event](t, resp)

	// Create an override for Apr 2 using composite ID
	compositeID := created.ID + "_2026-04-02T10:00:00Z"
	patchJSON(t, ts.URL+"/api/v1/events/"+url.PathEscape(compositeID),
		api.UpdateEventRequest{Title: api.NewOptString("Override")})

	// Delete that instance via composite ID
	delResp := doDelete(t, ts.URL+"/api/v1/events/"+url.PathEscape(compositeID))
	require.Equal(t, http.StatusOK, delResp.StatusCode)
	delResp.Body.Close()

	// Verify override is gone and instance is excluded
	listResp, _ := http.Get(ts.URL + "/api/v1/events?from=2026-04-01T00:00:00Z&to=2026-04-10T00:00:00Z")
	events := decodeJSON[[]api.Event](t, listResp)
	for _, e := range events {
		assert.NotEqual(t, "Override", e.Title, "override should have been deleted when instance was excluded")
	}
}

// --- Duration tests ---

func TestCreateEventWithDuration(t *testing.T) {
	ts := setupTestServer(t)
	body := api.CreateEventRequest{
		Title:     "Quick Meeting",
		AllDay:    false,
		StartTime: api.NewOptDateTime(mustTime("2026-03-15T10:00:00Z")),
		Duration:  api.NewOptString("PT1H30M"),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	event := decodeJSON[api.Event](t, resp)
	wantEnd := mustTime("2026-03-15T11:30:00Z")
	assert.True(t, event.EndTime.Value.Equal(wantEnd))
	assert.Equal(t, "PT1H30M", event.Duration.Value)
}

func TestCreateEventDurationAndEndTimeConflict(t *testing.T) {
	ts := setupTestServer(t)
	body := api.CreateEventRequest{
		Title:     "Conflict",
		AllDay:    false,
		StartTime: api.NewOptDateTime(mustTime("2026-03-15T10:00:00Z")),
		EndTime:   api.NewOptDateTime(mustTime("2026-03-15T11:00:00Z")),
		Duration:  api.NewOptString("PT1H"),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- Categories tests ---

func TestCreateEventWithCategories(t *testing.T) {
	ts := setupTestServer(t)
	body := api.CreateEventRequest{
		Title:      "Tagged Event",
		AllDay:     false,
		StartTime:  api.NewOptDateTime(mustTime("2026-03-15T10:00:00Z")),
		EndTime:    api.NewOptDateTime(mustTime("2026-03-15T11:00:00Z")),
		Categories: api.NewOptString("Work,Meeting"),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	event := decodeJSON[api.Event](t, resp)
	assert.Equal(t, "Work,Meeting", event.Categories.Value)

	// Update categories
	resp = patchJSON(t, ts.URL+"/api/v1/events/"+event.ID, api.UpdateEventRequest{
		Categories: api.NewOptString("Personal"),
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	updated := decodeJSON[api.Event](t, resp)
	assert.Equal(t, "Personal", updated.Categories.Value)
}

// --- URL tests ---

func TestCreateEventWithURL(t *testing.T) {
	ts := setupTestServer(t)
	u, _ := url.Parse("https://example.com/meeting")
	body := api.CreateEventRequest{
		Title:     "Linked Event",
		AllDay:    false,
		StartTime: api.NewOptDateTime(mustTime("2026-03-15T10:00:00Z")),
		EndTime:   api.NewOptDateTime(mustTime("2026-03-15T11:00:00Z")),
		URL:       api.NewOptURI(*u),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	event := decodeJSON[api.Event](t, resp)
	assert.Equal(t, "https://example.com/meeting", event.URL.Value.String())
}

func TestCreateEventWithBadURL(t *testing.T) {
	ts := setupTestServer(t)
	u, _ := url.Parse("ftp://example.com")
	body := api.CreateEventRequest{
		Title:     "Bad URL",
		AllDay:    false,
		StartTime: api.NewOptDateTime(mustTime("2026-03-15T10:00:00Z")),
		EndTime:   api.NewOptDateTime(mustTime("2026-03-15T11:00:00Z")),
		URL:       api.NewOptURI(*u),
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- iCal import/export with new properties ---

func TestImportExportWithRecurrenceID(t *testing.T) {
	ts := setupTestServer(t)
	ics := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
UID:recurring-1@test
DTSTART:20260401T100000Z
DTEND:20260401T110000Z
SUMMARY:Weekly
RRULE:FREQ=WEEKLY;COUNT=4
END:VEVENT
BEGIN:VEVENT
UID:recurring-1@test
RECURRENCE-ID:20260408T100000Z
DTSTART:20260408T140000Z
DTEND:20260408T150000Z
SUMMARY:Weekly (moved)
END:VEVENT
END:VCALENDAR`

	resp := postICS(t, ts.URL+"/api/v1/import", ics)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	result := decodeJSON[map[string]int](t, resp)
	assert.Equal(t, 2, result["imported"])

	// Export and verify RECURRENCE-ID in output
	exportResp, err := http.Get(ts.URL + "/api/v1/events.ics")
	require.NoError(t, err)
	defer exportResp.Body.Close()
	body, _ := io.ReadAll(exportResp.Body)
	icsStr := string(body)
	assert.Contains(t, icsStr, "RECURRENCE-ID:")
}

func TestImportExportDuration(t *testing.T) {
	ts := setupTestServer(t)
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
DTSTART:20260501T100000Z
DURATION:PT2H
SUMMARY:Duration Event
END:VEVENT
END:VCALENDAR`

	resp := postICS(t, ts.URL+"/api/v1/import", ics)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	result := decodeJSON[map[string]int](t, resp)
	assert.Equal(t, 1, result["imported"])

	// Export and verify DURATION in output
	exportResp, err := http.Get(ts.URL + "/api/v1/events.ics")
	require.NoError(t, err)
	defer exportResp.Body.Close()
	body, _ := io.ReadAll(exportResp.Body)
	icsStr := string(body)
	assert.Contains(t, icsStr, "DURATION:PT2H")
}

func TestImportExportCategories(t *testing.T) {
	ts := setupTestServer(t)
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
DTSTART:20260501T100000Z
DTEND:20260501T110000Z
SUMMARY:Categorized
CATEGORIES:Work,Meeting
END:VEVENT
END:VCALENDAR`

	resp := postICS(t, ts.URL+"/api/v1/import", ics)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Export and verify CATEGORIES in output
	exportResp, err := http.Get(ts.URL + "/api/v1/events.ics")
	require.NoError(t, err)
	defer exportResp.Body.Close()
	body, _ := io.ReadAll(exportResp.Body)
	icsStr := string(body)
	assert.Contains(t, icsStr, "CATEGORIES:")
}

func TestImportExportURL(t *testing.T) {
	ts := setupTestServer(t)
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
DTSTART:20260501T100000Z
DTEND:20260501T110000Z
SUMMARY:Linked
URL:https://example.com/event
END:VEVENT
END:VCALENDAR`

	resp := postICS(t, ts.URL+"/api/v1/import", ics)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Export and verify URL in output
	exportResp, err := http.Get(ts.URL + "/api/v1/events.ics")
	require.NoError(t, err)
	defer exportResp.Body.Close()
	body, _ := io.ReadAll(exportResp.Body)
	icsStr := string(body)
	assert.Contains(t, icsStr, "URL:https://example.com/event")
}

// --- Preferences tests ---

func TestGetPreferencesDefaults(t *testing.T) {
	ts := setupTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/preferences")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	prefs := decodeJSON[map[string]string](t, resp)
	// Preferences should be empty (defaultEventColor moved to calendars)
	assert.Empty(t, prefs)
}

func TestUpdatePreferencesNoAllowedKeys(t *testing.T) {
	ts := setupTestServer(t)

	// defaultEventColor is no longer a preference - should fail
	data, _ := json.Marshal(map[string]string{"defaultEventColor": "red"})
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/preferences", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdatePreferencesUnknownKey(t *testing.T) {
	ts := setupTestServer(t)
	data, _ := json.Marshal(map[string]string{"unknownKey": "value"})
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/preferences", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
