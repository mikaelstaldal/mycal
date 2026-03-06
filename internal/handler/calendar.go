package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/mikaelstaldal/mycal/internal/service"
)

func registerCalendarRoutes(mux *http.ServeMux, calSvc *service.CalendarService) {
	mux.HandleFunc("GET /api/v1/calendars", listCalendars(calSvc))
	mux.HandleFunc("PATCH /api/v1/calendars/{id}", updateCalendar(calSvc))
}

func updateCalendar(calSvc *service.CalendarService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid calendar id")
			return
		}
		var req struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		cal, err := calSvc.Update(id, req.Name, req.Color)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeError(w, http.StatusNotFound, "calendar not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update calendar")
			return
		}
		writeJSON(w, http.StatusOK, cal)
	}
}

func listCalendars(calSvc *service.CalendarService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		calendars, err := calSvc.List()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list calendars")
			return
		}
		writeJSON(w, http.StatusOK, calendars)
	}
}
