import { test, expect, type Page } from '@playwright/test';

// The sidebar footer's two controls — the light/dark toggle and Settings — are a
// three-repo contract: they are specified to look and sit identically in MyCal,
// MyMail and MyNotes. There is no shared stylesheet, so these assertions are this
// repo's whole rendered half of it. The other half is static and lives outside
// this repo: ../mysuite/tools/check-contract.py reads all three stylesheets and
// fails on a pinned value that disagrees. Neither sees what the other does — it
// cannot see geometry, and this suite cannot see the other two apps.
// See the `.sidebar-footer-btn` block in web/static/app.css for the derivations,
// and web/AGENTS.md for the contract itself — named rather than left as a bare
// "AGENTS.md", which now resolves to this directory's own file.
test.describe('Sidebar footer contract', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  const SETTINGS = '.sidebar-footer-btn[title="Settings"]';

  // Sum the children rather than reading scrollWidth: scrollWidth's behaviour on
  // an `overflow: visible` box is not guaranteed, and this row is one. Comparing
  // the content the row must hold against the width it has is unambiguous
  // everywhere.
  const footerOverflows = (page: Page) =>
    page.locator('.sidebar-footer-actions').evaluate(el => {
      const gap = parseFloat(getComputedStyle(el).columnGap) || 0;
      const kids = [...el.children];
      const needed =
        kids.reduce((sum, k) => sum + k.getBoundingClientRect().width, 0) + gap * (kids.length - 1);
      return needed > el.clientWidth + 0.5;
    });

  // By attribute rather than `.first()`: the spec mandates toggle-then-Settings,
  // but a positional locator would silently repoint if a third control ever
  // joined the row, and every assertion here would keep passing about the wrong
  // element. This matches in *both* theme states because Preact renders
  // aria-pressed={false} as the string "false" rather than dropping the
  // attribute — true of hyphenated attribute names, not of every prop. If that
  // ever changes this selector starts missing half the time rather than
  // failing outright, so it is worth knowing why it works.
  const THEME = '.sidebar-footer-btn[aria-pressed]';

  const boxes = async (page: Page) => ({
    theme: (await page.locator(THEME).boundingBox())!,
    settings: (await page.locator(SETTINGS).boundingBox())!,
    column: (await page.locator('.left-sidebar').boundingBox())!,
    viewportHeight: page.viewportSize()!.height,
  });

  // ---------------------------------------------------------------------------
  // Position on screen — the point of the whole exercise. Measured from the
  // window, not from the sidebar: three apps can each be correct against their
  // own container and still put the buttons in three different places, which is
  // exactly what happened before this. A user with all three open in tabs must
  // see nothing move when switching between them.
  // ---------------------------------------------------------------------------

  test('controls sit 8px from the window left and bottom edges', async ({ page }) => {
    const { theme, settings, viewportHeight } = await boxes(page);

    // The two numbers that are the MySuite claim. Both font-independent.
    expect(theme.x).toBeCloseTo(8, 0);
    expect(viewportHeight - (theme.y + theme.height)).toBeCloseTo(8, 0);
    expect(viewportHeight - (settings.y + settings.height)).toBeCloseTo(8, 0);

    // Settings follows the toggle by the 6px gap. Derived from the measured
    // width rather than hardcoded: a literal would pin this machine's system-ui
    // metrics, which the CSS comment warns are not portable.
    expect(settings.x).toBeCloseTo(theme.x + theme.width + 6, 0);

    // The footer's own left edge, not just the buttons'. The CSS warns that
    // giving .app a border-left would clip the footer while the buttons' 8px
    // still measured correct — this is the assertion that would catch it.
    const footer = (await page.locator('.sidebar-footer').boundingBox())!;
    expect(footer.x).toBeCloseTo(0, 0);
  });

  // B has to be the same in every view. Month and Year lay out differently from
  // the three that fill the viewport — the page scrolls rather than the column —
  // so without .app's `min-height: 100dvh` the footer lands wherever the grid
  // happens to end (measured 96px and 170px above the window, both varying with
  // window size) and MyCal moves against itself on a view switch.
  for (const [view, selector] of [
    ['Month', '.calendar'],
    ['Week', '.week-view'],
    ['Day', '.day-view'],
    ['Year', '.year-view'],
    ['Schedule', '.schedule-view'],
  ] as const) {
    test(`controls hold (8, 8) in the ${view} view`, async ({ page }) => {
      await page.getByRole('button', { name: view, exact: true }).click();
      await expect(page.locator(selector)).toBeVisible();

      const { theme, viewportHeight } = await boxes(page);
      expect(theme.x).toBeCloseTo(8, 0);
      expect(viewportHeight - (theme.y + theme.height)).toBeCloseTo(8, 0);
    });
  }

  // Month and Year page-scroll rather than column-scroll, so the footer stays
  // reachable through `position: sticky` rather than by the column being
  // viewport-bound. `min-height: 100dvh` puts it at (8, 8) when the grid is
  // short; this covers the other half, where the grid is tall and the page
  // scrolls under it.
  for (const [view, selector] of [['Month', '.calendar'], ['Year', '.year-view']] as const) {
    test(`footer stays on screen while the ${view} view page-scrolls`, async ({ page }) => {
      await page.setViewportSize({ width: 1280, height: 400 });
      await page.getByRole('button', { name: view, exact: true }).click();
      // Wait for the view itself, so the scroll below acts on a laid-out page
      // rather than whatever height the previous view happened to leave.
      await expect(page.locator(selector)).toBeVisible();

      const scrolled = await page.evaluate(() => {
        window.scrollTo(0, document.body.scrollHeight);
        return window.scrollY;
      });
      expect(scrolled).toBeGreaterThan(0); // otherwise this asserts nothing

      const footer = await page.locator('.sidebar-footer').boundingBox();
      const height = page.viewportSize()!.height;
      expect(footer!.y).toBeGreaterThanOrEqual(0);
      expect(footer!.y + footer!.height).toBeLessThanOrEqual(height + 1);

      await expect(page.locator(SETTINGS)).toBeVisible();
    });
  }

  // ---------------------------------------------------------------------------
  // The pinned declarations
  // ---------------------------------------------------------------------------

  // Geometry assertions only catch a violation that *moves something*, and
  // several of the contract's pins deliberately do not: `font-weight: 400` is
  // what the UA button rule already gives, `flex-shrink: 0` does nothing until
  // the row is under pressure, `text-align: center` is a button's default. They
  // are pinned because the three apps reach those values by three different
  // routes and an ordinary edit in one repo would break the match invisibly —
  // which means the geometry tests cannot be what protects them.
  //
  // So assert the computed values themselves. This is font-independent and
  // platform-independent: it reads what the cascade resolved, not what the text
  // measured. It is also the only assertion here that would survive the labels
  // changing.
  test('the pinned declarations resolve to the contract values', async ({ page }) => {
    for (const selector of [THEME, SETTINGS]) {
      const cs = await page.locator(selector).evaluate(el => {
        const s = getComputedStyle(el);
        return {
          fontSize: s.fontSize, lineHeight: s.lineHeight, fontWeight: s.fontWeight,
          fontStyle: s.fontStyle, textAlign: s.textAlign, flexShrink: s.flexShrink,
          whiteSpace: s.whiteSpace, display: s.display, borderTopWidth: s.borderTopWidth,
          borderRadius: s.borderTopLeftRadius, padding: s.padding, columnGap: s.columnGap,
        };
      });
      // 0.80rem at a 16px root, on a 1.5 line box — the 29.2px acceptance height
      // is these two plus 8px padding and 2px border.
      expect(cs.fontSize, selector).toBe('12.8px');
      expect(cs.lineHeight, selector).toBe('19.2px');
      expect(cs.padding, selector).toBe('4px 8px');
      expect(cs.borderTopWidth, selector).toBe('1px');
      expect(cs.borderRadius, selector).toBe('6px');
      expect(cs.columnGap, selector).toBe('6px');
      // The rule says `inline-flex`; the computed value is `flex` because a flex
      // item's display is blockified. Asserting the computed value, not the
      // declaration — they legitimately differ here.
      expect(cs.display, selector).toBe('flex');
      expect(cs.whiteSpace, selector).toBe('nowrap');
      // The three pinned inherited properties, and the one that keeps overflow
      // rather than a silent squeeze as the failure mode.
      expect(cs.fontWeight, selector).toBe('400');
      expect(cs.fontStyle, selector).toBe('normal');
      expect(cs.textAlign, selector).toBe('center');
      expect(cs.flexShrink, selector).toBe('0');
    }

    // The row itself.
    const row = await page.locator('.sidebar-footer-actions').evaluate(el => {
      const s = getComputedStyle(el);
      return { display: s.display, flexWrap: s.flexWrap, columnGap: s.columnGap };
    });
    expect(row.display).toBe('flex');
    expect(row.flexWrap).toBe('nowrap');
    expect(row.columnGap).toBe('6px');
  });

  // Some pins cannot be checked by computed value at all, because in MyCal the
  // UA default already equals the contract value: `font-weight: 400` and
  // `text-align: center` are what a <button> computes with or without our rule.
  // Deleting either changes nothing here — and would change everything in
  // MyNotes, whose `button { font: inherit }` means its buttons take those from
  // `body` instead. That asymmetry is exactly why the contract pins them, and it
  // means the app where the value is already right is the app whose rendering
  // cannot detect the pin going missing.
  //
  // So read the rule itself out of the CSSOM. This asserts the declaration is
  // present, not merely that the result looks right.
  test('the pinned declarations are actually declared, not inherited', async ({ page }) => {
    const declared = await page.evaluate(() => {
      for (const sheet of [...document.styleSheets]) {
        let rules: CSSRuleList;
        try { rules = sheet.cssRules; } catch { continue; } // cross-origin
        for (const rule of [...rules]) {
          if (rule instanceof CSSStyleRule && rule.selectorText === '.sidebar-footer-btn') {
            return {
              fontWeight: rule.style.fontWeight,
              fontStyle: rule.style.fontStyle,
              textAlign: rule.style.textAlign,
              flexShrink: rule.style.flexShrink,
              whiteSpace: rule.style.whiteSpace,
              fontFamily: rule.style.fontFamily,
            };
          }
        }
      }
      return null;
    });

    expect(declared, '.sidebar-footer-btn rule not found in any stylesheet').not.toBeNull();
    expect(declared!.fontWeight).toBe('400');
    expect(declared!.fontStyle).toBe('normal');
    expect(declared!.textAlign).toBe('center');
    expect(declared!.flexShrink).toBe('0');
    expect(declared!.whiteSpace).toBe('nowrap');
    // Inheriting is itself the contract here — a literal stack would stop these
    // controls following the app's own typography.
    expect(declared!.fontFamily).toBe('inherit');
  });

  // ---------------------------------------------------------------------------
  // Size and stability
  // ---------------------------------------------------------------------------

  test('controls hold one row and do not move when the theme is toggled', async ({ page }) => {
    const theme = page.getByRole('button', { name: 'Switch to dark mode' });
    await expect(theme).toBeVisible();

    const before = await page.locator(SETTINGS).boundingBox();
    const themeBefore = await theme.boundingBox();

    await theme.click();
    await expect(page.getByRole('button', { name: 'Switch to light mode' })).toBeVisible();

    const after = await page.locator(SETTINGS).boundingBox();
    const themeAfter = await page.getByRole('button', { name: 'Switch to light mode' }).boundingBox();

    // toBeCloseTo's second argument is decimal places, so 0 is a 0.5px window —
    // loose enough to survive fractional layout, tight enough to catch the 2px
    // "Light" vs "Dark" shift the grid stacking exists to prevent.
    expect(themeAfter!.width).toBeCloseTo(themeBefore!.width, 0);
    expect(after!.x).toBeCloseTo(before!.x, 0);

    // 29.2px at a 16px root font is the height the three apps share; the comment
    // on .sidebar-footer-btn calls it the acceptance measurement, so pin it.
    expect(themeAfter!.height).toBeCloseTo(29.2, 0);
    expect(after!.height).toBeCloseTo(29.2, 0);

    // Side by side, inside the FOOTER's content box — not the column's: the
    // footer deliberately reaches further left than the column.
    const inner = await page.locator('.sidebar-footer').evaluate(el => {
      const r = el.getBoundingClientRect();
      const cs = getComputedStyle(el);
      return {
        left: r.left + el.clientLeft + parseFloat(cs.paddingLeft),
        right: r.left + el.clientLeft + el.clientWidth - parseFloat(cs.paddingRight),
      };
    });
    expect(after!.y).toBeCloseTo(themeAfter!.y, 0);
    expect(themeAfter!.x).toBeGreaterThanOrEqual(inner.left);
    expect(after!.x + after!.width).toBeLessThanOrEqual(inner.right);
  });

  // The row is nowrap and the buttons do not shrink, so a wider font overflows
  // rather than reflowing — and the only font anyone measures here is this
  // container's. This turns the CSS's slack claim into an assertion, since the
  // overflow branch is otherwise never exercised.
  //
  // 1.1x, not the 1.3x where it actually breaks: the point is to prove the slack
  // is real, not to pin how much of it there is. The row's width is text in
  // whatever system-ui resolves to, which the CSS comment right beside it warns
  // is not portable — so an assertion sitting near the boundary would be pinning
  // exactly the number that comment says not to trust.
  test('the row absorbs a 10% wider font without overflowing', async ({ page }) => {
    const measure = () =>
      page.locator('.sidebar-footer-actions').evaluate(el => {
        const gap = parseFloat(getComputedStyle(el).columnGap) || 0;
        const kids = [...el.children];
        return {
          needed: kids.reduce((s, k) => s + k.getBoundingClientRect().width, 0) + gap * (kids.length - 1),
          available: el.clientWidth,
        };
      });

    const before = await measure();
    expect(before.needed, `pair ${before.needed} in ${before.available}`).toBeLessThanOrEqual(before.available);

    await page.locator('.sidebar-footer-btn').evaluateAll(els => {
      for (const el of els) {
        el.style.fontSize = parseFloat(getComputedStyle(el).fontSize) * 1.1 + 'px';
      }
    });

    const after = await measure();
    // Reported either way, so a failure says how far over rather than just "red".
    expect(after.needed, `at 1.1x font: pair ${after.needed} in ${after.available}`)
      .toBeLessThanOrEqual(after.available);
  });

  // The footer buttons are sized in rem, so the column has to be too — a px
  // column keeps its width while they grow and pushes Settings out over the
  // calendar. That starts at a 20px root, Chrome's "Large" setting, so it is
  // reachable from the browser's own menu (WCAG 1.4.4).
  for (const root of [20, 24]) {
    test(`controls stay inside the column at a ${root}px root font`, async ({ page }) => {
      // Set through the CSSOM rather than addStyleTag — the app's CSP has no
      // 'unsafe-inline' in style-src, so an injected <style> is rejected.
      await page.evaluate(r => { document.documentElement.style.fontSize = `${r}px`; }, root);

      const { settings, column } = await boxes(page);
      expect(column.width).toBeGreaterThan(200);
      expect(settings.x + settings.width).toBeLessThanOrEqual(column.x + column.width);
      expect(await footerOverflows(page)).toBe(false);

      // .brand shares --sidebar-width, so it claims the same width up in the top
      // bar. Growing the column must not push the page into a horizontal scroll
      // — that would be the same 1.4.4 failure moved one row up.
      const hScroll = await page.evaluate(
        () => document.documentElement.scrollWidth > document.documentElement.clientWidth,
      );
      expect(hScroll).toBe(false);
    });
  }

  // Below 600px the sidebar's panels are hidden and the column stacks under the
  // main content, but the footer has to survive that — it is the only way to
  // reach Settings on a phone. Note the 8px *bottom* rule is deliberately out of
  // scope at this breakpoint (.app's padding drops to 4px); only the left edge
  // is asserted here, which is what the CSS documents.
  test('footer survives the narrow layout', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 700 });

    await expect(page.locator('.sidebar-footer')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Switch to dark mode' })).toBeVisible();
    await expect(page.locator(SETTINGS)).toBeVisible();
    await expect(page.locator('.mini-month')).toBeHidden();

    const { theme, settings, column } = await boxes(page);
    expect(theme.x).toBeCloseTo(8, 0);
    expect(settings.x + settings.width).toBeLessThanOrEqual(column.x + column.width);
    expect(await footerOverflows(page)).toBe(false);

    // Still usable, not just present.
    await page.locator(SETTINGS).click();
    await expect(page.locator('dialog.settings-dialog')).toBeVisible();
  });

  // ---------------------------------------------------------------------------
  // Colour
  // ---------------------------------------------------------------------------

  // The colour actually painted behind a control: walk up to the first ancestor
  // with a non-transparent background.
  //
  // Not the footer's own background — that is right in MyCal only because the
  // footer happens to declare one, and it would read rgba(0,0,0,0) in MyNotes,
  // whose footer is transparent and inherits its sidebar's paint. The contract is
  // about the resolved backdrop, not about which element supplies it, so this is
  // the method it names. Same number here either way; this is the version that is
  // not carrying a MyCal-shaped assumption.
  //
  // Starts at the PARENT: an element's own background is not its backdrop.
  // Starting at the button is correct today only because these carry
  // `background: none`, so the loop would walk straight past them — nothing in
  // the contract guarantees that. Used in a hover context it would return the
  // button's own fill as the "backdrop" and every figure derived from it would be
  // wrong while looking entirely plausible. That is not hypothetical: an earlier
  // version of this file did start at `el`, and the first measurement taken after
  // it was fixed came back different, because clicking the theme toggle leaves the
  // pointer on it.
  const backdropOf = (page: Page, selector: string) =>
    page.locator(selector).first().evaluate(el => {
      for (let n: Element | null = el.parentElement; n; n = n.parentElement) {
        const c = getComputedStyle(n).backgroundColor;
        if (c && c !== 'rgba(0, 0, 0, 0)' && c !== 'transparent') return c;
      }
      // Distinguishable from a colour, deliberately: a walk that finds nothing
      // must not fall through to a default, or it becomes a run that measured
      // nothing and looks like a pass.
      return null;
    });

  // Resolve a custom property to the same rgb() form getComputedStyle reports for
  // a background, so token and measurement can be compared without a hex/rgb
  // conversion in the test. Reading the property off :root gives "#f3f4f6", which
  // never equals "rgb(243, 244, 246)" and would make an equality assertion fail
  // for a reason that has nothing to do with the contract.
  const tokenColor = (page: Page, name: string) =>
    page.evaluate(n => {
      const probe = document.createElement('div');
      probe.style.backgroundColor = `var(${n})`;
      document.body.appendChild(probe);
      const v = getComputedStyle(probe).backgroundColor;
      probe.remove();
      return v;
    }, name);

  const useDarkTheme = async (page: Page) => {
    await page.getByRole('button', { name: 'Switch to dark mode' }).click();
    await expect(page.getByRole('button', { name: 'Switch to light mode' })).toBeVisible();
    // The click leaves the pointer sitting on the toggle, so any colour read from
    // a .sidebar-footer-btn afterwards is read in its HOVER state. Every resting
    // figure below would then be measured against the wrong thing while looking
    // entirely reasonable. Move the pointer off before measuring anything.
    await page.mouse.move(0, 0);
    // Then let the colour transition finish. Swapping the theme re-resolves the
    // tokens under a 0.12s transition, so the buttons spend that long reporting a
    // blend of the two palettes. See settledStyle.
    await settledStyle(page, SETTINGS, 'color');
  };

  // These buttons carry `transition: background 0.12s, color 0.12s,
  // border-color 0.12s` — mandated by the contract, so it cannot be removed to
  // make measuring easier. Every one of those three properties therefore reports
  // an intermediate value for 120ms after anything that changes it, and the
  // intermediate values are ordinary colours that look entirely plausible.
  //
  // This is not a theoretical hazard. Reading the label colour straight after the
  // theme toggle returned #4b5563 — the *light* theme's resting value — against a
  // dark backdrop, for 2.347:1 and a failing assertion that pointed at the
  // palette instead of at the clock. Note what did not catch it: the test already
  // asserted the element was not hovered, and it genuinely was not. The state was
  // right and the timing was wrong.
  //
  // So poll until two consecutive reads agree, rather than sleeping a guessed
  // interval, and throw if it never settles. A timeout that returned the last
  // value read would be a measurement of the transition reported as a
  // measurement of the colour — the same failure, quieter.
  // Two equal reads are necessary but not sufficient on their own: the transition
  // interpolates in 8-bit channels, so a slow segment can serialise to the same
  // rgb() twice in a row while still running, and the poll would return a colour
  // that is on its way somewhere. So also require that no transition is in flight
  // — getAnimations() reports CSS transitions, and an element with none pending
  // returns an empty list, which is the ordinary case here.
  const settledStyle = async (page: Page, selector: string, prop: 'color' | 'backgroundColor') => {
    const read = () =>
      page.locator(selector).first().evaluate(
        (el, p) => ({
          value: getComputedStyle(el)[p as 'color' | 'backgroundColor'],
          running: el.getAnimations().length,
        }),
        prop,
      );
    let prev = await read();
    for (let i = 0; i < 40; i++) {
      await page.waitForTimeout(25);
      const now = await read();
      if (now.value === prev.value && now.running === 0) return now.value;
      prev = now;
    }
    throw new Error(`${prop} of ${selector} never settled`);
  };

  // WCAG 1.4.11 (AA) wants 3:1 between the focus indicator and the colours next
  // to it. Asserting the indicator merely *exists* is not enough — the
  // translucent ring the spec first called for existed too, and measured 1.28:1.
  // So compute the real contrast, and check the outline is offset clear of the
  // button's own border: drawn tight against it the neighbour is --border, which
  // caps dark at 2.803:1.
  const relLum = (rgb: [number, number, number]) => {
    const f = (c: number) => {
      const s = c / 255;
      return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
    };
    return 0.2126 * f(rgb[0]) + 0.7152 * f(rgb[1]) + 0.0722 * f(rgb[2]);
  };
  const contrast = (a: [number, number, number], b: [number, number, number]) => {
    const [hi, lo] = [relLum(a), relLum(b)].sort((x, y) => y - x);
    return (hi + 0.05) / (lo + 0.05);
  };
  const parseRgb = (s: string): [number, number, number] => {
    const m = s.match(/\d+(\.\d+)?/g)!;
    return [Number(m[0]), Number(m[1]), Number(m[2])];
  };

  // Chromium withholds :focus-visible from a programmatic focus() when the last
  // interaction was a pointer, which would make every assertion below vacuous.
  // Establish keyboard modality, then assert the element really matched — so
  // this fails loudly rather than silently if that ever stops holding.
  const focusAndRead = async (page: Page, selector: string) => {
    await page.keyboard.press('Tab');
    const s = await page.locator(selector).first().evaluate(el => {
      (el as HTMLElement).focus();
      const cs = getComputedStyle(el);
      return {
        focusVisible: el.matches(':focus-visible'),
        outlineStyle: cs.outlineStyle,
        outlineWidth: cs.outlineWidth,
        outlineOffset: cs.outlineOffset,
        outlineColor: cs.outlineColor,
      };
    });
    return { ...s, backdrop: await backdropOf(page, selector) };
  };

  for (const dark of [false, true]) {
    test(`focus indicator meets 3:1 against its backdrop in ${dark ? 'dark' : 'light'} mode`, async ({ page }) => {
      if (dark) await useDarkTheme(page);

      const s = await focusAndRead(page, SETTINGS);
      expect(s.focusVisible).toBe(true);

      // Guard both parses: an rgba()/oklch()/color() value would fall out of
      // parseRgb as something bogus and could produce a meaningless pass.
      // null means the walk found nothing opaque, which is a broken measurement
      // rather than a failing one — distinguish it from a bad colour value.
      //
      // A throw rather than `expect(s.backdrop).not.toBeNull()`, which is what
      // this was. Two reasons, and the type error was the smaller one: an expect
      // does not narrow `string | null` for tsc, and it can be softened into
      // something that records a failure and lets the run continue — into
      // parseRgb(null), which is the meaningless pass this paragraph exists to
      // prevent. A broken measurement should stop the test where it broke.
      const { backdrop } = s;
      if (backdrop === null) {
        throw new Error('no opaque backdrop found above the control');
      }
      expect(backdrop).toMatch(/^rgb\(/);
      expect(s.outlineColor).toMatch(/^rgb\(/);
      expect(s.outlineStyle).toBe('solid');
      expect(parseFloat(s.outlineWidth)).toBeGreaterThanOrEqual(2);
      // Offset clear of the button's border — this is what lifts dark past 3:1.
      expect(parseFloat(s.outlineOffset)).toBeGreaterThanOrEqual(2);
      expect(contrast(parseRgb(s.outlineColor), parseRgb(backdrop))).toBeGreaterThanOrEqual(3);
    });
  }

  // The two defects below are the reason this section exists. Both shipped, both
  // were fixed by pinning the backdrop to --surface, and both came straight back
  // when the owner asked for the footer's box to be removed — with the whole
  // suite green each time, because nothing here measured colour except the focus
  // ring. A contrast failure and an invisible hover state are exactly the kind of
  // defect that renders without complaint.
  for (const dark of [false, true]) {
    const mode = dark ? 'dark' : 'light';

    // MyCal's backdrop is a recorded deviation, made on the owner's instruction:
    // the controls sit on the page background rather than on a --surface panel,
    // and the three apps are accepted to differ in this one colour. The shared
    // rule was rewritten to record a backdrop per app rather than name one for
    // all three, so this is the contract's value for MyCal, not a departure from
    // it — the cross-repo guard is green on it. Asserted
    // against the token rather than a literal so the palette stays the single
    // source of the value — the ratios below are what pin the value itself.
    test(`controls sit on the page background, not a panel, in ${mode} mode`, async ({ page }) => {
      if (dark) await useDarkTheme(page);
      const backdrop = await backdropOf(page, SETTINGS);
      expect(backdrop, 'no opaque backdrop found above the control').not.toBeNull();
      expect(backdrop).toBe(await tokenColor(page, '--bg'));
      // The state this repo was in before the owner's call, spelled out so that
      // restoring it fails here rather than only in the cross-repo guard.
      expect(backdrop).not.toBe(await tokenColor(page, '--surface'));
    });

    // WCAG 1.4.3 (AA). The label is 12.8px at weight 400 — normal text, so the
    // threshold is 4.5:1 and not the 3:1 large-text allowance. --text-subtle on
    // --bg measures 4.393:1, which is the failure that shipped.
    test(`resting label meets 4.5:1 against its backdrop in ${mode} mode`, async ({ page }) => {
      if (dark) await useDarkTheme(page);
      // Assert the state that was measured, not just the number: the resting
      // colour and the hover colour are different declarations, and reading one
      // while believing it is the other passes for the wrong reason. This is
      // necessary and not sufficient — it says nothing about the transition, so
      // the colour itself is read through settledStyle.
      const hovered = await page.locator(SETTINGS).evaluate(el => el.matches(':hover'));
      expect(hovered, 'measured in the hover state — this is not the resting colour').toBe(false);
      const color = await settledStyle(page, SETTINGS, 'color');
      const backdrop = await backdropOf(page, SETTINGS);
      expect(backdrop, 'no opaque backdrop found above the control').not.toBeNull();
      expect(color).toMatch(/^rgb\(/);
      expect(backdrop).toMatch(/^rgb\(/);
      expect(contrast(parseRgb(color), parseRgb(backdrop!))).toBeGreaterThanOrEqual(4.5);
    });

    // There is no WCAG threshold for a hover fill, and the mandated one is faint
    // by design (1.125:1 light, 1.721:1 dark) — so this pins the only thing that
    // is unambiguously broken: a fill the same colour as what it is drawn on.
    // That is not a hypothetical either. In MyCal's light palette --hover-bg and
    // --bg are both #f3f4f6, so the mandated fill painted over itself at exactly
    // 1.000:1 and the buttons lost their hover fill. (Not their whole hover
    // state — the rule moves the label and border colours as well, and those
    // still changed. This assertion covers the fill alone.)
    test(`hover fill is distinguishable from its backdrop in ${mode} mode`, async ({ page }) => {
      if (dark) await useDarkTheme(page);
      // Read the backdrop BEFORE hovering. The walk starts at the parent so it is
      // hover-safe by construction, but taking it first means the assertion does
      // not depend on that remaining true.
      const backdrop = await backdropOf(page, SETTINGS);
      expect(backdrop, 'no opaque backdrop found above the control').not.toBeNull();

      await page.locator(SETTINGS).hover();
      const fill = await settledStyle(page, SETTINGS, 'backgroundColor');
      // Guards the vacuous pass: with the hover rule gone the button keeps
      // `background: none` and reads rgba(0, 0, 0, 0), which parses to black and
      // would score a huge ratio against a light backdrop. rgb( excludes it.
      expect(fill, 'no opaque hover fill — did the hover rule apply?').toMatch(/^rgb\(/);
      expect(fill, 'hover fill is the same colour as its backdrop').not.toBe(backdrop);
      expect(contrast(parseRgb(fill), parseRgb(backdrop!))).toBeGreaterThan(1);
    });
  }

  // MyCal's two deviations from the shared colours are recorded upstream as
  // LIGHT-ONLY: dark still owes its values to the tokens the contract names, and
  // that is the whole shape of the divergence.
  //
  // Nothing above can check it. Point the dark aliases at light's values and both
  // thresholds still pass — dark --text-muted is #d1d5db (12.039:1, clears 4.5)
  // and dark --border is #374151, which is *the same colour as* dark --hover-bg,
  // so even the fill lands on the mandated value by coincidence. Measured, not
  // assumed: that mutation was applied and the six ratio assertions stayed green.
  //
  // So read the declarations rather than the results — the same reason the
  // typography pins are read out of the CSSOM. Asserting the token each theme is
  // *named against* is the only way to catch a substitution that resolves to an
  // identical colour today and stops doing so the day the palette moves.
  test('the dark aliases name the shared tokens, not their current values', async ({ page }) => {
    const decl = await page.evaluate(() => {
      for (const sheet of [...document.styleSheets]) {
        let rules: CSSRuleList;
        try { rules = sheet.cssRules; } catch { continue; } // cross-origin
        for (const rule of [...rules]) {
          // Exact match: the stylesheet has several `[data-theme="dark"] .foo`
          // rules, and the palette block is the bare selector.
          if (rule instanceof CSSStyleRule && rule.selectorText === '[data-theme="dark"]') {
            return {
              text: rule.style.getPropertyValue('--sidebar-footer-text').trim(),
              hover: rule.style.getPropertyValue('--sidebar-footer-hover-bg').trim(),
            };
          }
        }
      }
      return null;
    });

    expect(decl, '[data-theme="dark"] palette rule not found in any stylesheet').not.toBeNull();
    expect(decl!.text).toBe('var(--text-subtle)');
    expect(decl!.hover).toBe('var(--hover-bg)');
  });

  // §8.3: this footer is sticky and the document scrolls under it in the month
  // and year views, so it needs an opaque background of its own regardless of
  // what colour that is.
  //
  // This assertion cannot be folded into the backdrop test above, and the reason
  // is the whole point of it. Now that the footer paints --bg — the same colour
  // as the page behind it — deleting the declaration changes nothing visible at
  // the moment of the edit, and the backdrop walk keeps passing because it simply
  // falls through to <body> and finds the identical colour. The colour is
  // redundant; the opacity is load-bearing. The defect surfaces later, in a
  // different view, as content sliding through the buttons.
  test('the footer paints an opaque background of its own', async ({ page }) => {
    const own = await page.locator('.sidebar-footer').evaluate(el => getComputedStyle(el).backgroundColor);
    expect(own, '.sidebar-footer has no background of its own').toMatch(/^rgb\(/);
    expect(own).toBe(await tokenColor(page, '--bg'));
  });

  // An outline is painted under forced colours; a box-shadow is not, which is
  // why the ring the spec first called for would have needed a media-query
  // patch. The base rule carries it, so there is nothing theme-specific here.
  test('controls keep a focus indicator under forced colors', async ({ page }) => {
    await page.emulateMedia({ forcedColors: 'active' });
    const s = await focusAndRead(page, THEME);
    expect(s.focusVisible).toBe(true);
    expect(s.outlineStyle).not.toBe('none');
    expect(parseFloat(s.outlineWidth)).toBeGreaterThan(0);
  });
});
