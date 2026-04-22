package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/mikaelstaldal/mycal/internal/api"
	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/sanitize"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation error")
)

type EventService struct {
	repo    repository.EventRepository
	calRepo repository.CalendarRepository
}

func NewEventService(repo repository.EventRepository, calRepo repository.CalendarRepository) *EventService {
	return &EventService{repo: repo, calRepo: calRepo}
}

func (s *EventService) ListAll(calendarIDs []int64) ([]model.Event, error) {
	events, err := s.repo.ListAll(calendarIDs)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []model.Event{}
	}
	return events, nil
}

func (s *EventService) List(from, to string, calendarIDs []int64) ([]model.Event, error) {
	events, err := s.repo.List(from, to, calendarIDs)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []model.Event{}
	}

	// Expand recurring events
	recurring, err := s.repo.ListRecurring(to, calendarIDs)
	if err != nil {
		return nil, err
	}

	fromTime, _ := time.Parse(time.RFC3339, from)
	toTime, _ := time.Parse(time.RFC3339, to)

	var expanded []model.Event
	for _, re := range recurring {
		expanded = append(expanded, expandRecurring(re, fromTime, toTime)...)
	}

	// Fetch overrides for recurring parents and apply them
	if len(recurring) > 0 {
		parentIDs := make([]int64, len(recurring))
		for i, re := range recurring {
			parentIDs[i] = re.ID
		}
		overrides, err := s.repo.ListOverrides(parentIDs, from, to)
		if err != nil {
			return nil, err
		}
		if len(overrides) > 0 {
			expanded = applyOverrides(expanded, overrides, fromTime, toTime)
		}
	}

	if len(expanded) > 0 {
		events = mergeEvents(events, expanded)
	}

	return events, nil
}

// applyOverrides replaces expanded instances with their overrides.
// An override matches an expanded instance when parentID matches and
// RecurrenceOriginalStart matches the instance's StartTime.
func applyOverrides(expanded []model.Event, overrides []model.Event, from, to time.Time) []model.Event {
	// Build lookup: parentID+originalStart -> override
	type overrideKey struct {
		parentID      int64
		originalStart string
	}
	overrideMap := make(map[overrideKey]model.Event, len(overrides))
	for _, o := range overrides {
		if o.RecurrenceParentID != nil {
			overrideMap[overrideKey{*o.RecurrenceParentID, o.RecurrenceOriginalStart}] = o
		}
	}

	var result []model.Event
	replaced := make(map[overrideKey]bool)

	for _, inst := range expanded {
		key := overrideKey{inst.ID, inst.StartTime}
		if override, ok := overrideMap[key]; ok {
			// Replace it with override if it falls in the query window
			oStart, _ := time.Parse(time.RFC3339, override.StartTime)
			oEnd, _ := time.Parse(time.RFC3339, override.EndTime)
			if oEnd.After(from) && oStart.Before(to) {
				result = append(result, override)
			}
			replaced[key] = true
		} else {
			result = append(result, inst)
		}
	}

	// Include any overrides not matched to expanded instances (edge case: override moved outside normal expansion)
	for key, override := range overrideMap {
		if !replaced[key] {
			oStart, _ := time.Parse(time.RFC3339, override.StartTime)
			oEnd, _ := time.Parse(time.RFC3339, override.EndTime)
			if oEnd.After(from) && oStart.Before(to) {
				result = append(result, override)
			}
		}
	}

	return result
}

func (s *EventService) Search(query, from, to string, calendarIDs []int64) ([]model.Event, error) {
	events, err := s.repo.Search(query, from, to, calendarIDs)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []model.Event{}
	}
	return events, nil
}

func (s *EventService) GetByID(id int64) (*model.Event, error) {
	e, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, ErrNotFound
	}
	return e, nil
}

func (s *EventService) GetInstance(parentID int64, instanceStart string) (*model.Event, error) {
	if _, err := time.Parse(time.RFC3339, instanceStart); err != nil {
		return nil, fmt.Errorf("%w: instance_start must be RFC 3339 format", ErrValidation)
	}

	// Try override first
	override, err := s.repo.GetOverride(parentID, instanceStart)
	if err != nil {
		return nil, err
	}
	if override != nil {
		return override, nil
	}

	// Fall back to parent and construct the instance
	parent, err := s.repo.GetByID(parentID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, ErrNotFound
	}
	if !parent.IsRecurring() {
		return nil, ErrNotFound
	}

	// Verify instanceStart is a valid occurrence by expanding
	instStartTime, _ := time.Parse(time.RFC3339, instanceStart)
	parentStart, _ := time.Parse(time.RFC3339, parent.StartTime)
	parentEnd, _ := time.Parse(time.RFC3339, parent.EndTime)
	dur := parentEnd.Sub(parentStart)

	inst := *parent
	inst.StartTime = instanceStart
	inst.EndTime = instStartTime.Add(dur).Format(time.RFC3339)
	inst.RecurrenceIndex = 1 // non-zero to indicate it's an expanded instance
	return &inst, nil
}

func (s *EventService) Create(req *api.CreateEventRequest) (*model.Event, error) {
	startTime, endTime, err := ValidateCreateEventRequest(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	e := &model.Event{
		Title:                sanitize.HTML(req.Title),
		Description:          sanitize.HTML(req.Description.Or("")),
		StartTime:            startTime,
		EndTime:              endTime,
		AllDay:               req.AllDay,
		Color:                req.Color.Or(""),
		RecurrenceFreq:       string(req.RecurrenceFreq.Value),
		RecurrenceCount:      req.RecurrenceCount.Or(0),
		RecurrenceUntil:      req.RecurrenceUntil.Or(""),
		RecurrenceInterval:   req.RecurrenceInterval.Or(0),
		RecurrenceByDay:      req.RecurrenceByDay.Or(""),
		RecurrenceByMonthDay: req.RecurrenceByMonthday.Or(""),
		RecurrenceByMonth:    req.RecurrenceByMonth.Or(""),
		ExDates:              req.Exdates.Or(""),
		RDates:               req.Rdates.Or(""),
		Duration:             req.Duration.Or(""),
		Categories:           req.Categories.Or(""),
		ReminderMinutes:      req.ReminderMinutes.Or(0),
		Location:             req.Location.Or(""),
	}
	if req.URL.Set {
		e.URL = req.URL.Value.String()
	}
	if req.Latitude.Set && !req.Latitude.Null {
		v := req.Latitude.Value
		e.Latitude = &v
	}
	if req.Longitude.Set && !req.Longitude.Null {
		v := req.Longitude.Value
		e.Longitude = &v
	}
	if err := s.repo.Create(e); err != nil {
		return nil, err
	}
	return e, nil
}

const dateOnly = "2006-01-02"

func (s *EventService) Update(id int64, req *api.UpdateEventRequest) (*model.Event, error) {
	if err := ValidateUpdateEventRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrNotFound
	}

	if req.Title.Set {
		existing.Title = req.Title.Value
	}
	if req.Description.Set {
		existing.Description = sanitize.HTML(req.Description.Value)
	}
	if req.AllDay.Set {
		existing.AllDay = req.AllDay.Value
	}
	if req.Color.Set {
		existing.Color = req.Color.Value
	}
	if req.RecurrenceFreq.Set {
		existing.RecurrenceFreq = string(req.RecurrenceFreq.Value)
	}
	if req.RecurrenceCount.Set {
		existing.RecurrenceCount = req.RecurrenceCount.Value
	}
	if req.RecurrenceUntil.Set {
		existing.RecurrenceUntil = req.RecurrenceUntil.Value
	}
	if req.RecurrenceInterval.Set {
		existing.RecurrenceInterval = req.RecurrenceInterval.Value
	}
	if req.RecurrenceByDay.Set {
		existing.RecurrenceByDay = req.RecurrenceByDay.Value
	}
	if req.RecurrenceByMonthday.Set {
		existing.RecurrenceByMonthDay = req.RecurrenceByMonthday.Value
	}
	if req.RecurrenceByMonth.Set {
		existing.RecurrenceByMonth = req.RecurrenceByMonth.Value
	}
	if req.Exdates.Set {
		existing.ExDates = req.Exdates.Value
	}
	if req.Rdates.Set {
		existing.RDates = req.Rdates.Value
	}
	if req.Duration.Set {
		existing.Duration = req.Duration.Value
	}
	if req.Categories.Set {
		existing.Categories = req.Categories.Value
	}
	if req.URL.Set {
		existing.URL = req.URL.Value.String()
	}
	if req.ReminderMinutes.Set {
		existing.ReminderMinutes = req.ReminderMinutes.Value
	}
	if req.Location.Set {
		existing.Location = req.Location.Value
	}
	if req.Latitude.Set && !req.Latitude.Null {
		v := req.Latitude.Value
		existing.Latitude = &v
	}
	if req.Longitude.Set && !req.Longitude.Null {
		v := req.Longitude.Value
		existing.Longitude = &v
	}

	// If Duration is set, recompute EndTime
	if req.Duration.Set && req.Duration.Value != "" {
		dur, err := model.ParseDuration(req.Duration.Value)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid duration: %s", ErrValidation, err.Error())
		}
		start, err := time.Parse(time.RFC3339, existing.StartTime)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid start_time for duration computation", ErrValidation)
		}
		existing.EndTime = start.Add(dur).Format(time.RFC3339)
	}

	// Apply start/end times
	if req.StartDate.Set {
		d := req.StartDate.Value
		existing.StartTime = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}
	if req.StartTime.Set {
		existing.StartTime = req.StartTime.Value.UTC().Format(time.RFC3339)
	}
	if req.EndDate.Set {
		d := req.EndDate.Value
		endT := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		startParsed, _ := time.Parse(time.RFC3339, existing.StartTime)
		if !endT.After(startParsed) {
			endT = startParsed.AddDate(0, 0, 1)
		}
		existing.EndTime = endT.Format(time.RFC3339)
	}
	if req.EndTime.Set {
		existing.EndTime = req.EndTime.Value.UTC().Format(time.RFC3339)
	}

	// If toggling to all-day and no new start time provided, normalize existing times
	if req.AllDay.Set && req.AllDay.Value && !req.StartDate.Set && !req.StartTime.Set {
		start, _ := time.Parse(time.RFC3339, existing.StartTime)
		normalized := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
		existing.StartTime = normalized.Format(time.RFC3339)
		if !req.EndDate.Set && !req.EndTime.Set {
			existing.EndTime = normalized.AddDate(0, 0, 1).Format(time.RFC3339)
		}
	}

	// Validate final times are consistent
	start, _ := time.Parse(time.RFC3339, existing.StartTime)
	end, _ := time.Parse(time.RFC3339, existing.EndTime)
	if !end.After(start) {
		return nil, fmt.Errorf("%w: end_time must be after start_time", ErrValidation)
	}

	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *EventService) CreateOrUpdateOverride(parentID int64, instanceStart string, req *api.UpdateEventRequest) (*model.Event, error) {
	if _, err := time.Parse(time.RFC3339, instanceStart); err != nil {
		return nil, fmt.Errorf("%w: instance_start must be RFC 3339 format", ErrValidation)
	}
	if err := ValidateUpdateEventRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}

	parent, err := s.repo.GetByID(parentID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, ErrNotFound
	}
	if !parent.IsRecurring() {
		return nil, fmt.Errorf("%w: event is not recurring", ErrValidation)
	}

	// Check for existing override
	existing, err := s.repo.GetOverride(parentID, instanceStart)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		// Update existing override
		return s.Update(existing.ID, req)
	}

	// Create a new override as a copy of the parent with updates applied
	override := &model.Event{
		Title:                   parent.Title,
		Description:             parent.Description,
		StartTime:               instanceStart,
		EndTime:                 "", // will be computed
		AllDay:                  parent.AllDay,
		Color:                   parent.Color,
		Duration:                parent.Duration,
		Categories:              parent.Categories,
		URL:                     parent.URL,
		ReminderMinutes:         parent.ReminderMinutes,
		Location:                parent.Location,
		Latitude:                parent.Latitude,
		Longitude:               parent.Longitude,
		RecurrenceParentID:      &parentID,
		RecurrenceOriginalStart: instanceStart,
	}

	// Compute EndTime from parent's duration
	parentStart, _ := time.Parse(time.RFC3339, parent.StartTime)
	parentEnd, _ := time.Parse(time.RFC3339, parent.EndTime)
	dur := parentEnd.Sub(parentStart)
	instStart, _ := time.Parse(time.RFC3339, instanceStart)
	override.EndTime = instStart.Add(dur).Format(time.RFC3339)

	// Apply updates
	if req.Title.Set {
		override.Title = req.Title.Value
	}
	if req.Description.Set {
		override.Description = sanitize.HTML(req.Description.Value)
	}
	if req.StartTime.Set {
		override.StartTime = req.StartTime.Value.UTC().Format(time.RFC3339)
	}
	if req.StartDate.Set {
		d := req.StartDate.Value
		override.StartTime = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}
	if req.EndTime.Set {
		override.EndTime = req.EndTime.Value.UTC().Format(time.RFC3339)
	}
	if req.EndDate.Set {
		d := req.EndDate.Value
		override.EndTime = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}
	if req.AllDay.Set {
		override.AllDay = req.AllDay.Value
	}
	if req.Color.Set {
		override.Color = req.Color.Value
	}
	if req.Duration.Set {
		override.Duration = req.Duration.Value
		if req.Duration.Value != "" {
			d, err := model.ParseDuration(req.Duration.Value)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid duration: %s", ErrValidation, err.Error())
			}
			s2, _ := time.Parse(time.RFC3339, override.StartTime)
			override.EndTime = s2.Add(d).Format(time.RFC3339)
		}
	}
	if req.Categories.Set {
		override.Categories = req.Categories.Value
	}
	if req.URL.Set {
		override.URL = req.URL.Value.String()
	}
	if req.ReminderMinutes.Set {
		override.ReminderMinutes = req.ReminderMinutes.Value
	}
	if req.Location.Set {
		override.Location = req.Location.Value
	}
	if req.Latitude.Set && !req.Latitude.Null {
		v := req.Latitude.Value
		override.Latitude = &v
	}
	if req.Longitude.Set && !req.Longitude.Null {
		v := req.Longitude.Value
		override.Longitude = &v
	}

	if err := s.repo.Create(override); err != nil {
		return nil, err
	}
	return override, nil
}

// buildEventForImport validates a parsed event and returns a model.Event ready to persist.
func buildEventForImport(e model.Event) (*model.Event, error) {
	req := &api.CreateEventRequest{
		Title:  e.Title,
		AllDay: e.AllDay,
	}
	if e.AllDay {
		if t, err := time.Parse(time.RFC3339, e.StartTime); err == nil {
			req.StartDate = api.NewOptDate(t)
		} else if t, err := time.Parse(dateOnly, e.StartTime); err == nil {
			req.StartDate = api.NewOptDate(t)
		}
		if t, err := time.Parse(time.RFC3339, e.EndTime); err == nil {
			req.EndDate = api.NewOptDate(t)
		} else if t, err := time.Parse(dateOnly, e.EndTime); err == nil {
			req.EndDate = api.NewOptDate(t)
		}
	} else {
		if t, err := time.Parse(time.RFC3339, e.StartTime); err == nil {
			req.StartTime = api.NewOptDateTime(t)
		}
		if t, err := time.Parse(time.RFC3339, e.EndTime); err == nil {
			req.EndTime = api.NewOptDateTime(t)
		}
	}
	// Don't pass Duration — EndTime is already computed by the iCal decoder.
	if e.Description != "" {
		req.Description = api.NewOptString(e.Description)
	}
	if e.Color != "" {
		req.Color = api.NewOptString(e.Color)
	}
	if e.RecurrenceFreq != "" {
		req.RecurrenceFreq = api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreq(e.RecurrenceFreq))
	}
	if e.RecurrenceCount != 0 {
		req.RecurrenceCount = api.NewOptInt(e.RecurrenceCount)
	}
	if e.RecurrenceUntil != "" {
		req.RecurrenceUntil = api.NewOptString(e.RecurrenceUntil)
	}
	if e.RecurrenceInterval != 0 {
		req.RecurrenceInterval = api.NewOptInt(e.RecurrenceInterval)
	}
	if e.RecurrenceByDay != "" {
		req.RecurrenceByDay = api.NewOptString(e.RecurrenceByDay)
	}
	if e.RecurrenceByMonthDay != "" {
		req.RecurrenceByMonthday = api.NewOptString(e.RecurrenceByMonthDay)
	}
	if e.RecurrenceByMonth != "" {
		req.RecurrenceByMonth = api.NewOptString(e.RecurrenceByMonth)
	}
	if e.ExDates != "" {
		req.Exdates = api.NewOptString(e.ExDates)
	}
	if e.RDates != "" {
		req.Rdates = api.NewOptString(e.RDates)
	}
	if e.Categories != "" {
		req.Categories = api.NewOptString(e.Categories)
	}
	if e.URL != "" {
		if u, err := url.Parse(e.URL); err == nil {
			req.URL = api.NewOptURI(*u)
		}
	}
	if e.ReminderMinutes != 0 {
		req.ReminderMinutes = api.NewOptInt(e.ReminderMinutes)
	}
	if e.Location != "" {
		req.Location = api.NewOptString(e.Location)
	}
	if e.Latitude != nil {
		req.Latitude = api.NewOptNilFloat64(*e.Latitude)
	}
	if e.Longitude != nil {
		req.Longitude = api.NewOptNilFloat64(*e.Longitude)
	}

	startTime, endTime, err := ValidateCreateEventRequest(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	return &model.Event{
		Title:                e.Title,
		Description:          e.Description,
		StartTime:            startTime,
		EndTime:              endTime,
		AllDay:               e.AllDay,
		Color:                e.Color,
		RecurrenceFreq:       e.RecurrenceFreq,
		RecurrenceCount:      e.RecurrenceCount,
		RecurrenceUntil:      e.RecurrenceUntil,
		RecurrenceInterval:   e.RecurrenceInterval,
		RecurrenceByDay:      e.RecurrenceByDay,
		RecurrenceByMonthDay: e.RecurrenceByMonthDay,
		RecurrenceByMonth:    e.RecurrenceByMonth,
		ExDates:              e.ExDates,
		RDates:               e.RDates,
		Duration:             e.Duration,
		Categories:           e.Categories,
		URL:                  e.URL,
		ReminderMinutes:      e.ReminderMinutes,
		Location:             e.Location,
		Latitude:             e.Latitude,
		Longitude:            e.Longitude,
		IcsUID:               e.ImportUID,
	}, nil
}

func (s *EventService) ImportSingle(events []model.Event, calendarName string) (*model.Event, error) {
	if len(calendarName) > model.MaxCalendarNameLength {
		return nil, fmt.Errorf("%w: calendar name must be at most %d characters", ErrValidation, model.MaxCalendarNameLength)
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("%w: iCal source contains no events", ErrValidation)
	}
	// Reject recurrence overrides — importing a modification without the parent recurring event doesn't make sense
	for _, e := range events {
		if e.RecurrenceOriginalStart != "" {
			return nil, fmt.Errorf("%w: iCal source contains a recurrence override (RECURRENCE-ID), which is not supported for single event import", ErrValidation)
		}
	}
	if len(events) > 1 {
		return nil, fmt.Errorf("%w: iCal source contains %d events, expected exactly one", ErrValidation, len(events))
	}

	calendarID, err := s.resolveCalendarName(calendarName)
	if err != nil {
		return nil, err
	}

	ev, err := buildEventForImport(events[0])
	if err != nil {
		return nil, err
	}
	ev.CalendarID = calendarID
	if err := s.repo.Create(ev); err != nil {
		return nil, err
	}
	return ev, nil
}

func (s *EventService) Import(events []model.Event, calendarName string) (int, error) {
	if len(calendarName) > model.MaxCalendarNameLength {
		return 0, fmt.Errorf("%w: calendar name must be at most %d characters", ErrValidation, model.MaxCalendarNameLength)
	}

	calendarID, err := s.resolveCalendarName(calendarName)
	if err != nil {
		return 0, err
	}

	imported := 0

	// Separate parents and overrides
	var parents []model.Event
	var overrides []model.Event
	for _, e := range events {
		if e.RecurrenceOriginalStart != "" {
			overrides = append(overrides, e)
		} else {
			parents = append(parents, e)
		}
	}

	// Track created parent IDs by their import UID
	parentByUID := make(map[string]int64)

	for _, e := range parents {
		ev, err := buildEventForImport(e)
		if err != nil {
			continue
		}
		ev.CalendarID = calendarID
		if err := s.repo.Create(ev); err != nil {
			continue
		}
		if e.ImportUID != "" {
			parentByUID[e.ImportUID] = ev.ID
		}
		imported++
	}

	// Import overrides matched by ImportUID
	for _, e := range overrides {
		parentID, ok := parentByUID[e.ImportUID]
		if !ok {
			continue
		}
		ev := &model.Event{
			Title:                   e.Title,
			Description:             sanitize.HTML(e.Description),
			StartTime:               e.StartTime,
			EndTime:                 e.EndTime,
			AllDay:                  e.AllDay,
			Color:                   e.Color,
			Duration:                e.Duration,
			Categories:              e.Categories,
			URL:                     e.URL,
			ReminderMinutes:         e.ReminderMinutes,
			Location:                e.Location,
			Latitude:                e.Latitude,
			Longitude:               e.Longitude,
			CalendarID:              calendarID,
			RecurrenceParentID:      &parentID,
			RecurrenceOriginalStart: e.RecurrenceOriginalStart,
		}
		if err := s.repo.Create(ev); err != nil {
			continue
		}
		imported++
	}

	return imported, nil
}

func (s *EventService) AddExDate(id int64, instanceStart string) (*model.Event, error) {
	if _, err := time.Parse(time.RFC3339, instanceStart); err != nil {
		return nil, fmt.Errorf("%w: instance_start must be RFC 3339 format", ErrValidation)
	}
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrNotFound
	}
	if !existing.IsRecurring() {
		return nil, fmt.Errorf("%w: event is not recurring", ErrValidation)
	}

	// Append to existing EXDATE list
	if existing.ExDates == "" {
		existing.ExDates = instanceStart
	} else {
		existing.ExDates = existing.ExDates + "," + instanceStart
	}

	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}

	// Also delete any override for this instance
	override, err := s.repo.GetOverride(id, instanceStart)
	if err != nil {
		return nil, err
	}
	if override != nil {
		_ = s.repo.Delete(override.ID)
	}

	return existing, nil
}

func (s *EventService) RemoveExDate(id int64, instanceStart string) (*model.Event, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrNotFound
	}

	// Remove the specified EXDATE
	var remaining []string
	for _, exd := range strings.Split(existing.ExDates, ",") {
		exd = strings.TrimSpace(exd)
		if exd != "" && exd != instanceStart {
			remaining = append(remaining, exd)
		}
	}
	existing.ExDates = strings.Join(remaining, ",")

	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *EventService) resolveCalendarName(name string) (int64, error) {
	if name == "" {
		return 0, nil
	}
	cal, err := s.calRepo.GetCalendarByName(name)
	if err != nil {
		return 0, err
	}
	if cal != nil {
		return cal.ID, nil
	}
	newCal := &model.Calendar{Name: name, Color: "dodgerblue"}
	if err := s.calRepo.CreateCalendar(newCal); err != nil {
		return 0, err
	}
	return newCal.ID, nil
}

func (s *EventService) Delete(id int64) error {
	// Also delete all overrides for this parent
	_ = s.repo.DeleteByParentID(id)

	err := s.repo.Delete(id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
