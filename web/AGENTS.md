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
  - `e2e/tests/sidebar-footer.spec.ts` is this repo's half of the contract. All three repos'
    suites now run in CI. `../mysuite/spec/sidebar-footer.md` §9.1 is the authority on that
    and carries the evidence — do not re-derive it from this line, and correct it there first.

    **Three suites running is still not a cross-repo check.** Each asserts against its own app
    and cannot see the other two, so all three stay green through a divergence. That gap is
    what `../mysuite/tools/check-contract.py` exists for, with the limits noted above.

    (This bullet has now been confidently false twice. It read "MyMail and MyNotes have no e2e
    suite at all" until both grew one; it then said their CI had never executed until all three
    went green. The second time it was **already annotated** with the warning that it dates the
    moment either repo pushes, and already pointed at §9.1 as the authority — and it went stale
    anyway. A condition written into a claim only helps if somebody notices the condition
    fired, and a push produces no commit, no diff and nothing for a review or a `grep` to catch.
    Nothing in this repo changes when a sentence about another repo goes false, so this line has
    no owner here. Read it for the lesson; go to §9.1 for the fact.)

    MyCal's own suite runs from `.github/workflows/main.yml` → `./test-e2e.sh`, and gates
    publishing. Note what that
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

**The badge's resting position is a pinned value, and MyCal is the app that made it one.** The
contract pins the badge's appearance and its placement; the owner ruled on the resting window
position after this repo's measurements showed MyCal and MyMail disagreeing at 1280px wide.
`../mysuite/spec/app-logo.md` §4 is the authority for the number — do not re-derive it from
here, and correct it there first.

**Both halves of it are authored, and that is the point.** The horizontal offset comes from
`--app-padding-x` on `.app`, and the vertical one from `.brand`'s `align-self: flex-start` plus
its `margin-top`. Neither is a remainder of anything. **This is the defect that made the rule
necessary:** the badge used to be centred in `.top-bar`, so its distance from the top was
whatever the `<h1>` date heading's line count left over — it moved with viewport width, view,
date and locale, and it was measured three different values across ordinary desktop widths in a
single view. If you find an older comment or report describing that behaviour, it predates the
fix. `e2e/tests/calendar-views.spec.ts` pins the resting position now, and asserts the bar is
still growing when it checks — so the test cannot be satisfied by stopping the heading wrapping.

The badge still scrolls away with the page in the views that scroll (§9.1); the fix is about
where it rests, not about pinning it to the window. That gap is open and is the owner's.

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
- **Touching either of the badge's two anchors.** Horizontally, `--app-padding-x` sets its
  distance from the window's left edge and **cannot move the footer**, which cancels that exact
  token with a negative margin — a token the footer contract is immune to and the logo is not
  (§9.5) — and it is redefined again inside the narrow-screen media block. Vertically, it is
  `.brand`'s `align-self: flex-start` and `margin-top`: deleting either hands the position back
  to `.top-bar`'s centring and re-creates the defect the rule exists for. Both carry their
  derivation at the point of use. `.top-bar`'s `min-height` no longer positions the **badge**, but
  it is not inert: it floors the bar, and the nav's centring rides on it — measured, `nav h1` sits
  3px higher without it. No test catches its removal, since the badge holds 14 either way. The bar
  must also stay free to grow, so do not "simplify" it into a fixed `height` either.
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
- **The demo build is a second shipped surface**, and it is what GitHub Pages publishes. Demo mode
  swaps the top bar's action cluster — three buttons out, the demo badge in — and that badge's
  detail span is clipped at its own breakpoint, so the `<h1>` wraps at **different widths there
  than in the real build, and non-monotonically**: the bar measures 40px at 1510, 67.219px from
  1505 down to 1440, and 40px again at 1439. **The badge no longer follows it** — measured at
  (16, 14) across that whole band after the offset was authored — but the swing is still there and
  will still surprise anyone measuring anything else in that bar. Re-measure the demo build
  separately; "verified in the real build" does not cover Pages.

**What catches any of this — and it is now two different answers.**

**The resting position is covered, and the guard was proved rather than trusted.**
`e2e/tests/calendar-views.spec.ts` pins the badge's x, y, width and height across five widths and
all five views, and asserts the bar is still growing while it does. It was mutation-tested in both
directions before being believed: deleting the offset declarations fails it on the y assertion
with the exact pre-fix number, and "fixing" the position by stopping the heading wrapping fails it
on the precondition instead.

**Everything else about the badge is still uncovered.** No assertion anywhere pins its colour,
radius, glyph size, centring or the `aria-hidden`; the only other one that touches it is
`calendar-views.spec.ts:12`, `expect(page.locator('.brand-logo svg')).toBeVisible()`, which is
satisfied by any non-empty box and cannot distinguish one glyph size from another. The cross-repo
guard does not know the logo exists, and a check there is deliberately deferred until MyNotes
lands. `../mysuite/spec/app-logo.md` §9.4 is the authority on suite coverage across the three
apps — correct it there first. For everything in that second group the prose above is still the
only guard that fires before the edit.

## The app-name label is governed from outside this repo

`<span class="brand-name">MyCal</span>` — the text beside the badge, rendered by `.brand` in
`web/ts/app.tsx` and styled by `.brand-name` and `.brand` in `web/static/app.css`. Its font, size
and placement are a three-repo contract, defined in the sibling `mysuite` repository —
[`../mysuite/spec/app-name-label.md`](../../mysuite/spec/app-name-label.md) — and **not here**.
Read it before changing anything below; its values are not restated in this repo.

**Cite it by filename, never by a bare section number.** Four documents in that repository now
have a §2, §3 and §4, so `app-name-label.md §4.1` is unambiguous and `§4.1` is not.

**This label used to be explicitly out of scope** — `../mysuite/spec/app-logo.md` §2 records that
ruling, with the owner's words and the condition attached. **The owner has since reopened it**
(`app-name-label.md` §2.1), so that exclusion is superseded rather than violated. If you find a
comment or report here describing the label as unspecified, it predates `app-name-label.md`.

> **That supersession is a specimen worth keeping, and it is `AGENTS.md` §3.5's shape.** The
> comment in `calendar-views.spec.ts` saying the label was out of scope was **accurate when it was
> written, in a file nobody touched, and was made false by a ruling in another repository.**
> Nothing in this repo changed, so no diff, no review and no `grep` could have caught it. It is
> the second instance in one round. **A claim here about another repo's contents has no owner in
> this repo** — when you cite one, prefer pointing at the section over restating what it says.

**Two things are MyCal's own and are recorded upstream as exemptions, not as defects:**

- **The ≤600px hide.** `@media (max-width: 600px) { .brand-logo, .brand-name { display: none } }`
  is sanctioned in `app-name-label.md` §4.4 — the owner accepted it rather than requiring the other
  two apps to match. **That one rule is now load-bearing for two contracts at once**: it is also
  the badge's exemption in `app-logo.md` §9.3. It is one rule with two selectors, so splitting the
  list to touch one element is an edit against both documents.
- **The label's vertical position is a remainder, and the contract declines to close that.**
  See below — it is the first entry in the list, because it is the one that will actually bite.

**Edits that break this silently.** Each is an ordinary tidy-up that leaves `./build.sh`, the Go
tests and most of the e2e suite green. Every "caught by" below was established by making the edit
and running the suite, not by reading the assertions.

- **Padding or resizing `.brand-reload-btn` moves the label.** This is the one to know. The label
  does not opt out of `.brand`'s `align-items: center` the way `.brand-logo` does, so its `y` is a
  remainder of the row's height — and the row's height is **its tallest item**, which includes the
  Reload button. Measured: `padding: 4px` → `20px` on that button moves the label 14px down while
  the badge does not move at all.

  **The contract knows and deliberately leaves it open.** `app-name-label.md` §4.2 records this
  exact path — MyCal's and MyMail's Reload buttons are strangers in the row — and §4.3 is the
  ruling that declines to close it: authoring the offset would move three shipping apps' labels by
  ~0.8px, so the owner constrained the observable and named the operands instead, explicitly **not**
  extending `app-logo.md` §4.2's *authored, not arrived at* to the label. **So this prose and the
  assertion below are the whole guard.**

  Note what `.brand-logo`'s comment block in `app.css` says about the same two levers. It is
  **accurate and is about the badge**, which now opts out and no longer moves. Nobody had written
  down that those levers still move the *label*, for real, today. The repo documented the hazard
  for the element it then protected, and the unprotected element beside it inherited it.
  *Caught by* `calendar-views.spec.ts` — *the app-name label renders at the contract typography*,
  on the `y` assertion, with the exact 28.797 the mutation produces.
- **Declaring `font-size`, `font-family` or `font-weight` on `.brand-name`** — including by
  "moving the label's type onto the label" from `.brand`. It changes nothing rendered when the
  values match, and MyCal has **no** typographic declaration on `.brand-name` today; everything
  reaches it by inheritance. *Caught by both.* `../mysuite/tools/check-contract.py` asserts that
  `.brand-name` declares none of those three — **MyCal is the only app that guard can exist for**,
  being the only one of the three whose label has an element at all, and it was added after a
  `font-size: 1.4rem` on this very selector, a visible three-way divergence, ran green.
  `calendar-views.spec.ts` catches it too, and only because it follows `app-name-label.md` §6.1:
  it resolves the values **off the element the text actually inherits from**, so a nearer
  declaration is what it reads (verified — `1.4rem` there fails it at `22.4px`). A suite that read
  `.brand` instead would have reproduced the same false green.
- **Changing `body`'s `font-family`.** The label's font is declared nowhere nearer —
  `.brand-name` and `.brand` both inherit it (`app-name-label.md` §3.1 pins the declared stack, and
  only the stack: what the platform resolves `system-ui` to is a per-machine reading and no
  contract can pin it). *Caught by both*, and they cover different halves. `check-contract.py`
  reads `body`'s declaration and the script itself labels that line **WEAK** — it says nothing
  about the cascade from `body` to the label, nor about which face renders. The suite asserts the
  resolved stack **on the label**, which is the cascade half, and is what fires if something
  nearer starts declaring a family. Verified by reordering the stack: both go red.
- **Normalising `.brand`'s `1.1rem`** — to `1.10rem`, `1.1em` or `17.6px`. All three are caught,
  and by different tools, which is the reason to write them out rather than as one row. Measured:

  | edit | renders identically | `check-contract.py` | the suite |
  |---|---|---|---|
  | `1.10rem` | at every root | **FAIL** | pass |
  | `1.1em` | at every root — nothing between `body` and `.brand` sets a `font-size` | **FAIL** | pass |
  | `17.6px` | at a 16px root only | **FAIL** | **FAIL** |

  So the source-text pin is the **only** guard for the first two, and the suite's 24px-root test
  is a second, independent one for the third. `1.1em` is the row to note: it is invisible to any
  number of root sizes *here*, and it would stop being invisible the day anything between `body`
  and `.brand` took a `font-size` — a condition nothing records and nothing enforces.
- **Deleting `.brand-name`'s `min-width: 0`.** **It is inert today, and the reason to keep it is
  not the reason usually given.** `overflow: hidden` is what zeroes a flex item's automatic minimum
  size — the rule's own comment says so and is right — so deleting `min-width` alone changes
  nothing rendered at all. Deleting `overflow` alone changes nothing this suite can *see* either:
  the box keeps its width and the text keeps its length, and what stops happening is the clipping,
  which is paint. Only deleting **both** grows the label's box and pushes Reload out of the
  column, and that is the only one of the three a measurement catches. So this is the
  delayed-hazard shape `app-logo.md` §5 describes for the glyph: the edit that arms the breakage
  and the edit that fires it are separate, and neither looks wrong on its own. *Caught by*
  `calendar-views.spec.ts` — *the label keeps the declarations that clip it, including the inert
  one*, which reads the resolved values and is the **only** thing that fires on the arming edit;
  the geometric test beside it fires only once both are gone.
- **Changing `.brand`'s `gap`.** It is the badge→label gap and is pinned by both contracts
  (`app-logo.md` §3.4, `app-name-label.md` §3.3), and it is load-bearing outside this repo — it is
  part of the width MyNotes was widened to. *Caught by* both label tests, which measure it to the
  label's **ink**, not off the `gap` property.
- **Changing `--sidebar-width`.** It sets `.brand`'s width and therefore how much room the label
  has before it truncates. *Caught by* `calendar-views.spec.ts`, but as a **Reload** failure — the
  assertion pins that button's right edge to the column's, so the report will not mention the
  label.

**What catches the label — and it is two tools, not one.** `app-name-label.md` §7 divides the
guard three ways and this repo sits on all three sides of it:

- **`../mysuite/tools/check-contract.py`**, cross-repo and static. It holds the two things no
  rendered test can: the `1.1rem` **source text**, and `.brand-name` declaring no typography.
  **Run it after changing anything in this section** — nothing runs it automatically, and the
  script says so itself on every run. It reads a working tree, not a commit.
- **`e2e/tests/calendar-views.spec.ts`**, rendered and MyCal-only. Six tests: four written for
  this contract — typography at a 16px root, the rem/px pair at 24px, the truncation geometry,
  and the clipping declarations — plus the pre-existing text and Reload-ordering checks, which
  cover the label's content and its position in the row but nothing about how it looks.
- **Neither**, for the things `app-name-label.md` §7.3 says cannot be guarded — chiefly which
  font face the platform actually resolves.

**The four new tests were mutation-tested before being believed**, per
`../mysuite/spec/measurement-protocol.md`: nine deliberate breakages, each confirmed red **on its
own assertion** and reverted green. Two cautions from that exercise, both of which cost a wrong
conclusion before they were caught:

- One run came back fully green and was *not* a gap in the test — the mutation had prepended a
  `padding` declaration that the rule's own later `padding` overrode, so nothing had actually
  changed. **A mutation that does not mutate is indistinguishable from a test that does not
  catch.** Check the edit took effect before concluding anything from a green run.
- The mutations were run through Playwright and **not** through `check-contract.py`, and three
  rows above were written as "caught by nothing" on that basis. All three were wrong; the static
  check held every one. **Run both tools before writing down what catches an edit.**

**Why the 16px root is the one that matters, and why a sweep is not a substitute.** The label's
height is `1.5 × 1.1 × root`, which overtakes the 28px badge at a root of about 16.97px. Below
that the badge is the tallest thing in the row and the label is centred against it; above it the
label sets the row's height and lands flush. **So the offset exists only at the default root — the
one every reader sees — and a test at 20px or 24px would find the label perfectly aligned and be
measuring the one case where it is absent.** Do not "improve" that test by sweeping root sizes and
dropping 16.

**Still uncovered:** the label's colour, its `text-overflow` behaviour as actually *painted*
(nothing here can see an ellipsis glyph — only that the box clips), and its appearance in the demo
build, which is a second shipped surface and is what Pages publishes. `app-name-label.md` §7.2 is
the authority on what each app's suite must assert; §8.1 carries the per-app breakdown of this
list. Correct either there first.

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
