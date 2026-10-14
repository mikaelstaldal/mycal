// Command mcp is a stdio Model Context Protocol server for the MyCal API.
//
// It is an HTTP client of a running MyCal server, like mycal-cli, and exposes
// exactly the bearer-protected API routes an API token can reach. The contract
// is ../mysuite/spec/mcp-server.md; read it before changing what is exposed or
// how.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/mikaelstaldal/mycal/internal/clihttp"
	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const usage = `Usage: mycal-mcp [flags]

A Model Context Protocol server on stdin/stdout, for an MCP client to start as a
subprocess. It reads the MyCal API over HTTP, exposing what an API token can reach.

Flags:
  -help             Print help and exit successfully
  -url URL          Server base URL, including optional path (default MYCAL_URL or http://127.0.0.1:8080)
  -token-file PATH  Read API token from a file (default MYCAL_TOKEN_FILE)

Without -token-file or MYCAL_TOKEN_FILE, no Authorization header is sent.
HTTP is allowed only for literal loopback addresses; remote servers require HTTPS.
Redirects and environment HTTP proxies are disabled.

Resources:
  mycal://calendars                       the calendars the token can read
  mycal://events{?from,to,calendar_id*,limit,offset}
                                          events in a time range; from and to are required
  mycal://events/{id}                     one event, as JSON
  mycal://events/{id}/ics                 one event, as iCalendar
  mycal://calendar.ics{?calendar_id*}     whole calendars, as iCalendar

Dates accept RFC 3339, YYYY-MM-DD, or YYYY-MM-DD[T or space]HH:MM[:SS], the last
two in this process's local time zone.

Tools:
  search_events                           full-text search
`

// maxBody is the cap on every response body read, resource or tool
// (mcp-server.md §2.4). A body over it is refused, never truncated.
const maxBody = 1 << 20

const (
	eventsTemplate   = "mycal://events{?from,to,calendar_id*,limit,offset}"
	calendarTemplate = "mycal://calendar.ics{?calendar_id*}"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, "mycal-mcp:", err)
		os.Exit(1)
	}
}

// run parses the flags and serves MCP on transport until the client
// disconnects. stdout receives only -help output; the transport owns the
// process's real stdout.
func run(ctx context.Context, args []string, stdout io.Writer, transport mcp.Transport) error {
	c, help, err := configure(args)
	if help {
		_, err := io.WriteString(stdout, usage)
		return err
	}
	if err != nil {
		return err
	}
	return newServer(c).Run(ctx, transport)
}

// configure reads the flags and the token file, failing only on bad
// configuration. An unreachable server is reported per request instead.
func configure(args []string) (*client, bool, error) {
	flags := flag.NewFlagSet("mycal-mcp", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	base := flags.String("url", clihttp.URLDefault(), "server base URL")
	tokenFile := flags.String("token-file", os.Getenv("MYCAL_TOKEN_FILE"), "token file")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("%w\n%s", err, usage)
	}
	if flags.NArg() != 0 {
		return nil, false, fmt.Errorf("unexpected argument %q\n%s", flags.Arg(0), usage)
	}
	endpoint, err := clihttp.ParseBaseURL(*base)
	if err != nil {
		return nil, false, err
	}
	tokenFileProvided := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "token-file" {
			tokenFileProvided = true
		}
	})
	if tokenFileProvided && *tokenFile == "" {
		return nil, false, errors.New("-token-file must specify a nonempty path")
	}
	token := ""
	if *tokenFile != "" {
		if token, err = readToken(*tokenFile); err != nil {
			return nil, false, err
		}
	}
	return &client{base: endpoint, token: token, http: clihttp.NewClient()}, false, nil
}

// readToken applies mycal-cli's token-file rules.
func readToken(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("read token: %w", err)
	}
	defer func() { _ = file.Close() }()
	tokenBytes, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil {
		return "", fmt.Errorf("read token: %w", err)
	}
	if len(tokenBytes) > 4096 {
		return "", errors.New("token input is too large")
	}
	token := strings.TrimSpace(string(tokenBytes))
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", errors.New("token input must contain one token")
	}
	return token, nil
}

func version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	return "devel"
}

func newServer(c *client) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "mycal-mcp", Version: version()}, &mcp.ServerOptions{
		Instructions: "Read-only access to a MyCal server, limited to the calendars the API token was granted. " +
			"Start with mycal://calendars; list events in a time range with mycal://events?from=…&to=… " +
			"(dates such as 2026-10-01 are accepted); use the search_events tool to search by text.",
		// No subscriptions, no list_changed, no logging (mcp-server.md §2.2).
		Capabilities:      &mcp.ServerCapabilities{Resources: &mcp.ResourceCapabilities{}, Tools: &mcp.ToolCapabilities{}},
		CompletionHandler: c.complete,
	})
	server.AddResource(&mcp.Resource{URI: "mycal://calendars", Name: "calendars", MIMEType: "application/json",
		Description: "The calendars this token can read, with their IDs."}, c.read)
	server.AddResource(&mcp.Resource{URI: "mycal://calendar.ics", Name: "calendar-ics", MIMEType: "text/calendar",
		Description: "Every calendar this token can read, as one iCalendar feed."}, c.read)
	server.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: eventsTemplate, Name: "events", MIMEType: "application/json",
		Description: "Events overlapping [from, to), recurring events expanded, ordered by start. from and to are required; " +
			"calendar_id repeats to choose calendars; limit is 1-200 and offset skips, both optional; without limit, every event in the range from offset on is returned."}, c.read)
	server.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: "mycal://events/{id}", Name: "event", MIMEType: "application/json",
		Description: "One event. A recurring event's instance has the ID parentID_instanceStart, percent-encoded."}, c.read)
	server.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: "mycal://events/{id}/ics", Name: "event-ics", MIMEType: "text/calendar",
		Description: "One event as iCalendar."}, c.read)
	server.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: calendarTemplate, Name: "calendars-ics", MIMEType: "text/calendar",
		Description: "The chosen calendars as one iCalendar feed; calendar_id repeats."}, c.read)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_events",
		InputSchema: searchEventsSchema(),
		Description: "Search events by text, newest first when paged. Recurring events are not expanded. from and to, given together, restrict the time range.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(false)},
	}, c.searchEvents)
	return server
}

// client is an HTTP client of one MyCal server.
type client struct {
	base  *url.URL
	token string
	http  *http.Client
}

// apiError is a non-2xx response.
type apiError struct {
	status  int
	message string
}

func (e *apiError) Error() string { return e.message }

// get reads one API resource, refusing a body over maxBody.
func (c *client) get(ctx context.Context, resource string, query url.Values) (contentType string, body []byte, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, clihttp.APIURL(c.base, resource, query), nil)
	if err != nil {
		return "", nil, err
	}
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	request.Header.Set("Accept", "*/*")
	response, err := c.http.Do(request)
	if err != nil {
		return "", nil, errors.New(c.redact("request failed: " + err.Error()))
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", nil, c.statusError(response)
	}
	body, err = io.ReadAll(io.LimitReader(response.Body, maxBody+1))
	if err != nil {
		return "", nil, fmt.Errorf("read response: %w", err)
	}
	if len(body) > maxBody {
		return "", nil, fmt.Errorf("response exceeds the %d-byte (1 MiB) limit", maxBody)
	}
	// A server or proxy echoing the Authorization header must not put the
	// secret in front of the client (mcp-server.md §2.1). Error text is
	// redacted, but a successful body is refused whole: its bytes are passed
	// through unchanged or not at all.
	if c.containsToken(body) {
		return "", nil, errors.New("the response contains the API token, so it is not passed on")
	}
	return response.Header.Get("Content-Type"), body, nil
}

func (c *client) statusError(response *http.Response) error {
	switch {
	case response.StatusCode == http.StatusUnauthorized && c.token != "":
		return &apiError{response.StatusCode, "the API token is expired or revoked; run mycal-token login"}
	case response.StatusCode == http.StatusUnauthorized:
		return &apiError{response.StatusCode, "the server requires authentication; set MYCAL_TOKEN_FILE or pass -token-file"}
	case response.StatusCode == http.StatusForbidden:
		return &apiError{response.StatusCode, "outside this token's scope"}
	}
	// The error body is held to the same cap as any other: its error field is
	// carried whole or not at all, never cut short. The API's own errors use
	// `error`; ogen's request-decoding errors use `error_message`.
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxBody+1))
	message := fmt.Sprintf("HTTP %d", response.StatusCode)
	switch detail := errorField(raw); {
	case err != nil || len(raw) > maxBody:
	case detail != "":
		message += ": " + detail
	case len(raw) <= 1024 && utf8.Valid(raw) && strings.TrimSpace(string(raw)) != "" && !c.containsToken(raw):
		// Not the API's error shape, perhaps a proxy's; short enough to show whole.
		message += ": " + strings.TrimSpace(string(raw))
	}
	return &apiError{response.StatusCode, c.redact(message)}
}

// errorField returns the first of `error` and `error_message` that is a
// nonempty string. Each is decoded on its own, so a field of another type
// cannot hide the other.
func errorField(raw []byte) string {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return ""
	}
	for _, name := range []string{"error", "error_message"} {
		var value string
		if json.Unmarshal(fields[name], &value) == nil && value != "" {
			return value
		}
	}
	return ""
}

// containsToken reports whether body carries the token, either as its bytes
// or, in JSON, as a string that decodes to it: `\u0072…` is the same secret to
// anything that parses the body. Every string token is checked, keys and
// duplicate members included, since a decoded map keeps only the last of a
// repeated key.
func (c *client) containsToken(body []byte) bool {
	if c.token == "" {
		return false
	}
	if bytes.Contains(body, []byte(c.token)) {
		return true
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	for {
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		if text, ok := token.(string); ok && strings.Contains(text, c.token) {
			return true
		}
	}
}

// redact removes the token from text that came from elsewhere — a server or
// proxy echoing the Authorization header must not put it in front of the
// client (mcp-server.md §2.1).
func (c *client) redact(text string) string {
	if c.token == "" {
		return text
	}
	return strings.ReplaceAll(text, c.token, "[token]")
}

// refuse rejects a resource URI or tool call this server will not forward. It
// uses the internal-error code, not invalid params: go-sdk v1.8.0 gives
// "Resource not found" the invalid-params code, and a client that shows every
// -32602 as not found would hide the message (mcp-server.md §5.8).
func refuse(format string, args ...any) error {
	return &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: fmt.Sprintf(format, args...)}
}

// parseDateTime is mycal-cli's: it preserves explicit offsets and interprets
// unzoned input in this process's local time zone.
func parseDateTime(value string) (time.Time, error) {
	if stamp, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return stamp, nil
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		if stamp, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			// ParseInLocation can move a nonexistent local wall time across a DST gap.
			wall, _ := time.Parse(layout, value)
			const wallLayout = "2006-01-02T15:04:05.999999999"
			if stamp.Format(wallLayout) != wall.Format(wallLayout) {
				return time.Time{}, fmt.Errorf("local time %q does not exist in %s", value, time.Local)
			}
			return stamp, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date or timestamp %q", value)
}

// setRange converts from and to as mycal-cli does (mcp-server.md §2.3): both
// or neither, from before to.
func setRange(query url.Values, from, to string) error {
	if (from == "") != (to == "") {
		return errors.New("from and to must be given together")
	}
	if from == "" {
		return nil
	}
	start, err := parseDateTime(from)
	if err != nil {
		return fmt.Errorf("from: %w", err)
	}
	end, err := parseDateTime(to)
	if err != nil {
		return fmt.Errorf("to: %w", err)
	}
	if !start.Before(end) {
		return errors.New("from must precede to")
	}
	query.Set("from", start.Format(time.RFC3339Nano))
	query.Set("to", end.Format(time.RFC3339Nano))
	return nil
}

// target is the API request a resource URI names.
type target struct {
	resource string
	query    url.Values
}

// resolve maps a resource URI to its API request. The SDK has already chosen
// this server by matching a template, but its matcher neither extracts
// reordered or repeated query parameters nor rejects unknown ones, so the URI
// is parsed here (mcp-server.md §3).
func resolve(raw string) (target, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "mycal" || u.Opaque != "" || u.User != nil || u.Fragment != "" {
		return target{}, refuse("invalid resource URI %q", raw)
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return target{}, refuse("invalid query in %q", raw)
	}
	var segments []string
	if path := strings.TrimPrefix(u.EscapedPath(), "/"); path != "" {
		for _, escaped := range strings.Split(path, "/") {
			segment, err := url.PathUnescape(escaped)
			if err != nil || segment == "" {
				return target{}, refuse("invalid resource URI %q", raw)
			}
			segments = append(segments, segment)
		}
	}
	// only accepts the named query parameters; only calendar_id may repeat.
	only := func(t target, allowed ...string) (target, error) {
		if u.ForceQuery && u.RawQuery == "" {
			return target{}, refuse("%q has an empty query", raw)
		}
		for key, values := range query {
			if !slices.Contains(allowed, key) {
				if len(allowed) == 0 {
					return target{}, refuse("%q takes no query parameters", raw)
				}
				return target{}, refuse("unknown query parameter %q; use the search_events tool to search", key)
			}
			if key != "calendar_id" && len(values) != 1 {
				return target{}, refuse("%s may appear only once", key)
			}
		}
		t.query = query
		return t, nil
	}
	if len(segments) > 0 {
		id, start, err := model.ParseEventID(segments[0])
		if err == nil && start != "" {
			_, err = time.Parse(time.RFC3339, start)
		}
		if err != nil || id <= 0 {
			return target{}, refuse("%q: invalid event ID; a recurring instance's is parentID_instanceStart", raw)
		}
	}
	switch {
	case u.Host == "calendars" && len(segments) == 0:
		return only(target{resource: "/calendars"})
	case u.Host == "calendar.ics" && len(segments) == 0:
		return only(target{resource: "/events.ics"}, "calendar_id")
	case u.Host == "events" && len(segments) == 0:
		t, err := only(target{resource: "/events"}, "from", "to", "calendar_id", "limit", "offset")
		if err != nil {
			return target{}, err
		}
		if t.query.Get("from") == "" || t.query.Get("to") == "" {
			return target{}, refuse("%q needs from and to; use the search_events tool to search without a range", raw)
		}
		if err := setRange(t.query, t.query.Get("from"), t.query.Get("to")); err != nil {
			return target{}, refuse("%q: %v", raw, err)
		}
		return t, nil
	case u.Host == "events" && len(segments) == 1:
		return only(target{resource: "/events/" + url.PathEscape(segments[0])})
	case u.Host == "events" && len(segments) == 2 && segments[1] == "ics":
		return only(target{resource: "/events/" + url.PathEscape(segments[0]) + "/ics"})
	}
	return target{}, mcp.ResourceNotFoundError(raw)
}

func (c *client) read(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := request.Params.URI
	t, err := resolve(uri)
	if err != nil {
		return nil, err
	}
	contentType, body, err := c.get(ctx, t.resource, t.query)
	if err != nil {
		if apiErr, ok := errors.AsType[*apiError](err); ok && apiErr.status == http.StatusNotFound {
			return nil, mcp.ResourceNotFoundError(uri)
		}
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: err.Error()}
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = "application/octet-stream"
	}
	contents := &mcp.ResourceContents{URI: uri, MIMEType: mediaType}
	textual := strings.HasPrefix(mediaType, "text/") || mediaType == "application/json"
	// go-sdk v1.8.0 drops an empty text field from the wire but keeps an empty,
	// non-nil blob, so an empty body is always an empty blob (mcp-server.md §2.4).
	if textual && utf8.Valid(body) && len(body) > 0 {
		contents.Text = string(body)
	} else {
		contents.Blob = append([]byte{}, body...)
	}
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{contents}}, nil
}

type searchEventsInput struct {
	Q          string  `json:"q" jsonschema:"search text, 1-200 bytes"`
	From       string  `json:"from,omitempty" jsonschema:"start of the time range, given together with to: RFC 3339, YYYY-MM-DD, or YYYY-MM-DD HH:MM[:SS] in local time"`
	To         string  `json:"to,omitempty" jsonschema:"exclusive end of the time range, given together with from"`
	CalendarID []int64 `json:"calendar_id,omitempty" jsonschema:"calendar IDs to search; all the token's calendars when omitted"`
	Limit      *int    `json:"limit,omitempty" jsonschema:"page size, 1-200; without it, every match from offset on is returned"`
	Offset     *int    `json:"offset,omitempty" jsonschema:"number of results to skip"`
}

// searchEventsSchema is the inferred schema with calendar_id typed as an
// array: the inference allows null for any slice, and the contract types list
// arguments as JSON arrays (mcp-server.md §2.3). Unknown arguments stay
// refused.
func searchEventsSchema() *jsonschema.Schema {
	schema, err := jsonschema.For[searchEventsInput](nil)
	if err != nil {
		panic(err)
	}
	schema.Properties["calendar_id"].Types = nil
	schema.Properties["calendar_id"].Type = "array"
	return schema
}

func (c *client) searchEvents(ctx context.Context, _ *mcp.CallToolRequest, in searchEventsInput) (*mcp.CallToolResult, any, error) {
	query := url.Values{"q": {in.Q}}
	if err := setRange(query, in.From, in.To); err != nil {
		return nil, nil, err
	}
	for _, id := range in.CalendarID {
		query.Add("calendar_id", strconv.FormatInt(id, 10))
	}
	if in.Limit != nil {
		query.Set("limit", strconv.Itoa(*in.Limit))
	}
	if in.Offset != nil {
		query.Set("offset", strconv.Itoa(*in.Offset))
	}
	_, body, err := c.get(ctx, "/events", query)
	if err != nil {
		return nil, nil, err
	}
	// The text block is the server's bytes. The API answers with a JSON array,
	// and go-sdk v1.8.0 appends a second, re-serialised text block for any
	// structured output that is not an object, so there is no structured output
	// here (mcp-server.md §2.3).
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}, nil, nil
}

// complete offers calendar IDs for the calendar_id argument of the events and
// iCalendar templates, and nothing for any other argument (mcp-server.md
// §2.6).
func (c *client) complete(ctx context.Context, request *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	result := &mcp.CompleteResult{Completion: mcp.CompletionResultDetails{Values: []string{}}}
	ref := request.Params.Ref
	if ref == nil || ref.Type != "ref/resource" || (ref.URI != eventsTemplate && ref.URI != calendarTemplate) || request.Params.Argument.Name != "calendar_id" {
		return result, nil
	}
	_, body, err := c.get(ctx, "/calendars", nil)
	if err != nil {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: err.Error()}
	}
	var calendars []struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(body, &calendars); err != nil {
		return nil, &jsonrpc.Error{Code: jsonrpc.CodeInternalError, Message: "invalid calendar list from server"}
	}
	for _, calendar := range calendars {
		if id := strconv.FormatInt(calendar.ID, 10); strings.HasPrefix(id, request.Params.Argument.Value) {
			result.Completion.Values = append(result.Completion.Values, id)
		}
	}
	// The protocol caps a completion at 100 values.
	if len(result.Completion.Values) > 100 {
		result.Completion.Total = len(result.Completion.Values)
		result.Completion.HasMore = true
		result.Completion.Values = result.Completion.Values[:100]
	}
	return result, nil
}
