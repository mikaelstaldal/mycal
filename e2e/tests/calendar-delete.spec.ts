import { test, expect } from '@playwright/test';

test('delete calendar confirms, cancels, and protects the default', async ({ page, request }) => {
  const imported = await request.post('/api/v1/import?calendar=Delete%20test', {
    headers: { 'Content-Type': 'text/calendar' },
    data: 'BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:delete-test\r\nDTSTART:20261021T100000Z\r\nDTEND:20261021T110000Z\r\nSUMMARY:Delete this event\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n',
  });
  expect(imported.ok()).toBeTruthy();
  const calendars = await (await request.get('/api/v1/calendars')).json();
  const calendar = calendars.find((c: { name: string }) => c.name === 'Delete test');
  expect((await request.delete('/api/v1/calendars/0')).status()).toBe(400);
  const start = new Date();
  const end = new Date(start.getTime() + 60 * 60 * 1000);
  const kept = await request.post('/api/v1/events', { data: {
    title: 'Keep default event', all_day: false, start_time: start.toISOString(), end_time: end.toISOString(),
  } });
  expect(kept.ok()).toBeTruthy();
  await page.goto('/');
  await page.locator('.calendar-sidebar-item').filter({ hasText: 'Default' }).hover();
  await page.locator('.calendar-sidebar-item').filter({ hasText: 'Default' }).getByRole('button', { name: 'Edit calendar', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Delete calendar', exact: true })).toHaveCount(0);
  await page.getByRole('button', { name: 'Cancel', exact: true }).click();
  await page.locator('.calendar-sidebar-item').filter({ hasText: 'Default' }).click();
  await expect(page.getByText('Keep default event', { exact: true })).toBeHidden();
  const row = page.locator('.calendar-sidebar-item').filter({ hasText: 'Delete test' });
  await row.hover();
  await row.getByRole('button', { name: 'Edit calendar', exact: true }).click();
  page.once('dialog', async dialog => {
    expect(dialog.message()).toContain('Delete test');
    expect(dialog.message()).toContain('all of its events');
    await dialog.dismiss();
  });
  await page.getByRole('button', { name: 'Delete calendar', exact: true }).click();
  expect((await (await request.get('/api/v1/calendars')).json()).some((c: { id: number }) => c.id === calendar.id)).toBeTruthy();
  page.once('dialog', dialog => dialog.accept());
  const deleted = page.waitForResponse(response => response.request().method() === 'DELETE' && response.url().endsWith(`/calendars/${calendar.id}`));
  await page.getByRole('button', { name: 'Delete calendar', exact: true }).click();
  expect((await deleted).status()).toBe(204);
  await expect(page.locator('.calendar-sidebar')).toBeHidden();
  await expect(page.getByText('Keep default event', { exact: true })).toBeVisible();
  expect((await request.delete(`/api/v1/calendars/${calendar.id}`)).status()).toBe(404);
  const events = await (await request.get(`/api/v1/events?from=2026-10-01T00:00:00Z&to=2026-11-01T00:00:00Z&calendar_id=${calendar.id}`)).json();
  expect(events).toEqual([]);
});
