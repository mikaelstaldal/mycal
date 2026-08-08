# AI coding agent instructions — end-to-end tests

This file covers the Playwright suite under `e2e/`, and is loaded in addition to the
repository-root `AGENTS.md`, which keeps the project overview, the build and code-generation
commands, the Go tests and the security guidelines. Paths below are relative to the repository
root, as they are there.

`test-e2e.sh` takes the same arguments as `playwright test`, so
`./test-e2e.sh tests/sidebar-footer.spec.ts -g "focus"` works.

Prefer it over starting a server by hand. The two things it exists to prevent are easy to hit and
neither announces itself:

- **A stale server.** `web/embed.go` bakes `web/static/` into the binary, so a running `./mycal`
  keeps serving the CSS and JS it started with — `./build.sh` alone changes nothing it serves. The
  suite then passes or fails against assets that are not the ones you edited. When a measurement
  disagrees with the source, check this first:
  ```bash
  curl -s http://localhost:8089/app.css | md5sum   # must match
  md5sum web/static/app.css
  ```
- **A stale database, or someone else's server on the port.** Reusing a data directory is how an
  "empty" run silently becomes a run against whatever the last one left behind; and if something
  already holds 8089, a hand-started server exits on bind failure while the tests run happily
  against the squatter.

If you do start one by hand, `-public-url` must match the test baseURL origin
(`http://localhost:8089`), or CSRF rejects mutating requests with 403.

Not *every* mutating request, and not every write test — which is what this said until it was
measured. `csrf.Middleware` allows a non-GET carrying neither `Origin` nor `Referer` ("native
client, allow", go-server-common v1.8.0 `csrf/csrf.go`), so Playwright's `request` fixture sails
through a mismatched flag, while anything the browser issues from inside the page carries the
page's Origin and is rejected. The flag is still required — it would bite the moment a spec
clicked Save rather than posting its fixture. The measured table is in `test-e2e.sh`, in the
comment above the line that starts the server; it is not repeated here.

*Important:* interactively, use the `playwright-test` command from `e2e/` and nothing else —
do not invent variants. `test-e2e.sh` falls back to `./node_modules/.bin/playwright test` when
that wrapper is absent, which is the case in CI; that fallback is sanctioned and is the only one.
