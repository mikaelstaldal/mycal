import { test, expect, Page } from '@playwright/test';
import { clearAllEvents, createEventViaAPI, todayDate } from './helpers';

// Letter shortcuts activate the visible, enabled button carrying the matching
// data-shortcut, scoped to the open modal dialog when there is one.

async function pressOnBody(page: Page, key: string) {
  await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
  await page.keyboard.press(key);
}

test.describe('Keyboard shortcuts', () => {
  test.beforeEach(async ({ page, request }) => {
    await clearAllEvents(request);
    await page.goto('/');
    await page.getByRole('button', { name: 'Month', exact: true }).click();
    await expect(page.locator('.calendar-grid')).toBeVisible();
  });

  test('view letters switch view and are underlined', async ({ page }) => {
    for (const key of ['t', 'y', 'm', 'w', 'd', 's']) {
      await expect(page.locator(`nav button[data-shortcut="${key}"] u`)).toHaveText(new RegExp(key, 'i'));
    }
    await pressOnBody(page, 'y');
    await expect(page.getByRole('button', { name: 'Year', exact: true })).toHaveClass(/active/);
    await pressOnBody(page, 'w');
    await expect(page.getByRole('button', { name: 'Week', exact: true })).toHaveClass(/active/);
    await pressOnBody(page, 'd');
    await expect(page.getByRole('button', { name: 'Day', exact: true })).toHaveClass(/active/);
    await pressOnBody(page, 's');
    await expect(page.getByRole('button', { name: 'Schedule', exact: true })).toHaveClass(/active/);
    await pressOnBody(page, 'm');
    await expect(page.getByRole('button', { name: 'Month', exact: true })).toHaveClass(/active/);

    // Typing in a field, or with a modifier, does not trigger a shortcut.
    const search = page.locator('.search-input');
    await search.focus();
    await page.keyboard.type('wd');
    await expect(search).toHaveValue('wd');
    await search.fill('');
    await page.keyboard.press('Shift+W');
    await search.blur();
    await page.keyboard.press('Shift+W');
    await expect(page.getByRole('button', { name: 'Month', exact: true })).toHaveClass(/active/);
  });

  test('N opens a new event on today', async ({ page }) => {
    await expect(page.locator('.new-event-btn u')).toHaveText('N');
    await pressOnBody(page, 'n');
    const dialog = page.locator('dialog.event-dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog.locator('h2')).toHaveText('New Event');
    await expect(dialog.getByRole('textbox', { name: 'Title' })).toBeFocused();
    await page.keyboard.type('Typed');
    await expect(dialog.getByRole('textbox', { name: 'Title' })).toHaveValue('Typed');
    await dialog.getByRole('button', { name: 'Save' }).click();
    await expect(page.locator('.day.today .event-chip', { hasText: 'Typed' })).toBeVisible();
  });

  test('event dialog letters act on the event, D deletes with confirmation', async ({ page, request }) => {
    const today = todayDate();
    await createEventViaAPI(request, {
      title: 'Shortcut Event',
      all_day: false,
      start_time: `${today}T10:00:00Z`,
      end_time: `${today}T11:00:00Z`,
    });
    await page.reload();
    await page.getByRole('button', { name: 'Month', exact: true }).click();
    await page.locator('.event-chip', { hasText: 'Shortcut Event' }).click();
    const dialog = page.locator('dialog.event-dialog');
    await expect(dialog).toBeVisible();

    // E opens the editor; W, which means Week outside the dialog, does nothing here.
    await page.keyboard.press('w');
    await expect(page.getByRole('button', { name: 'Month', exact: true })).toHaveClass(/active/);
    await page.keyboard.press('e');
    await expect(dialog.getByRole('button', { name: 'Save' })).toBeVisible();
    await dialog.getByRole('button', { name: 'Cancel' }).click();
    await expect(dialog).not.toBeVisible();

    await page.locator('.event-chip', { hasText: 'Shortcut Event' }).click();
    await expect(dialog).toBeVisible();
    await page.keyboard.press('d');
    const confirmDialog = page.locator('dialog.confirm-dialog');
    await expect(confirmDialog).toBeVisible();
    // The confirmation owns the keyboard: D again does not confirm.
    await page.keyboard.press('d');
    await expect(confirmDialog).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(confirmDialog).not.toBeVisible();
    await expect(dialog).toBeVisible();

    await pressOnBody(page, 'd');
    await expect(confirmDialog).toBeVisible();
    await confirmDialog.getByRole('button', { name: 'Delete' }).click();
    await expect(page.locator('.event-chip', { hasText: 'Shortcut Event' })).not.toBeVisible();
  });
});
