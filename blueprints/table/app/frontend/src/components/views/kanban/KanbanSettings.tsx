import { useState, useRef, useEffect } from 'react';
import type { Field } from '../../../types';
import type { KanbanConfig } from './types';

interface KanbanSettingsProps {
  config: KanbanConfig;
  fields: Field[];
  onConfigChange: (config: Partial<KanbanConfig>) => void;
  onClose: () => void;
}

export function KanbanSettings({
  config,
  fields,
  onConfigChange,
  onClose,
}: KanbanSettingsProps) {
  const panelRef = useRef<HTMLDivElement>(null);

  // Close on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [onClose]);

  // Get fields suitable for stacking (single_select, user, linked record with single link)
  const stackableFields = fields.filter(
    (f) => f.type === 'single_select' || f.type === 'user'
  );

  // Get attachment fields for cover images
  const attachmentFields = fields.filter((f) => f.type === 'attachment');

  // Get fields suitable for card coloring (single_select, multi_select)
  const colorableFields = fields.filter(
    (f) => f.type === 'single_select' || f.type === 'multi_select'
  );

  // Get all non-computed fields for display selection
  const displayableFields = fields.filter(
    (f) => !f.is_computed && !f.is_primary
  );

  return (
    <div
      ref={panelRef}
      className="absolute top-12 right-0 z-50 w-80 bg-white rounded-lg shadow-xl border border-slate-200 overflow-hidden"
    >
      {/* Header */}
      <div className="px-4 py-3 border-b border-slate-200 flex items-center justify-between bg-slate-50">
        <h3 className="font-semibold text-gray-900">Kanban Settings</h3>
        <button
          onClick={onClose}
          className="p-1 text-slate-400 hover:text-slate-600 rounded"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div className="p-4 space-y-5 max-h-[70vh] overflow-y-auto">
        {/* Stack by field */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1.5">
            Stack by
          </label>
          <select
            value={config.groupBy || ''}
            onChange={(e) => onConfigChange({ groupBy: e.target.value || null })}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-primary"
          >
            <option value="">Select a field...</option>
            {stackableFields.map((field) => (
              <option key={field.id} value={field.id}>
                {field.name} ({getFieldTypeLabel(field.type)})
              </option>
            ))}
          </select>
          {stackableFields.length === 0 && (
            <p className="mt-1 text-xs text-amber-600">
              Add a Single Select or User field to use Kanban view
            </p>
          )}
        </div>

        {/* Card size */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1.5">
            Card size
          </label>
          <div className="flex gap-2">
            {(['small', 'medium', 'large'] as const).map((size) => (
              <button
                key={size}
                onClick={() => onConfigChange({ cardSize: size })}
                className={`
                  flex-1 px-3 py-2 text-sm rounded-lg border transition-colors
                  ${config.cardSize === size
                    ? 'bg-primary text-white border-primary'
                    : 'bg-white text-gray-700 border-slate-300 hover:bg-slate-50'}
                `}
              >
                {size.charAt(0).toUpperCase() + size.slice(1)}
              </button>
            ))}
          </div>
        </div>

        {/* Cover image field */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1.5">
            Card cover image
          </label>
          <select
            value={config.coverField || ''}
            onChange={(e) => onConfigChange({ coverField: e.target.value || null })}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-primary"
          >
            <option value="">None</option>
            {attachmentFields.map((field) => (
              <option key={field.id} value={field.id}>
                {field.name}
              </option>
            ))}
          </select>
          {config.coverField && (
            <div className="mt-2 flex items-center gap-2">
              <input
                type="checkbox"
                id="fit-image"
                checked={config.cardCoverFit === 'contain'}
                onChange={(e) =>
                  onConfigChange({ cardCoverFit: e.target.checked ? 'contain' : 'cover' })
                }
                className="rounded border-slate-300 text-primary focus:ring-primary"
              />
              <label htmlFor="fit-image" className="text-sm text-gray-600">
                Fit image (don't crop)
              </label>
            </div>
          )}
        </div>

        {/* Card color field */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1.5">
            Color cards by
          </label>
          <select
            value={config.cardColorField || ''}
            onChange={(e) => onConfigChange({ cardColorField: e.target.value || null })}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-primary"
          >
            <option value="">None</option>
            {colorableFields.map((field) => (
              <option key={field.id} value={field.id}>
                {field.name}
              </option>
            ))}
          </select>
        </div>

        <hr className="border-slate-200" />

        {/* Card fields selection */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1.5">
            Fields to show on cards
          </label>
          <CardFieldsSelector
            fields={displayableFields}
            selectedFields={config.cardFields}
            onChange={(cardFields) => onConfigChange({ cardFields })}
          />
          <p className="mt-1 text-xs text-slate-500">
            Drag to reorder. Number shown depends on card size.
          </p>
        </div>

        <hr className="border-slate-200" />

        {/* Additional options */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="hide-empty"
              checked={config.hideEmptyColumns}
              onChange={(e) => onConfigChange({ hideEmptyColumns: e.target.checked })}
              className="rounded border-slate-300 text-primary focus:ring-primary"
            />
            <label htmlFor="hide-empty" className="text-sm text-gray-600">
              Hide empty columns
            </label>
          </div>
        </div>
      </div>
    </div>
  );
}

interface CardFieldsSelectorProps {
  fields: Field[];
  selectedFields: string[];
  onChange: (selected: string[]) => void;
}

function CardFieldsSelector({ fields, selectedFields, onChange }: CardFieldsSelectorProps) {
  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);

  const handleToggle = (fieldId: string) => {
    if (selectedFields.includes(fieldId)) {
      onChange(selectedFields.filter((id) => id !== fieldId));
    } else {
      onChange([...selectedFields, fieldId]);
    }
  };

  const handleDragStart = (index: number) => {
    setDraggedIndex(index);
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    if (draggedIndex === null || draggedIndex === index) return;

    const newOrder = [...selectedFields];
    const [removed] = newOrder.splice(draggedIndex, 1);
    newOrder.splice(index, 0, removed);
    onChange(newOrder);
    setDraggedIndex(index);
  };

  const handleDragEnd = () => {
    setDraggedIndex(null);
  };

  // Show selected fields first (in order), then unselected
  const sortedFields = [
    ...selectedFields.map((id) => fields.find((f) => f.id === id)).filter(Boolean) as Field[],
    ...fields.filter((f) => !selectedFields.includes(f.id)),
  ];

  return (
    <div className="max-h-48 overflow-y-auto border border-slate-200 rounded-lg">
      {sortedFields.map((field) => {
        const isSelected = selectedFields.includes(field.id);
        const selectedIndex = selectedFields.indexOf(field.id);

        return (
          <div
            key={field.id}
            draggable={isSelected}
            onDragStart={() => handleDragStart(selectedIndex)}
            onDragOver={(e) => isSelected ? handleDragOver(e, selectedIndex) : undefined}
            onDragEnd={handleDragEnd}
            className={`
              flex items-center gap-2 px-3 py-2 border-b border-slate-100 last:border-b-0
              ${isSelected ? 'bg-primary/5 cursor-grab' : 'hover:bg-slate-50'}
            `}
          >
            <input
              type="checkbox"
              checked={isSelected}
              onChange={() => handleToggle(field.id)}
              className="rounded border-slate-300 text-primary focus:ring-primary"
            />
            {isSelected && (
              <svg className="w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
              </svg>
            )}
            <span className="text-sm text-gray-700 flex-1">{field.name}</span>
            <span className="text-xs text-slate-400">{getFieldTypeLabel(field.type)}</span>
          </div>
        );
      })}
      {fields.length === 0 && (
        <div className="p-4 text-sm text-slate-500 text-center">
          No fields available
        </div>
      )}
    </div>
  );
}

function getFieldTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    text: 'Text',
    long_text: 'Long Text',
    number: 'Number',
    currency: 'Currency',
    percent: 'Percent',
    single_select: 'Single Select',
    multi_select: 'Multi Select',
    date: 'Date',
    datetime: 'Date & Time',
    checkbox: 'Checkbox',
    rating: 'Rating',
    email: 'Email',
    url: 'URL',
    phone: 'Phone',
    user: 'User',
    attachment: 'Attachment',
    link: 'Link',
    formula: 'Formula',
    rollup: 'Rollup',
    lookup: 'Lookup',
    created_time: 'Created Time',
    last_modified_time: 'Modified Time',
  };
  return labels[type] || type;
}
