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
			created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		);
		CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time);
		CREATE INDEX IF NOT EXISTS idx_events_end_time ON events(end_time);
		CREATE INDEX IF NOT EXISTS idx_events_recurrence_parent_id ON events(recurrence_parent_id);

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

	_, err = db.Exec(`INSERT INTO events_fts(events_fts) VALUES('rebuild')`)
	return err
}

const selectColumns = `id, title, description, start_time, end_time, all_day, color, recurrence_freq, recurrence_count, recurrence_until, recurrence_interval, recurrence_by_day, recurrence_by_monthday, recurrence_by_month, exdates, rdates, recurrence_parent_id, recurrence_original_start, duration, categories, url, reminder_minutes, location, latitude, longitude, created_at, updated_at`

func scanEvent(scanner interface{ Scan(...any) error }) (model.Event, error) {
	var e model.Event
	var lat, lon sql.NullFloat64
	var parentID sql.NullInt64
	err := scanner.Scan(&e.ID, &e.Title, &e.Description, &e.StartTime, &e.EndTime, &e.AllDay, &e.Color, &e.RecurrenceFreq, &e.RecurrenceCount, &e.RecurrenceUntil, &e.RecurrenceInterval, &e.RecurrenceByDay, &e.RecurrenceByMonthDay, &e.RecurrenceByMonth, &e.ExDates, &e.RDates, &parentID, &e.RecurrenceOriginalStart, &e.Duration, &e.Categories, &e.URL, &e.ReminderMinutes, &e.Location, &lat, &lon, &e.CreatedAt, &e.UpdatedAt)
	if lat.Valid {
		e.Latitude = &lat.Float64
	}
	if lon.Valid {
		e.Longitude = &lon.Float64
	}
	if parentID.Valid {
		e.RecurrenceParentID = &parentID.Int64
	}
	return e, err
}

func (r *SQLiteRepository) List(from, to string) ([]model.Event, error) {
	rows, err := r.db.Query(
		`SELECT `+selectColumns+`
		 FROM events WHERE start_time < ? AND end_time > ? AND recurrence_freq = '' AND recurrence_parent_id IS NULL ORDER BY start_time`,
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

	sb.WriteString(`SELECT e.id, e.title, e.description, e.start_time, e.end_time, e.all_day, e.color, e.recurrence_freq, e.recurrence_count, e.recurrence_until, e.recurrence_interval, e.recurrence_by_day, e.recurrence_by_monthday, e.recurrence_by_month, e.exdates, e.rdates, e.recurrence_parent_id, e.recurrence_original_start, e.duration, e.categories, e.url, e.reminder_minutes, e.location, e.latitude, e.longitude, e.created_at, e.updated_at
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
		`INSERT INTO events (title, description, start_time, end_time, all_day, color, recurrence_freq, recurrence_count, recurrence_until, recurrence_interval, recurrence_by_day, recurrence_by_monthday, recurrence_by_month, exdates, rdates, recurrence_parent_id, recurrence_original_start, duration, categories, url, reminder_minutes, location, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.Title, event.Description, event.StartTime, event.EndTime, event.AllDay, event.Color, event.RecurrenceFreq, event.RecurrenceCount, event.RecurrenceUntil, event.RecurrenceInterval, event.RecurrenceByDay, event.RecurrenceByMonthDay, event.RecurrenceByMonth, event.ExDates, event.RDates, event.RecurrenceParentID, event.RecurrenceOriginalStart, event.Duration, event.Categories, event.URL, event.ReminderMinutes, event.Location, event.Latitude, event.Longitude,
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
		`UPDATE events SET title=?, description=?, start_time=?, end_time=?, all_day=?, color=?, recurrence_freq=?, recurrence_count=?, recurrence_until=?, recurrence_interval=?, recurrence_by_day=?, recurrence_by_monthday=?, recurrence_by_month=?, exdates=?, rdates=?, recurrence_parent_id=?, recurrence_original_start=?, duration=?, categories=?, url=?, reminder_minutes=?, location=?, latitude=?, longitude=?,
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
		event.Title, event.Description, event.StartTime, event.EndTime, event.AllDay, event.Color, event.RecurrenceFreq, event.RecurrenceCount, event.RecurrenceUntil, event.RecurrenceInterval, event.RecurrenceByDay, event.RecurrenceByMonthDay, event.RecurrenceByMonth, event.ExDates, event.RDates, event.RecurrenceParentID, event.RecurrenceOriginalStart, event.Duration, event.Categories, event.URL, event.ReminderMinutes, event.Location, event.Latitude, event.Longitude, event.ID,
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
		 FROM events WHERE recurrence_freq != '' AND start_time < ? AND recurrence_parent_id IS NULL ORDER BY start_time`,
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

func (r *SQLiteRepository) ListOverrides(parentIDs []int64) ([]model.Event, error) {
	if len(parentIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(parentIDs))
	args := make([]any, len(parentIDs))
	for i, id := range parentIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := `SELECT ` + selectColumns + `
		FROM events WHERE recurrence_parent_id IN (` + strings.Join(placeholders, ",") + `) ORDER BY start_time`
	rows, err := r.db.Query(query, args...)
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

func (r *SQLiteRepository) GetOverride(parentID int64, originalStart string) (*model.Event, error) {
	e, err := scanEvent(r.db.QueryRow(
		`SELECT `+selectColumns+`
		 FROM events WHERE recurrence_parent_id = ? AND recurrence_original_start = ?`, parentID, originalStart,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *SQLiteRepository) DeleteByParentID(parentID int64) error {
	_, err := r.db.Exec(`DELETE FROM events WHERE recurrence_parent_id = ?`, parentID)
	return err
}
