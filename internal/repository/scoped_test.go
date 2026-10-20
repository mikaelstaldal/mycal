package repository

import (
	"testing"

	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopedRepositoryOverrideRequiresParentCalendar(t *testing.T) {
	repo := setupTestRepo(t)
	_, err := repo.db.Exec("INSERT INTO calendars (id, name) VALUES (7, 'Private')")
	require.NoError(t, err)
	parent := &model.Event{Title: "Secret", CalendarID: 7, StartTime: "2026-01-05T10:00:00Z", EndTime: "2026-01-05T11:00:00Z", RecurrenceFreq: "WEEKLY"}
	require.NoError(t, repo.Create(parent))
	override := &model.Event{Title: "Secret edited", StartTime: "2026-01-12T10:00:00Z", EndTime: "2026-01-12T11:00:00Z", RecurrenceParentID: &parent.ID, RecurrenceOriginalStart: "2026-01-12T10:00:00Z"}
	require.NoError(t, repo.Create(override))
	for _, allowed := range []map[int64]bool{{0: true}, {0: true, 7: true}} {
		scoped := &ScopedRepository{SQLiteRepository: repo, Allowed: allowed}
		want := 0
		if allowed[7] {
			want = 1
		}
		for name, read := range map[string]func() ([]model.Event, error){
			"list":   func() ([]model.Event, error) { return scoped.List("2026-01-12T00:00:00Z", "2026-01-13T00:00:00Z", nil) },
			"feed":   func() ([]model.Event, error) { return scoped.ListAll([]int64{0}) },
			"search": func() ([]model.Event, error) { return scoped.Search("edited", "", "", nil) },
			"overrides": func() ([]model.Event, error) {
				return scoped.ListOverrides([]int64{parent.ID}, "2026-01-12T00:00:00Z", "2026-01-13T00:00:00Z")
			},
		} {
			events, err := read()
			require.NoError(t, err, name)
			expected := want
			if name == "list" {
				expected = 0
			} // List excludes overrides; ListOverrides supplies them.
			assert.Len(t, events, expected, name)
		}
		event, err := scoped.GetByID(override.ID)
		require.NoError(t, err)
		assert.Equal(t, want == 0, event == nil)
		event, err = scoped.GetOverride(parent.ID, override.RecurrenceOriginalStart)
		require.NoError(t, err)
		assert.Equal(t, want == 0, event == nil)
	}
}

func TestMigrationRepairsOverrideCalendar(t *testing.T) {
	repo := setupTestRepo(t)
	_, err := repo.db.Exec("INSERT INTO calendars (id, name) VALUES (7, 'Private')")
	require.NoError(t, err)
	parent := &model.Event{Title: "Secret", CalendarID: 7, StartTime: "2026-01-05T10:00:00Z", EndTime: "2026-01-05T11:00:00Z"}
	require.NoError(t, repo.Create(parent))
	override := &model.Event{Title: "Edited", RecurrenceParentID: &parent.ID, StartTime: parent.StartTime, EndTime: parent.EndTime}
	require.NoError(t, repo.Create(override))
	// Simulate the previous schema with its incorrectly filed override.
	_, err = repo.db.Exec("DROP INDEX idx_events_feed_id; ALTER TABLE events DROP COLUMN feed_id; PRAGMA user_version = 4")
	require.NoError(t, err)
	db, err := OpenDB(MemoryDSN(t.Name()), 0)
	require.NoError(t, err)
	defer db.Close()
	repaired, err := repo.GetByID(override.ID)
	require.NoError(t, err)
	assert.Equal(t, parent.CalendarID, repaired.CalendarID)
	untouched, err := repo.GetByID(parent.ID)
	require.NoError(t, err)
	assert.Equal(t, parent.CalendarID, untouched.CalendarID)
}
