package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/mikaelstaldal/go-server-common/sqlite"
)

// ErrLegacySchema is returned for a database that predates the user_version
// scheme. mycal used to reconcile such databases in Go, inspecting them column by
// column; that code is gone, so one is now refused rather than guessed at.
//
// The last build of that generation is d89ff9b, which reconciled on every startup
// without recording that it had; user_version arrived in the commit after it. Any
// database still in the pre-versioning shape has therefore not been started since.
var ErrLegacySchema = errors.New("database predates schema versioning")

// OpenDB opens the SQLite database at path, enables foreign keys, sets the
// busy_timeout pragma (0 = skip), applies any extraPragmas, sizes the connection
// pool, and runs pending schema migrations. Connection setup (DSN, pragmas, WAL
// mode) is delegated to the shared sqlite package.
//
// Migrations are applied by sqlite.MigrateStrict rather than by Open, so that the
// pool is sized before anything migrates and so that each batch decides what it
// needs only after taking the write lock. mycal has one migrating process, so the
// second property is uniformity with the other services rather than a race it has.
func OpenDB(path string, busyTimeout int, extraPragmas ...string) (*sql.DB, error) {
	// Passing no migrations here leaves migrating to initSchema while still
	// letting the shared package build the DSN, bake in pragmas and set WAL mode.
	db, err := sqlite.Open(path, busyTimeout, nil, extraPragmas...)
	if err != nil {
		return nil, err
	}

	// Sized before migrating rather than by the caller afterwards: MigrateStrict
	// takes a connection per batch, and a pool still at its defaults is one the
	// caller has not chosen. This cannot precede the WAL step, which sqlite.Open
	// performs before it returns the *sql.DB there is anything to configure.
	numConns := runtime.GOMAXPROCS(0)
	db.SetMaxOpenConns(numConns)
	db.SetMaxIdleConns(numConns)

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
// carries. Derived from migrations rather than maintained by hand, so that it
// cannot disagree with the version MigrateStrict refuses a database for being
// newer than.
var currentSchemaVersion = len(migrations)

// migrations is the schema, one slice per version: migrations[0] takes a database
// from user_version 0 to 1, migrations[1] from 1 to 2.
//
// schemaV1Indexes is concatenated rather than folded into schemaV1 so that
// schemaV1 stays byte-identical to the historical v1 statement list. The two were
// separate because the indexes name columns that legacy databases only gained
// during reconciliation, and so had to be created after it; with reconciliation
// gone the ordering is no longer load-bearing, only the grouping is.
var migrations = [][]string{
	append(append([]string{}, schemaV1...), schemaV1Indexes...),
	{`ALTER TABLE events ADD COLUMN note_slug TEXT NOT NULL DEFAULT ''`},
	{`CREATE TABLE api_tokens (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 slug TEXT NOT NULL UNIQUE,
 name TEXT NOT NULL,
 token_hash BLOB NOT NULL UNIQUE,
 created_at TEXT NOT NULL,
 expires_at TEXT NOT NULL
 )`, `CREATE TABLE api_token_calendars (
 token_id INTEGER NOT NULL REFERENCES api_tokens(id) ON DELETE CASCADE,
 calendar_id INTEGER NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
 PRIMARY KEY(token_id, calendar_id)
 )`},
}

// initSchema refuses a pre-user_version database and hands everything else to
// sqlite.MigrateStrict.
//
// A database stamped past the latest migration was written by a newer mycal, so
// this binary does not know the shape it would be writing to; MigrateStrict
// refuses it with sqlite.ErrSchemaTooNew. A database at user_version 0 that
// already has an events table is the opposite case: written by a mycal from
// before the versioning scheme, in a shape this binary no longer models. Every
// statement below assumes it is building a schema from nothing, so running them
// against an existing one could complete without error and still be wrong — the
// damaged shapes the old non-transactional migrator could leave behind take the
// columns but not the data. Refusing is the only honest answer.
//
// The guard is also what lets migrations[0] assume an empty database: it carries
// no FTS rebuild, because the index it would rebuild can have no rows to hold.
//
// The check reads user_version outside the write lock MigrateStrict takes, so it
// assumes one migrating process — true of mycal, where OpenDB is reached only from
// server startup and -export-ics opens the database read-only without migrating.
// Anything that gives mycal a second read-write entry point, an -import or -repair
// mode say, invalidates that: between this check and the lock, another process
// could create the very table being checked for. The detection would then have to
// move inside the transaction, and this comment is the notice that it has not.
func initSchema(db *sql.DB) error {
	version, err := sqlite.UserVersion(db)
	if err != nil {
		return err
	}
	if version == 0 && tableExists(db, "events") {
		return fmt.Errorf("%w: user_version is 0 with an existing events table. "+
			"This build cannot migrate it: the code that reconciled pre-versioning "+
			"databases has been removed. Start it once with an older mycal build to "+
			"migrate and stamp it, then upgrade again", ErrLegacySchema)
	}

	// Background rather than a caller's context: this runs once at startup, before
	// anything that could be cancelled exists, and a migration abandoned halfway
	// through its batch is worth less than one that finishes.
	return sqlite.MigrateStrict(context.Background(), db, migrations)
}

// tableExists reports whether a table with the given name is present.
func tableExists(db *sql.DB, table string) bool {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n)
	return n > 0
}

// columnExists reports whether the given column is present on the table.
func columnExists(db *sql.DB, table, column string) bool {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&n)
	return n > 0
}

// schemaV1 contains every DDL statement for the current schema (version 0 → 1).
// All statements use IF NOT EXISTS so the migration is safe to re-run and is a
// no-op against an existing database that already carries this schema.
//
// This is the historical v1 statement list and is not edited. A database reaches
// the current shape by running it and then every later migration in turn, so
// changing it would mean two databases at the same user_version having been built
// from different text.
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

// schemaV1Indexes are part of migration 0 and are concatenated onto schemaV1 by
// migrations. They are a separate slice only to keep schemaV1 byte-identical to
// its historical text; the order the two are applied in no longer matters, since
// the columns they index are in schemaV1's own CREATE TABLE statements. They were
// once applied last because legacy databases gained those columns during a
// reconciliation that has since been deleted — do not read this separation as an
// ordering constraint, and do not fold it away either.
var schemaV1Indexes = []string{
	`CREATE INDEX IF NOT EXISTS idx_events_ics_uid ON events(ics_uid)`,
	`CREATE INDEX IF NOT EXISTS idx_events_calendar_id ON events(calendar_id)`,
	`CREATE INDEX IF NOT EXISTS idx_feeds_calendar_id ON feeds(calendar_id)`,
}
