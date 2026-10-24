package token

import (
	"context"
	"fmt"
	"github.com/mikaelstaldal/go-server-common/devicegrant"
	"testing"
	"time"

	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestCalendarScopeLimits(t *testing.T) {
	db, err := repository.OpenDB(t.TempDir()+"/test.sqlite", 5000)
	require.NoError(t, err)
	defer db.Close()
	store := Store{DB: db}
	ids := make([]int64, 101)
	scopes := make([]string, 101)
	for i := range ids {
		ids[i] = int64(i + 1)
		scopes[i] = fmt.Sprintf("calendar:%d", ids[i])
		_, err = db.Exec(`INSERT INTO calendars(id,name,color) VALUES(?,?, 'red')`, ids[i], scopes[i])
		require.NoError(t, err)
	}
	_, err = scopeIDs(scopes[:100])
	require.NoError(t, err)
	_, err = scopeIDs(scopes)
	require.ErrorContains(t, err, "at most 100")
	require.ErrorIs(t, store.DeviceGrant("").ValidateScope(context.Background(), nil, scopes), devicegrant.ErrInvalidGrant)
	_, _, err = store.Create("Reader", time.Now().Add(time.Hour), ids[:100])
	require.NoError(t, err)
	_, _, err = store.Create("Reader", time.Now().Add(time.Hour), ids)
	require.ErrorContains(t, err, "at most 100")
}
