export interface CalendarEvent {
  id: string;
  title: string;
  description: string;
  start_time: string;
  end_time: string;
  all_day: boolean;
  color: string;
  calendar_id: number;
  location?: string;
  latitude?: number | null;
  longitude?: number | null;
  reminder_minutes?: number | null;
  recurrence_freq?: string;
  recurrence_count?: number;
  recurrence_until?: string;
  recurrence_interval?: number;
  recurrence_by_day?: string;
  recurrence_by_monthday?: string;
  recurrence_by_month?: string;
  exdates?: string;
  rdates?: string;
  duration?: string;
  categories?: string;
  url?: string;
  parent_id?: string;
  recurrence_parent_id?: string;
  _editInstance?: boolean;
}

export interface CalendarMeta {
  id: number;
  name: string;
  color: string;
}

export interface Feed {
  id: number;
  url: string;
  name: string;
  color: string;
  last_refreshed?: string;
  calendar_name?: string;
  refresh_interval_minutes?: number;
  last_refreshed_at?: string;
  enabled?: boolean;
  last_error?: string;
}

export interface AppConfig {
  defaultView: 'year' | 'month' | 'week' | 'day' | 'schedule';
  dayStartHour: number;
  weekStartDay: number;
  defaultEventColor: string;
  mapProvider: 'none' | 'openstreetmap' | 'google';
  googleMapsApiKey: string;
  calendarColors?: Record<number, string>;
}

export interface Preferences {
  weekStartDay?: number;
  reminders?: boolean;
  [key: string]: any;
}
