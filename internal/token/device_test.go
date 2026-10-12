package token

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mikaelstaldal/go-server-common/devicegrant"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Consent belongs to the resource approved, even when an identifier is reused.
func TestApprovedDeviceScopeCannotFollowReusedResourceID(t *testing.T) {
	db, err := repository.OpenDB(t.TempDir()+"/test.sqlite", 5000)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`INSERT INTO calendars(id,name,color) VALUES(104,'Keep','red'),(105,'Original','blue')`)
	require.NoError(t, err)
	store := Store{DB: db}
	mux := http.NewServeMux()
	MountDeviceGrant(mux, store.DeviceGrant("https://example.test/"))
	send := func(path, body, content string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Content-Type", content)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	approved := func(id int64) devicegrant.Authorization {
		w := send("/api/v1/oauth/device_authorization", url.Values{"client_id": {"mycal-token"}, "name": {"Reader"}}.Encode(), "application/x-www-form-urlencoded")
		require.Equal(t, 200, w.Code)
		var a devicegrant.Authorization
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &a))
		w = send("/api/v1/oauth/device/"+a.UserCode, fmt.Sprintf(`{"approve":true,"scope":["calendar:%d"],"lifetime":"30d"}`, id), "application/json")
		require.Equal(t, 200, w.Code)
		return a
	}
	removed, kept := approved(105), approved(104)
	_, err = db.Exec(`DELETE FROM calendars WHERE id=105`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO calendars(id,name,color) VALUES(105,'Replacement','blue')`)
	require.NoError(t, err)
	poll := func(a devicegrant.Authorization) *httptest.ResponseRecorder {
		return send("/api/v1/oauth/token", url.Values{"client_id": {"mycal-token"}, "grant_type": {devicegrant.GrantType}, "device_code": {a.DeviceCode}}.Encode(), "application/x-www-form-urlencoded")
	}
	w := poll(removed)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_grant")
	w = poll(kept)
	require.Equal(t, 200, w.Code)
	var issued devicegrant.Token
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &issued))
	allowed, err := store.Validate(issued.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, map[int64]bool{104: true}, allowed)
}
