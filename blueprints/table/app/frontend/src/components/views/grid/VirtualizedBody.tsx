import { useRef, useMemo, Fragment, memo } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import type { TableRecord, Field, CellValue } from '../../../types';
import { CellEditor } from './CellEditor';
import { FillHandle } from './FillHandle';

// Row height options
const ROW_HEIGHTS = {
  short: { height: 36, class: 'h-9' },
  medium: { height: 56, class: 'h-14' },
  tall: { height: 96, class: 'h-24' },
  extra_tall: { height: 144, class: 'h-36' },
} as const;

type RowHeightKey = keyof typeof ROW_HEIGHTS;

// Virtual row types
type VirtualRowType = 'group-header' | 'record' | 'summary' | 'add-row';

interface VirtualRow {
  type: VirtualRowType;
  key: string;
  // For group headers
  group?: string;
  groupRecordCount?: number;
  isCollapsed?: boolean;
  // For record rows
  record?: TableRecord;
  rowIndex?: number;
}

interface VirtualizedBodyProps {
  groupedRecords: { group: string; records: TableRecord[] }[];
  visibleFields: Field[];
  columnWidths: Record<string, number>;
  rowHeight: RowHeightKey;
  frozenColumnCount: number;
  collapsedGroups: Set<string>;
  selectedCell: { recordId: string; fieldId: string } | null;
  editingCell: { recordId: string; fieldId: string } | null;
  selectedRows: Set<string>;
  groupBy: string | null;
  showSummaryBar: boolean;
  recordIndexMap: Map<string, number>;
  // Callbacks
  onCellClick: (recordId: string, fieldId: string) => void;
  onCellDoubleClick: (recordId: string, fieldId: string) => void;
  onCellChange: (recordId: string, fieldId: string, value: CellValue) => void;
  onCancelEdit: () => void;
  onFillEnd: (deltaRows: number, deltaCols: number) => void;
  onToggleRowSelection: (recordId: string, e: React.MouseEvent) => void;
  onExpandRecord: (record: TableRecord) => void;
  onRowContextMenu: (e: React.MouseEvent, recordId: string) => void;
  onToggleGroupCollapse: (group: string) => void;
  onAddRow: () => void;
  onClearCellRange: () => void;
  getRowColor: (record: TableRecord) => string | undefined;
  getFrozenColumnOffset: (colIdx: number) => number;
  isCellInRange: (rowIdx: number, colIdx: number) => boolean;
  // Summary bar component (passed as children or render prop)
  renderSummaryBar?: () => React.ReactNode;
}

// Wrap with React.memo to prevent unnecessary re-renders when props haven't changed.
// This is important for performance since VirtualizedBody receives many callback props.
export const VirtualizedBody = memo(function VirtualizedBody({
  groupedRecords,
  visibleFields,
  columnWidths,
  rowHeight,
  frozenColumnCount,
  collapsedGroups,
  selectedCell,
  editingCell,
  selectedRows,
  groupBy,
  showSummaryBar,
  recordIndexMap,
  onCellClick,
  onCellDoubleClick,
  onCellChange,
  onCancelEdit,
  onFillEnd,
  onToggleRowSelection,
  onExpandRecord,
  onRowContextMenu,
  onToggleGroupCollapse,
  onAddRow,
  onClearCellRange,
  getRowColor,
  getFrozenColumnOffset,
  isCellInRange,
  renderSummaryBar,
}: VirtualizedBodyProps) {
  const parentRef = useRef<HTMLDivElement>(null);

  // Build flat array of virtual rows for virtualization
  const virtualRows = useMemo<VirtualRow[]>(() => {
    const rows: VirtualRow[] = [];

    groupedRecords.forEach(({ group, records: groupRecords }) => {
      // Add group header if grouping is enabled
      if (groupBy) {
        rows.push({
          type: 'group-header',
          key: `group-${group}`,
          group,
          groupRecordCount: groupRecords.length,
          isCollapsed: collapsedGroups.has(group),
        });
      }

      // Add record rows if not collapsed
      if (!collapsedGroups.has(group)) {
        groupRecords.forEach((record) => {
          rows.push({
            type: 'record',
            key: `record-${record.id}`,
            record,
            rowIndex: recordIndexMap.get(record.id) ?? 0,
          });
        });
      }
    });

    // Add summary bar row
    if (showSummaryBar) {
      rows.push({
        type: 'summary',
        key: 'summary-bar',
      });
    }

    // Add "Add row" button row
    rows.push({
      type: 'add-row',
      key: 'add-row',
    });

    return rows;
  }, [groupedRecords, groupBy, collapsedGroups, showSummaryBar, recordIndexMap]);

  // Calculate row height based on type
  const getRowHeight = (index: number): number => {
    const row = virtualRows[index];
    if (!row) return ROW_HEIGHTS[rowHeight].height;

    switch (row.type) {
      case 'group-header':
        return 40;
      case 'summary':
        return ROW_HEIGHTS[rowHeight].height;
      case 'add-row':
        return ROW_HEIGHTS[rowHeight].height;
      case 'record':
      default:
        return ROW_HEIGHTS[rowHeight].height;
    }
  };

  // Set up virtualizer
  const rowVirtualizer = useVirtualizer({
    count: virtualRows.length,
    getScrollElement: () => parentRef.current,
    estimateSize: (index) => getRowHeight(index),
    overscan: 10,
  });

  const currentRowHeight = ROW_HEIGHTS[rowHeight];

  // Render a single virtual row based on its type
  const renderVirtualRow = (row: VirtualRow, virtualRow: ReturnType<typeof rowVirtualizer.getVirtualItems>[0]) => {
    switch (row.type) {
      case 'group-header':
        return (
          <tr
            key={row.key}
            className="bg-slate-50"
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: `${virtualRow.size}px`,
              transform: `translateY(${virtualRow.start}px)`,
            }}
          >
            <td colSpan={visibleFields.length + 2} className="border-b border-slate-200 px-3 py-2">
              <button
                onClick={() => onToggleGroupCollapse(row.group || '')}
                className="flex items-center gap-2 text-sm font-semibold text-slate-700"
              >
                <svg
                  className={`w-4 h-4 transition-transform ${row.isCollapsed ? '-rotate-90' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
                <span>{row.group || '(Empty)'}</span>
                <span className="text-xs text-slate-500 font-normal">{row.groupRecordCount}</span>
              </button>
            </td>
          </tr>
        );

      case 'record':
        if (!row.record) return null;
        const record = row.record;
        const rowIndex = row.rowIndex ?? 0;
        const rowColor = getRowColor(record);
        return (
          <tr
            key={row.key}
            data-row={rowIndex}
            className={`group ${selectedRows.has(record.id) ? 'bg-primary-50' : 'hover:bg-slate-50'}`}
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: `${virtualRow.size}px`,
              transform: `translateY(${virtualRow.start}px)`,
              backgroundColor: rowColor && !selectedRows.has(record.id) ? `${rowColor}15` : undefined,
            }}
            onContextMenu={(e) => onRowContextMenu(e, record.id)}
          >
            {/* Row number and checkbox - frozen */}
            <td className="border-b border-r border-slate-200 p-0 sticky left-0 bg-white z-10 relative w-16">
              {rowColor && (
                <div
                  className="absolute left-0 top-0 bottom-0 w-1"
                  style={{ backgroundColor: rowColor }}
                />
              )}
              <div className={`flex items-center justify-center ${currentRowHeight.class} gap-1`}>
                <input
                  type="checkbox"
                  checked={selectedRows.has(record.id)}
                  onChange={(e) => onToggleRowSelection(record.id, e as unknown as React.MouseEvent)}
                  className="w-4 h-4 rounded border-gray-300 opacity-0 group-hover:opacity-100 checked:opacity-100"
                />
                <span className="text-xs text-gray-400 w-6 text-center group-hover:hidden">
                  {rowIndex + 1}
                </span>
                <button
                  onClick={() => onExpandRecord(record)}
                  className="hidden group-hover:block text-gray-400 hover:text-gray-600"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
                  </svg>
                </button>
              </div>
            </td>

            {/* Data cells */}
            {visibleFields.map((field, colIdx) => {
              const isSelected = selectedCell?.recordId === record.id && selectedCell?.fieldId === field.id;
              const isEditing = editingCell?.recordId === record.id && editingCell?.fieldId === field.id;
              const isInRange = isCellInRange(rowIndex, colIdx);
              const isFrozen = colIdx < frozenColumnCount;
              const isLastFrozen = colIdx === frozenColumnCount - 1;
              const value = record.values[field.id] ?? null;

              return (
                <td
                  key={field.id}
                  data-row={rowIndex}
                  data-field={field.id}
                  className={`border-b border-r border-slate-200 p-0 relative ${
                    isSelected ? 'ring-2 ring-primary ring-inset z-10' : ''
                  } ${isInRange && !isSelected ? 'bg-primary-50/50' : ''} ${
                    isFrozen ? 'sticky bg-white z-10' : ''
                  } ${isLastFrozen && frozenColumnCount > 0 ? 'after:absolute after:right-0 after:top-0 after:bottom-0 after:w-0.5 after:bg-slate-300 after:shadow-sm' : ''}`}
                  style={{
                    width: columnWidths[field.id] || field.width || 200,
                    height: currentRowHeight.height,
                    ...(isFrozen ? { left: getFrozenColumnOffset(colIdx) } : {}),
                  }}
                  onClick={() => {
                    onCellClick(record.id, field.id);
                    onClearCellRange();
                  }}
                  onDoubleClick={() => onCellDoubleClick(record.id, field.id)}
                >
                  <CellEditor
                    field={field}
                    value={value}
                    isEditing={isEditing}
                    onChange={(newValue) => onCellChange(record.id, field.id, newValue)}
                    onCancel={onCancelEdit}
                    rowHeight={rowHeight}
                  />
                  {isSelected && !isEditing && (
                    <FillHandle
                      onFillStart={() => {}}
                      onFillMove={() => {}}
                      onFillEnd={onFillEnd}
                    />
                  )}
                </td>
              );
            })}

            {/* Empty cell for add field column */}
            <td className="border-b border-slate-200" style={{ height: currentRowHeight.height }} />
          </tr>
        );

      case 'summary':
        return (
          <tr
            key={row.key}
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: `${virtualRow.size}px`,
              transform: `translateY(${virtualRow.start}px)`,
            }}
          >
            <td colSpan={visibleFields.length + 2} className="p-0">
              {renderSummaryBar?.()}
            </td>
          </tr>
        );

      case 'add-row':
        return (
          <tr
            key={row.key}
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: `${virtualRow.size}px`,
              transform: `translateY(${virtualRow.start}px)`,
            }}
          >
            <td colSpan={visibleFields.length + 2} className="border-b border-slate-200 p-0">
              <button
                onClick={onAddRow}
                className={`w-full ${currentRowHeight.class} text-left px-4 text-sm text-slate-500 hover:bg-slate-50 flex items-center gap-2`}
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Add row
              </button>
            </td>
          </tr>
        );

      default:
        return null;
    }
  };

  return (
    <div
      ref={parentRef}
      className="flex-1 overflow-auto"
      style={{ contain: 'strict' }}
    >
      <div
        style={{
          height: `${rowVirtualizer.getTotalSize()}px`,
          width: '100%',
          position: 'relative',
        }}
      >
        <table className="w-full border-collapse" style={{ tableLayout: 'fixed' }}>
          <colgroup>
            <col style={{ width: 64 }} />
            {visibleFields.map((field) => (
              <col key={field.id} style={{ width: columnWidths[field.id] || field.width || 200 }} />
            ))}
            <col style={{ width: 128 }} />
          </colgroup>
          <tbody>
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const row = virtualRows[virtualRow.index];
              if (!row) return null;
              return (
                <Fragment key={virtualRow.key}>
                  {renderVirtualRow(row, virtualRow)}
                </Fragment>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
});
