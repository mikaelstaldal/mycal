package handler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mikaelstaldal/mycal/internal/handler"
	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/service"
	_ "modernc.org/sqlite"
)

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
	svc := service.NewEventService(repo)
	router := handler.NewRouter(svc)
	ts := httptest.NewServer(router)
	t.Cleanup(func() {
		ts.Close()
		db.Close()
	})
	return ts
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	return resp
}

func putJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	return resp
}

func doDelete(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	return resp
}

func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return v
}

func createTestEvent(t *testing.T, ts *httptest.Server) model.Event {
	t.Helper()
	body := model.CreateEventRequest{
		Title:     "Test Event",
		StartTime: "2026-03-15T10:00:00Z",
		EndTime:   "2026-03-15T11:00:00Z",
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create event: got status %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	return decodeJSON[model.Event](t, resp)
}

// --- CRUD tests ---

func TestCreateEvent(t *testing.T) {
	ts := setupTestServer(t)
	body := model.CreateEventRequest{
		Title:       "Meeting",
		Description: "Team sync",
		StartTime:   "2026-03-15T10:00:00Z",
		EndTime:     "2026-03-15T11:00:00Z",
		Location:    "Room 42",
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	event := decodeJSON[model.Event](t, resp)
	if event.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if event.Title != "Meeting" {
		t.Errorf("title = %q, want %q", event.Title, "Meeting")
	}
	if event.Description != "Team sync" {
		t.Errorf("description = %q, want %q", event.Description, "Team sync")
	}
	if event.Location != "Room 42" {
		t.Errorf("location = %q, want %q", event.Location, "Room 42")
	}
	if event.CreatedAt == "" {
		t.Error("expected non-empty created_at")
	}
	if event.UpdatedAt == "" {
		t.Error("expected non-empty updated_at")
	}
}

func TestCreateEvent_ValidationErrors(t *testing.T) {
	ts := setupTestServer(t)

	tests := []struct {
		name string
		body model.CreateEventRequest
	}{
		{
			name: "missing title",
			body: model.CreateEventRequest{
				StartTime: "2026-03-15T10:00:00Z",
				EndTime:   "2026-03-15T11:00:00Z",
			},
		},
		{
			name: "missing start_time",
			body: model.CreateEventRequest{
				Title:   "Event",
				EndTime: "2026-03-15T11:00:00Z",
			},
		},
		{
			name: "bad start_time format",
			body: model.CreateEventRequest{
				Title:     "Event",
				StartTime: "not-a-date",
				EndTime:   "2026-03-15T11:00:00Z",
			},
		},
		{
			name: "end before start",
			body: model.CreateEventRequest{
				Title:     "Event",
				StartTime: "2026-03-15T11:00:00Z",
				EndTime:   "2026-03-15T10:00:00Z",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := postJSON(t, ts.URL+"/api/v1/events", tc.body)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusBadRequest)
			}
		})
	}
}

func TestGetEvent(t *testing.T) {
	ts := setupTestServer(t)
	created := createTestEvent(t, ts)

	resp, err := http.Get(ts.URL + "/api/v1/events/" + itoa(created.ID))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
	event := decodeJSON[model.Event](t, resp)
	if event.ID != created.ID {
		t.Errorf("id = %d, want %d", event.ID, created.ID)
	}
	if event.Title != "Test Event" {
		t.Errorf("title = %q, want %q", event.Title, "Test Event")
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/events/99999")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestGetEvent_InvalidID(t *testing.T) {
	ts := setupTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/events/abc")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestUpdateEvent(t *testing.T) {
	ts := setupTestServer(t)
	created := createTestEvent(t, ts)

	newTitle := "Updated Title"
	resp := putJSON(t, ts.URL+"/api/v1/events/"+itoa(created.ID), model.UpdateEventRequest{
		Title: &newTitle,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
	updated := decodeJSON[model.Event](t, resp)
	if updated.Title != "Updated Title" {
		t.Errorf("title = %q, want %q", updated.Title, "Updated Title")
	}
	// Unchanged fields should be preserved
	if updated.StartTime != created.StartTime {
		t.Errorf("start_time = %q, want %q", updated.StartTime, created.StartTime)
	}
}

func TestUpdateEvent_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	title := "X"
	resp := putJSON(t, ts.URL+"/api/v1/events/99999", model.UpdateEventRequest{
		Title: &title,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestDeleteEvent(t *testing.T) {
	ts := setupTestServer(t)
	created := createTestEvent(t, ts)

	resp := doDelete(t, ts.URL+"/api/v1/events/"+itoa(created.ID))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: got status %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	// Confirm it's gone
	resp2, err := http.Get(ts.URL + "/api/v1/events/" + itoa(created.ID))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("after delete: got status %d, want %d", resp2.StatusCode, http.StatusNotFound)
	}
}

func TestDeleteEvent_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	resp := doDelete(t, ts.URL+"/api/v1/events/99999")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// --- List and search tests ---

func TestListEvents(t *testing.T) {
	ts := setupTestServer(t)

	// Create events at different times
	for _, start := range []string{"2026-03-10T10:00:00Z", "2026-03-15T10:00:00Z", "2026-03-20T10:00:00Z"} {
		postJSON(t, ts.URL+"/api/v1/events", model.CreateEventRequest{
			Title:     "Event at " + start,
			StartTime: start,
			EndTime:   start[:11] + "11:00:00Z",
		}).Body.Close()
	}

	// Query a range that includes only the middle event
	resp, err := http.Get(ts.URL + "/api/v1/events?from=2026-03-14T00:00:00Z&to=2026-03-16T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
	events := decodeJSON[[]model.Event](t, resp)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Title != "Event at 2026-03-15T10:00:00Z" {
		t.Errorf("title = %q", events[0].Title)
	}
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
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("GET %s: got status %d, want %d", path, resp.StatusCode, http.StatusBadRequest)
		}
	}
}

func TestSearchEvents(t *testing.T) {
	ts := setupTestServer(t)

	postJSON(t, ts.URL+"/api/v1/events", model.CreateEventRequest{
		Title:     "Go Conference",
		StartTime: "2026-03-15T10:00:00Z",
		EndTime:   "2026-03-15T18:00:00Z",
	}).Body.Close()
	postJSON(t, ts.URL+"/api/v1/events", model.CreateEventRequest{
		Title:     "Lunch Break",
		StartTime: "2026-03-15T12:00:00Z",
		EndTime:   "2026-03-15T13:00:00Z",
	}).Body.Close()

	resp, err := http.Get(ts.URL + "/api/v1/events?q=Conference")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
	events := decodeJSON[[]model.Event](t, resp)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Title != "Go Conference" {
		t.Errorf("title = %q, want %q", events[0].Title, "Go Conference")
	}
}

// --- All-day events ---

func TestCreateAllDayEvent(t *testing.T) {
	ts := setupTestServer(t)
	body := model.CreateEventRequest{
		Title:     "Holiday",
		StartTime: "2026-06-15",
		AllDay:    true,
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	event := decodeJSON[model.Event](t, resp)
	if !event.AllDay {
		t.Error("expected all_day to be true")
	}
	if event.Title != "Holiday" {
		t.Errorf("title = %q, want %q", event.Title, "Holiday")
	}
}

// --- Recurring events ---

func TestRecurringEventExpansion(t *testing.T) {
	ts := setupTestServer(t)

	// Create a weekly recurring event
	body := model.CreateEventRequest{
		Title:          "Weekly Standup",
		StartTime:      "2026-03-02T09:00:00Z",
		EndTime:        "2026-03-02T09:30:00Z",
		RecurrenceFreq: "WEEKLY",
		RecurrenceCount: 10,
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: got status %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	resp.Body.Close()

	// List over a 3-week range — should get 3 instances
	listResp, err := http.Get(ts.URL + "/api/v1/events?from=2026-03-01T00:00:00Z&to=2026-03-22T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list: got status %d, want %d", listResp.StatusCode, http.StatusOK)
	}
	events := decodeJSON[[]model.Event](t, listResp)
	if len(events) < 3 {
		t.Errorf("got %d events, want at least 3 recurring instances", len(events))
	}
}

// --- Delete with EXDATE ---

func TestDeleteWithInstanceStart(t *testing.T) {
	ts := setupTestServer(t)

	body := model.CreateEventRequest{
		Title:          "Daily Standup",
		StartTime:      "2026-03-01T09:00:00Z",
		EndTime:        "2026-03-01T09:30:00Z",
		RecurrenceFreq: "DAILY",
		RecurrenceCount: 30,
	}
	resp := postJSON(t, ts.URL+"/api/v1/events", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: got status %d", resp.StatusCode)
	}
	created := decodeJSON[model.Event](t, resp)

	// Delete a single instance via instance_start query param
	delResp := doDelete(t, ts.URL+"/api/v1/events/"+itoa(created.ID)+"?instance_start=2026-03-05T09:00:00Z")
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("delete instance: got status %d, want %d", delResp.StatusCode, http.StatusOK)
	}
	updated := decodeJSON[model.Event](t, delResp)
	if !strings.Contains(updated.ExDates, "2026-03-05T09:00:00Z") {
		t.Errorf("exdates = %q, want it to contain the excluded date", updated.ExDates)
	}
}

// --- iCal export ---

func TestExportICal(t *testing.T) {
	ts := setupTestServer(t)
	createTestEvent(t, ts)

	resp, err := http.Get(ts.URL + "/api/v1/events.ics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/calendar") {
		t.Errorf("content-type = %q, want text/calendar", ct)
	}
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

	resp := postJSON(t, ts.URL+"/api/v1/import", map[string]string{"ics_content": ics})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
	result := decodeJSON[map[string]int](t, resp)
	if result["imported"] != 2 {
		t.Errorf("imported = %d, want 2", result["imported"])
	}
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

	resp := postJSON(t, ts.URL+"/api/v1/import-single", map[string]string{"ics_content": ics})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	event := decodeJSON[model.Event](t, resp)
	if event.Title != "Single Import" {
		t.Errorf("title = %q, want %q", event.Title, "Single Import")
	}
	if event.Location != "Office" {
		t.Errorf("location = %q, want %q", event.Location, "Office")
	}
}

func itoa(n int64) string {
	return fmt.Sprintf("%d", n)
}
