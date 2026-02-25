import { APIRequestContext } from '@playwright/test';

export interface EventData {
  title: string;
  start_time: string;
  end_time: string;
  all_day?: boolean;
  description?: string;
  location?: string;
  color?: string;
}

export async function createEventViaAPI(
  request: APIRequestContext,
  event: EventData
): Promise<{ id: number }> {
  const response = await request.post('/api/v1/events', { data: event });
  if (!response.ok()) {
    throw new Error(`Failed to create event: ${response.status()} ${await response.text()}`);
  }
  return response.json();
}

export async function clearAllEvents(request: APIRequestContext): Promise<void> {
  const response = await request.get('/api/v1/events', {
    params: { from: '1900-01-01T00:00:00Z', to: '2200-01-01T00:00:00Z' },
  });
  if (!response.ok()) return;
  const events = await response.json();
  for (const event of events) {
    await request.delete(`/api/v1/events/${event.id}`);
  }
}

export function todayDate(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

export function todayMonthYear(): string {
  const d = new Date();
  return d.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
}
