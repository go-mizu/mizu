import { useState, useEffect, useRef, useCallback } from 'react';
import type { TableRecord, Field, Table } from '../../../types';
import { useBaseStore } from '../../../stores/baseStore';
import { recordsApi, tablesApi } from '../../../api/client';

interface LinkedRecordPickerProps {
  field: Field;
  selectedIds: string[];
  onChange: (ids: string[]) => void;
  onClose: () => void;
  position?: { x: number; y: number };
}

export function LinkedRecordPicker({
  field,
  selectedIds,
  onChange,
  onClose,
  position,
}: LinkedRecordPickerProps) {
  const { tables } = useBaseStore();
  const [search, setSearch] = useState('');
  const [records, setRecords] = useState<TableRecord[]>([]);
  const [linkedTable, setLinkedTable] = useState<Table | null>(null);
  const [primaryField, setPrimaryField] = useState<Field | null>(null);
  const [loading, setLoading] = useState(true);
  const [highlightedIndex, setHighlightedIndex] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const linkedTableId = field.options?.linked_table_id;

  // Fetch linked table info and records
  useEffect(() => {
    if (!linkedTableId) return;

    const loadData = async () => {
      setLoading(true);
      try {
        // Find the linked table from cache or fetch it
        let table = tables.find((t) => t.id === linkedTableId);
        let tableFields: Field[] | undefined;
        if (!table) {
          const result = await tablesApi.get(linkedTableId);
          table = result.table;
          tableFields = result.fields;
        } else {
          tableFields = table.fields;
        }
        if (table) {
          setLinkedTable(table);
          // Find primary field
          const primary = tableFields?.find((f: Field) => f.is_primary);
          setPrimaryField(primary || null);
        }

        // Fetch records from linked table
        const result = await recordsApi.list(linkedTableId);
        setRecords(result.records || []);
      } catch (error) {
        console.error('Failed to load linked records:', error);
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [linkedTableId, tables]);

  // Filter records by search
  const filteredRecords = records.filter((record) => {
    if (!search.trim()) return true;
    const searchLower = search.toLowerCase();

    // Search in all text values
    return Object.values(record.values).some((value) => {
      if (typeof value === 'string') {
        return value.toLowerCase().includes(searchLower);
      }
      return false;
    });
  });

  // Get primary value for a record
  const getPrimaryValue = useCallback(
    (record: TableRecord): string => {
      if (primaryField) {
        const value = record.values[primaryField.id];
        if (typeof value === 'string') return value;
        if (value !== null && value !== undefined) return String(value);
      }

      // Fallback to first string value
      for (const value of Object.values(record.values)) {
        if (typeof value === 'string' && value) return value;
      }

      return `Record ${record.id.slice(-6)}`;
    },
    [primaryField]
  );

  // Handle keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setHighlightedIndex((prev) =>
          Math.min(prev + 1, filteredRecords.length - 1)
        );
        break;
      case 'ArrowUp':
        e.preventDefault();
        setHighlightedIndex((prev) => Math.max(prev - 1, 0));
        break;
      case 'Enter':
        e.preventDefault();
        if (filteredRecords[highlightedIndex]) {
          toggleRecord(filteredRecords[highlightedIndex].id);
        }
        break;
      case 'Escape':
        e.preventDefault();
        onClose();
        break;
    }
  };

  // Toggle record selection
  const toggleRecord = (recordId: string) => {
    const newIds = selectedIds.includes(recordId)
      ? selectedIds.filter((id) => id !== recordId)
      : [...selectedIds, recordId];
    onChange(newIds);
  };

  // Focus input on mount
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  // Handle click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        onClose();
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [onClose]);

  // Reset highlighted index when search changes
  useEffect(() => {
    setHighlightedIndex(0);
  }, [search]);

  const positionStyle = position
    ? { left: position.x, top: position.y }
    : {};

  return (
    <div
      ref={containerRef}
      className="fixed z-50 bg-white rounded-lg shadow-xl border border-slate-200 w-80 max-h-[400px] flex flex-col"
      style={positionStyle}
      onKeyDown={handleKeyDown}
    >
      {/* Header */}
      <div className="p-3 border-b border-slate-200">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm font-medium text-slate-700">
            Link to {linkedTable?.name || 'records'}
          </span>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Search input */}
        <div className="relative">
          <svg
            className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            ref={inputRef}
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Find a record..."
            className="w-full pl-8 pr-3 py-2 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
          />
        </div>
      </div>

      {/* Records list */}
      <div className="flex-1 overflow-auto">
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          </div>
        ) : filteredRecords.length === 0 ? (
          <div className="py-8 text-center text-sm text-slate-500">
            {search ? 'No records found' : 'No records available'}
          </div>
        ) : (
          <div className="py-1">
            {filteredRecords.map((record, index) => {
              const isSelected = selectedIds.includes(record.id);
              const isHighlighted = index === highlightedIndex;
              const primaryValue = getPrimaryValue(record);

              return (
                <button
                  key={record.id}
                  onClick={() => toggleRecord(record.id)}
                  onMouseEnter={() => setHighlightedIndex(index)}
                  className={`w-full px-3 py-2 flex items-center gap-2 text-left text-sm transition-colors ${
                    isHighlighted ? 'bg-slate-100' : ''
                  } ${isSelected ? 'bg-primary-50' : ''} hover:bg-slate-100`}
                >
                  {/* Checkbox */}
                  <div
                    className={`w-4 h-4 rounded border flex items-center justify-center flex-shrink-0 ${
                      isSelected
                        ? 'bg-primary border-primary text-white'
                        : 'border-slate-300'
                    }`}
                  >
                    {isSelected && (
                      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                      </svg>
                    )}
                  </div>

                  {/* Record primary value */}
                  <span className="truncate text-slate-700">{primaryValue}</span>
                </button>
              );
            })}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="p-2 border-t border-slate-200 flex items-center justify-between">
        <button
          onClick={() => {
            // TODO: Create new record in linked table
            console.log('Create new record');
          }}
          className="text-sm text-primary hover:text-primary-dark flex items-center gap-1"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Create new record
        </button>

        {selectedIds.length > 0 && (
          <span className="text-xs text-slate-500">
            {selectedIds.length} selected
          </span>
        )}
      </div>
    </div>
  );
}
