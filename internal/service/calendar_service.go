package service

import (
	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/repository"
)

type CalendarService struct {
	repo repository.CalendarRepository
}

func NewCalendarService(repo repository.CalendarRepository) *CalendarService {
	return &CalendarService{repo: repo}
}

func (s *CalendarService) List() ([]model.Calendar, error) {
	calendars, err := s.repo.ListCalendars()
	if err != nil {
		return nil, err
	}
	if calendars == nil {
		calendars = []model.Calendar{}
	}
	return calendars, nil
}

func (s *CalendarService) Update(id int64, name, color string) (*model.Calendar, error) {
	cal, err := s.repo.GetCalendarByID(id)
	if err != nil {
		return nil, err
	}
	if cal == nil {
		return nil, ErrNotFound
	}
	if name != "" {
		cal.Name = name
	}
	if color != "" {
		cal.Color = color
	}
	if err := s.repo.UpdateCalendar(cal); err != nil {
		return nil, err
	}
	return cal, nil
}

// GetOrCreateByName looks up a calendar by name, creates it if missing, and returns its ID.
func (s *CalendarService) GetOrCreateByName(name string) (int64, error) {
	if name == "" {
		return 0, nil // default calendar
	}
	cal, err := s.repo.GetCalendarByName(name)
	if err != nil {
		return 0, err
	}
	if cal != nil {
		return cal.ID, nil
	}
	newCal := &model.Calendar{
		Name:  name,
		Color: "dodgerblue",
	}
	if err := s.repo.CreateCalendar(newCal); err != nil {
		return 0, err
	}
	return newCal.ID, nil
}
