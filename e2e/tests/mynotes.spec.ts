import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';
import { clearAllEvents, createEventViaAPI, todayDate } from './helpers';

// MyNotes is a separate app deployed next to mycal. These tests stand in for it
// with routed responses on the same origin: the note API, and a stub of the
// render kit's host page that records what mycal pushes into it. That keeps the
// integration's own wiring — picker, note_slug round-trip, driving
// render()/setTheme() — under test without a second server.
const MYNOTES_URL = 'http://localhost:8089/mynotes';

const NOTE = {
  slug: 'sprint-planning',
  title: 'Sprint planning',
  content: '# Sprint planning\n\nAgenda for the kickoff.\n',
  updated_at: '2026-07-01T09:00:00Z',
};

// Stands in for <mynotesUrl>/render/: exposes the same MyNotesRender contract,
// dropping the Markdown it is given into the DOM verbatim.
const RENDER_HOST_HTML = `<!DOCTYPE html>
<html><body><div id="note"></div><script>
  globalThis.MyNotesRender = {
    render(markdown) { document.getElementById('note').textContent = markdown; return Promise.resolve(); },
    setTheme(theme) { document.documentElement.dataset.theme = theme; },
  };
</script></body></html>`;

async function stubMynotes(page: Page): Promise<void> {
  await page.route('**/mynotes/render/', route =>
    route.fulfill({ contentType: 'text/html', body: RENDER_HOST_HTML }));
  await page.route('**/mynotes/api/v1/notes?*', route =>
    route.fulfill({ contentType: 'application/json', body: JSON.stringify({ total: 1, notes: [NOTE] }) }));
  await page.route('**/mynotes/api/v1/notes/*', route =>
    route.fulfill({ contentType: 'application/json', body: JSON.stringify(NOTE) }));
  await page.addInitScript(url => {
    localStorage.setItem('mycal-settings', JSON.stringify({ mynotesUrl: url }));
  }, MYNOTES_URL);
}

test.describe('MyNotes integration', () => {
  test.beforeEach(async ({ page, request }) => {
    await clearAllEvents(request);
    await stubMynotes(page);
    await page.goto('/');
    await page.getByRole('button', { name: 'Month' }).click();
    await expect(page.locator('.calendar-grid')).toBeVisible();
  });

  test('link a note through the picker and persist it', async ({ page, request }) => {
    const today = todayDate();
    await createEventViaAPI(request, {
      title: 'Kickoff',
      all_day: false,
      start_time: `${today}T10:00:00Z`,
      end_time: `${today}T11:00:00Z`,
    });
    await page.reload();
    await page.getByRole('button', { name: 'Month' }).click();

    await page.locator('.event-chip', { hasText: 'Kickoff' }).click();
    const dialog = page.locator('dialog.event-dialog');
    await dialog.getByRole('button', { name: 'Edit' }).click();

    await dialog.getByRole('searchbox', { name: 'Note' }).fill('Spr');
    await dialog.getByRole('button', { name: NOTE.title }).click();
    await expect(dialog.locator('.note-picker-selected')).toContainText(NOTE.slug);

    await dialog.getByRole('button', { name: 'Save' }).click();

    const events = await (await request.get('/api/v1/events', {
      params: { from: '1900-01-01T00:00:00Z', to: '2200-01-01T00:00:00Z' },
    })).json();
    expect(events[0].note_slug).toBe(NOTE.slug);
  });

  test('show the linked note through the render kit', async ({ page, request }) => {
    const today = todayDate();
    await createEventViaAPI(request, {
      title: 'Kickoff',
      all_day: false,
      start_time: `${today}T10:00:00Z`,
      end_time: `${today}T11:00:00Z`,
      note_slug: NOTE.slug,
    });
    await page.reload();
    await page.getByRole('button', { name: 'Month' }).click();

    await page.locator('.event-chip', { hasText: 'Kickoff' }).click();
    const dialog = page.locator('dialog.event-dialog');

    // Header links to the note in MyNotes, content comes from the render kit.
    await expect(dialog.locator('.note-panel-header')).toContainText(NOTE.title);
    await expect(dialog.locator('.note-panel-header a')).toHaveAttribute(
      'href', `${MYNOTES_URL}/notes/${NOTE.slug}`);
    await expect(dialog.frameLocator('.note-panel-frame').locator('#note'))
      .toContainText('Agenda for the kickoff.');
  });

  test('unlink a note', async ({ page, request }) => {
    const today = todayDate();
    await createEventViaAPI(request, {
      title: 'Kickoff',
      all_day: false,
      start_time: `${today}T10:00:00Z`,
      end_time: `${today}T11:00:00Z`,
      note_slug: NOTE.slug,
    });
    await page.reload();
    await page.getByRole('button', { name: 'Month' }).click();

    await page.locator('.event-chip', { hasText: 'Kickoff' }).click();
    const dialog = page.locator('dialog.event-dialog');
    await dialog.getByRole('button', { name: 'Edit' }).click();
    await dialog.getByRole('button', { name: 'Unlink' }).click();
    await dialog.getByRole('button', { name: 'Save' }).click();

    const events = await (await request.get('/api/v1/events', {
      params: { from: '1900-01-01T00:00:00Z', to: '2200-01-01T00:00:00Z' },
    })).json();
    expect(events[0].note_slug).toBeUndefined();
  });
});
