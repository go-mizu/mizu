import { useState } from 'react';
import { useBaseStore } from '../../stores/baseStore';
import type { Sort, FieldType } from '../../types';

interface SortBuilderProps {
  onClose: () => void;
}

const SORTABLE_TYPES: FieldType[] = [
  'text', 'long_text', 'number', 'currency', 'percent', 'single_select',
  'date', 'datetime', 'checkbox', 'rating', 'duration', 'phone', 'email',
  'url', 'created_time', 'last_modified_time', 'autonumber',
];

export function SortBuilder({ onClose }: SortBuilderProps) {
  const { fields, sorts, setSorts } = useBaseStore();
  const [localSorts, setLocalSorts] = useState<Sort[]>(sorts.length > 0 ? [...sorts] : []);

  const sortableFields = fields.filter(f => SORTABLE_TYPES.includes(f.type));

  const addSort = () => {
    if (sortableFields.length === 0) return;

    // Find a field that isn't already being sorted
    const usedFieldIds = localSorts.map(s => s.field_id);
    const availableField = sortableFields.find(f => !usedFieldIds.includes(f.id)) || sortableFields[0];

    setLocalSorts([
      ...localSorts,
      {
        field_id: availableField.id,
        direction: 'asc',
      },
    ]);
  };

  const updateSort = (index: number, updates: Partial<Sort>) => {
    setLocalSorts(localSorts.map((s, i) => (i === index ? { ...s, ...updates } : s)));
  };

  const removeSort = (index: number) => {
    setLocalSorts(localSorts.filter((_, i) => i !== index));
  };

  const moveSort = (index: number, direction: 'up' | 'down') => {
    const newIndex = direction === 'up' ? index - 1 : index + 1;
    if (newIndex < 0 || newIndex >= localSorts.length) return;

    const newSorts = [...localSorts];
    const [removed] = newSorts.splice(index, 1);
    newSorts.splice(newIndex, 0, removed);
    setLocalSorts(newSorts);
  };

  const applySorts = () => {
    setSorts(localSorts);
    onClose();
  };

  const clearSorts = () => {
    setLocalSorts([]);
    setSorts([]);
    onClose();
  };

  return (
    <div className="p-4 min-w-[350px]">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-medium text-gray-900">Sort records</h3>
        <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div className="space-y-3">
        {localSorts.map((sort, index) => {
          const field = fields.find(f => f.id === sort.field_id);

          return (
            <div key={index} className="flex items-center gap-2">
              {/* Reorder buttons */}
              <div className="flex flex-col">
                <button
                  onClick={() => moveSort(index, 'up')}
                  disabled={index === 0}
                  className="text-gray-400 hover:text-gray-600 disabled:opacity-30"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
                  </svg>
                </button>
                <button
                  onClick={() => moveSort(index, 'down')}
                  disabled={index === localSorts.length - 1}
                  className="text-gray-400 hover:text-gray-600 disabled:opacity-30"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </button>
              </div>

              {/* Field selector */}
              <select
                value={sort.field_id}
                onChange={(e) => updateSort(index, { field_id: e.target.value })}
                className="input py-1 flex-1"
              >
                {sortableFields.map((f) => (
                  <option key={f.id} value={f.id}>{f.name}</option>
                ))}
              </select>

              {/* Direction selector */}
              <select
                value={sort.direction}
                onChange={(e) => updateSort(index, { direction: e.target.value as 'asc' | 'desc' })}
                className="input py-1 w-32"
              >
                <option value="asc">
                  {field?.type === 'text' || field?.type === 'long_text' ? 'A → Z' : 'Low → High'}
                </option>
                <option value="desc">
                  {field?.type === 'text' || field?.type === 'long_text' ? 'Z → A' : 'High → Low'}
                </option>
              </select>

              {/* Remove button */}
              <button
                onClick={() => removeSort(index)}
                className="p-1 text-gray-400 hover:text-gray-600"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          );
        })}
      </div>

      {/* Add sort button */}
      <button
        onClick={addSort}
        className="mt-3 text-sm text-primary hover:text-primary-600 flex items-center gap-1"
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
        Add sort
      </button>

      {/* Action buttons */}
      <div className="mt-4 pt-3 border-t border-gray-200 flex items-center justify-between">
        <button
          onClick={clearSorts}
          className="text-sm text-gray-600 hover:text-gray-800"
        >
          Clear all
        </button>
        <button
          onClick={applySorts}
          className="btn btn-primary py-1.5 px-4 text-sm"
        >
          Apply sorts
        </button>
      </div>
    </div>
  );
}
