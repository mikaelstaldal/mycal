package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mikaelstaldal/mycal/internal/handler"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/service"
	"github.com/mikaelstaldal/mycal/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestCalendarTokenAccess(t *testing.T) {
	db, err := repository.OpenDB(filepath.Join(t.TempDir(), "calendar.db"), 5000)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`INSERT INTO calendars(id,name) VALUES(1,'Work'),(2,'Private');
 INSERT INTO events(id,title,start_time,end_time,calendar_id,recurrence_freq) VALUES
 (1,'default visible','2026-10-01T10:00:00Z','2026-10-01T11:00:00Z',0,''),
 (2,'work visible','2026-10-01T10:00:00Z','2026-10-01T11:00:00Z',1,''),
 (3,'private secret','2026-10-01T10:00:00Z','2026-10-01T11:00:00Z',2,''),
 (4,'repeat visible','2026-10-01T10:00:00Z','2026-10-01T11:00:00Z',1,'DAILY');
 INSERT INTO events(id,title,start_time,end_time,calendar_id,recurrence_parent_id,recurrence_original_start) VALUES
 (5,'override secret','2026-10-02T10:00:00Z','2026-10-02T11:00:00Z',2,4,'2026-10-02T10:00:00Z')`)
	require.NoError(t, err)
	store := &token.Store{DB: db}
	record, secret, err := store.Create("reader", time.Now().Add(time.Hour), []int64{0, 1})
	require.NoError(t, err)
	digest := sha256.Sum256([]byte(secret))
	var stored []byte
	require.NoError(t, db.QueryRow(`SELECT token_hash FROM api_tokens WHERE id=?`, record.ID).Scan(&stored))
	assert.Equal(t, digest[:], stored)
	repo, err := repository.NewSQLiteRepository(db)
	require.NoError(t, err)
	mux := http.NewServeMux()
	mux.Handle("/api/v1/", handler.NewRouter(service.NewEventService(repo, repo), service.NewPreferencesService(repo), nil, service.NewCalendarService(repo)))
	mux.HandleFunc("/api/v1/tokens", store.Management)
	mux.HandleFunc("/api/v1/tokens/", store.Management)
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	require.NoError(t, err)
	authFile := filepath.Join(t.TempDir(), "htpasswd")
	require.NoError(t, os.WriteFile(authFile, append([]byte("user:"), hash...), 0600))
	h, err := buildHTTPHandler(mux, httpServerOptions{addr: "127.0.0.1", port: 8080, basicAuthFile: authFile, basicAuthRealm: "test", tokenStore: store})
	require.NoError(t, err)
	request := func(method, path, auth, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, strings.NewReader(body))
		if auth != "" {
			r.Header.Set("Authorization", auth)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	window := "from=2026-10-01T00:00:00Z&to=2026-10-04T00:00:00Z"
	for _, path := range []string{"/api/v1/calendars", "/api/v1/events?" + window, "/api/v1/events?q=visible", "/api/v1/events.ics", "/calendar.ics", "/api/v1/events/2", "/api/v1/events/2/ics", "/api/v1/events/4_2026-10-02T10:00:00Z", "/api/v1/events/4_2026-10-02T10:00:00Z/ics"} {
		t.Run(path, func(t *testing.T) {
			w := request("GET", path, "Bearer "+secret, "")
			require.Equal(t, 200, w.Code, w.Body.String())
			assert.NotContains(t, w.Body.String(), "secret")
			assert.NotContains(t, w.Body.String(), "Private")
			assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		})
	}
	for _, path := range []string{"/api/v1/events?" + window + "&calendar_id=2", "/api/v1/events?q=secret", "/api/v1/events?" + window + "&calendar=Private"} {
		w := request("GET", path, "Bearer "+secret, "")
		require.Equal(t, 200, w.Code)
		assert.JSONEq(t, "[]", w.Body.String())
	}
	for _, path := range []string{"/api/v1/events/3", "/api/v1/events/3/ics", "/api/v1/events/5"} {
		w := request("GET", path, "Bearer "+secret, "")
		assert.Equal(t, 404, w.Code)
		assert.NotContains(t, w.Body.String(), "secret")
	}
	for _, method := range []string{"POST", "PATCH", "DELETE", "PUT", "HEAD"} {
		assert.Equal(t, 403, request(method, "/api/v1/events/2", "Bearer "+secret, "{}").Code)
	}
	for _, path := range []string{"/", "/api/v1/preferences", "/api/v1/feeds", "/api/v1/tokens", "/api/v1/events/2/extra"} {
		assert.Equal(t, 403, request("GET", path, "Bearer "+secret, "").Code)
	}
	assert.Equal(t, 401, request("GET", "/api/v1/events?"+window, "", "").Code)
	assert.Equal(t, 401, request("GET", "/api/v1/calendars", "Bearer invalid", "").Code)
	basic := httptest.NewRequest("GET", "http://127.0.0.1:8080/api/v1/tokens", nil)
	basic.SetBasicAuth("user", "password")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, basic)
	require.Equal(t, 200, w.Code)
	assert.NotContains(t, w.Body.String(), secret)
	body := `{"name":"CLI","expires_at":"` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + `","calendar_ids":[0]}`
	created := request("POST", "/api/v1/tokens", basic.Header.Get("Authorization"), body)
	require.Equal(t, 201, created.Code, created.Body.String())
	var result struct {
		Token string `json:"token"`
		Slug  string `json:"slug"`
	}
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &result))
	assert.NotEmpty(t, result.Token)
	assert.Equal(t, 204, request("DELETE", "/api/v1/tokens/"+result.Slug, basic.Header.Get("Authorization"), "").Code)
	assert.Equal(t, 401, request("GET", "/api/v1/calendars", "Bearer "+result.Token, "").Code)
	_, err = db.Exec(`UPDATE api_tokens SET expires_at='2000-01-01T00:00:00Z' WHERE id=?`, record.ID)
	require.NoError(t, err)
	assert.Equal(t, 401, request("GET", "/api/v1/calendars", "Bearer "+secret, "").Code)
}

func TestTokenValidationAndMigration(t *testing.T) {
	db, err := repository.OpenDB(filepath.Join(t.TempDir(), "db"), 5000)
	require.NoError(t, err)
	defer db.Close()
	store := token.Store{DB: db}
	for _, ids := range [][]int64{nil, {}, {-1}, {0, 0}, {999}} {
		_, _, err := store.Create("reader", time.Now().Add(time.Hour), ids)
		require.Error(t, err)
	}
	for _, name := range []string{"", " ", strings.Repeat("x", 201), "bad\nname"} {
		_, _, err := store.Create(name, time.Now().Add(time.Hour), []int64{0})
		require.Error(t, err)
	}
	_, _, err = store.Create("reader", time.Now().Add(-time.Hour), []int64{0})
	require.Error(t, err)
	for _, slug := range []string{"reader", "reader-2"} {
		r, _, err := store.Create("reader", time.Now().Add(time.Hour), []int64{0})
		require.NoError(t, err)
		assert.Equal(t, slug, r.Slug)
	}
	items, err := store.List()
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestTokenReadUsesAuthorizationSnapshot(t *testing.T) {
	db, err := repository.OpenDB(filepath.Join(t.TempDir(), "snapshot.db"), 5000)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`INSERT INTO calendars(id,name) VALUES(1,'Granted'),(2,'Private'); INSERT INTO events(id,title,start_time,end_time,calendar_id) VALUES(1,'old authorized content','2026-10-01T10:00:00Z','2026-10-01T11:00:00Z',1)`)
	require.NoError(t, err)
	store := token.Store{DB: db}
	record, secret, err := store.Create("snapshot", time.Now().Add(time.Hour), []int64{1})
	require.NoError(t, err)
	h := store.Bearer(http.NotFoundHandler(), func(tx *sql.Tx, allowed map[int64]bool) http.Handler {
		// Another connection moves the event and revokes the token after authorization.
		_, err := db.Exec(`UPDATE events SET calendar_id=2,title='new private content' WHERE id=1; DELETE FROM api_tokens WHERE id=?`, record.ID)
		require.NoError(t, err)
		repo := &repository.ScopedRepository{SQLiteRepository: repository.NewSnapshotRepository(tx), Allowed: allowed}
		return handler.NewRouter(service.NewEventService(repo, repo), nil, nil, service.NewCalendarService(repo))
	})
	r := httptest.NewRequest("GET", "/api/v1/events/1", nil)
	r.Header.Set("Authorization", "Bearer "+secret)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "old authorized content")
	assert.NotContains(t, w.Body.String(), "private content")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assert.Equal(t, 401, w.Code)
}

func TestTokenMigrationPreservesExistingCalendarData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upgrade.db")
	db, err := repository.OpenDB(path, 5000)
	require.NoError(t, err)
	// Reconstruct a v2 database containing production calendar and event data.
	_, err = db.Exec(`DROP TABLE api_token_calendars; DROP TABLE api_tokens;
 PRAGMA user_version=2;
 INSERT INTO calendars(id,name) VALUES(1,'Existing calendar');
 INSERT INTO events(id,title,start_time,end_time,calendar_id,note_slug) VALUES(1,'Existing event','2026-10-01T10:00:00Z','2026-10-01T11:00:00Z',1,'existing-note'); INSERT INTO feeds(id,url,calendar_id) VALUES(1,'https://example.com/calendar.ics',1)`)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	db, err = repository.OpenDB(path, 5000)
	require.NoError(t, err)
	defer db.Close()
	var title, note, name string
	require.NoError(t, db.QueryRow(`SELECT e.title,e.note_slug,c.name FROM events e JOIN calendars c ON c.id=e.calendar_id WHERE e.id=1`).Scan(&title, &note, &name))
	assert.Equal(t, "Existing event", title)
	assert.Equal(t, "existing-note", note)
	assert.Equal(t, "Existing calendar", name)
	var feedURL string
	require.NoError(t, db.QueryRow(`SELECT url FROM feeds WHERE id=1`).Scan(&feedURL))
	assert.Equal(t, "https://example.com/calendar.ics", feedURL)
	store := token.Store{DB: db}
	_, value, err := store.Create("migrated", time.Now().Add(time.Hour), []int64{1})
	require.NoError(t, err)
	allowed, err := store.Validate(value)
	require.NoError(t, err)
	assert.Equal(t, map[int64]bool{1: true}, allowed)
}

func TestNormalizedTokenSlugFitsAPIContract(t *testing.T) {
	db, err := repository.OpenDB(filepath.Join(t.TempDir(), "slug.db"), 5000)
	require.NoError(t, err)
	defer db.Close()
	store := token.Store{DB: db}
	slugs := []string{}
	for i := 0; i < 2; i++ {
		record, _, err := store.Create(strings.Repeat("㎯", 66), time.Now().Add(time.Hour), []int64{0})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(record.Slug), 220)
		slugs = append(slugs, record.Slug)
	}
	assert.NotEqual(t, slugs[0], slugs[1])
	for _, slug := range slugs {
		found, err := store.Revoke(slug)
		require.NoError(t, err)
		assert.True(t, found)
	}
}
