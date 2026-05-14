package repository

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mikaelstaldal/mycal/internal/model"
	_ "modernc.org/sqlite"
)

func setupTestRepo(t *testing.T) *SQLiteRepository {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	repo, err := NewSQLiteRepository(db)
	require.NoError(t, err)
	return repo
}

func createTestEvent(t *testing.T, repo *SQLiteRepository, title, desc, start, end string) *model.Event {
	t.Helper()
	e := &model.Event{
		Title:       title,
		Description: desc,
		StartTime:   start,
		EndTime:     end,
	}
	err := repo.Create(e)
	require.NoError(t, err)
	return e
}

func TestSearchByTitle(t *testing.T) {
	repo := setupTestRepo(t)
	createTestEvent(t, repo, "Team Meeting", "Weekly sync", "2026-02-18T10:00:00Z", "2026-02-18T11:00:00Z")
	createTestEvent(t, repo, "Lunch Break", "Cafeteria", "2026-02-18T12:00:00Z", "2026-02-18T13:00:00Z")

	results, err := repo.Search("meeting", "", "", nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Team Meeting", results[0].Title)
}

func TestSearchByDescription(t *testing.T) {
	repo := setupTestRepo(t)
	createTestEvent(t, repo, "Event A", "Important discussion about budgets", "2026-02-18T10:00:00Z", "2026-02-18T11:00:00Z")
	createTestEvent(t, repo, "Event B", "Casual chat", "2026-02-18T12:00:00Z", "2026-02-18T13:00:00Z")

	results, err := repo.Search("budgets", "", "", nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Event A", results[0].Title)
}

func TestSearchWithTimeRange(t *testing.T) {
	repo := setupTestRepo(t)
	createTestEvent(t, repo, "Morning Meeting", "Standup", "2026-02-18T09:00:00Z", "2026-02-18T10:00:00Z")
	createTestEvent(t, repo, "Afternoon Meeting", "Review", "2026-02-18T15:00:00Z", "2026-02-18T16:00:00Z")

	results, err := repo.Search("meeting", "2026-02-18T14:00:00Z", "2026-02-18T17:00:00Z", nil)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Afternoon Meeting", results[0].Title)
}

func TestSearchAfterUpdate(t *testing.T) {
	repo := setupTestRepo(t)
	e := createTestEvent(t, repo, "Old Title", "Description", "2026-02-18T10:00:00Z", "2026-02-18T11:00:00Z")

	e.Title = "New Title"
	err := repo.Update(e)
	require.NoError(t, err)

	results, err := repo.Search("Old", "", "", nil)
	require.NoError(t, err)
	assert.Empty(t, results)

	results, err = repo.Search("New", "", "", nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestSearchAfterDelete(t *testing.T) {
	repo := setupTestRepo(t)
	e := createTestEvent(t, repo, "Deletable Event", "Will be removed", "2026-02-18T10:00:00Z", "2026-02-18T11:00:00Z")

	err := repo.Delete(e.ID)
	require.NoError(t, err)

	results, err := repo.Search("Deletable", "", "", nil)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchSpecialCharacters(t *testing.T) {
	repo := setupTestRepo(t)
	createTestEvent(t, repo, `Event with "quotes"`, "Has special chars: AND OR NOT", "2026-02-18T10:00:00Z", "2026-02-18T11:00:00Z")

	results, err := repo.Search(`"quotes"`, "", "", nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// FTS5 operators should be safely quoted
	results, err = repo.Search("AND OR NOT", "", "", nil)
	require.NoError(t, err)
	// Should not error even with FTS5 operator-like terms
}

func TestSearchEmptyQuery(t *testing.T) {
	repo := setupTestRepo(t)
	createTestEvent(t, repo, "Some Event", "Description", "2026-02-18T10:00:00Z", "2026-02-18T11:00:00Z")

	results, err := repo.Search("", "", "", nil)
	require.NoError(t, err)
	assert.Nil(t, results)
}
