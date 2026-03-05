package service

import (
	"fmt"

	"github.com/mikaelstaldal/mycal/internal/repository"
)

var allowedPreferences = map[string]string{
	"defaultEventColor": "dodgerblue",
}

type PreferencesService struct {
	repo repository.PreferencesRepository
}

func NewPreferencesService(repo repository.PreferencesRepository) *PreferencesService {
	return &PreferencesService{repo: repo}
}

func (s *PreferencesService) GetAll() (map[string]string, error) {
	stored, err := s.repo.GetAllPreferences()
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(allowedPreferences))
	for k, def := range allowedPreferences {
		if v, ok := stored[k]; ok {
			result[k] = v
		} else {
			result[k] = def
		}
	}
	return result, nil
}

func (s *PreferencesService) Update(prefs map[string]string) (map[string]string, error) {
	for k, v := range prefs {
		if _, ok := allowedPreferences[k]; !ok {
			return nil, fmt.Errorf("%w: unknown preference key: %s", ErrValidation, k)
		}
		if err := s.repo.SetPreference(k, v); err != nil {
			return nil, err
		}
		_ = v
	}
	return s.GetAll()
}
