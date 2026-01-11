import { useMemo } from 'react';
import type { CalendarEvent as CalendarEventType, CalendarConfig } from './types';
import { formatTime } from './types';

interface CalendarEventProps {
  event: CalendarEventType;
  config: CalendarConfig;
  onClick: () => void;
  variant?: 'month' | 'week' | 'day';
  isSpanStart?: boolean;
  isSpanEnd?: boolean;
  showFullDetails?: boolean;
}

export function CalendarEvent({
  event,
  config,
  onClick,
  variant = 'month',
  isSpanStart = true,
  isSpanEnd = true,
  showFullDetails = false,
}: CalendarEventProps) {
  // Generate lighter background color
  const bgColor = useMemo(() => {
    // Convert hex to RGB and create lighter version
    const hex = event.color.replace('#', '');
    const r = parseInt(hex.substring(0, 2), 16);
    const g = parseInt(hex.substring(2, 4), 16);
    const b = parseInt(hex.substring(4, 6), 16);

    // Create a light background (10% opacity effect)
    return `rgba(${r}, ${g}, ${b}, 0.15)`;
  }, [event.color]);

  // Format time display
  const timeDisplay = useMemo(() => {
    if (!config.showTime || event.isAllDay) return null;
    return formatTime(event.startDate);
  }, [config.showTime, event.isAllDay, event.startDate]);

  // Get base classes based on variant
  const baseClasses = useMemo(() => {
    switch (variant) {
      case 'day':
        return 'p-2 rounded-md';
      case 'week':
        return 'px-2 py-1 rounded-md text-xs';
      case 'month':
      default:
        return config.eventSize === 'comfortable'
          ? 'px-2 py-1.5 rounded text-xs'
          : 'px-1.5 py-0.5 rounded text-xs';
    }
  }, [variant, config.eventSize]);

  // Span styling for multi-day events
  const spanClasses = useMemo(() => {
    if (!event.isMultiDay) return '';

    let classes = '';
    if (!isSpanStart) classes += ' rounded-l-none -ml-1 pl-2';
    if (!isSpanEnd) classes += ' rounded-r-none -mr-1 pr-2';

    return classes;
  }, [event.isMultiDay, isSpanStart, isSpanEnd]);

  if (variant === 'day' || showFullDetails) {
    // Day view - full details
    return (
      <button
        onClick={onClick}
        className={`w-full text-left ${baseClasses} ${spanClasses} hover:brightness-95 transition-all cursor-pointer group`}
        style={{
          backgroundColor: bgColor,
          borderLeft: `3px solid ${event.color}`,
        }}
      >
        <div className="flex items-start gap-2">
          {/* Cover image */}
          {event.coverImage && (
            <img
              src={event.coverImage.thumbnail_url || event.coverImage.url}
              alt=""
              className="w-10 h-10 rounded object-cover flex-shrink-0"
              loading="lazy"
            />
          )}

          <div className="flex-1 min-w-0">
            {/* Title */}
            <div className="font-medium text-gray-900 truncate group-hover:text-primary">
              {event.title || 'Untitled'}
            </div>

            {/* Time */}
            {timeDisplay && (
              <div className="text-gray-500 text-xs mt-0.5">
                {timeDisplay}
                {event.endDate && !event.isAllDay && (
                  <> - {formatTime(event.endDate)}</>
                )}
              </div>
            )}

            {/* Display fields */}
            {event.displayValues.length > 0 && (
              <div className="mt-1 space-y-0.5">
                {event.displayValues.slice(0, 3).map(({ field, value }) => (
                  <div
                    key={field.id}
                    className="text-xs text-gray-500 truncate"
                  >
                    <span className="text-gray-400">{field.name}:</span>{' '}
                    {formatValue(value)}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </button>
    );
  }

  if (variant === 'week') {
    // Week view - medium details
    return (
      <button
        onClick={onClick}
        className={`w-full text-left ${baseClasses} ${spanClasses} hover:brightness-95 transition-all cursor-pointer overflow-hidden`}
        style={{
          backgroundColor: bgColor,
          borderLeft: `2px solid ${event.color}`,
        }}
      >
        <div className="flex items-center gap-1.5">
          {event.coverImage && (
            <img
              src={event.coverImage.thumbnail_url || event.coverImage.url}
              alt=""
              className="w-5 h-5 rounded object-cover flex-shrink-0"
              loading="lazy"
            />
          )}
          <div className="flex-1 min-w-0">
            <div className="font-medium text-gray-900 truncate text-xs">
              {event.title || 'Untitled'}
            </div>
            {timeDisplay && (
              <div className="text-gray-500 text-[10px]">{timeDisplay}</div>
            )}
          </div>
        </div>
      </button>
    );
  }

  // Month view - compact
  return (
    <button
      onClick={onClick}
      className={`w-full text-left ${baseClasses} ${spanClasses} hover:brightness-95 transition-all cursor-pointer overflow-hidden`}
      style={{
        backgroundColor: event.isMultiDay ? event.color : bgColor,
        borderLeft: event.isMultiDay ? 'none' : `2px solid ${event.color}`,
        color: event.isMultiDay ? 'white' : 'inherit',
      }}
    >
      <div className="flex items-center gap-1">
        {/* Show small color dot for non-multi-day events */}
        {!event.isMultiDay && config.eventSize === 'compact' && (
          <span
            className="w-1.5 h-1.5 rounded-full flex-shrink-0"
            style={{ backgroundColor: event.color }}
          />
        )}

        {event.coverImage && !event.isMultiDay && (
          <img
            src={event.coverImage.thumbnail_url || event.coverImage.url}
            alt=""
            className="w-4 h-4 rounded object-cover flex-shrink-0"
            loading="lazy"
          />
        )}

        <span className="truncate flex-1 text-gray-800" style={{
          color: event.isMultiDay ? 'white' : 'inherit',
        }}>
          {timeDisplay && (
            <span className="text-gray-500 mr-1" style={{
              color: event.isMultiDay ? 'rgba(255,255,255,0.8)' : 'inherit',
            }}>
              {timeDisplay}
            </span>
          )}
          {isSpanStart ? (event.title || 'Untitled') : ''}
        </span>
      </div>
    </button>
  );
}

// Helper to format cell values for display
function formatValue(value: unknown): string {
  if (value === null || value === undefined) return '';
  if (Array.isArray(value)) {
    if (value.length === 0) return '';
    if (typeof value[0] === 'object' && 'name' in value[0]) {
      return value.map((v) => v.name).join(', ');
    }
    return value.join(', ');
  }
  if (typeof value === 'boolean') return value ? 'Yes' : 'No';
  if (typeof value === 'object' && value !== null && 'name' in value) {
    return (value as { name: string }).name;
  }
  return String(value);
}
