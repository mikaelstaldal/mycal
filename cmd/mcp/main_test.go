package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mikaelstaldal/go-server-common/csrf"
	"github.com/mikaelstaldal/mycal/internal/handler"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/service"
	"github.com/mikaelstaldal/mycal/internal/token"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The token is granted calendar 1, Work. Event 3 is in calendar 2, Private,
// and so is an override (5) of the recurring Work event 4.
const seed = `INSERT INTO calendars(id,name) VALUES(1,'Work'),(2,'Private');
INSERT INTO events(id,title,start_time,end_time,calendar_id,recurrence_freq) VALUES
 (2,'Budget meeting','2026-10-01T10:00:00Z','2026-10-01T11:00:00Z',1,''),
 (3,'Private dentist','2026-10-01T12:00:00Z','2026-10-01T13:00:00Z',2,''),
 (4,'Daily standup','2026-10-01T08:00:00Z','2026-10-01T08:15:00Z',1,'DAILY');
INSERT INTO events(id,title,start_time,end_time,calendar_id,recurrence_parent_id,recurrence_original_start) VALUES
 (5,'Private standup','2026-10-02T08:00:00Z','2026-10-02T08:15:00Z',2,4,'2026-10-02T08:00:00Z')`

// fixture is a real MyCal API behind the real bearer middleware and token
// store — authorization is the server's, so the tests must not mock it.
type fixture struct {
	url      string
	store    *token.Store
	secret   string
	requests *recorder
}

// recorder keeps the method and path of every request the fixture answers,
// so the drift test sees what the MCP server actually requests rather than
// what it would resolve a URI to.
type recorder struct {
	mu       sync.Mutex
	requests []string
}

func (r *recorder) take() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	requests := r.requests
	r.requests = nil
	return requests
}

// newFixture serves the API as main's buildHTTPHandler wires it, without the
// public discovery routes (/openapi.yaml, /.well-known/api-catalog), which
// main serves ahead of the bearer middleware and this fixture does not. With
// basicAuth, a request without a bearer token gets 401, as from a server with
// Basic authentication configured; without it, such a request reaches the
// API, as from a server with none.
func newFixture(t *testing.T, basicAuth bool) *fixture {
	t.Helper()
	db, err := repository.OpenDB(filepath.Join(t.TempDir(), "calendar.db"), 5000)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(seed)
	require.NoError(t, err)
	f := &fixture{store: &token.Store{DB: db}, requests: &recorder{}}
	_, f.secret, err = f.store.Create("reader", time.Now().Add(time.Hour), []int64{1})
	require.NoError(t, err)
	repo, err := repository.NewSQLiteRepository(db)
	require.NoError(t, err)
	api := handler.NewRouter(service.NewEventService(repo, repo), service.NewPreferencesService(repo), nil, service.NewCalendarService(repo))
	mux := http.NewServeMux()
	token.MountDeviceGrant(mux, f.store.DeviceGrant("http://127.0.0.1/device-approval.html"))
	mux.HandleFunc("/api/v1/tokens", f.store.Management)
	mux.HandleFunc("/api/v1/tokens/", f.store.Management)
	mux.Handle("/api/v1/", api)
	mux.Handle("GET /calendar.ics", api)
	var full http.Handler = mux
	if basicAuth {
		full = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	}
	full = token.PublicDeviceAccess(full, mux)
	full = f.store.Bearer(full, func(tx *sql.Tx, allowed map[int64]bool) http.Handler {
		scoped := &repository.ScopedRepository{SQLiteRepository: repository.NewSnapshotRepository(tx), Allowed: allowed}
		return handler.NewRouter(service.NewEventService(scoped, scoped), nil, nil, service.NewCalendarService(scoped))
	})
	full = csrf.MiddlewareOrigins("http://example.test")(full)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests.mu.Lock()
		f.requests.requests = append(f.requests.requests, r.Method+" "+r.URL.EscapedPath())
		f.requests.mu.Unlock()
		full.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	f.url = server.URL
	return f
}

func (f *fixture) tokenFile(t *testing.T) string {
	path := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(path, []byte(f.secret+"\n"), 0o600))
	return path
}

// connect configures a client from flags, exactly as the binary does, and
// connects an MCP session to it in memory.
func connect(t *testing.T, args ...string) *mcp.ClientSession {
	t.Helper()
	t.Setenv("MYCAL_URL", "")
	t.Setenv("MYCAL_TOKEN_FILE", "")
	c, help, err := configure(args)
	require.NoError(t, err)
	require.False(t, help)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	serverSession, err := newServer(c).Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func read(t *testing.T, session *mcp.ClientSession, uri string) (*mcp.ResourceContents, error) {
	t.Helper()
	result, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		return nil, err
	}
	require.Len(t, result.Contents, 1)
	return result.Contents[0], nil
}

func call(t *testing.T, session *mcp.ClientSession, arguments map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "search_events", Arguments: arguments})
	require.NoError(t, err)
	return result
}

func callText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	return text.Text
}

// TestSurfaceMatchesTokenSurface is the drift test (mcp-server.md §4.1):
// every resource, template and tool requests only API operations a token may
// use, and every such operation is requested by one of them or excluded.
//
// Its bound: the operations a token may use are enumerated from openapi.yaml,
// probed through the real bearer middleware with IDs the token can reach. An
// operation openapi.yaml does not document is invisible to it.
func TestSurfaceMatchesTokenSurface(t *testing.T) {
	f := newFixture(t, true)
	raw, err := os.ReadFile("../../openapi.yaml")
	require.NoError(t, err)
	var document struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &document))
	parameter := regexp.MustCompile(`\\\{[a-z_0-9]+\\\}`)
	type operation struct {
		method  string
		pattern *regexp.Regexp
	}
	operations := map[string]operation{}
	permitted := []string{}
	for path, methods := range document.Paths {
		concrete := strings.NewReplacer("{id}", "2", "{user_code}", "ABCD-EFGH", "{slug}", "x").Replace(path)
		for method := range methods {
			method = strings.ToUpper(method)
			if !slices.Contains([]string{"GET", "PUT", "POST", "PATCH", "DELETE"}, method) {
				continue
			}
			key := method + " " + path
			operations[key] = operation{method, regexp.MustCompile("^" + parameter.ReplaceAllString(regexp.QuoteMeta(path), "[^/]+") + "$")}
			request, err := http.NewRequest(method, f.url+concrete, nil)
			require.NoError(t, err)
			request.Header.Set("Authorization", "Bearer "+f.secret)
			if path == "/api/v1/events" {
				request.URL.RawQuery = "from=2026-10-01T00:00:00Z&to=2026-10-05T00:00:00Z"
			}
			response, err := http.DefaultClient.Do(request)
			require.NoError(t, err)
			_ = response.Body.Close()
			if response.StatusCode != http.StatusForbidden {
				permitted = append(permitted, key)
			}
		}
	}
	require.NotEmpty(t, permitted)

	session := connect(t, "-url", f.url, "-token-file", f.tokenFile(t))
	ctx := context.Background()
	samples := map[string]string{
		"mycal://calendars":       "mycal://calendars",
		"mycal://calendar.ics":    "mycal://calendar.ics",
		eventsTemplate:            "mycal://events?from=2026-10-01&to=2026-10-05",
		"mycal://events/{id}":     "mycal://events/2",
		"mycal://events/{id}/ics": "mycal://events/2/ics",
		calendarTemplate:          "mycal://calendar.ics?calendar_id=1",
	}
	// Each tool's operation is pinned: a resource covering the same operation
	// would otherwise hide a tool that requests the wrong one.
	toolSamples := map[string]struct {
		arguments map[string]any
		operation string
	}{
		"search_events": {map[string]any{"q": "budget"}, "GET /api/v1/events"},
	}
	covered := []string{}
	// cover records the API operations one read or call requested.
	cover := func(what string) []string {
		requested := f.requests.take()
		require.NotEmpty(t, requested, "%s made no request", what)
		documented := []string{}
		for _, request := range requested {
			method, path, _ := strings.Cut(request, " ")
			matched := ""
			// A literal path wins over a parameterised one.
			for key, op := range operations {
				if op.method == method && op.pattern.MatchString(path) &&
					(matched == "" || strings.Count(key, "{") < strings.Count(matched, "{")) {
					matched = key
				}
			}
			require.NotEmpty(t, matched, "%s requested %s, which is not in openapi.yaml", what, request)
			covered = append(covered, matched)
			documented = append(documented, matched)
		}
		return documented
	}
	f.requests.take()
	listed := []string{}
	resources, err := session.ListResources(ctx, nil)
	require.NoError(t, err)
	for _, resource := range resources.Resources {
		listed = append(listed, resource.URI)
	}
	templates, err := session.ListResourceTemplates(ctx, nil)
	require.NoError(t, err)
	for _, template := range templates.ResourceTemplates {
		listed = append(listed, template.URITemplate)
	}
	for _, uri := range listed {
		sample, ok := samples[uri]
		require.True(t, ok, "no sample for %s", uri)
		_, err := read(t, session, sample)
		require.NoError(t, err, sample)
		cover(uri)
	}
	tools, err := session.ListTools(ctx, nil)
	require.NoError(t, err)
	for _, tool := range tools.Tools {
		sample, ok := toolSamples[tool.Name]
		require.True(t, ok, "no sample for tool %s", tool.Name)
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tool.Name, Arguments: sample.arguments})
		require.NoError(t, err)
		require.False(t, result.IsError, tool.Name)
		assert.Equal(t, []string{sample.operation}, cover(tool.Name), tool.Name)
	}

	// mcp-server.md §2.3 excludes /calendar.ics, an alias of
	// /api/v1/events.ics, and HEAD, which is not probed here. It also excludes
	// the public discovery routes, which are not in openapi.yaml and not in
	// this fixture.
	covered = append(covered, "GET /calendar.ics")
	slices.Sort(covered)
	covered = slices.Compact(covered)
	slices.Sort(permitted)
	assert.Equal(t, permitted, covered)
}

func TestResourcesReturnWhatTheAPIReturns(t *testing.T) {
	f := newFixture(t, true)
	session := connect(t, "-url", f.url, "-token-file", f.tokenFile(t))
	for _, tc := range []struct {
		uri, mime, contains string
	}{
		{"mycal://calendars", "application/json", `"name":"Work"`},
		{"mycal://events?from=2026-10-01&to=2026-10-03", "application/json", "Budget meeting"},
		{"mycal://events?to=2026-10-03T00%3A00%3A00Z&calendar_id=1&from=2026-10-01T00%3A00%3A00Z", "application/json", "Daily standup"},
		{"mycal://events/2", "application/json", "Budget meeting"},
		{"mycal://events/4_2026-10-03T08%3A00%3A00Z", "application/json", "Daily standup"},
		{"mycal://events/2/ics", "text/calendar", "SUMMARY:Budget meeting"},
		{"mycal://calendar.ics", "text/calendar", "SUMMARY:Daily standup"},
		{"mycal://calendar.ics?calendar_id=1", "text/calendar", "SUMMARY:Budget meeting"},
	} {
		contents, err := read(t, session, tc.uri)
		require.NoError(t, err, tc.uri)
		assert.Equal(t, tc.uri, contents.URI)
		assert.Equal(t, tc.mime, contents.MIMEType, tc.uri)
		assert.Contains(t, contents.Text, tc.contains, tc.uri)
		assert.Empty(t, contents.Blob, tc.uri)
	}

	// Paging, added to the API for this server: two pages of one, then none.
	titles := func(uri string) []string {
		contents, err := read(t, session, uri)
		require.NoError(t, err, uri)
		var events []struct {
			Title string `json:"title"`
		}
		require.NoError(t, json.Unmarshal([]byte(contents.Text), &events))
		out := []string{}
		for _, event := range events {
			out = append(out, event.Title)
		}
		return out
	}
	window := "from=2026-10-01T00%3A00%3A00Z&to=2026-10-02T00%3A00%3A00Z"
	assert.Equal(t, []string{"Daily standup"}, titles("mycal://events?"+window+"&limit=1"))
	assert.Equal(t, []string{"Budget meeting"}, titles("mycal://events?"+window+"&limit=1&offset=1"))
	assert.Empty(t, titles("mycal://events?"+window+"&limit=1&offset=2"))

	_, err := read(t, session, "mycal://events/999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Resource not found")
}

// TestScope: MyCal scopes by filtering, not refusing (mcp-server.md §2.3).
func TestScope(t *testing.T) {
	f := newFixture(t, true)
	session := connect(t, "-url", f.url, "-token-file", f.tokenFile(t))
	for _, uri := range []string{
		"mycal://calendars",
		"mycal://events?from=2026-10-01&to=2026-10-05",
		"mycal://events?from=2026-10-01&to=2026-10-05&calendar_id=2",
		"mycal://calendar.ics",
		"mycal://calendar.ics?calendar_id=2",
	} {
		contents, err := read(t, session, uri)
		require.NoError(t, err, uri)
		assert.NotContains(t, contents.Text, "Private", uri)
	}
	for _, uri := range []string{"mycal://events/3", "mycal://events/3/ics", "mycal://events/5"} {
		_, err := read(t, session, uri)
		require.Error(t, err, uri)
		assert.Contains(t, err.Error(), "Resource not found", uri)
	}
	result := call(t, session, map[string]any{"q": "Private"})
	assert.False(t, result.IsError)
	assert.Equal(t, "[]", strings.TrimSpace(callText(t, result)))
	result = call(t, session, map[string]any{"q": "dentist", "calendar_id": []int{2}})
	assert.False(t, result.IsError)
	assert.NotContains(t, callText(t, result), "dentist")
}

func TestSearchEventsTool(t *testing.T) {
	f := newFixture(t, true)
	session := connect(t, "-url", f.url, "-token-file", f.tokenFile(t))
	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, tools.Tools, 1)
	tool := tools.Tools[0]
	require.NotNil(t, tool.Annotations)
	assert.True(t, tool.Annotations.ReadOnlyHint)
	require.NotNil(t, tool.Annotations.OpenWorldHint)
	assert.False(t, *tool.Annotations.OpenWorldHint)
	schema, err := json.Marshal(tool.InputSchema)
	require.NoError(t, err)
	var parsed struct {
		Required   []string `json:"required"`
		Properties map[string]struct {
			Type any `json:"type"`
		} `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(schema, &parsed))
	assert.Equal(t, []string{"q"}, parsed.Required)
	assert.Equal(t, "array", parsed.Properties["calendar_id"].Type)

	result := call(t, session, map[string]any{"q": "budget"})
	assert.False(t, result.IsError)
	assert.Contains(t, callText(t, result), "Budget meeting")
	assert.Nil(t, result.StructuredContent, "the API's array is not structured output (mcp-server.md §2.3)")

	result = call(t, session, map[string]any{"q": "standup", "from": "2026-10-01", "to": "2026-10-05", "calendar_id": []int{1}, "limit": 1})
	assert.False(t, result.IsError)
	assert.Contains(t, callText(t, result), "Daily standup")

	result = call(t, session, map[string]any{"q": "budget", "from": "2026-11-01", "to": "2026-11-02"})
	assert.False(t, result.IsError)
	assert.NotContains(t, callText(t, result), "Budget meeting", "the range is forwarded")

	for _, arguments := range []map[string]any{
		{"q": "x", "from": "2026-10-01"},
		{"q": "x", "from": "2026-10-05", "to": "2026-10-01"},
		{"q": "x", "from": "yesterday", "to": "2026-10-01"},
	} {
		result := call(t, session, arguments)
		assert.True(t, result.IsError, "%v", arguments)
	}
	result = call(t, session, map[string]any{"q": "x", "limit": 1000})
	assert.True(t, result.IsError)
	assert.Contains(t, callText(t, result), "HTTP 400: ")
}

func TestArgumentsAreValidated(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1) }))
	defer server.Close()
	session := connect(t, "-url", server.URL)
	for _, arguments := range []map[string]any{
		{},
		{"q": "x", "calendar": "Work"},
		{"q": "x", "calendar_id": 1},
		{"q": "x", "from": "2026-10-01"},
	} {
		assert.True(t, call(t, session, arguments).IsError, "%v", arguments)
	}
	assert.Zero(t, requests.Load(), "a refused call makes no HTTP request")
}

func TestQueryParametersAreParsedHere(t *testing.T) {
	var requests atomic.Int32
	var last atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		last.Store(r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	session := connect(t, "-url", server.URL)

	_, err := read(t, session, "mycal://events?calendar_id=2&to=2026-10-02T00%3A00%3A00Z&calendar_id=1&from=2026-10-01T00%3A00%3A00Z&limit=5")
	require.NoError(t, err)
	assert.Equal(t, "calendar_id=2&calendar_id=1&from=2026-10-01T00%3A00%3A00Z&limit=5&to=2026-10-02T00%3A00%3A00Z", last.Load())
	_, err = read(t, session, "mycal://events?from=2026-10-01&to=2026-10-02")
	require.NoError(t, err)
	local := func(date string) string {
		stamp, err := time.ParseInLocation("2006-01-02", date, time.Local)
		require.NoError(t, err)
		return stamp.Format(time.RFC3339Nano)
	}
	assert.Contains(t, last.Load(), "from="+strings.ReplaceAll(strings.ReplaceAll(local("2026-10-01"), ":", "%3A"), "+", "%2B"), "a date is mycal-cli's local midnight")

	before := requests.Load()
	for uri, want := range map[string]struct {
		message string
		code    int64
	}{
		// Refused here with the internal-error code (mcp-server.md §5.8).
		"mycal://events?from=2026-10-01&to=2026-10-02&q=x": {"use the search_events tool", jsonrpc.CodeInternalError},
		"mycal://events?from=2026-10-01":                   {"needs from and to", jsonrpc.CodeInternalError},
		"mycal://events?from=2026-10-02&to=2026-10-01":     {"from must precede to", jsonrpc.CodeInternalError},
		"mycal://events?from=2026-10-01&to=x":              {"invalid date", jsonrpc.CodeInternalError},
		"mycal://events?from=a&from=b&to=c":                {"only once", jsonrpc.CodeInternalError},
		"mycal://calendar.ics?calendar=Work":               {"unknown query parameter", jsonrpc.CodeInternalError},
		"mycal://events/abc":                               {"invalid event ID", jsonrpc.CodeInternalError},
		"mycal://events/4_tomorrow":                        {"invalid event ID", jsonrpc.CodeInternalError},
		// No template takes a query here, so the SDK's dispatch refuses it.
		"mycal://calendars?limit=1": {"Resource not found", jsonrpc.CodeInvalidParams},
		"mycal://events/2?x=1":      {"Resource not found", jsonrpc.CodeInvalidParams},
	} {
		_, err := read(t, session, uri)
		require.Error(t, err, uri)
		assert.Contains(t, err.Error(), want.message, uri)
		var wire *jsonrpc.Error
		require.ErrorAs(t, err, &wire, uri)
		assert.Equal(t, want.code, wire.Code, uri)
	}
	assert.Equal(t, before, requests.Load(), "a refused URI makes no HTTP request")
}

func TestResponseCap(t *testing.T) {
	size := maxBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".ics") {
			w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
			_, _ = w.Write(bytes.Repeat([]byte("c"), size))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`["` + strings.Repeat("p", size-len(`[""]`)) + `"]`))
	}))
	defer server.Close()
	session := connect(t, "-url", server.URL)
	search := func() *mcp.CallToolResult { return call(t, session, map[string]any{"q": "x"}) }

	contents, err := read(t, session, "mycal://calendar.ics")
	require.NoError(t, err)
	assert.Len(t, contents.Text, maxBody)
	assert.False(t, search().IsError)

	size = maxBody + 1
	_, err = read(t, session, "mycal://calendar.ics")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 MiB")
	result := search()
	assert.True(t, result.IsError)
	assert.Contains(t, callText(t, result), "1 MiB")
}

func TestAuthenticationErrors(t *testing.T) {
	f := newFixture(t, true)

	session := connect(t, "-url", f.url)
	_, err := read(t, session, "mycal://calendars")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires authentication; set MYCAL_TOKEN_FILE")

	tokenFile := f.tokenFile(t)
	records, err := f.store.List()
	require.NoError(t, err)
	_, err = f.store.Revoke(records[0].Slug)
	require.NoError(t, err)
	session = connect(t, "-url", f.url, "-token-file", tokenFile)
	_, err = read(t, session, "mycal://calendars")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired or revoked; run mycal-token login")
	assert.NotContains(t, err.Error(), f.secret)
	result := call(t, session, map[string]any{"q": "x"})
	assert.True(t, result.IsError)
	assert.Contains(t, callText(t, result), "expired or revoked")
	assert.NotContains(t, callText(t, result), f.secret)
}

func TestUnauthenticatedAgainstOpenServer(t *testing.T) {
	f := newFixture(t, false)
	session := connect(t, "-url", f.url)
	contents, err := read(t, session, "mycal://events/3")
	require.NoError(t, err)
	assert.Contains(t, contents.Text, "Private dentist")
}

func TestServerDown(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	session := connect(t, "-url", server.URL)
	_, err := read(t, session, "mycal://calendars")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
	assert.True(t, call(t, session, map[string]any{"q": "x"}).IsError)
}

func TestServerErrors(t *testing.T) {
	const secret = "review-test-secret"
	longError := strings.Repeat("e", 5000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/calendars":
			// A server or proxy echoing the credential back.
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"` + r.Header.Get("Authorization") + `"}`))
		case "/api/v1/events/1":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("upstream saw " + r.Header.Get("Authorization")))
		case "/api/v1/events/2":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"` + longError + `"}`))
		case "/api/v1/events/3":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"` + strings.Repeat("h", maxBody) + `"}`))
		case "/api/v1/events/4":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error_message":"decode params"}`))
		case "/api/v1/events/5":
			// A successful body echoing the credential, JSON-escaped.
			_, _ = w.Write([]byte(`{"title":"\u0072eview-test-secret"}`))
		case "/api/v1/events/6/ics":
			w.Header().Set("Content-Type", "text/calendar")
			_, _ = w.Write([]byte("SUMMARY:" + r.Header.Get("Authorization")))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"failed for ` + r.Header.Get("Authorization") + `"}`))
		}
	}))
	defer server.Close()
	tokenFile := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenFile, []byte(secret), 0o600))
	session := connect(t, "-url", server.URL, "-token-file", tokenFile)

	for uri, want := range map[string]string{
		"mycal://calendars":    "HTTP 500: Bearer [token]",
		"mycal://events/1":     "HTTP 502",
		"mycal://events/2":     "HTTP 500: " + longError,
		"mycal://events/4":     "HTTP 400: decode params",
		"mycal://events/5":     "contains the API token",
		"mycal://events/6/ics": "contains the API token",
	} {
		_, err := read(t, session, uri)
		require.Error(t, err, uri)
		assert.Contains(t, err.Error(), want, uri)
		assert.NotContains(t, err.Error(), secret, uri)
	}
	_, err := read(t, session, "mycal://events/3")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "hhhh", "an error body over the cap is not shown, not even in part")

	result := call(t, session, map[string]any{"q": "x"})
	assert.True(t, result.IsError)
	assert.Contains(t, callText(t, result), "failed for Bearer [token]")
	assert.NotContains(t, callText(t, result), secret)
}

func TestErrorField(t *testing.T) {
	for body, want := range map[string]string{
		`{"error":"primary"}`:                             "primary",
		`{"error_message":"secondary"}`:                   "secondary",
		`{"error":"primary","error_message":"secondary"}`: "primary",
		`{"error":"primary","error_message":123}`:         "primary",
		`{"error":123,"error_message":"secondary"}`:       "secondary",
		`{"message":"neither"}`:                           "",
		`not json`:                                        "",
	} {
		assert.Equal(t, want, errorField([]byte(body)), body)
	}
}

func TestSearchJSONIsPassedThrough(t *testing.T) {
	const body = " [ { \"title\" : \"x\" } ]\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	session := connect(t, "-url", server.URL)
	result := call(t, session, map[string]any{"q": "x"})
	assert.Equal(t, body, callText(t, result), "one text block, the server's bytes")
}

// TestBlobCap: MyCal's resources are all text, so a blob is a response that
// is not, here an iCalendar feed served as application/octet-stream.
func TestBlobCap(t *testing.T) {
	size := maxBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(bytes.Repeat([]byte{0x89}, size))
	}))
	defer server.Close()
	session := connect(t, "-url", server.URL)
	contents, err := read(t, session, "mycal://calendar.ics")
	require.NoError(t, err)
	assert.Len(t, contents.Blob, maxBody)
	assert.Empty(t, contents.Text)
	size = maxBody + 1
	_, err = read(t, session, "mycal://calendar.ics")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 MiB")
}

func TestEmptyBodyIsAnEmptyBlob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
	}))
	defer server.Close()
	session := connect(t, "-url", server.URL)
	contents, err := read(t, session, "mycal://calendar.ics")
	require.NoError(t, err)
	assert.Equal(t, "text/calendar", contents.MIMEType)
	assert.NotNil(t, contents.Blob)
	assert.Empty(t, contents.Blob)
}

func TestCompletion(t *testing.T) {
	f := newFixture(t, true)
	session := connect(t, "-url", f.url, "-token-file", f.tokenFile(t))
	complete := func(uri, argument, value string) []string {
		result, err := session.Complete(context.Background(), &mcp.CompleteParams{
			Ref:      &mcp.CompleteReference{Type: "ref/resource", URI: uri},
			Argument: mcp.CompleteParamsArgument{Name: argument, Value: value},
		})
		require.NoError(t, err)
		return result.Completion.Values
	}
	assert.Equal(t, []string{"1"}, complete(eventsTemplate, "calendar_id", ""), "only the token's calendars")
	assert.Equal(t, []string{"1"}, complete(calendarTemplate, "calendar_id", "1"))
	assert.Empty(t, complete(eventsTemplate, "calendar_id", "2"))
	assert.Empty(t, complete(eventsTemplate, "from", ""))
	assert.Empty(t, complete("mycal://events/{id}", "id", ""))
}

func TestCapabilities(t *testing.T) {
	session := connect(t)
	initialized := session.InitializeResult()
	assert.Equal(t, "mycal-mcp", initialized.ServerInfo.Name)
	capabilities := initialized.Capabilities
	require.NotNil(t, capabilities.Resources)
	assert.False(t, capabilities.Resources.Subscribe)
	assert.False(t, capabilities.Resources.ListChanged)
	require.NotNil(t, capabilities.Tools)
	assert.False(t, capabilities.Tools.ListChanged)
	assert.NotNil(t, capabilities.Completions)
	assert.Nil(t, capabilities.Prompts)
	// Logging is deprecated in the SDK's Go API, so check the wire form.
	wire, err := json.Marshal(capabilities)
	require.NoError(t, err)
	assert.NotContains(t, string(wire), `"logging"`)
}

func TestStartup(t *testing.T) {
	t.Setenv("MYCAL_URL", "")
	t.Setenv("MYCAL_TOKEN_FILE", "")
	var stdout bytes.Buffer
	require.NoError(t, run(context.Background(), []string{"-help"}, &stdout, nil))
	assert.Contains(t, stdout.String(), "Usage: mycal-mcp")

	empty := filepath.Join(t.TempDir(), "empty")
	require.NoError(t, os.WriteFile(empty, nil, 0o600))
	for _, args := range [][]string{
		{"-url", "http://example.com"},
		{"-url", "not a url"},
		{"-token-file", filepath.Join(t.TempDir(), "missing")},
		{"-token-file", empty},
		{"-token-file", ""},
		{"extra"},
	} {
		stdout.Reset()
		err := run(context.Background(), args, &stdout, nil)
		assert.Error(t, err, fmt.Sprint(args))
		assert.Empty(t, stdout.String(), fmt.Sprint(args))
	}

	t.Setenv("MYCAL_TOKEN_FILE", empty)
	_, _, err := configure(nil)
	assert.Error(t, err, "the environment's token file is read too")
}

// TestTokenHiddenInJSON: a token JSON-escaped in an earlier duplicate member,
// which a decoded map would drop, and in an error body shown as it is.
func TestTokenHiddenInJSON(t *testing.T) {
	const secret = "review-test-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/events/7":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"x":"\u0072eview-test-secret","x":"safe"}`))
		case "/api/v1/events/8":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"message":"Bearer \u0072eview-test-secret"}`))
		}
	}))
	defer server.Close()
	tokenFile := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenFile, []byte(secret), 0o600))
	session := connect(t, "-url", server.URL, "-token-file", tokenFile)
	_, err := read(t, session, "mycal://events/7")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contains the API token")
	_, err = read(t, session, "mycal://events/8")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 502")
	assert.NotContains(t, err.Error(), "eview-test-secret", "the body is not shown")
}
