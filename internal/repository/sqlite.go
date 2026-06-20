package repository

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/mikaelstaldal/mycal/internal/model"
	_ "modernc.org/sqlite"
)

type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository wraps an already-opened database. Schema migrations are
// run by OpenDB, not here, so this can wrap a read-only connection too.
func NewSQLiteRepository(db *sql.DB) (*SQLiteRepository, error) {
	return &SQLiteRepository{db: db}, nil
}

const selectColumnsBase = `e.id, e.title, e.description, e.start_time, e.end_time, e.all_day, e.color, e.recurrence_freq, e.recurrence_count, e.recurrence_until, e.recurrence_interval, e.recurrence_by_day, e.recurrence_by_monthday, e.recurrence_by_month, e.exdates, e.rdates, e.recurrence_parent_id, e.recurrence_original_start, e.duration, e.categories, e.url, e.reminder_minutes, e.location, e.latitude, e.longitude, e.calendar_id, COALESCE(cal.name, ''), e.ics_uid, e.created_at, e.updated_at`

const fromEventsJoin = ` FROM events e LEFT JOIN calendars cal ON e.calendar_id = cal.id`

func scanEvent(scanner interface{ Scan(...any) error }) (model.Event, error) {
	var e model.Event
	var lat, lon sql.NullFloat64
	var parentID sql.NullInt64
	err := scanner.Scan(&e.ID, &e.Title, &e.Description, &e.StartTime, &e.EndTime, &e.AllDay, &e.Color, &e.RecurrenceFreq, &e.RecurrenceCount, &e.RecurrenceUntil, &e.RecurrenceInterval, &e.RecurrenceByDay, &e.RecurrenceByMonthDay, &e.RecurrenceByMonth, &e.ExDates, &e.RDates, &parentID, &e.RecurrenceOriginalStart, &e.Duration, &e.Categories, &e.URL, &e.ReminderMinutes, &e.Location, &lat, &lon, &e.CalendarID, &e.CalendarName, &e.IcsUID, &e.CreatedAt, &e.UpdatedAt)
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

func calendarIDFilter(calendarIDs []int64) (string, []any) {
	if calendarIDs == nil {
		return "", nil
	}
	if len(calendarIDs) == 0 {
		return " AND 1=0", nil
	}
	placeholders := make([]string, len(calendarIDs))
	args := make([]any, len(calendarIDs))
	for i, id := range calendarIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	return " AND e.calendar_id IN (" + strings.Join(placeholders, ",") + ")", args
}

func (r *SQLiteRepository) List(from, to string, calendarIDs []int64) ([]model.Event, error) {
	filterSQL, filterArgs := calendarIDFilter(calendarIDs)
	args := []any{to, from}
	args = append(args, filterArgs...)
	rows, err := r.db.Query(
		`SELECT `+selectColumnsBase+fromEventsJoin+` WHERE e.start_time < ? AND e.end_time > ? AND e.recurrence_freq = '' AND e.recurrence_parent_id IS NULL`+filterSQL+` ORDER BY e.start_time, e.created_at`,
		args...,
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

func (r *SQLiteRepository) ListAll(calendarIDs []int64) ([]model.Event, error) {
	filterSQL, filterArgs := calendarIDFilter(calendarIDs)
	query := `SELECT ` + selectColumnsBase + fromEventsJoin + ` WHERE 1=1` + filterSQL + ` ORDER BY e.start_time, e.created_at`
	rows, err := r.db.Query(query, filterArgs...)
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

func (r *SQLiteRepository) Search(query, from, to string, calendarIDs []int64) ([]model.Event, error) {
	ftsQuery := sanitizeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	var sb strings.Builder
	var args []any

	sb.WriteString(`SELECT ` + selectColumnsBase + `
		FROM events e
		LEFT JOIN calendars cal ON e.calendar_id = cal.id
		JOIN events_fts f ON e.id = f.rowid
		WHERE events_fts MATCH ?`)
	args = append(args, ftsQuery)

	if from != "" && to != "" {
		sb.WriteString(` AND e.start_time < ? AND e.end_time > ?`)
		args = append(args, to, from)
	}

	filterSQL, filterArgs := calendarIDFilter(calendarIDs)
	if filterSQL != "" {
		sb.WriteString(filterSQL)
		args = append(args, filterArgs...)
	}

	sb.WriteString(` ORDER BY e.start_time DESC`)

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
		`SELECT `+selectColumnsBase+fromEventsJoin+` WHERE e.id = ?`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *SQLiteRepository) Create(event *model.Event) error {
	err := r.db.QueryRow(
		`INSERT INTO events (title, description, start_time, end_time, all_day, color, recurrence_freq, recurrence_count, recurrence_until, recurrence_interval, recurrence_by_day, recurrence_by_monthday, recurrence_by_month, exdates, rdates, recurrence_parent_id, recurrence_original_start, duration, categories, url, reminder_minutes, location, latitude, longitude, calendar_id, ics_uid) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id, created_at, updated_at`,
		event.Title, event.Description, event.StartTime, event.EndTime, event.AllDay, event.Color, event.RecurrenceFreq, event.RecurrenceCount, event.RecurrenceUntil, event.RecurrenceInterval, event.RecurrenceByDay, event.RecurrenceByMonthDay, event.RecurrenceByMonth, event.ExDates, event.RDates, event.RecurrenceParentID, event.RecurrenceOriginalStart, event.Duration, event.Categories, event.URL, event.ReminderMinutes, event.Location, event.Latitude, event.Longitude, event.CalendarID, event.IcsUID,
	).Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt)
	if err != nil {
		return err
	}
	if event.CalendarID != 0 {
		return r.db.QueryRow(
			`SELECT COALESCE(name, '') FROM calendars WHERE id = ?`, event.CalendarID,
		).Scan(&event.CalendarName)
	}
	return nil
}

func (r *SQLiteRepository) Update(event *model.Event) error {
	return r.db.QueryRow(
		`UPDATE events SET title=?, description=?, start_time=?, end_time=?, all_day=?, color=?, recurrence_freq=?, recurrence_count=?, recurrence_until=?, recurrence_interval=?, recurrence_by_day=?, recurrence_by_monthday=?, recurrence_by_month=?, exdates=?, rdates=?, recurrence_parent_id=?, recurrence_original_start=?, duration=?, categories=?, url=?, reminder_minutes=?, location=?, latitude=?, longitude=?, calendar_id=?, ics_uid=?,
		updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=? RETURNING updated_at`,
		event.Title, event.Description, event.StartTime, event.EndTime, event.AllDay, event.Color, event.RecurrenceFreq, event.RecurrenceCount, event.RecurrenceUntil, event.RecurrenceInterval, event.RecurrenceByDay, event.RecurrenceByMonthDay, event.RecurrenceByMonth, event.ExDates, event.RDates, event.RecurrenceParentID, event.RecurrenceOriginalStart, event.Duration, event.Categories, event.URL, event.ReminderMinutes, event.Location, event.Latitude, event.Longitude, event.CalendarID, event.IcsUID, event.ID,
	).Scan(&event.UpdatedAt)
}

func (r *SQLiteRepository) ListRecurring(to string, calendarIDs []int64) ([]model.Event, error) {
	filterSQL, filterArgs := calendarIDFilter(calendarIDs)
	args := []any{to}
	args = append(args, filterArgs...)
	rows, err := r.db.Query(
		`SELECT `+selectColumnsBase+fromEventsJoin+` WHERE e.recurrence_freq != '' AND e.start_time < ? AND e.recurrence_parent_id IS NULL`+filterSQL+` ORDER BY e.start_time, e.created_at`,
		args...,
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

func (r *SQLiteRepository) ListOverrides(parentIDs []int64, from, to string) ([]model.Event, error) {
	if len(parentIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(parentIDs))
	args := make([]any, len(parentIDs))
	for i, id := range parentIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	// Include overrides whose new time overlaps the window, or whose original
	// occurrence falls within the window (so we can suppress the generated instance).
	query := `SELECT ` + selectColumnsBase + fromEventsJoin +
		` WHERE e.recurrence_parent_id IN (` + strings.Join(placeholders, ",") + `)` +
		` AND ((e.start_time < ? AND e.end_time > ?) OR (e.recurrence_original_start >= ? AND e.recurrence_original_start < ?))` +
		` ORDER BY e.start_time, e.created_at`
	args = append(args, to, from, from, to)
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
		`SELECT `+selectColumnsBase+fromEventsJoin+` WHERE e.recurrence_parent_id = ? AND e.recurrence_original_start = ?`, parentID, originalStart,
	))
	if errors.Is(err, sql.ErrNoRows) {
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

func (r *SQLiteRepository) FilterExistingIcsUIDs(uids []string) (map[string]bool, error) {
	if len(uids) == 0 {
		return map[string]bool{}, nil
	}
	placeholders := make([]string, len(uids))
	args := make([]any, len(uids))
	for i, uid := range uids {
		placeholders[i] = "?"
		args[i] = uid
	}
	query := "SELECT ics_uid FROM events WHERE ics_uid IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	existing := make(map[string]bool, len(uids))
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		existing[uid] = true
	}
	return existing, rows.Err()
}

// Feed repository methods

func (r *SQLiteRepository) CreateFeed(feed *model.Feed) error {
	result, err := r.db.Exec(
		`INSERT INTO feeds (url, calendar_id, refresh_interval_minutes, enabled) VALUES (?, ?, ?, ?)`,
		feed.URL, feed.CalendarID, feed.RefreshIntervalMinutes, feed.Enabled,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	feed.ID = id
	return r.db.QueryRow(
		`SELECT f.created_at, f.updated_at, COALESCE(c.name, '') FROM feeds f LEFT JOIN calendars c ON f.calendar_id = c.id WHERE f.id = ?`, id,
	).Scan(&feed.CreatedAt, &feed.UpdatedAt, &feed.CalendarName)

}

func (r *SQLiteRepository) GetFeedByID(id int64) (*model.Feed, error) {
	var f model.Feed
	err := r.db.QueryRow(
		`SELECT f.id, f.url, f.calendar_id, COALESCE(c.name, ''), f.refresh_interval_minutes, f.last_refreshed_at, f.last_error, f.enabled, f.created_at, f.updated_at FROM feeds f LEFT JOIN calendars c ON f.calendar_id = c.id WHERE f.id = ?`, id,
	).Scan(&f.ID, &f.URL, &f.CalendarID, &f.CalendarName, &f.RefreshIntervalMinutes, &f.LastRefreshedAt, &f.LastError, &f.Enabled, &f.CreatedAt, &f.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *SQLiteRepository) ListFeeds() ([]model.Feed, error) {
	rows, err := r.db.Query(
		`SELECT f.id, f.url, f.calendar_id, COALESCE(c.name, ''), f.refresh_interval_minutes, f.last_refreshed_at, f.last_error, f.enabled, f.created_at, f.updated_at FROM feeds f LEFT JOIN calendars c ON f.calendar_id = c.id ORDER BY f.created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []model.Feed
	for rows.Next() {
		var f model.Feed
		if err := rows.Scan(&f.ID, &f.URL, &f.CalendarID, &f.CalendarName, &f.RefreshIntervalMinutes, &f.LastRefreshedAt, &f.LastError, &f.Enabled, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	return feeds, rows.Err()
}

func (r *SQLiteRepository) UpdateFeed(feed *model.Feed) error {
	_, err := r.db.Exec(
		`UPDATE feeds SET url=?, calendar_id=?, refresh_interval_minutes=?, last_refreshed_at=?, last_error=?, enabled=?, updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
		feed.URL, feed.CalendarID, feed.RefreshIntervalMinutes, feed.LastRefreshedAt, feed.LastError, feed.Enabled, feed.ID,
	)
	if err != nil {
		return err
	}
	return r.db.QueryRow(
		`SELECT f.updated_at, COALESCE(c.name, '') FROM feeds f LEFT JOIN calendars c ON f.calendar_id = c.id WHERE f.id = ?`, feed.ID,
	).Scan(&feed.UpdatedAt, &feed.CalendarName)
}

func (r *SQLiteRepository) DeleteFeed(id int64) error {
	result, err := r.db.Exec(`DELETE FROM feeds WHERE id = ?`, id)
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

func (r *SQLiteRepository) GetAllPreferences() (map[string]string, error) {
	rows, err := r.db.Query(`SELECT key, value FROM preferences`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prefs := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		prefs[k] = v
	}
	return prefs, rows.Err()
}

func (r *SQLiteRepository) GetPreference(key string) (string, bool, error) {
	var value string
	err := r.db.QueryRow(`SELECT value FROM preferences WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (r *SQLiteRepository) SetPreference(key, value string) error {
	_, err := r.db.Exec(`INSERT INTO preferences (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (r *SQLiteRepository) DeletePreference(key string) error {
	_, err := r.db.Exec(`DELETE FROM preferences WHERE key = ?`, key)
	return err
}

// Calendar repository methods

func (r *SQLiteRepository) ListCalendars() ([]model.Calendar, error) {
	rows, err := r.db.Query(`SELECT id, name, color FROM calendars ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calendars []model.Calendar
	for rows.Next() {
		var c model.Calendar
		if err := rows.Scan(&c.ID, &c.Name, &c.Color); err != nil {
			return nil, err
		}
		calendars = append(calendars, c)
	}
	return calendars, rows.Err()
}

func (r *SQLiteRepository) GetCalendarByID(id int64) (*model.Calendar, error) {
	var c model.Calendar
	err := r.db.QueryRow(`SELECT id, name, color FROM calendars WHERE id = ?`, id).Scan(&c.ID, &c.Name, &c.Color)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *SQLiteRepository) GetCalendarByName(name string) (*model.Calendar, error) {
	var c model.Calendar
	err := r.db.QueryRow(`SELECT id, name, color FROM calendars WHERE name = ?`, name).Scan(&c.ID, &c.Name, &c.Color)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *SQLiteRepository) CreateCalendar(cal *model.Calendar) error {
	result, err := r.db.Exec(`INSERT INTO calendars (name, color) VALUES (?, ?)`, cal.Name, cal.Color)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	cal.ID = id
	return nil
}

func (r *SQLiteRepository) UpdateCalendar(cal *model.Calendar) error {
	_, err := r.db.Exec(`UPDATE calendars SET name = ?, color = ? WHERE id = ?`, cal.Name, cal.Color, cal.ID)
	return err
}

func (r *SQLiteRepository) DeleteCalendarIfUnused(id int64) error {
	_, err := r.db.Exec(`DELETE FROM calendars WHERE id = ? AND id != 0
		AND NOT EXISTS (SELECT 1 FROM events WHERE calendar_id = ?)
		AND NOT EXISTS (SELECT 1 FROM feeds WHERE calendar_id = ?)`, id, id, id)
	return err
}
