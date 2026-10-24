import { test, expect } from '@playwright/test';
import { readFile } from 'node:fs/promises';
import { clearAllEvents, createEventViaAPI, todayDate } from './helpers';

test.beforeEach(async ({ request }) => {
  await clearAllEvents(request);
});

for (const allDay of [false, true]) {
  test(`download a single ${allDay ? 'all-day' : 'timed'} event`, async ({ page, request }) => {
    const today = todayDate();
    const tomorrow = new Date(`${today}T00:00:00Z`);
    tomorrow.setUTCDate(tomorrow.getUTCDate() + 1);
    const response = await request.post('/api/v1/events', { data: {
      title: 'Möte / 会議',
      all_day: allDay,
      ...(allDay
        ? { start_date: today, end_date: tomorrow.toISOString().slice(0, 10) }
        : { start_time: `${today}T10:00:00Z`, end_time: `${today}T11:00:00Z` }),
      description: 'Bring lunch',
      location: 'Park',
    } });
    expect(response.ok()).toBeTruthy();
    const event = await response.json();
    await createEventViaAPI(request, {
      title: 'Other event', all_day: false,
      start_time: `${today}T12:00:00Z`, end_time: `${today}T13:00:00Z`,
    });
    const expected = await request.get(`/api/v1/events/${event.id}/ics`);
    expect(expected.ok()).toBeTruthy();
    await page.goto('/');
    await page.getByRole('button', { name: 'Month', exact: true }).click();
    await page.locator('.event-chip', { hasText: 'Möte / 会議' }).click();
    const dialog = page.locator('dialog.event-dialog');
    const downloading = page.waitForEvent('download');
    await dialog.getByRole('button', { name: 'Export', exact: true }).click();
    const download = await downloading;
    expect(download.suggestedFilename()).toBe('Möte  会議.ics');
    const content = await readFile((await download.path())!, 'utf8');
    expect(content).toBe(await expected.text());
    expect(content.match(/BEGIN:VEVENT/g)).toHaveLength(1);
    expect(content).not.toContain('Other event');
    await expect(dialog).toBeVisible();
    await dialog.getByRole('button', { name: 'Edit', exact: true }).click();
    await expect(dialog.getByRole('button', { name: 'Export', exact: true })).toHaveCount(0);
  });
}

test('export failure appears in the event dialog', async ({ page, request }) => {
  const today = todayDate();
  await createEventViaAPI(request, {
    title: 'Failed export', all_day: false,
    start_time: `${today}T10:00:00Z`, end_time: `${today}T11:00:00Z`,
  });
  await page.goto('/');
  await page.getByRole('button', { name: 'Month', exact: true }).click();
  await page.locator('.event-chip', { hasText: 'Failed export' }).click();
  await page.route('**/api/v1/events/*/ics', route => route.fulfill({ status: 400 }));
  const dialog = page.locator('dialog.event-dialog');
  await dialog.getByRole('button', { name: 'Export', exact: true }).click();
  await expect(dialog.locator('.error')).toHaveText('Failed to get event data (400)');
  await expect(dialog).toBeVisible();
});
