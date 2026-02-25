import { test, expect } from '@playwright/test';

test.describe('Settings', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('open and close settings dialog', async ({ page }) => {
    // Click settings button (⚙ icon)
    await page.locator('.settings-btn[title="Settings"]').click();

    const dialog = page.locator('dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole('heading', { name: 'Settings' })).toBeVisible();

    // Close with the Close button
    await dialog.getByRole('button', { name: 'Close' }).click();
    await expect(dialog).not.toBeVisible();
  });

  test('change week start day persists on reload', async ({ page }) => {
    // Open settings
    await page.locator('.settings-btn[title="Settings"]').click();

    const dialog = page.locator('dialog');
    await expect(dialog).toBeVisible();

    // Change week start to Sunday
    await dialog.getByRole('combobox', { name: 'Week starts on' }).selectOption('Sunday');

    // Close settings
    await dialog.getByRole('button', { name: 'Close' }).click();

    // Check that the calendar header starts with Sun
    const firstWeekday = page.locator('.calendar-header .weekday').first();
    await expect(firstWeekday).toContainText('Sun');

    // Reload and verify persistence
    await page.reload();

    const firstWeekdayAfterReload = page.locator('.calendar-header .weekday').first();
    await expect(firstWeekdayAfterReload).toContainText('Sun');
  });
});
