package handler

import (
	"net/http"

	"github.com/mikaelstaldal/mycal/internal/service"
)

func registerCalendarRoutes(mux *http.ServeMux, calSvc *service.CalendarService) {
	mux.HandleFunc("GET /api/v1/calendars", listCalendars(calSvc))
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
