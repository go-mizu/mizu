import type { TableRecord, Field, CellValue, Attachment } from '../../../types';

export type CalendarViewMode = 'month' | 'week' | 'day';

export interface CalendarConfig {
  dateField: string | null;
  endDateField: string | null;
  viewMode: CalendarViewMode;
  weekStart: 0 | 1; // 0=Sunday, 1=Monday
  showWeekends: boolean;
  colorField: string | null;
  coverField: string | null;
  displayFields: string[];
  eventSize: 'compact' | 'comfortable';
  showTime: boolean;
}

export const DEFAULT_CALENDAR_CONFIG: CalendarConfig = {
  dateField: null,
  endDateField: null,
  viewMode: 'month',
  weekStart: 0,
  showWeekends: true,
  colorField: null,
  coverField: null,
  displayFields: [],
  eventSize: 'compact',
  showTime: false,
};

export interface CalendarDay {
  date: Date;
  dateKey: string; // YYYY-MM-DD format
  isCurrentMonth: boolean;
  isToday: boolean;
  isWeekend: boolean;
  events: CalendarEvent[];
}

export interface CalendarEvent {
  record: TableRecord;
  title: string;
  startDate: Date;
  endDate: Date | null;
  isAllDay: boolean;
  isMultiDay: boolean;
  color: string;
  coverImage?: Attachment;
  displayValues: { field: Field; value: CellValue }[];
}

export interface EventSpan {
  event: CalendarEvent;
  weekIndex: number;
  startCol: number;
  endCol: number;
  row: number;
  isStart: boolean;
  isEnd: boolean;
}

export interface TimeSlot {
  hour: number;
  minutes: number;
  label: string;
}

export interface CalendarWeek {
  weekStart: Date;
  weekEnd: Date;
  days: CalendarDay[];
}

// Colors for events when no color field is set
export const DEFAULT_EVENT_COLORS = [
  '#2D7FF9', // Blue
  '#20C933', // Green
  '#FCAE40', // Orange
  '#F82B60', // Red
  '#8B46FF', // Purple
  '#0D9488', // Teal
  '#EC4899', // Pink
  '#F97316', // Amber
];

// Get consistent color for a record based on ID
export function getDefaultEventColor(recordId: string): string {
  let hash = 0;
  for (let i = 0; i < recordId.length; i++) {
    hash = recordId.charCodeAt(i) + ((hash << 5) - hash);
  }
  return DEFAULT_EVENT_COLORS[Math.abs(hash) % DEFAULT_EVENT_COLORS.length];
}

// Date utilities
export function formatDateKey(date: Date): string {
  return date.toISOString().split('T')[0];
}

export function isSameDay(date1: Date, date2: Date): boolean {
  return (
    date1.getFullYear() === date2.getFullYear() &&
    date1.getMonth() === date2.getMonth() &&
    date1.getDate() === date2.getDate()
  );
}

export function isWeekend(date: Date): boolean {
  const day = date.getDay();
  return day === 0 || day === 6;
}

export function addDays(date: Date, days: number): Date {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}

export function startOfDay(date: Date): Date {
  const result = new Date(date);
  result.setHours(0, 0, 0, 0);
  return result;
}

export function endOfDay(date: Date): Date {
  const result = new Date(date);
  result.setHours(23, 59, 59, 999);
  return result;
}

export function startOfWeek(date: Date, weekStart: 0 | 1 = 0): Date {
  const result = new Date(date);
  const day = result.getDay();
  const diff = (day < weekStart ? 7 : 0) + day - weekStart;
  result.setDate(result.getDate() - diff);
  result.setHours(0, 0, 0, 0);
  return result;
}

export function endOfWeek(date: Date, weekStart: 0 | 1 = 0): Date {
  const result = startOfWeek(date, weekStart);
  result.setDate(result.getDate() + 6);
  result.setHours(23, 59, 59, 999);
  return result;
}

export function startOfMonth(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), 1);
}

export function endOfMonth(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth() + 1, 0);
}

export function differenceInDays(date1: Date, date2: Date): number {
  const d1 = startOfDay(date1);
  const d2 = startOfDay(date2);
  return Math.round((d1.getTime() - d2.getTime()) / (1000 * 60 * 60 * 24));
}

export function differenceInMinutes(date1: Date, date2: Date): number {
  return Math.round((date1.getTime() - date2.getTime()) / (1000 * 60));
}

export function formatTime(date: Date, use24Hour = false): string {
  if (use24Hour) {
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    });
  }
  return date.toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  });
}

export function formatWeekRange(startDate: Date, endDate: Date): string {
  const startMonth = startDate.toLocaleDateString('en-US', { month: 'short' });
  const endMonth = endDate.toLocaleDateString('en-US', { month: 'short' });
  const startDay = startDate.getDate();
  const endDay = endDate.getDate();
  const year = endDate.getFullYear();

  if (startMonth === endMonth) {
    return `${startMonth} ${startDay} - ${endDay}, ${year}`;
  }
  return `${startMonth} ${startDay} - ${endMonth} ${endDay}, ${year}`;
}

export function formatDayHeader(date: Date): string {
  return date.toLocaleDateString('en-US', {
    weekday: 'long',
    month: 'long',
    day: 'numeric',
    year: 'numeric',
  });
}
