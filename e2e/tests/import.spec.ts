import { test, expect } from '@playwright/test';
import { clearAllEvents } from './helpers';
import path from 'path';

test.beforeEach(async ({ page, request }) => {
  await clearAllEvents(request);
  await page.goto('/');
});

test.describe('Import', () => {
  test('import ICS file successfully', async ({ page }) => {
    // Click the import button (⬇ icon)
    await page.locator('.settings-btn[title="Import"]').click();

    // Import dialog should open
    const dialog = page.locator('dialog.import-dialog');
    await expect(dialog).toBeVisible();

    // Upload the fixture file
    const fileInput = dialog.locator('input[type="file"]');
    await fileInput.setInputFiles(path.resolve(__dirname, 'fixtures/single-event.ics'));

    // Click Import Event button
    await dialog.getByRole('button', { name: 'Import Event' }).click();

    // Should show success message
    await expect(dialog.locator('.import-success')).toBeVisible();

    // Close dialog
    await dialog.getByRole('button', { name: 'Close' }).click();

    // Navigate to March 2026 to see the imported event
    const heading = page.locator('nav h1');
    while (!(await heading.textContent())?.includes('March 2026')) {
      await page.getByRole('button', { name: '▶' }).click();
      await page.waitForTimeout(100);
    }

    // Event should appear on calendar
    await expect(page.locator('.event-chip', { hasText: 'Imported Test Event' })).toBeVisible();
  });

  test('import invalid file shows error', async ({ page }) => {
    await page.locator('.settings-btn[title="Import"]').click();

    const dialog = page.locator('dialog.import-dialog');
    await expect(dialog).toBeVisible();

    // Upload invalid ICS content
    const fileInput = dialog.locator('input[type="file"]');
    await fileInput.setInputFiles({
      name: 'invalid.ics',
      mimeType: 'text/calendar',
      buffer: Buffer.from('This is not valid ICS content'),
    });

    await dialog.getByRole('button', { name: 'Import Event' }).click();

    // Should show error (either in dialog or as toast)
    const hasError = await Promise.race([
      dialog.locator('.error').waitFor({ timeout: 3000 }).then(() => true).catch(() => false),
      page.locator('.toast-error').waitFor({ timeout: 3000 }).then(() => true).catch(() => false),
    ]);
    expect(hasError).toBeTruthy();
  });
});
