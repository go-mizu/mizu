import { useState, useMemo } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { TableRecord, Field, CellValue } from '../../../types';
import { RecordSidebar } from '../RecordSidebar';

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

  const toggleSelection = (recordId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const newSelection = new Set(selectedRecords);
    if (e.shiftKey && selectedRecords.size > 0) {
      const lastSelected = Array.from(selectedRecords).pop()!;
      const lastIndex = records.findIndex(r => r.id === lastSelected);
      const currentIndex = records.findIndex(r => r.id === recordId);
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
      <div className="flex items-center justify-between p-4 border-b border-slate-200 bg-white">
        <div className="flex items-center gap-2">
          {selectedRecords.size > 0 && (
            <>
              <span className="text-sm text-gray-600">
                {selectedRecords.size} selected
              </span>
              <button
                onClick={handleDeleteSelected}
                className="text-sm text-danger hover:text-danger-600"
              >
                Delete
              </button>
            </>
          )}
        </div>
        <div className="text-sm text-gray-500">
          {records.length} record{records.length !== 1 ? 's' : ''}
        </div>
      </div>

      {/* List */}
      <div className="flex-1 overflow-auto">
        {/* Select all header */}
        <div className="sticky top-0 bg-slate-50 border-b border-slate-200 px-4 py-2 flex items-center gap-3">
          <input
            type="checkbox"
            checked={selectedRecords.size === records.length && records.length > 0}
            onChange={toggleAll}
            className="w-4 h-4 rounded border-gray-300"
          />
          <span className="text-sm font-semibold text-slate-700">
            {primaryField?.name || 'Name'}
          </span>
        </div>

        {/* Records */}
        <div className="divide-y divide-gray-100">
          {records.map((record) => {
            const isSelected = selectedRecords.has(record.id);
            const isEditing = editingRecord === record.id;
            const primaryValue = primaryField
              ? (record.values[primaryField.id] as string) || 'Untitled'
              : 'Untitled';

            return (
              <div
                key={record.id}
                className={`px-4 py-3 flex items-start gap-3 cursor-pointer hover:bg-slate-50 ${
                  isSelected ? 'bg-primary-50' : ''
                }`}
                onClick={() => setExpandedRecord(record)}
              >
                {/* Checkbox */}
                <input
                  type="checkbox"
                  checked={isSelected}
                  onChange={(e) => toggleSelection(record.id, e as any)}
                  onClick={(e) => e.stopPropagation()}
                  className="w-4 h-4 rounded border-gray-300 mt-1"
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
                      className="font-semibold text-gray-900 truncate hover:text-primary"
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
                        <div key={field.id} className="text-sm text-gray-500">
                          <span className="text-slate-400">{field.name}: </span>
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
                  className="text-gray-400 hover:text-gray-600 p-1"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
                  </svg>
                </button>
              </div>
            );
          })}
        </div>

        {/* Add record */}
        <button
          onClick={handleAddRecord}
          className="w-full px-4 py-3 text-left text-sm text-gray-500 hover:bg-gray-50 flex items-center gap-2"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add record
        </button>

        {/* Empty state */}
        {records.length === 0 && (
          <div className="flex items-center justify-center h-64 text-gray-500">
            <div className="text-center">
              <svg className="w-12 h-12 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
              <p>No records yet</p>
              <button
                onClick={handleAddRecord}
                className="mt-2 btn btn-primary"
              >
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
