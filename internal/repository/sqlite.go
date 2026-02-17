package repository

import (
	"database/sql"
	"fmt"

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
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			title       TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			start_time  TEXT NOT NULL,
			end_time    TEXT NOT NULL,
			color       TEXT NOT NULL DEFAULT '',
			created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		);
		CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time);
		CREATE INDEX IF NOT EXISTS idx_events_end_time ON events(end_time);
	`)
	return err
}

func (r *SQLiteRepository) List(from, to string) ([]model.Event, error) {
	rows, err := r.db.Query(
		`SELECT id, title, description, start_time, end_time, color, created_at, updated_at
		 FROM events WHERE start_time < ? AND end_time > ? ORDER BY start_time`,
		to, from,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var e model.Event
		if err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.StartTime, &e.EndTime, &e.Color, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *SQLiteRepository) GetByID(id int64) (*model.Event, error) {
	var e model.Event
	err := r.db.QueryRow(
		`SELECT id, title, description, start_time, end_time, color, created_at, updated_at
		 FROM events WHERE id = ?`, id,
	).Scan(&e.ID, &e.Title, &e.Description, &e.StartTime, &e.EndTime, &e.Color, &e.CreatedAt, &e.UpdatedAt)
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
		`INSERT INTO events (title, description, start_time, end_time, color) VALUES (?, ?, ?, ?, ?)`,
		event.Title, event.Description, event.StartTime, event.EndTime, event.Color,
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
		`UPDATE events SET title=?, description=?, start_time=?, end_time=?, color=?,
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
		event.Title, event.Description, event.StartTime, event.EndTime, event.Color, event.ID,
	)
	if err != nil {
		return err
	}
	return r.db.QueryRow(
		`SELECT updated_at FROM events WHERE id = ?`, event.ID,
	).Scan(&event.UpdatedAt)
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
