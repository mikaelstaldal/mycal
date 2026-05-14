package api_test

// Integration tests written from the perspective of an external REST API consumer.
// All request/response bodies use plain structs and maps — no generated api package types.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mikaelstaldal/mycal/internal/handler"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/service"
)

// ---- plain JSON types used as an external client would define them ----

type jsonEvent struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	AllDay          bool     `json:"all_day"`
	StartTime       string   `json:"start_time,omitempty"`
	EndTime         string   `json:"end_time,omitempty"`
	StartDate       string   `json:"start_date,omitempty"`
	EndDate         string   `json:"end_date,omitempty"`
	Description     string   `json:"description,omitempty"`
	Location        string   `json:"location,omitempty"`
	Latitude        *float64 `json:"latitude,omitempty"`
	Longitude       *float64 `json:"longitude,omitempty"`
	ReminderMinutes *int     `json:"reminder_minutes,omitempty"`
	Categories      string   `json:"categories,omitempty"`
	Color           string   `json:"color,omitempty"`
	CalendarID      int64    `json:"calendar_id,omitempty"`
	CalendarName    string   `json:"calendar_name,omitempty"`
	RecurrenceFreq  string   `json:"recurrence_freq,omitempty"`
	RecurrenceCount *int     `json:"recurrence_count,omitempty"`
	Exdates         string   `json:"exdates,omitempty"`
	Duration        string   `json:"duration,omitempty"`
}

type jsonCalendar struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type jsonError struct {
	Error string `json:"error"`
}

// ---- helpers ----

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	repo, err := repository.NewSQLiteRepository(db)
	require.NoError(t, err)
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

func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return b
}

func postJSONMap(t *testing.T, url string, body map[string]any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := http.Post(url, "application/json", strings.NewReader(string(data)))
	require.NoError(t, err)
	return resp
}

func patchJSONMap(t *testing.T, rawURL string, body map[string]any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPatch, rawURL, strings.NewReader(string(data)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func decodeEvent(t *testing.T, b []byte) jsonEvent {
	t.Helper()
	var e jsonEvent
	err := json.Unmarshal(b, &e)
	require.NoError(t, err, "body: %s", b)
	return e
}

func decodeEvents(t *testing.T, b []byte) []jsonEvent {
	t.Helper()
	var events []jsonEvent
	err := json.Unmarshal(b, &events)
	require.NoError(t, err, "body: %s", b)
	return events
}

func decodeCalendars(t *testing.T, b []byte) []jsonCalendar {
	t.Helper()
	var cals []jsonCalendar
	err := json.Unmarshal(b, &cals)
	require.NoError(t, err, "body: %s", b)
	return cals
}

// createJSONEvent posts a timed event and returns the decoded body.
func createJSONEvent(t *testing.T, ts *httptest.Server, body map[string]any) jsonEvent {
	t.Helper()
	resp := postJSONMap(t, ts.URL+"/api/v1/events", body)
	b := readBody(t, resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "body: %s", b)
	return decodeEvent(t, b)
}

// ---- Calendar tests ----

func TestCalendars_DefaultCalendarExists(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/calendars")
	require.NoError(t, err)
	cals := decodeCalendars(t, readBody(t, resp))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, cals, "expected at least one calendar (the default)")
	var hasDefault bool
	for _, c := range cals {
		if c.ID == 0 {
			hasDefault = true
		}
	}
	assert.True(t, hasDefault, "expected default calendar with id=0")
}

func TestCalendars_UpdateNameAndColor(t *testing.T) {
	ts := setupTestServer(t)

	resp := patchJSONMap(t, ts.URL+"/api/v1/calendars/0", map[string]any{
		"name":  "My Calendar",
		"color": "dodgerblue",
	})
	b := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "body: %s", b)
	var cal jsonCalendar
	err := json.Unmarshal(b, &cal)
	require.NoError(t, err)
	assert.Equal(t, "My Calendar", cal.Name)
	assert.Equal(t, "dodgerblue", cal.Color)
}

func TestCalendars_CreatedByImport(t *testing.T) {
	ts := setupTestServer(t)

	ics := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
DTSTART:20260601T100000Z
DTEND:20260601T110000Z
SUMMARY:Work Event
END:VEVENT
END:VCALENDAR`

	resp, err := http.Post(ts.URL+"/api/v1/import?calendar=Work", "text/calendar", strings.NewReader(ics))
	require.NoError(t, err)
	readBody(t, resp) // drain
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	calsResp, err := http.Get(ts.URL + "/api/v1/calendars")
	require.NoError(t, err)
	cals := decodeCalendars(t, readBody(t, calsResp))

	var found bool
	for _, c := range cals {
		if c.Name == "Work" {
			found = true
		}
	}
	assert.True(t, found, "expected a calendar named 'Work' after import, got %+v", cals)
}

// ---- Event tests with plain JSON ----

func TestCreateAndGetEvent(t *testing.T) {
	ts := setupTestServer(t)

	created := createJSONEvent(t, ts, map[string]any{
		"title":       "Team Meeting",
		"all_day":     false,
		"start_time":  "2026-05-10T14:00:00Z",
		"end_time":    "2026-05-10T15:00:00Z",
		"description": "Quarterly review",
	})

	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Team Meeting", created.Title)

	resp, err := http.Get(ts.URL + "/api/v1/events/" + created.ID)
	require.NoError(t, err)
	got := decodeEvent(t, readBody(t, resp))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "Team Meeting", got.Title)
}

func TestUpdateEvent(t *testing.T) {
	ts := setupTestServer(t)

	created := createJSONEvent(t, ts, map[string]any{
		"title":      "Stand-up",
		"all_day":    false,
		"start_time": "2026-05-12T09:00:00Z",
		"end_time":   "2026-05-12T09:30:00Z",
	})

	resp := patchJSONMap(t, ts.URL+"/api/v1/events/"+created.ID, map[string]any{
		"title":    "Daily Stand-up",
		"location": "Room 1",
	})
	b := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "body: %s", b)
	updated := decodeEvent(t, b)
	assert.Equal(t, "Daily Stand-up", updated.Title)
	assert.Equal(t, "Room 1", updated.Location)
	// start_time unchanged
	assert.Equal(t, "2026-05-12T09:00:00Z", updated.StartTime)
}

func TestDeleteEvent(t *testing.T) {
	ts := setupTestServer(t)

	created := createJSONEvent(t, ts, map[string]any{
		"title":      "Temporary",
		"all_day":    false,
		"start_time": "2026-05-13T10:00:00Z",
		"end_time":   "2026-05-13T11:00:00Z",
	})

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/events/"+created.ID, nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	resp2, _ := http.Get(ts.URL + "/api/v1/events/" + created.ID)
	resp2.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

func TestListEvents_TimeRange(t *testing.T) {
	ts := setupTestServer(t)

	for _, s := range []string{"2026-06-01T10:00:00Z", "2026-06-15T10:00:00Z", "2026-06-29T10:00:00Z"} {
		createJSONEvent(t, ts, map[string]any{
			"title":      "Event " + s,
			"all_day":    false,
			"start_time": s,
			"end_time":   strings.Replace(s, "T10", "T11", 1),
		})
	}

	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-06-14T00:00:00Z&to=2026-06-16T00:00:00Z")
	require.NoError(t, err)
	events := decodeEvents(t, readBody(t, resp))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, events, 1)
	assert.Equal(t, "Event 2026-06-15T10:00:00Z", events[0].Title)
}

func TestSearchEvents(t *testing.T) {
	ts := setupTestServer(t)

	createJSONEvent(t, ts, map[string]any{
		"title":      "Board Meeting",
		"all_day":    false,
		"start_time": "2026-07-01T09:00:00Z",
		"end_time":   "2026-07-01T10:00:00Z",
	})
	createJSONEvent(t, ts, map[string]any{
		"title":      "Doctor Appointment",
		"all_day":    false,
		"start_time": "2026-07-02T11:00:00Z",
		"end_time":   "2026-07-02T12:00:00Z",
	})

	resp, err := http.Get(ts.URL + "/api/v1/events?q=Board")
	require.NoError(t, err)
	events := decodeEvents(t, readBody(t, resp))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, events, 1)
	assert.Equal(t, "Board Meeting", events[0].Title)
}

func TestAllDayEvent(t *testing.T) {
	ts := setupTestServer(t)

	created := createJSONEvent(t, ts, map[string]any{
		"title":      "Birthday",
		"all_day":    true,
		"start_date": "2026-08-01",
	})

	assert.True(t, created.AllDay)
	assert.Equal(t, "2026-08-01", created.StartDate)
}

// ---- Location and reminder ----

func TestEventWithLocationAndCoordinates(t *testing.T) {
	ts := setupTestServer(t)

	lat, lon := 59.3293, 18.0686
	resp := postJSONMap(t, ts.URL+"/api/v1/events", map[string]any{
		"title":      "Conference in Stockholm",
		"all_day":    false,
		"start_time": "2026-09-01T08:00:00Z",
		"end_time":   "2026-09-01T18:00:00Z",
		"location":   "Stockholm, Sweden",
		"latitude":   lat,
		"longitude":  lon,
	})
	b := readBody(t, resp)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "body: %s", b)
	event := decodeEvent(t, b)
	assert.Equal(t, "Stockholm, Sweden", event.Location)
	require.NotNil(t, event.Latitude)
	assert.Equal(t, lat, *event.Latitude)
	require.NotNil(t, event.Longitude)
	assert.Equal(t, lon, *event.Longitude)
}

func TestEventWithReminderMinutes(t *testing.T) {
	ts := setupTestServer(t)

	mins := 30
	resp := postJSONMap(t, ts.URL+"/api/v1/events", map[string]any{
		"title":            "Flight",
		"all_day":          false,
		"start_time":       "2026-10-01T06:00:00Z",
		"end_time":         "2026-10-01T09:00:00Z",
		"reminder_minutes": mins,
	})
	b := readBody(t, resp)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "body: %s", b)
	event := decodeEvent(t, b)
	require.NotNil(t, event.ReminderMinutes)
	assert.Equal(t, mins, *event.ReminderMinutes)
}

// ---- Calendar filtering ----

func TestFilterByCalendarID(t *testing.T) {
	ts := setupTestServer(t)

	// Import into "Work" calendar to get a non-default calendar_id
	ics := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
DTSTART:20261001T100000Z
DTEND:20261001T110000Z
SUMMARY:Work Event
END:VEVENT
END:VCALENDAR`
	importResp, err := http.Post(ts.URL+"/api/v1/import?calendar=Work", "text/calendar", strings.NewReader(ics))
	require.NoError(t, err)
	readBody(t, importResp)

	// Also create a default-calendar event in the same range
	createJSONEvent(t, ts, map[string]any{
		"title":      "Personal Event",
		"all_day":    false,
		"start_time": "2026-10-01T12:00:00Z",
		"end_time":   "2026-10-01T13:00:00Z",
	})

	// Get the Work calendar's ID
	calsResp, err := http.Get(ts.URL + "/api/v1/calendars")
	require.NoError(t, err)
	cals := decodeCalendars(t, readBody(t, calsResp))
	var workCalID int64 = -1
	for _, c := range cals {
		if c.Name == "Work" {
			workCalID = c.ID
		}
	}
	require.GreaterOrEqual(t, workCalID, int64(0), "Work calendar not found")

	// List filtered to Work calendar
	url := fmt.Sprintf("%s/api/v1/events?from=2026-10-01T00:00:00Z&to=2026-10-02T00:00:00Z&calendar_id=%d", ts.URL, workCalID)
	resp, err := http.Get(url)
	require.NoError(t, err)
	events := decodeEvents(t, readBody(t, resp))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, events, 1, "only Work calendar")
	assert.Equal(t, "Work Event", events[0].Title)
}

func TestFilterByCalendarName(t *testing.T) {
	ts := setupTestServer(t)

	ics := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
DTSTART:20261101T100000Z
DTEND:20261101T110000Z
SUMMARY:Sports Event
END:VEVENT
END:VCALENDAR`
	importResp, err := http.Post(ts.URL+"/api/v1/import?calendar=Sports", "text/calendar", strings.NewReader(ics))
	require.NoError(t, err)
	readBody(t, importResp)

	createJSONEvent(t, ts, map[string]any{
		"title":      "Other Event",
		"all_day":    false,
		"start_time": "2026-11-01T12:00:00Z",
		"end_time":   "2026-11-01T13:00:00Z",
	})

	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-11-01T00:00:00Z&to=2026-11-02T00:00:00Z&calendar=Sports")
	require.NoError(t, err)
	events := decodeEvents(t, readBody(t, resp))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, events, 1, "only Sports calendar")
	assert.Equal(t, "Sports Event", events[0].Title)
}

// ---- iCal feed ----

func TestCalendarICSConvenienceURL(t *testing.T) {
	ts := setupTestServer(t)

	createJSONEvent(t, ts, map[string]any{
		"title":      "ICS Feed Test",
		"all_day":    false,
		"start_time": "2026-03-15T10:00:00Z",
		"end_time":   "2026-03-15T11:00:00Z",
	})

	resp, err := http.Get(ts.URL + "/calendar.ics")
	require.NoError(t, err)
	b := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/calendar"))
	assert.Contains(t, string(b), "SUMMARY:ICS Feed Test")
}

func TestICSFeedFilteredByCalendar(t *testing.T) {
	ts := setupTestServer(t)

	ics := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
DTSTART:20261201T100000Z
DTEND:20261201T110000Z
SUMMARY:Private Event
END:VEVENT
END:VCALENDAR`
	importResp, err := http.Post(ts.URL+"/api/v1/import?calendar=Private", "text/calendar", strings.NewReader(ics))
	require.NoError(t, err)
	readBody(t, importResp)

	createJSONEvent(t, ts, map[string]any{
		"title":      "Public Event",
		"all_day":    false,
		"start_time": "2026-12-02T10:00:00Z",
		"end_time":   "2026-12-02T11:00:00Z",
	})

	resp, err := http.Get(ts.URL + "/api/v1/events.ics?calendar=Private")
	require.NoError(t, err)
	b := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	icsStr := string(b)
	assert.Contains(t, icsStr, "SUMMARY:Private Event")
	assert.NotContains(t, icsStr, "SUMMARY:Public Event")
}

// ---- Error cases ----

func TestGetNonExistentEvent(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/events/99999")
	require.NoError(t, err)
	b := readBody(t, resp)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	var errBody jsonError
	err = json.Unmarshal(b, &errBody)
	require.NoError(t, err)
	assert.NotEmpty(t, errBody.Error)
}

func TestCreateEvent_MissingTitle(t *testing.T) {
	ts := setupTestServer(t)

	resp := postJSONMap(t, ts.URL+"/api/v1/events", map[string]any{
		"title":      "",
		"all_day":    false,
		"start_time": "2026-03-15T10:00:00Z",
		"end_time":   "2026-03-15T11:00:00Z",
	})
	b := readBody(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "body: %s", b)
	var errBody jsonError
	err := json.Unmarshal(b, &errBody)
	require.NoError(t, err)
	assert.NotEmpty(t, errBody.Error)
}

func TestListEvents_MissingRangeParams(t *testing.T) {
	ts := setupTestServer(t)

	for _, path := range []string{
		"/api/v1/events",
		"/api/v1/events?from=2026-03-01T00:00:00Z",
		"/api/v1/events?to=2026-03-31T00:00:00Z",
	} {
		resp, err := http.Get(ts.URL + path)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "GET %s", path)
	}
}

// ---- Recurring event tests ----

func TestRecurring_DailyExpansion(t *testing.T) {
	ts := setupTestServer(t)

	created := createJSONEvent(t, ts, map[string]any{
		"title":            "Daily Standup",
		"all_day":          false,
		"start_time":       "2026-05-04T09:00:00Z",
		"end_time":         "2026-05-04T09:30:00Z",
		"recurrence_freq":  "DAILY",
		"recurrence_count": 5,
	})

	assert.Equal(t, "DAILY", created.RecurrenceFreq)

	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-05-04T00:00:00Z&to=2026-05-09T00:00:00Z")
	require.NoError(t, err)
	events := decodeEvents(t, readBody(t, resp))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, events, 5, "5 daily instances")
	assert.Equal(t, "2026-05-04T09:00:00Z", events[0].StartTime)
	assert.Equal(t, "2026-05-08T09:00:00Z", events[4].StartTime)
	// All instances share the same parent ID
	for _, e := range events {
		assert.Equal(t, created.CalendarID, e.CalendarID)
	}
}

func TestRecurring_WeeklyExpansion(t *testing.T) {
	ts := setupTestServer(t)

	createJSONEvent(t, ts, map[string]any{
		"title":           "Weekly Review",
		"all_day":         false,
		"start_time":      "2026-05-04T10:00:00Z", // Monday
		"end_time":        "2026-05-04T11:00:00Z",
		"recurrence_freq": "WEEKLY",
	})

	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-05-04T00:00:00Z&to=2026-05-25T00:00:00Z")
	require.NoError(t, err)
	events := decodeEvents(t, readBody(t, resp))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, events, 3, "3 weekly instances")
	wantStarts := []string{"2026-05-04T10:00:00Z", "2026-05-11T10:00:00Z", "2026-05-18T10:00:00Z"}
	for i, want := range wantStarts {
		assert.Equal(t, want, events[i].StartTime, "instance %d start", i)
	}
}

func TestRecurring_CountLimitsExpansion(t *testing.T) {
	ts := setupTestServer(t)

	createJSONEvent(t, ts, map[string]any{
		"title":            "Limited Recurring",
		"all_day":          false,
		"start_time":       "2026-06-01T08:00:00Z",
		"end_time":         "2026-06-01T09:00:00Z",
		"recurrence_freq":  "DAILY",
		"recurrence_count": 3,
	})

	// Query a wide window — count=3 should cap at 3 regardless
	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-06-01T00:00:00Z&to=2026-12-31T00:00:00Z")
	require.NoError(t, err)
	events := decodeEvents(t, readBody(t, resp))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, events, 3, "count limit")
}

func TestRecurring_UntilLimitsExpansion(t *testing.T) {
	ts := setupTestServer(t)

	createJSONEvent(t, ts, map[string]any{
		"title":            "Until Recurring",
		"all_day":          false,
		"start_time":       "2026-07-01T08:00:00Z",
		"end_time":         "2026-07-01T09:00:00Z",
		"recurrence_freq":  "DAILY",
		"recurrence_until": "2026-07-03T23:59:59Z",
	})

	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-07-01T00:00:00Z&to=2026-12-31T00:00:00Z")
	require.NoError(t, err)
	events := decodeEvents(t, readBody(t, resp))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, events, 3, "Jul 1-3")
}

func TestRecurring_ExDateExcludesInstance(t *testing.T) {
	ts := setupTestServer(t)

	created := createJSONEvent(t, ts, map[string]any{
		"title":            "Daily with exception",
		"all_day":          false,
		"start_time":       "2026-08-03T10:00:00Z",
		"end_time":         "2026-08-03T11:00:00Z",
		"recurrence_freq":  "DAILY",
		"recurrence_count": 5,
	})

	// Delete the second instance (Aug 4) — the API adds it to exdates
	instanceID := fmt.Sprintf("%s_2026-08-04T10:00:00Z", created.ID)
	// Deleting a recurring instance adds it to exdates and returns 200 with the updated parent
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/events/"+instanceID, nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// List — should now have 4 instances (Aug 3, 5, 6, 7) with Aug 4 excluded
	listResp, err := http.Get(ts.URL + "/api/v1/events?from=2026-08-03T00:00:00Z&to=2026-08-08T00:00:00Z")
	require.NoError(t, err)
	events := decodeEvents(t, readBody(t, listResp))
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	assert.Len(t, events, 4, "after excluding one")
	for _, e := range events {
		assert.NotEqual(t, "2026-08-04T10:00:00Z", e.StartTime, "excluded instance (Aug 4) should not appear in listing")
	}
}

func TestRecurring_InstanceOverride(t *testing.T) {
	ts := setupTestServer(t)

	created := createJSONEvent(t, ts, map[string]any{
		"title":            "Weekly Sync",
		"all_day":          false,
		"start_time":       "2026-09-07T14:00:00Z", // Monday
		"end_time":         "2026-09-07T15:00:00Z",
		"recurrence_freq":  "WEEKLY",
		"recurrence_count": 4,
	})

	// Override the second instance (Sep 14) with a different title
	instanceID := fmt.Sprintf("%s_2026-09-14T14:00:00Z", created.ID)
	resp := patchJSONMap(t, ts.URL+"/api/v1/events/"+instanceID, map[string]any{
		"title": "Weekly Sync (rescheduled)",
	})
	b := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "body: %s", b)

	// Fetch the overridden instance directly
	getResp, err := http.Get(ts.URL + "/api/v1/events/" + instanceID)
	require.NoError(t, err)
	overridden := decodeEvent(t, readBody(t, getResp))
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Equal(t, "Weekly Sync (rescheduled)", overridden.Title)

	// In the list, the override should appear with the new title while others keep the original
	listResp, err := http.Get(ts.URL + "/api/v1/events?from=2026-09-07T00:00:00Z&to=2026-10-05T00:00:00Z")
	require.NoError(t, err)
	events := decodeEvents(t, readBody(t, listResp))
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	assert.Len(t, events, 4)
	var foundOverride bool
	for _, e := range events {
		if e.StartTime == "2026-09-14T14:00:00Z" {
			foundOverride = true
			assert.Equal(t, "Weekly Sync (rescheduled)", e.Title, "Sep 14 title")
		}
	}
	assert.True(t, foundOverride, "overridden instance not found in list")
}

func TestRecurring_MonthlyByDay(t *testing.T) {
	ts := setupTestServer(t)

	createJSONEvent(t, ts, map[string]any{
		"title":             "Monthly Board",
		"all_day":           false,
		"start_time":        "2026-05-11T09:00:00Z", // 2nd Monday of May
		"end_time":          "2026-05-11T10:00:00Z",
		"recurrence_freq":   "MONTHLY",
		"recurrence_by_day": "2MO",
		"recurrence_count":  3,
	})

	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-05-01T00:00:00Z&to=2026-08-01T00:00:00Z")
	require.NoError(t, err)
	events := decodeEvents(t, readBody(t, resp))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, events, 3, "3 monthly instances")
	// 2nd Monday: May 11, Jun 8, Jul 13
	wantStarts := []string{"2026-05-11T09:00:00Z", "2026-06-08T09:00:00Z", "2026-07-13T09:00:00Z"}
	for i, want := range wantStarts {
		assert.Equal(t, want, events[i].StartTime, "instance %d start", i)
	}
}

func TestRecurring_GetParentEvent(t *testing.T) {
	ts := setupTestServer(t)

	created := createJSONEvent(t, ts, map[string]any{
		"title":            "Recurring Event",
		"all_day":          false,
		"start_time":       "2026-10-01T08:00:00Z",
		"end_time":         "2026-10-01T09:00:00Z",
		"recurrence_freq":  "WEEKLY",
		"recurrence_count": 4,
	})

	// GET the parent by its plain numeric ID should return the event with recurrence info
	resp, err := http.Get(ts.URL + "/api/v1/events/" + created.ID)
	require.NoError(t, err)
	got := decodeEvent(t, readBody(t, resp))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "WEEKLY", got.RecurrenceFreq)
	require.NotNil(t, got.RecurrenceCount)
	assert.Equal(t, 4, *got.RecurrenceCount)
}
