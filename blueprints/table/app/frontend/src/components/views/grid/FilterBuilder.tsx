import { useState, useRef, useEffect } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { Field, Filter, FieldType } from '../../../types';

interface FilterBuilderProps {
  isOpen: boolean;
  onClose: () => void;
  anchorRef?: React.RefObject<HTMLElement | null>;
}

// Operators per field type
const OPERATORS: Record<string, { value: string; label: string }[]> = {
  text: [
    { value: 'is', label: 'is' },
    { value: 'is_not', label: 'is not' },
    { value: 'contains', label: 'contains' },
    { value: 'not_contains', label: 'does not contain' },
    { value: 'is_empty', label: 'is empty' },
    { value: 'is_not_empty', label: 'is not empty' },
    { value: 'starts_with', label: 'starts with' },
    { value: 'ends_with', label: 'ends with' },
  ],
  number: [
    { value: '=', label: '=' },
    { value: '!=', label: '!=' },
    { value: '<', label: '<' },
    { value: '<=', label: '<=' },
    { value: '>', label: '>' },
    { value: '>=', label: '>=' },
    { value: 'is_empty', label: 'is empty' },
    { value: 'is_not_empty', label: 'is not empty' },
  ],
  date: [
    { value: 'is', label: 'is' },
    { value: 'is_not', label: 'is not' },
    { value: 'is_before', label: 'is before' },
    { value: 'is_after', label: 'is after' },
    { value: 'is_on_or_before', label: 'is on or before' },
    { value: 'is_on_or_after', label: 'is on or after' },
    { value: 'is_within', label: 'is within' },
    { value: 'is_empty', label: 'is empty' },
    { value: 'is_not_empty', label: 'is not empty' },
  ],
  select: [
    { value: 'is', label: 'is' },
    { value: 'is_not', label: 'is not' },
    { value: 'is_any_of', label: 'is any of' },
    { value: 'is_none_of', label: 'is none of' },
    { value: 'is_empty', label: 'is empty' },
    { value: 'is_not_empty', label: 'is not empty' },
  ],
  checkbox: [
    { value: 'is_checked', label: 'is checked' },
    { value: 'is_unchecked', label: 'is unchecked' },
  ],
  user: [
    { value: 'is', label: 'is' },
    { value: 'is_not', label: 'is not' },
    { value: 'is_any_of', label: 'is any of' },
    { value: 'is_empty', label: 'is empty' },
    { value: 'is_not_empty', label: 'is not empty' },
  ],
  attachment: [
    { value: 'is_empty', label: 'has no attachments' },
    { value: 'is_not_empty', label: 'has attachments' },
    { value: 'filename_contains', label: 'filename contains' },
  ],
};

// Map field types to operator categories
function getOperatorCategory(fieldType: FieldType): string {
  switch (fieldType) {
    case 'text':
    case 'single_line_text':
    case 'long_text':
    case 'rich_text':
    case 'email':
    case 'url':
    case 'phone':
    case 'barcode':
      return 'text';
    case 'number':
    case 'currency':
    case 'percent':
    case 'rating':
    case 'duration':
    case 'autonumber':
      return 'number';
    case 'date':
    case 'datetime':
    case 'created_time':
    case 'last_modified_time':
      return 'date';
    case 'single_select':
    case 'multi_select':
      return 'select';
    case 'checkbox':
      return 'checkbox';
    case 'user':
    case 'collaborator':
    case 'created_by':
    case 'last_modified_by':
      return 'user';
    case 'attachment':
      return 'attachment';
    default:
      return 'text';
  }
}

// Check if operator needs a value input
function operatorNeedsValue(operator: string): boolean {
  return !['is_empty', 'is_not_empty', 'is_checked', 'is_unchecked'].includes(operator);
}

export function FilterBuilder({ isOpen, onClose, anchorRef }: FilterBuilderProps) {
  const { fields, currentView, setFilters } = useBaseStore();
  const [localFilters, setLocalFilters] = useState<Filter[]>([]);
  const [conjunction, setConjunction] = useState<'and' | 'or'>('and');
  const panelRef = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState({ top: 0, left: 0 });

  // Initialize filters from view
  useEffect(() => {
    if (currentView?.filters) {
      setLocalFilters(currentView.filters.length > 0 ? currentView.filters : []);
    } else {
      setLocalFilters([]);
    }
  }, [currentView?.filters, isOpen]);

  // Position panel below anchor
  useEffect(() => {
    if (isOpen && anchorRef?.current) {
      const rect = anchorRef.current.getBoundingClientRect();
      setPosition({
        top: rect.bottom + 4,
        left: Math.max(8, rect.left),
      });
    }
  }, [isOpen, anchorRef]);

  // Close on outside click
  useEffect(() => {
    if (!isOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node) &&
          anchorRef?.current && !anchorRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [isOpen, onClose, anchorRef]);

  const filterableFields = fields.filter(f =>
    !['button', 'formula', 'rollup', 'count', 'lookup', 'link'].includes(f.type)
  );

  const addFilter = () => {
    const firstField = filterableFields[0];
    if (!firstField) return;

    const category = getOperatorCategory(firstField.type);
    const operators = OPERATORS[category] || OPERATORS.text;

    setLocalFilters([
      ...localFilters,
      {
        field_id: firstField.id,
        operator: operators[0].value,
        value: '',
      },
    ]);
  };

  const updateFilter = (index: number, updates: Partial<Filter>) => {
    const newFilters = [...localFilters];
    newFilters[index] = { ...newFilters[index], ...updates };

    // Reset value when field or operator changes
    if (updates.field_id || updates.operator) {
      const field = fields.find(f => f.id === (updates.field_id || newFilters[index].field_id));
      if (field && updates.field_id) {
        const category = getOperatorCategory(field.type);
        const operators = OPERATORS[category] || OPERATORS.text;
        newFilters[index].operator = operators[0].value;
        newFilters[index].value = '';
      }
    }

    setLocalFilters(newFilters);
  };

  const removeFilter = (index: number) => {
    setLocalFilters(localFilters.filter((_, i) => i !== index));
  };

  const applyFilters = () => {
    setFilters(localFilters);
    onClose();
  };

  const clearFilters = () => {
    setLocalFilters([]);
    setFilters([]);
  };

  const renderValueInput = (filter: Filter, index: number, field: Field) => {
    if (!operatorNeedsValue(filter.operator)) {
      return null;
    }

    const category = getOperatorCategory(field.type);

    // Select field - show dropdown of choices
    if (category === 'select' && (filter.operator === 'is' || filter.operator === 'is_not')) {
      const choices = field.options?.choices || [];
      return (
        <select
          value={filter.value as string || ''}
          onChange={(e) => updateFilter(index, { value: e.target.value })}
          className="flex-1 min-w-[120px] px-2 py-1.5 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
        >
          <option value="">Select...</option>
          {choices.map((choice: { id: string; name: string; color: string }) => (
            <option key={choice.id} value={choice.id}>{choice.name}</option>
          ))}
        </select>
      );
    }

    // Multi-select for is_any_of, is_none_of
    if (category === 'select' && ['is_any_of', 'is_none_of'].includes(filter.operator)) {
      const choices = field.options?.choices || [];
      const selectedValues = (filter.value as string[]) || [];
      return (
        <div className="flex-1 min-w-[150px]">
          <div className="flex flex-wrap gap-1 p-1.5 border border-slate-200 rounded-md min-h-[32px]">
            {choices.map((choice: { id: string; name: string; color: string }) => {
              const isSelected = selectedValues.includes(choice.id);
              return (
                <button
                  key={choice.id}
                  onClick={() => {
                    const newValues = isSelected
                      ? selectedValues.filter(v => v !== choice.id)
                      : [...selectedValues, choice.id];
                    updateFilter(index, { value: newValues });
                  }}
                  className={`px-2 py-0.5 text-xs rounded-full transition-colors ${
                    isSelected
                      ? 'text-white'
                      : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                  }`}
                  style={isSelected ? { backgroundColor: choice.color } : {}}
                >
                  {choice.name}
                </button>
              );
            })}
          </div>
        </div>
      );
    }

    // Date field
    if (category === 'date') {
      if (filter.operator === 'is_within') {
        return (
          <div className="flex items-center gap-2">
            <input
              type="number"
              min="1"
              value={filter.value as number || 7}
              onChange={(e) => updateFilter(index, { value: parseInt(e.target.value) || 7 })}
              className="w-16 px-2 py-1.5 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
            />
            <span className="text-sm text-slate-600">days</span>
          </div>
        );
      }
      return (
        <input
          type="date"
          value={filter.value as string || ''}
          onChange={(e) => updateFilter(index, { value: e.target.value })}
          className="flex-1 min-w-[140px] px-2 py-1.5 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
        />
      );
    }

    // Number field
    if (category === 'number') {
      return (
        <input
          type="number"
          value={filter.value as number ?? ''}
          onChange={(e) => updateFilter(index, { value: e.target.value ? parseFloat(e.target.value) : '' })}
          className="flex-1 min-w-[100px] px-2 py-1.5 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
          placeholder="Value..."
        />
      );
    }

    // Checkbox - no value needed, handled by operator
    if (category === 'checkbox') {
      return null;
    }

    // Default text input
    return (
      <input
        type="text"
        value={filter.value as string || ''}
        onChange={(e) => updateFilter(index, { value: e.target.value })}
        className="flex-1 min-w-[120px] px-2 py-1.5 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
        placeholder="Value..."
      />
    );
  };

  if (!isOpen) return null;

  return (
    <div
      ref={panelRef}
      className="fixed z-50 bg-white rounded-lg shadow-xl border border-slate-200 min-w-[400px] max-w-[600px]"
      style={{ top: position.top, left: position.left }}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200">
        <div className="flex items-center gap-2">
          <svg className="w-4 h-4 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
          </svg>
          <span className="font-medium text-slate-700">Filter</span>
          {localFilters.length > 0 && (
            <span className="px-1.5 py-0.5 text-xs bg-primary-100 text-primary-700 rounded-full">
              {localFilters.length}
            </span>
          )}
        </div>
        <button onClick={onClose} className="text-slate-400 hover:text-slate-600">
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Filter rows */}
      <div className="p-4 space-y-3 max-h-[400px] overflow-y-auto">
        {localFilters.length === 0 ? (
          <div className="text-sm text-slate-500 text-center py-4">
            No filters applied. Add a filter to narrow down records.
          </div>
        ) : (
          localFilters.map((filter, index) => {
            const field = fields.find(f => f.id === filter.field_id);
            const category = field ? getOperatorCategory(field.type) : 'text';
            const operators = OPERATORS[category] || OPERATORS.text;

            return (
              <div key={index} className="flex items-start gap-2">
                {/* Conjunction */}
                {index > 0 && (
                  <button
                    onClick={() => setConjunction(conjunction === 'and' ? 'or' : 'and')}
                    className="px-2 py-1.5 text-xs font-medium text-slate-500 bg-slate-100 rounded hover:bg-slate-200 uppercase"
                  >
                    {conjunction}
                  </button>
                )}
                {index === 0 && (
                  <span className="px-2 py-1.5 text-xs font-medium text-slate-500 uppercase">Where</span>
                )}

                {/* Field selector */}
                <select
                  value={filter.field_id}
                  onChange={(e) => updateFilter(index, { field_id: e.target.value })}
                  className="min-w-[120px] px-2 py-1.5 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  {filterableFields.map(f => (
                    <option key={f.id} value={f.id}>{f.name}</option>
                  ))}
                </select>

                {/* Operator selector */}
                <select
                  value={filter.operator}
                  onChange={(e) => updateFilter(index, { operator: e.target.value })}
                  className="min-w-[120px] px-2 py-1.5 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  {operators.map(op => (
                    <option key={op.value} value={op.value}>{op.label}</option>
                  ))}
                </select>

                {/* Value input */}
                {field && renderValueInput(filter, index, field)}

                {/* Remove button */}
                <button
                  onClick={() => removeFilter(index)}
                  className="p-1.5 text-slate-400 hover:text-red-500 rounded hover:bg-red-50"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            );
          })
        )}

        {/* Add filter button */}
        <button
          onClick={addFilter}
          className="flex items-center gap-2 text-sm text-primary hover:text-primary-dark"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add filter
        </button>
      </div>

      {/* Footer */}
      <div className="flex items-center justify-between px-4 py-3 border-t border-slate-200 bg-slate-50">
        <button
          onClick={clearFilters}
          className="text-sm text-slate-600 hover:text-slate-800"
        >
          Clear all
        </button>
        <div className="flex items-center gap-2">
          <button
            onClick={onClose}
            className="px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-100 rounded-md"
          >
            Cancel
          </button>
          <button
            onClick={applyFilters}
            className="px-3 py-1.5 text-sm text-white bg-primary hover:bg-primary-dark rounded-md"
          >
            Apply filters
          </button>
        </div>
      </div>
    </div>
  );
}
