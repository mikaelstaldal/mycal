import { test, expect } from '@playwright/test';

const feedURI = 'webcal://1.1.1.1/calendar.ics?token=a%2Fb';

test.beforeEach(async ({ request }) => {
  const response = await request.get('/api/v1/feeds');
  expect(response.ok()).toBeTruthy();
  for (const feed of await response.json() as { id: number }[]) {
    const deleted = await request.delete(`/api/v1/feeds/${feed.id}`);
    expect(deleted.ok()).toBeTruthy();
  }
});

test('webcal handoff requires confirmation, adds a feed, and avoids duplicates', async ({ page, request }) => {
  await page.goto('/#subscribe=' + encodeURIComponent(feedURI));

  const prompt = page.locator('dialog.confirm-dialog');
  await expect(prompt).toBeVisible();
  await expect(prompt).toContainText(feedURI);
  await expect(prompt).toContainText('from 1.1.1.1');
  await expect(page).toHaveURL(/\/$/); // URI removed from browser history before the API call
  expect(await (await request.get('/api/v1/feeds')).json()).toEqual([]);

  await prompt.getByRole('button', { name: 'Subscribe' }).click();
  const feedsDialog = page.locator('dialog.feeds-dialog');
  await expect(feedsDialog).toBeVisible();
  await expect(feedsDialog.locator('.feed-item-url')).toHaveText(feedURI);
  expect((await (await request.get('/api/v1/feeds')).json()).length).toBe(1);

  await page.goto('/#subscribe=' + encodeURIComponent(feedURI.replace('webcal:', 'webcals:')));
  await page.locator('dialog.confirm-dialog').getByRole('button', { name: 'Subscribe' }).click();
  await expect(page.locator('.toast-item', { hasText: 'Already subscribed to this feed' })).toBeVisible();
  expect((await (await request.get('/api/v1/feeds')).json()).length).toBe(1);

  // The dialog is already open; a second handoff must reload its list.
  const secondURI = 'webcal://1.1.1.1/second.ics';
  await page.goto('/#subscribe=' + encodeURIComponent(secondURI));
  await page.locator('dialog.confirm-dialog').getByRole('button', { name: 'Subscribe' }).click();
  await expect(feedsDialog.locator('.feed-item-url')).toHaveCount(2);
  await expect(feedsDialog.locator('.feed-item-url', { hasText: secondURI })).toBeVisible();
});

test('cancelling or opening an invalid handoff does not add a feed', async ({ page, request }) => {
  await page.goto('/#subscribe=' + encodeURIComponent(feedURI));
  await page.locator('dialog.confirm-dialog').getByRole('button', { name: 'Cancel' }).click();
  expect(await (await request.get('/api/v1/feeds')).json()).toEqual([]);

  await page.goto('/#subscribe=' + encodeURIComponent('https://1.1.1.1/calendar.ics'));
  await expect(page.locator('.toast-error')).toContainText('Invalid calendar subscription link');
  await expect(page.locator('dialog.confirm-dialog')).not.toBeVisible();

  await page.goto('/#subscribe=' + encodeURIComponent('webcal://calendar.example.com\n.evil.test/x'));
  await expect(page.locator('.toast-error', { hasText: 'Invalid calendar subscription link' }).last()).toBeVisible();
  await expect(page.locator('dialog.confirm-dialog')).not.toBeVisible();
  expect(await (await request.get('/api/v1/feeds')).json()).toEqual([]);
});

test('registers an HTTPS handler URL on the MyCal origin', async ({ page, request }) => {
  await page.route('https://mycal.test/**', async route => {
    const url = new URL(route.request().url());
    const response = await request.get(`http://localhost:8089${url.pathname}${url.search}`);
    await route.fulfill({ response });
  });
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'registerProtocolHandler', {
      configurable: true,
      value: (scheme: string, url: string) => {
        (window as typeof window & { handlerRegistration?: { scheme: string; url: string } })
          .handlerRegistration = { scheme, url };
      }
    });
  });
  await page.goto('https://mycal.test/');
  await page.getByRole('button', { name: 'Feed Subscriptions' }).click();
  await page.getByRole('button', { name: 'Use MyCal for webcal links' }).click();
  const registration = await page.evaluate(() =>
    (window as typeof window & { handlerRegistration?: { scheme: string; url: string } }).handlerRegistration
  );
  expect(registration).toEqual({ scheme: 'webcal', url: 'https://mycal.test/#subscribe=%s' });
});

test('explains that browser registration requires HTTPS', async ({ page }) => {
  await page.goto('/');
  await page.getByRole('button', { name: 'Feed Subscriptions' }).click();
  const dialog = page.locator('dialog.feeds-dialog');
  await expect(dialog.getByRole('button', { name: 'Use MyCal for webcal links' })).toBeDisabled();
  await expect(dialog).toContainText('Available when MyCal is opened over HTTPS.');
});
