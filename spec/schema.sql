-- Generated from a freshly migrated database. Do not edit by hand.
-- Regenerate with:
-- go test ./internal/repository -run TestSchemaSnapshotMatchesFreshDatabase -update-schema

CREATE TABLE events (
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
	, note_slug TEXT NOT NULL DEFAULT '');

CREATE INDEX idx_events_start_time ON events(start_time);

CREATE INDEX idx_events_end_time ON events(end_time);

CREATE INDEX idx_events_recurrence_parent_id ON events(recurrence_parent_id);

CREATE INDEX idx_events_time_range ON events(start_time, end_time);

CREATE VIRTUAL TABLE events_fts USING fts5(
		title, description, content='events', content_rowid='id'
	);

CREATE TRIGGER events_ai AFTER INSERT ON events BEGIN
		INSERT INTO events_fts(rowid, title, description) VALUES (new.id, new.title, new.description);
	END;

CREATE TRIGGER events_ad AFTER DELETE ON events BEGIN
		INSERT INTO events_fts(events_fts, rowid, title, description) VALUES('delete', old.id, old.title, old.description);
	END;

CREATE TRIGGER events_au AFTER UPDATE ON events BEGIN
		INSERT INTO events_fts(events_fts, rowid, title, description) VALUES('delete', old.id, old.title, old.description);
		INSERT INTO events_fts(rowid, title, description) VALUES (new.id, new.title, new.description);
	END;

CREATE TABLE preferences (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL DEFAULT ''
	);

CREATE TABLE feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL,
		refresh_interval_minutes INTEGER NOT NULL DEFAULT 60,
		last_refreshed_at TEXT NOT NULL DEFAULT '',
		last_error TEXT NOT NULL DEFAULT '',
		enabled INTEGER NOT NULL DEFAULT 1,
		calendar_id INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
	);

CREATE TABLE calendars (
		id    INTEGER PRIMARY KEY AUTOINCREMENT,
		name  TEXT NOT NULL UNIQUE,
		color TEXT NOT NULL DEFAULT 'dodgerblue'
	);

CREATE INDEX idx_events_ics_uid ON events(ics_uid);

CREATE INDEX idx_events_calendar_id ON events(calendar_id);

CREATE INDEX idx_feeds_calendar_id ON feeds(calendar_id);

PRAGMA user_version = 2;
