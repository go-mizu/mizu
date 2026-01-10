import { useMemo, useRef, useEffect } from 'react';
import type { TableRecord } from '../../../types';
import type { CalendarConfig, CalendarEvent as CalendarEventType } from './types';
import {
  startOfWeek,
  endOfWeek,
  addDays,
  isSameDay,
  formatDateKey,
  isWeekend,
} from './types';
import { CalendarEvent } from './CalendarEvent';

interface CalendarWeekViewProps {
  currentDate: Date;
  events: CalendarEventType[];
  config: CalendarConfig;
  onEventClick: (record: TableRecord) => void;
  onAddRecord: (date: Date) => void;
}

const HOUR_HEIGHT = 60; // pixels per hour
const START_HOUR = 6; // 6 AM
const END_HOUR = 22; // 10 PM
const TOTAL_HOURS = END_HOUR - START_HOUR;

export function CalendarWeekView({
  currentDate,
  events,
  config,
  onEventClick,
  onAddRecord,
}: CalendarWeekViewProps) {
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // Scroll to current time on mount
  useEffect(() => {
    if (scrollContainerRef.current) {
      const now = new Date();
      const currentHour = now.getHours();
      if (currentHour >= START_HOUR && currentHour <= END_HOUR) {
        const scrollPosition = (currentHour - START_HOUR - 1) * HOUR_HEIGHT;
        scrollContainerRef.current.scrollTop = Math.max(0, scrollPosition);
      }
    }
  }, []);

  // Calculate week bounds
  const weekStart = useMemo(
    () => startOfWeek(currentDate, config.weekStart),
    [currentDate, config.weekStart]
  );
  const weekEnd = useMemo(
    () => endOfWeek(currentDate, config.weekStart),
    [currentDate, config.weekStart]
  );

  // Build days array
  const days = useMemo(() => {
    const result: Date[] = [];
    let day = new Date(weekStart);

    while (day <= weekEnd) {
      if (config.showWeekends || !isWeekend(day)) {
        result.push(new Date(day));
      }
      day = addDays(day, 1);
    }

    return result;
  }, [weekStart, weekEnd, config.showWeekends]);

  // Separate all-day and timed events
  const { allDayEvents, timedEventsByDay } = useMemo(() => {
    const allDay: CalendarEventType[] = [];
    const timed = new Map<string, CalendarEventType[]>();

    events.forEach((event) => {
      if (event.isAllDay || event.isMultiDay) {
        // Check if event overlaps with this week
        const endDate = event.endDate || event.startDate;
        if (event.startDate <= weekEnd && endDate >= weekStart) {
          allDay.push(event);
        }
      } else {
        // Timed event - check if it's in this week
        const dateKey = formatDateKey(event.startDate);
        if (event.startDate >= weekStart && event.startDate <= weekEnd) {
          if (!timed.has(dateKey)) {
            timed.set(dateKey, []);
          }
          timed.get(dateKey)!.push(event);
        }
      }
    });

    return { allDayEvents: allDay, timedEventsByDay: timed };
  }, [events, weekStart, weekEnd]);

  // Generate time slots
  const timeSlots = useMemo(() => {
    const slots = [];
    for (let hour = START_HOUR; hour <= END_HOUR; hour++) {
      slots.push({
        hour,
        label:
          hour === 0
            ? '12 AM'
            : hour < 12
            ? `${hour} AM`
            : hour === 12
            ? '12 PM'
            : `${hour - 12} PM`,
      });
    }
    return slots;
  }, []);

  // Calculate event positions
  const getEventStyle = (event: CalendarEventType) => {
    const startMinutes =
      event.startDate.getHours() * 60 + event.startDate.getMinutes();
    const endDate = event.endDate || new Date(event.startDate.getTime() + 60 * 60 * 1000);
    const endMinutes = endDate.getHours() * 60 + endDate.getMinutes();

    const startFromGrid = startMinutes - START_HOUR * 60;
    const duration = endMinutes - startMinutes;

    return {
      top: Math.max(0, (startFromGrid / 60) * HOUR_HEIGHT),
      height: Math.max(30, (duration / 60) * HOUR_HEIGHT),
    };
  };

  // Check if a time slot is current
  const now = new Date();
  const isToday = (date: Date) => isSameDay(date, now);
  const currentMinutes = now.getHours() * 60 + now.getMinutes();
  const currentTimeTop = ((currentMinutes - START_HOUR * 60) / 60) * HOUR_HEIGHT;

  const columnCount = days.length;

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Header with day names */}
      <div className="flex border-b border-slate-200 bg-white sticky top-0 z-20">
        {/* Time column header */}
        <div className="w-16 flex-shrink-0 border-r border-slate-200" />

        {/* Day headers */}
        <div className="flex-1 grid" style={{ gridTemplateColumns: `repeat(${columnCount}, minmax(0, 1fr))` }}>
          {days.map((day) => {
            const dayIsToday = isToday(day);
            return (
              <div
                key={formatDateKey(day)}
                className="border-r border-slate-200 p-2 text-center"
              >
                <div className="text-xs text-gray-500 uppercase">
                  {day.toLocaleDateString('en-US', { weekday: 'short' })}
                </div>
                <div
                  className={`text-lg font-semibold ${
                    dayIsToday
                      ? 'w-8 h-8 mx-auto flex items-center justify-center bg-primary text-white rounded-full'
                      : 'text-gray-900'
                  }`}
                >
                  {day.getDate()}
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* All-day events section */}
      {allDayEvents.length > 0 && (
        <div className="flex border-b border-slate-200 bg-slate-50">
          <div className="w-16 flex-shrink-0 border-r border-slate-200 p-1 text-xs text-gray-500 text-right pr-2">
            All-day
          </div>
          <div
            className="flex-1 grid gap-1 p-1 min-h-[32px]"
            style={{ gridTemplateColumns: `repeat(${columnCount}, minmax(0, 1fr))` }}
          >
            {days.map((day) => {
              const dayKey = formatDateKey(day);
              const dayAllDayEvents = allDayEvents.filter((event) => {
                const endDate = event.endDate || event.startDate;
                return event.startDate <= day && endDate >= day;
              });

              return (
                <div key={dayKey} className="min-h-[24px] space-y-0.5">
                  {dayAllDayEvents.slice(0, 2).map((event) => (
                    <CalendarEvent
                      key={event.record.id}
                      event={event}
                      config={config}
                      onClick={() => onEventClick(event.record)}
                      variant="week"
                      isSpanStart={isSameDay(day, event.startDate)}
                      isSpanEnd={isSameDay(day, event.endDate || event.startDate)}
                    />
                  ))}
                  {dayAllDayEvents.length > 2 && (
                    <div className="text-xs text-gray-500 px-1">
                      +{dayAllDayEvents.length - 2} more
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Time grid */}
      <div ref={scrollContainerRef} className="flex-1 overflow-auto">
        <div className="flex min-h-full">
          {/* Time labels */}
          <div className="w-16 flex-shrink-0 border-r border-slate-200 bg-white sticky left-0 z-10">
            {timeSlots.map((slot) => (
              <div
                key={slot.hour}
                className="border-b border-slate-100 text-xs text-gray-500 text-right pr-2 relative"
                style={{ height: HOUR_HEIGHT }}
              >
                <span className="absolute -top-2 right-2">{slot.label}</span>
              </div>
            ))}
          </div>

          {/* Day columns */}
          <div
            className="flex-1 grid relative"
            style={{ gridTemplateColumns: `repeat(${columnCount}, minmax(0, 1fr))` }}
          >
            {/* Grid lines */}
            <div
              className="absolute inset-0 pointer-events-none"
              style={{ gridColumn: `1 / -1` }}
            >
              {timeSlots.map((slot) => (
                <div
                  key={slot.hour}
                  className="border-b border-slate-100"
                  style={{ height: HOUR_HEIGHT }}
                />
              ))}
            </div>

            {/* Current time indicator */}
            {days.some(isToday) && currentTimeTop >= 0 && currentTimeTop <= TOTAL_HOURS * HOUR_HEIGHT && (
              <div
                className="absolute left-0 right-0 flex items-center z-20 pointer-events-none"
                style={{ top: currentTimeTop }}
              >
                <div className="w-2 h-2 bg-red-500 rounded-full -ml-1" />
                <div className="flex-1 h-0.5 bg-red-500" />
              </div>
            )}

            {/* Day columns with events */}
            {days.map((day) => {
              const dayKey = formatDateKey(day);
              const dayEvents = timedEventsByDay.get(dayKey) || [];
              const dayIsToday = isToday(day);

              return (
                <div
                  key={dayKey}
                  className={`border-r border-slate-200 relative ${
                    dayIsToday ? 'bg-primary/5' : ''
                  }`}
                  style={{ height: TOTAL_HOURS * HOUR_HEIGHT }}
                  onClick={(e) => {
                    // Calculate clicked time
                    const rect = e.currentTarget.getBoundingClientRect();
                    const y = e.clientY - rect.top;
                    const hours = Math.floor(y / HOUR_HEIGHT) + START_HOUR;
                    const minutes = Math.floor((y % HOUR_HEIGHT) / (HOUR_HEIGHT / 2)) * 30;

                    const clickedDate = new Date(day);
                    clickedDate.setHours(hours, minutes, 0, 0);
                    onAddRecord(clickedDate);
                  }}
                >
                  {/* Events */}
                  {dayEvents.map((event, eventIndex) => {
                    const style = getEventStyle(event);

                    // Simple overlap handling - offset overlapping events
                    const overlappingEvents = dayEvents.filter((e, i) => {
                      if (i >= eventIndex) return false;
                      const eStyle = getEventStyle(e);
                      return (
                        style.top < eStyle.top + eStyle.height &&
                        style.top + style.height > eStyle.top
                      );
                    });
                    const offsetIndex = overlappingEvents.length;

                    return (
                      <div
                        key={event.record.id}
                        className="absolute left-1 right-1"
                        style={{
                          top: style.top,
                          height: style.height,
                          left: `calc(4px + ${offsetIndex * 20}%)`,
                          right: `calc(4px + ${offsetIndex * 5}%)`,
                          zIndex: eventIndex + 1,
                        }}
                        onClick={(e) => e.stopPropagation()}
                      >
                        <CalendarEvent
                          event={event}
                          config={config}
                          onClick={() => onEventClick(event.record)}
                          variant="week"
                          showFullDetails={style.height > 60}
                        />
                      </div>
                    );
                  })}
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
