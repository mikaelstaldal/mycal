package handler

import (
	"errors"
	"io"
	"log"
	"net/http"
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

func listEvents(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")

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

func importSingleEvent(svc *service.EventService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ICSContent string `json:"ics_content"`
			URL        string `json:"url"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		var icsReader io.Reader
		if req.ICSContent != "" {
			if len(req.ICSContent) > maxImportSize {
				writeError(w, http.StatusBadRequest, "content too large (max 5MB)")
				return
			}
			icsReader = strings.NewReader(req.ICSContent)
		} else if req.URL != "" {
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Get(req.URL)
			if err != nil {
				writeError(w, http.StatusBadRequest, "failed to fetch URL")
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				writeError(w, http.StatusBadRequest, "URL returned non-200 status")
				return
			}
			icsReader = io.LimitReader(resp.Body, maxImportSize)
		} else {
			writeError(w, http.StatusBadRequest, "ics_content or url is required")
			return
		}

		events, err := ical.Decode(icsReader)
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
		var req struct {
			ICSContent string `json:"ics_content"`
			URL        string `json:"url"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		var icsReader io.Reader
		if req.ICSContent != "" {
			if len(req.ICSContent) > maxImportSize {
				writeError(w, http.StatusBadRequest, "content too large (max 5MB)")
				return
			}
			icsReader = strings.NewReader(req.ICSContent)
		} else if req.URL != "" {
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Get(req.URL)
			if err != nil {
				writeError(w, http.StatusBadRequest, "failed to fetch URL")
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				writeError(w, http.StatusBadRequest, "URL returned non-200 status")
				return
			}
			icsReader = io.LimitReader(resp.Body, maxImportSize)
		} else {
			writeError(w, http.StatusBadRequest, "ics_content or url is required")
			return
		}

		events, err := ical.Decode(icsReader)
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
