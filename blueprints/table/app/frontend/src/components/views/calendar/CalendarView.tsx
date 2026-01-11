import { useState, useMemo, useCallback, useEffect } from 'react';
import {
  DndContext,
  DragOverlay,
  useSensor,
  useSensors,
  PointerSensor,
  type DragStartEvent,
  type DragEndEvent,
} from '@dnd-kit/core';
import { useBaseStore } from '../../../stores/baseStore';
import type { TableRecord, Field, Attachment } from '../../../types';
import { RecordSidebar } from '../RecordSidebar';
import { CalendarMonthView } from './CalendarMonthView';
import { CalendarWeekView } from './CalendarWeekView';
import { CalendarDayView } from './CalendarDayView';
import { CalendarSettings } from './CalendarSettings';
import { CalendarEvent as CalendarEventComponent } from './CalendarEvent';
import {
  type CalendarConfig,
  type CalendarEvent,
  type CalendarViewMode,
  DEFAULT_CALENDAR_CONFIG,
  getDefaultEventColor,
  addDays,
  startOfWeek,
  endOfWeek,
  formatWeekRange,
  formatDayHeader,
  isSameDay,
} from './types';

export function CalendarView() {
  const {
    currentView,
    fields,
    createRecord,
    updateCellValue,
    getSortedRecords,
    updateViewConfig,
  } = useBaseStore();

  // Get filtered and sorted records
  const records = getSortedRecords();

  // State
  const [currentDate, setCurrentDate] = useState(new Date());
  const [expandedRecord, setExpandedRecord] = useState<TableRecord | null>(null);
  const [showSettings, setShowSettings] = useState(false);
  const [draggedEvent, setDraggedEvent] = useState<CalendarEvent | null>(null);

  // Load config from view
  const config = useMemo((): CalendarConfig => {
    if (!currentView?.config) return DEFAULT_CALENDAR_CONFIG;
    const viewConfig =
      typeof currentView.config === 'string'
        ? JSON.parse(currentView.config)
        : currentView.config;
    return {
      ...DEFAULT_CALENDAR_CONFIG,
      dateField: viewConfig.dateField || null,
      endDateField: viewConfig.endDateField || null,
      viewMode: viewConfig.viewMode || 'month',
      weekStart: viewConfig.weekStart ?? 0,
      showWeekends: viewConfig.showWeekends ?? true,
      colorField: viewConfig.colorField || null,
      coverField: viewConfig.coverField || null,
      displayFields: viewConfig.displayFields || [],
      eventSize: viewConfig.eventSize || 'compact',
      showTime: viewConfig.showTime ?? false,
    };
  }, [currentView?.config]);

  // Update config handler
  const handleConfigChange = useCallback(
    (updates: Partial<CalendarConfig>) => {
      const newConfig = { ...config, ...updates };
      const viewConfig =
        currentView?.config && typeof currentView.config === 'object'
          ? { ...currentView.config }
          : {};
      updateViewConfig({
        ...viewConfig,
        dateField: newConfig.dateField,
        endDateField: newConfig.endDateField,
        viewMode: newConfig.viewMode,
        weekStart: newConfig.weekStart,
        showWeekends: newConfig.showWeekends,
        colorField: newConfig.colorField,
        coverField: newConfig.coverField,
        displayFields: newConfig.displayFields,
        eventSize: newConfig.eventSize,
        showTime: newConfig.showTime,
      });
    },
    [config, currentView?.config, updateViewConfig]
  );

  // Get date field
  const dateField = useMemo(() => {
    if (config.dateField) {
      return fields.find((f) => f.id === config.dateField);
    }
    // Auto-detect first date/datetime field
    return fields.find((f) => f.type === 'date' || f.type === 'datetime');
  }, [fields, config.dateField]);

  // Get end date field
  const endDateField = useMemo(() => {
    if (config.endDateField) {
      return fields.find((f) => f.id === config.endDateField);
    }
    return undefined;
  }, [fields, config.endDateField]);

  // Auto-set dateField in config if not set
  useEffect(() => {
    if (!config.dateField && dateField) {
      handleConfigChange({ dateField: dateField.id });
    }
  }, [config.dateField, dateField, handleConfigChange]);

  // Get color field
  const colorField = useMemo(() => {
    if (config.colorField) {
      return fields.find((f) => f.id === config.colorField);
    }
    return undefined;
  }, [fields, config.colorField]);

  // Get cover field
  const coverField = useMemo(() => {
    if (config.coverField) {
      return fields.find((f) => f.id === config.coverField);
    }
    return undefined;
  }, [fields, config.coverField]);

  // Get display fields
  const displayFields = useMemo(() => {
    return config.displayFields
      .map((id) => fields.find((f) => f.id === id))
      .filter((f): f is Field => Boolean(f));
  }, [fields, config.displayFields]);

  // Get primary field for display
  const primaryField = useMemo(() => {
    return (
      fields.find((f) => f.is_primary) ||
      fields.find((f) => f.type === 'text') ||
      fields[0]
    );
  }, [fields]);

  // Build calendar events from records
  const events = useMemo((): CalendarEvent[] => {
    if (!dateField) return [];

    return records
      .filter((record) => {
        const dateValue = record.values[dateField.id];
        return dateValue !== null && dateValue !== undefined;
      })
      .map((record) => {
        const startDateValue = record.values[dateField.id] as string;
        const startDate = new Date(startDateValue);

        let endDate: Date | null = null;
        let isMultiDay = false;

        if (endDateField) {
          const endDateValue = record.values[endDateField.id] as string;
          if (endDateValue) {
            endDate = new Date(endDateValue);
            isMultiDay = !isSameDay(startDate, endDate);
          }
        }

        // Determine if all-day (no time component)
        const isAllDay =
          dateField.type === 'date' ||
          (startDate.getHours() === 0 &&
            startDate.getMinutes() === 0 &&
            startDate.getSeconds() === 0);

        // Get event color
        let color = getDefaultEventColor(record.id);
        if (colorField) {
          const colorValue = record.values[colorField.id] as string;
          if (colorValue) {
            const choice = colorField.options?.choices?.find(
              (c) => c.id === colorValue
            );
            if (choice) {
              color = choice.color;
            }
          }
        }

        // Get cover image
        let coverImage: Attachment | undefined;
        if (coverField) {
          const attachments = record.values[coverField.id] as Attachment[] | undefined;
          if (attachments && attachments.length > 0) {
            // Find first image
            coverImage = attachments.find((a) =>
              a.mime_type?.startsWith('image/')
            );
          }
        }

        // Get display values
        const displayValues = displayFields.map((field) => ({
          field,
          value: record.values[field.id],
        }));

        // Get title
        const title = primaryField
          ? String(record.values[primaryField.id] || 'Untitled')
          : 'Untitled';

        return {
          record,
          title,
          startDate,
          endDate,
          isAllDay,
          isMultiDay,
          color,
          coverImage,
          displayValues,
        };
      });
  }, [
    records,
    dateField,
    endDateField,
    colorField,
    coverField,
    displayFields,
    primaryField,
  ]);

  // Navigation handlers
  const goToPrevious = useCallback(() => {
    switch (config.viewMode) {
      case 'month':
        setCurrentDate(
          new Date(currentDate.getFullYear(), currentDate.getMonth() - 1, 1)
        );
        break;
      case 'week':
        setCurrentDate(addDays(currentDate, -7));
        break;
      case 'day':
        setCurrentDate(addDays(currentDate, -1));
        break;
    }
  }, [config.viewMode, currentDate]);

  const goToNext = useCallback(() => {
    switch (config.viewMode) {
      case 'month':
        setCurrentDate(
          new Date(currentDate.getFullYear(), currentDate.getMonth() + 1, 1)
        );
        break;
      case 'week':
        setCurrentDate(addDays(currentDate, 7));
        break;
      case 'day':
        setCurrentDate(addDays(currentDate, 1));
        break;
    }
  }, [config.viewMode, currentDate]);

  const goToToday = useCallback(() => {
    setCurrentDate(new Date());
  }, []);

  // Format header title based on view mode
  const headerTitle = useMemo(() => {
    switch (config.viewMode) {
      case 'month':
        return currentDate.toLocaleDateString('en-US', {
          month: 'long',
          year: 'numeric',
        });
      case 'week':
        const weekStartDate = startOfWeek(currentDate, config.weekStart);
        const weekEndDate = endOfWeek(currentDate, config.weekStart);
        return formatWeekRange(weekStartDate, weekEndDate);
      case 'day':
        return formatDayHeader(currentDate);
    }
  }, [config.viewMode, config.weekStart, currentDate]);

  // Handle view mode change
  const handleViewModeChange = useCallback(
    (mode: CalendarViewMode) => {
      handleConfigChange({ viewMode: mode });
    },
    [handleConfigChange]
  );

  // Add record handler
  const handleAddRecord = useCallback(
    async (date: Date) => {
      if (!dateField) return;

      let dateStr: string;
      if (dateField.type === 'datetime') {
        dateStr = date.toISOString();
      } else {
        dateStr = date.toISOString().split('T')[0];
      }

      const record = await createRecord({ [dateField.id]: dateStr });
      setExpandedRecord(record);
    },
    [dateField, createRecord]
  );

  // Event click handler
  const handleEventClick = useCallback((record: TableRecord) => {
    setExpandedRecord(record);
  }, []);

  // DnD sensors
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 5,
      },
    })
  );

  // DnD handlers
  const handleDragStart = useCallback(
    (event: DragStartEvent) => {
      const eventId = event.active.id as string;
      const calendarEvent = events.find((e) => e.record.id === eventId);
      if (calendarEvent) {
        setDraggedEvent(calendarEvent);
      }
    },
    [events]
  );

  const handleDragEnd = useCallback(
    async (event: DragEndEvent) => {
      setDraggedEvent(null);

      const { active, over } = event;
      if (!over || !dateField) return;

      const recordId = active.id as string;
      const targetDate = over.data.current?.date as Date | undefined;

      if (targetDate) {
        let dateStr: string;
        if (dateField.type === 'datetime') {
          dateStr = targetDate.toISOString();
        } else {
          dateStr = targetDate.toISOString().split('T')[0];
        }
        await updateCellValue(recordId, dateField.id, dateStr);
      }
    },
    [dateField, updateCellValue]
  );

  // No date field - show empty state
  if (!dateField) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500">
        <div className="text-center max-w-md">
          <svg
            className="w-16 h-16 mx-auto mb-4 text-gray-300"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
            />
          </svg>
          <h3 className="text-xl font-semibold text-gray-900 mb-2">
            No date field
          </h3>
          <p className="text-sm text-gray-500 mb-4">
            Calendar view requires a Date or DateTime field to display records.
          </p>
          <button
            onClick={() => setShowSettings(true)}
            className="px-4 py-2 bg-primary text-white rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
          >
            Configure Calendar
          </button>
        </div>

        {showSettings && (
          <CalendarSettings
            config={config}
            fields={fields}
            onConfigChange={handleConfigChange}
            onClose={() => setShowSettings(false)}
          />
        )}
      </div>
    );
  }

  return (
    <DndContext
      sensors={sensors}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
    >
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Toolbar */}
        <div className="flex items-center justify-between px-4 py-2 border-b border-slate-200 bg-white">
          {/* Left: Navigation */}
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-semibold text-gray-900 min-w-[200px]">
              {headerTitle}
            </h2>
            <div className="flex gap-1">
              <button
                onClick={goToPrevious}
                className="p-2 hover:bg-slate-100 rounded-md"
                aria-label="Previous"
              >
                <svg
                  className="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15 19l-7-7 7-7"
                  />
                </svg>
              </button>
              <button
                onClick={goToNext}
                className="p-2 hover:bg-slate-100 rounded-md"
                aria-label="Next"
              >
                <svg
                  className="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 5l7 7-7 7"
                  />
                </svg>
              </button>
            </div>
            <button
              onClick={goToToday}
              className="px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-slate-100 rounded-lg transition-colors"
            >
              Today
            </button>
          </div>

          {/* Center: View mode toggle */}
          <div className="flex items-center gap-1 bg-slate-100 rounded-lg p-0.5">
            <button
              onClick={() => handleViewModeChange('month')}
              className={`px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
                config.viewMode === 'month'
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              Month
            </button>
            <button
              onClick={() => handleViewModeChange('week')}
              className={`px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
                config.viewMode === 'week'
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              Week
            </button>
            <button
              onClick={() => handleViewModeChange('day')}
              className={`px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
                config.viewMode === 'day'
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              Day
            </button>
          </div>

          {/* Right: Settings */}
          <div className="flex items-center gap-2 relative">
            <span className="text-xs text-slate-400">
              {events.length} event{events.length !== 1 ? 's' : ''}
            </span>
            <button
              onClick={() => setShowSettings(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-gray-600 hover:bg-slate-100 rounded-lg transition-colors"
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
                  d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                />
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                />
              </svg>
              Settings
            </button>

            {showSettings && (
              <CalendarSettings
                config={config}
                fields={fields}
                onConfigChange={handleConfigChange}
                onClose={() => setShowSettings(false)}
              />
            )}
          </div>
        </div>

        {/* Calendar content */}
        <div className="flex-1 overflow-hidden border border-slate-200 rounded-xl m-4 bg-white">
          {config.viewMode === 'month' && (
            <CalendarMonthView
              currentDate={currentDate}
              events={events}
              config={config}
              onEventClick={handleEventClick}
              onAddRecord={handleAddRecord}
            />
          )}
          {config.viewMode === 'week' && (
            <CalendarWeekView
              currentDate={currentDate}
              events={events}
              config={config}
              onEventClick={handleEventClick}
              onAddRecord={handleAddRecord}
            />
          )}
          {config.viewMode === 'day' && (
            <CalendarDayView
              currentDate={currentDate}
              events={events}
              config={config}
              onEventClick={handleEventClick}
              onAddRecord={handleAddRecord}
            />
          )}
        </div>

        {/* Drag overlay */}
        <DragOverlay>
          {draggedEvent && (
            <div className="opacity-80 shadow-lg">
              <CalendarEventComponent
                event={draggedEvent}
                config={config}
                onClick={() => {}}
                variant="month"
              />
            </div>
          )}
        </DragOverlay>

        {/* Record sidebar */}
        {expandedRecord && (
          <RecordSidebar
            record={expandedRecord}
            onClose={() => setExpandedRecord(null)}
            onNavigate={(direction) => {
              const currentIndex = records.findIndex(r => r.id === expandedRecord.id);
              const newIndex = direction === 'prev' ? currentIndex - 1 : currentIndex + 1;
              if (newIndex >= 0 && newIndex < records.length) {
                setExpandedRecord(records[newIndex]);
              }
            }}
            hasPrev={records.findIndex(r => r.id === expandedRecord.id) > 0}
            hasNext={records.findIndex(r => r.id === expandedRecord.id) < records.length - 1}
            position={records.findIndex(r => r.id === expandedRecord.id) + 1}
            total={records.length}
          />
        )}
      </div>
    </DndContext>
  );
}
