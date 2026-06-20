package repository

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrateLegacyDatabase verifies that a pre-user_version database (with the
// obsolete calendar_name columns and defaultEventColor preference) is reconciled
// to the v1 schema without losing data.
func TestMigrateLegacyDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.sqlite")

	// Build an old-style schema by hand: calendar_name present, calendar_id and
	// ics_uid absent, user_version left at 0.
	raw, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = raw.Exec(`
		CREATE TABLE events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			all_day INTEGER NOT NULL DEFAULT 0,
			color TEXT NOT NULL DEFAULT '',
			recurrence_freq TEXT NOT NULL DEFAULT '',
			recurrence_count INTEGER NOT NULL DEFAULT 0,
			recurrence_until TEXT NOT NULL DEFAULT '',
			recurrence_interval INTEGER NOT NULL DEFAULT 0,
			recurrence_by_day TEXT NOT NULL DEFAULT '',
			recurrence_by_monthday TEXT NOT NULL DEFAULT '',
			recurrence_by_month TEXT NOT NULL DEFAULT '',
			exdates TEXT NOT NULL DEFAULT '',
			rdates TEXT NOT NULL DEFAULT '',
			recurrence_parent_id INTEGER,
			recurrence_original_start TEXT NOT NULL DEFAULT '',
			duration TEXT NOT NULL DEFAULT '',
			categories TEXT NOT NULL DEFAULT '',
			url TEXT NOT NULL DEFAULT '',
			reminder_minutes INTEGER NOT NULL DEFAULT 0,
			location TEXT NOT NULL DEFAULT '',
			latitude REAL,
			longitude REAL,
			calendar_name TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		);
		CREATE TABLE preferences (key TEXT PRIMARY KEY, value TEXT NOT NULL DEFAULT '');
		CREATE TABLE feeds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT NOT NULL,
			calendar_name TEXT NOT NULL DEFAULT '',
			refresh_interval_minutes INTEGER NOT NULL DEFAULT 60,
			last_refreshed_at TEXT NOT NULL DEFAULT '',
			last_error TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		);
		INSERT INTO events (title, start_time, end_time, calendar_name) VALUES ('Work meeting', '2026-03-15T10:00:00Z', '2026-03-15T11:00:00Z', 'Work');
		INSERT INTO feeds (url, calendar_name) VALUES ('https://example.com/cal.ics', 'Work');
		INSERT INTO preferences (key, value) VALUES ('defaultEventColor', 'tomato');
	`)
	require.NoError(t, err)
	require.NoError(t, raw.Close())

	// Re-open through OpenDB to run the migration.
	db, err := OpenDB(dbPath, 5000)
	require.NoError(t, err)
	defer db.Close()

	var version int
	require.NoError(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	assert.Equal(t, 1, version, "should be stamped at v1")

	// calendar_name dropped from both tables.
	assert.False(t, columnExists(db, "events", "calendar_name"))
	assert.False(t, columnExists(db, "feeds", "calendar_name"))
	assert.True(t, columnExists(db, "events", "ics_uid"))
	assert.True(t, columnExists(db, "events", "calendar_id"))
	assert.True(t, columnExists(db, "feeds", "calendar_id"))

	// The "Work" calendar was created and the event points at it.
	var calID int64
	require.NoError(t, db.QueryRow("SELECT id FROM calendars WHERE name = 'Work'").Scan(&calID))
	assert.NotZero(t, calID)

	repo, err := NewSQLiteRepository(db)
	require.NoError(t, err)
	events, err := repo.List("2026-03-01T00:00:00Z", "2026-04-01T00:00:00Z", nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "Work meeting", events[0].Title)
	assert.Equal(t, calID, events[0].CalendarID)
	assert.Equal(t, "Work", events[0].CalendarName)

	// defaultEventColor migrated to the default calendar and removed.
	var defColor string
	require.NoError(t, db.QueryRow("SELECT color FROM calendars WHERE id = 0").Scan(&defColor))
	assert.Equal(t, "tomato", defColor)
	var prefCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM preferences WHERE key = 'defaultEventColor'").Scan(&prefCount))
	assert.Zero(t, prefCount)
}

// TestFreshDatabaseIsVersioned verifies a brand-new database lands at v1.
func TestFreshDatabaseIsVersioned(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "fresh.sqlite"), 5000)
	require.NoError(t, err)
	defer db.Close()

	var version int
	require.NoError(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	assert.Equal(t, 1, version)

	// WAL mode is active on a file-backed database.
	var mode string
	require.NoError(t, db.QueryRow("PRAGMA journal_mode").Scan(&mode))
	assert.Equal(t, "wal", mode)
}
