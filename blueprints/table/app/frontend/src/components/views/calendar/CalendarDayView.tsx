import { useMemo, useRef, useEffect } from 'react';
import type { TableRecord } from '../../../types';
import type { CalendarConfig, CalendarEvent as CalendarEventType } from './types';
import { isSameDay, startOfDay, endOfDay } from './types';
import { CalendarEvent } from './CalendarEvent';

interface CalendarDayViewProps {
  currentDate: Date;
  events: CalendarEventType[];
  config: CalendarConfig;
  onEventClick: (record: TableRecord) => void;
  onAddRecord: (date: Date) => void;
}

const HOUR_HEIGHT = 80; // pixels per hour - taller for day view
const START_HOUR = 0; // 12 AM
const END_HOUR = 24; // 12 AM next day
const TOTAL_HOURS = END_HOUR - START_HOUR;

export function CalendarDayView({
  currentDate,
  events,
  config,
  onEventClick,
  onAddRecord,
}: CalendarDayViewProps) {
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // Scroll to current time or business hours on mount
  useEffect(() => {
    if (scrollContainerRef.current) {
      const now = new Date();
      const isToday = isSameDay(currentDate, now);

      if (isToday) {
        const currentHour = now.getHours();
        const scrollPosition = (currentHour - 1) * HOUR_HEIGHT;
        scrollContainerRef.current.scrollTop = Math.max(0, scrollPosition);
      } else {
        // Scroll to 8 AM for other days
        scrollContainerRef.current.scrollTop = 8 * HOUR_HEIGHT;
      }
    }
  }, [currentDate]);

  // Filter events for this day
  const { allDayEvents, timedEvents } = useMemo(() => {
    const dayStart = startOfDay(currentDate);
    const dayEnd = endOfDay(currentDate);

    const allDay: CalendarEventType[] = [];
    const timed: CalendarEventType[] = [];

    events.forEach((event) => {
      const endDate = event.endDate || event.startDate;

      // Check if event overlaps with this day
      if (event.startDate <= dayEnd && endDate >= dayStart) {
        if (event.isAllDay || event.isMultiDay) {
          allDay.push(event);
        } else if (isSameDay(event.startDate, currentDate)) {
          timed.push(event);
        }
      }
    });

    // Sort timed events by start time
    timed.sort((a, b) => a.startDate.getTime() - b.startDate.getTime());

    return { allDayEvents: allDay, timedEvents: timed };
  }, [events, currentDate]);

  // Generate time slots
  const timeSlots = useMemo(() => {
    const slots = [];
    for (let hour = START_HOUR; hour < END_HOUR; hour++) {
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
    const endDate =
      event.endDate || new Date(event.startDate.getTime() + 60 * 60 * 1000);
    const endMinutes = endDate.getHours() * 60 + endDate.getMinutes();

    const startFromGrid = startMinutes - START_HOUR * 60;
    const duration = Math.max(30, endMinutes - startMinutes); // Minimum 30 min

    return {
      top: (startFromGrid / 60) * HOUR_HEIGHT,
      height: (duration / 60) * HOUR_HEIGHT,
    };
  };

  // Calculate overlapping events for positioning
  const eventPositions = useMemo(() => {
    const positions = new Map<string, { left: string; width: string }>();

    // Group overlapping events
    const groups: CalendarEventType[][] = [];

    timedEvents.forEach((event) => {
      const style = getEventStyle(event);
      const eventTop = style.top;
      const eventBottom = style.top + style.height;

      // Find a group this event overlaps with
      let foundGroup: CalendarEventType[] | null = null;

      for (const group of groups) {
        const groupOverlaps = group.some((e) => {
          const eStyle = getEventStyle(e);
          const eTop = eStyle.top;
          const eBottom = eStyle.top + eStyle.height;
          return eventTop < eBottom && eventBottom > eTop;
        });

        if (groupOverlaps) {
          foundGroup = group;
          break;
        }
      }

      if (foundGroup) {
        foundGroup.push(event);
      } else {
        groups.push([event]);
      }
    });

    // Calculate positions for each group
    groups.forEach((group) => {
      const width = 100 / group.length;
      group.forEach((event, index) => {
        positions.set(event.record.id, {
          left: `${index * width}%`,
          width: `${width}%`,
        });
      });
    });

    return positions;
  }, [timedEvents]);

  // Current time indicator
  const now = new Date();
  const isToday = isSameDay(currentDate, now);
  const currentMinutes = now.getHours() * 60 + now.getMinutes();
  const currentTimeTop = ((currentMinutes - START_HOUR * 60) / 60) * HOUR_HEIGHT;

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Day header */}
      <div className="flex items-center justify-center py-4 border-b border-slate-200 bg-white">
        <div className="text-center">
          <div className="text-sm text-gray-500 uppercase">
            {currentDate.toLocaleDateString('en-US', { weekday: 'long' })}
          </div>
          <div
            className={`text-3xl font-bold ${
              isToday
                ? 'w-12 h-12 mx-auto flex items-center justify-center bg-primary text-white rounded-full'
                : 'text-gray-900'
            }`}
          >
            {currentDate.getDate()}
          </div>
          <div className="text-sm text-gray-500">
            {currentDate.toLocaleDateString('en-US', {
              month: 'long',
              year: 'numeric',
            })}
          </div>
        </div>
      </div>

      {/* All-day events */}
      {allDayEvents.length > 0 && (
        <div className="flex border-b border-slate-200 bg-slate-50">
          <div className="w-20 flex-shrink-0 border-r border-slate-200 p-2 text-xs text-gray-500 text-right">
            All-day
          </div>
          <div className="flex-1 p-2 space-y-1">
            {allDayEvents.map((event) => (
              <CalendarEvent
                key={event.record.id}
                event={event}
                config={config}
                onClick={() => onEventClick(event.record)}
                variant="day"
              />
            ))}
          </div>
        </div>
      )}

      {/* Time grid */}
      <div ref={scrollContainerRef} className="flex-1 overflow-auto">
        <div className="flex min-h-full">
          {/* Time labels */}
          <div className="w-20 flex-shrink-0 border-r border-slate-200 bg-white sticky left-0">
            {timeSlots.map((slot) => (
              <div
                key={slot.hour}
                className="border-b border-slate-100 text-xs text-gray-500 text-right pr-3 relative"
                style={{ height: HOUR_HEIGHT }}
              >
                <span className="absolute -top-2 right-3">{slot.label}</span>
              </div>
            ))}
          </div>

          {/* Event area */}
          <div
            className={`flex-1 relative ${isToday ? 'bg-primary/5' : ''}`}
            onClick={(e) => {
              // Calculate clicked time
              const rect = e.currentTarget.getBoundingClientRect();
              const y = e.clientY - rect.top + (scrollContainerRef.current?.scrollTop || 0);
              const hours = Math.floor(y / HOUR_HEIGHT) + START_HOUR;
              const minutes = Math.floor((y % HOUR_HEIGHT) / (HOUR_HEIGHT / 4)) * 15;

              const clickedDate = new Date(currentDate);
              clickedDate.setHours(hours, minutes, 0, 0);
              onAddRecord(clickedDate);
            }}
          >
            {/* Hour grid lines */}
            {timeSlots.map((slot) => (
              <div
                key={slot.hour}
                className="border-b border-slate-100"
                style={{ height: HOUR_HEIGHT }}
              >
                {/* Half-hour line */}
                <div
                  className="border-b border-slate-100/50"
                  style={{ height: HOUR_HEIGHT / 2 }}
                />
              </div>
            ))}

            {/* Current time indicator */}
            {isToday && currentTimeTop >= 0 && currentTimeTop <= TOTAL_HOURS * HOUR_HEIGHT && (
              <div
                className="absolute left-0 right-0 flex items-center z-20 pointer-events-none"
                style={{ top: currentTimeTop }}
              >
                <div className="w-3 h-3 bg-red-500 rounded-full -ml-1.5" />
                <div className="flex-1 h-0.5 bg-red-500" />
              </div>
            )}

            {/* Events */}
            {timedEvents.map((event) => {
              const style = getEventStyle(event);
              const position = eventPositions.get(event.record.id) || {
                left: '0%',
                width: '100%',
              };

              return (
                <div
                  key={event.record.id}
                  className="absolute"
                  style={{
                    top: style.top,
                    height: style.height,
                    left: `calc(${position.left} + 4px)`,
                    width: `calc(${position.width} - 8px)`,
                    zIndex: 10,
                  }}
                  onClick={(e) => e.stopPropagation()}
                >
                  <CalendarEvent
                    event={event}
                    config={config}
                    onClick={() => onEventClick(event.record)}
                    variant="day"
                    showFullDetails={true}
                  />
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
