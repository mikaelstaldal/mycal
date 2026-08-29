package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/mikaelstaldal/go-server-common/sqlite"
)

// OpenDB opens the SQLite database at path, enables foreign keys, sets the
// busy_timeout pragma (0 = skip), applies any extraPragmas, and runs pending
// schema migrations. Connection setup (DSN, pragmas, WAL mode) is delegated to
// the shared sqlite package; the imperative v1 schema migration — which
// reconciles pre-user_version legacy databases and so cannot be expressed as
// the flat statement list sqlite.Migrate takes — is applied by initSchema.
func OpenDB(path string, busyTimeout int, extraPragmas ...string) (*sql.DB, error) {
	// Passing no migrations leaves migration to initSchema while still letting
	// the shared package build the DSN, bake in pragmas, and enable WAL mode.
	db, err := sqlite.Open(path, busyTimeout, nil, extraPragmas...)
	if err != nil {
		return nil, err
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// MemoryDSN returns a DSN for an in-memory database that stays correct when more
// than one connection is used. Tests pass t.Name() to get a database of their own.
//
// A bare ":memory:" database is private to the connection that opened it, so a
// second connection from the same *sql.DB gets a second, empty database. That is
// invisible for as long as a caller happens to reuse one pooled connection and
// surfaces as a confusing "no such table" the moment it does not — and neither
// the number of connections nor when they are taken is under the caller's
// control: sqlite.Open's WAL step and sqlite.MigrateStrict both acquire
// connections of their own. cache=shared makes every connection see the same
// database.
//
// A shared-cache database is identified by its name, so distinct names must stay
// distinct: two callers mapped onto one name would share a database, which is the
// problem this function exists to avoid rather than a milder version of it. Every
// byte outside [A-Za-z0-9] is therefore escaped as _<hex>_ rather than replaced.
// That is injective over any string: an unescaped run can never contain '_', so an
// escape cannot be confused with the text around it, and no two names can meet.
//
// Bytes rather than runes, deliberately. Ranging over a string decodes invalid
// UTF-8 to utf8.RuneError, which would map every such byte onto one escape and
// collide names that differ — and this takes a string, not a valid-UTF-8 string.
// Escaping the bytes costs nothing and removes the exception.
func MemoryDSN(name string) string {
	var b strings.Builder
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "_%02x_", c)
		}
	}
	return "file:" + b.String() + "?mode=memory&cache=shared"
}

// currentSchemaVersion is the PRAGMA user_version a fully migrated database
// carries. Bump it when adding a migration block to initSchema.
const currentSchemaVersion = 2

// execQuerier is satisfied by both *sql.DB and *sql.Tx so migration helpers can
// run against either.
type execQuerier interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

// initSchema applies pending schema migrations using PRAGMA user_version.
// Each if-block is independent so multiple migrations can apply in one startup.
// When the database is already at the latest version no statements run, so it is
// safe to call against a read-only connection.
//
// A database stamped past currentSchemaVersion was written by a newer mycal, so
// this binary does not know the shape it would be writing to: it is refused with
// sqlite.ErrSchemaTooNew rather than operated on.
func initSchema(db *sql.DB) error {
	version, err := sqlite.UserVersion(db)
	if err != nil {
		return err
	}
	if version > currentSchemaVersion {
		return fmt.Errorf("%w: database is at schema version %d, this binary knows %d",
			sqlite.ErrSchemaTooNew, version, currentSchemaVersion)
	}

	if version < 1 {
		// A pre-existing events table means this database was created before the
		// user_version scheme; it may carry legacy columns that need reconciling.
		legacy := tableExists(db, "events")

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration to v1: %w", err)
		}
		defer tx.Rollback()

		for _, stmt := range schemaV1 {
			if _, err := tx.Exec(stmt); err != nil {
				preview := stmt
				if len(preview) > 60 {
					preview = preview[:60]
				}
				return fmt.Errorf("schema v1 %q: %w", preview, err)
			}
		}

		// Sync the FTS index with the existing events before reconcileLegacy runs
		// any UPDATE: the update triggers delete-then-insert each touched row in
		// events_fts, which corrupts the index if the row was never indexed. A
		// no-op on a fresh, empty database.
		if _, err := tx.Exec(`INSERT INTO events_fts(events_fts) VALUES('rebuild')`); err != nil {
			return fmt.Errorf("rebuild events_fts: %w", err)
		}

		if legacy {
			if err := reconcileLegacy(tx); err != nil {
				return fmt.Errorf("reconcile legacy schema: %w", err)
			}
		}

		// Indexes on ics_uid/calendar_id must come after reconcileLegacy, which
		// adds those columns to legacy tables that predate them.
		for _, stmt := range schemaV1Indexes {
			if _, err := tx.Exec(stmt); err != nil {
				return fmt.Errorf("schema v1 index %q: %w", stmt, err)
			}
		}

		if _, err := tx.Exec("PRAGMA user_version = 1"); err != nil {
			return fmt.Errorf("set user_version = 1: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration to v1: %w", err)
		}
	}

	if version < 2 {
		// v2 adds events.note_slug, the link to a MyNotes note. A fresh database
		// gets it here too: schemaV1 is left as the historical v1 schema and is
		// never edited, so every database reaches the current shape the same way.
		// Guarded by columnExists so re-running is harmless.
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration to v2: %w", err)
		}
		defer tx.Rollback()

		if !columnExists(tx, "events", "note_slug") {
			if _, err := tx.Exec(`ALTER TABLE events ADD COLUMN note_slug TEXT NOT NULL DEFAULT ''`); err != nil {
				return fmt.Errorf("schema v2 add note_slug: %w", err)
			}
		}

		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", currentSchemaVersion)); err != nil {
			return fmt.Errorf("set user_version = %d: %w", currentSchemaVersion, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration to v2: %w", err)
		}
	}

	return nil
}

// tableExists reports whether a table with the given name is present.
func tableExists(q execQuerier, table string) bool {
	var n int
	_ = q.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n)
	return n > 0
}

// columnExists reports whether the given column is present on the table.
func columnExists(q execQuerier, table, column string) bool {
	var n int
	_ = q.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&n)
	return n > 0
}

// reconcileLegacy brings a pre-user_version database up to the v1 schema. Every
// step is guarded so it is a no-op on databases already at the final state.
// Ordered: add new columns, migrate calendar_name data into the calendars table
// and calendar_id, then drop the obsolete calendar_name columns.
func reconcileLegacy(tx *sql.Tx) error {
	if !columnExists(tx, "events", "ics_uid") {
		if _, err := tx.Exec(`ALTER TABLE events ADD COLUMN ics_uid TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if !columnExists(tx, "events", "calendar_id") {
		if _, err := tx.Exec(`ALTER TABLE events ADD COLUMN calendar_id INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if !columnExists(tx, "feeds", "calendar_id") {
		if _, err := tx.Exec(`ALTER TABLE feeds ADD COLUMN calendar_id INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	// Migrate the obsolete defaultEventColor preference to the default calendar's color.
	var prefColor string
	if err := tx.QueryRow(`SELECT value FROM preferences WHERE key = 'defaultEventColor'`).Scan(&prefColor); err == nil && prefColor != "" {
		if _, err := tx.Exec(`UPDATE calendars SET color = ? WHERE id = 0`, prefColor); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM preferences WHERE key = 'defaultEventColor'`); err != nil {
			return err
		}
	}

	// Migrate existing calendar_name values into the calendars table, then populate
	// calendar_id from them, then drop the calendar_name columns.
	if columnExists(tx, "events", "calendar_name") {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO calendars (name, color)
			SELECT DISTINCT calendar_name, 'dodgerblue' FROM events WHERE calendar_name != ''`); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE events SET calendar_id = COALESCE((SELECT id FROM calendars WHERE name = events.calendar_name), 0) WHERE calendar_name != '' AND calendar_id = 0`); err != nil {
			return err
		}
		if _, err := tx.Exec(`ALTER TABLE events DROP COLUMN calendar_name`); err != nil {
			return err
		}
	}
	if columnExists(tx, "feeds", "calendar_name") {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO calendars (name, color)
			SELECT DISTINCT calendar_name, 'dodgerblue' FROM feeds WHERE calendar_name != ''`); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE feeds SET calendar_id = COALESCE((SELECT id FROM calendars WHERE name = feeds.calendar_name), 0) WHERE calendar_name != '' AND calendar_id = 0`); err != nil {
			return err
		}
		if _, err := tx.Exec(`ALTER TABLE feeds DROP COLUMN calendar_name`); err != nil {
			return err
		}
	}

	return nil
}

// schemaV1 contains every DDL statement for the current schema (version 0 → 1).
// All statements use IF NOT EXISTS so the migration is safe to re-run and is a
// no-op against an existing database that already carries this schema. Legacy
// reconciliation of pre-user_version databases is handled by reconcileLegacy.
var schemaV1 = []string{
	`CREATE TABLE IF NOT EXISTS events (
		id               INTEGER PRIMARY KEY AUTOINCREMENT,
		title            TEXT NOT NULL,
		description      TEXT NOT NULL DEFAULT '',
		start_time       TEXT NOT NULL,
		end_time         TEXT NOT NULL,
		all_day          INTEGER NOT NULL DEFAULT 0,
		color            TEXT NOT NULL DEFAULT '',
		recurrence_freq  TEXT NOT NULL DEFAULT '',
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
		duration         TEXT NOT NULL DEFAULT '',
		categories       TEXT NOT NULL DEFAULT '',
		url              TEXT NOT NULL DEFAULT '',
		reminder_minutes INTEGER NOT NULL DEFAULT 0,
		location         TEXT NOT NULL DEFAULT '',
		latitude         REAL,
		longitude        REAL,
		ics_uid          TEXT NOT NULL DEFAULT '',
		calendar_id      INTEGER NOT NULL DEFAULT 0,
		created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
		updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
	)`,

	`CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time)`,
	`CREATE INDEX IF NOT EXISTS idx_events_end_time ON events(end_time)`,
	`CREATE INDEX IF NOT EXISTS idx_events_recurrence_parent_id ON events(recurrence_parent_id)`,
	`CREATE INDEX IF NOT EXISTS idx_events_time_range ON events(start_time, end_time)`,

	`CREATE VIRTUAL TABLE IF NOT EXISTS events_fts USING fts5(
		title, description, content='events', content_rowid='id'
	)`,

	`CREATE TRIGGER IF NOT EXISTS events_ai AFTER INSERT ON events BEGIN
		INSERT INTO events_fts(rowid, title, description) VALUES (new.id, new.title, new.description);
	END`,
	`CREATE TRIGGER IF NOT EXISTS events_ad AFTER DELETE ON events BEGIN
		INSERT INTO events_fts(events_fts, rowid, title, description) VALUES('delete', old.id, old.title, old.description);
	END`,
	`CREATE TRIGGER IF NOT EXISTS events_au AFTER UPDATE ON events BEGIN
		INSERT INTO events_fts(events_fts, rowid, title, description) VALUES('delete', old.id, old.title, old.description);
		INSERT INTO events_fts(rowid, title, description) VALUES (new.id, new.title, new.description);
	END`,

	`CREATE TABLE IF NOT EXISTS preferences (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL DEFAULT ''
	)`,

	`CREATE TABLE IF NOT EXISTS feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL,
		refresh_interval_minutes INTEGER NOT NULL DEFAULT 60,
		last_refreshed_at TEXT NOT NULL DEFAULT '',
		last_error TEXT NOT NULL DEFAULT '',
		enabled INTEGER NOT NULL DEFAULT 1,
		calendar_id INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
	)`,

	`CREATE TABLE IF NOT EXISTS calendars (
		id    INTEGER PRIMARY KEY AUTOINCREMENT,
		name  TEXT NOT NULL UNIQUE,
		color TEXT NOT NULL DEFAULT 'dodgerblue'
	)`,

	// id=0 is reserved for the default calendar. OR IGNORE makes concurrent
	// first-run inserts race-safe.
	`INSERT OR IGNORE INTO calendars (id, name, color) VALUES (0, 'Default', 'dodgerblue')`,
}

// schemaV1Indexes are indexes on columns (ics_uid, calendar_id) that legacy
// databases gain only after reconcileLegacy runs, so they are created last.
var schemaV1Indexes = []string{
	`CREATE INDEX IF NOT EXISTS idx_events_ics_uid ON events(ics_uid)`,
	`CREATE INDEX IF NOT EXISTS idx_events_calendar_id ON events(calendar_id)`,
	`CREATE INDEX IF NOT EXISTS idx_feeds_calendar_id ON feeds(calendar_id)`,
}
