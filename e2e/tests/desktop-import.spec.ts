import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';
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

async function openDesktop(page: Page, baseURL: string) {
  // A different loopback origin models the native helper's short-lived page.
  await page.route('http://127.0.0.1:54321/**', route => route.fulfill({
    contentType: 'text/html',
    body: `<button>Continue to MyCal</button><script>
      const nonce = 'abcdefghijklmnopqrstuvwx';
      let receiver;
      document.querySelector('button').onclick = () => {
        receiver = window.open(${JSON.stringify(baseURL + '/#desktop-import=http%3A%2F%2F127.0.0.1%3A54321%2Fabcdefghijklmnopqrstuvwx')}, '_blank');
      };
      addEventListener('message', event => {
        if (event.origin === ${JSON.stringify(new URL(baseURL).origin)} && event.source === receiver && event.data?.type === 'mycal-desktop-ready') {
          receiver.postMessage({type: 'mycal-desktop-file', nonce, content: ${JSON.stringify(calendar).replaceAll('<', '\\u003c')}}, event.origin);
        }
      });
    </script>`,
  }));
  await page.goto('http://127.0.0.1:54321/secret');
  const opened = page.waitForEvent('popup');
  await page.getByRole('button', { name: 'Continue to MyCal' }).click();
  return opened;
}

test('desktop handoff opens confirmation and imports only when confirmed', async ({ page, request, baseURL, context }) => {
  // A normal top-level GET includes existing Lax login cookies across sites.
  await context.addCookies([{ name: 'desktop-login', value: 'present', url: baseURL!, sameSite: 'Lax' }]);
  const navigation = context.waitForEvent('request', req => req.isNavigationRequest() && req.url().startsWith(baseURL!));
  const mycal = await openDesktop(page, baseURL!);
  expect((await (await navigation).allHeaders()).cookie).toContain('desktop-login=present');

  const dialog = mycal.getByRole('dialog');
  await expect(dialog.getByRole('heading', { name: 'Open iCalendar File' })).toBeVisible();
  await expect(dialog.locator('textarea')).toHaveValue(calendar.replaceAll('\r\n', '\n'));
  expect(await mycal.evaluate(() => 'desktopInjection' in window)).toBeFalsy();
  expect(await events(request)).toHaveLength(0);
  expect(new URL(mycal.url()).hash).toBe('');
  await dialog.getByRole('button', { name: 'Import All', exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await expect(mycal.locator('.toast-item:not(.toast-error)')).toBeVisible();
  const imported = await events(request);
  expect(imported).toHaveLength(1);
  expect(imported[0].title).toBe('Desktop file event');
  await mycal.reload();
  await expect(mycal.getByRole('heading', { name: 'Open iCalendar File' })).not.toBeVisible();
});

test('cancelling a desktop import leaves the calendar unchanged', async ({ page, request, baseURL }) => {
  const mycal = await openDesktop(page, baseURL!);
  const dialog = mycal.getByRole('dialog');
  await expect(dialog.getByRole('heading', { name: 'Open iCalendar File' })).toBeVisible();
  await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
  await expect(dialog).not.toBeVisible();
  expect(await events(request)).toHaveLength(0);
});
