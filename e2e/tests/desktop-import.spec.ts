import { test, expect } from '@playwright/test';
import { clearAllEvents } from './helpers';

const calendar = `BEGIN:VCALENDAR\r
VERSION:2.0\r
PRODID:-//MyCal Desktop Test//EN\r
BEGIN:VEVENT\r
UID:desktop-test\r
DTSTAMP:20260101T000000Z\r
DTSTART:20261025T100000Z\r
DTEND:20261025T110000Z\r
SUMMARY:Desktop file event\r
DESCRIPTION:</textarea><script>window.desktopInjection=true</script>\r
END:VEVENT\r
END:VCALENDAR\r
`;

test.beforeEach(async ({ request }) => {
  await clearAllEvents(request);
});

async function events(request: Parameters<typeof clearAllEvents>[0]) {
  const response = await request.get('/api/v1/events?from=2026-10-25T00:00:00Z&to=2026-10-26T00:00:00Z');
  expect(response.ok()).toBeTruthy();
  return response.json();
}

function fragment(content = calendar): string {
  return '#import=' + btoa(Array.from(new TextEncoder().encode(content), byte => String.fromCharCode(byte)).join(''))
    .replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '');
}

test('desktop handoff opens confirmation and imports only when confirmed', async ({ page, request, context }) => {
  const requested: string[] = [];
  page.on('request', req => requested.push(req.url()));
  await page.goto('/' + fragment());
  const mycal = page;
  expect(context.pages()).toHaveLength(1);

  const dialog = mycal.getByRole('dialog');
  await expect(dialog.getByRole('heading', { name: 'Import Event' })).toBeVisible();
  await expect(dialog.locator('textarea')).toHaveValue(calendar.replaceAll('\r\n', '\n'));
  expect(await mycal.evaluate(() => 'desktopInjection' in window)).toBeFalsy();
  expect(await events(request)).toHaveLength(0);
  expect(new URL(mycal.url()).hash).toBe('');
  expect(requested.every(url => !url.includes('import='))).toBeTruthy();
  await dialog.getByRole('button', { name: 'Import Event', exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await expect(mycal.locator('.toast-item:not(.toast-error)')).toBeVisible();
  const imported = await events(request);
  expect(imported).toHaveLength(1);
  expect(imported[0].title).toBe('Desktop file event');
  await mycal.reload();
  await expect(mycal.getByRole('heading', { name: 'Import Event' })).not.toBeVisible();
});

test('cancelling a desktop import leaves the calendar unchanged', async ({ page, request }) => {
  await page.goto('/' + fragment());
  const mycal = page;
  const dialog = mycal.getByRole('dialog');
  await expect(dialog.getByRole('heading', { name: 'Import Event' })).toBeVisible();
  await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
  await expect(dialog).not.toBeVisible();
  expect(await events(request)).toHaveLength(0);
});

test('multiple events are rejected by the single-event importer without writes', async ({ page, request }) => {
  const second = calendar.slice(calendar.indexOf('BEGIN:VEVENT')).replace('desktop-test', 'second-test');
  await page.goto('/' + fragment(calendar.replace('END:VCALENDAR', second)));
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  await dialog.getByRole('button', { name: 'Import Event', exact: true }).click();
  await expect(page.locator('.toast-error')).toBeVisible();
  await expect(dialog).toBeVisible();
  expect(await events(request)).toHaveLength(0);
});

test('malformed fragment is cleared and shows an error without importing', async ({ page, request }) => {
  await page.goto('/#import=bad!');
  await expect(page.locator('.toast-error')).toBeVisible();
  expect(new URL(page.url()).hash).toBe('');
  await expect(page.getByRole('dialog')).not.toBeVisible();
  expect(await events(request)).toHaveLength(0);
});

test('an existing tab can receive a new import fragment', async ({ page }) => {
  await page.goto('/');
  await page.evaluate(hash => { location.hash = hash; }, fragment());
  await expect(page.getByRole('dialog').getByRole('heading', { name: 'Import Event' })).toBeVisible();
  expect(new URL(page.url()).hash).toBe('');
});
