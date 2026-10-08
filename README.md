# MyCal

A personal calendar application with iCalendar support. Go backend, SQLite storage, REST API, and a built-in Preact web frontend.

See live demo at <https://mikaelstaldal.github.io/mycal/>.

## Clients

In addition to the built-in web interface, there is also 

* A [native Android app](https://github.com/mikaelstaldal/mycal-android)

## Features

- Monthly calendar grid with event display
- Create, view, edit, and delete events
- Color-coded events
- iCalendar (RFC 5545) import and feed for subscribing from other calendar apps
- JSON REST API for future native clients
- Single binary with embedded frontend — no bundler, no npm install to build
- Sibling-app integrations: share an event by email through
  [MyMail](https://github.com/mikaelstaldal/mymail), and link a note from
  [MyNotes](https://github.com/mikaelstaldal/mynotes) to an event — see below

## Getting Started

```bash
./build.sh && ./mycal
```

Open http://localhost:8080 in your browser.

### Options

| Flag                | Default                | Description                                                                                              |
|---------------------|------------------------|----------------------------------------------------------------------------------------------------------|
| `-port`             | 8080                   | port to listen on                                                                                        |
| `-addr`             | `127.0.0.1`            | address to listen on (wildcard binds require `-public-url`)                                              |
| `-data`             | `data`                 | directory to store data in                                                                               |
| `-public-url`       | *(local authorities)* | public-facing base URL for Host and CSRF validation (required for wildcard binds and reverse proxies), e.g. `https://example.com/mycal` |
| `-https`            | false                  | set `Strict-Transport-Security` header (use when served behind a TLS-terminating proxy)                  |
| `-basic-auth-file`  | *(disabled)*           | enable HTTP basic auth with username and password from given file in htpasswd format (bcrypt only)       |
| `-basic-auth-realm` | `mycal`                | realm for HTTP basic auth                                                                                |
| `-export-ics`       |                        | export all events to an .ics file and exit                                                               |
| `-demo-server`      |                        | run the browser-only demo (no database, no REST API); see [Demo mode](#demo-mode)                        |
| `-demo-bundle`      |                        | write a static demo site to this new directory and exit                                                  |

### Host validation and reverse proxies

All HTTP routes, including the UI, API, calendar feeds and demo server, reject
foreign or malformed request Hosts with HTTP 421 before authentication and routing.
Without `-public-url`, allowed authorities are the concrete bind address plus
`localhost`, `127.0.0.1` and `[::1]`, at the listener port. Browser writes from
these local HTTP origins are allowed by CSRF validation too.

Set `-public-url` to the browser-facing URL when using a reverse proxy or a
wildcard bind (`-addr 0.0.0.0`, `-addr ::`, or an empty address). For example:

```bash
./mycal -addr 0.0.0.0 -port 8089 -public-url https://calendar.example.com:8443/mycal -https
```

The public authority and HTTPS origin are allowed alongside the local authorities.
Include any custom public port; the URL path does not affect Host or CSRF checks.
A proxy must preserve the public Host or send an allowed local Host, and strip the
public path prefix before forwarding. Forwarded headers such as
`X-Forwarded-Host` do not grant access. `-https` enables HSTS; it does not change
the local listener from HTTP to HTTPS. Static demo bundles use their hosting
server's Host policy.

### Authentication

HTTP basic authentication can be enabled by providing an [htpasswd](https://httpd.apache.org/docs/current/programs/htpasswd.html) file with bcrypt-hashed passwords:

```bash
# Create a new htpasswd file with a user (requires apache2-utils or httpd-tools)
htpasswd -Bc htpasswd admin

# Start with authentication enabled
./mycal -basic-auth-file htpasswd
```

When enabled, all endpoints (UI, API, and iCalendar feed) require valid credentials. The browser will prompt for a username and password automatically.

The htpasswd file is parsed strictly at startup: every non-blank line must be a `username:bcrypt-hash` pair with no duplicate usernames, and the file must not be empty. Anything else aborts startup — a bad entry named by file and line number, an entryless file by name — rather than silently leaving out a login you believe exists.

## API

See the [OpenAPI specification](openapi.yaml).

## Demo mode

Demo mode runs the full web UI with no backend at all. A service worker
intercepts every `/api/v1` request and answers it from IndexedDB in the browser,
so events and calendars are created, searched, edited, and deleted exactly as
they are against the real server — they just never leave the machine. It opens on
a week of made-up sample events, timed and all-day, placed relative to the day of
the visit rather than the day the bundle was built; clearing the site's data
resets the demo to those. A modal on the first visit says as much, so nobody
writes anything they care about into it.

```bash
./mycal -demo-server                 # serve the demo on http://127.0.0.1:8080
./mycal -demo-bundle /tmp/mycal-demo # or write it out as a static site
```

Neither mode opens a database, so neither takes `-data` or `-export-ics`.

`-demo-bundle` writes plain files that any web server can host — no backend, no
build step. It is built for the origin root by default; for a subdirectory, pass
the same `-public-url` the server would take (`-public-url
https://example.com/mycal`), which becomes the page's `<base href>`. The path may
contain only `A-Z a-z 0-9 . _ ~ - /`, since it is injected into that attribute.
**Service workers need a secure context**, so serve the bundle over HTTPS or from
`localhost`.

The demo is published to GitHub Pages from `main` by
[`.github/workflows/pages.yml`](.github/workflows/pages.yml), which builds the
bundle for whatever URL Pages reports. Enable Pages with **GitHub Actions** as
its source for that to work.

What the demo does not do, because the browser is the backend:

- Importing and exporting iCalendar files, and subscribing to calendar feeds —
  all of which need a server to fetch and parse.
- Sharing an event by e-mail, and linking notes from MyNotes: both assume a
  sibling app on the same origin behind the same auth realm.
- **Recurring events.** The in-browser backend stores the recurrence fields but
  never expands a rule into its occurrences, so the repeat controls are hidden
  and every event is a single one.

## MyMail and MyNotes integrations

Both integrations assume the sibling app runs on the **same origin** as mycal,
behind the same authentication realm, so the browser's existing credentials
carry over and no CORS, token exchange or server-to-server call is involved.
Their base URLs are derived from `-public-url` by replacing its path — a mycal
at `https://example.com/mycal` looks for MyMail at `https://example.com/mymail`
and MyNotes at `https://example.com/mynotes` — and injected into the page as
`window.__serverConfig`. A mycal served from the origin root derives nothing;
either URL can also be set by hand under Settings.

- **MyMail** — the *Share* button in the event dialog mails the event as an
  `.ics` attachment.
- **MyNotes** — an event can link one note (stored as `note_slug` on the event).
  In edit mode a title search picks the note; in view mode the dialog shows the
  note's content, rendered by MyNotes' own **render kit** (`/mynotes/render/`)
  loaded in an iframe and driven through its `render()` / `setTheme()` API. The
  full MyNotes Markdown dialect — callouts, wikilinks, tables, math, Mermaid
  diagrams, icons — therefore renders exactly as it does in MyNotes, and mycal
  never parses Markdown itself. MyNotes serves that page with
  `frame-ancestors 'self'`, so the two must share an origin for this to work.

## iCalendar Feed

Subscribe to your calendar from any app that supports iCalendar (Google Calendar, Apple Calendar, Thunderbird, etc.) using:

```text
https://your-mycal-server/calendar.ics
```

## Calendar subscriptions

MyCal accepts HTTP, HTTPS, `webcal://`, and `webcals://` feed URLs in **Feed Subscriptions**.
To open `webcal:` links clicked on other websites in MyCal, open MyCal over HTTPS and select
**Use MyCal for webcal links** in that dialog. Allow the handler in your browser, then choose
MyCal for webcal links if prompted. When a link opens MyCal, confirm the subscription there.
The browser handler covers `webcal:` links; `webcals:` can be added manually. MyCal fetches
both webcal schemes over HTTPS.

## E2E Tests

End-to-end tests use [Playwright](https://playwright.dev/) and live in the `e2e/` directory.

```bash
# Install dependencies (first time)
cd e2e && npm ci && npx playwright install chromium && cd ..

# Build, then run. test-e2e.sh starts the server itself on a fresh database,
# checks it is serving the assets you just built, and tears both down after.
./build.sh && ./test-e2e.sh

# It forwards its arguments to Playwright
./test-e2e.sh --headed
./test-e2e.sh tests/sidebar-footer.spec.ts
```

Start the server by hand only if you have a reason to. The binary embeds `web/static/`, so one
started before a rebuild serves the old assets and the tests then measure something other than
what you changed. `test-e2e.sh` configures the public URL as `http://localhost:8089`;
the shared Host and CSRF policy also allows the local HTTP aliases at the listener port.

## Tech Stack

- **Backend:** Go with `net/http` (Go 1.22+ routing)
- **Database:** SQLite via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (pure Go, no CGO)
- **Frontend:** [Preact](https://preactjs.com/) with JSX, written in TypeScript (`web/ts/`) and compiled by `tsc`
- **Vendoring:** Preact, [Quill](https://quilljs.com/) and [Leaflet](https://leafletjs.com/) are committed under `web/static/third_party/` and loaded via an import map — no CDN at runtime
- **API:** described in [openapi.yaml](openapi.yaml); Go server stubs generated with [ogen](https://ogen.dev/), TypeScript types with [openapi-typescript](https://github.com/openapi-ts/openapi-typescript)

## Operations Guide

See [OPERATIONS.md](OPERATIONS.md) for production installation, reverse proxy configuration, systemd service setup, and authentication.

## Credits

See [CREDITS.md](CREDITS.md) for the third-party libraries, assets and data sources used.

## License

Copyright 2026 Mikael Ståldal.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## API tokens and command-line clients

`./build.sh` also builds `mycal-token` and `mycal-cli`. Tokens grant read-only
access to one or more calendars, including the default calendar (ID `0`). They
allow calendar listing, event listing and search, individual events and recurrence
instances, and iCalendar exports. Collections contain only granted calendars.
Tokens cannot change events, manage tokens, read preferences or access feeds or
the web UI. Tokens expire and can be revoked; only their SHA-256 hashes are stored.
Scoped recurring views omit overrides outside the grant and retain the original
parent occurrence. Revocation slugs identify current tokens and can be reused
once revoked or expired. The management API accepts any future expiry; the CLI
limits lifetimes to ten years.

`mycal-token` creates and revokes tokens using full access (Basic authentication
when enabled). Store a single `username:password` line in a credentials file, or
use `-user USER` to prompt for a password without echo. Servers without Basic
auth need no credentials. Creation prints only the secret to stdout and its
revocation slug to stderr; the secret cannot be retrieved later.

```bash
MYCAL_URL=https://example.com/mycal ./mycal-token -credentials-file ./credentials \
  create -name 'Calendar reader' -lifetime 30d -calendars 0,1 > token
chmod 600 token
MYCAL_URL=https://example.com/mycal MYCAL_TOKEN_FILE=./token ./mycal-cli calendars list
MYCAL_URL=https://example.com/mycal ./mycal-cli -token-file ./token events list \
  -from 2026-10-01T00:00:00Z -to 2026-11-01T00:00:00Z -calendars 1
MYCAL_URL=https://example.com/mycal ./mycal-cli -token-file ./token events search -q meeting
MYCAL_URL=https://example.com/mycal ./mycal-cli -token-file ./token calendars ics > calendar.ics
MYCAL_URL=https://example.com/mycal ./mycal-token -credentials-file ./credentials revoke calendar-reader
```

Keep credential and token files readable only by their owner. Put global flags
before the command. `MYCAL_URL` defaults both clients to a server URL; `-url`
overrides it. `MYCAL_TOKEN_FILE` supplies the read client's token file; explicit
`-token-file` or `-token-stdin` overrides it. Without a token, the client sends no
Authorization header. Both clients preserve deployment path prefixes, require
HTTPS except for literal loopback addresses, disable environment HTTP proxies,
and refuse redirects. JSON and iCalendar output is passed through to stdout;
errors go to stderr with a nonzero exit status. Run either client with `help` for
all commands and flags.

The API exposes full-access `GET/POST /api/v1/tokens` and
`DELETE /api/v1/tokens/{slug}`. Send a token as `Authorization: Bearer SECRET`.
Invalid, expired or revoked tokens return `401`; prohibited routes or methods
return `403`. An individual event outside the scope returns `404`.
