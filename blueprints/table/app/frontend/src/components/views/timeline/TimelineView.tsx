import { useState, useMemo } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { TableRecord, Field } from '../../../types';
import { RecordModal } from '../RecordModal';

type TimeScale = 'day' | 'week' | 'month' | 'quarter' | 'year';

export function TimelineView() {
  const { currentView, fields, getSortedRecords, createRecord } = useBaseStore();
  const records = getSortedRecords();
  const [timeScale, setTimeScale] = useState<TimeScale>('month');
  const [expandedRecord, setExpandedRecord] = useState<TableRecord | null>(null);
  const [viewStart, setViewStart] = useState(() => {
    const now = new Date();
    return new Date(now.getFullYear(), now.getMonth(), 1);
  });

  // Get start and end date fields
  const dateFields = useMemo(() => {
    return fields.filter(f => ['date', 'datetime'].includes(f.type));
  }, [fields]);

  const startField = useMemo(() => {
    const startFieldId = currentView?.settings?.start_field_id || currentView?.config?.dateField;
    if (startFieldId) {
      return fields.find(f => f.id === startFieldId);
    }
    return dateFields[0];
  }, [fields, dateFields, currentView]);

  const endField = useMemo(() => {
    const endFieldId = currentView?.settings?.end_field_id;
    if (endFieldId) {
      return fields.find(f => f.id === endFieldId);
    }
    return dateFields[1] || dateFields[0];
  }, [fields, dateFields, currentView]);

  // Get primary field for display
  const primaryField = fields.find(f => f.type === 'text') || fields[0];

  // Calculate visible time range
  const { viewEnd, columnWidth, columns } = useMemo(() => {
    let end: Date;
    let colWidth: number;
    let cols: { date: Date; label: string }[] = [];

    const start = new Date(viewStart);

    switch (timeScale) {
      case 'day':
        end = new Date(start);
        end.setDate(end.getDate() + 30);
        colWidth = 40;
        for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
          cols.push({
            date: new Date(d),
            label: d.getDate().toString(),
          });
        }
        break;
      case 'week':
        end = new Date(start);
        end.setDate(end.getDate() + 84); // 12 weeks
        colWidth = 80;
        for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 7)) {
          cols.push({
            date: new Date(d),
            label: `W${Math.ceil(d.getDate() / 7)}`,
          });
        }
        break;
      case 'month':
        end = new Date(start);
        end.setMonth(end.getMonth() + 12);
        colWidth = 100;
        for (let d = new Date(start); d <= end; d.setMonth(d.getMonth() + 1)) {
          cols.push({
            date: new Date(d),
            label: d.toLocaleDateString('en-US', { month: 'short' }),
          });
        }
        break;
      case 'quarter':
        end = new Date(start);
        end.setMonth(end.getMonth() + 24);
        colWidth = 150;
        for (let d = new Date(start); d <= end; d.setMonth(d.getMonth() + 3)) {
          const q = Math.floor(d.getMonth() / 3) + 1;
          cols.push({
            date: new Date(d),
            label: `Q${q} ${d.getFullYear()}`,
          });
        }
        break;
      case 'year':
        end = new Date(start);
        end.setFullYear(end.getFullYear() + 5);
        colWidth = 200;
        for (let d = new Date(start); d <= end; d.setFullYear(d.getFullYear() + 1)) {
          cols.push({
            date: new Date(d),
            label: d.getFullYear().toString(),
          });
        }
        break;
    }

    return { viewEnd: end, columnWidth: colWidth, columns: cols };
  }, [viewStart, timeScale]);

  // Calculate bar positions for records
  const recordBars = useMemo(() => {
    if (!startField) return [];

    const totalWidth = columns.length * columnWidth;
    const msPerPixel = (viewEnd.getTime() - viewStart.getTime()) / totalWidth;

    return records
      .filter(record => {
        const startDate = record.values[startField.id];
        return startDate !== null && startDate !== undefined;
      })
      .map(record => {
        const startDate = new Date(record.values[startField.id] as string);
        const endDate = endField && record.values[endField.id]
          ? new Date(record.values[endField.id] as string)
          : new Date(startDate.getTime() + 86400000); // Default 1 day duration

        const startX = Math.max(0, (startDate.getTime() - viewStart.getTime()) / msPerPixel);
        const endX = Math.min(totalWidth, (endDate.getTime() - viewStart.getTime()) / msPerPixel);
        const width = Math.max(20, endX - startX);

        return {
          record,
          left: startX,
          width,
          title: primaryField ? (record.values[primaryField.id] as string) || 'Untitled' : 'Untitled',
          color: getRecordColor(record, fields),
        };
      });
  }, [records, startField, endField, primaryField, fields, viewStart, viewEnd, columns.length, columnWidth]);

  const navigateTimeline = (direction: 'prev' | 'next') => {
    const newStart = new Date(viewStart);
    switch (timeScale) {
      case 'day':
        newStart.setDate(newStart.getDate() + (direction === 'next' ? 7 : -7));
        break;
      case 'week':
        newStart.setDate(newStart.getDate() + (direction === 'next' ? 28 : -28));
        break;
      case 'month':
        newStart.setMonth(newStart.getMonth() + (direction === 'next' ? 3 : -3));
        break;
      case 'quarter':
        newStart.setMonth(newStart.getMonth() + (direction === 'next' ? 6 : -6));
        break;
      case 'year':
        newStart.setFullYear(newStart.getFullYear() + (direction === 'next' ? 1 : -1));
        break;
    }
    setViewStart(newStart);
  };

  const goToToday = () => {
    const now = new Date();
    setViewStart(new Date(now.getFullYear(), now.getMonth(), 1));
  };

  const handleAddRecord = async (date: Date) => {
    if (!startField) return;
    await createRecord({ [startField.id]: date.toISOString().split('T')[0] });
  };

  if (!startField) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500">
        <div className="text-center">
          <svg className="w-12 h-12 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <h3 className="text-lg font-medium text-gray-900 mb-1">No date field</h3>
          <p className="text-sm text-gray-500">Add a Date field to use Timeline view</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Header controls */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200 bg-white">
        <div className="flex items-center gap-4">
          <h2 className="text-lg font-semibold text-gray-900">
            {viewStart.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}
          </h2>
          <div className="flex gap-1">
            <button
              onClick={() => navigateTimeline('prev')}
              className="p-2 hover:bg-gray-100 rounded-md"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
            </button>
            <button
              onClick={() => navigateTimeline('next')}
              className="p-2 hover:bg-gray-100 rounded-md"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </button>
          </div>
          <button onClick={goToToday} className="btn btn-secondary py-1 px-3 text-sm">
            Today
          </button>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-600">Scale:</span>
          <select
            value={timeScale}
            onChange={(e) => setTimeScale(e.target.value as TimeScale)}
            className="input py-1 w-32"
          >
            <option value="day">Day</option>
            <option value="week">Week</option>
            <option value="month">Month</option>
            <option value="quarter">Quarter</option>
            <option value="year">Year</option>
          </select>
        </div>
      </div>

      {/* Timeline grid */}
      <div className="flex-1 overflow-auto">
        <div className="min-w-max">
          {/* Column headers */}
          <div className="flex border-b border-gray-200 bg-gray-50 sticky top-0 z-10">
            <div className="w-48 flex-shrink-0 p-2 border-r border-gray-200 font-medium text-sm text-gray-700">
              Records
            </div>
            <div className="flex">
              {columns.map((col, i) => (
                <div
                  key={i}
                  className="border-r border-gray-200 p-2 text-center text-sm text-gray-600"
                  style={{ width: columnWidth }}
                >
                  {col.label}
                </div>
              ))}
            </div>
          </div>

          {/* Record rows */}
          <div className="relative">
            {recordBars.map((bar) => (
              <div
                key={bar.record.id}
                className="flex border-b border-gray-100 hover:bg-gray-50"
                style={{ height: 40 }}
              >
                {/* Record name */}
                <div className="w-48 flex-shrink-0 px-2 flex items-center border-r border-gray-200">
                  <button
                    onClick={() => setExpandedRecord(bar.record)}
                    className="text-sm text-gray-900 truncate hover:text-primary"
                  >
                    {bar.title}
                  </button>
                </div>

                {/* Timeline bar area */}
                <div className="relative flex-1" style={{ width: columns.length * columnWidth }}>
                  {/* Background grid lines */}
                  <div className="absolute inset-0 flex">
                    {columns.map((_, i) => (
                      <div
                        key={i}
                        className="border-r border-gray-100"
                        style={{ width: columnWidth }}
                      />
                    ))}
                  </div>

                  {/* Bar */}
                  <div
                    className="absolute top-1 h-6 rounded cursor-pointer hover:opacity-80 flex items-center px-2 text-xs text-white truncate"
                    style={{
                      left: bar.left,
                      width: bar.width,
                      backgroundColor: bar.color,
                    }}
                    onClick={() => setExpandedRecord(bar.record)}
                  >
                    {bar.width > 60 && bar.title}
                  </div>
                </div>
              </div>
            ))}

            {/* Add record row */}
            <div className="flex border-b border-gray-100 hover:bg-gray-50" style={{ height: 40 }}>
              <div className="w-48 flex-shrink-0 px-2 flex items-center border-r border-gray-200">
                <button
                  onClick={() => handleAddRecord(viewStart)}
                  className="text-sm text-gray-500 hover:text-gray-700 flex items-center gap-1"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                  Add record
                </button>
              </div>
              <div className="flex-1" style={{ width: columns.length * columnWidth }} />
            </div>
          </div>
        </div>
      </div>

      {/* Empty state */}
      {recordBars.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          <div className="text-center">
            <p className="text-gray-500">No records with dates to display</p>
            <button
              onClick={() => handleAddRecord(new Date())}
              className="mt-2 btn btn-primary pointer-events-auto"
            >
              Add record
            </button>
          </div>
        </div>
      )}

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

// Get a color for a record based on select field value
function getRecordColor(record: TableRecord, fields: Field[]): string {
  // Find a single_select field with a value
  const selectField = fields.find(f => f.type === 'single_select' && record.values[f.id]);
  if (selectField) {
    const value = record.values[selectField.id] as string;
    const choice = selectField.options?.choices?.find(c => c.id === value);
    if (choice) return choice.color;
  }

  // Default colors based on record position
  const colors = ['#2D7FF9', '#20C933', '#FCAE40', '#F82B60', '#8B46FF', '#0D9488'];
  const hash = record.id.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
  return colors[hash % colors.length];
}
