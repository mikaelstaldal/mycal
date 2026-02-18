package service

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/repository"
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
	return events, nil
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
		Title:       req.Title,
		Description: req.Description,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		AllDay:      req.AllDay,
		Color:       req.Color,
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
		existing.Description = *req.Description
	}
	if req.AllDay != nil {
		existing.AllDay = *req.AllDay
	}
	if req.Color != nil {
		existing.Color = *req.Color
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

func (s *EventService) Import(events []model.Event) (int, error) {
	imported := 0
	for _, e := range events {
		req := &model.CreateEventRequest{
			Title:       e.Title,
			Description: e.Description,
			StartTime:   e.StartTime,
			EndTime:     e.EndTime,
			AllDay:      e.AllDay,
		}
		if err := req.Validate(); err != nil {
			continue
		}
		ev := &model.Event{
			Title:       e.Title,
			Description: e.Description,
			StartTime:   req.StartTime,
			EndTime:     req.EndTime,
			AllDay:      e.AllDay,
		}
		if err := s.repo.Create(ev); err != nil {
			continue
		}
		imported++
	}
	return imported, nil
}

func (s *EventService) Delete(id int64) error {
	err := s.repo.Delete(id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
