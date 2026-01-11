import { useMemo, useState } from 'react';
import type { TableRecord } from '../../../types';
import type { CalendarConfig, CalendarEvent as CalendarEventType, CalendarDay, EventSpan } from './types';
import {
  startOfWeek,
  addDays,
  isSameDay,
  formatDateKey,
  isWeekend,
} from './types';
import { CalendarEvent } from './CalendarEvent';

interface CalendarMonthViewProps {
  currentDate: Date;
  events: CalendarEventType[];
  config: CalendarConfig;
  onEventClick: (record: TableRecord) => void;
  onAddRecord: (date: Date) => void;
}

const WEEKDAY_LABELS_SUN = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const WEEKDAY_LABELS_MON = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

const MAX_VISIBLE_EVENTS = 3;

export function CalendarMonthView({
  currentDate,
  events,
  config,
  onEventClick,
  onAddRecord,
}: CalendarMonthViewProps) {
  const [moreEventsPopover, setMoreEventsPopover] = useState<{
    date: Date;
    events: CalendarEventType[];
    position: { x: number; y: number };
  } | null>(null);

  // Calculate which days to show
  const days = config.showWeekends ? 7 : 5;
  const weekdayLabels = useMemo(() => {
    const labels = config.weekStart === 0 ? WEEKDAY_LABELS_SUN : WEEKDAY_LABELS_MON;
    return config.showWeekends ? labels : labels.filter((_, i) => i < 5);
  }, [config.weekStart, config.showWeekends]);

  // Build calendar grid
  const calendarWeeks = useMemo(() => {
    const year = currentDate.getFullYear();
    const month = currentDate.getMonth();
    const firstOfMonth = new Date(year, month, 1);
    const lastOfMonth = new Date(year, month + 1, 0);

    // Start from the beginning of the week containing the first of the month
    const gridStart = startOfWeek(firstOfMonth, config.weekStart);

    // Build 6 weeks (42 days) to ensure full grid
    const weeks: CalendarDay[][] = [];
    let currentDay = new Date(gridStart);

    for (let week = 0; week < 6; week++) {
      const weekDays: CalendarDay[] = [];

      for (let day = 0; day < 7; day++) {
        // Skip weekends if configured
        if (!config.showWeekends && isWeekend(currentDay)) {
          currentDay = addDays(currentDay, 1);
          continue;
        }

        const today = new Date();
        weekDays.push({
          date: new Date(currentDay),
          dateKey: formatDateKey(currentDay),
          isCurrentMonth: currentDay.getMonth() === month,
          isToday: isSameDay(currentDay, today),
          isWeekend: isWeekend(currentDay),
          events: [],
        });

        currentDay = addDays(currentDay, 1);
      }

      weeks.push(weekDays);

      // Stop if we've passed the month and have at least 4 weeks
      if (currentDay > lastOfMonth && weeks.length >= 4) {
        // Check if the last week is completely in the next month
        const lastWeek = weeks[weeks.length - 1];
        if (lastWeek.every((d) => !d.isCurrentMonth) && weeks.length > 4) {
          weeks.pop();
        }
        break;
      }
    }

    return weeks;
  }, [currentDate, config.weekStart, config.showWeekends]);

  // Assign single-day events to days
  const singleDayEventsByDate = useMemo(() => {
    const map = new Map<string, CalendarEventType[]>();

    events
      .filter((event) => !event.isMultiDay)
      .forEach((event) => {
        const dateKey = formatDateKey(event.startDate);
        if (!map.has(dateKey)) {
          map.set(dateKey, []);
        }
        map.get(dateKey)!.push(event);
      });

    return map;
  }, [events]);

  // Calculate multi-day event spans
  const multiDayEventSpans = useMemo(() => {
    const multiDayEvents = events.filter((event) => event.isMultiDay);
    const spans: EventSpan[] = [];

    multiDayEvents.forEach((event) => {
      // Calculate the row index for this event (for stacking)
      const existingRows = new Set<number>();

      calendarWeeks.forEach((week, weekIndex) => {
        if (week.length === 0) return;

        const weekStart = week[0].date;
        const weekEnd = week[week.length - 1].date;
        const endDate = event.endDate || event.startDate;

        // Check if event overlaps with this week
        if (event.startDate <= weekEnd && endDate >= weekStart) {
          // Find start and end columns within this week
          let startCol = 0;
          let endCol = week.length - 1;

          for (let i = 0; i < week.length; i++) {
            if (isSameDay(week[i].date, event.startDate) || week[i].date > event.startDate) {
              startCol = i;
              break;
            }
          }

          for (let i = week.length - 1; i >= 0; i--) {
            if (isSameDay(week[i].date, endDate) || week[i].date < endDate) {
              endCol = i;
              break;
            }
          }

          // If event starts before this week, start at column 0
          if (event.startDate < weekStart) {
            startCol = 0;
          }

          // If event ends after this week, end at last column
          if (endDate > weekEnd) {
            endCol = week.length - 1;
          }

          // Find available row
          let row = 0;
          while (existingRows.has(row)) {
            row++;
          }
          existingRows.add(row);

          spans.push({
            event,
            weekIndex,
            startCol,
            endCol,
            row,
            isStart: isSameDay(week[startCol].date, event.startDate),
            isEnd: isSameDay(week[endCol].date, endDate),
          });
        }
      });
    });

    return spans;
  }, [events, calendarWeeks]);

  // Group spans by week
  const spansByWeek = useMemo(() => {
    const map = new Map<number, EventSpan[]>();
    multiDayEventSpans.forEach((span) => {
      if (!map.has(span.weekIndex)) {
        map.set(span.weekIndex, []);
      }
      map.get(span.weekIndex)!.push(span);
    });
    return map;
  }, [multiDayEventSpans]);

  // Handle more events click
  const handleMoreClick = (
    e: React.MouseEvent,
    date: Date,
    allEvents: CalendarEventType[]
  ) => {
    e.stopPropagation();
    const rect = (e.target as HTMLElement).getBoundingClientRect();
    setMoreEventsPopover({
      date,
      events: allEvents,
      position: { x: rect.left, y: rect.bottom + 4 },
    });
  };

  return (
    <div className="flex-1 flex flex-col">
      {/* Week day headers */}
      <div
        className="grid bg-slate-50 border-b border-slate-200"
        style={{ gridTemplateColumns: `repeat(${days}, minmax(0, 1fr))` }}
      >
        {weekdayLabels.map((day) => (
          <div
            key={day}
            className="p-2 text-center text-sm font-semibold text-slate-600"
          >
            {day}
          </div>
        ))}
      </div>

      {/* Calendar grid */}
      <div className="flex-1 flex flex-col">
        {calendarWeeks.map((week, weekIndex) => (
          <div
            key={weekIndex}
            className="flex-1 grid border-b border-slate-200 relative"
            style={{
              gridTemplateColumns: `repeat(${days}, minmax(0, 1fr))`,
              minHeight: '120px',
            }}
          >
            {/* Multi-day event spans */}
            <div
              className="absolute inset-x-0 top-8 pointer-events-none"
              style={{ zIndex: 10 }}
            >
              {spansByWeek.get(weekIndex)?.map((span, spanIndex) => {
                const colWidth = 100 / days;
                const left = span.startCol * colWidth;
                const width = (span.endCol - span.startCol + 1) * colWidth;

                return (
                  <div
                    key={`${span.event.record.id}-${spanIndex}`}
                    className="absolute pointer-events-auto"
                    style={{
                      left: `calc(${left}% + 2px)`,
                      width: `calc(${width}% - 4px)`,
                      top: `${span.row * 24}px`,
                    }}
                  >
                    <CalendarEvent
                      event={span.event}
                      config={config}
                      onClick={() => onEventClick(span.event.record)}
                      variant="month"
                      isSpanStart={span.isStart}
                      isSpanEnd={span.isEnd}
                    />
                  </div>
                );
              })}
            </div>

            {/* Day cells */}
            {week.map((day) => {
              const dayEvents = singleDayEventsByDate.get(day.dateKey) || [];
              const multiDayCount = (spansByWeek.get(weekIndex) || []).filter(
                (span) => span.startCol <= week.indexOf(day) && span.endCol >= week.indexOf(day)
              ).length;

              // Reserve space for multi-day events
              const availableSlots = MAX_VISIBLE_EVENTS - Math.min(multiDayCount, MAX_VISIBLE_EVENTS - 1);
              const visibleEvents = dayEvents.slice(0, Math.max(1, availableSlots));
              const hiddenCount = dayEvents.length - visibleEvents.length;

              return (
                <div
                  key={day.dateKey}
                  className={`border-r border-slate-200 p-1 overflow-hidden group ${
                    !day.isCurrentMonth ? 'bg-slate-50' : ''
                  } ${day.isWeekend && day.isCurrentMonth ? 'bg-slate-50/50' : ''}`}
                >
                  {/* Day header */}
                  <div className="flex items-center justify-between mb-1">
                    <span
                      className={`w-7 h-7 flex items-center justify-center text-sm rounded-full ${
                        day.isToday
                          ? 'bg-primary text-white font-medium'
                          : day.isCurrentMonth
                          ? 'text-gray-900'
                          : 'text-gray-400'
                      }`}
                    >
                      {day.date.getDate()}
                    </span>

                    {day.isCurrentMonth && (
                      <button
                        onClick={() => onAddRecord(day.date)}
                        className="w-5 h-5 flex items-center justify-center text-gray-400 hover:text-gray-600 hover:bg-slate-100 rounded opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        <svg
                          className="w-4 h-4"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M12 4v16m8-8H4"
                          />
                        </svg>
                      </button>
                    )}
                  </div>

                  {/* Single-day events */}
                  <div
                    className="space-y-0.5"
                    style={{ marginTop: `${multiDayCount * 24 + (multiDayCount > 0 ? 4 : 0)}px` }}
                  >
                    {visibleEvents.map((event) => (
                      <CalendarEvent
                        key={event.record.id}
                        event={event}
                        config={config}
                        onClick={() => onEventClick(event.record)}
                        variant="month"
                      />
                    ))}

                    {hiddenCount > 0 && (
                      <button
                        onClick={(e) =>
                          handleMoreClick(e, day.date, dayEvents)
                        }
                        className="text-xs text-gray-500 hover:text-gray-700 hover:bg-slate-100 rounded px-1 py-0.5 w-full text-left"
                      >
                        +{hiddenCount} more
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        ))}
      </div>

      {/* More events popover */}
      {moreEventsPopover && (
        <>
          <div
            className="fixed inset-0 z-40"
            onClick={() => setMoreEventsPopover(null)}
          />
          <div
            className="fixed z-50 bg-white rounded-lg shadow-xl border border-slate-200 p-3 w-64"
            style={{
              left: Math.min(moreEventsPopover.position.x, window.innerWidth - 280),
              top: Math.min(moreEventsPopover.position.y, window.innerHeight - 300),
            }}
          >
            <div className="text-sm font-semibold text-gray-900 mb-2">
              {moreEventsPopover.date.toLocaleDateString('en-US', {
                weekday: 'long',
                month: 'long',
                day: 'numeric',
              })}
            </div>
            <div className="space-y-1 max-h-60 overflow-y-auto">
              {moreEventsPopover.events.map((event) => (
                <CalendarEvent
                  key={event.record.id}
                  event={event}
                  config={config}
                  onClick={() => {
                    onEventClick(event.record);
                    setMoreEventsPopover(null);
                  }}
                  variant="month"
                />
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
