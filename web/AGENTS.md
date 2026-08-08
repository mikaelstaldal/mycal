# AI coding agent instructions — web frontend

This file covers the TypeScript/Preact frontend under `web/`, and is loaded in addition to the
repository-root `AGENTS.md`, which keeps the project overview, the build and code-generation
commands, the Go tests, how to run the `e2e/` Playwright suite and the security guidelines. The
suite's own instructions are in `e2e/AGENTS.md`. Paths below are relative to the repository root,
as they are there.

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

## Architecture

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

  **The definition is [`../mysuite/spec/sidebar-footer.md`](../../mysuite/spec/sidebar-footer.md)**
  (a sibling checkout of <https://github.com/mikaelstaldal/mysuite>; keep the path — the
  relative links resolve in a checkout, and the cross-repo check below assumes the four repos
  are siblings). **The link targets in this file read `../../mysuite/…` while the visible text
  and the bare paths in prose read `../mysuite/…`, and that mismatch is deliberate:** prose
  paths are relative to the repository root per the note at the top of this file, but a markdown
  link resolves from *this* file, which is one directory deeper. Do not "fix" the targets to
  match the prose — that is what made all five of them dead until it was measured. Read it
  before changing anything in the
  `.sidebar-footer-btn` rule, `.sidebar-footer`, or the theme toggle's markup — including the
  declarations that look redundant, which are pinned deliberately and for reasons the CSS
  comments give at the point of use. It also carries a withdrawn-rules table, so an older
  comment or report elsewhere is not authority for re-deriving a superseded rule. The
  verification procedure it depends on is
  [`../mysuite/spec/measurement-protocol.md`](../../mysuite/spec/measurement-protocol.md) —
  a green build proves nothing about geometry here, for the reason given in `e2e/AGENTS.md` under
  the stale-server trap.
  If the path does not resolve for you, clone the repository beside this one; until you do, the
  `.sidebar-footer-btn` comment block carries every resolved value and is self-sufficient —
  what you would be missing is the reasoning and the withdrawn-rules table, not the numbers.

  **Changing any of this is a change in all three repositories.** There is no shared
  stylesheet, and nothing runs automatically when you edit this file — so a local "fix" here
  breaks the contract quietly rather than failing. That sentence is repeated from the shared
  spec on purpose: the person about to edit this CSS is exactly the person who may never open
  a pointer.

  One thing *can* detect the drift, and it is not automatic: `../mysuite/tools/check-contract.py`
  reads all three stylesheets and fails when a pinned value disagrees. Run it after changing
  anything here. It is a static reader, so it sees only what the spec pins and nothing
  geometric — it prints its own limits on every run, and they are worth reading, because a
  green run there means less than it looks like it does. (This paragraph said "no cross-repo
  test" until that script existed; the annotation forty lines below already cited the guard by
  name, so the file was contradicting itself.)

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
    opaque background of its own*, and, since it exists, by
    `../mysuite/tools/check-contract.py`: deleting it fails that guard with
    `§5.3 backdrop is as recorded` in both themes, exit 1. Re-verified by deleting it rather
    than reasoned from the guard's source. Nothing else catches it — the other six colour
    assertions stayed green, because the backdrop walk falls straight through to `<body>` and
    finds the identical colour. The colour is redundant; the opacity is not, and content
    slides through the buttons when the month view scrolls. (This said "and **by nothing
    else**" while the guard already caught it; the annotation predated the script.)
  - resolving `.sidebar-footer-btn`'s `color` or hover `background` straight off
    `--text-subtle` / `--hover-bg` instead of the `--sidebar-footer-*` aliases — caught in
    light by *resting label meets 4.5:1* (4.393:1) and *hover fill is distinguishable from
    its backdrop* (1.000:1). Those two aliases are light-only deviations and exist because
    MyCal's light `--bg` is `#f3f4f6`, which is also its `--hover-bg`.
  - collapsing those aliases into one unscoped pair — caught by *the dark aliases name the
    shared tokens*, and inside this repo by nothing else: the six ratio assertions stayed
    green when it was applied. The two halves are not equally exposed, though, and it is
    worth knowing which is which. Deleting the **label** alias makes dark resolve
    `--text-muted` `#d1d5db`, a genuinely different colour, so the cross-repo guard catches
    it too (verified — `FAIL §5.1 resting text [dark]`). Deleting the **fill** alias makes
    dark resolve `--border`, which *is* `#374151`, the same hex as `--hover-bg` — so the
    wrong token yields the right colour and the guard passes clean (also verified). The
    CSSOM assertion is the only check anywhere that holds that half.
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
  - `e2e/tests/sidebar-footer.spec.ts` is this repo's half of the contract, and the only half
    that runs anywhere — not the only one that exists. MyMail and MyNotes both have suites of
    their own now, on unpushed branches whose CI steps have never executed. The distinction is
    coverage *in effect* versus coverage on disk, and it is the whole difference: a suite
    nobody has run is not a guard.

    That is a claim about two other repositories, so it dates the moment either of them
    pushes. `../mysuite/spec/sidebar-footer.md` §9.1 carries the live status of all three and
    is the authority — do not re-derive it from this line, and correct it there first. (This
    read "MyMail and MyNotes have no e2e suite at all" until both grew one, which is the shape
    to watch for: it stayed grammatical and confident while quietly becoming false.)

    It runs in CI (`.github/workflows/main.yml` → `./test-e2e.sh`) and gates publishing. Note what that
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
  blue badge — so a change to either belongs in both. **The badge around it is a three-repo
  contract** — see *The app logo is governed from outside this repo* below before touching either.
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

## The app logo is governed from outside this repo

The badge at the top left — `.brand-logo` in `web/static/app.css`, drawn by
`web/ts/components/Logo.tsx`, rendered inside `.brand` in `web/ts/app.tsx` — implements a
contract shared with the sibling MyMail and MyNotes apps. **The badge is shared; the mark inside
it is MyCal's own** — that distinction is the contract (§6), so the three badges are the same
size and colour and sit in the same place in their own chrome while each app draws its own
picture. It is defined in the sibling
`mysuite` repository — [`../mysuite/spec/app-logo.md`](../../mysuite/spec/app-logo.md) — and **not
here**. Its values are not restated in this repo; read them there, along with
[`../mysuite/spec/measurement-protocol.md`](../../mysuite/spec/measurement-protocol.md), which is
binding on any number you report about it. **Changing any of it is a change in all three
repositories.**

**Know what it pins before you go looking for a rule that is not there.** The contract pins the
badge's appearance and its *placement* — top left of the app's own chrome, ahead of the app-name
label in reading order — and deliberately **not** its window coordinates (§4). MyCal is the
reason: our badge is centred in `.top-bar`, whose height is set by the `<h1>` date heading
wrapping, so its vertical position moves with viewport width, view, date, locale and root font
size, and the page-scrolling views carry it off-screen entirely (§4.1, §9.1). Do not "restore" a
coordinate you found in an old comment or report — there has never been one to restore.

**Edits that break it silently.** Each is an ordinary tidy-up that leaves `./build.sh` and the
Go tests green, and each is MyCal-specific — the sibling repos break in different places.

- **Routing `Logo.tsx` through the Lucide bundle**, or **changing its `viewBox` or
  `stroke-width` without changing `favicon.svg` too** — or the reverse. Both break the favicon
  parity the icons list above already states; the contract records it (§6.3) but leaves it
  MyCal's own, and nothing anywhere compares the two files.
- **Moving the glyph's size out of `.brand-logo svg` onto the SVG as `width`/`height`
  attributes.** The contract mandates the glyph *renders* at its size, measured — never "the
  attribute says so" (§3.2) — precisely because the two shipped apps set it in different layers.
  **The failure here is delayed, which is what makes it dangerous, and the direction is the
  opposite of the obvious one:** an author CSS rule beats a presentation attribute, so adding the
  attributes changes nothing at all while `.brand-logo svg` stands. The size only jumps the day
  someone deletes that "now-redundant" CSS rule. *(Verified by setting the attributes on the live
  SVG and re-measuring: the box stayed put until the CSS rule was removed, then followed the
  attributes.)*
- **Adjusting `--primary` to make the badge read better.** It is a cross-contract operand (§7.3):
  it is also the focus-outline colour in `.sidebar-footer-btn:focus-visible`, which
  [`../mysuite/spec/sidebar-footer.md`](../../mysuite/spec/sidebar-footer.md) §6.2 holds to a WCAG
  1.4.11 obligation — a different contract, in a section nobody editing a logo would think to
  open. It also takes the badge fill out of this contract's §7.1 in both themes at once. If you
  think the fill needs changing, say so upstream rather than changing it.
- **Changing `.brand`'s `gap`.** Pinned by §3.4, which is authority for the value — and
  load-bearing outside this repo: it is part of the width MyNotes is being widened to in order to
  fit its own badge (§10.1). It looks like a local spacing tweak and is not.
- **Touching the badge's horizontal or vertical anchor.** `--app-padding-x` sets the badge's
  distance from the window's left edge and **cannot move the footer**, which cancels that exact
  token with a negative margin — a token the footer contract is immune to and the logo is not
  (§9.5) — and it is redefined again inside the narrow-screen media block in `app.css`.
  `.top-bar`'s `min-height` is the vertical equivalent; its comment at the point of use says what
  it is holding, and why the bar is not allowed to collapse to its contents.
- **"Fixing" the `@media (max-width: 600px)` rule that hides the badge.** The declaration is
  `.brand-logo, .brand-name { display: none }` — **one rule, two selectors**, so splitting the
  list to touch the label is the same edit with the same effect. It is a **sanctioned exemption**
  recorded with MyCal's reason in §9.3: there is no room for the mark and the label, and Reload
  rides in the same block and must stay reachable. Leave it, and note it is MyCal's alone — not a
  licence for another app to hide its badge.
- **Redrawing the mark smaller inside its `viewBox`.** §3.3 sets a floor on how much of the glyph
  box the rendered ink must span, on the larger axis. A redraw that adds breathing room around the
  mark passes every box-and-glyph-box check while visibly failing "the three look like one
  product" — which is the whole reason that rule exists.

**Two things that are not breakages but will cost you an afternoon:**

- **`getByText('8')` matches the logo.** The mark draws a real `<text>8</text>`, so
  `.brand-logo`'s `textContent` is `"8"` and an exact-text locator matches it along with the day
  numbers (§9.5). It is hidden from the accessibility tree but not from the DOM, so a header
  assertion written by text can land on the mark.
- **The demo build is a second shipped surface** with different badge geometry, and it is what
  GitHub Pages publishes. Demo mode swaps the top bar's action cluster — three buttons out, the
  demo badge in — and that badge's detail span is clipped at its own breakpoint, so the `<h1>`
  wraps at different widths and the badge's vertical position follows a different, **non-monotonic**
  curve (§4.1, §9.5). *(Measured — the clip and the jump flip at the same width.)*

**What catches any of this: essentially nothing, and that is a coverage sweep, not a red run.**
Sweeping every spec in `e2e/tests/`, the frontend unit tests and
`../mysuite/tools/check-contract.py` finds exactly one assertion touching the badge —
`e2e/tests/calendar-views.spec.ts:12`, `expect(page.locator('.brand-logo svg')).toBeVisible()` —
and it is satisfied by any non-empty box, so it cannot distinguish one glyph size from another.
Nothing pins the size, colour, radius, position, centring or the `aria-hidden`. The cross-repo
guard does not know the logo exists; a check there is deliberately deferred until MyNotes lands
(§9.4). **Nothing here has been mutation-tested**, so read the list as "nothing asserts this" —
the weaker and the honest claim. The prose above is the only guard that fires before the edit.

## Demo mode

- **Intercepting at the network layer is the point**: the frontend is unchanged
  between demo and real, so nothing in `views/` or `components/` needs to know.
- **The accepted divergences** are the ones the demo notice lists
  (`components/DemoDialog.tsx`) — don't add more silently. Today: iCalendar
  import/export, feed subscriptions, MyMail sharing, MyNotes linking, and
  **recurring events**, which the emulated backend stores but never expands into
  occurrences. Each one is hidden in the UI rather than left to fail: see the
  `demo` guards in `app.tsx`, `Settings.tsx` and `EventForm.tsx`.
- The `web/ts/demo/` sources are **worker code**: excluded from `web/ts/tsconfig.json` and
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
