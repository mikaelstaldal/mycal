package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mikaelstaldal/mycal/internal/ical"
	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/service"
)

func registerEventRoutes(mux *http.ServeMux, svc *service.EventService) {
	mux.HandleFunc("GET /api/v1/events", listEvents(svc))
	mux.HandleFunc("GET /api/v1/events.ics", exportICalFeed(svc))
	mux.HandleFunc("POST /api/v1/events", createEvent(svc))
	mux.HandleFunc("GET /api/v1/events/{id}", getEvent(svc))
	mux.HandleFunc("PUT /api/v1/events/{id}", updateEvent(svc))
	mux.HandleFunc("DELETE /api/v1/events/{id}", deleteEvent(svc))
	mux.HandleFunc("POST /api/v1/import", importEvents(svc))
	mux.HandleFunc("POST /api/v1/import-single", importSingleEvent(svc))
}

const maxSearchQueryLength = 200

func listEvents(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")

		if len(q) > maxSearchQueryLength {
			writeError(w, http.StatusBadRequest, "search query too long")
			return
		}

		if q != "" {
			events, err := svc.Search(q, from, to)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to search events")
				return
			}
			writeJSON(w, http.StatusOK, events)
			return
		}

		if from == "" || to == "" {
			writeError(w, http.StatusBadRequest, "from and to query parameters are required")
			return
		}
		events, err := svc.List(from, to)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list events")
			return
		}
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
		writeJSON(w, http.StatusCreated, event)
	}
}

func getEvent(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		event, err := svc.GetByID(id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeError(w, http.StatusNotFound, "event not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to get event")
			return
		}
		writeJSON(w, http.StatusOK, event)
	}
}

func updateEvent(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var req model.UpdateEventRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		// Check for instance_start query param (override a single instance)
		instanceStart := r.URL.Query().Get("instance_start")
		if instanceStart != "" {
			event, err := svc.CreateOrUpdateOverride(id, instanceStart, &req)
			if err != nil {
				if errors.Is(err, service.ErrNotFound) {
					writeError(w, http.StatusNotFound, "event not found")
					return
				}
				if errors.Is(err, service.ErrValidation) {
					writeError(w, http.StatusBadRequest, err.Error())
					return
				}
				writeError(w, http.StatusInternalServerError, "failed to update instance")
				return
			}
			writeJSON(w, http.StatusOK, event)
			return
		}

		event, err := svc.Update(id, &req)
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
		writeJSON(w, http.StatusOK, event)
	}
}

func deleteEvent(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		// If instance_start is provided, add EXDATE instead of deleting
		instanceStart := r.URL.Query().Get("instance_start")
		if instanceStart != "" {
			event, err := svc.AddExDate(id, instanceStart)
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
			writeJSON(w, http.StatusOK, event)
			return
		}

		if err := svc.Delete(id); err != nil {
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
func NewCalendarFeedHandler(svc *service.EventService) http.Handler {
	return withMiddleware(exportICalFeed(svc))
}

const maxImportSize = 5 * 1024 * 1024 // 5MB

// validateExternalURL checks that the URL is safe to fetch (not localhost or private IPs).
func validateExternalURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https")
	}
	hostname := u.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}
	// Block localhost names
	lower := strings.ToLower(hostname)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") {
		return fmt.Errorf("URL must not point to localhost")
	}

	// Resolve hostname through DNS with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %s: %v", hostname, err)
	}

	// Check if any resolved IP is private
	for _, ip := range ips {
		if ip.IP.IsLoopback() || ip.IP.IsPrivate() || ip.IP.IsLinkLocalUnicast() || ip.IP.IsLinkLocalMulticast() || ip.IP.IsUnspecified() {
			return fmt.Errorf("URL must not point to a private or local address")
		}
	}
	return nil
}

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
		if err := validateExternalURL(req.URL); err != nil {
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

		events, err := ical.Decode(reader)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to parse iCal data")
			return
		}

		event, err := svc.ImportSingle(events)
		if err != nil {
			if errors.Is(err, service.ErrValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to import event")
			return
		}

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

		events, err := ical.Decode(reader)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to parse iCal data")
			return
		}

		imported, err := svc.Import(events)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to import events")
			return
		}

		writeJSON(w, http.StatusOK, map[string]int{"imported": imported})
	}
}

func exportICalFeed(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events, err := svc.ListAll()
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
