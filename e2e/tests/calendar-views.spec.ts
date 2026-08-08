import { test, expect } from '@playwright/test';
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
  // (../../mysuite/spec/app-logo.md §4). It used to be a *remainder*: .brand was
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
