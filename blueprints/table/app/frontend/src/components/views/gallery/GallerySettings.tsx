import { useState, useRef, useEffect } from 'react';
import type { Field } from '../../../types';
import type { GalleryConfig, AspectRatio } from './types';
import { ASPECT_RATIOS } from './types';

interface GallerySettingsProps {
  config: GalleryConfig;
  fields: Field[];
  onConfigChange: (config: Partial<GalleryConfig>) => void;
  onClose: () => void;
}

export function GallerySettings({
  config,
  fields,
  onConfigChange,
  onClose,
}: GallerySettingsProps) {
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [onClose]);

  const attachmentFields = fields.filter((f) => f.type === 'attachment');
  const textFields = fields.filter(
    (f) => f.type === 'text' || f.type === 'long_text'
  );
  const colorableFields = fields.filter(
    (f) => f.type === 'single_select' || f.type === 'multi_select'
  );
  const displayableFields = fields.filter(
    (f) => !f.is_computed && !f.is_primary && f.type !== 'attachment'
  );

  return (
    <div
      ref={panelRef}
      className="absolute top-12 right-0 z-50 w-80 bg-white rounded-lg shadow-xl border border-slate-200 overflow-hidden"
    >
      <div className="px-4 py-3 border-b border-slate-200 flex items-center justify-between bg-slate-50">
        <h3 className="font-semibold text-gray-900">Gallery Settings</h3>
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
        {/* Cover image field */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1.5">
            Cover image
          </label>
          <select
            value={config.coverField || ''}
            onChange={(e) => onConfigChange({ coverField: e.target.value || null })}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-primary"
          >
            <option value="">None (show placeholder)</option>
            {attachmentFields.map((field) => (
              <option key={field.id} value={field.id}>
                {field.name}
              </option>
            ))}
          </select>
          {attachmentFields.length === 0 && (
            <p className="mt-1 text-xs text-amber-600">
              Add an Attachment field to display cover images
            </p>
          )}
        </div>

        {/* Image fit */}
        {config.coverField && (
          <div className="flex items-center gap-2">
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

        {/* Aspect ratio */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1.5">
            Cover aspect ratio
          </label>
          <div className="grid grid-cols-4 gap-2">
            {(Object.keys(ASPECT_RATIOS) as AspectRatio[]).map((ratio) => (
              <button
                key={ratio}
                onClick={() => onConfigChange({ aspectRatio: ratio })}
                className={`
                  px-2 py-1.5 text-xs rounded-lg border transition-colors
                  ${config.aspectRatio === ratio
                    ? 'bg-primary text-white border-primary'
                    : 'bg-white text-gray-700 border-slate-300 hover:bg-slate-50'}
                `}
              >
                {ratio}
              </button>
            ))}
          </div>
        </div>

        {/* Title field */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1.5">
            Card title
          </label>
          <select
            value={config.titleField || ''}
            onChange={(e) => onConfigChange({ titleField: e.target.value || null })}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:ring-2 focus:ring-primary focus:border-primary"
          >
            <option value="">Auto (first text field)</option>
            {textFields.map((field) => (
              <option key={field.id} value={field.id}>
                {field.name}
              </option>
            ))}
          </select>
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

        {/* Show empty cards */}
        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            id="show-empty"
            checked={config.showEmptyCards}
            onChange={(e) => onConfigChange({ showEmptyCards: e.target.checked })}
            className="rounded border-slate-300 text-primary focus:ring-primary"
          />
          <label htmlFor="show-empty" className="text-sm text-gray-600">
            Show cards without cover images
          </label>
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
    single_line_text: 'Text',
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
    collaborator: 'Collaborator',
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
