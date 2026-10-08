package token

import (
	"database/sql"
	"net/http"
	"strings"
)

// Bearer selects a dedicated read handler, bypassing Basic only for bearer
// requests. Authentication and all response reads share one SQLite snapshot.
// SQLite does not enforce TxOptions.ReadOnly: the method and route allowlist
// below is the write barrier; the snapshot is for consistent scoped reads.
func (s Store) Bearer(full http.Handler, scoped func(*sql.Tx, map[int64]bool) http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		scheme, secret, _ := strings.Cut(header, " ")
		if !strings.EqualFold(scheme, "Bearer") {
			full.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		tx, err := s.DB.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
		if err != nil {
			writeError(w, 500, "database error")
			return
		}
		defer tx.Rollback()
		_, allowed, err := validateWith(r.Context(), tx, strings.TrimSpace(secret))
		if err != nil {
			writeError(w, 500, "database error")
			return
		}
		if allowed == nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeError(w, 401, "unauthorized")
			return
		}
		path := r.URL.Path
		permitted := path == "/api/v1/calendars" || path == "/api/v1/events" || path == "/api/v1/events.ics" || path == "/calendar.ics"
		if strings.HasPrefix(path, "/api/v1/events/") {
			parts := strings.Split(strings.TrimPrefix(path, "/api/v1/events/"), "/")
			permitted = parts[0] != "" && (len(parts) == 1 || (len(parts) == 2 && parts[1] == "ics"))
		}
		if r.Method != http.MethodGet || !permitted {
			writeError(w, 403, "token cannot access this resource")
			return
		}
		scoped(tx, allowed).ServeHTTP(w, r)
	})
}
