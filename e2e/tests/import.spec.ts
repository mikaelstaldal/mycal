import { test, expect } from '@playwright/test';
import { clearAllEvents } from './helpers';
import path from 'path';

test.describe('Import', () => {
  test.beforeEach(async ({ page, request }) => {
    await clearAllEvents(request);
    await page.goto('/');
    // Switch to month view for these tests
    await page.getByRole('button', { name: 'Month' }).click();
    await expect(page.locator('.calendar-grid')).toBeVisible();
  });

  test('import ICS file successfully', async ({ page }) => {
    // Click the import event button (⬇ icon)
    await page.locator('.settings-btn[title="Import Event"]').click();

    // Import dialog should open
    const dialog = page.locator('dialog.import-dialog');
    await expect(dialog).toBeVisible();

    // Upload the fixture file
    const fileInput = dialog.locator('input[type="file"]');
    await fileInput.setInputFiles(path.resolve(__dirname, 'fixtures/single-event.ics'));

    // Click Import Event button
    await dialog.getByRole('button', { name: 'Import Event' }).click();

    // Dialog should close and a success toast should appear
    await expect(dialog).not.toBeVisible();
    await expect(page.locator('.toast:not(.toast-error)')).toBeVisible();

    // Navigate to March 2026 to see the imported event
    const heading = page.locator('nav h1');
    const monthNames = ['January','February','March','April','May','June','July','August','September','October','November','December'];
    while (!(await heading.textContent())?.includes('March 2026')) {
      const text = await heading.textContent() || '';
      const parts = text.trim().split(' ');
      const monthIdx = monthNames.indexOf(parts[0]);
      const yearNum = parseInt(parts[1]);
      const isAfterTarget = yearNum > 2026 || (yearNum === 2026 && monthIdx > 2);
      if (isAfterTarget) {
        await page.locator('nav.nav').getByRole('button', { name: '◀' }).click();
      } else {
        await page.locator('nav.nav').getByRole('button', { name: '▶' }).click();
      }
      await page.waitForTimeout(100);
    }

    // Event should appear on calendar
    await expect(page.locator('.event-chip', { hasText: 'Imported Test Event' })).toBeVisible();
  });

  test('import invalid file shows error', async ({ page }) => {
    await page.locator('.settings-btn[title="Import Event"]').click();

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
