package repository

import "github.com/mikaelstaldal/mycal/internal/model"

type EventRepository interface {
	List(from, to string) ([]model.Event, error)
	ListAll() ([]model.Event, error)
	ListRecurring(to string) ([]model.Event, error)
	Search(query, from, to string) ([]model.Event, error)
	GetByID(id int64) (*model.Event, error)
	Create(event *model.Event) error
	Update(event *model.Event) error
	Delete(id int64) error
	ListOverrides(parentIDs []int64) ([]model.Event, error)
	GetOverride(parentID int64, originalStart string) (*model.Event, error)
	DeleteByParentID(parentID int64) error
}
