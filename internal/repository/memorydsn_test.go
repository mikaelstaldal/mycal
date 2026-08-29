package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryDSNNamesDoNotCollide asserts the property MemoryDSN exists to provide:
// distinct names get distinct databases. A shared-cache database is identified by
// its name, so any two names mapping onto one DSN would be handed the same
// database — the failure this helper was written to remove.
//
// The names below all differ only in characters that have to be escaped. One
// earlier version replaced each of them with '_' and collapsed every one of these
// pairs; a later one escaped runes rather than bytes and collapsed the invalid
// UTF-8 pair. Asserted as distinctness rather than against expected strings, so
// the encoding can change — as it has, twice — without rewriting the test.
//
// A finite corpus cannot establish injectivity. It can hold the cases we know
// about, which is what this is.
func TestMemoryDSNNamesDoNotCollide(t *testing.T) {
	names := []string{
		"TestX/a-b",
		"TestX/a_b",
		"TestX/a b",
		"TestX/a.b",
		"TestX/a%b",
		"TestX/a/b",
		"TestX_a_b",
		"TestX/ab",
		"TestX/aåb",
		"TestX/a__b",
		// Invalid UTF-8. Ranging over a string decodes both of these to
		// utf8.RuneError, so a rune-wise encoding maps them onto one name.
		"TestX/a" + string([]byte{0xff}),
		"TestX/a" + string([]byte{0xfe}),
	}

	seen := make(map[string]string, len(names))
	for _, name := range names {
		dsn := MemoryDSN(name)
		if prev, ok := seen[dsn]; ok {
			t.Errorf("MemoryDSN(%q) and MemoryDSN(%q) both give %q", name, prev, dsn)
		}
		seen[dsn] = name
	}
}

// TestMemoryDSNDatabasesAreIndependent is the same property observed rather than
// computed: two names that the previous normalisation collapsed must give two
// databases, not one shared between them.
func TestMemoryDSNDatabasesAreIndependent(t *testing.T) {
	first, err := OpenDB(MemoryDSN(t.Name()+"/a-b"), 0)
	require.NoError(t, err)
	defer first.Close()

	second, err := OpenDB(MemoryDSN(t.Name()+"/a_b"), 0)
	require.NoError(t, err)
	defer second.Close()

	_, err = first.Exec(
		`INSERT INTO events (title, start_time, end_time) VALUES ('only in the first', '2026-01-01T00:00:00Z', '2026-01-01T01:00:00Z')`)
	require.NoError(t, err)

	var n int
	require.NoError(t, second.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&n))
	assert.Zero(t, n, "the second database must not see the first one's rows")
}
