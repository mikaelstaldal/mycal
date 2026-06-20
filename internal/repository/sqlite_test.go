package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mikaelstaldal/mycal/internal/model"
)

func newTestRepo(t *testing.T) *SQLiteRepository {
	t.Helper()
	db, err := OpenDB(":memory:", 0)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repo, err := NewSQLiteRepository(db)
	require.NoError(t, err)
	return repo
}

func TestCreateAndGetByID(t *testing.T) {
	repo := newTestRepo(t)
	e := &model.Event{
		Title:     "Test Event",
		StartTime: "2026-03-15T10:00:00Z",
		EndTime:   "2026-03-15T11:00:00Z",
	}
	err := repo.Create(e)
	require.NoError(t, err)
	assert.NotZero(t, e.ID)
	assert.NotEmpty(t, e.CreatedAt)

	got, err := repo.GetByID(e.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Event", got.Title)
}

func TestGetByIDNotFound(t *testing.T) {
	repo := newTestRepo(t)
	got, err := repo.GetByID(999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestList(t *testing.T) {
	repo := newTestRepo(t)
	events := []model.Event{
		{Title: "Jan Event", StartTime: "2026-01-10T10:00:00Z", EndTime: "2026-01-10T11:00:00Z"},
		{Title: "Feb Event", StartTime: "2026-02-15T10:00:00Z", EndTime: "2026-02-15T11:00:00Z"},
		{Title: "Mar Event", StartTime: "2026-03-20T10:00:00Z", EndTime: "2026-03-20T11:00:00Z"},
	}
	for i := range events {
		err := repo.Create(&events[i])
		require.NoError(t, err)
	}

	got, err := repo.List("2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z", nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Feb Event", got[0].Title)
}

func TestUpdate(t *testing.T) {
	repo := newTestRepo(t)
	e := &model.Event{
		Title:     "Original",
		StartTime: "2026-03-15T10:00:00Z",
		EndTime:   "2026-03-15T11:00:00Z",
	}
	err := repo.Create(e)
	require.NoError(t, err)
	e.Title = "Updated"
	err = repo.Update(e)
	require.NoError(t, err)
	got, _ := repo.GetByID(e.ID)
	assert.Equal(t, "Updated", got.Title)
}

func TestDelete(t *testing.T) {
	repo := newTestRepo(t)
	e := &model.Event{
		Title:     "To Delete",
		StartTime: "2026-03-15T10:00:00Z",
		EndTime:   "2026-03-15T11:00:00Z",
	}
	err := repo.Create(e)
	require.NoError(t, err)
	err = repo.Delete(e.ID)
	require.NoError(t, err)
	got, _ := repo.GetByID(e.ID)
	assert.Nil(t, got)
}

func TestDeleteNotFound(t *testing.T) {
	repo := newTestRepo(t)
	err := repo.Delete(999)
	assert.Error(t, err)
}

func TestCreateAndGetWithLocation(t *testing.T) {
	repo := newTestRepo(t)
	lat := 59.3293
	lon := 18.0686
	e := &model.Event{
		Title:     "Located Event",
		StartTime: "2026-03-15T10:00:00Z",
		EndTime:   "2026-03-15T11:00:00Z",
		Location:  "Stockholm Office",
		Latitude:  &lat,
		Longitude: &lon,
	}
	err := repo.Create(e)
	require.NoError(t, err)

	got, err := repo.GetByID(e.ID)
	require.NoError(t, err)
	assert.Equal(t, "Stockholm Office", got.Location)
	require.NotNil(t, got.Latitude)
	assert.Equal(t, 59.3293, *got.Latitude)
	require.NotNil(t, got.Longitude)
	assert.Equal(t, 18.0686, *got.Longitude)
}

// --- Preferences tests ---

func TestSetAndGetPreference(t *testing.T) {
	repo := newTestRepo(t)
	err := repo.SetPreference("defaultEventColor", "red")
	require.NoError(t, err)
	val, ok, err := repo.GetPreference("defaultEventColor")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "red", val)
}

func TestGetPreferenceNotFound(t *testing.T) {
	repo := newTestRepo(t)
	_, ok, err := repo.GetPreference("nonexistent")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestGetAllPreferences(t *testing.T) {
	repo := newTestRepo(t)
	err := repo.SetPreference("a", "1")
	require.NoError(t, err)
	err = repo.SetPreference("b", "2")
	require.NoError(t, err)
	prefs, err := repo.GetAllPreferences()
	require.NoError(t, err)
	assert.Len(t, prefs, 2)
	assert.Equal(t, "1", prefs["a"])
	assert.Equal(t, "2", prefs["b"])
}

func TestUpsertPreference(t *testing.T) {
	repo := newTestRepo(t)
	err := repo.SetPreference("key", "v1")
	require.NoError(t, err)
	err = repo.SetPreference("key", "v2")
	require.NoError(t, err)
	val, _, _ := repo.GetPreference("key")
	assert.Equal(t, "v2", val)
}

func TestDeletePreference(t *testing.T) {
	repo := newTestRepo(t)
	err := repo.SetPreference("key", "val")
	require.NoError(t, err)
	err = repo.DeletePreference("key")
	require.NoError(t, err)
	_, ok, _ := repo.GetPreference("key")
	assert.False(t, ok)
}

func TestCreateWithoutCoordinates(t *testing.T) {
	repo := newTestRepo(t)
	e := &model.Event{
		Title:     "No Coords Event",
		StartTime: "2026-03-15T10:00:00Z",
		EndTime:   "2026-03-15T11:00:00Z",
		Location:  "Somewhere",
	}
	err := repo.Create(e)
	require.NoError(t, err)

	got, err := repo.GetByID(e.ID)
	require.NoError(t, err)
	assert.Equal(t, "Somewhere", got.Location)
	assert.Nil(t, got.Latitude)
	assert.Nil(t, got.Longitude)
}
