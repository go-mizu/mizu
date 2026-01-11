import { useState, useMemo, useCallback, useRef } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { TableRecord, Field } from '../../../types';
import { RecordSidebar } from '../RecordSidebar';
import { TimelineSettings } from './TimelineSettings';
import { RecordBar } from './RecordBar';
import { RecordPreview } from './RecordPreview';
import { TodayMarker } from './TodayMarker';

type TimeScale = 'day' | 'week' | 'month' | 'quarter' | 'year';

interface RecordBarData {
  record: TableRecord;
  left: number;
  width: number;
  title: string;
  color: string;
  row: number;
  stackIndex: number;
  groupKey: string;
}

interface GroupData {
  key: string;
  label: string;
  records: RecordBarData[];
  color?: string;
  isCollapsed: boolean;
  rowCount: number;
}

export function TimelineView() {
  const { currentView, fields, getSortedRecords, createRecord, updateRecord } = useBaseStore();
  const records = getSortedRecords();

  const [timeScale, setTimeScale] = useState<TimeScale>(
    (currentView?.settings?.time_scale as TimeScale) || 'month'
  );
  const [expandedRecord, setExpandedRecord] = useState<TableRecord | null>(null);
  const [showSettings, setShowSettings] = useState(false);
  const [hoveredRecord, setHoveredRecord] = useState<{ record: TableRecord; position: { x: number; y: number } } | null>(null);
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(
    new Set(currentView?.settings?.collapsed_groups || [])
  );

  const hoverTimeoutRef = useRef<number | null>(null);
  const gridRef = useRef<HTMLDivElement>(null);

  const [viewStart, setViewStart] = useState(() => {
    const now = new Date();
    return new Date(now.getFullYear(), now.getMonth(), 1);
  });

  // Get configuration from view settings
  const startFieldId = currentView?.settings?.start_field_id || currentView?.config?.dateField;
  const endFieldId = currentView?.settings?.end_field_id || currentView?.config?.endDateField;
  const groupFieldId = currentView?.settings?.group_field_id;
  const colorFieldId = currentView?.settings?.color_field_id;
  const labelFieldIds = currentView?.settings?.label_field_ids;
  const showTodayMarker = currentView?.settings?.show_today_marker ?? true;
  // Dependencies will be implemented in a future update
  // const showDependencies = currentView?.settings?.show_dependencies ?? true;
  const rowHeightSetting = currentView?.settings?.timeline_row_height || 'medium';

  const rowHeight = rowHeightSetting === 'compact' ? 32 : rowHeightSetting === 'tall' ? 48 : 40;

  // Get date fields
  const dateFields = useMemo(() => {
    return fields.filter(f => ['date', 'datetime'].includes(f.type));
  }, [fields]);

  const startField = useMemo(() => {
    if (startFieldId) {
      return fields.find(f => f.id === startFieldId);
    }
    return dateFields[0];
  }, [fields, dateFields, startFieldId]);

  const endField = useMemo(() => {
    if (endFieldId) {
      return fields.find(f => f.id === endFieldId);
    }
    return dateFields[1] || dateFields[0];
  }, [fields, dateFields, endFieldId]);

  // Get grouping field
  const groupField = useMemo(() => {
    if (groupFieldId) {
      return fields.find(f => f.id === groupFieldId);
    }
    return null;
  }, [fields, groupFieldId]);

  // Get color field
  const colorField = useMemo(() => {
    if (colorFieldId) {
      return fields.find(f => f.id === colorFieldId);
    }
    // Default to first single_select field
    return fields.find(f => f.type === 'single_select');
  }, [fields, colorFieldId]);

  // Get primary field for display
  const primaryField = fields.find(f => f.type === 'text' || (f.type as string) === 'single_line_text') || fields[0];

  // Calculate visible time range and columns
  const { columnWidth, columns, totalWidth, msPerPixel } = useMemo(() => {
    let end: Date;
    let colWidth: number;
    const cols: { date: Date; label: string; subLabel?: string }[] = [];

    const start = new Date(viewStart);

    switch (timeScale) {
      case 'day':
        end = new Date(start);
        end.setDate(end.getDate() + 60);
        colWidth = 40;
        for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
          cols.push({
            date: new Date(d),
            label: d.getDate().toString(),
            subLabel: d.getDate() === 1 ? d.toLocaleDateString('en-US', { month: 'short' }) : undefined,
          });
        }
        break;
      case 'week':
        end = new Date(start);
        end.setDate(end.getDate() + 168); // 24 weeks
        colWidth = 60;
        for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 7)) {
          const weekNum = getWeekNumber(d);
          cols.push({
            date: new Date(d),
            label: `W${weekNum}`,
            subLabel: d.getDate() <= 7 ? d.toLocaleDateString('en-US', { month: 'short' }) : undefined,
          });
        }
        break;
      case 'month':
        end = new Date(start);
        end.setMonth(end.getMonth() + 18);
        colWidth = 80;
        for (let d = new Date(start); d <= end; d.setMonth(d.getMonth() + 1)) {
          cols.push({
            date: new Date(d),
            label: d.toLocaleDateString('en-US', { month: 'short' }),
            subLabel: d.getMonth() === 0 ? d.getFullYear().toString() : undefined,
          });
        }
        break;
      case 'quarter':
        end = new Date(start);
        end.setMonth(end.getMonth() + 36);
        colWidth = 120;
        for (let d = new Date(start); d <= end; d.setMonth(d.getMonth() + 3)) {
          const q = Math.floor(d.getMonth() / 3) + 1;
          cols.push({
            date: new Date(d),
            label: `Q${q}`,
            subLabel: q === 1 ? d.getFullYear().toString() : undefined,
          });
        }
        break;
      case 'year':
        end = new Date(start);
        end.setFullYear(end.getFullYear() + 8);
        colWidth = 150;
        for (let d = new Date(start); d <= end; d.setFullYear(d.getFullYear() + 1)) {
          cols.push({
            date: new Date(d),
            label: d.getFullYear().toString(),
          });
        }
        break;
    }

    const total = cols.length * colWidth;
    const msPerPx = (end.getTime() - start.getTime()) / total;

    return { columnWidth: colWidth, columns: cols, totalWidth: total, msPerPixel: msPerPx };
  }, [viewStart, timeScale]);

  // Calculate today marker position
  const todayPosition = useMemo(() => {
    const today = new Date();
    const position = (today.getTime() - viewStart.getTime()) / msPerPixel;
    return position;
  }, [viewStart, msPerPixel]);

  // Group records and calculate bar positions
  const { groups, totalHeight } = useMemo(() => {
    if (!startField) return { groups: [], totalHeight: 0 };

    const recordsWithDates = records.filter(record => {
      const startDate = record.values[startField.id];
      return startDate !== null && startDate !== undefined;
    });

    // Group records
    const groupedRecords = new Map<string, TableRecord[]>();

    if (groupField) {
      recordsWithDates.forEach(record => {
        const groupValue = record.values[groupField.id];
        let groupKey = '(Empty)';

        if (groupValue !== null && groupValue !== undefined && groupValue !== '') {
          if (groupField.type === 'single_select') {
            const choice = groupField.options?.choices?.find(c => c.id === groupValue);
            groupKey = choice?.name || String(groupValue);
          } else {
            groupKey = String(groupValue);
          }
        }

        if (!groupedRecords.has(groupKey)) {
          groupedRecords.set(groupKey, []);
        }
        groupedRecords.get(groupKey)!.push(record);
      });
    } else {
      groupedRecords.set('All Records', recordsWithDates);
    }

    // Calculate bar positions for each group
    const processedGroups: GroupData[] = [];
    let currentRow = 0;

    // Sort groups - put '(Empty)' at the end
    const sortedGroupKeys = Array.from(groupedRecords.keys()).sort((a, b) => {
      if (a === '(Empty)') return 1;
      if (b === '(Empty)') return -1;
      return a.localeCompare(b);
    });

    sortedGroupKeys.forEach(groupKey => {
      const groupRecords = groupedRecords.get(groupKey)!;
      const isCollapsed = collapsedGroups.has(groupKey);

      // Calculate bar positions
      const bars: RecordBarData[] = [];
      const occupiedSlots: { left: number; right: number; row: number }[] = [];

      groupRecords.forEach(record => {
        const startDate = new Date(record.values[startField.id] as string);
        const endDate = endField && record.values[endField.id]
          ? new Date(record.values[endField.id] as string)
          : new Date(startDate.getTime() + 86400000);

        const startX = Math.max(0, (startDate.getTime() - viewStart.getTime()) / msPerPixel);
        const endX = Math.min(totalWidth, (endDate.getTime() - viewStart.getTime()) / msPerPixel);
        const width = Math.max(30, endX - startX);

        // Find a row that doesn't overlap
        let stackIndex = 0;
        for (let i = 0; i <= occupiedSlots.length; i++) {
          const overlapping = occupiedSlots.filter(
            slot => slot.row === i && !(startX >= slot.right || startX + width <= slot.left)
          );
          if (overlapping.length === 0) {
            stackIndex = i;
            break;
          }
        }

        occupiedSlots.push({ left: startX, right: startX + width, row: stackIndex });

        bars.push({
          record,
          left: startX,
          width,
          title: primaryField ? (record.values[primaryField.id] as string) || 'Untitled' : 'Untitled',
          color: getRecordColor(record, colorField || fields.find(f => f.type === 'single_select'), fields),
          row: currentRow,
          stackIndex,
          groupKey,
        });
      });

      // Calculate row count based on stacking
      const maxStack = bars.length > 0 ? Math.max(...bars.map(b => b.stackIndex)) + 1 : 1;
      const rowCount = isCollapsed ? 0 : maxStack;

      // Get group color from single_select field if applicable
      let groupColor: string | undefined;
      if (groupField?.type === 'single_select') {
        const choice = groupField.options?.choices?.find(c => c.name === groupKey);
        groupColor = choice?.color;
      }

      processedGroups.push({
        key: groupKey,
        label: groupKey,
        records: bars,
        color: groupColor,
        isCollapsed,
        rowCount,
      });

      currentRow += rowCount;
    });

    const total = processedGroups.reduce((sum, g) => sum + (g.isCollapsed ? 0 : g.rowCount) * rowHeight + 36, 0);

    return { groups: processedGroups, totalHeight: total };
  }, [records, startField, endField, primaryField, colorField, groupField, fields, viewStart, msPerPixel, totalWidth, collapsedGroups, rowHeight]);

  // Navigation
  const navigateTimeline = (direction: 'prev' | 'next') => {
    const newStart = new Date(viewStart);
    switch (timeScale) {
      case 'day':
        newStart.setDate(newStart.getDate() + (direction === 'next' ? 14 : -14));
        break;
      case 'week':
        newStart.setDate(newStart.getDate() + (direction === 'next' ? 56 : -56));
        break;
      case 'month':
        newStart.setMonth(newStart.getMonth() + (direction === 'next' ? 6 : -6));
        break;
      case 'quarter':
        newStart.setMonth(newStart.getMonth() + (direction === 'next' ? 12 : -12));
        break;
      case 'year':
        newStart.setFullYear(newStart.getFullYear() + (direction === 'next' ? 2 : -2));
        break;
    }
    setViewStart(newStart);
  };

  const goToToday = () => {
    const now = new Date();
    switch (timeScale) {
      case 'day':
      case 'week':
        setViewStart(new Date(now.getFullYear(), now.getMonth(), now.getDate() - 7));
        break;
      default:
        setViewStart(new Date(now.getFullYear(), now.getMonth(), 1));
    }
  };

  // Handlers
  const handleAddRecord = async (date?: Date) => {
    if (!startField) return;
    const recordDate = date || viewStart;
    await createRecord({ [startField.id]: recordDate.toISOString().split('T')[0] });
  };

  const handleDateChange = useCallback(async (recordId: string, startDate: string, endDate?: string) => {
    const updates: Record<string, unknown> = {};
    if (startField) {
      updates[startField.id] = startDate;
    }
    if (endField && endDate) {
      updates[endField.id] = endDate;
    }
    await updateRecord(recordId, updates);
  }, [startField, endField, updateRecord]);

  const handleRecordHover = useCallback((record: TableRecord | null, event?: React.MouseEvent) => {
    if (hoverTimeoutRef.current !== null) {
      window.clearTimeout(hoverTimeoutRef.current);
    }

    if (record && event) {
      hoverTimeoutRef.current = window.setTimeout(() => {
        setHoveredRecord({
          record,
          position: { x: event.clientX, y: event.clientY },
        });
      }, 500);
    } else {
      setHoveredRecord(null);
    }
  }, []);

  const toggleGroupCollapse = (groupKey: string) => {
    setCollapsedGroups(prev => {
      const next = new Set(prev);
      if (next.has(groupKey)) {
        next.delete(groupKey);
      } else {
        next.add(groupKey);
      }
      return next;
    });
  };

  // Empty state
  if (!startField) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500">
        <div className="text-center">
          <svg className="w-16 h-16 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <h3 className="text-lg font-medium text-gray-900 mb-2">No date field available</h3>
          <p className="text-sm text-gray-500 mb-4">Add a Date or DateTime field to use Timeline view</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-slate-50">
      {/* Header controls */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200 bg-white">
        <div className="flex items-center gap-4">
          <h2 className="text-lg font-semibold text-gray-900">
            {viewStart.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}
          </h2>
          <div className="flex gap-1">
            <button
              onClick={() => navigateTimeline('prev')}
              className="p-1.5 hover:bg-slate-100 rounded-md transition-colors"
              title="Previous"
            >
              <svg className="w-5 h-5 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
            </button>
            <button
              onClick={() => navigateTimeline('next')}
              className="p-1.5 hover:bg-slate-100 rounded-md transition-colors"
              title="Next"
            >
              <svg className="w-5 h-5 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </button>
          </div>
          <button
            onClick={goToToday}
            className="px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-100 rounded-md transition-colors"
          >
            Today
          </button>
        </div>

        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <span className="text-sm text-slate-600">Scale:</span>
            <select
              value={timeScale}
              onChange={(e) => setTimeScale(e.target.value as TimeScale)}
              className="input py-1.5 w-28 text-sm"
            >
              <option value="day">Day</option>
              <option value="week">Week</option>
              <option value="month">Month</option>
              <option value="quarter">Quarter</option>
              <option value="year">Year</option>
            </select>
          </div>

          <div className="relative">
            <button
              onClick={() => setShowSettings(!showSettings)}
              className={`p-2 rounded-md transition-colors ${showSettings ? 'bg-primary/10 text-primary' : 'hover:bg-slate-100 text-slate-600'}`}
              title="Timeline Settings"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
            </button>
            <TimelineSettings isOpen={showSettings} onClose={() => setShowSettings(false)} />
          </div>
        </div>
      </div>

      {/* Timeline grid */}
      <div ref={gridRef} className="flex-1 overflow-auto">
        <div className="min-w-max">
          {/* Column headers */}
          <div className="flex border-b border-slate-200 bg-white sticky top-0 z-20">
            <div className="w-56 flex-shrink-0 p-3 border-r border-slate-200 bg-slate-50">
              <span className="font-medium text-sm text-slate-700">
                {groupField ? groupField.name : 'Records'}
              </span>
            </div>
            <div className="flex relative">
              {columns.map((col, i) => {
                const isToday = isSameDay(col.date, new Date());
                return (
                  <div
                    key={i}
                    className={`border-r border-slate-100 p-2 text-center flex flex-col justify-center ${
                      isToday ? 'bg-red-50' : ''
                    }`}
                    style={{ width: columnWidth }}
                  >
                    {col.subLabel && (
                      <span className="text-xs text-slate-400 font-medium">{col.subLabel}</span>
                    )}
                    <span className={`text-sm ${isToday ? 'text-red-600 font-semibold' : 'text-slate-600'}`}>
                      {col.label}
                    </span>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Groups and records */}
          <div className="relative">
            {/* Today marker */}
            {showTodayMarker && (
              <TodayMarker
                left={todayPosition + 224}
                height={totalHeight + 100}
                visible={todayPosition >= 0 && todayPosition <= totalWidth}
              />
            )}

            {groups.map((group) => (
              <div key={group.key} className="border-b border-slate-200">
                {/* Group header */}
                {groupField && (
                  <div
                    className="flex items-center gap-2 px-3 py-2 bg-slate-50 border-b border-slate-100 cursor-pointer hover:bg-slate-100 transition-colors"
                    onClick={() => toggleGroupCollapse(group.key)}
                  >
                    <button className="p-0.5 hover:bg-slate-200 rounded">
                      <svg
                        className={`w-4 h-4 text-slate-500 transition-transform ${group.isCollapsed ? '' : 'rotate-90'}`}
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                      </svg>
                    </button>
                    {group.color && (
                      <span
                        className="w-3 h-3 rounded-full flex-shrink-0"
                        style={{ backgroundColor: group.color }}
                      />
                    )}
                    <span className="font-medium text-sm text-slate-700">{group.label}</span>
                    <span className="text-xs text-slate-500">({group.records.length})</span>
                  </div>
                )}

                {/* Records area */}
                {!group.isCollapsed && (
                  <div
                    className="flex"
                    style={{ minHeight: Math.max(rowHeight, group.rowCount * rowHeight) }}
                  >
                    {/* Sidebar */}
                    <div className="w-56 flex-shrink-0 border-r border-slate-200 bg-white">
                      {!groupField && group.records.map((bar) => (
                        <div
                          key={bar.record.id}
                          className="px-3 flex items-center border-b border-slate-50 hover:bg-slate-50"
                          style={{ height: rowHeight }}
                        >
                          <button
                            onClick={() => setExpandedRecord(bar.record)}
                            className="text-sm text-slate-700 truncate hover:text-primary transition-colors"
                          >
                            {bar.title}
                          </button>
                        </div>
                      ))}
                    </div>

                    {/* Timeline area */}
                    <div
                      className="relative flex-1"
                      style={{
                        width: totalWidth,
                        height: Math.max(rowHeight, group.rowCount * rowHeight),
                      }}
                    >
                      {/* Grid lines */}
                      <div className="absolute inset-0 flex">
                        {columns.map((col, i) => {
                          const isToday = isSameDay(col.date, new Date());
                          const isWeekend = col.date.getDay() === 0 || col.date.getDay() === 6;
                          return (
                            <div
                              key={i}
                              className={`border-r border-slate-100 ${
                                isToday ? 'bg-red-50/50' : isWeekend && timeScale === 'day' ? 'bg-slate-50' : ''
                              }`}
                              style={{ width: columnWidth }}
                            />
                          );
                        })}
                      </div>

                      {/* Record bars */}
                      {group.records.map((bar) => (
                        <RecordBar
                          key={bar.record.id}
                          record={bar.record}
                          left={bar.left}
                          width={bar.width}
                          title={bar.title}
                          color={bar.color}
                          fields={fields}
                          labelFieldIds={labelFieldIds}
                          rowHeight={rowHeight}
                          stackIndex={bar.stackIndex}
                          msPerPixel={msPerPixel}
                          startField={startField}
                          endField={endField}
                          onRecordClick={setExpandedRecord}
                          onDateChange={handleDateChange}
                          onHover={handleRecordHover}
                        />
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ))}

            {/* Add record row */}
            <div className="flex border-b border-slate-100 hover:bg-slate-50" style={{ height: rowHeight }}>
              <div className="w-56 flex-shrink-0 px-3 flex items-center border-r border-slate-200">
                <button
                  onClick={() => handleAddRecord()}
                  className="text-sm text-slate-500 hover:text-slate-700 flex items-center gap-1.5 transition-colors"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                  Add record
                </button>
              </div>
              <div className="flex-1" style={{ width: totalWidth }} />
            </div>
          </div>
        </div>
      </div>

      {/* Empty state overlay */}
      {records.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none bg-white/80">
          <div className="text-center pointer-events-auto">
            <svg className="w-16 h-16 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            <p className="text-gray-500 mb-4">No records to display</p>
            <button
              onClick={() => handleAddRecord(new Date())}
              className="btn btn-primary"
            >
              Create first record
            </button>
          </div>
        </div>
      )}

      {/* Record preview tooltip */}
      {hoveredRecord && (
        <RecordPreview
          record={hoveredRecord.record}
          fields={fields}
          position={hoveredRecord.position}
          onClose={() => setHoveredRecord(null)}
          onEdit={() => {
            setExpandedRecord(hoveredRecord.record);
            setHoveredRecord(null);
          }}
        />
      )}

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
  );
}

// Helper functions
function getRecordColor(record: TableRecord, colorField: Field | undefined, fields: Field[]): string {
  if (colorField) {
    const value = record.values[colorField.id] as string;
    if (value) {
      const choice = colorField.options?.choices?.find(c => c.id === value);
      if (choice) return choice.color;
    }
  }

  // Fallback to first single_select field with a value
  const selectField = fields.find(f => f.type === 'single_select' && record.values[f.id]);
  if (selectField) {
    const value = record.values[selectField.id] as string;
    const choice = selectField.options?.choices?.find(c => c.id === value);
    if (choice) return choice.color;
  }

  // Default colors based on record position
  const colors = ['#2D7FF9', '#20C933', '#FCAE40', '#F82B60', '#8B46FF', '#0D9488', '#EC4899', '#F97316'];
  const hash = record.id.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
  return colors[hash % colors.length];
}

function isSameDay(d1: Date, d2: Date): boolean {
  return d1.getFullYear() === d2.getFullYear() &&
    d1.getMonth() === d2.getMonth() &&
    d1.getDate() === d2.getDate();
}

function getWeekNumber(date: Date): number {
  const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
  const dayNum = d.getUTCDay() || 7;
  d.setUTCDate(d.getUTCDate() + 4 - dayNum);
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1));
  return Math.ceil((((d.getTime() - yearStart.getTime()) / 86400000) + 1) / 7);
}
