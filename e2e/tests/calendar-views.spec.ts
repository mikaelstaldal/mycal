import { test, expect, type Page } from '@playwright/test';
import { clearAllEvents } from './helpers';

test.describe('Calendar Views', () => {
  test.beforeEach(async ({ page, request }) => {
    await clearAllEvents(request);
    await page.goto('/');
  });

  test('app shell renders the brand mark and label', async ({ page }) => {
    await expect(page.locator('.brand-name')).toHaveText('MyCal');
    await expect(page.locator('.brand-logo svg')).toBeVisible();
  });

  // The badge's window position is a three-repo contract value
  // (../mysuite/spec/app-logo.md §4). It used to be a *remainder*: .brand was
  // centred in .top-bar, so the badge's y was whatever the date heading's line
  // count left over — measured 14 at 1920 but 27.609 at 1280 and 44.406 at 1024,
  // and it differed per view because the heading text does.
  //
  // So the widths below are not arbitrary: 1280 and 1024 are widths where the
  // heading wraps and the bar grows, which is exactly where the old mechanism
  // failed. The assertion that the bar is TALLER than 40px at 1280 is the
  // precondition — without it this test would keep passing against a build that
  // had simply stopped the bar growing, which is the other way to break it.
  test('the brand badge holds (16, 14) however the heading wraps', async ({ page }) => {
    const badge = page.locator('.brand-logo');
    // Exact values, not toBeCloseTo: both offsets are authored in px, so every
    // reading here is a whole number. A tolerance would swallow the half-pixel
    // drift that is the first sign of the position becoming computed again.
    const at = async (label: string) => {
      const box = (await badge.boundingBox())!;
      expect(box.x, `x ${label}`).toBe(16);
      expect(box.y, `y ${label}`).toBe(14);
      return box;
    };

    for (const width of [1920, 1366, 1280, 1152, 1024]) {
      await page.setViewportSize({ width, height: 720 });
      await expect(badge).toBeVisible();
      const box = await at(`at ${width}px`);
      expect(box.width, `width at ${width}px`).toBe(28);
      expect(box.height, `height at ${width}px`).toBe(28);
    }

    // Per view, because the heading text differs and so does whether it wraps.
    // Each iteration asserts the view actually changed before measuring: a click
    // that silently no-ops would otherwise leave this re-measuring the previous
    // view and still passing. (Below ~1000px the search input overlaps these
    // buttons and the click really can fail to land.)
    await page.setViewportSize({ width: 1280, height: 720 });
    const views: [string, string][] = [
      ['Year', '.year-view'],
      ['Month', '.calendar-grid'],
      ['Week', '.week-view'],
      ['Day', '.day-view-grid'],
      ['Schedule', '.schedule-view'],
    ];
    let viewsWithAGrowingBar = 0;
    for (const [view, marker] of views) {
      await page.getByRole('button', { name: view, exact: true }).click();
      await expect(page.locator(marker), `${view} view did not render`).toBeVisible();
      await at(`in ${view} view at 1280px`);
      if ((await page.locator('.top-bar').boundingBox())!.height > 40) viewsWithAGrowingBar++;
    }

    // The precondition, and it is what stops this test being satisfiable the
    // wrong way: making the badge sit at 14 by stopping the heading wrapping is
    // not the fix, and a position assertion alone would go green against it.
    //
    // Counted rather than asserted per view on purpose. WHICH headings wrap at
    // 1280px depends on today's date and the host's locale — `Aug 2 – 8, 2026`
    // wraps and `2026` does not — so naming the views would pin this suite to a
    // calendar. What must hold on any date is that at least one of them did.
    expect(viewsWithAGrowingBar,
      'no view wrapped its heading at 1280px, so the growing-bar case went untested')
      .toBeGreaterThan(0);

    // A larger root font, which is the only case that can see `.brand-logo`'s
    // own `align-self`. At a 16px root the badge is the tallest thing in .brand
    // — by 1.61px — so it lands at 14 whether or not it opts out of .brand's
    // centring, and deleting that declaration changes nothing measurable here.
    // Raise the root and the rem-sized label overtakes it: without the opt-out
    // the badge is centred in a taller row and drops to 19.797px.
    //
    // That is the shape the sidebar-footer suite keeps meeting — a pin this
    // app's default rendering cannot defend. Asserting the behaviour at a root
    // size that exposes it beats asserting the declaration text.
    for (const rootSize of ['20px', '24px']) {
      await page.evaluate(size => { document.documentElement.style.fontSize = size; }, rootSize);
      await at(`at a ${rootSize} root font`);
    }
    await page.evaluate(() => { document.documentElement.style.removeProperty('font-size'); });
  });

  // The app-name label beside the badge. Its typography and its placement are a
  // three-repo contract, defined in the sibling `mysuite` repository and NOT
  // here — the same arrangement as the badge above.
  // `../mysuite/spec/app-name-label.md` is the authority; `app-logo.md` §2
  // records the earlier ruling that put the label out of scope, and
  // `app-name-label.md` §2.1 records the owner reopening it. Values are not
  // restated here beyond what these assertions have to name.
  //
  // Everything below is read off the RENDERED page. That is not a stylistic
  // preference here: MyCal declares no `font-size` and no `font-weight` on
  // `.brand-name` at all — both are on `.brand` and reach the label by
  // inheritance — so an assertion that read a rule off `.brand-name` would find
  // nothing, and one that read `.brand` would keep passing after a refactor that
  // moved the text somewhere else. (`.brand-name` staying free of typography is
  // itself enforced, from outside this repo — see web/AGENTS.md.)
  const brandLabel = (page: Page) => page.evaluate(() => {
    const brand = document.querySelector('.brand') as HTMLElement;
    const badge = document.querySelector('.brand-logo') as HTMLElement | null;
    // Every "could not measure" leaves by this door, carrying its own reason.
    // The alternative — one flag for all of them — cannot tell a label that
    // vanished from a label that merely moved, and the caller then reports the
    // wrong thing confidently. `app-name-label.md` §6.1.
    const fail = (reason: string, n: number, badgeFound: boolean) =>
      ({ measured: false as const, reason, labelTextNodeCount: n, badgeFound });

    // The label's text node, found by walking DESCENDANTS rather than by
    // selector or by direct children — see above for why the element is not the
    // thing to trust, and note that a direct-children search would report a
    // wrapped label as a missing one (`app-name-label.md` §6.1 requires the
    // descendant search for exactly that reason).
    //
    // The badge subtree is EXCLUDED deliberately. MyCal's mark draws a real
    // <text>8</text>, so `.brand-logo`'s textContent is "8" and a naive text
    // walk finds the logo before it finds the label (web/AGENTS.md records the
    // same trap for `getByText('8')`). Without this filter a build that lost the
    // label entirely would measure the mark's "8" and report a font size.
    const texts: Text[] = [];
    const walk = document.createTreeWalker(brand, NodeFilter.SHOW_TEXT);
    for (let n = walk.nextNode(); n; n = walk.nextNode()) {
      if (!n.textContent?.trim()) continue;
      if (badge?.contains(n)) continue;
      texts.push(n as Text);
    }
    if (texts.length !== 1) {
      return fail(`expected exactly one label text node in .brand, found ${texts.length}`, texts.length, !!badge);
    }
    // The badge is the frame every offset below is measured against, so a badge
    // with no box makes them plausible-looking nonsense rather than an error —
    // and it HAS no box below 600px, where this helper will eventually be
    // called. Zeros must not leave this function as measurements.
    if (!badge) return fail('.brand-logo is absent', texts.length, false);
    const b = badge.getBoundingClientRect();
    if (b.width === 0 || b.height === 0) {
      return fail('.brand-logo has no box — hidden at this width?', texts.length, false);
    }

    const text = texts[0];
    // Resolved at runtime from whatever element the text actually inherits from,
    // and reported, so that moving the text breaks this loudly instead of
    // leaving a green assertion pointed at the wrong element.
    const host = text.parentElement as HTMLElement;
    const cs = getComputedStyle(host);

    // BOTH boxes, each named. They are different measurements and the
    // difference is the half-leading: the block box is `line-height` tall, the
    // ink box is the text run the reader actually sees inside it. Quoting one
    // where the other is meant is how two apps that agree can look 2px apart.
    const block = host.getBoundingClientRect();
    const range = document.createRange();
    range.selectNodeContents(text);
    const ink = range.getBoundingClientRect();

    return {
      measured: true as const,
      reason: '',
      labelTextNodeCount: texts.length,
      badgeFound: true,
      text: text.textContent,
      // Every number below is in px and half of them are rem-derived, so the
      // root size is part of the measurement, not context for it.
      rootFontSize: getComputedStyle(document.documentElement).fontSize,
      // Which element was measured. `.brand-name` today; anything else means
      // the markup moved and every number below is about a different thing.
      hostIsBrandName: host.classList.contains('brand-name'),
      hostTag: host.tagName,
      fontSize: cs.fontSize,
      fontWeight: cs.fontWeight,
      fontFamily: cs.fontFamily,
      lineHeight: cs.lineHeight,
      blockBox: { x: block.x, y: block.y, w: block.width, h: block.height, bottom: block.bottom },
      inkBox: { x: ink.x, y: ink.y, w: ink.width, h: ink.height, bottom: ink.bottom },
      // Half-leading, top and bottom. Recorded, not pinned — see the test.
      leadingTop: ink.top - block.top,
      leadingBottom: block.bottom - ink.bottom,
      // Measured to the INK's leading edge, not read off the `gap` property and
      // not taken from the badge's container. Both boxes' gaps are returned so
      // they can be cross-checked against each other rather than against an
      // expectation.
      gapToInk: ink.left - b.right,
      gapToBlock: block.left - b.right,
      badgeBox: { x: b.x, y: b.y, w: b.width, h: b.height },
    } as const;
  });

  // Asserted at a 16px root SPECIFICALLY, and the reason is the whole point of
  // the test rather than a detail of it.
  //
  // The label is centred in `.brand` (it does not opt out of the flex
  // `align-items: center` the way `.brand-logo` does above), so its y is
  // 14 + max(0, (.brand height − label height) / 2), where `.brand`'s height is
  // its tallest child. The label's own height is 1.5 × 1.1 × root = 1.65 × root,
  // which overtakes the 28px badge at a root of ~16.97px. So the term is
  // non-zero ONLY below ~17px:
  //
  //     16px root -> label y = 14.797   (badge is the tallest thing in the row)
  //     18px root -> label y = 14
  //     24px root -> label y = 14
  //     32px root -> label y = 14
  //
  // A sweep that omitted 16 would find the label perfectly aligned and would be
  // measuring the one case where the offset is absent. The default root is both
  // the only case that is off AND the case every reader sees, so it is the one
  // that has to be pinned.
  test('the app-name label renders at the contract typography, at a 16px root', async ({ page }) => {
    const m = await brandLabel(page);

    // Establish what was measured before believing any of it. A build that lost
    // the label, or grew a second text child in `.brand`, fails here rather than
    // reporting numbers about something else.
    expect(m.measured, m.reason).toBe(true);
    if (!m.measured) return;   // narrows the union; the assertion above fails the run
    expect(m.text).toBe('MyCal');
    expect(m.hostIsBrandName, `label text now lives in <${m.hostTag}>, not .brand-name`).toBe(true);

    // The root size is a precondition, not context. Every px figure below is
    // 1.1rem or 1.5 x 1.1rem of it, and the whole argument for asserting at 16
    // collapses if the browser's default is something else — without this the
    // failure reads as a bare "expected 17.6px" and points at the CSS.
    expect(m.rootFontSize, 'not a 16px root — every figure below is derived from it').toBe('16px');

    // Typography. Resolved values, on the element the text inherits from.
    // 17.6px is 1.1rem at this suite's 16px root — a rem value in the CSS,
    // asserted in px because that is what getComputedStyle returns. The rem-ness
    // itself is asserted at the second root below.
    expect(m.fontSize).toBe('17.6px');
    expect(m.fontWeight).toBe('600');
    expect(m.lineHeight).toBe('26.4px');   // body's unitless 1.5, so 1.5 x 17.6
    // The DECLARED stack, which is machine-independent and is what the contract
    // pins. Which face the platform resolves `system-ui` to is a per-machine
    // reading and is pinned by nothing, here or upstream.
    expect(m.fontFamily, 'the label inherits its stack from body — nothing nearer declares one')
      .toBe('system-ui, -apple-system, "Segoe UI", Roboto, sans-serif');

    // Placement, relative to the badge.
    expect(m.badgeBox.y, 'badge moved — every offset below is measured from it').toBe(14);
    expect(m.gapToInk, 'gap from the badge to the label ink').toBe(8);
    // The two gaps must agree: the label's box starts at its text today, and if
    // that stops being true (padding, an indent, a stray inline element) the
    // single number everyone quotes has silently become ambiguous. Compared
    // loosely on purpose — the ink edge comes from a Range rect and carries the
    // face's left side bearing, so exact equality here would pin a per-machine
    // reading, which is the thing the ink-box assertions below refuse to do.
    expect(m.gapToBlock, 'block-box gap disagrees with ink-box gap').toBeCloseTo(m.gapToInk, 1);

    // The label's own y. This is a REMAINDER, not an authored offset, and it is
    // one by decision: the owner declined to move three shipping apps' labels by
    // 0.8px to author it, and the contract names the operands instead
    // (`app-name-label.md` §4.2 for the path, §4.3 for the ruling). The operands
    // are `.brand`'s tallest item — so `.brand-reload-btn`'s padding and the
    // label's own line-height — and web/AGENTS.md carries them.
    //
    // The exact value is 14.796875: 14 + (28 − 26.390625) / 2, where 26.390625
    // is the engine's 1/64px snap of a 26.4px line box. Asserted to 3 decimals
    // rather than exactly, so a future engine's snap does not fail this for a
    // non-defect — and that tolerance cannot swallow what this exists to catch,
    // since padding Reload by 20px moves it to 28.796875, i.e. by 14px.
    expect(m.blockBox.y, 'label y at a 16px root').toBeCloseTo(14.796875, 3);
    expect(m.blockBox.h, 'label block box is its line box').toBeCloseTo(26.4, 1);
    expect(m.blockBox.x, 'label x = .app padding + badge + gap, all px').toBe(52);

    // The ink box is INSIDE the block box, and the difference is the
    // half-leading. Recorded as a relationship rather than pinned to a number:
    // the ink box's height and width come from the resolved face's metrics, and
    // `system-ui` resolves to different faces on different machines — CI's is
    // wider than a typical developer box's. Pinning either would go red on a
    // font change, which is not a defect in this app.
    expect(m.inkBox.w, 'label ink has no width — nothing was rendered').toBeGreaterThan(0);
    expect(m.blockBox.w, 'label block box has no width').toBeGreaterThan(0);
    expect(m.leadingTop, 'ink escapes the top of its line box').toBeGreaterThanOrEqual(0);
    expect(m.leadingBottom, 'ink escapes the bottom of its line box').toBeGreaterThanOrEqual(0);
  });

  // The control for the test above, and the reason it is a separate test rather
  // than another loop iteration: it demonstrates that the 16px assertion is the
  // tight one. At 24px the label is taller than the badge, the centring term
  // goes to zero, and the label lands on the badge's own 14 — so this passes
  // whether or not anything is wrong at the default root.
  //
  // It earns its place by pinning the other half: the label is `rem` and the
  // badge is `px`, and 26.4px here against 17.6px above is what distinguishes
  // the two. A `font-size: 17.6px` "simplification" would pass every assertion
  // in the test above and fail this one.
  test('the label is rem while the badge is px — and at a 24px root the offset is gone', async ({ page }) => {
    await page.evaluate(() => { document.documentElement.style.fontSize = '24px'; });
    const m = await brandLabel(page);
    expect(m.measured, m.reason).toBe(true);
    if (!m.measured) return;
    // Same two preconditions as its sibling: the root must actually have taken,
    // and the element measured must still be the label. Without the second, a
    // build where the text moved reports the right size off the wrong element
    // here while failing loudly in the test above.
    expect(m.rootFontSize, 'the root size did not take').toBe('24px');
    expect(m.hostIsBrandName, `label text now lives in <${m.hostTag}>, not .brand-name`).toBe(true);

    expect(m.fontSize, '1.1rem at a 24px root').toBe('26.4px');
    expect(m.badgeBox.w, 'the badge is px and must NOT have grown').toBe(28);
    expect(m.badgeBox.h).toBe(28);
    expect(m.badgeBox.y).toBe(14);
    expect(m.gapToInk, 'the gap is px too').toBe(8);
    expect(m.blockBox.y, 'at this root the label is the tallest thing in .brand').toBeCloseTo(14, 2);

    await page.evaluate(() => { document.documentElement.style.removeProperty('font-size'); });
  });

  // `.brand-name`'s four declarations keep a long app name inside the column.
  // "MyCal" is far too short to exercise any of them — it renders about 50px
  // into a 132px box on this machine — so this substitutes a name long enough to
  // force the case. Nothing else in the suite can reach it.
  //
  // WHICH declaration does the work is not what the comment above the rule says,
  // and it was established by deleting them one at a time and re-running rather
  // than by reading the spec:
  //
  //   delete `min-width: 0`      -> nothing changes at all. Green, correctly.
  //   delete `overflow: hidden`  -> the box still holds its 132px and the text
  //                                 still measures 380, so nothing here can see
  //                                 it — what changes is that the overflow is
  //                                 no longer clipped, which is paint, and
  //                                 nothing in this suite reads paint.
  //   delete both                -> the label's box grows to its content and
  //                                 Reload leaves the column. Red, below.
  //
  // `overflow: hidden` is what zeroes a flex item's automatic minimum size (the
  // rule's own comment says so, and it is right) — so `min-width: 0` is inert
  // while it stands, and deleting it alone is genuinely a no-op. It is not
  // pointless: it is the declaration that keeps this working if `overflow` ever
  // changes. That makes it the delayed-hazard shape §5 of the logo contract
  // describes for the glyph — the edit that arms the breakage and the edit that
  // fires it are separate, and neither looks wrong alone.
  test('a long app name is truncated instead of pushing Reload out of the column', async ({ page }) => {
    const label = page.locator('.brand-name');
    const reload = page.getByRole('button', { name: 'Reload' });
    const brand = page.locator('.brand');

    const before = (await brand.boundingBox())!;
    await label.evaluate(el => { el.textContent = 'MyCalendarApplicationWithAVeryLongName'; });

    // The precondition is about the FIXTURE, not about the outcome: the text has
    // to be too long for the whole brand block, so that truncating it is the
    // only way to fit. Measured off a Range, which gives the text run's real
    // width whether or not its box is clipping it — `scrollWidth` cannot do this
    // job, because when the defect is present the box grows and scrollWidth and
    // clientWidth agree at the overflowed size.
    const ink = await label.evaluate(el => {
      const r = document.createRange();
      r.selectNodeContents(el.firstChild!);
      return r.getBoundingClientRect().width;
    });
    expect(ink, 'the substituted name is not longer than the brand block — nothing was forced')
      .toBeGreaterThan(before.width);

    const box = await label.evaluate(el => el.clientWidth);
    expect(box, 'the label box grew to fit the text instead of truncating it').toBeLessThan(ink);

    const [after, btn] = await Promise.all([brand.boundingBox(), reload.boundingBox()]);
    // `.brand` is a fixed `width` with `flex-shrink: 0` and no `flex-grow`, so
    // today it CANNOT grow and this cannot fail. Kept deliberately: it is the
    // assertion that fires the day that width becomes `fit-content`, `auto` or a
    // `min-width`, which is the obvious way to "fix" a truncated label.
    expect(after!.width, '.brand is no longer a fixed width — it grew to fit the label').toBe(before.width);
    expect(btn!.x + btn!.width, 'Reload was pushed out of the brand block')
      .toBeLessThanOrEqual(after!.x + after!.width);
  });

  // The four declarations, read as computed values.
  //
  // This is the assertion shape `tests/sidebar-footer.spec.ts` uses and for the
  // same reason: MyCal's own rendering cannot detect two of these going missing
  // (see the mutation table above), so geometry alone would report coverage it
  // does not have. `min-width` in particular is inert TODAY and load-bearing the
  // moment `overflow` changes, which is precisely the case a rendered
  // measurement cannot see.
  //
  // Values, not the rule text: this reads the resolved style off the element,
  // so it survives the rule being renamed, merged or moved, and fails only when
  // the label actually stops being clipped.
  test('the label keeps the declarations that clip it, including the inert one', async ({ page }) => {
    const cs = await page.locator('.brand-name').evaluate(el => {
      const s = getComputedStyle(el);
      return { minWidth: s.minWidth, overflowX: s.overflowX, textOverflow: s.textOverflow, whiteSpace: s.whiteSpace };
    });
    expect(cs.minWidth, 'inert while overflow is hidden — kept so that stops being load-bearing').toBe('0px');
    expect(cs.overflowX, 'this is what actually zeroes the flex minimum size').toBe('hidden');
    expect(cs.textOverflow).toBe('ellipsis');
    expect(cs.whiteSpace, 'a wrapping label would grow .brand vertically instead').toBe('nowrap');
  });

  test('shows week view by default with correct heading', async ({ page }) => {
    const heading = page.locator('nav h1');
    await expect(heading).toBeVisible();

    const now = new Date();
    await expect(heading).toContainText(String(now.getFullYear()));

    await expect(page.locator('.week-view')).toBeVisible();
  });

  test('navigate months with prev/next buttons', async ({ page }) => {
    const heading = page.locator('nav h1');
    const nav = page.getByRole('navigation');
    const initialText = await heading.textContent();

    await nav.getByRole('button', { name: 'Next' }).click();
    await expect(heading).not.toHaveText(initialText!);

    await nav.getByRole('button', { name: 'Previous' }).click();
    await expect(heading).toHaveText(initialText!);
  });

  test('today button returns to current week', async ({ page }) => {
    const heading = page.locator('nav h1');
    const nav = page.getByRole('navigation');
    const initialText = await heading.textContent();

    // Navigate away
    await nav.getByRole('button', { name: 'Next' }).click();
    await nav.getByRole('button', { name: 'Next' }).click();
    await expect(heading).not.toHaveText(initialText!);

    // Click Today
    await page.getByRole('button', { name: 'Today' }).click();
    await expect(heading).toHaveText(initialText!);
  });

  test('switch to Week view', async ({ page }) => {
    await page.getByRole('button', { name: 'Week' }).click();
    await expect(page.locator('.week-view')).toBeVisible();
  });

  test('switch to Day view', async ({ page }) => {
    await page.getByRole('button', { name: 'Day', exact: true }).click();
    await expect(page.locator('.day-view')).toBeVisible();
  });

  test('switch to Year view', async ({ page }) => {
    await page.getByRole('button', { name: 'Year' }).click();
    await expect(page.locator('.year-view')).toBeVisible();
    await expect(page.locator('.year-month')).toHaveCount(12);
  });

  // The three views that scroll internally are sized by flexing to the bottom of
  // a viewport-height .app. They used to be capped with a max-height offset that
  // guessed the chrome above them, which left dead space under one and pushed
  // another off the bottom of the page — hence measuring the gap from both
  // sides. These assume the two-column layout: below 600px the media query
  // stacks the sidebar under the view and the gap is the sidebar's height.
  for (const [name, viewSelector, bodySelector] of [
    ['Week', '.week-view', '.week-body'],
    ['Day', '.day-view', '.day-view-body'],
    ['Schedule', '.schedule-view', '.schedule-view'],
  ] as const) {
    test(`${name} view fills the viewport without scrolling the page`, async ({ page }) => {
      await page.getByRole('button', { name, exact: true }).click();
      await expect(page.locator(viewSelector)).toBeVisible();

      const { viewBottom, footerBottom, bodyHeight, innerHeight } = await page.evaluate(
        ([view, body]) => ({
          viewBottom: document.querySelector(view)!.getBoundingClientRect().bottom,
          footerBottom: document.querySelector('.sidebar-footer')!.getBoundingClientRect().bottom,
          bodyHeight: document.querySelector(body)!.clientHeight,
          innerHeight: window.innerHeight,
        }),
        [viewSelector, bodySelector],
      );

      // The view ends at .app's 8px bottom padding: no dead space under it, and
      // no overhang past it either.
      expect(innerHeight - viewBottom).toBeGreaterThanOrEqual(0);
      expect(innerHeight - viewBottom).toBeLessThanOrEqual(10);
      // The sidebar's footer box ends on that same edge. Its buttons sit 8px
      // above it — that inset is the shared MySuite footer spec — so what is
      // pinned here is the box, not the glyphs.
      expect(Math.abs(footerBottom - viewBottom)).toBeLessThanOrEqual(1);
      // What fills the view is the scrollable body, not chrome squeezing it out.
      expect(bodyHeight).toBeGreaterThan(200);
    });
  }

  // The left column is height-bound in those views, so its panels have to hold
  // their size and let the column scroll instead. A .mini-month squeezed to its
  // borders is the failure this pins; a short window is the cheap way to force
  // the overflow, but a long enough calendar list does it at any height.
  test('left sidebar scrolls rather than squeezing the mini month', async ({ page }) => {
    await expect(page.locator('.week-view')).toBeVisible();
    const miniHeight = () =>
      page.evaluate(() => document.querySelector('.mini-month')!.getBoundingClientRect().height);

    // Measured with room to spare, so the assertion below is against this
    // machine's own rendering rather than a hardcoded pixel count.
    const relaxed = await miniHeight();
    expect(relaxed).toBeGreaterThan(0);

    await page.setViewportSize({ width: 1280, height: 300 });
    // .sidebar-content is the scrollport, not .left-sidebar: the footer is its
    // sibling so that a scrollbar here cannot narrow the footer's buttons.
    const { scrollHeight, clientHeight, footerExists, footerInsideScrollport } = await page.evaluate(() => {
      const content = document.querySelector('.sidebar-content')!;
      return {
        scrollHeight: content.scrollHeight,
        clientHeight: content.clientHeight,
        footerExists: !!document.querySelector('.sidebar-footer'),
        footerInsideScrollport: content.contains(document.querySelector('.sidebar-footer')!),
      };
    });

    expect(await miniHeight()).toBeCloseTo(relaxed, 0);
    expect(scrollHeight).toBeGreaterThan(clientHeight);
    // A scrollbar here must never narrow the footer. Asserted structurally —
    // the footer is not inside the scrollport — rather than by comparing widths:
    // a width comparison holds under the regression too (the footer would still
    // out-measure the column thanks to its negative margin) and has no teeth at
    // all under the overlay scrollbars this headless Chromium uses. Containment
    // is the actual invariant and it is deterministic on every platform.
    // Node.contains(null) is false, so without this the assertion below would
    // pass if the footer vanished entirely — absent and outside-the-scrollport
    // are different results and only one of them is the invariant.
    expect(footerExists).toBe(true);
    expect(footerInsideScrollport).toBe(false);
  });

  // Reload rides in the brand block rather than the top bar's action cluster:
  // after the label, flush with the right edge .brand shares with the column
  // below it. The narrow-screen bar drops the mark and the label, and has to
  // keep the button — which is the half of this that has already regressed once.
  test('Reload sits after the brand label, at any width', async ({ page }) => {
    const reload = page.getByRole('button', { name: 'Reload' });
    await expect(reload).toBeVisible();

    const [btn, label, sidebar] = await Promise.all([
      reload.boundingBox(),
      page.locator('.brand-name').boundingBox(),
      page.locator('.left-sidebar').boundingBox(),
    ]);
    expect(btn!.x).toBeGreaterThanOrEqual(label!.x + label!.width);
    expect(btn!.x + btn!.width).toBeCloseTo(sidebar!.x + sidebar!.width, 0);

    await page.setViewportSize({ width: 375, height: 700 });
    await expect(page.locator('.brand-name')).toBeHidden();
    await expect(reload).toBeVisible();
  });
});
