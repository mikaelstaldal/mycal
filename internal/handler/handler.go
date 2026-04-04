package handler

import (
	"log"
	"net/http"

	"github.com/mikaelstaldal/mycal/internal/api"
	"github.com/mikaelstaldal/mycal/internal/service"
)

// NewRouter creates an HTTP handler for all API routes using the ogen-generated server.
func NewRouter(svc *service.EventService, prefSvc *service.PreferencesService, feedSvc *service.FeedService, calSvc *service.CalendarService) http.Handler {
	impl := &handlerImpl{
		svc:     svc,
		prefSvc: prefSvc,
		feedSvc: feedSvc,
		calSvc:  calSvc,
	}
	server, err := api.NewServer(impl)
	if err != nil {
		log.Fatalf("ogen server init: %v", err)
	}
	return withMiddleware(addCalendarDispositionHeader(server))
}

// addCalendarDispositionHeader sets Content-Disposition for iCalendar feed responses.
func addCalendarDispositionHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/events.ics" || r.URL.Path == "/calendar.ics" {
			w.Header().Set("Content-Disposition", `attachment; filename="mycal.ics"`)
		}
		next.ServeHTTP(w, r)
	})
}
