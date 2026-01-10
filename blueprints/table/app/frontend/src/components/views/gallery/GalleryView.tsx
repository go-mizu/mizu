import { useState, useMemo } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { TableRecord, Field } from '../../../types';
import { RecordModal } from '../RecordModal';

export function GalleryView() {
  const { currentView, fields, records, createRecord } = useBaseStore();
  const [expandedRecord, setExpandedRecord] = useState<TableRecord | null>(null);

  // Get the cover field (first attachment field or configured)
  const coverField = useMemo(() => {
    const coverFieldId = currentView?.config?.coverField;
    if (coverFieldId) {
      return fields.find(f => f.id === coverFieldId);
    }
    return fields.find(f => f.type === 'attachment');
  }, [fields, currentView?.config?.coverField]);

  // Get primary field for display
  const primaryField = fields.find(f => f.type === 'text') || fields[0];

  // Get fields to show on cards (exclude cover and primary)
  const cardFields = useMemo(() => {
    return fields.filter(f =>
      f.id !== coverField?.id &&
      f.id !== primaryField?.id &&
      !['attachment', 'long_text'].includes(f.type)
    ).slice(0, 4);
  }, [fields, coverField, primaryField]);

  const getRecordTitle = (record: TableRecord): string => {
    if (!primaryField) return 'Untitled';
    const value = record.values[primaryField.id];
    return value ? String(value) : 'Untitled';
  };

  const getCoverImage = (record: TableRecord): string | null => {
    if (!coverField) return null;
    const attachments = record.values[coverField.id] as { url: string }[] | undefined;
    return attachments?.[0]?.url || null;
  };

  const handleAddRecord = async () => {
    await createRecord({});
  };

  return (
    <div className="flex-1 p-4 overflow-auto">
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
        {records.map((record) => {
          const coverImage = getCoverImage(record);

          return (
            <div
              key={record.id}
              onClick={() => setExpandedRecord(record)}
              className="bg-white rounded-lg border border-gray-200 overflow-hidden cursor-pointer hover:shadow-lg transition-shadow"
            >
              {/* Cover image */}
              <div className="aspect-video bg-gray-100 relative">
                {coverImage ? (
                  <img
                    src={coverImage}
                    alt=""
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <div className="w-full h-full flex items-center justify-center">
                    <svg className="w-12 h-12 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                    </svg>
                  </div>
                )}
              </div>

              {/* Card content */}
              <div className="p-3">
                <h3 className="font-medium text-gray-900 mb-2 truncate">
                  {getRecordTitle(record)}
                </h3>

                <div className="space-y-1">
                  {cardFields.map((field) => {
                    const value = record.values[field.id];
                    if (value === null || value === undefined) return null;

                    return (
                      <div key={field.id} className="flex items-start gap-2 text-sm">
                        <span className="text-gray-500 flex-shrink-0">{field.name}:</span>
                        <span className="text-gray-900 truncate">
                          {renderFieldValue(field, value)}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          );
        })}

        {/* Add new card */}
        <button
          onClick={handleAddRecord}
          className="bg-gray-50 rounded-lg border-2 border-dashed border-gray-200 flex items-center justify-center min-h-[200px] hover:border-gray-300 hover:bg-gray-100 transition-colors"
        >
          <div className="text-center">
            <svg className="w-8 h-8 mx-auto text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            <span className="text-sm text-gray-500">Add record</span>
          </div>
        </button>
      </div>

      {records.length === 0 && (
        <div className="flex-1 flex items-center justify-center text-gray-500 min-h-[400px]">
          <div className="text-center">
            <svg className="w-12 h-12 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <h3 className="text-lg font-medium text-gray-900 mb-1">No records</h3>
            <p className="text-sm text-gray-500 mb-4">Create your first record to see it here</p>
            <button onClick={handleAddRecord} className="btn btn-primary">
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
    case 'single_select':
    case 'multi_select':
      if (Array.isArray(value)) {
        return value.length + ' selected';
      }
      return String(value);
    default:
      return String(value);
  }
}
