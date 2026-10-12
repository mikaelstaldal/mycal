// Verifies explicit consent, prefilled codes and the resulting ordinary token.
import { test, expect } from '@playwright/test';

test('approve and deny a device login', async ({ page, request }) => {
  const authorization = await request.post('/api/v1/oauth/device_authorization', { form: { client_id: 'mycal-token', name: 'E2E device', scope: 'calendar:0', lifetime: '30d' } });
  expect(authorization.status()).toBe(200);
  const grant = await authorization.json();
  const target = '/?settings=api-tokens&user_code=' + encodeURIComponent(grant.user_code);
  await page.goto(target);
  await expect(page.getByRole('heading', { name: 'Approve device' })).toBeVisible();
  await expect(page.getByText('E2E device', { exact: true })).toBeVisible();
  await expect(page.getByText('Requester (IP | User-Agent)', { exact: true })).toBeVisible();
  await expect(page.getByText(grant.user_code, { exact: true })).toBeVisible();
  const approve = page.getByRole('button', { name: 'Approve', exact: true });
  const selected = page.locator('.device-approval input[type="checkbox"]').first();
  await selected.uncheck();
  await expect(approve).toBeDisabled();
  await selected.check();
  await page.getByLabel('Granted lifetime').fill('7d');
  await approve.click();
  await expect(page.getByRole('status')).toContainText('Approved.');
  const issued = await request.post('/api/v1/oauth/token', { form: { client_id: 'mycal-token', device_code: grant.device_code, grant_type: 'urn:ietf:params:oauth:grant-type:device_code' } });
  expect(issued.status()).toBe(200);
  const token = await issued.json();
  expect(token.access_token).toMatch(/mycal_[A-Za-z0-9_-]{43}/);
  expect(token.scope).toBe('calendar:0');
  expect(token.expires_in).toBeLessThanOrEqual(7 * 86400);
  const another = await request.post('/api/v1/oauth/device_authorization', { form: { client_id: 'mycal-token', name: 'Denied device' } });
  const denied = await another.json();
  await page.getByLabel('User code').fill('  ' + denied.user_code.toLowerCase().replace('-', '') + '  ');
  await page.getByRole('button', { name: 'Review request' }).click();
  await expect(page.getByText('Denied device', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Deny', exact: true }).click();
  await expect(page.getByRole('status')).toContainText('Denied.');
  const result = await request.post('/api/v1/oauth/token', { form: { client_id: 'mycal-token', device_code: denied.device_code, grant_type: 'urn:ietf:params:oauth:grant-type:device_code' } });
  expect(result.status()).toBe(400);
  expect((await result.json()).error).toBe('access_denied');
});

// A plain verification link is a one-time entry point; a retried load can recover.
test('plain approval link clears URL and calendar loading error', async ({ page }) => {
  await page.route('**/api/v1/calendars', route => route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"load failed"}' }));
  await page.goto('/?settings=api-tokens');
  await expect(page.getByRole('dialog')).toBeVisible();
  await expect(page.getByText('Could not load calendars.', { exact: true })).toBeVisible();
  await expect(page).not.toHaveURL(/settings=api-tokens/);
  await page.getByRole('dialog').getByRole('button', { name: 'Close', exact: true }).last().click();
  await page.unroute('**/api/v1/calendars');
  await page.reload();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await page.getByRole('button', { name: 'Settings', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Approve device' })).toBeVisible();
  await expect(page.getByText('Could not load calendars.', { exact: true })).toHaveCount(0);
});
