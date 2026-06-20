import { showNetworkErrorToast } from '../util/toast.js';
import type { components } from './types.js';

type Event = components['schemas']['Event'];
type Calendar = components['schemas']['Calendar'];
type Feed = components['schemas']['Feed'];
type Preferences = components['schemas']['Preferences'];
type CreateEventRequest = components['schemas']['CreateEventRequest'];
type UpdateEventRequest = components['schemas']['UpdateEventRequest'];
type CreateFeedRequest = components['schemas']['CreateFeedRequest'];
type UpdateCalendarRequest = components['schemas']['UpdateCalendarRequest'];

// The app may be served from a sub-path; derive the API base from the document base URI.
const APP_BASE = new URL('.', document.baseURI).pathname.replace(/\/$/, '');
const BASE = APP_BASE + '/api/v1';

export class NotFoundError extends Error {
  constructor() { super('Not found'); this.name = 'NotFoundError'; }
}

function delay(ms: number): Promise<void> {
  return new Promise(r => setTimeout(r, ms));
}

async function fetchWithRetry(url: string, init: RequestInit): Promise<Response> {
  const isSafe = ['GET', 'HEAD'].includes((init.method ?? 'GET').toUpperCase());
  let pastAutoRetry = false;
  while (true) {
    try {
      return await fetch(url, init);
    } catch (e) {
      if (!(e instanceof TypeError)) throw e;
      if (isSafe && !pastAutoRetry) {
        await delay(2000);
        pastAutoRetry = true;
        continue;
      }
      await new Promise<void>(resolve => {
        showNetworkErrorToast('Network error. Please check your connection.', resolve);
      });
    }
  }
}

async function handle<T>(res: Response): Promise<T> {
  if (res.status === 401) { window.location.reload(); throw new Error('Unauthorized'); }
  if (res.status === 404) throw new NotFoundError();
  if (res.status === 204) return undefined as T;
  const data = await res.json() as unknown;
  if (!res.ok) {
    const err = data as { error?: string };
    throw new Error(err.error ?? res.statusText);
  }
  return data as T;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const init: RequestInit = { method, headers: { 'Content-Type': 'application/json' } };
  if (body !== undefined) init.body = JSON.stringify(body);
  return handle<T>(await fetchWithRetry(BASE + path, init));
}

// Import endpoints accept either a raw iCalendar body (text/calendar) or a JSON { url }.
async function importRequest<T>(endpoint: string, contentOrUrl: string, calendar?: string): Promise<T> {
  const isUrl = contentOrUrl.startsWith('http');
  let path = endpoint;
  if (calendar) path += `?calendar=${encodeURIComponent(calendar)}`;
  const res = await fetchWithRetry(BASE + path, {
    method: 'POST',
    headers: { 'Content-Type': isUrl ? 'application/json' : 'text/calendar' },
    body: isUrl ? JSON.stringify({ url: contentOrUrl }) : contentOrUrl,
  });
  return handle<T>(res);
}

export const api = {
  events: {
    list(from: string, to: string, calendarIds?: number[] | null): Promise<Event[]> {
      let path = `/events?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`;
      if (calendarIds) {
        for (const id of calendarIds) path += `&calendar_id=${encodeURIComponent(id)}`;
      }
      return request<Event[]>('GET', path);
    },

    get: (id: string) =>
      request<Event>('GET', `/events/${encodeURIComponent(id)}`),

    create: (data: CreateEventRequest) =>
      request<Event>('POST', '/events', data),

    update: (id: string, data: UpdateEventRequest) =>
      request<Event>('PATCH', `/events/${encodeURIComponent(id)}`, data),

    delete: (id: string) =>
      request<void>('DELETE', `/events/${encodeURIComponent(id)}`),

    search: (query: string) =>
      request<Event[]>('GET', `/events?q=${encodeURIComponent(query)}`),

    getICS: async (id: string): Promise<string> => {
      const res = await fetchWithRetry(`${BASE}/events/${encodeURIComponent(id)}/ics`, { method: 'GET' });
      if (res.status === 401) { window.location.reload(); throw new Error('Unauthorized'); }
      if (res.status === 404) throw new NotFoundError();
      if (!res.ok) throw new Error(`Failed to get event data (${res.status})`);
      return res.text();
    },
  },

  calendars: {
    list: () =>
      request<Calendar[]>('GET', '/calendars'),

    update: (id: number, data: UpdateCalendarRequest) =>
      request<Calendar>('PATCH', `/calendars/${encodeURIComponent(id)}`, data),
  },

  feeds: {
    list: () =>
      request<Feed[]>('GET', '/feeds'),

    create: (data: CreateFeedRequest) =>
      request<Feed>('POST', '/feeds', data),

    delete: (id: number) =>
      request<void>('DELETE', `/feeds/${encodeURIComponent(id)}`),

    refresh: (id: number) =>
      request<Feed>('POST', `/feeds/${encodeURIComponent(id)}/refresh`),
  },

  preferences: {
    get: () =>
      request<Preferences>('GET', '/preferences'),

    update: (prefs: Preferences) =>
      request<Preferences>('PATCH', '/preferences', prefs),
  },

  import: {
    single: (contentOrUrl: string, calendar?: string) =>
      importRequest<Event>('/import-single', contentOrUrl, calendar),

    bulk: (contentOrUrl: string, calendar?: string) =>
      importRequest<{ imported: number }>('/import', contentOrUrl, calendar),
  },
};
