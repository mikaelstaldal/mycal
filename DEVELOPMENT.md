# MyCal development guide

[Project overview](README.md) · [User manual](USER_MANUAL.md)

See [AGENTS.md](AGENTS.md) for build prerequisites, architecture, and development conventions.

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

## Publishing the demo

The demo is published to GitHub Pages from `main` by
[`.github/workflows/main.yml`](.github/workflows/main.yml), which builds the
bundle for whatever URL Pages reports. Enable Pages with **GitHub Actions** as
its source for that to work.
