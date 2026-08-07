# AI coding agent instructions

This file provides guidance to AI coding agents when working with code in this repository.

## Project Overview

This is a calendar application with a Go backend, REST API and an embedded TypeScript web frontend. 
The codebase includes OpenAPI specs, SQLite storage, and multiple calendar view components (day, week, month, year, schedule).
There is also an Android (Kotlin/Compose) app using the same REST API in another repository.

## Build & Run

```bash
./build.sh                              # compile TypeScript, generate API code, build binary
./mycal                                 # serves on :8080
./mycal -port 3000 -data /path/to/data  # custom address and data path
```

`ogen`, `tsc`, `openapi-typescript` and `node` must be on `$PATH`

## Code generation

Two code generation steps run automatically in `build.sh`:

**TypeScript types from OpenAPI** (`openapi-typescript`):
- `web/ts/api/types.ts` is generated from `openapi.yaml` by `openapi-typescript` — never manually edit it.
- The frontend uses these generated types directly (e.g. `components['schemas']['Event']`), aliasing them per-file where needed. There is no hand-maintained re-export layer.
- App-only types not in the API (e.g. `AppConfig`) live next to their logic, e.g. exported from `web/ts/util/config.ts`.
- After changing `openapi.yaml`, run `openapi-typescript openapi.yaml -o web/ts/api/types.ts` (or `./build.sh`) before running `tsc`.

**Go HTTP server stubs** (`ogen`):
- `internal/api/` is generated from `openapi.yaml` using [ogen](https://ogen.dev/).
- Always run `go generate ./...` (or `./build.sh`) before building after changing `openapi.yaml`.
- Never manually edit any file in `internal/api/` — all changes are overwritten by `go generate`.
- To add or change API behaviour, edit `openapi.yaml` and regenerate, then update the implementation in `internal/handler/impl.go`.

## Tests

Go tests should use the `github.com/stretchr/testify` library for assertions. 
Use `require` for critical checks (like error handling) to stop test execution early, and `assert` for other checks.

```bash
go test ./...                           # all tests
go test ./internal/repository/ -v       # repository tests only
go test ./internal/repository/ -run TestList  # single test
```

## Frontend tests

Frontend unit tests are `*.test.mjs` files directly under `web/ts/`, written
against Node's built-in `node:test` and `node:assert/strict` — no test runner,
no package manager, nothing to install beyond `node` itself. `build.sh` runs
them after `tsc`.

```bash
node --test web/ts/date-utils.test.mjs   # one file
node --test web/ts/*.test.mjs            # all of them
```

- **They import the compiled output**, `web/static/…`, not the `.ts` source, so
  `tsc` must have run first. The test files stay plain `.mjs` — `tsc` only
  picks up `*.ts`/`*.tsx`, so they are never compiled or type-checked.
- **A test names the module it covers** in a header comment, along with what is
  under test and why it is worth pinning.
- **The module under test is pulled in with a top-level `await import(…)`,**
  not a static `import`. Static imports are hoisted above the file body, so
  anything the module needs in place at load time — a `window`/`document` on
  `globalThis`, the `process.env.TZ` `date-utils.test.mjs` sets — has to be
  arranged first, and only a dynamic import runs after it.
- **Nothing may depend on the machine running the test.** `date-utils.test.mjs`
  fixes the zone rather than inheriting it, and asserts no output of
  `toLocaleDateString`/`toLocaleTimeString`, since that would pin the host's
  ICU locale rather than this code.

## E2E Tests

Playwright end-to-end tests live in `e2e/`. **Run them with `./build.sh && ./test-e2e.sh`** — that
script is what CI runs, and it starts the server itself on a fresh database, checks the server is
actually serving the assets on disk, and tears both down afterwards. It takes the same arguments
as `playwright test`, so `./test-e2e.sh tests/sidebar-footer.spec.ts -g "focus"` works.

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
(`http://localhost:8089`) or CSRF rejects every mutating request with 403 and every write test
fails.

*Important:* interactively, use the `playwright-test` command from `e2e/` and nothing else —
do not invent variants. `test-e2e.sh` falls back to `./node_modules/.bin/playwright test` when
that wrapper is absent, which is the case in CI; that fallback is sanctioned and is the only one.

## Verification

Use the `playwright-cli` skill to verify that the web frontend looks reasonable.

## Architecture

Go backend with embedded Preact+JSX frontend. TypeScript source in `web/ts/`, compiled by `tsc` to `web/static/` which Go embeds. Single binary serves both JSON API and static files.

**Layered backend** (`main.go` wires everything):
- **model** → Event struct, request types (Create/Update), validation. Datetimes are RFC 3339 strings throughout.
- **repository** → `EventRepository` interface + SQLite implementation (`modernc.org/sqlite`, pure Go). Schema auto-created on startup.
- **service** → Business logic wrapping repository. Returns typed sentinel errors (`ErrNotFound`, `ErrValidation`).
- **handler** → HTTP handlers using Go 1.22+ `ServeMux` routing patterns (`"GET /api/v1/events/{id}"`). JSON helpers and middleware (logging, recovery, CORS).

**Frontend** (`web/static/`, embedded via `web/embed.go`):
- Preact loaded from vendor files via import map in `index.html` (`vendor/preact/preact.module.js`, `…/hooks.module.js`, `…/jsx-runtime.module.js`)
- `web/ts/app.tsx` is the root component (compiled to `web/static/app.js` by `tsc`)
- Native `<dialog>` element for the event form (no client-side routing)
- API calls go through the centralized client in `api/client.ts`: a single `request<T>` helper with network retry, 401-reload, 404 `NotFoundError`, exposed as a namespaced `api.*` surface (`api.events`, `api.calendars`, `api.feeds`, `api.preferences`, `api.import`)
- Toasts are an observable store (`util/toast.ts`); any module calls `showToast(...)` / `showNetworkErrorToast(...)` and the `<Toast>` component subscribes
- Main stylesheet is `web/static/app.css`. Its palette is deliberately shared with MyNotes: the
  five variables MyNotes exposes (`--fg`, `--muted`, `--primary`, `--link`, `--danger`) carry the
  same values here as in that repo's `web/static/render/note.css`, so a note framed by the render
  kit in `<NotePanel>` sits on the same colours as the rest of the UI. MyCal's own variables
  (`--surface`, `--hover-bg`, `--tag-bg`, …) are derived from the same Tailwind ramp. `--text` is
  an alias of `--fg`. Change a colour there and change it here — never hardcode a hex outside the
  `:root` / `[data-theme="dark"]` blocks.
- **The sidebar footer's two controls are a three-repo contract, defined outside this repo.**
  The light/dark toggle and Settings in `.sidebar-footer` are specified to look identical, and
  to sit at the same window coordinates, in MyCal, MyMail and MyNotes — so that someone with
  all three open in browser tabs sees nothing move when switching between them. MyCal
  implements that contract; it does not define it.

  **The definition is [`../mysuite/spec/sidebar-footer.md`](../mysuite/spec/sidebar-footer.md)**
  (sibling checkout, no remote yet). Read it before changing anything in the
  `.sidebar-footer-btn` rule, `.sidebar-footer`, or the theme toggle's markup — including the
  declarations that look redundant, which are pinned deliberately and for reasons the CSS
  comments give at the point of use. It also carries a withdrawn-rules table, so an older
  comment or report elsewhere is not authority for re-deriving a superseded rule. The
  verification procedure it depends on is
  [`../mysuite/spec/measurement-protocol.md`](../mysuite/spec/measurement-protocol.md) —
  a green build proves nothing about geometry here, for the reason given under E2E Tests above.
  That checkout has no remote yet, so if the path does not resolve for you the
  `.sidebar-footer-btn` comment block carries every resolved value and is self-sufficient;
  what you would be missing is the reasoning and the withdrawn-rules table, not the numbers.

  **Changing any of this is a change in all three repositories.** Nothing anywhere can detect
  that one of them has drifted — there is no shared stylesheet and no cross-repo test — so a
  local "fix" here silently breaks the contract rather than failing. That sentence is repeated
  from the shared spec on purpose: the person about to edit this CSS is exactly the person who
  may never open a pointer.

  **Edits that break it silently.** Each is an ordinary tidy-up that leaves `./build.sh` and
  the Go tests green. Each was applied deliberately to find out which assertion fires, so the
  annotations below are measured rather than assumed:
  - normalising `font-size: 0.80rem` to `0.8rem` — **nothing catches this.** The computed and
    serialised values are identical, so no test can distinguish them; the shared spec says so
    too (§2.1). A convention held by review, not by CI.
  - deleting a "redundant" `flex-shrink: 0`, `text-align: center`, `font-weight: 400` or
    `font-style: normal` — caught by *the pinned declarations are actually declared*, which
    reads the rule out of the CSSOM. It has to: the first three are no-ops in **this** repo,
    so nothing about MyCal's rendering changes when they go missing.
  - folding `.sidebar-footer-btn` into a shared icon-button class — caught by the same test,
    which looks up the rule by selector
  - restoring `outline: none` on `:focus-visible`, or shrinking the 2px outline or its offset
    — caught by the three focus tests
  - changing the padding, border, radius or gap — caught by *the pinned declarations resolve
    to the contract values*
  - renaming `.sidebar-footer-btn` or changing `title="Settings"` — every test in
    `sidebar-footer.spec.ts` locates through one or the other, so this fails loudly
  - changing `--sidebar-width` without re-measuring `.brand`, which shares it — caught by
    `calendar-views.spec.ts`, which pins Reload's right edge to the column's
  - removing `.sidebar-footer`'s negative `margin-left` — caught by the (8, 8) tests. Giving
    `.app` a `border-left` would clip the footer instead, and **nothing catches that**: the
    buttons' own 8px still measures correct.
  - repainting `.sidebar-footer` on `--surface` — caught by *controls sit on the page
    background, not a panel* in both themes, and by *the footer paints an opaque background
    of its own*. This is the state MyCal shipped until the owner asked for the box to be
    removed; the deviation from the shared spec is sanctioned and recorded upstream.
  - deleting `.sidebar-footer`'s `background` as redundant — caught by *the footer paints an
    opaque background of its own*, and **by nothing else**. Verified by deleting it: the
    other six colour assertions stayed green, because the backdrop walk falls straight
    through to `<body>` and finds the identical colour. The colour is redundant; the opacity
    is not, and content slides through the buttons when the month view scrolls.
  - resolving `.sidebar-footer-btn`'s `color` or hover `background` straight off
    `--text-subtle` / `--hover-bg` instead of the `--sidebar-footer-*` aliases — caught in
    light by *resting label meets 4.5:1* (4.393:1) and *hover fill is distinguishable from
    its backdrop* (1.000:1). Those two aliases are light-only deviations and exist because
    MyCal's light `--bg` is `#f3f4f6`, which is also its `--hover-bg`.
  - collapsing those aliases into one unscoped pair — caught by *the dark aliases name the
    shared tokens*, and by nothing else. Verified: the six ratio assertions stayed green,
    because dark `--border` and dark `--hover-bg` are both `#374151`, so the wrong token
    resolves to the right colour today. A CSSOM read is the only thing that can see it.
  - measuring any of the above against a server you did not restart after `./build.sh` — a
    stale binary makes a broken change look fine. `./test-e2e.sh` refuses to run in that
    state; a hand-started server does not.
  - reading any colour off these buttons within 120ms of a theme switch or a hover — the
    mandated `transition` covers `background`, `color` and `border-color`, so all three
    report a blend of the two palettes for that long. This cost a debugging round: the
    resting label read `#4b5563` (the *light* value) against a dark backdrop for 2.347:1,
    which points at the palette and not at the clock. Assertions that the element is not
    hovered do not catch it — the state is right and the timing is wrong. Use the spec's
    `settledStyle` helper, which polls until two reads agree.

  What is MyCal's own, and therefore lives here:
  - `e2e/tests/sidebar-footer.spec.ts` is this repo's half of the contract, and the only
    automated check it has anywhere — MyMail and MyNotes have no e2e suite at all. It runs in
    CI (`.github/workflows/main.yml` → `./test-e2e.sh`) and gates publishing. Note what that
    does and does not mean: the workflow triggers on `push` to `main`, so a breaking commit is
    already on `main` by the time the suite is red — what the gate prevents is a broken
    contract reaching Pages or the rolling release, not the commit landing. Read the suite
    alongside the spec; the annotations above say which assertion covers what.
  - Two assertions do most of that work and they check different things: one reads the
    *computed* values, the other reads the *declarations* out of the CSSOM. The second exists
    because MyCal's `<button>` already inherits `font-weight: 400` and `text-align: center`
    from the UA, so deleting those pins changes nothing observable **here** while breaking the
    match with MyNotes, whose `button { font: inherit }` sends it to `body` instead. The app
    where a value is already right is the app whose rendering cannot detect the pin going
    missing.
  - App-wide focus indicators are a known deferred gap, not an oversight — see the shared
    spec's open-items section. `.search-input:focus` still carries the `outline: none` pattern
    this contract removed from the footer, and its own rule says so.
  - `.sidebar-footer-btn` and `title="Settings"` are pinned by that suite — do not rename them.
  - `--sidebar-width` sizes the left column **and** `.brand` above it, which must stay equal
    (`calendar-views.spec.ts` asserts Reload's right edge against the column's), and must stay
    in `rem` so the column grows with the rem-sized buttons inside it.
  - MyCal reaches the shared position by a mechanism the other two do not need: the footer
    cancels `.app`'s horizontal padding via `--app-padding-x` and drops its own bottom padding,
    because an e2e assertion pins the footer box's bottom edge to the view beside it. The spec
    sanctions this explicitly; it is an implementation of the contract, not a deviation.
  - The narrow (≤600px) layout is out of scope for the 8px *bottom* rule — `.app`'s padding
    drops to 4px there. The left edge still holds.
- Icons are Lucide, rendered inline as `<svg stroke="currentColor">` by `components/Icon.tsx`. The
  vendored bundle carries **only** the icons listed in the `ICONS` array of
  `web/ts/vendor/gen-lucide.mjs` — to use a new one, add its kebab-case name there, re-run
  `web/ts/vendor/rebuild.sh`, and commit the regenerated `web/static/vendor/lucide-<version>.js`.
  `<Icon>` silently renders nothing for a name that is not in the bundle.
- The one deliberate exception is `components/Logo.tsx`, the app mark in the top bar's
  `.brand-logo` badge. It is MyCal's own glyph, not a Lucide icon, so it is hand-written inline
  SVG and must **not** be routed through the Lucide bundle. It shares its geometry with
  `web/static/favicon.svg` — differing only in that it draws in `currentColor` to invert onto the
  blue badge — so a change to either belongs in both.
- **A button is either an icon or a label, never both.** Buttons that show text (Save, Delete,
  Close, Add Feed, …) stay text-only; `<Icon>` is for the buttons that would otherwise be a bare
  glyph — the top-bar actions, the nav and mini-month arrows, dialog dismiss ✕, the calendar
  sidebar's edit controls, the feed row actions. The one exception is the left sidebar's footer
  (`.sidebar-footer-btn`: the light/dark toggle and Settings), which pairs icon and label to match
  the same two buttons in MyNotes' sidebar footer.
- An icon-only button needs `title` **and** `aria-label`: the SVG is `aria-hidden`, so without one
  the button has no accessible name. Keep those names distinct from the visible labels elsewhere on
  the page — the e2e suite locates buttons by accessible name, and Playwright matches substrings.

**TypeScript** (`web/ts/`, compiled to `web/static/`):
- `web/ts/tsconfig.json`: target ES2020, module/moduleResolution Node16, `jsx: react-jsx` + `jsxImportSource: preact` (automatic runtime — no `import { h }`), `strict: true`
- `web/ts/package.json` contains `{ "type": "module" }` — **required** so `tsc` emits ESM under Node16; without it the output is CommonJS `require()` and the browser shows a blank page
- Directory taxonomy: `api/` (client + generated types), `util/` (lowercase utilities), `components/` (reusable widgets), `layout/` (Nav, sidebars), `views/` (calendar views). Component files are `PascalCase.tsx`; utilities are `lowercase.ts`
- Vendored preact `.d.ts` live under `web/ts/vendor/preact/` (referenced via tsconfig `paths`, excluded from the build); Leaflet/Quill/Lucide ambient declarations live in `web/ts/vendor/` (Lucide's bundle is mapped to the bare specifier `lucide-icons` in both tsconfig `paths` and the `index.html` import map)
- Relative imports use `.js` extensions (TypeScript ESM convention — tsc resolves `.ts`/`.tsx`, emits `.js`)
- Every `.js` under `web/static/` outside `web/static/vendor/**` is emitted by `tsc` — do not edit directly; `.gitignore` keeps those out of Git. The rest of what lives there is hand-maintained and tracked, and is edited in place: `app.css`, `index.html` and the favicons

**Key design decisions:**
- `UpdateEventRequest` uses pointer fields for partial updates (nil = unchanged)
- Repository returns `nil, nil` for not-found on GetByID; service layer maps this to `ErrNotFound`
- Repository Delete returns `sql.ErrNoRows` for not-found; service maps to `ErrNotFound`
- List endpoint requires `from`/`to` query params; uses overlapping range query (`start_time < to AND end_time > from`)
- iCalendar (RFC 5545) feed at `/calendar.ics` and `/api/v1/events.ics` — `internal/ical` package encodes events, no external dependency
- The REST API is mounted at the specific `/api/v1/` prefix (not the broader `/api/`) so the compiled frontend module served at `/api/client.js` (from the `web/ts/api/` source dir) falls through to the static file handler. Keep this mount narrow.

**Sibling app integrations (MyMail, MyNotes):**
- Both assume the sibling runs on the **same origin** behind the same auth realm. `main.go` derives their base URLs from `-public-url` (`deriveSiblingURL`) and injects them as `window.__serverConfig`; Settings can override. All calls are browser-side with `credentials: 'include'` — no server-to-server traffic.
- MyNotes: an event links one note through `note_slug` (schema v2 migration in `internal/repository/db.go`). `web/ts/util/mynotes.ts` is the API client; `web/ts/components/NotePanel.tsx` frames the **MyNotes render kit** (`<mynotesUrl>/render/`) and drives its `render()` / `setTheme()` API.
- Never re-implement the MyNotes Markdown dialect here — the render kit is the single renderer, shared with the MyNotes web UI and the Android app. The kit's output carries root-relative links and image sources (it has no `<base>`); `NotePanel` resolves them against the MyNotes base URL and forces links into a new tab. MyCal's CSP needs `frame-src 'self'`, and MyNotes serves `/render/` with `frame-ancestors 'self'`.

## Demo mode

`-demo-server` and `-demo-bundle DIR` build the web UI with **no backend**: a
service worker (`web/ts/demo-sw.ts` + `web/ts/demo/`) intercepts `/api/v1` and
answers it from IndexedDB. `main.go` injects `window.__serverConfig={demo:true}`
(same mechanism as the sibling URLs); `app.tsx` then waits for the worker to be
installed and in control before rendering, so the first request cannot escape it.
`-demo-bundle` writes the same thing out as static files, which
`.github/workflows/main.yml` publishes to GitHub Pages.

- **Intercepting at the network layer is the point**: the frontend is unchanged
  between demo and real, so nothing in `views/` or `components/` needs to know.
- **Parity with the Go server is the contract.** `web/ts/demo/` re-implements
  `internal/service` plus the relevant parts of `internal/repository`,
  `internal/model` and `internal/sanitize`; every function names the Go original
  it mirrors. When you change validation, sanitization, search, or the overlap
  query on the server, change it there too.
- **The accepted divergences** are the ones the demo notice lists
  (`components/DemoDialog.tsx`) — don't add more silently. Today: iCalendar
  import/export, feed subscriptions, MyMail sharing, MyNotes linking, and
  **recurring events**, which the emulated backend stores but never expands into
  occurrences. Each one is hidden in the UI rather than left to fail: see the
  `demo` guards in `app.tsx`, `Settings.tsx` and `EventForm.tsx`.
- These sources are **worker code**: excluded from `web/ts/tsconfig.json` and
  built by `web/ts/demo/tsconfig.json` against the WebWorker lib. They are
  classic scripts sharing one global scope via `importScripts`, so they use no
  `import`/`export` — adding one silently turns a file into a module and its
  declarations vanish from the shared scope.
- **Nothing is hardcoded to the origin root.** `api/client.ts`, `demo-client.ts`
  and the worker's scope matching all resolve against `<base href>`, which
  `-demo-bundle` sets from `-public-url`, so a bundle works under a subpath.
- **The starting content lives in `demo/seed.ts`** and mirrors no Go package —
  the real server has no seeding command. It is written by `initialState()` on a
  fresh store, so the dates are computed at first visit, never at build time: a
  bundle published once and served for months must still open on *this* week.
  Events are addressed by weekday within the locale's week (ported from
  `config.ts`), which keeps them on the week the visitor is actually shown, and
  every weekday carries something so the day and schedule views are never empty.
  Rows are built directly rather than posted through the API, so anything a
  create request would normalise — `sanitizeHTML` on descriptions, UTC timed
  stamps, UTC-midnight all-day with an exclusive end — has to be applied there.

## Go development

Always run `go mod tidy` after modifying the `go.mod` file.

## API

The API is defined in the OpenAPI spec in `openapi.yaml`, update it whenever the API is changed. 
The OpenAPI spec is used to generate server and client code, so it must be accurate.

In addition to the web UI here, the API is consumed by mobile clients outside of this repository. Keep the API backwards 
compatible whenever possible, but breaking changes are acceptable if hard to avoid. Make sure to report any breaking changes 
so that clients can be updated.

## Database

Make migrations when changing the database schema, assume there is production data that needs to be preserved. 

## Version control

Git is used for version control. When creating new files, make sure to add them to Git.

## Security Guidelines

### Input sanitization — apply uniformly across all write paths

- Sanitize HTML (using `sanitize.HTML`) on **every** write path: interactive create, interactive update, iCal import, and override import. Never sanitize only on creation and skip it on update.
- Apply the same validation to imported data that you apply to interactively-entered data. An import path is not a trusted source.
- In the frontend, never assign HTML to `innerHTML` directly. Use the library's own API (e.g. `quill.setContents(quill.clipboard.convert(value))`) so the library's sanitizer runs.

### URL and scheme validation

- Validate URL schemes (allow only `http:`, `https:`, `mailto:`) on **all** paths — both interactive and import. Never store a URL without scheme validation.
- In the frontend, validate `href` values before rendering them as links; fall back to plain text for disallowed schemes.

### SSRF prevention

- Validate URLs at **connection time** (inside a custom `DialContext`), not just before the fetch. Validate-then-fetch is a TOCTOU race (DNS rebinding).
- Set `CheckRedirect` to re-validate every redirect target; do not follow redirects blindly to potentially private addresses.
- Re-run URL validation on every periodic re-fetch of stored feed URLs, not just on initial registration.
- Return generic error messages to the client for DNS/network errors (e.g. "could not resolve URL host"). Log details server-side only.

### HTTP server hardening

- Apply a global request body size limit (`http.MaxBytesHandler` or `MaxBytesReader`) to **all** handlers, not just file upload endpoints.
- Set both `ReadTimeout` and `ReadHeaderTimeout` on `http.Server` to prevent Slowloris-style attacks.

### Security headers and CSP

- Keep the Content-Security-Policy tight. When adding new outbound resources (tile servers, CDNs, APIs), update **both** `connect-src` and any other relevant directive (`img-src`, `script-src`) at the same time.
- Include `frame-ancestors 'none'` in the CSP in addition to `X-Frame-Options: DENY`; modern browsers prefer the CSP directive.
- Avoid `'unsafe-inline'` in `style-src`; prefer nonces or hashes.

### API design

- GET requests must never modify the database or create side effects. Filtering by a non-existent resource name should return an empty result, not create the resource.
- Add `maxLength` constraints in `openapi.yaml` for all string query parameters, not just request body fields.
