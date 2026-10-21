import { test, expect } from '@playwright/test';

const url = `https://example.com/${'long-calendar-path/'.repeat(20)}calendar.ics?token=secret%2Bvalue`;

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/feeds', route => route.fulfill({ json: [{
    id: 1, url, enabled: true, refresh_interval_minutes: 60,
  }] }));
  await page.goto('/');
  await page.getByRole('button', { name: 'Feed Subscriptions' }).click();
});

test('copies the full URL even when its display is truncated', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  const label = page.locator('.feed-item-url');
  expect(await label.evaluate(el => el.scrollWidth > el.clientWidth)).toBe(true);
  await page.getByRole('button', { name: 'Copy URL', exact: true }).click();
  await expect(page.getByText('Feed URL copied', { exact: true })).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(url);
});

test('reports clipboard failure without claiming success', async ({ page }) => {
  await page.evaluate(() => {
    Object.defineProperty(navigator, 'clipboard', { value: undefined, configurable: true });
  });
  await page.getByRole('button', { name: 'Copy URL', exact: true }).click();
  await expect(page.locator('.feed-error')).toContainText('Could not copy feed URL');
  await expect(page.getByText('Feed URL copied', { exact: true })).toHaveCount(0);
});

test('clears a copy error when a retry succeeds', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.evaluate(() => {
    const writeText = navigator.clipboard.writeText.bind(navigator.clipboard);
    let first = true;
    navigator.clipboard.writeText = text => {
      if (first) {
        first = false;
        return Promise.reject(new Error('Permission denied'));
      }
      return writeText(text);
    };
  });
  const copy = page.getByRole('button', { name: 'Copy URL', exact: true });
  await copy.click();
  await expect(page.locator('.feed-error')).toBeVisible();
  await copy.click();
  await expect(page.locator('.feed-error')).toHaveCount(0);
  await expect(page.locator('dialog.feeds-dialog').getByRole('status')).toHaveText('Feed URL copied');
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(url);
});
