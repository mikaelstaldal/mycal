package repository

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikaelstaldal/go-server-common/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLegacyDatabaseIsRefused builds a pre-user_version database — the shape the
// deleted reconciliation code existed to repair — and asserts that it is now
// refused rather than migrated, and that the refusal migrates nothing. It does
// not leave the file untouched: sqlite.Open converts any database still at
// user_version 0 to WAL before the guard can look, which is asserted below.
//
// The fixture is kept from the migration test this replaces: it is the only
// artefact in the repository describing what a legacy mycal database looked like,
// and it is a real shape rather than an invented one, the pre-versioning build
// having added events.calendar_name itself before later dropping it.
func TestLegacyDatabaseIsRefused(t *testing.T) {
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

	_, err = OpenDB(dbPath, 5000)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLegacySchema)
	assert.Contains(t, err.Error(), "older mycal build",
		"the refusal has to tell the operator what to do about it")

	// Nothing was migrated: the database is still in its legacy shape. Checked
	// through a fresh handle, since OpenDB closed the one it failed on.
	raw, err = sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer raw.Close()

	var version int
	require.NoError(t, raw.QueryRow("PRAGMA user_version").Scan(&version))
	assert.Zero(t, version, "a refused database must not be stamped")

	var n int
	require.NoError(t, raw.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('events') WHERE name = 'calendar_name'`).Scan(&n))
	assert.Equal(t, 1, n, "calendar_name must still be there — nothing reconciled it")

	require.NoError(t, raw.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('events') WHERE name = 'note_slug'`).Scan(&n))
	assert.Zero(t, n, "no migration may have run")

	require.NoError(t, raw.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'calendars'`).Scan(&n))
	assert.Zero(t, n, "migration 0 must not have created anything")

	// The one thing a refusal does change. sqlite.Open sets WAL mode on any
	// database still at user_version 0 before returning, so the journal mode has
	// already moved by the time the guard can look. It is a property of the file
	// rather than of its contents, an older mycal reads a WAL database fine, and
	// avoiding it would mean not using the shared Open at all.
	var mode string
	require.NoError(t, raw.QueryRow("PRAGMA journal_mode").Scan(&mode))
	assert.Equal(t, "wal", mode)
}

// TestFreshDatabaseIsVersioned verifies a brand-new database lands at the
// current schema version, carrying every column added by later migrations.
func TestFreshDatabaseIsVersioned(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "fresh.sqlite"), 5000)
	require.NoError(t, err)
	defer db.Close()

	var version int
	require.NoError(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	assert.Equal(t, currentSchemaVersion, version)
	assert.True(t, columnExists(db, "events", "note_slug"))

	// WAL mode is active on a file-backed database.
	var mode string
	require.NoError(t, db.QueryRow("PRAGMA journal_mode").Scan(&mode))
	assert.Equal(t, "wal", mode)

	// Every object migration 0 is responsible for. schemaV1Indexes moved from a
	// second statement list into the same slice as schemaV1, and a statement lost
	// in that move would otherwise only show up as a slow query much later.
	for _, name := range []string{
		"idx_events_start_time", "idx_events_end_time", "idx_events_recurrence_parent_id",
		"idx_events_time_range", "idx_events_ics_uid", "idx_events_calendar_id",
		"idx_feeds_calendar_id",
	} {
		var n int
		require.NoError(t, db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, name).Scan(&n))
		assert.Equal(t, 1, n, "index %s should exist", name)
	}
	for _, name := range []string{"events_ai", "events_ad", "events_au"} {
		var n int
		require.NoError(t, db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'trigger' AND name = ?`, name).Scan(&n))
		assert.Equal(t, 1, n, "trigger %s should exist", name)
	}
	for _, name := range []string{"events", "events_fts", "feeds", "calendars", "preferences"} {
		assert.True(t, tableExists(db, name), "table %s should exist", name)
	}

	// The default calendar migration 0 seeds.
	var calName string
	require.NoError(t, db.QueryRow(`SELECT name FROM calendars WHERE id = 0`).Scan(&calName))
	assert.Equal(t, "Default", calName)
}

// TestFreshInstallIsNotRefused covers the legacy guard's false-positive case: a
// real first start creates the file before anything reads user_version, so the
// guard sees an existing, empty database and must let it through. Getting this
// wrong would break every new installation while leaving every existing one fine.
func TestFreshInstallIsNotRefused(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fresh.sqlite")

	// An existing but empty file, as sql.Open leaves one.
	raw, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	require.NoError(t, raw.Ping())
	require.NoError(t, raw.Close())

	db, err := OpenDB(dbPath, 5000)
	require.NoError(t, err, "an empty database is not a legacy database")
	defer db.Close()

	var version int
	require.NoError(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	assert.Equal(t, currentSchemaVersion, version)
}

// TestReopenAppliesNothing checks that opening an already-migrated database takes
// no write lock. MigrateStrict returns before starting a transaction once
// user_version equals len(migrations), which is what keeps OpenDB usable against a
// database another process is writing.
func TestReopenAppliesNothing(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "reopen.sqlite")

	db, err := OpenDB(dbPath, 5000)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	before := schemaFingerprint(t, dbPath)

	db2, err := OpenDB(dbPath, 5000)
	require.NoError(t, err)
	defer db2.Close()

	var version int
	require.NoError(t, db2.QueryRow("PRAGMA user_version").Scan(&version))
	assert.Equal(t, currentSchemaVersion, version)
	assert.Equal(t, before, schemaFingerprint(t, dbPath), "reopening must not alter the schema")
}

// schemaFingerprint is every object in sqlite_master, in name order.
func schemaFingerprint(t *testing.T, dbPath string) string {
	t.Helper()
	raw, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer raw.Close()
	rows, err := raw.Query(`SELECT type, name, COALESCE(sql, '') FROM sqlite_master ORDER BY name`)
	require.NoError(t, err)
	defer rows.Close()
	var sb strings.Builder
	for rows.Next() {
		var typ, name, ddl string
		require.NoError(t, rows.Scan(&typ, &name, &ddl))
		sb.WriteString(typ + "\x00" + name + "\x00" + ddl + "\n")
	}
	require.NoError(t, rows.Err())
	return sb.String()
}

// TestSchemaTooNewIsRefused verifies that a database stamped with a user_version
// this binary does not know is refused rather than operated on.
func TestSchemaTooNewIsRefused(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "future.sqlite")

	db, err := OpenDB(dbPath, 5000)
	require.NoError(t, err)
	_, err = db.Exec(fmt.Sprintf("PRAGMA user_version = %d", currentSchemaVersion+1))
	require.NoError(t, err)
	require.NoError(t, db.Close())

	_, err = OpenDB(dbPath, 5000)
	require.Error(t, err)
	assert.ErrorIs(t, err, sqlite.ErrSchemaTooNew)
}
