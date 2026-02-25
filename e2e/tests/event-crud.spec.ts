import { test, expect } from '@playwright/test';
import { clearAllEvents, createEventViaAPI, todayDate } from './helpers';

test.beforeEach(async ({ page, request }) => {
  await clearAllEvents(request);
  await page.goto('/');
});

test.describe('Event CRUD', () => {
  test('create event by clicking day cell and filling form', async ({ page }) => {
    // Click on today's day number in the calendar
    const todayCell = page.locator('.day.today .day-number');
    await todayCell.click();

    // Dialog should open
    const dialog = page.locator('dialog.event-dialog');
    await expect(dialog).toBeVisible();

    // Fill in the title
    await dialog.getByRole('textbox', { name: 'Title' }).fill('Test Event');

    // Save
    await dialog.getByRole('button', { name: 'Save' }).click();

    // Event should appear on calendar
    await expect(page.locator('.event-chip', { hasText: 'Test Event' })).toBeVisible();
  });

  test('click existing event opens dialog with details', async ({ page, request }) => {
    const today = todayDate();
    await createEventViaAPI(request, {
      title: 'Existing Event',
      start_time: `${today}T10:00:00Z`,
      end_time: `${today}T11:00:00Z`,
    });

    await page.reload();

    // Click the event chip
    await page.locator('.event-chip', { hasText: 'Existing Event' }).click();

    // Dialog opens with event details
    const dialog = page.locator('dialog.event-dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole('textbox', { name: 'Title' })).toHaveValue('Existing Event');
  });

  test('edit event title', async ({ page, request }) => {
    const today = todayDate();
    await createEventViaAPI(request, {
      title: 'Original Title',
      start_time: `${today}T10:00:00Z`,
      end_time: `${today}T11:00:00Z`,
    });

    await page.reload();

    // Click the event
    await page.locator('.event-chip', { hasText: 'Original Title' }).click();

    const dialog = page.locator('dialog.event-dialog');
    await expect(dialog).toBeVisible();

    // Click Edit button to enter edit mode
    await dialog.getByRole('button', { name: 'Edit' }).click();

    // Clear and type new title
    await dialog.getByRole('textbox', { name: 'Title' }).fill('Updated Title');

    // Save
    await dialog.getByRole('button', { name: 'Save' }).click();

    // Updated title should appear
    await expect(page.locator('.event-chip', { hasText: 'Updated Title' })).toBeVisible();
    await expect(page.locator('.event-chip', { hasText: 'Original Title' })).not.toBeVisible();
  });

  test('delete event', async ({ page, request }) => {
    const today = todayDate();
    await createEventViaAPI(request, {
      title: 'Delete Me',
      start_time: `${today}T10:00:00Z`,
      end_time: `${today}T11:00:00Z`,
    });

    await page.reload();

    // Click the event
    await page.locator('.event-chip', { hasText: 'Delete Me' }).click();

    const dialog = page.locator('dialog.event-dialog');
    await expect(dialog).toBeVisible();

    // Handle the browser confirm dialog before clicking Delete
    page.on('dialog', (d) => d.accept());

    // Click Delete button (available directly in read-only view)
    await dialog.getByRole('button', { name: 'Delete' }).click();

    // Event should be removed
    await expect(page.locator('.event-chip', { hasText: 'Delete Me' })).not.toBeVisible();
  });

  test('create all-day event', async ({ page }) => {
    const todayCell = page.locator('.day.today .day-number');
    await todayCell.click();

    const dialog = page.locator('dialog.event-dialog');
    await expect(dialog).toBeVisible();

    await dialog.getByRole('textbox', { name: 'Title' }).fill('All Day Meeting');

    // Check all-day checkbox
    await dialog.getByRole('checkbox', { name: 'All day' }).check();

    await dialog.getByRole('button', { name: 'Save' }).click();

    await expect(page.locator('.event-chip', { hasText: 'All Day Meeting' })).toBeVisible();
  });

  test('validation: save with empty title shows error', async ({ page }) => {
    const todayCell = page.locator('.day.today .day-number');
    await todayCell.click();

    const dialog = page.locator('dialog.event-dialog');
    await expect(dialog).toBeVisible();

    // Try to save without filling title
    await dialog.getByRole('button', { name: 'Save' }).click();

    // Error message should appear
    await expect(dialog.locator('.error')).toBeVisible();
  });
});
