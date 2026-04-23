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
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo, err := repository.NewSQLiteRepository(db)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
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
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return b
}

func postJSONMap(t *testing.T, url string, body map[string]any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := http.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	return resp
}

func patchJSONMap(t *testing.T, rawURL string, body map[string]any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPatch, rawURL, strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	return resp
}

func decodeEvent(t *testing.T, b []byte) jsonEvent {
	t.Helper()
	var e jsonEvent
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatalf("decode event: %v\nbody: %s", err, b)
	}
	return e
}

func decodeEvents(t *testing.T, b []byte) []jsonEvent {
	t.Helper()
	var events []jsonEvent
	if err := json.Unmarshal(b, &events); err != nil {
		t.Fatalf("decode events: %v\nbody: %s", err, b)
	}
	return events
}

func decodeCalendars(t *testing.T, b []byte) []jsonCalendar {
	t.Helper()
	var cals []jsonCalendar
	if err := json.Unmarshal(b, &cals); err != nil {
		t.Fatalf("decode calendars: %v\nbody: %s", err, b)
	}
	return cals
}

// createJSONEvent posts a timed event and returns the decoded body.
func createJSONEvent(t *testing.T, ts *httptest.Server, body map[string]any) jsonEvent {
	t.Helper()
	resp := postJSONMap(t, ts.URL+"/api/v1/events", body)
	b := readBody(t, resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create event: got %d, want 201\nbody: %s", resp.StatusCode, b)
	}
	return decodeEvent(t, b)
}

// ---- Calendar tests ----

func TestCalendars_DefaultCalendarExists(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/calendars")
	if err != nil {
		t.Fatal(err)
	}
	cals := decodeCalendars(t, readBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	if len(cals) == 0 {
		t.Fatal("expected at least one calendar (the default)")
	}
	var hasDefault bool
	for _, c := range cals {
		if c.ID == 0 {
			hasDefault = true
		}
	}
	if !hasDefault {
		t.Error("expected default calendar with id=0")
	}
}

func TestCalendars_UpdateNameAndColor(t *testing.T) {
	ts := setupTestServer(t)

	resp := patchJSONMap(t, ts.URL+"/api/v1/calendars/0", map[string]any{
		"name":  "My Calendar",
		"color": "dodgerblue",
	})
	b := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update calendar: got %d, want 200\nbody: %s", resp.StatusCode, b)
	}
	var cal jsonCalendar
	if err := json.Unmarshal(b, &cal); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cal.Name != "My Calendar" {
		t.Errorf("name = %q, want %q", cal.Name, "My Calendar")
	}
	if cal.Color != "dodgerblue" {
		t.Errorf("color = %q, want %q", cal.Color, "dodgerblue")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	readBody(t, resp) // drain
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("import: got %d, want 200", resp.StatusCode)
	}

	calsResp, err := http.Get(ts.URL + "/api/v1/calendars")
	if err != nil {
		t.Fatal(err)
	}
	cals := decodeCalendars(t, readBody(t, calsResp))

	var found bool
	for _, c := range cals {
		if c.Name == "Work" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a calendar named 'Work' after import, got %+v", cals)
	}
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

	if created.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if created.Title != "Team Meeting" {
		t.Errorf("title = %q, want %q", created.Title, "Team Meeting")
	}

	resp, err := http.Get(ts.URL + "/api/v1/events/" + created.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := decodeEvent(t, readBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get: got %d, want 200", resp.StatusCode)
	}
	if got.Title != "Team Meeting" {
		t.Errorf("title = %q, want %q", got.Title, "Team Meeting")
	}
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
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update: got %d, want 200\nbody: %s", resp.StatusCode, b)
	}
	updated := decodeEvent(t, b)
	if updated.Title != "Daily Stand-up" {
		t.Errorf("title = %q, want %q", updated.Title, "Daily Stand-up")
	}
	if updated.Location != "Room 1" {
		t.Errorf("location = %q, want %q", updated.Location, "Room 1")
	}
	// start_time unchanged
	if updated.StartTime != "2026-05-12T09:00:00Z" {
		t.Errorf("start_time = %q, want unchanged", updated.StartTime)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: got %d, want 204", resp.StatusCode)
	}

	resp2, _ := http.Get(ts.URL + "/api/v1/events/" + created.ID)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("after delete: got %d, want 404", resp2.StatusCode)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	events := decodeEvents(t, readBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: got %d, want 200", resp.StatusCode)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Title != "Event 2026-06-15T10:00:00Z" {
		t.Errorf("wrong event returned: %q", events[0].Title)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	events := decodeEvents(t, readBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search: got %d, want 200", resp.StatusCode)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Title != "Board Meeting" {
		t.Errorf("title = %q, want %q", events[0].Title, "Board Meeting")
	}
}

func TestAllDayEvent(t *testing.T) {
	ts := setupTestServer(t)

	created := createJSONEvent(t, ts, map[string]any{
		"title":      "Birthday",
		"all_day":    true,
		"start_date": "2026-08-01",
	})

	if !created.AllDay {
		t.Error("expected all_day = true")
	}
	if created.StartDate != "2026-08-01" {
		t.Errorf("start_date = %q, want %q", created.StartDate, "2026-08-01")
	}
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
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: got %d, want 201\nbody: %s", resp.StatusCode, b)
	}
	event := decodeEvent(t, b)
	if event.Location != "Stockholm, Sweden" {
		t.Errorf("location = %q, want %q", event.Location, "Stockholm, Sweden")
	}
	if event.Latitude == nil || *event.Latitude != lat {
		t.Errorf("latitude = %v, want %v", event.Latitude, lat)
	}
	if event.Longitude == nil || *event.Longitude != lon {
		t.Errorf("longitude = %v, want %v", event.Longitude, lon)
	}
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
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: got %d, want 201\nbody: %s", resp.StatusCode, b)
	}
	event := decodeEvent(t, b)
	if event.ReminderMinutes == nil || *event.ReminderMinutes != mins {
		t.Errorf("reminder_minutes = %v, want %d", event.ReminderMinutes, mins)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	readBody(t, importResp)

	// Also create a default-calendar event in the same range
	createJSONEvent(t, ts, map[string]any{
		"title":      "Personal Event",
		"all_day":    false,
		"start_time": "2026-10-01T12:00:00Z",
		"end_time":   "2026-10-01T13:00:00Z",
	})

	// Get the Work calendar's ID
	calsResp, _ := http.Get(ts.URL + "/api/v1/calendars")
	cals := decodeCalendars(t, readBody(t, calsResp))
	var workCalID int64 = -1
	for _, c := range cals {
		if c.Name == "Work" {
			workCalID = c.ID
		}
	}
	if workCalID < 0 {
		t.Fatal("Work calendar not found")
	}

	// List filtered to Work calendar
	url := fmt.Sprintf("%s/api/v1/events?from=2026-10-01T00:00:00Z&to=2026-10-02T00:00:00Z&calendar_id=%d", ts.URL, workCalID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	events := decodeEvents(t, readBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: got %d, want 200", resp.StatusCode)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (only Work calendar)", len(events))
	}
	if events[0].Title != "Work Event" {
		t.Errorf("title = %q, want %q", events[0].Title, "Work Event")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	readBody(t, importResp)

	createJSONEvent(t, ts, map[string]any{
		"title":      "Other Event",
		"all_day":    false,
		"start_time": "2026-11-01T12:00:00Z",
		"end_time":   "2026-11-01T13:00:00Z",
	})

	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-11-01T00:00:00Z&to=2026-11-02T00:00:00Z&calendar=Sports")
	if err != nil {
		t.Fatal(err)
	}
	events := decodeEvents(t, readBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: got %d, want 200", resp.StatusCode)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (only Sports calendar)", len(events))
	}
	if events[0].Title != "Sports Event" {
		t.Errorf("title = %q, want %q", events[0].Title, "Sports Event")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	b := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/calendar") {
		t.Errorf("Content-Type = %q, want text/calendar", ct)
	}
	if !strings.Contains(string(b), "SUMMARY:ICS Feed Test") {
		t.Error("expected SUMMARY:ICS Feed Test in calendar.ics output")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	readBody(t, importResp)

	createJSONEvent(t, ts, map[string]any{
		"title":      "Public Event",
		"all_day":    false,
		"start_time": "2026-12-02T10:00:00Z",
		"end_time":   "2026-12-02T11:00:00Z",
	})

	resp, err := http.Get(ts.URL + "/api/v1/events.ics?calendar=Private")
	if err != nil {
		t.Fatal(err)
	}
	b := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	icsStr := string(b)
	if !strings.Contains(icsStr, "SUMMARY:Private Event") {
		t.Error("expected Private Event in filtered ics feed")
	}
	if strings.Contains(icsStr, "SUMMARY:Public Event") {
		t.Error("Public Event should not appear in Private calendar feed")
	}
}

// ---- Error cases ----

func TestGetNonExistentEvent(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/events/99999")
	if err != nil {
		t.Fatal(err)
	}
	b := readBody(t, resp)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("got %d, want 404", resp.StatusCode)
	}
	var errBody jsonError
	if err := json.Unmarshal(b, &errBody); err != nil {
		t.Fatalf("error response should be valid JSON: %v", err)
	}
	if errBody.Error == "" {
		t.Error("expected non-empty error field in response")
	}
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
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400\nbody: %s", resp.StatusCode, b)
	}
	var errBody jsonError
	if err := json.Unmarshal(b, &errBody); err != nil {
		t.Fatalf("error response should be valid JSON: %v", err)
	}
	if errBody.Error == "" {
		t.Error("expected non-empty error field")
	}
}

func TestListEvents_MissingRangeParams(t *testing.T) {
	ts := setupTestServer(t)

	for _, path := range []string{
		"/api/v1/events",
		"/api/v1/events?from=2026-03-01T00:00:00Z",
		"/api/v1/events?to=2026-03-31T00:00:00Z",
	} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("GET %s: got %d, want 400", path, resp.StatusCode)
		}
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

	if created.RecurrenceFreq != "DAILY" {
		t.Errorf("recurrence_freq = %q, want DAILY", created.RecurrenceFreq)
	}

	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-05-04T00:00:00Z&to=2026-05-09T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	events := decodeEvents(t, readBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: got %d, want 200", resp.StatusCode)
	}
	if len(events) != 5 {
		t.Fatalf("got %d events, want 5 daily instances", len(events))
	}
	if events[0].StartTime != "2026-05-04T09:00:00Z" {
		t.Errorf("first instance start = %q, want 2026-05-04T09:00:00Z", events[0].StartTime)
	}
	if events[4].StartTime != "2026-05-08T09:00:00Z" {
		t.Errorf("last instance start = %q, want 2026-05-08T09:00:00Z", events[4].StartTime)
	}
	// All instances share the same parent ID
	for _, e := range events {
		if e.CalendarID != created.CalendarID {
			t.Errorf("instance calendar_id = %d, want %d", e.CalendarID, created.CalendarID)
		}
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
	if err != nil {
		t.Fatal(err)
	}
	events := decodeEvents(t, readBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: got %d, want 200", resp.StatusCode)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3 weekly instances", len(events))
	}
	wantStarts := []string{"2026-05-04T10:00:00Z", "2026-05-11T10:00:00Z", "2026-05-18T10:00:00Z"}
	for i, want := range wantStarts {
		if events[i].StartTime != want {
			t.Errorf("instance %d start = %q, want %q", i, events[i].StartTime, want)
		}
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
	if err != nil {
		t.Fatal(err)
	}
	events := decodeEvents(t, readBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: got %d, want 200", resp.StatusCode)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want exactly 3 (count limit)", len(events))
	}
}

func TestRecurring_UntilLimitsExpansion(t *testing.T) {
	ts := setupTestServer(t)

	createJSONEvent(t, ts, map[string]any{
		"title":              "Until Recurring",
		"all_day":            false,
		"start_time":         "2026-07-01T08:00:00Z",
		"end_time":           "2026-07-01T09:00:00Z",
		"recurrence_freq":    "DAILY",
		"recurrence_until":   "2026-07-03T23:59:59Z",
	})

	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-07-01T00:00:00Z&to=2026-12-31T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	events := decodeEvents(t, readBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: got %d, want 200", resp.StatusCode)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events (Jul 1-3), want 3", len(events))
	}
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
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete instance: got %d, want 200", resp.StatusCode)
	}

	// List — should now have 4 instances (Aug 3, 5, 6, 7) with Aug 4 excluded
	listResp, err := http.Get(ts.URL + "/api/v1/events?from=2026-08-03T00:00:00Z&to=2026-08-08T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	events := decodeEvents(t, readBody(t, listResp))
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list: got %d, want 200", listResp.StatusCode)
	}
	if len(events) != 4 {
		t.Fatalf("got %d events after excluding one, want 4", len(events))
	}
	for _, e := range events {
		if e.StartTime == "2026-08-04T10:00:00Z" {
			t.Error("excluded instance (Aug 4) should not appear in listing")
		}
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
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch instance: got %d, want 200\nbody: %s", resp.StatusCode, b)
	}

	// Fetch the overridden instance directly
	getResp, err := http.Get(ts.URL + "/api/v1/events/" + instanceID)
	if err != nil {
		t.Fatal(err)
	}
	overridden := decodeEvent(t, readBody(t, getResp))
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get instance: got %d, want 200", getResp.StatusCode)
	}
	if overridden.Title != "Weekly Sync (rescheduled)" {
		t.Errorf("overridden title = %q, want %q", overridden.Title, "Weekly Sync (rescheduled)")
	}

	// In the list, the override should appear with the new title while others keep the original
	listResp, err := http.Get(ts.URL + "/api/v1/events?from=2026-09-07T00:00:00Z&to=2026-10-05T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	events := decodeEvents(t, readBody(t, listResp))
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list: got %d, want 200", listResp.StatusCode)
	}
	if len(events) != 4 {
		t.Fatalf("got %d events, want 4", len(events))
	}
	var foundOverride bool
	for _, e := range events {
		if e.StartTime == "2026-09-14T14:00:00Z" {
			foundOverride = true
			if e.Title != "Weekly Sync (rescheduled)" {
				t.Errorf("Sep 14 title = %q, want %q", e.Title, "Weekly Sync (rescheduled)")
			}
		}
	}
	if !foundOverride {
		t.Error("overridden instance not found in list")
	}
}

func TestRecurring_MonthlyByDay(t *testing.T) {
	ts := setupTestServer(t)

	createJSONEvent(t, ts, map[string]any{
		"title":                 "Monthly Board",
		"all_day":               false,
		"start_time":            "2026-05-11T09:00:00Z", // 2nd Monday of May
		"end_time":              "2026-05-11T10:00:00Z",
		"recurrence_freq":       "MONTHLY",
		"recurrence_by_day":     "2MO",
		"recurrence_count":      3,
	})

	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-05-01T00:00:00Z&to=2026-08-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	events := decodeEvents(t, readBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: got %d, want 200", resp.StatusCode)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3 monthly instances", len(events))
	}
	// 2nd Monday: May 11, Jun 8, Jul 13
	wantStarts := []string{"2026-05-11T09:00:00Z", "2026-06-08T09:00:00Z", "2026-07-13T09:00:00Z"}
	for i, want := range wantStarts {
		if events[i].StartTime != want {
			t.Errorf("instance %d start = %q, want %q", i, events[i].StartTime, want)
		}
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
	if err != nil {
		t.Fatal(err)
	}
	got := decodeEvent(t, readBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get: got %d, want 200", resp.StatusCode)
	}
	if got.RecurrenceFreq != "WEEKLY" {
		t.Errorf("recurrence_freq = %q, want WEEKLY", got.RecurrenceFreq)
	}
	if got.RecurrenceCount == nil || *got.RecurrenceCount != 4 {
		t.Errorf("recurrence_count = %v, want 4", got.RecurrenceCount)
	}
}
