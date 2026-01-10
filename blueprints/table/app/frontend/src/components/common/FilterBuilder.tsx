import { useState } from 'react';
import { useBaseStore } from '../../stores/baseStore';
import type { Field, FieldType, Filter } from '../../types';

interface FilterBuilderProps {
  onClose: () => void;
}

// Operators by field type
const OPERATORS: Record<string, { label: string; value: string }[]> = {
  text: [
    { label: 'contains', value: 'contains' },
    { label: 'does not contain', value: 'not_contains' },
    { label: 'is', value: 'is' },
    { label: 'is not', value: 'is_not' },
    { label: 'is empty', value: 'is_empty' },
    { label: 'is not empty', value: 'is_not_empty' },
    { label: 'starts with', value: 'starts_with' },
    { label: 'ends with', value: 'ends_with' },
  ],
  number: [
    { label: '=', value: 'eq' },
    { label: '≠', value: 'neq' },
    { label: '<', value: 'lt' },
    { label: '≤', value: 'lte' },
    { label: '>', value: 'gt' },
    { label: '≥', value: 'gte' },
    { label: 'is empty', value: 'is_empty' },
    { label: 'is not empty', value: 'is_not_empty' },
  ],
  select: [
    { label: 'is', value: 'is' },
    { label: 'is not', value: 'is_not' },
    { label: 'is any of', value: 'is_any_of' },
    { label: 'is none of', value: 'is_none_of' },
    { label: 'is empty', value: 'is_empty' },
    { label: 'is not empty', value: 'is_not_empty' },
  ],
  date: [
    { label: 'is', value: 'is' },
    { label: 'is before', value: 'is_before' },
    { label: 'is after', value: 'is_after' },
    { label: 'is on or before', value: 'is_on_or_before' },
    { label: 'is on or after', value: 'is_on_or_after' },
    { label: 'is within', value: 'is_within' },
    { label: 'is empty', value: 'is_empty' },
    { label: 'is not empty', value: 'is_not_empty' },
  ],
  checkbox: [
    { label: 'is checked', value: 'is_checked' },
    { label: 'is not checked', value: 'is_not_checked' },
  ],
};

const getOperatorsForFieldType = (type: FieldType): { label: string; value: string }[] => {
  if (['text', 'long_text', 'email', 'url', 'phone'].includes(type)) {
    return OPERATORS.text;
  }
  if (['number', 'currency', 'percent', 'rating', 'duration'].includes(type)) {
    return OPERATORS.number;
  }
  if (['single_select', 'multi_select'].includes(type)) {
    return OPERATORS.select;
  }
  if (['date', 'datetime', 'created_time', 'last_modified_time'].includes(type)) {
    return OPERATORS.date;
  }
  if (type === 'checkbox') {
    return OPERATORS.checkbox;
  }
  return OPERATORS.text;
};

const needsValue = (operator: string): boolean => {
  return !['is_empty', 'is_not_empty', 'is_checked', 'is_not_checked'].includes(operator);
};

export function FilterBuilder({ onClose }: FilterBuilderProps) {
  const { fields, filters, setFilters } = useBaseStore();
  const [localFilters, setLocalFilters] = useState<Filter[]>(filters.length > 0 ? [...filters] : []);
  const [conjunction, setConjunction] = useState<'and' | 'or'>('and');

  const filterableFields = fields.filter(f => !['formula', 'rollup', 'count', 'lookup'].includes(f.type));

  const addFilter = () => {
    if (filterableFields.length === 0) return;
    const defaultField = filterableFields[0];
    const operators = getOperatorsForFieldType(defaultField.type);
    setLocalFilters([
      ...localFilters,
      {
        field_id: defaultField.id,
        operator: operators[0].value,
        value: null,
      },
    ]);
  };

  const updateFilter = (index: number, updates: Partial<Filter>) => {
    setLocalFilters(localFilters.map((f, i) => (i === index ? { ...f, ...updates } : f)));
  };

  const removeFilter = (index: number) => {
    setLocalFilters(localFilters.filter((_, i) => i !== index));
  };

  const applyFilters = () => {
    setFilters(localFilters, conjunction);
    onClose();
  };

  const clearFilters = () => {
    setLocalFilters([]);
    setFilters([], 'and');
    onClose();
  };

  const renderValueInput = (filter: Filter, index: number, field: Field) => {
    if (!needsValue(filter.operator)) {
      return null;
    }

    const fieldType = field.type;

    // Select fields
    if (['single_select', 'multi_select'].includes(fieldType)) {
      const options = field.options?.choices || field.select_options || [];
      if (filter.operator === 'is_any_of' || filter.operator === 'is_none_of') {
        // Multi-select for any of / none of
        const selectedValues = Array.isArray(filter.value) ? filter.value : [];
        return (
          <div className="flex flex-wrap gap-1">
            {options.map((opt: { id: string; name: string; color: string }) => (
              <button
                key={opt.id}
                type="button"
                onClick={() => {
                  const newValues = selectedValues.includes(opt.id)
                    ? selectedValues.filter((v: string) => v !== opt.id)
                    : [...selectedValues, opt.id];
                  updateFilter(index, { value: newValues });
                }}
                className={`px-2 py-1 text-xs rounded-full border transition-colors ${
                  selectedValues.includes(opt.id)
                    ? 'border-primary bg-primary-50'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
                style={selectedValues.includes(opt.id) ? { backgroundColor: opt.color + '20', color: opt.color } : {}}
              >
                {opt.name}
              </button>
            ))}
          </div>
        );
      }
      return (
        <select
          value={(filter.value as string) || ''}
          onChange={(e) => updateFilter(index, { value: e.target.value || null })}
          className="input py-1"
        >
          <option value="">Select...</option>
          {options.map((opt: { id: string; name: string }) => (
            <option key={opt.id} value={opt.id}>{opt.name}</option>
          ))}
        </select>
      );
    }

    // Number fields
    if (['number', 'currency', 'percent', 'rating', 'duration'].includes(fieldType)) {
      return (
        <input
          type="number"
          value={(filter.value as number) ?? ''}
          onChange={(e) => updateFilter(index, { value: e.target.value ? parseFloat(e.target.value) : null })}
          className="input py-1"
          placeholder="Value"
        />
      );
    }

    // Date fields
    if (['date', 'datetime', 'created_time', 'last_modified_time'].includes(fieldType)) {
      if (filter.operator === 'is_within') {
        return (
          <select
            value={(filter.value as string) || ''}
            onChange={(e) => updateFilter(index, { value: e.target.value || null })}
            className="input py-1"
          >
            <option value="">Select...</option>
            <option value="past_week">Past week</option>
            <option value="past_month">Past month</option>
            <option value="past_year">Past year</option>
            <option value="next_week">Next week</option>
            <option value="next_month">Next month</option>
            <option value="next_year">Next year</option>
            <option value="today">Today</option>
            <option value="tomorrow">Tomorrow</option>
            <option value="yesterday">Yesterday</option>
          </select>
        );
      }
      return (
        <input
          type="date"
          value={(filter.value as string) || ''}
          onChange={(e) => updateFilter(index, { value: e.target.value || null })}
          className="input py-1"
        />
      );
    }

    // Text fields (default)
    return (
      <input
        type="text"
        value={(filter.value as string) || ''}
        onChange={(e) => updateFilter(index, { value: e.target.value || null })}
        className="input py-1"
        placeholder="Value"
      />
    );
  };

  return (
    <div className="p-4 min-w-[400px]">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-medium text-gray-900">Filter records</h3>
        <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {localFilters.length > 0 && (
        <div className="mb-3">
          <span className="text-sm text-gray-600">Show records where</span>
          {localFilters.length > 1 && (
            <select
              value={conjunction}
              onChange={(e) => setConjunction(e.target.value as 'and' | 'or')}
              className="ml-2 text-sm border border-gray-300 rounded px-2 py-1"
            >
              <option value="and">all</option>
              <option value="or">any</option>
            </select>
          )}
          {localFilters.length > 1 && (
            <span className="text-sm text-gray-600 ml-1">conditions match</span>
          )}
        </div>
      )}

      <div className="space-y-3">
        {localFilters.map((filter, index) => {
          const field = fields.find(f => f.id === filter.field_id);
          const operators = field ? getOperatorsForFieldType(field.type) : OPERATORS.text;

          return (
            <div key={index} className="flex items-start gap-2">
              {/* Field selector */}
              <select
                value={filter.field_id}
                onChange={(e) => {
                  const newField = fields.find(f => f.id === e.target.value);
                  const newOperators = newField ? getOperatorsForFieldType(newField.type) : OPERATORS.text;
                  updateFilter(index, {
                    field_id: e.target.value,
                    operator: newOperators[0].value,
                    value: null,
                  });
                }}
                className="input py-1 w-32"
              >
                {filterableFields.map((f) => (
                  <option key={f.id} value={f.id}>{f.name}</option>
                ))}
              </select>

              {/* Operator selector */}
              <select
                value={filter.operator}
                onChange={(e) => updateFilter(index, { operator: e.target.value })}
                className="input py-1 w-36"
              >
                {operators.map((op) => (
                  <option key={op.value} value={op.value}>{op.label}</option>
                ))}
              </select>

              {/* Value input */}
              <div className="flex-1">
                {field && renderValueInput(filter, index, field)}
              </div>

              {/* Remove button */}
              <button
                onClick={() => removeFilter(index)}
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

      {/* Add filter button */}
      <button
        onClick={addFilter}
        className="mt-3 text-sm text-primary hover:text-primary-600 flex items-center gap-1"
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
        Add filter
      </button>

      {/* Action buttons */}
      <div className="mt-4 pt-3 border-t border-gray-200 flex items-center justify-between">
        <button
          onClick={clearFilters}
          className="text-sm text-gray-600 hover:text-gray-800"
        >
          Clear all
        </button>
        <button
          onClick={applyFilters}
          className="btn btn-primary py-1.5 px-4 text-sm"
        >
          Apply filters
        </button>
      </div>
    </div>
  );
}
