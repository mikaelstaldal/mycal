package repository

import "github.com/mikaelstaldal/mycal/internal/model"

// ScopedRepository restricts every event and calendar read to the token grants.
// The embedded repository must use the same snapshot as token validation.
type ScopedRepository struct {
	*SQLiteRepository
	Allowed map[int64]bool
}

func (r *ScopedRepository) ids(requested []int64) []int64 {
	ids := []int64{}
	if requested == nil {
		for id := range r.Allowed {
			ids = append(ids, id)
		}
	} else {
		for _, id := range requested {
			if r.Allowed[id] {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// visible also checks the parent so a misfiled override cannot cross token scopes.
func (r *ScopedRepository) visible(e *model.Event) (bool, error) {
	if !r.Allowed[e.CalendarID] {
		return false, nil
	}
	if e.RecurrenceParentID != nil {
		parent, err := r.SQLiteRepository.GetByID(*e.RecurrenceParentID)
		if err != nil {
			return false, err
		}
		return parent != nil && r.Allowed[parent.CalendarID], nil
	}
	return true, nil
}
func (r *ScopedRepository) events(events []model.Event, err error) ([]model.Event, error) {
	if err != nil {
		return nil, err
	}
	result := []model.Event{}
	for _, e := range events {
		ok, err := r.visible(&e)
		if err != nil {
			return nil, err
		}
		if ok {
			result = append(result, e)
		}
	}
	return result, nil
}
func (r *ScopedRepository) event(e *model.Event, err error) (*model.Event, error) {
	if err != nil || e == nil {
		return nil, err
	}
	ok, err := r.visible(e)
	if !ok || err != nil {
		return nil, err
	}
	return e, nil
}
func (r *ScopedRepository) List(from, to string, ids []int64) ([]model.Event, error) {
	return r.events(r.SQLiteRepository.List(from, to, r.ids(ids)))
}
func (r *ScopedRepository) ListAll(ids []int64) ([]model.Event, error) {
	return r.events(r.SQLiteRepository.ListAll(r.ids(ids)))
}
func (r *ScopedRepository) ListRecurring(to string, ids []int64) ([]model.Event, error) {
	return r.events(r.SQLiteRepository.ListRecurring(to, r.ids(ids)))
}
func (r *ScopedRepository) Search(q, from, to string, ids []int64) ([]model.Event, error) {
	return r.events(r.SQLiteRepository.Search(q, from, to, r.ids(ids)))
}
func (r *ScopedRepository) GetByID(id int64) (*model.Event, error) {
	return r.event(r.SQLiteRepository.GetByID(id))
}
func (r *ScopedRepository) GetOverride(id int64, start string) (*model.Event, error) {
	return r.event(r.SQLiteRepository.GetOverride(id, start))
}
func (r *ScopedRepository) ListOverrides(ids []int64, from, to string) ([]model.Event, error) {
	return r.events(r.SQLiteRepository.ListOverrides(ids, from, to))
}
func (r *ScopedRepository) ListCalendars() ([]model.Calendar, error) {
	calendars, err := r.SQLiteRepository.ListCalendars()
	result := []model.Calendar{}
	for _, c := range calendars {
		if r.Allowed[c.ID] {
			result = append(result, c)
		}
	}
	return result, err
}
func (r *ScopedRepository) GetCalendarByID(id int64) (*model.Calendar, error) {
	if !r.Allowed[id] {
		return nil, nil
	}
	return r.SQLiteRepository.GetCalendarByID(id)
}
func (r *ScopedRepository) GetCalendarByName(name string) (*model.Calendar, error) {
	c, err := r.SQLiteRepository.GetCalendarByName(name)
	if c != nil && !r.Allowed[c.ID] {
		return nil, err
	}
	return c, err
}
