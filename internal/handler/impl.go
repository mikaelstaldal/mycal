package handler

import (
	"bytes"
	"context"
	"errors"
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

// ---- Type conversion helpers ----

func toOptStr(s string) api.OptString {
	if s != "" {
		return api.NewOptString(s)
	}
	return api.OptString{}
}

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

func modelEventToAPI(e *model.Event) api.Event {
	ae := api.Event{
		ID:        e.StringID,
		Title:     e.Title,
		StartTime: e.StartTime,
		EndTime:   e.EndTime,
	}
	if e.ParentID != "" {
		ae.ParentID = api.NewOptString(e.ParentID)
	}
	if e.Description != "" {
		ae.Description = api.NewOptString(e.Description)
	}
	if e.AllDay {
		ae.AllDay = api.NewOptBool(true)
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
		ae.RecurrenceOriginalStart = api.NewOptString(e.RecurrenceOriginalStart)
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

func modelFeedToAPI(f *model.Feed) api.Feed {
	feedURL, _ := url.Parse(f.URL)
	af := api.Feed{
		ID:  f.StringID,
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

func apiCreateEventToModel(req *api.CreateEventRequest) *model.CreateEventRequest {
	m := &model.CreateEventRequest{
		Title:     req.Title,
		StartTime: req.StartTime,
	}
	if req.Description.Set {
		m.Description = req.Description.Value
	}
	if req.EndTime.Set {
		m.EndTime = req.EndTime.Value
	}
	if req.AllDay.Set {
		m.AllDay = req.AllDay.Value
	}
	if req.Color.Set {
		m.Color = req.Color.Value
	}
	if req.RecurrenceFreq.Set {
		m.RecurrenceFreq = string(req.RecurrenceFreq.Value)
	}
	if req.RecurrenceCount.Set {
		m.RecurrenceCount = req.RecurrenceCount.Value
	}
	if req.RecurrenceUntil.Set {
		m.RecurrenceUntil = req.RecurrenceUntil.Value
	}
	if req.RecurrenceInterval.Set {
		m.RecurrenceInterval = req.RecurrenceInterval.Value
	}
	if req.RecurrenceByDay.Set {
		m.RecurrenceByDay = req.RecurrenceByDay.Value
	}
	if req.RecurrenceByMonthday.Set {
		m.RecurrenceByMonthDay = req.RecurrenceByMonthday.Value
	}
	if req.RecurrenceByMonth.Set {
		m.RecurrenceByMonth = req.RecurrenceByMonth.Value
	}
	if req.Exdates.Set {
		m.ExDates = req.Exdates.Value
	}
	if req.Rdates.Set {
		m.RDates = req.Rdates.Value
	}
	if req.Duration.Set {
		m.Duration = req.Duration.Value
	}
	if req.Categories.Set {
		m.Categories = req.Categories.Value
	}
	if req.URL.Set {
		m.URL = req.URL.Value
	}
	if req.ReminderMinutes.Set {
		m.ReminderMinutes = req.ReminderMinutes.Value
	}
	if req.Location.Set {
		m.Location = req.Location.Value
	}
	if req.Latitude.Set && !req.Latitude.Null {
		v := req.Latitude.Value
		m.Latitude = &v
	}
	if req.Longitude.Set && !req.Longitude.Null {
		v := req.Longitude.Value
		m.Longitude = &v
	}
	return m
}

func apiUpdateEventToModel(req *api.UpdateEventRequest) *model.UpdateEventRequest {
	m := &model.UpdateEventRequest{}
	if req.Title.Set {
		m.Title = &req.Title.Value
	}
	if req.Description.Set {
		m.Description = &req.Description.Value
	}
	if req.StartTime.Set {
		m.StartTime = &req.StartTime.Value
	}
	if req.EndTime.Set {
		m.EndTime = &req.EndTime.Value
	}
	if req.AllDay.Set {
		m.AllDay = &req.AllDay.Value
	}
	if req.Color.Set {
		m.Color = &req.Color.Value
	}
	if req.RecurrenceFreq.Set {
		v := string(req.RecurrenceFreq.Value)
		m.RecurrenceFreq = &v
	}
	if req.RecurrenceCount.Set {
		m.RecurrenceCount = &req.RecurrenceCount.Value
	}
	if req.RecurrenceUntil.Set {
		m.RecurrenceUntil = &req.RecurrenceUntil.Value
	}
	if req.RecurrenceInterval.Set {
		m.RecurrenceInterval = &req.RecurrenceInterval.Value
	}
	if req.RecurrenceByDay.Set {
		m.RecurrenceByDay = &req.RecurrenceByDay.Value
	}
	if req.RecurrenceByMonthday.Set {
		m.RecurrenceByMonthDay = &req.RecurrenceByMonthday.Value
	}
	if req.RecurrenceByMonth.Set {
		m.RecurrenceByMonth = &req.RecurrenceByMonth.Value
	}
	if req.Exdates.Set {
		m.ExDates = &req.Exdates.Value
	}
	if req.Rdates.Set {
		m.RDates = &req.Rdates.Value
	}
	if req.Duration.Set {
		m.Duration = &req.Duration.Value
	}
	if req.Categories.Set {
		m.Categories = &req.Categories.Value
	}
	if req.URL.Set {
		v := req.URL.Value
		m.URL = &v
	}
	if req.ReminderMinutes.Set {
		m.ReminderMinutes = &req.ReminderMinutes.Value
	}
	if req.Location.Set {
		m.Location = &req.Location.Value
	}
	if req.Latitude.Set {
		if !req.Latitude.Null {
			v := req.Latitude.Value
			m.Latitude = &v
		}
		// null → nil pointer (same as "unchanged" in old behavior)
	}
	if req.Longitude.Set {
		if !req.Longitude.Null {
			v := req.Longitude.Value
			m.Longitude = &v
		}
	}
	return m
}

func apiCreateFeedToModel(req *api.CreateFeedRequest) *model.CreateFeedRequest {
	m := &model.CreateFeedRequest{
		URL: req.URL.String(),
	}
	if req.CalendarName.Set {
		m.CalendarName = req.CalendarName.Value
	}
	if req.CalendarColor.Set {
		m.CalendarColor = req.CalendarColor.Value
	}
	if req.RefreshIntervalMinutes.Set {
		m.RefreshIntervalMinutes = req.RefreshIntervalMinutes.Value
	}
	return m
}

func apiUpdateFeedToModel(req *api.UpdateFeedRequest) *model.UpdateFeedRequest {
	m := &model.UpdateFeedRequest{}
	if req.URL.Set {
		v := req.URL.Value.String()
		m.URL = &v
	}
	if req.CalendarName.Set {
		m.CalendarName = &req.CalendarName.Value
	}
	if req.RefreshIntervalMinutes.Set {
		m.RefreshIntervalMinutes = &req.RefreshIntervalMinutes.Value
	}
	if req.Enabled.Set {
		m.Enabled = &req.Enabled.Value
	}
	return m
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
// Returns reader, cleanup func, and error message (empty on success).
func getImportReaderFromURL(rawURL string) (io.ReadCloser, string) {
	if err := service.ValidateExternalURL(rawURL); err != nil {
		return nil, err.Error()
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, "failed to fetch URL"
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "URL returned non-200 status"
	}
	return resp.Body, ""
}

func icsResponse(svc *service.EventService, calSvc *service.CalendarService, calendarIDs []int, calendarNames []string) (io.Reader, string) {
	ids := parseCalendarIDsFromParams(calendarIDs, calendarNames, calSvc)
	events, err := svc.ListAll(ids)
	if err != nil {
		return nil, "failed to list events"
	}
	var buf bytes.Buffer
	if err := ical.Encode(&buf, events); err != nil {
		log.Printf("error encoding ical: %v", err)
		return nil, "failed to encode iCal"
	}
	return &buf, ""
}

// ---- Handler implementations ----

func (h *handlerImpl) APIV1PreferencesGet(ctx context.Context) (api.APIV1PreferencesGetRes, error) {
	prefs, err := h.prefSvc.GetAll()
	if err != nil {
		return &api.Error{Error: "failed to get preferences"}, nil
	}
	p := api.Preferences(prefs)
	return &p, nil
}

func (h *handlerImpl) APIV1PreferencesPatch(ctx context.Context, req api.Preferences) (api.APIV1PreferencesPatchRes, error) {
	result, err := h.prefSvc.Update(map[string]string(req))
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			return &api.APIV1PreferencesPatchBadRequest{Error: err.Error()}, nil
		}
		return &api.APIV1PreferencesPatchInternalServerError{Error: "failed to update preferences"}, nil
	}
	p := api.Preferences(result)
	return &p, nil
}

func (h *handlerImpl) APIV1CalendarsGet(ctx context.Context) (api.APIV1CalendarsGetRes, error) {
	calendars, err := h.calSvc.List()
	if err != nil {
		return &api.Error{Error: "failed to list calendars"}, nil
	}
	result := make(api.APIV1CalendarsGetOKApplicationJSON, len(calendars))
	for i, cal := range calendars {
		result[i] = api.Calendar{
			ID:    cal.ID,
			Name:  cal.Name,
			Color: cal.Color,
		}
	}
	return &result, nil
}

func (h *handlerImpl) APIV1CalendarsIDPatch(ctx context.Context, req *api.UpdateCalendarRequest, params api.APIV1CalendarsIDPatchParams) (api.APIV1CalendarsIDPatchRes, error) {
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
		if errors.Is(err, service.ErrValidation) {
			return &api.APIV1CalendarsIDPatchBadRequest{Error: err.Error()}, nil
		}
		if errors.Is(err, service.ErrNotFound) {
			return &api.APIV1CalendarsIDPatchNotFound{Error: "calendar not found"}, nil
		}
		return &api.APIV1CalendarsIDPatchInternalServerError{Error: "failed to update calendar"}, nil
	}
	return &api.Calendar{
		ID:    cal.ID,
		Name:  cal.Name,
		Color: cal.Color,
	}, nil
}

func (h *handlerImpl) APIV1EventsGet(ctx context.Context, params api.APIV1EventsGetParams) (api.APIV1EventsGetRes, error) {
	q := ""
	if params.Q.Set {
		q = params.Q.Value
	}
	if len(q) > maxSearchQueryLength {
		return &api.APIV1EventsGetBadRequest{Error: "search query too long"}, nil
	}

	calendarIDs := parseCalendarIDsFromParams(params.CalendarID, params.Calendar, h.calSvc)

	if q != "" {
		from := ""
		to := ""
		if params.From.Set {
			from = params.From.Value.UTC().Format(time.RFC3339)
		}
		if params.To.Set {
			to = params.To.Value.UTC().Format(time.RFC3339)
		}
		events, err := h.svc.Search(q, from, to, calendarIDs)
		if err != nil {
			return &api.APIV1EventsGetInternalServerError{Error: "failed to search events"}, nil
		}
		return eventsToRes(events), nil
	}

	if !params.From.Set || !params.To.Set {
		return &api.APIV1EventsGetBadRequest{Error: "from and to query parameters are required"}, nil
	}
	from := params.From.Value.UTC().Format(time.RFC3339)
	to := params.To.Value.UTC().Format(time.RFC3339)
	events, err := h.svc.List(from, to, calendarIDs)
	if err != nil {
		return &api.APIV1EventsGetInternalServerError{Error: "failed to list events"}, nil
	}
	return eventsToRes(events), nil
}

func eventsToRes(events []model.Event) *api.APIV1EventsGetOKApplicationJSON {
	result := make(api.APIV1EventsGetOKApplicationJSON, len(events))
	for i := range events {
		events[i].SetStringID()
		result[i] = modelEventToAPI(&events[i])
	}
	return &result
}

func (h *handlerImpl) APIV1EventsPost(ctx context.Context, req *api.CreateEventRequest) (api.APIV1EventsPostRes, error) {
	modelReq := apiCreateEventToModel(req)
	event, err := h.svc.Create(modelReq)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			return &api.APIV1EventsPostBadRequest{Error: err.Error()}, nil
		}
		return &api.APIV1EventsPostInternalServerError{Error: "failed to create event"}, nil
	}
	event.SetStringID()
	ae := modelEventToAPI(event)
	return &ae, nil
}

func (h *handlerImpl) APIV1EventsIDGet(ctx context.Context, params api.APIV1EventsIDGetParams) (api.APIV1EventsIDGetRes, error) {
	dbID, instanceStart, err := model.ParseEventID(params.ID)
	if err != nil {
		return &api.APIV1EventsIDGetBadRequest{Error: "invalid id"}, nil
	}
	var event *model.Event
	if instanceStart != "" {
		event, err = h.svc.GetInstance(dbID, instanceStart)
	} else {
		event, err = h.svc.GetByID(dbID)
	}
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &api.APIV1EventsIDGetNotFound{Error: "event not found"}, nil
		}
		return &api.APIV1EventsIDGetInternalServerError{Error: "failed to get event"}, nil
	}
	event.SetStringID()
	ae := modelEventToAPI(event)
	return &ae, nil
}

func (h *handlerImpl) APIV1EventsIDPatch(ctx context.Context, req *api.UpdateEventRequest, params api.APIV1EventsIDPatchParams) (api.APIV1EventsIDPatchRes, error) {
	dbID, instanceStart, err := model.ParseEventID(params.ID)
	if err != nil {
		return &api.APIV1EventsIDPatchBadRequest{Error: "invalid id"}, nil
	}
	modelReq := apiUpdateEventToModel(req)
	var event *model.Event
	if instanceStart != "" {
		event, err = h.svc.CreateOrUpdateOverride(dbID, instanceStart, modelReq)
	} else {
		event, err = h.svc.Update(dbID, modelReq)
	}
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &api.APIV1EventsIDPatchNotFound{Error: "event not found"}, nil
		}
		if errors.Is(err, service.ErrValidation) {
			return &api.APIV1EventsIDPatchBadRequest{Error: err.Error()}, nil
		}
		return &api.APIV1EventsIDPatchInternalServerError{Error: "failed to update event"}, nil
	}
	event.SetStringID()
	ae := modelEventToAPI(event)
	return &ae, nil
}

func (h *handlerImpl) APIV1EventsIDDelete(ctx context.Context, params api.APIV1EventsIDDeleteParams) (api.APIV1EventsIDDeleteRes, error) {
	dbID, instanceStart, err := model.ParseEventID(params.ID)
	if err != nil {
		return &api.APIV1EventsIDDeleteBadRequest{Error: "invalid id"}, nil
	}
	if instanceStart != "" {
		event, err := h.svc.AddExDate(dbID, instanceStart)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return &api.APIV1EventsIDDeleteNotFound{Error: "event not found"}, nil
			}
			if errors.Is(err, service.ErrValidation) {
				return &api.APIV1EventsIDDeleteBadRequest{Error: err.Error()}, nil
			}
			return &api.APIV1EventsIDDeleteInternalServerError{Error: "failed to add exception date"}, nil
		}
		event.SetStringID()
		ae := modelEventToAPI(event)
		return &ae, nil
	}
	if err := h.svc.Delete(dbID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &api.APIV1EventsIDDeleteNotFound{Error: "event not found"}, nil
		}
		return &api.APIV1EventsIDDeleteInternalServerError{Error: "failed to delete event"}, nil
	}
	return &api.APIV1EventsIDDeleteNoContent{}, nil
}

func (h *handlerImpl) APIV1EventsIcsGet(ctx context.Context, params api.APIV1EventsIcsGetParams) (api.APIV1EventsIcsGetRes, error) {
	reader, errMsg := icsResponse(h.svc, h.calSvc, params.CalendarID, params.Calendar)
	if errMsg != "" {
		return &api.Error{Error: errMsg}, nil
	}
	return &api.APIV1EventsIcsGetOK{Data: reader}, nil
}

func (h *handlerImpl) APIV1ImportPost(ctx context.Context, req api.APIV1ImportPostReq, params api.APIV1ImportPostParams) (api.APIV1ImportPostRes, error) {
	var reader io.Reader
	var cleanup func()

	switch r := req.(type) {
	case *api.APIV1ImportPostReqTextCalendar:
		reader = io.LimitReader(r.Data, maxImportSize)
		cleanup = func() {}
	case *api.APIV1ImportPostReqApplicationJSON:
		body, errMsg := getImportReaderFromURL(r.URL.String())
		if errMsg != "" {
			return &api.APIV1ImportPostBadRequest{Error: errMsg}, nil
		}
		reader = io.LimitReader(body, maxImportSize)
		cleanup = func() { body.Close() }
	default:
		return &api.APIV1ImportPostUnsupportedMediaType{Error: "Content-Type must be text/calendar or application/json"}, nil
	}
	defer cleanup()

	calendarName := ""
	if params.Calendar.Set {
		calendarName = params.Calendar.Value
	}

	events, err := ical.Decode(reader)
	if err != nil {
		return &api.APIV1ImportPostBadRequest{Error: "failed to parse iCalendar data"}, nil
	}
	imported, err := h.svc.Import(events, calendarName)
	if err != nil {
		return &api.APIV1ImportPostInternalServerError{Error: "failed to import events"}, nil
	}
	return &api.APIV1ImportPostOK{Imported: api.NewOptInt(imported)}, nil
}

func (h *handlerImpl) APIV1ImportSinglePost(ctx context.Context, req api.APIV1ImportSinglePostReq, params api.APIV1ImportSinglePostParams) (api.APIV1ImportSinglePostRes, error) {
	var reader io.Reader
	var cleanup func()

	switch r := req.(type) {
	case *api.APIV1ImportSinglePostReqTextCalendar:
		reader = io.LimitReader(r.Data, maxImportSize)
		cleanup = func() {}
	case *api.APIV1ImportSinglePostReqApplicationJSON:
		body, errMsg := getImportReaderFromURL(r.URL.String())
		if errMsg != "" {
			return &api.APIV1ImportSinglePostBadRequest{Error: errMsg}, nil
		}
		reader = io.LimitReader(body, maxImportSize)
		cleanup = func() { body.Close() }
	default:
		return &api.APIV1ImportSinglePostUnsupportedMediaType{Error: "Content-Type must be text/calendar or application/json"}, nil
	}
	defer cleanup()

	calendarName := ""
	if params.Calendar.Set {
		calendarName = params.Calendar.Value
	}

	events, err := ical.Decode(reader)
	if err != nil {
		return &api.APIV1ImportSinglePostBadRequest{Error: "failed to parse iCalendar data"}, nil
	}
	event, err := h.svc.ImportSingle(events, calendarName)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			return &api.APIV1ImportSinglePostBadRequest{Error: err.Error()}, nil
		}
		return &api.APIV1ImportSinglePostInternalServerError{Error: "failed to import event"}, nil
	}
	event.SetStringID()
	ae := modelEventToAPI(event)
	return &ae, nil
}

func (h *handlerImpl) APIV1FeedsGet(ctx context.Context) (api.APIV1FeedsGetRes, error) {
	feeds, err := h.feedSvc.List()
	if err != nil {
		return &api.Error{Error: "failed to list feeds"}, nil
	}
	result := make(api.APIV1FeedsGetOKApplicationJSON, len(feeds))
	for i := range feeds {
		feeds[i].SetStringID()
		result[i] = modelFeedToAPI(&feeds[i])
	}
	return &result, nil
}

func (h *handlerImpl) APIV1FeedsPost(ctx context.Context, req *api.CreateFeedRequest) (api.APIV1FeedsPostRes, error) {
	modelReq := apiCreateFeedToModel(req)
	feed, err := h.feedSvc.Create(modelReq)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			return &api.APIV1FeedsPostBadRequest{Error: err.Error()}, nil
		}
		return &api.APIV1FeedsPostInternalServerError{Error: "failed to create feed"}, nil
	}
	feed.SetStringID()
	af := modelFeedToAPI(feed)
	return &af, nil
}

func (h *handlerImpl) APIV1FeedsIDGet(ctx context.Context, params api.APIV1FeedsIDGetParams) (api.APIV1FeedsIDGetRes, error) {
	id, err := model.ParseFeedID(params.ID)
	if err != nil {
		return &api.APIV1FeedsIDGetNotFound{Error: "feed not found"}, nil
	}
	feed, err := h.feedSvc.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &api.APIV1FeedsIDGetNotFound{Error: "feed not found"}, nil
		}
		return &api.APIV1FeedsIDGetInternalServerError{Error: "failed to get feed"}, nil
	}
	feed.SetStringID()
	af := modelFeedToAPI(feed)
	return &af, nil
}

func (h *handlerImpl) APIV1FeedsIDPut(ctx context.Context, req *api.UpdateFeedRequest, params api.APIV1FeedsIDPutParams) (api.APIV1FeedsIDPutRes, error) {
	id, err := model.ParseFeedID(params.ID)
	if err != nil {
		return &api.APIV1FeedsIDPutNotFound{Error: "feed not found"}, nil
	}
	modelReq := apiUpdateFeedToModel(req)
	feed, err := h.feedSvc.Update(id, modelReq)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &api.APIV1FeedsIDPutNotFound{Error: "feed not found"}, nil
		}
		if errors.Is(err, service.ErrValidation) {
			return &api.APIV1FeedsIDPutBadRequest{Error: err.Error()}, nil
		}
		return &api.APIV1FeedsIDPutInternalServerError{Error: "failed to update feed"}, nil
	}
	feed.SetStringID()
	af := modelFeedToAPI(feed)
	return &af, nil
}

func (h *handlerImpl) APIV1FeedsIDDelete(ctx context.Context, params api.APIV1FeedsIDDeleteParams) (api.APIV1FeedsIDDeleteRes, error) {
	id, err := model.ParseFeedID(params.ID)
	if err != nil {
		return &api.APIV1FeedsIDDeleteNotFound{Error: "feed not found"}, nil
	}
	if err := h.feedSvc.Delete(id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &api.APIV1FeedsIDDeleteNotFound{Error: "feed not found"}, nil
		}
		return &api.APIV1FeedsIDDeleteInternalServerError{Error: "failed to delete feed"}, nil
	}
	return &api.APIV1FeedsIDDeleteNoContent{}, nil
}

func (h *handlerImpl) APIV1FeedsIDRefreshPost(ctx context.Context, params api.APIV1FeedsIDRefreshPostParams) (api.APIV1FeedsIDRefreshPostRes, error) {
	id, err := model.ParseFeedID(params.ID)
	if err != nil {
		return &api.APIV1FeedsIDRefreshPostNotFound{Error: "feed not found"}, nil
	}
	feed, err := h.feedSvc.RefreshFeed(id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return &api.APIV1FeedsIDRefreshPostNotFound{Error: "feed not found"}, nil
		}
		return &api.APIV1FeedsIDRefreshPostInternalServerError{Error: "failed to refresh feed"}, nil
	}
	feed.SetStringID()
	af := modelFeedToAPI(feed)
	return &af, nil
}

func (h *handlerImpl) CalendarIcsGet(ctx context.Context, params api.CalendarIcsGetParams) (api.CalendarIcsGetRes, error) {
	reader, errMsg := icsResponse(h.svc, h.calSvc, params.CalendarID, params.Calendar)
	if errMsg != "" {
		return &api.Error{Error: errMsg}, nil
	}
	return &api.CalendarIcsGetOK{Data: reader}, nil
}
