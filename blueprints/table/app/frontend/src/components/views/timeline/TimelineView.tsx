import { useState, useMemo, useCallback, useRef } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { TableRecord, Field, Dependency } from '../../../types';
import { RecordSidebar } from '../RecordSidebar';
import { TimelineSettings } from './TimelineSettings';
import { RecordBar } from './RecordBar';
import { RecordPreview } from './RecordPreview';
import { TodayMarker } from './TodayMarker';
import { TimelineSummaryBar } from './TimelineSummaryBar';
import { DependencyArrow, DependencyCreator } from './DependencyArrow';

type TimeScale = 'day' | 'week' | '2weeks' | 'month' | 'quarter' | 'year';
type LayoutMode = 'standard' | 'gantt';

interface RecordBarData {
  record: TableRecord;
  left: number;
  width: number;
  title: string;
  color: string;
  row: number;
  stackIndex: number;
  groupKey: string;
  top: number;
}

interface GroupData {
  key: string;
  label: string;
  records: RecordBarData[];
  color?: string;
  isCollapsed: boolean;
  rowCount: number;
  level: number;
  parentKey?: string;
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

  // New state for enhanced features
  const [showWeekends, setShowWeekends] = useState(currentView?.settings?.show_weekends ?? true);
  const [layoutMode, setLayoutMode] = useState<LayoutMode>('standard');
  const [showSummaryBar, setShowSummaryBar] = useState(true);

  // Drag-to-create state
  const [isCreating, setIsCreating] = useState(false);
  const [createStart, setCreateStart] = useState<{ x: number; date: Date; groupKey: string } | null>(null);
  const [createEnd, setCreateEnd] = useState<{ x: number; date: Date } | null>(null);

  // Dependency state
  const [dependencies, setDependencies] = useState<Dependency[]>([]);
  const [isCreatingDependency, setIsCreatingDependency] = useState(false);
  const [dependencySource, setDependencySource] = useState<{ recordId: string; x: number; y: number; side: 'start' | 'end' } | null>(null);
  const [dependencyTarget, setDependencyTarget] = useState<{ x: number; y: number } | null>(null);
  const [selectedDependency, setSelectedDependency] = useState<string | null>(null);
  const showDependencies = currentView?.settings?.show_dependencies ?? true;

  const hoverTimeoutRef = useRef<number | null>(null);
  const gridRef = useRef<HTMLDivElement>(null);
  const timelineContentRef = useRef<HTMLDivElement>(null);

  const [viewStart, setViewStart] = useState(() => {
    const now = new Date();
    return new Date(now.getFullYear(), now.getMonth(), 1);
  });

  // Get configuration from view settings
  const startFieldId = currentView?.settings?.start_field_id || currentView?.config?.dateField;
  const endFieldId = currentView?.settings?.end_field_id || currentView?.config?.endDateField;
  const groupFieldId = currentView?.settings?.group_field_id;
  const groupFieldIds = currentView?.settings?.group_field_ids || (groupFieldId ? [groupFieldId] : []);
  const colorFieldId = currentView?.settings?.color_field_id;
  const labelFieldIds = currentView?.settings?.label_field_ids;
  const showTodayMarker = currentView?.settings?.show_today_marker ?? true;
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

  // Get grouping fields
  const groupFields = useMemo(() => {
    return groupFieldIds
      .map(id => fields.find(f => f.id === id))
      .filter((f): f is Field => f !== undefined);
  }, [fields, groupFieldIds]);

  const groupField = groupFields[0];

  // Get color field
  const colorField = useMemo(() => {
    if (colorFieldId) {
      return fields.find(f => f.id === colorFieldId);
    }
    return fields.find(f => f.type === 'single_select');
  }, [fields, colorFieldId]);

  // Get primary field for display
  const primaryField = fields.find(f => f.type === 'text' || (f.type as string) === 'single_line_text') || fields[0];

  // Memoize record index map for O(1) lookups (performance optimization)
  const recordIndexMap = useMemo(
    () => new Map(records.map((r, i) => [r.id, i])),
    [records]
  );

  // Memoized navigation helper using recordIndexMap (O(1) instead of O(n))
  const getRecordNavigation = useCallback((record: TableRecord | null) => {
    if (!record) return { hasPrev: false, hasNext: false, position: 0 };
    const index = recordIndexMap.get(record.id) ?? -1;
    return {
      hasPrev: index > 0,
      hasNext: index < records.length - 1,
      position: index + 1,
    };
  }, [recordIndexMap, records.length]);

  const handleNavigateRecord = useCallback((direction: 'prev' | 'next') => {
    if (!expandedRecord) return;
    const currentIndex = recordIndexMap.get(expandedRecord.id) ?? -1;
    const newIndex = direction === 'prev' ? currentIndex - 1 : currentIndex + 1;
    if (newIndex >= 0 && newIndex < records.length) {
      setExpandedRecord(records[newIndex]);
    }
  }, [expandedRecord, recordIndexMap, records]);

  // Calculate visible time range and columns
  const { columnWidth, columns, totalWidth, msPerPixel } = useMemo(() => {
    let end: Date;
    let colWidth: number;
    const cols: { date: Date; label: string; subLabel?: string; isWeekend?: boolean }[] = [];

    const start = new Date(viewStart);

    switch (timeScale) {
      case 'day':
        end = new Date(start);
        end.setDate(end.getDate() + 60);
        colWidth = 40;
        for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
          const isWeekend = d.getDay() === 0 || d.getDay() === 6;
          cols.push({
            date: new Date(d),
            label: d.getDate().toString(),
            subLabel: d.getDate() === 1 ? d.toLocaleDateString('en-US', { month: 'short' }) : undefined,
            isWeekend,
          });
        }
        break;
      case 'week':
        end = new Date(start);
        end.setDate(end.getDate() + 168);
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
      case '2weeks':
        end = new Date(start);
        end.setDate(end.getDate() + 336);
        colWidth = 100;
        for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 14)) {
          const weekNum = getWeekNumber(d);
          cols.push({
            date: new Date(d),
            label: `W${weekNum}-${weekNum + 1}`,
            subLabel: d.getMonth() !== new Date(d.getTime() + 86400000 * 13).getMonth()
              ? d.toLocaleDateString('en-US', { month: 'short' })
              : undefined,
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

    // Filter weekends if needed (only for day view)
    const filteredCols = !showWeekends && timeScale === 'day'
      ? cols.filter(col => !col.isWeekend)
      : cols;

    const total = filteredCols.length * colWidth;
    const msPerPx = (end.getTime() - start.getTime()) / total;

    return {
      columnWidth: colWidth,
      columns: filteredCols,
      totalWidth: total,
      msPerPixel: msPerPx
    };
  }, [viewStart, timeScale, showWeekends]);

  // Calculate today marker position
  const todayPosition = useMemo(() => {
    const today = new Date();
    const position = (today.getTime() - viewStart.getTime()) / msPerPixel;
    return position;
  }, [viewStart, msPerPixel]);

  // Step 1: Filter records with valid dates (fewer dependencies = fewer recalculations)
  const recordsWithDates = useMemo(() => {
    if (!startField) return [];
    return records.filter(record => {
      const startDate = record.values[startField.id];
      return startDate !== null && startDate !== undefined;
    });
  }, [records, startField]);

  // Step 2: Group records (depends only on filtered records and groupField)
  const groupedRecordsMap = useMemo(() => {
    const grouped = new Map<string, TableRecord[]>();
    if (!groupField) {
      grouped.set('All Records', recordsWithDates);
      return grouped;
    }

    // Build choice lookup map once for O(1) access
    const choiceMap = new Map<string, string>();
    if (groupField.type === 'single_select') {
      groupField.options?.choices?.forEach(c => choiceMap.set(c.id, c.name));
    }

    recordsWithDates.forEach(record => {
      const groupValue = record.values[groupField.id];
      let groupKey = '(Empty)';

      if (groupValue !== null && groupValue !== undefined && groupValue !== '') {
        if (groupField.type === 'single_select') {
          groupKey = choiceMap.get(String(groupValue)) || String(groupValue);
        } else {
          groupKey = String(groupValue);
        }
      }

      const existing = grouped.get(groupKey);
      if (existing) {
        existing.push(record);
      } else {
        grouped.set(groupKey, [record]);
      }
    });

    return grouped;
  }, [recordsWithDates, groupField]);

  // Step 3: Calculate bar positions for each group (layout-dependent computation)
  const { groups, totalHeight, recordBars } = useMemo(() => {
    if (!startField) return { groups: [], totalHeight: 0, recordBars: new Map<string, RecordBarData>() };

    // Calculate bar positions for each group
    const processedGroups: GroupData[] = [];
    const allBars = new Map<string, RecordBarData>();
    let currentRow = 0;
    let cumulativeTop = 0;

    // Sort groups - put '(Empty)' at the end
    const sortedGroupKeys = Array.from(groupedRecordsMap.keys()).sort((a, b) => {
      if (a === '(Empty)') return 1;
      if (b === '(Empty)') return -1;
      return a.localeCompare(b);
    });

    sortedGroupKeys.forEach(groupKey => {
      const groupRecords = groupedRecordsMap.get(groupKey)!;
      const isCollapsed = collapsedGroups.has(groupKey);

      // Calculate bar positions
      const bars: RecordBarData[] = [];
      const occupiedSlots: { left: number; right: number; row: number }[] = [];

      // For Gantt mode, each record gets its own row
      const useGanttLayout = layoutMode === 'gantt';

      // Sort records by start date for Gantt layout
      const sortedRecords = useGanttLayout
        ? [...groupRecords].sort((a, b) => {
            const aStart = new Date(a.values[startField.id] as string).getTime();
            const bStart = new Date(b.values[startField.id] as string).getTime();
            return aStart - bStart;
          })
        : groupRecords;

      sortedRecords.forEach((record, recordIndex) => {
        const startDate = new Date(record.values[startField.id] as string);
        const endDate = endField && record.values[endField.id]
          ? new Date(record.values[endField.id] as string)
          : new Date(startDate.getTime() + 86400000);

        const startX = Math.max(0, (startDate.getTime() - viewStart.getTime()) / msPerPixel);
        const endX = Math.min(totalWidth, (endDate.getTime() - viewStart.getTime()) / msPerPixel);
        const width = Math.max(30, endX - startX);

        let stackIndex: number;

        if (useGanttLayout) {
          // Gantt: each record on its own row
          stackIndex = recordIndex;
        } else {
          // Standard: stack overlapping records
          stackIndex = 0;
          for (let i = 0; i <= occupiedSlots.length; i++) {
            const overlapping = occupiedSlots.filter(
              slot => slot.row === i && !(startX >= slot.right || startX + width <= slot.left)
            );
            if (overlapping.length === 0) {
              stackIndex = i;
              break;
            }
          }
        }

        occupiedSlots.push({ left: startX, right: startX + width, row: stackIndex });

        const barHeight = rowHeight === 32 ? 24 : rowHeight === 48 ? 36 : 28;
        const barData: RecordBarData = {
          record,
          left: startX,
          width,
          title: primaryField ? (record.values[primaryField.id] as string) || 'Untitled' : 'Untitled',
          color: getRecordColor(record, colorField || fields.find(f => f.type === 'single_select'), fields),
          row: currentRow,
          stackIndex,
          groupKey,
          top: cumulativeTop + (groupField ? 36 : 0) + 4 + stackIndex * (barHeight + 4),
        };

        bars.push(barData);
        allBars.set(record.id, barData);
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
        level: 0,
      });

      cumulativeTop += (groupField ? 36 : 0) + (isCollapsed ? 0 : rowCount * rowHeight);
      currentRow += rowCount;
    });

    const total = cumulativeTop + rowHeight; // Extra row for "Add record"

    return { groups: processedGroups, totalHeight: total, recordBars: allBars };
  }, [groupedRecordsMap, startField, endField, primaryField, colorField, groupField, fields, viewStart, msPerPixel, totalWidth, collapsedGroups, rowHeight, layoutMode]);

  // Navigation
  const navigateTimeline = (direction: 'prev' | 'next') => {
    const newStart = new Date(viewStart);
    switch (timeScale) {
      case 'day':
        newStart.setDate(newStart.getDate() + (direction === 'next' ? 14 : -14));
        break;
      case 'week':
      case '2weeks':
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
      case '2weeks':
        setViewStart(new Date(now.getFullYear(), now.getMonth(), now.getDate() - 7));
        break;
      default:
        setViewStart(new Date(now.getFullYear(), now.getMonth(), 1));
    }
  };

  // Handlers
  const handleAddRecord = async (date?: Date, groupKey?: string) => {
    if (!startField) return;
    const recordDate = date || viewStart;
    const values: Record<string, unknown> = {
      [startField.id]: recordDate.toISOString().split('T')[0],
    };

    // Set group field value if applicable
    if (groupField && groupKey && groupKey !== 'All Records' && groupKey !== '(Empty)') {
      if (groupField.type === 'single_select') {
        const choice = groupField.options?.choices?.find(c => c.name === groupKey);
        if (choice) values[groupField.id] = choice.id;
      } else {
        values[groupField.id] = groupKey;
      }
    }

    await createRecord(values);
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

  // Drag-to-create handlers
  const handleTimelineMouseDown = useCallback((e: React.MouseEvent, groupKey: string) => {
    if (e.target !== e.currentTarget) return; // Only on empty space
    if (!startField) return;

    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const date = new Date(viewStart.getTime() + x * msPerPixel);

    setIsCreating(true);
    setCreateStart({ x, date, groupKey });
    setCreateEnd({ x, date });
  }, [viewStart, msPerPixel, startField]);

  const handleTimelineMouseMove = useCallback((e: React.MouseEvent) => {
    if (!isCreating || !createStart) return;

    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const date = new Date(viewStart.getTime() + x * msPerPixel);

    setCreateEnd({ x, date });
  }, [isCreating, createStart, viewStart, msPerPixel]);

  const handleTimelineMouseUp = useCallback(async () => {
    if (!isCreating || !createStart || !createEnd || !startField) {
      setIsCreating(false);
      setCreateStart(null);
      setCreateEnd(null);
      return;
    }

    const startDate = createStart.date < createEnd.date ? createStart.date : createEnd.date;
    const endDate = createStart.date < createEnd.date ? createEnd.date : createStart.date;

    // Only create if dragged at least 5 pixels
    if (Math.abs(createEnd.x - createStart.x) > 5) {
      const values: Record<string, unknown> = {
        [startField.id]: startDate.toISOString().split('T')[0],
      };

      if (endField) {
        values[endField.id] = endDate.toISOString().split('T')[0];
      }

      // Set group field value if applicable
      if (groupField && createStart.groupKey && createStart.groupKey !== 'All Records' && createStart.groupKey !== '(Empty)') {
        if (groupField.type === 'single_select') {
          const choice = groupField.options?.choices?.find(c => c.name === createStart.groupKey);
          if (choice) values[groupField.id] = choice.id;
        } else {
          values[groupField.id] = createStart.groupKey;
        }
      }

      await createRecord(values);
    }

    setIsCreating(false);
    setCreateStart(null);
    setCreateEnd(null);
  }, [isCreating, createStart, createEnd, startField, endField, groupField, createRecord]);

  // Dependency handlers (_handleDependencyDragStart is for future use when wiring bar edge drag)
  const _handleDependencyDragStart = useCallback((recordId: string, x: number, y: number, side: 'start' | 'end') => {
    setIsCreatingDependency(true);
    setDependencySource({ recordId, x, y, side });
    setDependencyTarget({ x, y });
  }, []);
  void _handleDependencyDragStart; // Prevent unused variable warning

  const handleDependencyDragMove = useCallback((e: React.MouseEvent) => {
    if (!isCreatingDependency) return;
    const rect = timelineContentRef.current?.getBoundingClientRect();
    if (!rect) return;
    setDependencyTarget({
      x: e.clientX - rect.left + 224, // Account for sidebar
      y: e.clientY - rect.top,
    });
  }, [isCreatingDependency]);

  const handleDependencyDragEnd = useCallback((targetRecordId?: string) => {
    if (dependencySource && targetRecordId && targetRecordId !== dependencySource.recordId) {
      const newDependency: Dependency = {
        id: `dep-${Date.now()}`,
        table_id: '',
        source_record_id: dependencySource.recordId,
        target_record_id: targetRecordId,
        type: dependencySource.side === 'end' ? 'finish_to_start' : 'start_to_start',
        created_by: '',
        created_at: new Date().toISOString(),
      };
      setDependencies(prev => [...prev, newDependency]);
    }
    setIsCreatingDependency(false);
    setDependencySource(null);
    setDependencyTarget(null);
  }, [dependencySource]);

  const handleDeleteDependency = useCallback((depId: string) => {
    setDependencies(prev => prev.filter(d => d.id !== depId));
    setSelectedDependency(null);
  }, []);

  // Empty state
  if (!startField) {
    return (
      <div className="flex-1 flex items-center justify-center bg-[var(--at-bg)]">
        <div className="empty-state animate-fade-in">
          <div className="empty-state-icon-wrapper">
            <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          </div>
          <h3 className="empty-state-title">No date field available</h3>
          <p className="empty-state-description">Add a Date or DateTime field to use Timeline view</p>
          <button
            onClick={() => setShowSettings(true)}
            className="btn btn-primary mt-2"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            Configure Timeline
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-[var(--at-bg)]">
      {/* Header controls */}
      <div className="view-toolbar">
        <div className="view-toolbar-left">
          <h2 className="text-base font-semibold text-[var(--at-text)]">
            {viewStart.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}
          </h2>
          <div className="flex gap-1">
            <button
              onClick={() => navigateTimeline('prev')}
              className="icon-btn"
              title="Previous"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
            </button>
            <button
              onClick={() => navigateTimeline('next')}
              className="icon-btn"
              title="Next"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </button>
          </div>
          <button
            onClick={goToToday}
            className="btn btn-ghost text-sm"
          >
            Today
          </button>
        </div>

        <div className="view-toolbar-right">
          {/* Layout mode toggle */}
          <div className="segmented-control">
            <button
              onClick={() => setLayoutMode('standard')}
              className={`segmented-control-item ${layoutMode === 'standard' ? 'segmented-control-item-active' : ''}`}
              title="Standard layout"
            >
              Standard
            </button>
            <button
              onClick={() => setLayoutMode('gantt')}
              className={`segmented-control-item ${layoutMode === 'gantt' ? 'segmented-control-item-active' : ''}`}
              title="Gantt layout (one record per row)"
            >
              Gantt
            </button>
          </div>

          {/* Workdays toggle (day view only) */}
          {timeScale === 'day' && (
            <button
              onClick={() => setShowWeekends(!showWeekends)}
              className={`btn text-xs ${!showWeekends ? 'btn-primary' : 'btn-secondary'}`}
              title={showWeekends ? 'Show weekdays only' : 'Show all days'}
            >
              {showWeekends ? 'All days' : 'Weekdays'}
            </button>
          )}

          <div className="view-toolbar-divider" />

          <div className="flex items-center gap-2">
            <span className="text-sm text-[var(--at-text-secondary)]">Scale:</span>
            <select
              value={timeScale}
              onChange={(e) => setTimeScale(e.target.value as TimeScale)}
              className="input py-1.5 w-28 text-sm"
            >
              <option value="day">Day</option>
              <option value="week">Week</option>
              <option value="2weeks">2 Weeks</option>
              <option value="month">Month</option>
              <option value="quarter">Quarter</option>
              <option value="year">Year</option>
            </select>
          </div>

          {/* Summary bar toggle */}
          <button
            onClick={() => setShowSummaryBar(!showSummaryBar)}
            className={`icon-btn ${showSummaryBar ? 'bg-[var(--at-primary-soft)] text-[var(--at-primary)]' : ''}`}
            title="Toggle summary bar"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
          </button>

          <div className="relative">
            <button
              onClick={() => setShowSettings(!showSettings)}
              className={`icon-btn ${showSettings ? 'bg-[var(--at-primary-soft)] text-[var(--at-primary)]' : ''}`}
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
      <div
        ref={gridRef}
        className="flex-1 overflow-auto"
        onMouseMove={handleDependencyDragMove}
        onMouseUp={() => handleDependencyDragEnd()}
      >
        <div className="min-w-max" ref={timelineContentRef}>
          {/* Column headers */}
          <div className="flex border-b border-[var(--at-border)] bg-white sticky top-0 z-20">
            <div className="w-56 flex-shrink-0 p-3 border-r border-[var(--at-border)] bg-[var(--at-surface-muted)]">
              <span className="font-medium text-sm text-[var(--at-text)]">
                {groupField ? groupField.name : 'Records'}
              </span>
            </div>
            <div className="flex relative">
              {columns.map((col, i) => {
                const isToday = isSameDay(col.date, new Date());
                return (
                  <div
                    key={i}
                    className={`border-r border-[var(--at-border-light)] p-2 text-center flex flex-col justify-center ${
                      isToday ? 'bg-red-50' : col.isWeekend ? 'bg-[var(--at-surface-muted)]' : ''
                    }`}
                    style={{ width: columnWidth }}
                  >
                    {col.subLabel && (
                      <span className="text-xs text-[var(--at-muted)] font-medium">{col.subLabel}</span>
                    )}
                    <span className={`text-sm ${isToday ? 'text-red-600 font-semibold' : 'text-[var(--at-text-secondary)]'}`}>
                      {col.label}
                    </span>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Groups and records */}
          <div className="relative">
            {/* SVG overlay for dependencies */}
            {showDependencies && (
              <svg
                className="absolute inset-0 pointer-events-none z-15"
                style={{ overflow: 'visible', width: totalWidth + 224, height: totalHeight }}
              >
                {dependencies.map(dep => {
                  const sourceBar = recordBars.get(dep.source_record_id);
                  const targetBar = recordBars.get(dep.target_record_id);
                  if (!sourceBar || !targetBar) return null;

                  const barHeight = rowHeight === 32 ? 24 : rowHeight === 48 ? 36 : 28;

                  return (
                    <DependencyArrow
                      key={dep.id}
                      sourceX={sourceBar.left + 224}
                      sourceY={sourceBar.top + barHeight / 2}
                      sourceWidth={sourceBar.width}
                      targetX={targetBar.left + 224}
                      targetY={targetBar.top + barHeight / 2}
                      targetWidth={targetBar.width}
                      type={dep.type}
                      isHighlighted={selectedDependency === dep.id}
                      onClick={() => {
                        setSelectedDependency(dep.id);
                      }}
                    />
                  );
                })}

                {/* Dependency creation line */}
                {isCreatingDependency && dependencySource && dependencyTarget && (
                  <DependencyCreator
                    startX={dependencySource.x}
                    startY={dependencySource.y}
                    endX={dependencyTarget.x}
                    endY={dependencyTarget.y}
                  />
                )}
              </svg>
            )}

            {/* Today marker */}
            {showTodayMarker && (
              <TodayMarker
                left={todayPosition + 224}
                height={totalHeight + 100}
                visible={todayPosition >= 0 && todayPosition <= totalWidth}
              />
            )}

            {groups.map((group) => (
              <div key={group.key} className="border-b border-[var(--at-border)]">
                {/* Group header */}
                {groupField && (
                  <div
                    className="flex items-center gap-2 px-3 py-2 bg-[var(--at-surface-muted)] border-b border-[var(--at-border-light)] cursor-pointer hover:bg-[var(--at-surface-hover)] transition-colors"
                    onClick={() => toggleGroupCollapse(group.key)}
                  >
                    <button className="p-0.5 hover:bg-[var(--at-surface-hover)] rounded">
                      <svg
                        className={`w-4 h-4 text-[var(--at-muted)] transition-transform ${group.isCollapsed ? '' : 'rotate-90'}`}
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
                    <span className="font-medium text-sm text-[var(--at-text)]">{group.label}</span>
                    <span className="text-xs text-[var(--at-muted)]">({group.records.length})</span>
                  </div>
                )}

                {/* Records area */}
                {!group.isCollapsed && (
                  <div
                    className="flex"
                    style={{ minHeight: Math.max(rowHeight, group.rowCount * rowHeight) }}
                  >
                    {/* Sidebar */}
                    <div className="w-56 flex-shrink-0 border-r border-[var(--at-border)] bg-white">
                      {!groupField && group.records.map((bar) => (
                        <div
                          key={bar.record.id}
                          className="px-3 flex items-center border-b border-[var(--at-border-light)] hover:bg-[var(--at-surface-muted)]"
                          style={{ height: rowHeight }}
                        >
                          <button
                            onClick={() => setExpandedRecord(bar.record)}
                            className="text-sm text-[var(--at-text)] truncate hover:text-primary transition-colors"
                          >
                            {bar.title}
                          </button>
                        </div>
                      ))}
                    </div>

                    {/* Timeline area */}
                    <div
                      className={`relative flex-1 ${isCreating ? 'cursor-crosshair' : ''}`}
                      style={{
                        width: totalWidth,
                        height: Math.max(rowHeight, group.rowCount * rowHeight),
                      }}
                      onMouseDown={(e) => handleTimelineMouseDown(e, group.key)}
                      onMouseMove={handleTimelineMouseMove}
                      onMouseUp={handleTimelineMouseUp}
                      onMouseLeave={() => {
                        if (isCreating) {
                          setIsCreating(false);
                          setCreateStart(null);
                          setCreateEnd(null);
                        }
                      }}
                    >
                      {/* Grid lines */}
                      <div className="absolute inset-0 flex">
                        {columns.map((col, i) => {
                          const isToday = isSameDay(col.date, new Date());
                          return (
                            <div
                              key={i}
                              className={`border-r border-[var(--at-border-light)] ${
                                isToday ? 'bg-red-50/50' : col.isWeekend && timeScale === 'day' ? 'bg-[var(--at-surface-muted)]' : ''
                              }`}
                              style={{ width: columnWidth }}
                            />
                          );
                        })}
                      </div>

                      {/* Drag-to-create preview */}
                      {isCreating && createStart && createEnd && createStart.groupKey === group.key && (
                        <div
                          className="absolute rounded bg-primary/30 border-2 border-primary border-dashed z-30"
                          style={{
                            left: Math.min(createStart.x, createEnd.x),
                            width: Math.abs(createEnd.x - createStart.x),
                            height: rowHeight - 8,
                            top: 4,
                          }}
                        >
                          <span className="text-xs text-primary font-medium px-2">
                            New record
                          </span>
                        </div>
                      )}

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
            <div className="flex border-b border-[var(--at-border-light)] hover:bg-[var(--at-surface-muted)]" style={{ height: rowHeight }}>
              <div className="w-56 flex-shrink-0 px-3 flex items-center border-r border-[var(--at-border)]">
                <button
                  onClick={() => handleAddRecord()}
                  className="text-sm text-[var(--at-muted)] hover:text-[var(--at-text)] flex items-center gap-1.5 transition-colors"
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

        {/* Summary bar */}
        {showSummaryBar && startField && (
          <TimelineSummaryBar
            records={records}
            fields={fields}
            columns={columns}
            columnWidth={columnWidth}
            startField={startField}
            endField={endField}
          />
        )}
      </div>

      {/* Empty state overlay */}
      {records.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none bg-white/80">
          <div className="text-center pointer-events-auto">
            <svg className="w-16 h-16 mx-auto mb-4 text-[var(--at-muted)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            <p className="text-[var(--at-muted)] mb-4">No records to display</p>
            <button
              onClick={() => handleAddRecord(new Date())}
              className="btn btn-primary"
            >
              Create first record
            </button>
          </div>
        </div>
      )}

      {/* Selected dependency actions */}
      {selectedDependency && (
        <div className="fixed bottom-4 left-1/2 -translate-x-1/2 bg-white rounded-lg shadow-lg border border-[var(--at-border)] px-4 py-2 flex items-center gap-3 z-50">
          <span className="text-sm text-[var(--at-text-secondary)]">Dependency selected</span>
          <button
            onClick={() => handleDeleteDependency(selectedDependency)}
            className="text-sm text-red-600 hover:text-red-700 font-medium"
          >
            Delete
          </button>
          <button
            onClick={() => setSelectedDependency(null)}
            className="text-sm text-[var(--at-muted)] hover:text-[var(--at-text)]"
          >
            Cancel
          </button>
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
          onNavigate={handleNavigateRecord}
          hasPrev={getRecordNavigation(expandedRecord).hasPrev}
          hasNext={getRecordNavigation(expandedRecord).hasNext}
          position={getRecordNavigation(expandedRecord).position}
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
