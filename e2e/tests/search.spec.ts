import { test, expect } from '@playwright/test';
import { clearAllEvents, createEventViaAPI } from './helpers';

test.describe('Search', () => {
  test.beforeEach(async ({ request }) => {
    await clearAllEvents(request);
  });

  test('search finds matching events', async ({ page, request }) => {
    await createEventViaAPI(request, {
      title: 'Team Standup',
      start_time: '2026-03-01T09:00:00Z',
      end_time: '2026-03-01T09:30:00Z',
    });
    await createEventViaAPI(request, {
      title: 'Lunch Break',
      start_time: '2026-03-01T12:00:00Z',
      end_time: '2026-03-01T13:00:00Z',
    });

    await page.goto('/');

    const searchInput = page.getByRole('searchbox', { name: 'Search events...' });
    await searchInput.fill('Standup');

    // Wait for search results
    await expect(page.locator('.search-results')).toBeVisible();
    await expect(page.locator('.search-result-item')).toHaveCount(1);
    await expect(page.locator('.search-result-item', { hasText: 'Team Standup' })).toBeVisible();
  });

  test('search with no matches shows empty message', async ({ page, request }) => {
    await createEventViaAPI(request, {
      title: 'Some Event',
      start_time: '2026-03-01T09:00:00Z',
      end_time: '2026-03-01T10:00:00Z',
    });

    await page.goto('/');

    const searchInput = page.getByRole('searchbox', { name: 'Search events...' });
    await searchInput.fill('NonExistentXYZ');

    await expect(page.locator('.search-empty')).toBeVisible();
  });

  test('click search result opens event dialog', async ({ page, request }) => {
    await createEventViaAPI(request, {
      title: 'Clickable Event',
      start_time: '2026-03-01T14:00:00Z',
      end_time: '2026-03-01T15:00:00Z',
    });

    await page.goto('/');

    const searchInput = page.getByRole('searchbox', { name: 'Search events...' });
    await searchInput.fill('Clickable');

    await expect(page.locator('.search-result-item')).toBeVisible();
    await page.locator('.search-result-item').click();

    const dialog = page.locator('dialog.event-dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole('textbox', { name: 'Title' })).toHaveValue('Clickable Event');
  });

  test('clear search returns to calendar view', async ({ page, request }) => {
    await createEventViaAPI(request, {
      title: 'Test Event',
      start_time: '2026-03-01T09:00:00Z',
      end_time: '2026-03-01T10:00:00Z',
    });

    await page.goto('/');

    const searchInput = page.getByRole('searchbox', { name: 'Search events...' });
    await searchInput.fill('Test');
    await expect(page.locator('.search-results')).toBeVisible();

    // Clear search
    await searchInput.fill('');

    // Search results should disappear, calendar visible
    await expect(page.locator('.search-results')).not.toBeVisible();
    await expect(page.locator('.calendar-grid')).toBeVisible();
  });
});
