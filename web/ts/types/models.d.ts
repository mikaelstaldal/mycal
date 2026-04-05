import type { components } from './api.js';

// API schema types — generated from openapi.yaml via openapi-typescript
export type CalendarEvent = components['schemas']['Event'] & { _editInstance?: boolean };
export type CalendarMeta = components['schemas']['Calendar'];
export type Feed = components['schemas']['Feed'];
export type Preferences = components['schemas']['Preferences'];

// Request types
export type CreateEventRequest = components['schemas']['CreateEventRequest'];
export type UpdateEventRequest = components['schemas']['UpdateEventRequest'];
export type CreateFeedRequest = components['schemas']['CreateFeedRequest'];
export type UpdateCalendarRequest = components['schemas']['UpdateCalendarRequest'];

// AppConfig is app-level configuration, not part of the API
export interface AppConfig {
  defaultView: 'year' | 'month' | 'week' | 'day' | 'schedule';
  dayStartHour: number;
  weekStartDay: number;
  defaultEventColor: string;
  mapProvider: 'none' | 'openstreetmap' | 'google';
  googleMapsApiKey: string;
  calendarColors?: Record<number, string>;
}
