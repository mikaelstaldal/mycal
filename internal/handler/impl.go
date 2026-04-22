package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/mikaelstaldal/mycal/internal/api"
	"github.com/mikaelstaldal/mycal/internal/ical"
	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/service"
)

const (
	maxSearchQueryLength = 200
	maxImportSize        = 10 * 1024 * 1024 // 10 MiB
)

type handlerImpl struct {
	svc     *service.EventService
	prefSvc *service.PreferencesService
	feedSvc *service.FeedService
	calSvc  *service.CalendarService
}

// httpError is a sentinel error carrying an explicit HTTP status code.
type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

func badRequest(msg string) error  { return &httpError{http.StatusBadRequest, msg} }
func unsupported(msg string) error { return &httpError{http.StatusUnsupportedMediaType, msg} }

// NewError implements api.Handler — maps Go errors to HTTP error responses.
func (h *handlerImpl) NewError(_ context.Context, err error) *api.ErrorStatusCode {
	var he *httpError
	if errors.As(err, &he) {
		return &api.ErrorStatusCode{StatusCode: he.status, Response: api.Error{Error: he.msg}}
	}
	if errors.Is(err, service.ErrNotFound) {
		return &api.ErrorStatusCode{StatusCode: http.StatusNotFound, Response: api.Error{Error: err.Error()}}
	}
	if errors.Is(err, service.ErrValidation) {
		return &api.ErrorStatusCode{StatusCode: http.StatusBadRequest, Response: api.Error{Error: err.Error()}}
	}
	log.Printf("internal error: %v", err)
	return &api.ErrorStatusCode{StatusCode: http.StatusInternalServerError, Response: api.Error{Error: "internal server error"}}
}

// ---- Type conversion helpers ----

func toOptDateTime(s string) api.OptDateTime {
	if s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return api.NewOptDateTime(t)
		}
	}
	return api.OptDateTime{}
}

func toOptURI(s string) api.OptURI {
	if s != "" {
		if u, err := url.Parse(s); err == nil {
			return api.NewOptURI(*u)
		}
	}
	return api.OptURI{}
}

func modelEventToAPI(e *model.Event) *api.Event {
	ae := &api.Event{
		ID:    e.StringID,
		Title: e.Title,
	}
	if e.AllDay {
		ae.AllDay = api.NewOptBool(true)
		if t, err := time.Parse(time.RFC3339, e.StartTime); err == nil {
			ae.StartDate = api.NewOptDate(t)
		}
		if t, err := time.Parse(time.RFC3339, e.EndTime); err == nil {
			ae.EndDate = api.NewOptDate(t)
		}
	} else {
		if t, err := time.Parse(time.RFC3339, e.StartTime); err == nil {
			ae.StartTime = api.NewOptDateTime(t)
		}
		if t, err := time.Parse(time.RFC3339, e.EndTime); err == nil {
			ae.EndTime = api.NewOptDateTime(t)
		}
	}
	if e.ParentID != "" {
		ae.ParentID = api.NewOptString(e.ParentID)
	}
	if e.Description != "" {
		ae.Description = api.NewOptString(e.Description)
	}
	if e.Color != "" {
		ae.Color = api.NewOptString(e.Color)
	}
	if e.RecurrenceFreq != "" {
		ae.RecurrenceFreq = api.NewOptEventRecurrenceFreq(api.EventRecurrenceFreq(e.RecurrenceFreq))
	}
	if e.RecurrenceCount != 0 {
		ae.RecurrenceCount = api.NewOptInt(e.RecurrenceCount)
	}
	ae.RecurrenceUntil = toOptDateTime(e.RecurrenceUntil)
	if e.RecurrenceInterval != 0 {
		ae.RecurrenceInterval = api.NewOptInt(e.RecurrenceInterval)
	}
	if e.RecurrenceByDay != "" {
		ae.RecurrenceByDay = api.NewOptString(e.RecurrenceByDay)
	}
	if e.RecurrenceByMonthDay != "" {
		ae.RecurrenceByMonthday = api.NewOptString(e.RecurrenceByMonthDay)
	}
	if e.RecurrenceByMonth != "" {
		ae.RecurrenceByMonth = api.NewOptString(e.RecurrenceByMonth)
	}
	if e.ExDates != "" {
		ae.Exdates = api.NewOptString(e.ExDates)
	}
	if e.RDates != "" {
		ae.Rdates = api.NewOptString(e.RDates)
	}
	if e.RecurrenceParentID != nil {
		ae.RecurrenceParentID = api.NewOptNilInt64(*e.RecurrenceParentID)
	}
	if e.RecurrenceOriginalStart != "" {
		ae.RecurrenceOriginalStart = toOptDateTime(e.RecurrenceOriginalStart)
	}
	if e.Duration != "" {
		ae.Duration = api.NewOptString(e.Duration)
	}
	if e.Categories != "" {
		ae.Categories = api.NewOptString(e.Categories)
	}
	ae.URL = toOptURI(e.URL)
	if e.ReminderMinutes != 0 {
		ae.ReminderMinutes = api.NewOptInt(e.ReminderMinutes)
	}
	if e.Location != "" {
		ae.Location = api.NewOptString(e.Location)
	}
	if e.Latitude != nil {
		ae.Latitude = api.NewOptNilFloat64(*e.Latitude)
	}
	ae.CalendarID = api.NewOptInt64(e.CalendarID)
	if e.CalendarName != "" {
		ae.CalendarName = api.NewOptString(e.CalendarName)
	}
	ae.CreatedAt = toOptDateTime(e.CreatedAt)
	ae.UpdatedAt = toOptDateTime(e.UpdatedAt)
	return ae
}

func modelFeedToAPI(f *model.Feed) *api.Feed {
	feedURL, _ := url.Parse(f.URL)
	af := &api.Feed{
		ID:  f.ID,
		URL: *feedURL,
	}
	if f.CalendarID != 0 {
		af.CalendarID = api.NewOptInt64(f.CalendarID)
	}
	if f.CalendarName != "" {
		af.CalendarName = api.NewOptString(f.CalendarName)
	}
	if f.RefreshIntervalMinutes != 0 {
		af.RefreshIntervalMinutes = api.NewOptInt(f.RefreshIntervalMinutes)
	}
	af.LastRefreshedAt = toOptDateTime(f.LastRefreshedAt)
	if f.LastError != "" {
		af.LastError = api.NewOptString(f.LastError)
	}
	af.Enabled = api.NewOptBool(f.Enabled)
	af.CreatedAt = toOptDateTime(f.CreatedAt)
	af.UpdatedAt = toOptDateTime(f.UpdatedAt)
	return af
}

func parseCalendarIDsFromParams(calendarIDs []int, calendarNames []string, calSvc *service.CalendarService) []int64 {
	if len(calendarIDs) == 0 && len(calendarNames) == 0 {
		return nil // nil = all calendars
	}
	var ids []int64
	for _, id := range calendarIDs {
		ids = append(ids, int64(id))
	}
	for _, name := range calendarNames {
		calID, err := calSvc.GetOrCreateByName(name)
		if err == nil {
			ids = append(ids, calID)
		}
	}
	if ids == nil {
		ids = []int64{} // empty = match nothing
	}
	return ids
}

// getImportReaderFromURL fetches iCalendar data from the given URL.
func getImportReaderFromURL(rawURL string) (io.ReadCloser, error) {
	if err := service.ValidateExternalURL(rawURL); err != nil {
		return nil, badRequest(err.Error())
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, badRequest("failed to fetch URL")
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, badRequest("URL returned non-200 status")
	}
	return resp.Body, nil
}

func icsResponse(svc *service.EventService, calSvc *service.CalendarService, calendarIDs []int, calendarNames []string) (io.Reader, error) {
	ids := parseCalendarIDsFromParams(calendarIDs, calendarNames, calSvc)
	events, err := svc.ListAll(ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	var buf bytes.Buffer
	if err := ical.Encode(&buf, events); err != nil {
		return nil, fmt.Errorf("failed to encode iCal: %w", err)
	}
	return &buf, nil
}

// ---- Handler implementations ----

func (h *handlerImpl) APIV1PreferencesGet(ctx context.Context) (api.Preferences, error) {
	prefs, err := h.prefSvc.GetAll()
	if err != nil {
		return nil, err
	}
	return api.Preferences(prefs), nil
}

func (h *handlerImpl) APIV1PreferencesPatch(ctx context.Context, req api.Preferences) (api.Preferences, error) {
	result, err := h.prefSvc.Update(map[string]string(req))
	if err != nil {
		return nil, err
	}
	return api.Preferences(result), nil
}

func (h *handlerImpl) APIV1CalendarsGet(ctx context.Context) ([]api.Calendar, error) {
	calendars, err := h.calSvc.List()
	if err != nil {
		return nil, err
	}
	result := make([]api.Calendar, len(calendars))
	for i, cal := range calendars {
		result[i] = api.Calendar{ID: cal.ID, Name: cal.Name, Color: cal.Color}
	}
	return result, nil
}

func (h *handlerImpl) APIV1CalendarsIDPatch(ctx context.Context, req *api.UpdateCalendarRequest, params api.APIV1CalendarsIDPatchParams) (*api.Calendar, error) {
	name := ""
	if req.Name.Set {
		name = req.Name.Value
	}
	color := ""
	if req.Color.Set {
		color = req.Color.Value
	}
	cal, err := h.calSvc.Update(params.ID, name, color)
	if err != nil {
		return nil, err
	}
	return &api.Calendar{ID: cal.ID, Name: cal.Name, Color: cal.Color}, nil
}

func (h *handlerImpl) APIV1EventsGet(ctx context.Context, params api.APIV1EventsGetParams) ([]api.Event, error) {
	q := ""
	if params.Q.Set {
		q = params.Q.Value
	}
	if len(q) > maxSearchQueryLength {
		return nil, badRequest("search query too long")
	}

	calendarIDs := parseCalendarIDsFromParams(params.CalendarID, params.Calendar, h.calSvc)

	if q != "" {
		from, to := "", ""
		if params.From.Set {
			from = params.From.Value.UTC().Format(time.RFC3339)
		}
		if params.To.Set {
			to = params.To.Value.UTC().Format(time.RFC3339)
		}
		events, err := h.svc.Search(q, from, to, calendarIDs)
		if err != nil {
			return nil, err
		}
		return eventsToAPI(events), nil
	}

	if !params.From.Set || !params.To.Set {
		return nil, badRequest("from and to query parameters are required")
	}
	from := params.From.Value.UTC().Format(time.RFC3339)
	to := params.To.Value.UTC().Format(time.RFC3339)
	events, err := h.svc.List(from, to, calendarIDs)
	if err != nil {
		return nil, err
	}
	return eventsToAPI(events), nil
}

func eventsToAPI(events []model.Event) []api.Event {
	result := make([]api.Event, len(events))
	for i := range events {
		events[i].SetStringID()
		result[i] = *modelEventToAPI(&events[i])
	}
	return result
}

func (h *handlerImpl) APIV1EventsPost(ctx context.Context, req *api.CreateEventRequest) (*api.Event, error) {
	event, err := h.svc.Create(req)
	if err != nil {
		return nil, err
	}
	event.SetStringID()
	return modelEventToAPI(event), nil
}

func (h *handlerImpl) APIV1EventsIDGet(ctx context.Context, params api.APIV1EventsIDGetParams) (*api.Event, error) {
	dbID, instanceStart, err := model.ParseEventID(params.ID)
	if err != nil {
		return nil, badRequest("invalid id")
	}
	var event *model.Event
	if instanceStart != "" {
		event, err = h.svc.GetInstance(dbID, instanceStart)
	} else {
		event, err = h.svc.GetByID(dbID)
	}
	if err != nil {
		return nil, err
	}
	event.SetStringID()
	return modelEventToAPI(event), nil
}

func (h *handlerImpl) APIV1EventsIDPatch(ctx context.Context, req *api.UpdateEventRequest, params api.APIV1EventsIDPatchParams) (*api.Event, error) {
	dbID, instanceStart, err := model.ParseEventID(params.ID)
	if err != nil {
		return nil, badRequest("invalid id")
	}
	var event *model.Event
	if instanceStart != "" {
		event, err = h.svc.CreateOrUpdateOverride(dbID, instanceStart, req)
	} else {
		event, err = h.svc.Update(dbID, req)
	}
	if err != nil {
		return nil, err
	}
	event.SetStringID()
	return modelEventToAPI(event), nil
}

func (h *handlerImpl) APIV1EventsIDDelete(ctx context.Context, params api.APIV1EventsIDDeleteParams) (api.APIV1EventsIDDeleteRes, error) {
	dbID, instanceStart, err := model.ParseEventID(params.ID)
	if err != nil {
		return nil, badRequest("invalid id")
	}
	if instanceStart != "" {
		event, err := h.svc.AddExDate(dbID, instanceStart)
		if err != nil {
			return nil, err
		}
		event.SetStringID()
		return modelEventToAPI(event), nil
	}
	if err := h.svc.Delete(dbID); err != nil {
		return nil, err
	}
	return &api.APIV1EventsIDDeleteNoContent{}, nil
}

func (h *handlerImpl) APIV1EventsIcsGet(ctx context.Context, params api.APIV1EventsIcsGetParams) (api.APIV1EventsIcsGetOK, error) {
	reader, err := icsResponse(h.svc, h.calSvc, params.CalendarID, params.Calendar)
	if err != nil {
		return api.APIV1EventsIcsGetOK{}, err
	}
	return api.APIV1EventsIcsGetOK{Data: reader}, nil
}

func (h *handlerImpl) APIV1ImportPost(ctx context.Context, req api.APIV1ImportPostReq, params api.APIV1ImportPostParams) (*api.APIV1ImportPostOK, error) {
	var reader io.Reader
	var cleanup func()

	switch r := req.(type) {
	case *api.APIV1ImportPostReqTextCalendar:
		reader = io.LimitReader(r.Data, maxImportSize)
		cleanup = func() {}
	case *api.APIV1ImportPostReqApplicationJSON:
		body, err := getImportReaderFromURL(r.URL.String())
		if err != nil {
			return nil, err
		}
		reader = io.LimitReader(body, maxImportSize)
		cleanup = func() { body.Close() }
	default:
		return nil, unsupported("Content-Type must be text/calendar or application/json")
	}
	defer cleanup()

	calendarName := ""
	if params.Calendar.Set {
		calendarName = params.Calendar.Value
	}

	events, err := ical.Decode(reader)
	if err != nil {
		return nil, badRequest("failed to parse iCalendar data")
	}
	imported, err := h.svc.Import(events, calendarName)
	if err != nil {
		return nil, err
	}
	return &api.APIV1ImportPostOK{Imported: api.NewOptInt(imported)}, nil
}

func (h *handlerImpl) APIV1ImportSinglePost(ctx context.Context, req api.APIV1ImportSinglePostReq, params api.APIV1ImportSinglePostParams) (*api.Event, error) {
	var reader io.Reader
	var cleanup func()

	switch r := req.(type) {
	case *api.APIV1ImportSinglePostReqTextCalendar:
		reader = io.LimitReader(r.Data, maxImportSize)
		cleanup = func() {}
	case *api.APIV1ImportSinglePostReqApplicationJSON:
		body, err := getImportReaderFromURL(r.URL.String())
		if err != nil {
			return nil, err
		}
		reader = io.LimitReader(body, maxImportSize)
		cleanup = func() { body.Close() }
	default:
		return nil, unsupported("Content-Type must be text/calendar or application/json")
	}
	defer cleanup()

	calendarName := ""
	if params.Calendar.Set {
		calendarName = params.Calendar.Value
	}

	events, err := ical.Decode(reader)
	if err != nil {
		return nil, badRequest("failed to parse iCalendar data")
	}
	event, err := h.svc.ImportSingle(events, calendarName)
	if err != nil {
		return nil, err
	}
	event.SetStringID()
	return modelEventToAPI(event), nil
}

func (h *handlerImpl) APIV1FeedsGet(ctx context.Context) ([]api.Feed, error) {
	feeds, err := h.feedSvc.List()
	if err != nil {
		return nil, err
	}
	result := make([]api.Feed, len(feeds))
	for i := range feeds {
		result[i] = *modelFeedToAPI(&feeds[i])
	}
	return result, nil
}

func (h *handlerImpl) APIV1FeedsPost(ctx context.Context, req *api.CreateFeedRequest) (*api.Feed, error) {
	feed, err := h.feedSvc.Create(req)
	if err != nil {
		return nil, err
	}
	return modelFeedToAPI(feed), nil
}

func (h *handlerImpl) APIV1FeedsIDGet(ctx context.Context, params api.APIV1FeedsIDGetParams) (*api.Feed, error) {
	feed, err := h.feedSvc.GetByID(params.ID)
	if err != nil {
		return nil, err
	}
	return modelFeedToAPI(feed), nil
}

func (h *handlerImpl) APIV1FeedsIDPut(ctx context.Context, req *api.UpdateFeedRequest, params api.APIV1FeedsIDPutParams) (*api.Feed, error) {
	feed, err := h.feedSvc.Update(params.ID, req)
	if err != nil {
		return nil, err
	}
	return modelFeedToAPI(feed), nil
}

func (h *handlerImpl) APIV1FeedsIDDelete(ctx context.Context, params api.APIV1FeedsIDDeleteParams) error {
	return h.feedSvc.Delete(params.ID)
}

func (h *handlerImpl) APIV1FeedsIDRefreshPost(ctx context.Context, params api.APIV1FeedsIDRefreshPostParams) (*api.Feed, error) {
	feed, err := h.feedSvc.RefreshFeed(params.ID)
	if err != nil {
		return nil, err
	}
	return modelFeedToAPI(feed), nil
}

func (h *handlerImpl) CalendarIcsGet(ctx context.Context, params api.CalendarIcsGetParams) (api.CalendarIcsGetOK, error) {
	reader, err := icsResponse(h.svc, h.calSvc, params.CalendarID, params.Calendar)
	if err != nil {
		return api.CalendarIcsGetOK{}, err
	}
	return api.CalendarIcsGetOK{Data: reader}, nil
}
