// Command cli is the script-oriented client for the MyCal API.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mikaelstaldal/mycal/internal/clihttp"
	"github.com/mikaelstaldal/mycal/internal/model"
)

const usage = `Usage: mycal-cli [global flags] <command> [command flags]

Global flags (before the command):
  -url URL          Server base URL, including optional path (default MYCAL_URL or http://127.0.0.1:8080)
  -token-file PATH  Read API token from a file (default MYCAL_TOKEN_FILE)
  -token-stdin      Read API token from standard input

Token flags are optional and mutually exclusive; either overrides MYCAL_TOKEN_FILE.
Without a token flag or MYCAL_TOKEN_FILE, no Authorization header is sent.

Commands:
  calendars list
  events list -from DATE -to DATE [-calendars ID[,ID...]]
  events search -q TEXT [-from DATE -to DATE] [-calendars ID[,ID...]]
  events get ID
  events ics ID
  calendars ics [-calendars ID[,ID...]]

JSON and iCalendar responses go to stdout unchanged; errors go to stderr.
Dates accept RFC3339, YYYY-MM-DD, or YYYY-MM-DD[T or space]HH:MM[:SS].
Missing time defaults to 00:00:00; missing timezone uses the local timezone.
During a repeated daylight-saving hour, use an explicit offset to select the time.
Recurrence timestamps in event IDs must include a timezone and are sent unchanged.
Calendar 0 is the default calendar. Event IDs can include recurrence timestamps.
HTTP is allowed only for literal loopback addresses; remote servers require HTTPS.
Redirects and environment HTTP proxies are disabled.
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mycal-cli:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("mycal-cli", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	base := flags.String("url", clihttp.URLDefault(), "server base URL")
	tokenFile := flags.String("token-file", os.Getenv("MYCAL_TOKEN_FILE"), "token file")
	tokenStdin := flags.Bool("token-stdin", false, "read token from stdin")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = io.WriteString(stdout, usage)
			return nil
		}
		return fmt.Errorf("%w\n%s", err, usage)
	}
	command := flags.Args()
	if len(command) == 0 || command[0] == "help" {
		_, _ = io.WriteString(stdout, usage)
		return nil
	}
	endpoint, err := clihttp.ParseBaseURL(*base)
	if err != nil {
		return err
	}
	method, path, query, err := parseCommand(command)
	if err != nil {
		return err
	}
	tokenFileProvided := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "token-file" {
			tokenFileProvided = true
		}
	})
	if tokenFileProvided && *tokenStdin {
		return errors.New("specify at most one of -token-file or -token-stdin")
	}
	if tokenFileProvided && *tokenFile == "" {
		return errors.New("-token-file must specify a nonempty path")
	}
	token := ""
	if *tokenFile != "" || *tokenStdin {
		tokenReader := stdin
		if !*tokenStdin {
			file, openErr := os.Open(*tokenFile)
			if openErr != nil {
				return fmt.Errorf("read token: %w", openErr)
			}
			defer func() { _ = file.Close() }()
			tokenReader = file
		}
		tokenBytes, err := io.ReadAll(io.LimitReader(tokenReader, 4097))
		if err != nil {
			return fmt.Errorf("read token: %w", err)
		}
		if len(tokenBytes) > 4096 {
			return errors.New("token input is too large")
		}
		token = strings.TrimSpace(string(tokenBytes))
		if token == "" || strings.ContainsAny(token, " \t\r\n") {
			return errors.New("token input must contain one token")
		}
	}
	request, err := http.NewRequest(method, clihttp.APIURL(endpoint, path, query), nil)
	if err != nil {
		return err
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	request.Header.Set("Accept", "application/json")
	client := clihttp.NewClient()
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	_, err = io.Copy(stdout, response.Body)
	return err
}

func parseCommand(args []string) (string, string, url.Values, error) {
	bad := func() (string, string, url.Values, error) {
		return "", "", nil, fmt.Errorf("invalid command or arguments\n%s", usage)
	}
	if len(args) < 2 {
		return bad()
	}
	q := url.Values{}
	if args[0] == "calendars" && args[1] == "list" && len(args) == 2 {
		return http.MethodGet, "/calendars", q, nil
	}
	if args[0] == "events" && (args[1] == "get" || args[1] == "ics") {
		if len(args) != 3 {
			return bad()
		}
		if id, start, err := model.ParseEventID(args[2]); err != nil || id <= 0 {
			return "", "", nil, errors.New("invalid event ID")
		} else if start != "" {
			if _, err := time.Parse(time.RFC3339, start); err != nil {
				return "", "", nil, errors.New("invalid recurrence timestamp")
			}
		}
		path := "/events/" + url.PathEscape(args[2])
		if args[1] == "ics" {
			path += "/ics"
		}
		return http.MethodGet, path, q, nil
	}
	path := "/events"
	if args[0] == "calendars" && args[1] == "ics" {
		path = "/events.ics"
	} else if args[0] != "events" || (args[1] != "list" && args[1] != "search") {
		return bad()
	}
	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	calendars := fs.String("calendars", "", "comma-separated calendar IDs")
	from := fs.String("from", "", "inclusive date or timestamp")
	to := fs.String("to", "", "exclusive date or timestamp")
	search := fs.String("q", "", "search text")
	if err := fs.Parse(args[2:]); err != nil {
		return "", "", nil, err
	}
	if fs.NArg() != 0 {
		return bad()
	}
	if path == "/events.ics" && (*from != "" || *to != "" || *search != "") {
		return bad()
	}
	if args[1] == "list" && (*from == "" || *to == "") {
		return "", "", nil, errors.New("list requires -from and -to")
	}
	if args[1] == "search" {
		if (*from == "") != (*to == "") {
			return "", "", nil, errors.New("search requires both -from and -to, or neither")
		}
		if strings.TrimSpace(*search) == "" || len(*search) > 200 {
			return "", "", nil, errors.New("search requires -q of 1–200 bytes")
		}
		q.Set("q", *search)
	} else if *search != "" {
		return bad()
	}
	for key, value := range map[string]string{"from": *from, "to": *to} {
		if value != "" {
			stamp, err := parseDateTime(value)
			if err != nil {
				return "", "", nil, fmt.Errorf("-%s: %w", key, err)
			}
			q.Set(key, stamp.Format(time.RFC3339Nano))
		}
	}
	if *from != "" && *to != "" {
		start, _ := time.Parse(time.RFC3339Nano, q.Get("from"))
		end, _ := time.Parse(time.RFC3339Nano, q.Get("to"))
		if !start.Before(end) {
			return "", "", nil, errors.New("-from must precede -to")
		}
	}
	if *calendars != "" {
		seen := map[int64]bool{}
		for _, value := range strings.Split(*calendars, ",") {
			id, err := strconv.ParseInt(value, 10, 64)
			if err != nil || id < 0 || seen[id] || strconv.FormatInt(id, 10) != value {
				return "", "", nil, errors.New("calendar IDs must be unique nonnegative integers")
			}
			seen[id] = true
			q.Add("calendar_id", value)
		}
	}
	return http.MethodGet, path, q, nil
}

// parseDateTime preserves explicit offsets and interprets unzoned input locally.
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
