package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/mikaelstaldal/mycal/internal/model"
	_ "modernc.org/sqlite"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) (*SQLiteRepository, error) {
	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &SQLiteRepository{db: db}, nil
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			title            TEXT NOT NULL,
			description      TEXT NOT NULL DEFAULT '',
			start_time       TEXT NOT NULL,
			end_time         TEXT NOT NULL,
			all_day          INTEGER NOT NULL DEFAULT 0,
			color            TEXT NOT NULL DEFAULT '',
			recurrence_freq  TEXT NOT NULL DEFAULT '',
			recurrence_count INTEGER NOT NULL DEFAULT 0,
			created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		);
		CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time);
		CREATE INDEX IF NOT EXISTS idx_events_end_time ON events(end_time);

		CREATE VIRTUAL TABLE IF NOT EXISTS events_fts USING fts5(
			title, description, content='events', content_rowid='id'
		);

		CREATE TRIGGER IF NOT EXISTS events_ai AFTER INSERT ON events BEGIN
			INSERT INTO events_fts(rowid, title, description) VALUES (new.id, new.title, new.description);
		END;
		CREATE TRIGGER IF NOT EXISTS events_ad AFTER DELETE ON events BEGIN
			INSERT INTO events_fts(events_fts, rowid, title, description) VALUES('delete', old.id, old.title, old.description);
		END;
		CREATE TRIGGER IF NOT EXISTS events_au AFTER UPDATE ON events BEGIN
			INSERT INTO events_fts(events_fts, rowid, title, description) VALUES('delete', old.id, old.title, old.description);
			INSERT INTO events_fts(rowid, title, description) VALUES (new.id, new.title, new.description);
		END;
	`)
	if err != nil {
		return err
	}

	// Migration: add all_day column if it doesn't exist
	if err := migrateAddAllDay(db); err != nil {
		return err
	}

	// Migration: add recurrence columns if they don't exist
	if err := migrateAddRecurrence(db); err != nil {
		return err
	}

	// Migration: add reminder_minutes column if it doesn't exist
	if err := migrateAddReminderMinutes(db); err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO events_fts(events_fts) VALUES('rebuild')`)
	return err
}

func migrateAddAllDay(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(events)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasAllDay := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "all_day" {
			hasAllDay = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !hasAllDay {
		_, err := db.Exec(`ALTER TABLE events ADD COLUMN all_day INTEGER NOT NULL DEFAULT 0`)
		if err != nil {
			return fmt.Errorf("migrate add all_day: %w", err)
		}
	}
	return nil
}

func migrateAddRecurrence(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(events)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasFreq := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "recurrence_freq" {
			hasFreq = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !hasFreq {
		if _, err := db.Exec(`ALTER TABLE events ADD COLUMN recurrence_freq TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("migrate add recurrence_freq: %w", err)
		}
		if _, err := db.Exec(`ALTER TABLE events ADD COLUMN recurrence_count INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("migrate add recurrence_count: %w", err)
		}
	}
	return nil
}

func migrateAddReminderMinutes(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(events)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasReminder := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "reminder_minutes" {
			hasReminder = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !hasReminder {
		_, err := db.Exec(`ALTER TABLE events ADD COLUMN reminder_minutes INTEGER NOT NULL DEFAULT 0`)
		if err != nil {
			return fmt.Errorf("migrate add reminder_minutes: %w", err)
		}
	}
	return nil
}

const selectColumns = `id, title, description, start_time, end_time, all_day, color, recurrence_freq, recurrence_count, reminder_minutes, created_at, updated_at`

func scanEvent(scanner interface{ Scan(...any) error }) (model.Event, error) {
	var e model.Event
	err := scanner.Scan(&e.ID, &e.Title, &e.Description, &e.StartTime, &e.EndTime, &e.AllDay, &e.Color, &e.RecurrenceFreq, &e.RecurrenceCount, &e.ReminderMinutes, &e.CreatedAt, &e.UpdatedAt)
	return e, err
}

func (r *SQLiteRepository) List(from, to string) ([]model.Event, error) {
	rows, err := r.db.Query(
		`SELECT `+selectColumns+`
		 FROM events WHERE start_time < ? AND end_time > ? AND recurrence_freq = '' ORDER BY start_time`,
		to, from,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *SQLiteRepository) ListAll() ([]model.Event, error) {
	rows, err := r.db.Query(
		`SELECT ` + selectColumns + `
		 FROM events ORDER BY start_time`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func sanitizeFTSQuery(query string) string {
	terms := strings.Fields(query)
	quoted := make([]string, 0, len(terms))
	for _, t := range terms {
		escaped := strings.ReplaceAll(t, `"`, `""`)
		quoted = append(quoted, `"`+escaped+`"`)
	}
	return strings.Join(quoted, " ")
}

func (r *SQLiteRepository) Search(query, from, to string) ([]model.Event, error) {
	ftsQuery := sanitizeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	var sb strings.Builder
	var args []any

	sb.WriteString(`SELECT e.id, e.title, e.description, e.start_time, e.end_time, e.all_day, e.color, e.recurrence_freq, e.recurrence_count, e.reminder_minutes, e.created_at, e.updated_at
		FROM events e
		JOIN events_fts f ON e.id = f.rowid
		WHERE events_fts MATCH ?`)
	args = append(args, ftsQuery)

	if from != "" && to != "" {
		sb.WriteString(` AND e.start_time < ? AND e.end_time > ?`)
		args = append(args, to, from)
	}

	sb.WriteString(` ORDER BY f.rank`)

	rows, err := r.db.Query(sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *SQLiteRepository) GetByID(id int64) (*model.Event, error) {
	e, err := scanEvent(r.db.QueryRow(
		`SELECT `+selectColumns+`
		 FROM events WHERE id = ?`, id,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *SQLiteRepository) Create(event *model.Event) error {
	result, err := r.db.Exec(
		`INSERT INTO events (title, description, start_time, end_time, all_day, color, recurrence_freq, recurrence_count, reminder_minutes) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.Title, event.Description, event.StartTime, event.EndTime, event.AllDay, event.Color, event.RecurrenceFreq, event.RecurrenceCount, event.ReminderMinutes,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	event.ID = id

	return r.db.QueryRow(
		`SELECT created_at, updated_at FROM events WHERE id = ?`, id,
	).Scan(&event.CreatedAt, &event.UpdatedAt)
}

func (r *SQLiteRepository) Update(event *model.Event) error {
	_, err := r.db.Exec(
		`UPDATE events SET title=?, description=?, start_time=?, end_time=?, all_day=?, color=?, recurrence_freq=?, recurrence_count=?, reminder_minutes=?,
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
		event.Title, event.Description, event.StartTime, event.EndTime, event.AllDay, event.Color, event.RecurrenceFreq, event.RecurrenceCount, event.ReminderMinutes, event.ID,
	)
	if err != nil {
		return err
	}
	return r.db.QueryRow(
		`SELECT updated_at FROM events WHERE id = ?`, event.ID,
	).Scan(&event.UpdatedAt)
}

func (r *SQLiteRepository) ListRecurring(to string) ([]model.Event, error) {
	rows, err := r.db.Query(
		`SELECT `+selectColumns+`
		 FROM events WHERE recurrence_freq != '' AND start_time < ? ORDER BY start_time`,
		to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *SQLiteRepository) Delete(id int64) error {
	result, err := r.db.Exec(`DELETE FROM events WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
