import { useState, useCallback } from 'react';
import type { Field } from '../../../types';

// Filter operator type (reused from FilterBuilder)
type FilterOperator =
  | 'equals'
  | 'not_equals'
  | 'contains'
  | 'not_contains'
  | 'starts_with'
  | 'ends_with'
  | 'is_empty'
  | 'is_not_empty'
  | 'greater_than'
  | 'greater_than_or_equal'
  | 'less_than'
  | 'less_than_or_equal'
  | 'is_checked'
  | 'is_unchecked'
  | 'is_any_of'
  | 'is_none_of';

interface FilterCondition {
  fieldId: string;
  operator: FilterOperator;
  value: unknown;
}

interface FormatStyle {
  backgroundColor?: string;
  textColor?: string;
  bold?: boolean;
  italic?: boolean;
  strikethrough?: boolean;
}

export interface ConditionalFormat {
  id: string;
  name: string;
  conditions: FilterCondition[];
  conditionLogic: 'and' | 'or';
  style: FormatStyle;
  applyTo: 'row' | 'cell';
  fieldId?: string; // For cell-level formatting
  priority: number;
  enabled: boolean;
}

interface ConditionalFormatConfigProps {
  fields: Field[];
  formats: ConditionalFormat[];
  onChange: (formats: ConditionalFormat[]) => void;
  onClose: () => void;
}

// Preset colors
const PRESET_COLORS = [
  { name: 'Red', bg: '#FEE2E2', text: '#991B1B' },
  { name: 'Orange', bg: '#FFEDD5', text: '#9A3412' },
  { name: 'Yellow', bg: '#FEF3C7', text: '#92400E' },
  { name: 'Green', bg: '#D1FAE5', text: '#065F46' },
  { name: 'Blue', bg: '#DBEAFE', text: '#1E40AF' },
  { name: 'Purple', bg: '#EDE9FE', text: '#5B21B6' },
  { name: 'Pink', bg: '#FCE7F3', text: '#9D174D' },
  { name: 'Gray', bg: '#F3F4F6', text: '#374151' },
];

// Get operators for field type
function getOperatorsForFieldType(fieldType: string): { value: FilterOperator; label: string }[] {
  const baseOperators: { value: FilterOperator; label: string }[] = [
    { value: 'is_empty', label: 'Is empty' },
    { value: 'is_not_empty', label: 'Is not empty' },
  ];

  switch (fieldType) {
    case 'checkbox':
      return [
        { value: 'is_checked', label: 'Is checked' },
        { value: 'is_unchecked', label: 'Is unchecked' },
      ];
    case 'number':
    case 'currency':
    case 'percent':
    case 'rating':
    case 'duration':
      return [
        { value: 'equals', label: 'Equals' },
        { value: 'not_equals', label: 'Does not equal' },
        { value: 'greater_than', label: 'Greater than' },
        { value: 'greater_than_or_equal', label: 'Greater than or equal' },
        { value: 'less_than', label: 'Less than' },
        { value: 'less_than_or_equal', label: 'Less than or equal' },
        ...baseOperators,
      ];
    case 'single_select':
    case 'multi_select':
      return [
        { value: 'equals', label: 'Is' },
        { value: 'not_equals', label: 'Is not' },
        { value: 'is_any_of', label: 'Is any of' },
        { value: 'is_none_of', label: 'Is none of' },
        ...baseOperators,
      ];
    default:
      return [
        { value: 'equals', label: 'Equals' },
        { value: 'not_equals', label: 'Does not equal' },
        { value: 'contains', label: 'Contains' },
        { value: 'not_contains', label: 'Does not contain' },
        { value: 'starts_with', label: 'Starts with' },
        { value: 'ends_with', label: 'Ends with' },
        ...baseOperators,
      ];
  }
}

export function ConditionalFormatConfig({
  fields,
  formats,
  onChange,
  onClose,
}: ConditionalFormatConfigProps) {
  const [selectedFormat, setSelectedFormat] = useState<ConditionalFormat | null>(
    formats.length > 0 ? formats[0] : null
  );

  // Generate unique ID
  const generateId = () => `cf-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

  // Add new format
  const addFormat = useCallback(() => {
    const newFormat: ConditionalFormat = {
      id: generateId(),
      name: `Format ${formats.length + 1}`,
      conditions: [{ fieldId: fields[0]?.id || '', operator: 'equals', value: '' }],
      conditionLogic: 'and',
      style: { backgroundColor: PRESET_COLORS[0].bg, textColor: PRESET_COLORS[0].text },
      applyTo: 'row',
      priority: formats.length,
      enabled: true,
    };
    const newFormats = [...formats, newFormat];
    onChange(newFormats);
    setSelectedFormat(newFormat);
  }, [formats, fields, onChange]);

  // Delete format
  const deleteFormat = useCallback((id: string) => {
    const newFormats = formats.filter((f) => f.id !== id);
    onChange(newFormats);
    if (selectedFormat?.id === id) {
      setSelectedFormat(newFormats[0] || null);
    }
  }, [formats, selectedFormat, onChange]);

  // Update format
  const updateFormat = useCallback((id: string, updates: Partial<ConditionalFormat>) => {
    const newFormats = formats.map((f) => (f.id === id ? { ...f, ...updates } : f));
    onChange(newFormats);
    if (selectedFormat?.id === id) {
      setSelectedFormat({ ...selectedFormat, ...updates });
    }
  }, [formats, selectedFormat, onChange]);

  // Add condition to format
  const addCondition = useCallback((formatId: string) => {
    const format = formats.find((f) => f.id === formatId);
    if (!format) return;

    const newCondition: FilterCondition = {
      fieldId: fields[0]?.id || '',
      operator: 'equals',
      value: '',
    };

    updateFormat(formatId, { conditions: [...format.conditions, newCondition] });
  }, [formats, fields, updateFormat]);

  // Update condition
  const updateCondition = useCallback(
    (formatId: string, conditionIndex: number, updates: Partial<FilterCondition>) => {
      const format = formats.find((f) => f.id === formatId);
      if (!format) return;

      const newConditions = format.conditions.map((c, i) =>
        i === conditionIndex ? { ...c, ...updates } : c
      );
      updateFormat(formatId, { conditions: newConditions });
    },
    [formats, updateFormat]
  );

  // Delete condition
  const deleteCondition = useCallback((formatId: string, conditionIndex: number) => {
    const format = formats.find((f) => f.id === formatId);
    if (!format || format.conditions.length <= 1) return;

    const newConditions = format.conditions.filter((_, i) => i !== conditionIndex);
    updateFormat(formatId, { conditions: newConditions });
  }, [formats, updateFormat]);

  // Move format up/down
  const moveFormat = useCallback((id: string, direction: 'up' | 'down') => {
    const index = formats.findIndex((f) => f.id === id);
    if (index === -1) return;

    const newIndex = direction === 'up' ? index - 1 : index + 1;
    if (newIndex < 0 || newIndex >= formats.length) return;

    const newFormats = [...formats];
    [newFormats[index], newFormats[newIndex]] = [newFormats[newIndex], newFormats[index]];
    newFormats.forEach((f, i) => (f.priority = i));
    onChange(newFormats);
  }, [formats, onChange]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-3xl max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-200">
          <h3 className="text-lg font-semibold text-slate-900">Conditional Formatting</h3>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-600">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 flex overflow-hidden">
          {/* Rules list */}
          <div className="w-64 border-r border-slate-200 flex flex-col">
            <div className="p-2 border-b border-slate-200">
              <button
                onClick={addFormat}
                className="w-full px-3 py-2 text-sm text-primary hover:bg-primary-50 rounded-lg flex items-center gap-2"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Add rule
              </button>
            </div>

            <div className="flex-1 overflow-auto p-2 space-y-1">
              {formats.map((format, index) => (
                <div
                  key={format.id}
                  onClick={() => setSelectedFormat(format)}
                  className={`p-2 rounded-lg cursor-pointer flex items-center gap-2 ${
                    selectedFormat?.id === format.id
                      ? 'bg-primary-50 ring-1 ring-primary'
                      : 'hover:bg-slate-50'
                  }`}
                >
                  {/* Color preview */}
                  <div
                    className="w-4 h-4 rounded flex-shrink-0"
                    style={{
                      backgroundColor: format.style.backgroundColor,
                      border: '1px solid rgba(0,0,0,0.1)',
                    }}
                  />

                  {/* Name */}
                  <span className={`flex-1 text-sm truncate ${!format.enabled ? 'text-slate-400' : ''}`}>
                    {format.name}
                  </span>

                  {/* Priority arrows */}
                  <div className="flex flex-col">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        moveFormat(format.id, 'up');
                      }}
                      disabled={index === 0}
                      className="text-slate-400 hover:text-slate-600 disabled:opacity-30"
                    >
                      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
                      </svg>
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        moveFormat(format.id, 'down');
                      }}
                      disabled={index === formats.length - 1}
                      className="text-slate-400 hover:text-slate-600 disabled:opacity-30"
                    >
                      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                      </svg>
                    </button>
                  </div>
                </div>
              ))}

              {formats.length === 0 && (
                <p className="text-sm text-slate-500 text-center py-4">
                  No formatting rules yet
                </p>
              )}
            </div>
          </div>

          {/* Rule editor */}
          <div className="flex-1 overflow-auto p-4">
            {selectedFormat ? (
              <div className="space-y-6">
                {/* Name and enabled toggle */}
                <div className="flex items-center gap-4">
                  <input
                    type="text"
                    value={selectedFormat.name}
                    onChange={(e) => updateFormat(selectedFormat.id, { name: e.target.value })}
                    className="flex-1 px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                    placeholder="Rule name"
                  />

                  <label className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={selectedFormat.enabled}
                      onChange={(e) => updateFormat(selectedFormat.id, { enabled: e.target.checked })}
                      className="w-4 h-4 rounded border-slate-300"
                    />
                    Enabled
                  </label>

                  <button
                    onClick={() => deleteFormat(selectedFormat.id)}
                    className="p-2 text-red-500 hover:bg-red-50 rounded-lg"
                    title="Delete rule"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </div>

                {/* Conditions */}
                <div>
                  <h4 className="text-sm font-medium text-slate-700 mb-2">Conditions</h4>

                  <div className="space-y-2">
                    {selectedFormat.conditions.map((condition, index) => {
                      const field = fields.find((f) => f.id === condition.fieldId);
                      const operators = getOperatorsForFieldType(field?.type || 'text');

                      return (
                        <div key={index} className="flex items-center gap-2">
                          {index > 0 && (
                            <select
                              value={selectedFormat.conditionLogic}
                              onChange={(e) =>
                                updateFormat(selectedFormat.id, {
                                  conditionLogic: e.target.value as 'and' | 'or',
                                })
                              }
                              className="px-2 py-1 text-xs border border-slate-200 rounded"
                            >
                              <option value="and">AND</option>
                              <option value="or">OR</option>
                            </select>
                          )}

                          {/* Field select */}
                          <select
                            value={condition.fieldId}
                            onChange={(e) =>
                              updateCondition(selectedFormat.id, index, { fieldId: e.target.value })
                            }
                            className="px-2 py-1.5 text-sm border border-slate-200 rounded-lg"
                          >
                            {fields.map((f) => (
                              <option key={f.id} value={f.id}>
                                {f.name}
                              </option>
                            ))}
                          </select>

                          {/* Operator select */}
                          <select
                            value={condition.operator}
                            onChange={(e) =>
                              updateCondition(selectedFormat.id, index, {
                                operator: e.target.value as FilterOperator,
                              })
                            }
                            className="px-2 py-1.5 text-sm border border-slate-200 rounded-lg"
                          >
                            {operators.map((op) => (
                              <option key={op.value} value={op.value}>
                                {op.label}
                              </option>
                            ))}
                          </select>

                          {/* Value input (if needed) */}
                          {!['is_empty', 'is_not_empty', 'is_checked', 'is_unchecked'].includes(
                            condition.operator
                          ) && (
                            <input
                              type={
                                ['number', 'currency', 'percent', 'rating'].includes(field?.type || '')
                                  ? 'number'
                                  : 'text'
                              }
                              value={condition.value as string}
                              onChange={(e) =>
                                updateCondition(selectedFormat.id, index, { value: e.target.value })
                              }
                              className="px-2 py-1.5 text-sm border border-slate-200 rounded-lg flex-1"
                              placeholder="Value"
                            />
                          )}

                          {/* Delete condition */}
                          {selectedFormat.conditions.length > 1 && (
                            <button
                              onClick={() => deleteCondition(selectedFormat.id, index)}
                              className="p-1 text-slate-400 hover:text-red-500"
                            >
                              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                              </svg>
                            </button>
                          )}
                        </div>
                      );
                    })}
                  </div>

                  <button
                    onClick={() => addCondition(selectedFormat.id)}
                    className="mt-2 text-sm text-primary hover:underline"
                  >
                    + Add condition
                  </button>
                </div>

                {/* Apply to */}
                <div>
                  <h4 className="text-sm font-medium text-slate-700 mb-2">Apply to</h4>
                  <div className="flex gap-4">
                    <label className="flex items-center gap-2">
                      <input
                        type="radio"
                        name="applyTo"
                        checked={selectedFormat.applyTo === 'row'}
                        onChange={() => updateFormat(selectedFormat.id, { applyTo: 'row', fieldId: undefined })}
                        className="w-4 h-4"
                      />
                      <span className="text-sm">Entire row</span>
                    </label>
                    <label className="flex items-center gap-2">
                      <input
                        type="radio"
                        name="applyTo"
                        checked={selectedFormat.applyTo === 'cell'}
                        onChange={() =>
                          updateFormat(selectedFormat.id, { applyTo: 'cell', fieldId: fields[0]?.id })
                        }
                        className="w-4 h-4"
                      />
                      <span className="text-sm">Specific cell</span>
                    </label>
                  </div>

                  {selectedFormat.applyTo === 'cell' && (
                    <select
                      value={selectedFormat.fieldId || ''}
                      onChange={(e) => updateFormat(selectedFormat.id, { fieldId: e.target.value })}
                      className="mt-2 px-3 py-2 text-sm border border-slate-200 rounded-lg"
                    >
                      {fields.map((f) => (
                        <option key={f.id} value={f.id}>
                          {f.name}
                        </option>
                      ))}
                    </select>
                  )}
                </div>

                {/* Style */}
                <div>
                  <h4 className="text-sm font-medium text-slate-700 mb-2">Style</h4>

                  {/* Preset colors */}
                  <div className="flex gap-2 mb-3">
                    {PRESET_COLORS.map((preset) => (
                      <button
                        key={preset.name}
                        onClick={() =>
                          updateFormat(selectedFormat.id, {
                            style: {
                              ...selectedFormat.style,
                              backgroundColor: preset.bg,
                              textColor: preset.text,
                            },
                          })
                        }
                        className="w-8 h-8 rounded-lg border-2 transition-all"
                        style={{
                          backgroundColor: preset.bg,
                          borderColor:
                            selectedFormat.style.backgroundColor === preset.bg
                              ? preset.text
                              : 'transparent',
                        }}
                        title={preset.name}
                      />
                    ))}
                  </div>

                  {/* Text formatting */}
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() =>
                        updateFormat(selectedFormat.id, {
                          style: { ...selectedFormat.style, bold: !selectedFormat.style.bold },
                        })
                      }
                      className={`px-3 py-1.5 rounded-lg border ${
                        selectedFormat.style.bold
                          ? 'bg-primary-100 border-primary text-primary'
                          : 'border-slate-200 text-slate-600 hover:bg-slate-50'
                      }`}
                    >
                      <span className="font-bold">B</span>
                    </button>
                    <button
                      onClick={() =>
                        updateFormat(selectedFormat.id, {
                          style: { ...selectedFormat.style, italic: !selectedFormat.style.italic },
                        })
                      }
                      className={`px-3 py-1.5 rounded-lg border ${
                        selectedFormat.style.italic
                          ? 'bg-primary-100 border-primary text-primary'
                          : 'border-slate-200 text-slate-600 hover:bg-slate-50'
                      }`}
                    >
                      <span className="italic">I</span>
                    </button>
                    <button
                      onClick={() =>
                        updateFormat(selectedFormat.id, {
                          style: {
                            ...selectedFormat.style,
                            strikethrough: !selectedFormat.style.strikethrough,
                          },
                        })
                      }
                      className={`px-3 py-1.5 rounded-lg border ${
                        selectedFormat.style.strikethrough
                          ? 'bg-primary-100 border-primary text-primary'
                          : 'border-slate-200 text-slate-600 hover:bg-slate-50'
                      }`}
                    >
                      <span className="line-through">S</span>
                    </button>
                  </div>

                  {/* Preview */}
                  <div className="mt-4">
                    <h5 className="text-xs text-slate-500 mb-1">Preview</h5>
                    <div
                      className="px-4 py-2 rounded-lg"
                      style={{
                        backgroundColor: selectedFormat.style.backgroundColor,
                        color: selectedFormat.style.textColor,
                        fontWeight: selectedFormat.style.bold ? 'bold' : 'normal',
                        fontStyle: selectedFormat.style.italic ? 'italic' : 'normal',
                        textDecoration: selectedFormat.style.strikethrough ? 'line-through' : 'none',
                      }}
                    >
                      Sample text
                    </div>
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-center h-full text-slate-500">
                Select a rule to edit or create a new one
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 p-4 border-t border-slate-200">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-slate-700 hover:bg-slate-100 rounded-lg"
          >
            Done
          </button>
        </div>
      </div>
    </div>
  );
}

// Utility function to evaluate if a record matches conditional format
export function evaluateConditionalFormat(
  format: ConditionalFormat,
  record: { values: Record<string, unknown> },
  _fields: Field[]
): boolean {
  if (!format.enabled || format.conditions.length === 0) {
    return false;
  }

  const results = format.conditions.map((condition) => {
    const value = record.values[condition.fieldId];
    return evaluateCondition(condition, value);
  });

  if (format.conditionLogic === 'or') {
    return results.some((r) => r);
  }
  return results.every((r) => r);
}

function evaluateCondition(condition: FilterCondition, value: unknown): boolean {
  const strValue = value === null || value === undefined ? '' : String(value);
  const condValue = String(condition.value);

  switch (condition.operator) {
    case 'equals':
      return strValue === condValue;
    case 'not_equals':
      return strValue !== condValue;
    case 'contains':
      return strValue.toLowerCase().includes(condValue.toLowerCase());
    case 'not_contains':
      return !strValue.toLowerCase().includes(condValue.toLowerCase());
    case 'starts_with':
      return strValue.toLowerCase().startsWith(condValue.toLowerCase());
    case 'ends_with':
      return strValue.toLowerCase().endsWith(condValue.toLowerCase());
    case 'is_empty':
      return value === null || value === undefined || value === '';
    case 'is_not_empty':
      return value !== null && value !== undefined && value !== '';
    case 'greater_than':
      return Number(value) > Number(condition.value);
    case 'greater_than_or_equal':
      return Number(value) >= Number(condition.value);
    case 'less_than':
      return Number(value) < Number(condition.value);
    case 'less_than_or_equal':
      return Number(value) <= Number(condition.value);
    case 'is_checked':
      return value === true || value === 'true';
    case 'is_unchecked':
      return value === false || value === 'false' || value === null || value === undefined;
    case 'is_any_of':
      const anyValues = Array.isArray(condition.value) ? condition.value : [condition.value];
      return anyValues.includes(value);
    case 'is_none_of':
      const noneValues = Array.isArray(condition.value) ? condition.value : [condition.value];
      return !noneValues.includes(value);
    default:
      return false;
  }
}
