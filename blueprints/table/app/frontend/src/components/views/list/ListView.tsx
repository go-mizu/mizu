import { useState, useMemo, useRef, useCallback, memo } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import { useBaseStore } from '../../../stores/baseStore';
import type { TableRecord, Field, CellValue } from '../../../types';
import { RecordSidebar } from '../RecordSidebar';

// Virtualization threshold - use virtualization for large lists
const VIRTUALIZATION_THRESHOLD = 100;

export function ListView() {
  const {
    fields,
    getSortedRecords,
    createRecord,
    deleteRecord,
    updateCellValue,
  } = useBaseStore();

  const records = getSortedRecords();
  const [expandedRecord, setExpandedRecord] = useState<TableRecord | null>(null);
  const [selectedRecords, setSelectedRecords] = useState<Set<string>>(new Set());
  const [editingRecord, setEditingRecord] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');

  // Ref for virtualization container
  const parentRef = useRef<HTMLDivElement>(null);

  // Memoize record index map for O(1) lookups instead of repeated findIndex calls
  const recordIndexMap = useMemo(
    () => new Map(records.map((r, i) => [r.id, i])),
    [records]
  );

  // Determine if we should use virtualization
  const useVirtualization = records.length > VIRTUALIZATION_THRESHOLD;

  // Get primary field (first text field or first field)
  const primaryField = useMemo(() => {
    return fields.find(f => f.type === 'text') || fields[0];
  }, [fields]);

  // Get secondary fields for preview
  const secondaryFields = useMemo(() => {
    return fields.filter(f => f.id !== primaryField?.id).slice(0, 3);
  }, [fields, primaryField]);

  const handleAddRecord = async () => {
    await createRecord({});
  };

  const handleDeleteSelected = async () => {
    for (const id of selectedRecords) {
      await deleteRecord(id);
    }
    setSelectedRecords(new Set());
  };

  // Toggle selection - uses recordIndexMap for O(1) lookups in range selection
  const toggleSelection = (recordId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const newSelection = new Set(selectedRecords);
    if (e.shiftKey && selectedRecords.size > 0) {
      const lastSelected = Array.from(selectedRecords).pop()!;
      // Use recordIndexMap for O(1) lookup instead of O(n) findIndex
      const lastIndex = recordIndexMap.get(lastSelected) ?? -1;
      const currentIndex = recordIndexMap.get(recordId) ?? -1;
      if (lastIndex === -1 || currentIndex === -1) return;
      const start = Math.min(lastIndex, currentIndex);
      const end = Math.max(lastIndex, currentIndex);
      for (let i = start; i <= end; i++) {
        newSelection.add(records[i].id);
      }
    } else if (e.metaKey || e.ctrlKey) {
      if (newSelection.has(recordId)) {
        newSelection.delete(recordId);
      } else {
        newSelection.add(recordId);
      }
    } else {
      if (newSelection.has(recordId) && newSelection.size === 1) {
        newSelection.delete(recordId);
      } else {
        newSelection.clear();
        newSelection.add(recordId);
      }
    }
    setSelectedRecords(newSelection);
  };

  const toggleAll = () => {
    if (selectedRecords.size === records.length) {
      setSelectedRecords(new Set());
    } else {
      setSelectedRecords(new Set(records.map(r => r.id)));
    }
  };

  // Set up virtualizer for large lists
  const rowVirtualizer = useVirtualizer({
    count: records.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 72, // Estimated row height
    overscan: 5,
  });

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

  const startEditing = (record: TableRecord, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!primaryField) return;
    setEditingRecord(record.id);
    setEditValue((record.values[primaryField.id] as string) || '');
  };

  const saveEdit = async () => {
    if (editingRecord && primaryField) {
      await updateCellValue(editingRecord, primaryField.id, editValue);
    }
    setEditingRecord(null);
    setEditValue('');
  };

  const cancelEdit = () => {
    setEditingRecord(null);
    setEditValue('');
  };

  const formatCellValue = (value: CellValue, field: Field): string => {
    if (value === null || value === undefined) return '';

    switch (field.type) {
      case 'checkbox':
        return value ? '✓' : '';
      case 'single_select':
        const choice = field.options?.choices?.find(c => c.id === value);
        return choice?.name || '';
      case 'multi_select':
        const values = value as string[];
        return values
          .map(v => field.options?.choices?.find(c => c.id === v)?.name || v)
          .join(', ');
      case 'date':
      case 'datetime':
        return new Date(value as string).toLocaleDateString();
      case 'rating':
        return '★'.repeat(value as number);
      case 'currency':
        return `$${(value as number).toFixed(2)}`;
      case 'percent':
        return `${value}%`;
      default:
        return String(value);
    }
  };

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Toolbar */}
      <div className="view-toolbar">
        <div className="view-toolbar-left">
          {selectedRecords.size > 0 && (
            <>
              <span className="text-sm text-[var(--at-text-secondary)]">
                {selectedRecords.size} selected
              </span>
              <button
                onClick={handleDeleteSelected}
                className="btn btn-danger text-sm"
              >
                Delete
              </button>
            </>
          )}
        </div>
        <div className="view-toolbar-count">
          {records.length} record{records.length !== 1 ? 's' : ''}
        </div>
      </div>

      {/* List */}
      <div ref={parentRef} className="flex-1 overflow-auto">
        {/* Select all header */}
        <div className="sticky top-0 bg-[var(--at-surface-muted)] border-b border-[var(--at-border)] px-4 py-2 flex items-center gap-3 z-10">
          <input
            type="checkbox"
            checked={selectedRecords.size === records.length && records.length > 0}
            onChange={toggleAll}
            className="checkbox"
          />
          <span className="text-sm font-semibold text-[var(--at-text)]">
            {primaryField?.name || 'Name'}
          </span>
        </div>

        {/* Records - virtualized for large lists */}
        {useVirtualization ? (
          <div
            style={{
              height: `${rowVirtualizer.getTotalSize()}px`,
              width: '100%',
              position: 'relative',
            }}
          >
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const record = records[virtualRow.index];
              const isSelected = selectedRecords.has(record.id);
              const isEditing = editingRecord === record.id;
              const primaryValue = primaryField
                ? (record.values[primaryField.id] as string) || 'Untitled'
                : 'Untitled';

              return (
                <ListRowContent
                  key={record.id}
                  record={record}
                  isSelected={isSelected}
                  isEditing={isEditing}
                  primaryValue={primaryValue}
                  primaryField={primaryField}
                  secondaryFields={secondaryFields}
                  editValue={editValue}
                  setEditValue={setEditValue}
                  saveEdit={saveEdit}
                  cancelEdit={cancelEdit}
                  startEditing={startEditing}
                  toggleSelection={toggleSelection}
                  setExpandedRecord={setExpandedRecord}
                  formatCellValue={formatCellValue}
                  style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    width: '100%',
                    height: `${virtualRow.size}px`,
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                />
              );
            })}
          </div>
        ) : (
          <div className="divide-y divide-[var(--at-border)]">
            {records.map((record) => {
              const isSelected = selectedRecords.has(record.id);
              const isEditing = editingRecord === record.id;
              const primaryValue = primaryField
                ? (record.values[primaryField.id] as string) || 'Untitled'
                : 'Untitled';

              return (
                <ListRowContent
                  key={record.id}
                  record={record}
                  isSelected={isSelected}
                  isEditing={isEditing}
                  primaryValue={primaryValue}
                  primaryField={primaryField}
                  secondaryFields={secondaryFields}
                  editValue={editValue}
                  setEditValue={setEditValue}
                  saveEdit={saveEdit}
                  cancelEdit={cancelEdit}
                  startEditing={startEditing}
                  toggleSelection={toggleSelection}
                  setExpandedRecord={setExpandedRecord}
                  formatCellValue={formatCellValue}
                />
              );
            })}
          </div>
        )}

        {/* Add record */}
        <button
          onClick={handleAddRecord}
          className="w-full px-4 py-3 text-left text-sm text-[var(--at-text-secondary)] hover:bg-[var(--at-surface-hover)] flex items-center gap-2 transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add record
        </button>

        {/* Empty state */}
        {records.length === 0 && (
          <div className="flex items-center justify-center h-64 py-12">
            <div className="empty-state animate-fade-in">
              <div className="empty-state-icon-wrapper">
                <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                </svg>
              </div>
              <h3 className="empty-state-title">No records yet</h3>
              <p className="empty-state-description">Create your first record to get started</p>
              <button onClick={handleAddRecord} className="btn btn-primary mt-2">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Add your first record
              </button>
            </div>
          </div>
        )}
      </div>

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

// Memoized row component to prevent unnecessary re-renders
interface ListRowContentProps {
  record: TableRecord;
  isSelected: boolean;
  isEditing: boolean;
  primaryValue: string;
  primaryField: Field | undefined;
  secondaryFields: Field[];
  editValue: string;
  setEditValue: (value: string) => void;
  saveEdit: () => void;
  cancelEdit: () => void;
  startEditing: (record: TableRecord, e: React.MouseEvent) => void;
  toggleSelection: (recordId: string, e: React.MouseEvent) => void;
  setExpandedRecord: (record: TableRecord) => void;
  formatCellValue: (value: CellValue, field: Field) => string;
  style?: React.CSSProperties;
}

const ListRowContent = memo(function ListRowContent({
  record,
  isSelected,
  isEditing,
  primaryValue,
  primaryField: _primaryField, // Used for prop type consistency, actual display uses primaryValue
  secondaryFields,
  editValue,
  setEditValue,
  saveEdit,
  cancelEdit,
  startEditing,
  toggleSelection,
  setExpandedRecord,
  formatCellValue,
  style,
}: ListRowContentProps) {
  void _primaryField; // Suppress unused warning - kept for API consistency
  return (
    <div
      className={`px-4 py-3 flex items-start gap-3 cursor-pointer hover:bg-[var(--at-surface-hover)] border-b border-[var(--at-border)] ${
        isSelected ? 'bg-[var(--at-primary-soft)]' : ''
      }`}
      style={style}
      onClick={() => setExpandedRecord(record)}
    >
      {/* Checkbox */}
      <input
        type="checkbox"
        checked={isSelected}
        onChange={(e) => toggleSelection(record.id, e as unknown as React.MouseEvent)}
        onClick={(e) => e.stopPropagation()}
        className="checkbox mt-1"
      />

      {/* Content */}
      <div className="flex-1 min-w-0">
        {/* Primary field */}
        {isEditing ? (
          <div className="flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
            <input
              type="text"
              value={editValue}
              onChange={(e) => setEditValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') saveEdit();
                if (e.key === 'Escape') cancelEdit();
              }}
              className="input flex-1"
              autoFocus
            />
            <button onClick={saveEdit} className="btn btn-primary btn-sm">
              Save
            </button>
            <button onClick={cancelEdit} className="btn btn-secondary btn-sm">
              Cancel
            </button>
          </div>
        ) : (
          <div
            className="font-semibold text-[var(--at-text)] truncate hover:text-primary"
            onDoubleClick={(e) => startEditing(record, e)}
          >
            {primaryValue}
          </div>
        )}

        {/* Secondary fields */}
        {secondaryFields.length > 0 && !isEditing && (
          <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1">
            {secondaryFields.map(field => {
              const value = record.values[field.id];
              if (value === null || value === undefined) return null;

              return (
                <div key={field.id} className="text-sm text-[var(--at-text-secondary)]">
                  <span className="text-[var(--at-muted)]">{field.name}: </span>
                  {field.type === 'single_select' ? (
                    <SelectBadge field={field} value={value as string} />
                  ) : (
                    formatCellValue(value, field)
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Expand button */}
      <button
        onClick={(e) => {
          e.stopPropagation();
          setExpandedRecord(record);
        }}
        className="text-[var(--at-muted)] hover:text-[var(--at-text)] p-1 transition-colors"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
        </svg>
      </button>
    </div>
  );
});

function SelectBadge({ field, value }: { field: Field; value: string }) {
  const choice = field.options?.choices?.find(c => c.id === value);
  if (!choice) return <span>{value}</span>;

  return (
    <span
      className="inline-flex px-2 py-0.5 rounded text-xs font-medium"
      style={{ backgroundColor: choice.color + '20', color: choice.color }}
    >
      {choice.name}
    </span>
  );
}
