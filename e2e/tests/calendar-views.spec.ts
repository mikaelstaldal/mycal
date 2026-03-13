import { test, expect } from '@playwright/test';
import { clearAllEvents } from './helpers';

test.describe('Calendar Views', () => {
  test.beforeEach(async ({ page, request }) => {
    await clearAllEvents(request);
    await page.goto('/');
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

    await nav.getByRole('button', { name: '▶' }).click();
    await expect(heading).not.toHaveText(initialText!);

    await nav.getByRole('button', { name: '◀' }).click();
    await expect(heading).toHaveText(initialText!);
  });

  test('today button returns to current week', async ({ page }) => {
    const heading = page.locator('nav h1');
    const nav = page.getByRole('navigation');
    const initialText = await heading.textContent();

    // Navigate away
    await nav.getByRole('button', { name: '▶' }).click();
    await nav.getByRole('button', { name: '▶' }).click();
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
});
