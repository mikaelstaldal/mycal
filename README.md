# mycal

A personal calendar application with iCalendar support. Go backend, SQLite storage, REST API, and a built-in Preact web frontend.

## Status

*Note:* This application has mostly been vibe coded with AI tools, it has not been properly reviewed or extensively tested,
and is under active development. Use at your own risk, and backup your data often if you use for real.

## Features

- Monthly calendar grid with event display
- Create, view, edit, and delete events
- Color-coded events
- iCalendar (RFC 5545) import and feed for subscribing from other calendar apps
- JSON REST API for future native clients
- Single binary with embedded frontend — no JS build step

## Getting Started

```bash
go build -o mycal . && ./mycal
```

Open http://localhost:8080 in your browser.

### Options

| Flag               | Default      | Description                                         |
|--------------------|--------------|-----------------------------------------------------|
| `-addr`            | `:8080`      | Listen address                                      |
| `-db`              | `mycal.db`   | SQLite database file path                           |
| `-basic-auth-file` | *(disabled)* | Path to htpasswd file for HTTP basic authentication |

### Authentication

HTTP basic authentication can be enabled by providing an [htpasswd](https://httpd.apache.org/docs/current/programs/htpasswd.html) file with bcrypt-hashed passwords:

```bash
# Create a new htpasswd file with a user (requires apache2-utils or httpd-tools)
htpasswd -Bc htpasswd admin

# Start with authentication enabled
./mycal -basic-auth-file htpasswd
```

When enabled, all endpoints (UI, API, and iCalendar feed) require valid credentials. The browser will prompt for a username and password automatically.

## API

See [API docs](docs/API.md).

## iCalendar Feed

Subscribe to your calendar from any app that supports iCalendar (Google Calendar, Apple Calendar, Thunderbird, etc.) using:

## E2E Tests

End-to-end tests use [Playwright](https://playwright.dev/) and live in the `e2e/` directory.

```bash
# Install dependencies (first time)
cd e2e && npm install && npx playwright install chromium && cd ..

# Start the server on port 8089
go build -o mycal . && ./mycal -addr :8089 -db /tmp/mycal-e2e.db &

# Run tests
cd e2e && bash playwright-test

# Run headed (visible browser)
cd e2e && bash playwright-test --headed
```

## Tech Stack

- **Backend:** Go with `net/http` (Go 1.22+ routing)
- **Database:** SQLite via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (pure Go, no CGO)
- **Frontend:** [Preact](https://preactjs.com/) + [HTM](https://github.com/developit/htm) loaded from CDN

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
