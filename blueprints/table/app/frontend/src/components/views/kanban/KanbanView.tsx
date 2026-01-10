import { useState, useMemo } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { TableRecord, Field } from '../../../types';
import { RecordModal } from '../RecordModal';

interface KanbanColumn {
  id: string;
  name: string;
  color: string;
  records: TableRecord[];
}

export function KanbanView() {
  const { currentView, fields, records, createRecord, updateCellValue } = useBaseStore();
  const [expandedRecord, setExpandedRecord] = useState<TableRecord | null>(null);
  const [draggedRecord, setDraggedRecord] = useState<TableRecord | null>(null);
  const [dragOverColumn, setDragOverColumn] = useState<string | null>(null);

  // Get the field used for grouping (first single_select field or configured in view)
  const groupByField = useMemo(() => {
    const groupFieldId = currentView?.config?.groupBy;
    if (groupFieldId) {
      return fields.find(f => f.id === groupFieldId);
    }
    return fields.find(f => f.type === 'single_select');
  }, [fields, currentView?.config?.groupBy]);

  // Build columns from the groupBy field options
  const columns = useMemo((): KanbanColumn[] => {
    if (!groupByField) {
      return [{ id: '__all__', name: 'All Records', color: '#6b7280', records }];
    }

    const choices = groupByField.options?.choices || [];
    const columnMap = new Map<string, KanbanColumn>();

    // Create column for uncategorized
    columnMap.set('__uncategorized__', {
      id: '__uncategorized__',
      name: 'Uncategorized',
      color: '#6b7280',
      records: [],
    });

    // Create columns for each choice
    choices.forEach((choice: { id: string; name: string; color: string }) => {
      columnMap.set(choice.id, {
        id: choice.id,
        name: choice.name,
        color: choice.color,
        records: [],
      });
    });

    // Assign records to columns
    records.forEach((record) => {
      const value = record.values[groupByField.id] as string | undefined;
      if (value && columnMap.has(value)) {
        columnMap.get(value)!.records.push(record);
      } else {
        columnMap.get('__uncategorized__')!.records.push(record);
      }
    });

    // Return columns in order: choices first, then uncategorized
    const result: KanbanColumn[] = choices.map((choice: { id: string }) => columnMap.get(choice.id)!);
    const uncategorized = columnMap.get('__uncategorized__')!;
    if (uncategorized.records.length > 0 || choices.length === 0) {
      result.unshift(uncategorized);
    }

    return result;
  }, [groupByField, records]);

  const handleDragStart = (record: TableRecord) => {
    setDraggedRecord(record);
  };

  const handleDragOver = (e: React.DragEvent, columnId: string) => {
    e.preventDefault();
    setDragOverColumn(columnId);
  };

  const handleDragLeave = () => {
    setDragOverColumn(null);
  };

  const handleDrop = async (columnId: string) => {
    if (!draggedRecord || !groupByField) return;

    const newValue = columnId === '__uncategorized__' ? null : columnId;
    await updateCellValue(draggedRecord.id, groupByField.id, newValue);

    setDraggedRecord(null);
    setDragOverColumn(null);
  };

  const handleAddCard = async (columnId: string) => {
    if (!groupByField) {
      await createRecord({});
      return;
    }

    const initialValues = columnId !== '__uncategorized__' ? { [groupByField.id]: columnId } : {};
    await createRecord(initialValues);
  };

  // Get primary field for display (first text field)
  const primaryField = fields.find(f => f.type === 'text') || fields[0];

  const getRecordTitle = (record: TableRecord): string => {
    if (!primaryField) return 'Untitled';
    const value = record.values[primaryField.id];
    return value ? String(value) : 'Untitled';
  };

  if (!groupByField) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500">
        <div className="text-center">
          <svg className="w-12 h-12 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7m0 10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2h-2a2 2 0 00-2 2" />
          </svg>
          <h3 className="text-lg font-medium text-gray-900 mb-1">No grouping field</h3>
          <p className="text-sm text-gray-500">Add a Single Select field to use Kanban view</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-x-auto p-4">
      <div className="flex gap-4 min-h-full">
        {columns.map((column) => (
          <div
            key={column.id}
            className={`flex-shrink-0 w-72 bg-slate-50 rounded-xl border border-slate-200 flex flex-col ${
              dragOverColumn === column.id ? 'ring-2 ring-primary' : ''
            }`}
            onDragOver={(e) => handleDragOver(e, column.id)}
            onDragLeave={handleDragLeave}
            onDrop={() => handleDrop(column.id)}
          >
            {/* Column header */}
            <div className="p-3 flex items-center gap-2 border-b border-slate-200">
              <span
                className="w-3 h-3 rounded-full"
                style={{ backgroundColor: column.color }}
              />
              <span className="font-semibold text-gray-900">{column.name}</span>
              <span className="text-xs text-slate-500 ml-auto bg-white border border-slate-200 rounded-full px-2 py-0.5">
                {column.records.length}
              </span>
            </div>

            {/* Cards */}
            <div className="flex-1 overflow-y-auto p-2 space-y-2">
              {column.records.map((record) => (
                <div
                  key={record.id}
                  draggable
                  onDragStart={() => handleDragStart(record)}
                  onClick={() => setExpandedRecord(record)}
                  className={`bg-white rounded-lg shadow-sm border border-slate-200 p-3 cursor-pointer hover:shadow-md transition-shadow ${
                    draggedRecord?.id === record.id ? 'opacity-50' : ''
                  }`}
                >
                  <h4 className="text-sm font-medium text-gray-900 mb-2">
                    {getRecordTitle(record)}
                  </h4>

                  {/* Show a few other fields */}
                  <div className="space-y-1">
                    {fields.slice(0, 3).map((field) => {
                      if (field.id === primaryField?.id || field.id === groupByField.id) return null;
                      const value = record.values[field.id];
                      if (!value) return null;

                      return (
                        <div key={field.id} className="text-xs text-gray-500 truncate">
                          <span className="font-medium">{field.name}:</span>{' '}
                          {renderFieldValue(field, value)}
                        </div>
                      );
                    })}
                  </div>
                </div>
              ))}

              {/* Add card button */}
              <button
                onClick={() => handleAddCard(column.id)}
                className="w-full p-2 text-sm text-slate-500 hover:bg-white rounded-lg flex items-center justify-center gap-1 transition-colors border border-transparent hover:border-slate-200"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Add card
              </button>
            </div>
          </div>
        ))}
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

function renderFieldValue(field: Field, value: any): string {
  switch (field.type) {
    case 'checkbox':
      return value ? 'Yes' : 'No';
    case 'date':
    case 'datetime':
      return new Date(value).toLocaleDateString();
    case 'number':
      return value.toLocaleString();
    case 'currency':
      return `$${value.toLocaleString()}`;
    case 'percent':
      return `${value}%`;
    case 'rating':
      return '⭐'.repeat(value);
    default:
      return String(value);
  }
}
