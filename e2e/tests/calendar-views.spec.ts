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
      // The sidebar's footer buttons line up with that same edge.
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
    const { scrollHeight, clientHeight } = await page.evaluate(() => {
      const sidebar = document.querySelector('.left-sidebar')!;
      return { scrollHeight: sidebar.scrollHeight, clientHeight: sidebar.clientHeight };
    });

    expect(await miniHeight()).toBeCloseTo(relaxed, 0);
    expect(scrollHeight).toBeGreaterThan(clientHeight);
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
