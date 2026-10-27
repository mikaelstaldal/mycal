import { test, expect } from '@playwright/test';
import { clearAllEvents, createEventViaAPI, todayDate } from './helpers';

for (const view of ['Day', 'Week']) {
  for (const fails of [false, true]) {
    test(`${view} drag stays at the drop position while saving${fails ? ' and restores on failure' : ''}`, async ({ page, request }) => {
      await clearAllEvents(request);
      const today = todayDate();
      await createEventViaAPI(request, {
        title: 'Drag me', all_day: false,
        start_time: new Date(`${today}T10:00:00`).toISOString(),
        end_time: new Date(`${today}T11:00:00`).toISOString(),
      });
      await page.goto('/');
      await page.getByRole('button', { name: view, exact: true }).click();
      const source = page.locator('.week-event', { hasText: 'Drag me' });
      await source.scrollIntoViewIfNeeded();
      const before = (await source.boundingBox())!;
      let release!: () => void;
      const gate = new Promise<void>(resolve => { release = resolve; });
      await page.route('**/api/v1/events/*', async route => {
        if (route.request().method() !== 'PATCH') return route.continue();
        await gate;
        if (fails) await route.fulfill({ status: 400, body: '{}' });
        else await route.continue();
      });
      await page.mouse.move(before.x + 10, before.y + 10);
      await page.mouse.down();
      await page.mouse.move(before.x + 10, before.y + 58, { steps: 5 });
      await expect(page.locator('.drag-source')).toHaveCSS('opacity', '0.3');
      await page.mouse.up();
      const preview = page.locator('.drag-preview');
      await expect(preview).toBeVisible();
      await expect(preview).toHaveClass(/drag-pending/);
      expect((await preview.boundingBox())!.y).toBeCloseTo(before.y + 48, 0);
      expect((await page.locator('.drag-source').boundingBox())!.y).toBeCloseTo(before.y, 0);
      release();
      await expect(preview).toHaveCount(0);
      await expect(page.locator('.drag-source')).toHaveCount(0);
      expect((await source.boundingBox())!.y).toBeCloseTo(before.y + (fails ? 0 : 48), 0);
      await expect(page.locator('dialog.event-dialog')).not.toBeVisible();
      await source.click();
      await expect(page.locator('dialog.event-dialog')).toBeVisible();
    });
  }
}

for (const kind of ['cross-day', 'all-day', 'resize', 'all-day-no-change']) {
  test(`Week ${kind} preview survives the save`, async ({ page, request }) => {
    await clearAllEvents(request);
    const today = todayDate();
    const data = kind.startsWith('all-day')
      ? { title: 'Drag me', all_day: true, start_date: today,
          end_date: new Date(new Date(`${today}T00:00:00Z`).getTime() + 86400000).toISOString().slice(0, 10) }
      : { title: 'Drag me', all_day: false, start_time: new Date(`${today}T10:00:00`).toISOString(),
          end_time: new Date(`${today}T11:00:00`).toISOString() };
    expect((await request.post('/api/v1/events', { data })).ok()).toBeTruthy();
    await page.goto('/');
    await page.getByRole('button', { name: 'Week', exact: true }).click();
    const source = page.locator(kind.startsWith('all-day') ? '.allday-event' : '.week-event', { hasText: 'Drag me' });
    await source.scrollIntoViewIfNeeded();
    const before = (await source.boundingBox())!;
    const columns = page.locator(kind.startsWith('all-day') ? '.allday-cell' : '.week-day-events');
    const first = (await columns.nth(0).boundingBox())!;
    const second = (await columns.nth(1).boundingBox())!;
    // Move inward even when today is the last day of the displayed week.
    const dx = kind === 'all-day-no-change' ? 5 : kind === 'resize' ? 0 : (before.x > first.x + 5 * first.width ? -1 : 1) * (second.x - first.x);
    let release!: () => void;
    const gate = new Promise<void>(resolve => { release = resolve; });
    await page.route('**/api/v1/events/*', async route => {
      if (route.request().method() !== 'PATCH') return route.continue();
      await gate;
      await route.continue();
    });
    const y = kind === 'resize' ? before.y + before.height - 2 : before.y + 8;
    await page.mouse.move(before.x + 10, y);
    await page.mouse.down();
    await page.mouse.move(before.x + 10 + dx, y + (kind === 'resize' ? 48 : 0), { steps: 5 });
    await page.mouse.up();
    if (kind === 'all-day-no-change') {
      await expect(page.locator('.drag-preview')).toHaveCount(0);
      await expect(page.locator('dialog.event-dialog')).not.toBeVisible();
      await source.click();
      await expect(page.locator('dialog.event-dialog')).toBeVisible();
      return;
    }
    const preview = page.locator('.drag-preview');
    await expect(preview).toHaveClass(/drag-pending/);
    const pending = (await preview.boundingBox())!;
    expect(pending.x).toBeCloseTo(before.x + dx, 0);
    expect(pending.height).toBeCloseTo(before.height + (kind === 'resize' ? 48 : 0), 0);
    release();
    await expect(preview).toHaveCount(0);
    const after = (await source.boundingBox())!;
    expect(after.x).toBeCloseTo(pending.x, 0);
    expect(after.height).toBeCloseTo(pending.height, 0);
  });
}
