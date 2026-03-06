package handler

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mikaelstaldal/mycal/internal/ical"
	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/service"
)

func registerEventRoutes(mux *http.ServeMux, svc *service.EventService, calSvc *service.CalendarService) {
	mux.HandleFunc("GET /api/v1/events", listEvents(svc, calSvc))
	mux.HandleFunc("GET /api/v1/events.ics", exportICalFeed(svc, calSvc))
	mux.HandleFunc("POST /api/v1/events", createEvent(svc))
	mux.HandleFunc("GET /api/v1/events/{id...}", getEvent(svc))
	mux.HandleFunc("PATCH /api/v1/events/{id...}", updateEvent(svc))
	mux.HandleFunc("DELETE /api/v1/events/{id...}", deleteEvent(svc))
	mux.HandleFunc("POST /api/v1/import", importEvents(svc))
	mux.HandleFunc("POST /api/v1/import-single", importSingleEvent(svc))
}

func setStringID(e *model.Event) {
	e.SetStringID()
}

func setStringIDs(events []model.Event) {
	for i := range events {
		events[i].SetStringID()
	}
}

const maxSearchQueryLength = 200

func parseCalendarIDs(r *http.Request, calSvc *service.CalendarService) []int64 {
	// Support calendar_id query param (new)
	idStrs := r.URL.Query()["calendar_id"]
	// Support calendar name param (backward compat)
	nameStrs := r.URL.Query()["calendar"]

	if idStrs == nil && nameStrs == nil {
		return nil // nil = all calendars
	}

	var ids []int64
	for _, s := range idStrs {
		id, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	// Resolve names to IDs
	if calSvc != nil {
		for _, name := range nameStrs {
			calID, err := calSvc.GetOrCreateByName(name)
			if err == nil {
				ids = append(ids, calID)
			}
		}
	}
	if ids == nil {
		ids = []int64{} // empty = match nothing
	}
	return ids
}

func listEvents(svc *service.EventService, calSvc *service.CalendarService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		calendarIDs := parseCalendarIDs(r, calSvc)

		if len(q) > maxSearchQueryLength {
			writeError(w, http.StatusBadRequest, "search query too long")
			return
		}

		if q != "" {
			events, err := svc.Search(q, from, to, calendarIDs)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to search events")
				return
			}
			setStringIDs(events)
			writeJSON(w, http.StatusOK, events)
			return
		}

		if from == "" || to == "" {
			writeError(w, http.StatusBadRequest, "from and to query parameters are required")
			return
		}
		events, err := svc.List(from, to, calendarIDs)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list events")
			return
		}
		setStringIDs(events)
		writeJSON(w, http.StatusOK, events)
	}
}

func createEvent(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.CreateEventRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		event, err := svc.Create(&req)
		if err != nil {
			if errors.Is(err, service.ErrValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to create event")
			return
		}
		setStringID(event)
		writeJSON(w, http.StatusCreated, event)
	}
}

func getEvent(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawID, err := url.PathUnescape(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		dbID, instanceStart, err := model.ParseEventID(rawID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var event *model.Event
		if instanceStart != "" {
			event, err = svc.GetInstance(dbID, instanceStart)
		} else {
			event, err = svc.GetByID(dbID)
		}
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeError(w, http.StatusNotFound, "event not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to get event")
			return
		}
		setStringID(event)
		writeJSON(w, http.StatusOK, event)
	}
}

func updateEvent(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawID, err := url.PathUnescape(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		dbID, instanceStart, err := model.ParseEventID(rawID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var req model.UpdateEventRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		var event *model.Event
		if instanceStart != "" {
			event, err = svc.CreateOrUpdateOverride(dbID, instanceStart, &req)
		} else {
			event, err = svc.Update(dbID, &req)
		}
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeError(w, http.StatusNotFound, "event not found")
				return
			}
			if errors.Is(err, service.ErrValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update event")
			return
		}
		setStringID(event)
		writeJSON(w, http.StatusOK, event)
	}
}

func deleteEvent(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawID, err := url.PathUnescape(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		dbID, instanceStart, err := model.ParseEventID(rawID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		if instanceStart != "" {
			event, err := svc.AddExDate(dbID, instanceStart)
			if err != nil {
				if errors.Is(err, service.ErrNotFound) {
					writeError(w, http.StatusNotFound, "event not found")
					return
				}
				if errors.Is(err, service.ErrValidation) {
					writeError(w, http.StatusBadRequest, err.Error())
					return
				}
				writeError(w, http.StatusInternalServerError, "failed to add exception date")
				return
			}
			setStringID(event)
			writeJSON(w, http.StatusOK, event)
			return
		}

		if err := svc.Delete(dbID); err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeError(w, http.StatusNotFound, "event not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to delete event")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// NewCalendarFeedHandler returns a handler for the /calendar.ics convenience URL.
func NewCalendarFeedHandler(svc *service.EventService, calSvc *service.CalendarService) http.Handler {
	return withMiddleware(exportICalFeed(svc, calSvc))
}

const maxImportSize = 10 * 1024 * 1024 // 10 MiB

// importReader returns an io.Reader for iCalendar data based on the request's Content-Type.
// For text/calendar, the request body is used directly.
// For application/json, a URL is expected in the JSON body and fetched.
// The caller must call the returned cleanup function when done.
func importReader(w http.ResponseWriter, r *http.Request) (io.Reader, func(), bool) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "text/calendar") {
		return io.LimitReader(r.Body, maxImportSize), func() {}, true
	}
	if strings.HasPrefix(ct, "application/json") {
		var req struct {
			URL string `json:"url"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return nil, nil, false
		}
		if req.URL == "" {
			writeError(w, http.StatusBadRequest, "url is required")
			return nil, nil, false
		}
		if err := service.ValidateExternalURL(req.URL); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return nil, nil, false
		}
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(req.URL)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to fetch URL")
			return nil, nil, false
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			writeError(w, http.StatusBadRequest, "URL returned non-200 status")
			return nil, nil, false
		}
		return io.LimitReader(resp.Body, maxImportSize), func() { resp.Body.Close() }, true
	}
	writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be text/calendar or application/json")
	return nil, nil, false
}

func importSingleEvent(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reader, cleanup, ok := importReader(w, r)
		if !ok {
			return
		}
		defer cleanup()

		calendarName := r.URL.Query().Get("calendar")

		events, err := ical.Decode(reader)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to parse iCalendar data")
			return
		}

		event, err := svc.ImportSingle(events, calendarName)
		if err != nil {
			if errors.Is(err, service.ErrValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to import event")
			return
		}

		setStringID(event)
		writeJSON(w, http.StatusCreated, event)
	}
}

func importEvents(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reader, cleanup, ok := importReader(w, r)
		if !ok {
			return
		}
		defer cleanup()

		calendarName := r.URL.Query().Get("calendar")

		events, err := ical.Decode(reader)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to parse iCalendar data")
			return
		}

		imported, err := svc.Import(events, calendarName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to import events")
			return
		}

		writeJSON(w, http.StatusOK, map[string]int{"imported": imported})
	}
}

func exportICalFeed(svc *service.EventService, calSvc *service.CalendarService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		calendarIDs := parseCalendarIDs(r, calSvc)
		events, err := svc.ListAll(calendarIDs)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list events")
			return
		}
		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"mycal.ics\"")
		if err := ical.Encode(w, events); err != nil {
			log.Printf("error writing ical feed: %v", err)
		}
	}
}
