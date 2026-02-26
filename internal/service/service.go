package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/sanitize"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation error")
)

type EventService struct {
	repo repository.EventRepository
}

func NewEventService(repo repository.EventRepository) *EventService {
	return &EventService{repo: repo}
}

func (s *EventService) ListAll() ([]model.Event, error) {
	events, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []model.Event{}
	}
	return events, nil
}

func (s *EventService) List(from, to string) ([]model.Event, error) {
	events, err := s.repo.List(from, to)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []model.Event{}
	}

	// Expand recurring events
	recurring, err := s.repo.ListRecurring(to)
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
		overrides, err := s.repo.ListOverrides(parentIDs)
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
			// Replace with override if it falls in the query window
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

func (s *EventService) Search(query, from, to string) ([]model.Event, error) {
	events, err := s.repo.Search(query, from, to)
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

func (s *EventService) Create(req *model.CreateEventRequest) (*model.Event, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	e := &model.Event{
		Title:              req.Title,
		Description:        sanitize.HTML(req.Description),
		StartTime:          req.StartTime,
		EndTime:            req.EndTime,
		AllDay:             req.AllDay,
		Color:              req.Color,
		RecurrenceFreq:     req.RecurrenceFreq,
		RecurrenceCount:    req.RecurrenceCount,
		RecurrenceUntil:    req.RecurrenceUntil,
		RecurrenceInterval: req.RecurrenceInterval,
		RecurrenceByDay:    req.RecurrenceByDay,
		RecurrenceByMonthDay: req.RecurrenceByMonthDay,
		RecurrenceByMonth:  req.RecurrenceByMonth,
		ExDates:            req.ExDates,
		RDates:             req.RDates,
		Duration:           req.Duration,
		Categories:         req.Categories,
		URL:                req.URL,
		ReminderMinutes:    req.ReminderMinutes,
		Location:           req.Location,
		Latitude:           req.Latitude,
		Longitude:          req.Longitude,
	}
	if err := s.repo.Create(e); err != nil {
		return nil, err
	}
	return e, nil
}

const dateOnly = "2006-01-02"

func (s *EventService) Update(id int64, req *model.UpdateEventRequest) (*model.Event, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrNotFound
	}

	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Description != nil {
		existing.Description = sanitize.HTML(*req.Description)
	}
	if req.AllDay != nil {
		existing.AllDay = *req.AllDay
	}
	if req.Color != nil {
		existing.Color = *req.Color
	}
	if req.RecurrenceFreq != nil {
		existing.RecurrenceFreq = *req.RecurrenceFreq
	}
	if req.RecurrenceCount != nil {
		existing.RecurrenceCount = *req.RecurrenceCount
	}
	if req.RecurrenceUntil != nil {
		existing.RecurrenceUntil = *req.RecurrenceUntil
	}
	if req.RecurrenceInterval != nil {
		existing.RecurrenceInterval = *req.RecurrenceInterval
	}
	if req.RecurrenceByDay != nil {
		existing.RecurrenceByDay = *req.RecurrenceByDay
	}
	if req.RecurrenceByMonthDay != nil {
		existing.RecurrenceByMonthDay = *req.RecurrenceByMonthDay
	}
	if req.RecurrenceByMonth != nil {
		existing.RecurrenceByMonth = *req.RecurrenceByMonth
	}
	if req.ExDates != nil {
		existing.ExDates = *req.ExDates
	}
	if req.RDates != nil {
		existing.RDates = *req.RDates
	}
	if req.Duration != nil {
		existing.Duration = *req.Duration
	}
	if req.Categories != nil {
		existing.Categories = *req.Categories
	}
	if req.URL != nil {
		existing.URL = *req.URL
	}
	if req.ReminderMinutes != nil {
		existing.ReminderMinutes = *req.ReminderMinutes
	}
	if req.Location != nil {
		existing.Location = *req.Location
	}
	if req.Latitude != nil {
		existing.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		existing.Longitude = req.Longitude
	}

	// If Duration is set, recompute EndTime
	if req.Duration != nil && *req.Duration != "" {
		dur, err := model.ParseDuration(*req.Duration)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid duration: %s", ErrValidation, err.Error())
		}
		start, err := time.Parse(time.RFC3339, existing.StartTime)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid start_time for duration computation", ErrValidation)
		}
		existing.EndTime = start.Add(dur).Format(time.RFC3339)
	}

	if existing.AllDay {
		// Handle all-day event updates
		if req.StartTime != nil {
			start, err := time.Parse(dateOnly, *req.StartTime)
			if err != nil {
				return nil, fmt.Errorf("%w: start_time must be YYYY-MM-DD for all-day events", ErrValidation)
			}
			existing.StartTime = start.UTC().Format(time.RFC3339)
		}
		if req.EndTime != nil {
			end, err := time.Parse(dateOnly, *req.EndTime)
			if err != nil {
				return nil, fmt.Errorf("%w: end_time must be YYYY-MM-DD for all-day events", ErrValidation)
			}
			// Same date as start means single-day: advance end to next day
			startParsed, _ := time.Parse(time.RFC3339, existing.StartTime)
			if !end.After(startParsed) {
				end = startParsed.AddDate(0, 0, 1)
			}
			existing.EndTime = end.UTC().Format(time.RFC3339)
		}
		// If toggling to all-day and no new times provided, normalize existing times
		if req.AllDay != nil && *req.AllDay && req.StartTime == nil {
			start, _ := time.Parse(time.RFC3339, existing.StartTime)
			normalized := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
			existing.StartTime = normalized.Format(time.RFC3339)
			if req.EndTime == nil {
				existing.EndTime = normalized.AddDate(0, 0, 1).Format(time.RFC3339)
			}
		}
	} else {
		if req.StartTime != nil {
			existing.StartTime = *req.StartTime
		}
		if req.EndTime != nil {
			existing.EndTime = *req.EndTime
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

func (s *EventService) CreateOrUpdateOverride(parentID int64, instanceStart string, req *model.UpdateEventRequest) (*model.Event, error) {
	if _, err := time.Parse(time.RFC3339, instanceStart); err != nil {
		return nil, fmt.Errorf("%w: instance_start must be RFC 3339 format", ErrValidation)
	}
	if err := req.Validate(); err != nil {
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

	// Create new override as copy of parent with updates applied
	override := &model.Event{
		Title:              parent.Title,
		Description:        parent.Description,
		StartTime:          instanceStart,
		EndTime:            "", // will be computed
		AllDay:             parent.AllDay,
		Color:              parent.Color,
		Duration:           parent.Duration,
		Categories:         parent.Categories,
		URL:                parent.URL,
		ReminderMinutes:    parent.ReminderMinutes,
		Location:           parent.Location,
		Latitude:           parent.Latitude,
		Longitude:          parent.Longitude,
		RecurrenceParentID:    &parentID,
		RecurrenceOriginalStart: instanceStart,
	}

	// Compute EndTime from parent's duration
	parentStart, _ := time.Parse(time.RFC3339, parent.StartTime)
	parentEnd, _ := time.Parse(time.RFC3339, parent.EndTime)
	dur := parentEnd.Sub(parentStart)
	instStart, _ := time.Parse(time.RFC3339, instanceStart)
	override.EndTime = instStart.Add(dur).Format(time.RFC3339)

	// Apply updates
	if req.Title != nil {
		override.Title = *req.Title
	}
	if req.Description != nil {
		override.Description = sanitize.HTML(*req.Description)
	}
	if req.StartTime != nil {
		override.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		override.EndTime = *req.EndTime
	}
	if req.AllDay != nil {
		override.AllDay = *req.AllDay
	}
	if req.Color != nil {
		override.Color = *req.Color
	}
	if req.Duration != nil {
		override.Duration = *req.Duration
		if *req.Duration != "" {
			d, err := model.ParseDuration(*req.Duration)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid duration: %s", ErrValidation, err.Error())
			}
			s2, _ := time.Parse(time.RFC3339, override.StartTime)
			override.EndTime = s2.Add(d).Format(time.RFC3339)
		}
	}
	if req.Categories != nil {
		override.Categories = *req.Categories
	}
	if req.URL != nil {
		override.URL = *req.URL
	}
	if req.ReminderMinutes != nil {
		override.ReminderMinutes = *req.ReminderMinutes
	}
	if req.Location != nil {
		override.Location = *req.Location
	}
	if req.Latitude != nil {
		override.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		override.Longitude = req.Longitude
	}

	if err := s.repo.Create(override); err != nil {
		return nil, err
	}
	return override, nil
}

func (s *EventService) ImportSingle(events []model.Event) (*model.Event, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("%w: iCal source contains no events", ErrValidation)
	}
	// Filter out overrides for single import
	var parents []model.Event
	for _, e := range events {
		if e.RecurrenceOriginalStart == "" {
			parents = append(parents, e)
		}
	}
	if len(parents) == 0 {
		return nil, fmt.Errorf("%w: iCal source contains no parent events", ErrValidation)
	}
	if len(parents) > 1 {
		return nil, fmt.Errorf("%w: iCal source contains %d events, expected exactly one", ErrValidation, len(parents))
	}

	e := parents[0]
	startTime := e.StartTime
	endTime := e.EndTime
	if e.AllDay {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			startTime = t.Format(dateOnly)
		}
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			endTime = t.Format(dateOnly)
		}
	}
	// Don't pass Duration to CreateEventRequest when EndTime is already computed
	req := &model.CreateEventRequest{
		Title:              e.Title,
		Description:        e.Description,
		StartTime:          startTime,
		EndTime:            endTime,
		AllDay:             e.AllDay,
		Color:              e.Color,
		RecurrenceFreq:     e.RecurrenceFreq,
		RecurrenceCount:    e.RecurrenceCount,
		RecurrenceUntil:    e.RecurrenceUntil,
		RecurrenceInterval: e.RecurrenceInterval,
		RecurrenceByDay:    e.RecurrenceByDay,
		RecurrenceByMonthDay: e.RecurrenceByMonthDay,
		RecurrenceByMonth:  e.RecurrenceByMonth,
		ExDates:            e.ExDates,
		RDates:             e.RDates,
		Categories:         e.Categories,
		URL:                e.URL,
		ReminderMinutes:    e.ReminderMinutes,
		Location:           e.Location,
		Latitude:           e.Latitude,
		Longitude:          e.Longitude,
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	ev := &model.Event{
		Title:              e.Title,
		Description:        e.Description,
		StartTime:          req.StartTime,
		EndTime:            req.EndTime,
		AllDay:             e.AllDay,
		Color:              e.Color,
		RecurrenceFreq:     e.RecurrenceFreq,
		RecurrenceCount:    e.RecurrenceCount,
		RecurrenceUntil:    e.RecurrenceUntil,
		RecurrenceInterval: e.RecurrenceInterval,
		RecurrenceByDay:    e.RecurrenceByDay,
		RecurrenceByMonthDay: e.RecurrenceByMonthDay,
		RecurrenceByMonth:  e.RecurrenceByMonth,
		ExDates:            e.ExDates,
		RDates:             e.RDates,
		Duration:           e.Duration,
		Categories:         e.Categories,
		URL:                e.URL,
		ReminderMinutes:    e.ReminderMinutes,
		Location:           e.Location,
		Latitude:           e.Latitude,
		Longitude:          e.Longitude,
	}
	if err := s.repo.Create(ev); err != nil {
		return nil, err
	}
	return ev, nil
}

func (s *EventService) Import(events []model.Event) (int, error) {
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
		startTime := e.StartTime
		endTime := e.EndTime
		if e.AllDay {
			if t, err := time.Parse(time.RFC3339, startTime); err == nil {
				startTime = t.Format(dateOnly)
			}
			if t, err := time.Parse(time.RFC3339, endTime); err == nil {
				endTime = t.Format(dateOnly)
			}
		}
		// Don't pass Duration to CreateEventRequest when EndTime is already computed
		// (iCal decoder computes EndTime from DURATION). Pass EndTime for validation,
		// and store Duration on the Event directly.
		req := &model.CreateEventRequest{
			Title:              e.Title,
			Description:        e.Description,
			StartTime:          startTime,
			EndTime:            endTime,
			AllDay:             e.AllDay,
			Color:              e.Color,
			RecurrenceFreq:     e.RecurrenceFreq,
			RecurrenceCount:    e.RecurrenceCount,
			RecurrenceUntil:    e.RecurrenceUntil,
			RecurrenceInterval: e.RecurrenceInterval,
			RecurrenceByDay:    e.RecurrenceByDay,
			RecurrenceByMonthDay: e.RecurrenceByMonthDay,
			RecurrenceByMonth:  e.RecurrenceByMonth,
			ExDates:            e.ExDates,
			RDates:             e.RDates,
			Categories:         e.Categories,
			URL:                e.URL,
			ReminderMinutes:    e.ReminderMinutes,
			Location:           e.Location,
			Latitude:           e.Latitude,
			Longitude:          e.Longitude,
		}
		if err := req.Validate(); err != nil {
			continue
		}
		ev := &model.Event{
			Title:              e.Title,
			Description:        e.Description,
			StartTime:          req.StartTime,
			EndTime:            req.EndTime,
			AllDay:             e.AllDay,
			Color:              e.Color,
			RecurrenceFreq:     e.RecurrenceFreq,
			RecurrenceCount:    e.RecurrenceCount,
			RecurrenceUntil:    e.RecurrenceUntil,
			RecurrenceInterval: e.RecurrenceInterval,
			RecurrenceByDay:    e.RecurrenceByDay,
			RecurrenceByMonthDay: e.RecurrenceByMonthDay,
			RecurrenceByMonth:  e.RecurrenceByMonth,
			ExDates:            e.ExDates,
			RDates:             e.RDates,
			Duration:           e.Duration,
			Categories:         e.Categories,
			URL:                e.URL,
			ReminderMinutes:    e.ReminderMinutes,
			Location:           e.Location,
			Latitude:           e.Latitude,
			Longitude:          e.Longitude,
		}
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
			Title:              e.Title,
			Description:        sanitize.HTML(e.Description),
			StartTime:          e.StartTime,
			EndTime:            e.EndTime,
			AllDay:             e.AllDay,
			Color:              e.Color,
			Duration:           e.Duration,
			Categories:         e.Categories,
			URL:                e.URL,
			ReminderMinutes:    e.ReminderMinutes,
			Location:           e.Location,
			Latitude:           e.Latitude,
			Longitude:          e.Longitude,
			RecurrenceParentID:    &parentID,
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

func (s *EventService) Delete(id int64) error {
	// Also delete all overrides for this parent
	_ = s.repo.DeleteByParentID(id)

	err := s.repo.Delete(id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
