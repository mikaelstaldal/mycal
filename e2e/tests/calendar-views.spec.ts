import { test, expect } from '@playwright/test';
import { clearAllEvents } from './helpers';

test.describe('Calendar Views', () => {
  test.beforeEach(async ({ page, request }) => {
    await clearAllEvents(request);
    await page.goto('/');
  });

  test('shows month view by default with correct heading', async ({ page }) => {
    const heading = page.locator('nav h1');
    await expect(heading).toBeVisible();

    const now = new Date();
    const expectedMonth = now.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
    await expect(heading).toContainText(expectedMonth);

    await expect(page.locator('.calendar-grid')).toBeVisible();
  });

  test('navigate months with prev/next buttons', async ({ page }) => {
    const heading = page.locator('nav h1');
    const initialText = await heading.textContent();

    await page.getByRole('button', { name: '▶' }).click();
    await expect(heading).not.toHaveText(initialText!);

    await page.getByRole('button', { name: '◀' }).click();
    await expect(heading).toHaveText(initialText!);
  });

  test('today button returns to current month', async ({ page }) => {
    const heading = page.locator('nav h1');
    const now = new Date();
    const expectedMonth = now.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });

    // Navigate away
    await page.getByRole('button', { name: '▶' }).click();
    await page.getByRole('button', { name: '▶' }).click();

    // Click Today
    await page.getByRole('button', { name: 'Today' }).click();
    await expect(heading).toContainText(expectedMonth);
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
