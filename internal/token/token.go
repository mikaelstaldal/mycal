package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"
	"unicode"
)

// Store keeps only a SHA-256 digest of each random, 256-bit bearer secret.
type Store struct{ DB *sql.DB }

type Record struct {
	ID          int64   `json:"-"`
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   string  `json:"expires_at"`
	CalendarIDs []int64 `json:"calendar_ids"`
}

var ErrValidation = errors.New("invalid token request")

func invalid(message string) error { return fmt.Errorf("%w: %s", ErrValidation, message) }

func (s Store) Create(name string, expiry time.Time, calendars []int64) (Record, string, error) {
	var record Record
	if strings.TrimSpace(name) == "" || len(name) > 200 || !expiry.Truncate(time.Second).After(time.Now()) || len(calendars) == 0 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return record, "", invalid("name, future expiry, and at least one calendar are required")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return record, "", err
	}
	value := "mycal_" + base64.RawURLEncoding.EncodeToString(secret)
	digest := sha256.Sum256([]byte(value))
	tx, err := s.DB.Begin()
	if err != nil {
		return record, "", err
	}
	defer tx.Rollback()
	record = Record{Name: strings.TrimSpace(name), CreatedAt: time.Now().UTC().Format(time.RFC3339), ExpiresAt: expiry.UTC().Format(time.RFC3339), CalendarIDs: calendars}
	if _, err := tx.Exec(`DELETE FROM api_tokens WHERE expires_at <= ?`, record.CreatedAt); err != nil {
		return Record{}, "", err
	}
	base := slugify(record.Name)
	for number := 1; ; number++ {
		record.Slug = slugCandidate(base, number)
		err = tx.QueryRow(`INSERT INTO api_tokens(name,slug,token_hash,created_at,expires_at)
			VALUES(?,?,?,?,?) ON CONFLICT(slug) DO NOTHING RETURNING id`,
			record.Name, record.Slug, digest[:], record.CreatedAt, record.ExpiresAt).Scan(&record.ID)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return Record{}, "", err
		}
		break
	}
	seen := map[int64]bool{}
	for _, id := range calendars {
		if id < 0 || seen[id] {
			return Record{}, "", invalid("calendar IDs must be unique and nonnegative")
		}
		seen[id] = true
		var exists int
		if err := tx.QueryRow(`SELECT 1 FROM calendars WHERE id=?`, id).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Record{}, "", invalid("calendar not found")
			}
			return Record{}, "", err
		}
		if _, err := tx.Exec(`INSERT INTO api_token_calendars(token_id,calendar_id) VALUES(?,?)`, record.ID, id); err != nil {
			return Record{}, "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return Record{}, "", err
	}
	return record, value, nil
}

func (s Store) List() ([]Record, error) {
	rows, err := s.DB.Query(`SELECT t.id,t.slug,t.name,t.created_at,t.expires_at,f.calendar_id FROM api_tokens t LEFT JOIN api_token_calendars f ON f.token_id=t.id ORDER BY t.id DESC,f.calendar_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Record{}
	for rows.Next() {
		var r Record
		var calendarID sql.NullInt64
		if err := rows.Scan(&r.ID, &r.Slug, &r.Name, &r.CreatedAt, &r.ExpiresAt, &calendarID); err != nil {
			return nil, err
		}
		if len(items) == 0 || items[len(items)-1].ID != r.ID {
			r.CalendarIDs = []int64{}
			items = append(items, r)
		}
		if calendarID.Valid {
			items[len(items)-1].CalendarIDs = append(items[len(items)-1].CalendarIDs, calendarID.Int64)
		}
	}
	return items, rows.Err()
}

func (s Store) Revoke(slug string) (bool, error) {
	res, err := s.DB.Exec(`DELETE FROM api_tokens WHERE slug=?`, slug)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s Store) Validate(value string) (map[int64]bool, error) {
	_, allowed, err := s.validate(value)
	return allowed, err
}

func (s Store) validate(value string) (int64, map[int64]bool, error) {
	return validateWith(context.Background(), s.DB, value)
}

type tokenQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func validateWith(ctx context.Context, db tokenQueryer, value string) (int64, map[int64]bool, error) {
	if !strings.HasPrefix(value, "mycal_") || len(value) != 49 {
		return 0, nil, nil
	}
	digest := sha256.Sum256([]byte(value))
	// Fetch identity and grants in one statement, so ID reuse or revocation
	// between separate queries cannot substitute another token's calendars.
	rows, err := db.QueryContext(ctx, `SELECT t.id,t.expires_at,f.calendar_id
		FROM api_tokens t LEFT JOIN api_token_calendars f ON f.token_id=t.id
		WHERE t.token_hash=?`, digest[:])
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, nil, rows.Err()
	}
	var id int64
	var expiry string
	var calendarID sql.NullInt64
	if err := rows.Scan(&id, &expiry, &calendarID); err != nil {
		return 0, nil, err
	}
	until, err := time.Parse(time.RFC3339, expiry)
	if err != nil {
		return 0, nil, err
	}
	if !time.Now().Before(until) {
		return 0, nil, nil
	}
	allowed := map[int64]bool{}
	if calendarID.Valid {
		allowed[calendarID.Int64] = true
	}
	for rows.Next() {
		if err := rows.Scan(&id, &expiry, &calendarID); err != nil {
			return 0, nil, err
		}
		if calendarID.Valid {
			allowed[calendarID.Int64] = true
		}
	}
	return id, allowed, rows.Err()
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Management permits only the full-access caller. The outer Basic middleware
// authenticates it when Basic auth is configured.
func (s Store) Management(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Path == "/api/v1/tokens" {
		switch r.Method {
		case http.MethodGet:
			items, err := s.List()
			if err != nil {
				writeError(w, 500, "database error")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"total": len(items), "items": items})
		case http.MethodPost:
			var input struct {
				Name        string  `json:"name"`
				ExpiresAt   string  `json:"expires_at"`
				CalendarIDs []int64 `json:"calendar_ids"`
			}
			r.Body = http.MaxBytesReader(w, r.Body, 8192)
			dec := json.NewDecoder(r.Body)
			dec.DisallowUnknownFields()
			if err := dec.Decode(&input); err != nil {
				writeError(w, 400, "invalid token request")
				return
			}
			if err := dec.Decode(new(any)); err != io.EOF {
				writeError(w, 400, "invalid token request")
				return
			}
			expiry, err := time.Parse(time.RFC3339, input.ExpiresAt)
			if err != nil {
				writeError(w, 400, "invalid expiry")
				return
			}
			record, value, err := s.Create(input.Name, expiry, input.CalendarIDs)
			if err != nil {
				if errors.Is(err, ErrValidation) {
					writeError(w, 400, err.Error())
				} else {
					writeError(w, 500, "database error")
				}
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(struct {
				Record
				Token string `json:"token"`
			}{record, value})
		default:
			writeError(w, 405, "method not allowed")
		}
		return
	}
	if r.Method != http.MethodDelete {
		writeError(w, 405, "method not allowed")
		return
	}
	slug := strings.TrimPrefix(r.URL.Path, "/api/v1/tokens/")
	if !slugPattern.MatchString(slug) {
		writeError(w, 404, "token not found")
		return
	}
	ok, err := s.Revoke(slug)
	if err != nil {
		writeError(w, 500, "database error")
		return
	}
	if !ok {
		writeError(w, 404, "token not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func slugify(name string) string {
	s := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(norm.NFKD.String(name)), "-")
	s = strings.Trim(s, "-")
	// NFKD can expand a 200-byte name into a much longer ASCII slug.
	// Reserve 20 characters for the collision suffix within the API limit.
	if len(s) > 200 {
		s = strings.TrimRight(s[:200], "-")
	}
	if s == "" {
		s = "token"
	}
	return s
}
func slugCandidate(base string, number int) string {
	if number == 1 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, number)
}
