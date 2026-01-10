import { useState, useMemo } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { TableRecord } from '../../../types';
import { RecordModal } from '../RecordModal';

export function CalendarView() {
  const { currentView, fields, records, createRecord } = useBaseStore();
  const [currentDate, setCurrentDate] = useState(new Date());
  const [expandedRecord, setExpandedRecord] = useState<TableRecord | null>(null);

  // Get the date field used for the calendar
  const dateField = useMemo(() => {
    const dateFieldId = currentView?.config?.dateField;
    if (dateFieldId) {
      return fields.find(f => f.id === dateFieldId);
    }
    return fields.find(f => f.type === 'date' || f.type === 'datetime');
  }, [fields, currentView?.config?.dateField]);

  // Get primary field for display
  const primaryField = fields.find(f => f.type === 'text') || fields[0];

  // Calculate calendar days
  const calendarDays = useMemo(() => {
    const year = currentDate.getFullYear();
    const month = currentDate.getMonth();

    const firstDay = new Date(year, month, 1);
    const lastDay = new Date(year, month + 1, 0);
    const startPadding = firstDay.getDay();
    const totalDays = lastDay.getDate();

    const days: { date: Date; isCurrentMonth: boolean; records: TableRecord[] }[] = [];

    // Previous month padding
    for (let i = startPadding - 1; i >= 0; i--) {
      const date = new Date(year, month, -i);
      days.push({ date, isCurrentMonth: false, records: [] });
    }

    // Current month
    for (let i = 1; i <= totalDays; i++) {
      const date = new Date(year, month, i);
      days.push({ date, isCurrentMonth: true, records: [] });
    }

    // Next month padding (fill to 42 cells = 6 rows)
    const remaining = 42 - days.length;
    for (let i = 1; i <= remaining; i++) {
      const date = new Date(year, month + 1, i);
      days.push({ date, isCurrentMonth: false, records: [] });
    }

    // Assign records to days
    if (dateField) {
      records.forEach((record) => {
        const dateValue = record.values[dateField.id] as string | undefined;
        if (dateValue) {
          const recordDate = new Date(dateValue);
          const dayIndex = days.findIndex(d =>
            d.date.getFullYear() === recordDate.getFullYear() &&
            d.date.getMonth() === recordDate.getMonth() &&
            d.date.getDate() === recordDate.getDate()
          );
          if (dayIndex >= 0) {
            days[dayIndex].records.push(record);
          }
        }
      });
    }

    return days;
  }, [currentDate, dateField, records]);

  const goToPreviousMonth = () => {
    setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth() - 1, 1));
  };

  const goToNextMonth = () => {
    setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth() + 1, 1));
  };

  const goToToday = () => {
    setCurrentDate(new Date());
  };

  const handleAddRecord = async (date: Date) => {
    if (!dateField) return;
    const dateStr = date.toISOString().split('T')[0];
    await createRecord({ [dateField.id]: dateStr });
  };

  const getRecordTitle = (record: TableRecord): string => {
    if (!primaryField) return 'Untitled';
    const value = record.values[primaryField.id];
    return value ? String(value) : 'Untitled';
  };

  const isToday = (date: Date): boolean => {
    const today = new Date();
    return date.getFullYear() === today.getFullYear() &&
           date.getMonth() === today.getMonth() &&
           date.getDate() === today.getDate();
  };

  if (!dateField) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500">
        <div className="text-center">
          <svg className="w-12 h-12 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <h3 className="text-lg font-medium text-gray-900 mb-1">No date field</h3>
          <p className="text-sm text-gray-500">Add a Date field to use Calendar view</p>
        </div>
      </div>
    );
  }

  const weekDays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
  const monthName = currentDate.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });

  return (
    <div className="flex-1 flex flex-col p-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-4">
          <h2 className="text-xl font-semibold text-gray-900">{monthName}</h2>
          <div className="flex gap-1">
            <button
              onClick={goToPreviousMonth}
              className="p-2 hover:bg-gray-100 rounded-md"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
            </button>
            <button
              onClick={goToNextMonth}
              className="p-2 hover:bg-gray-100 rounded-md"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </button>
          </div>
        </div>
        <button
          onClick={goToToday}
          className="btn btn-secondary"
        >
          Today
        </button>
      </div>

      {/* Calendar grid */}
      <div className="flex-1 border border-gray-200 rounded-lg overflow-hidden">
        {/* Week day headers */}
        <div className="grid grid-cols-7 bg-gray-50 border-b border-gray-200">
          {weekDays.map((day) => (
            <div key={day} className="p-2 text-center text-sm font-medium text-gray-600">
              {day}
            </div>
          ))}
        </div>

        {/* Calendar days */}
        <div className="grid grid-cols-7 flex-1">
          {calendarDays.map((day, index) => (
            <div
              key={index}
              className={`min-h-[100px] border-b border-r border-gray-200 p-1 ${
                !day.isCurrentMonth ? 'bg-gray-50' : ''
              }`}
            >
              <div className="flex items-center justify-between mb-1">
                <span
                  className={`w-7 h-7 flex items-center justify-center text-sm rounded-full ${
                    isToday(day.date)
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
                    onClick={() => handleAddRecord(day.date)}
                    className="w-5 h-5 flex items-center justify-center text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded opacity-0 hover:opacity-100"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                    </svg>
                  </button>
                )}
              </div>

              {/* Records for this day */}
              <div className="space-y-1">
                {day.records.slice(0, 3).map((record) => (
                  <button
                    key={record.id}
                    onClick={() => setExpandedRecord(record)}
                    className="w-full text-left text-xs p-1 bg-primary-100 text-primary-700 rounded truncate hover:bg-primary-200"
                  >
                    {getRecordTitle(record)}
                  </button>
                ))}
                {day.records.length > 3 && (
                  <button className="text-xs text-gray-500 hover:text-gray-700">
                    +{day.records.length - 3} more
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Record modal */}
      {expandedRecord && (
        <RecordModal
          record={expandedRecord}
          onClose={() => setExpandedRecord(null)}
        />
      )}
    </div>
  );
}
