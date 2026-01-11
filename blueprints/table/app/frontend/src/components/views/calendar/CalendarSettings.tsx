import { useCallback, useEffect, useRef } from 'react';
import type { Field } from '../../../types';
import type { CalendarConfig } from './types';

interface CalendarSettingsProps {
  config: CalendarConfig;
  fields: Field[];
  onConfigChange: (updates: Partial<CalendarConfig>) => void;
  onClose: () => void;
}

export function CalendarSettings({
  config,
  fields,
  onConfigChange,
  onClose,
}: CalendarSettingsProps) {
  const panelRef = useRef<HTMLDivElement>(null);

  // Close on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (panelRef.current && !panelRef.current.contains(event.target as Node)) {
        onClose();
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [onClose]);

  // Close on escape
  useEffect(() => {
    function handleEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        onClose();
      }
    }
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [onClose]);

  // Get date fields
  const dateFields = fields.filter((f) => f.type === 'date' || f.type === 'datetime');

  // Get single_select fields for coloring
  const selectFields = fields.filter((f) => f.type === 'single_select');

  // Get attachment fields for cover images
  const attachmentFields = fields.filter((f) => f.type === 'attachment');

  // Get all fields for display (excluding hidden and primary)
  const displayableFields = fields.filter(
    (f) => !f.is_hidden && !f.is_primary && f.type !== 'attachment'
  );

  const handleDateFieldChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      onConfigChange({ dateField: e.target.value || null });
    },
    [onConfigChange]
  );

  const handleEndDateFieldChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      onConfigChange({ endDateField: e.target.value || null });
    },
    [onConfigChange]
  );

  const handleWeekStartChange = useCallback(
    (value: 0 | 1) => {
      onConfigChange({ weekStart: value });
    },
    [onConfigChange]
  );

  const handleShowWeekendsChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onConfigChange({ showWeekends: e.target.checked });
    },
    [onConfigChange]
  );

  const handleColorFieldChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      onConfigChange({ colorField: e.target.value || null });
    },
    [onConfigChange]
  );

  const handleCoverFieldChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      onConfigChange({ coverField: e.target.value || null });
    },
    [onConfigChange]
  );

  const handleDisplayFieldToggle = useCallback(
    (fieldId: string) => {
      const newDisplayFields = config.displayFields.includes(fieldId)
        ? config.displayFields.filter((id) => id !== fieldId)
        : [...config.displayFields, fieldId];
      onConfigChange({ displayFields: newDisplayFields });
    },
    [config.displayFields, onConfigChange]
  );

  const handleEventSizeChange = useCallback(
    (value: 'compact' | 'comfortable') => {
      onConfigChange({ eventSize: value });
    },
    [onConfigChange]
  );

  const handleShowTimeChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onConfigChange({ showTime: e.target.checked });
    },
    [onConfigChange]
  );

  return (
    <div
      ref={panelRef}
      className="absolute right-0 top-full mt-1 w-80 bg-white rounded-lg shadow-xl border border-slate-200 z-50 overflow-hidden"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200">
        <h3 className="text-sm font-semibold text-gray-900">Calendar Settings</h3>
        <button
          onClick={onClose}
          className="p-1 hover:bg-slate-100 rounded-md text-gray-500"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      </div>

      {/* Content */}
      <div className="p-4 space-y-4 max-h-[60vh] overflow-y-auto">
        {/* Date Field */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Date field
          </label>
          <select
            value={config.dateField || ''}
            onChange={handleDateFieldChange}
            className="w-full px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
          >
            <option value="">Select a date field...</option>
            {dateFields.map((field) => (
              <option key={field.id} value={field.id}>
                {field.name} ({field.type})
              </option>
            ))}
          </select>
        </div>

        {/* End Date Field */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            End date field <span className="text-gray-400">(optional)</span>
          </label>
          <select
            value={config.endDateField || ''}
            onChange={handleEndDateFieldChange}
            className="w-full px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
          >
            <option value="">None</option>
            {dateFields
              .filter((f) => f.id !== config.dateField)
              .map((field) => (
                <option key={field.id} value={field.id}>
                  {field.name} ({field.type})
                </option>
              ))}
          </select>
          <p className="text-xs text-gray-500 mt-1">
            Enable multi-day events by selecting an end date field
          </p>
        </div>

        <hr className="border-slate-200" />

        {/* Week Start */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Week starts on
          </label>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                checked={config.weekStart === 0}
                onChange={() => handleWeekStartChange(0)}
                className="w-4 h-4 text-primary focus:ring-primary"
              />
              <span className="text-sm text-gray-600">Sunday</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                checked={config.weekStart === 1}
                onChange={() => handleWeekStartChange(1)}
                className="w-4 h-4 text-primary focus:ring-primary"
              />
              <span className="text-sm text-gray-600">Monday</span>
            </label>
          </div>
        </div>

        {/* Show Weekends */}
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={config.showWeekends}
            onChange={handleShowWeekendsChange}
            className="w-4 h-4 text-primary rounded focus:ring-primary"
          />
          <span className="text-sm text-gray-600">Show weekends</span>
        </label>

        <hr className="border-slate-200" />

        {/* Event Color */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Event color
          </label>
          <select
            value={config.colorField || ''}
            onChange={handleColorFieldChange}
            className="w-full px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
          >
            <option value="">Default (blue)</option>
            {selectFields.map((field) => (
              <option key={field.id} value={field.id}>
                {field.name}
              </option>
            ))}
          </select>
        </div>

        {/* Cover Image */}
        {attachmentFields.length > 0 && (
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Cover image
            </label>
            <select
              value={config.coverField || ''}
              onChange={handleCoverFieldChange}
              className="w-full px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
            >
              <option value="">None</option>
              {attachmentFields.map((field) => (
                <option key={field.id} value={field.id}>
                  {field.name}
                </option>
              ))}
            </select>
          </div>
        )}

        <hr className="border-slate-200" />

        {/* Display Fields */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Display fields
          </label>
          <div className="space-y-2 max-h-40 overflow-y-auto">
            {displayableFields.map((field) => (
              <label
                key={field.id}
                className="flex items-center gap-2 cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={config.displayFields.includes(field.id)}
                  onChange={() => handleDisplayFieldToggle(field.id)}
                  className="w-4 h-4 text-primary rounded focus:ring-primary"
                />
                <span className="text-sm text-gray-600">{field.name}</span>
              </label>
            ))}
          </div>
        </div>

        {/* Event Size */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Event size
          </label>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                checked={config.eventSize === 'compact'}
                onChange={() => handleEventSizeChange('compact')}
                className="w-4 h-4 text-primary focus:ring-primary"
              />
              <span className="text-sm text-gray-600">Compact</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                checked={config.eventSize === 'comfortable'}
                onChange={() => handleEventSizeChange('comfortable')}
                className="w-4 h-4 text-primary focus:ring-primary"
              />
              <span className="text-sm text-gray-600">Comfortable</span>
            </label>
          </div>
        </div>

        {/* Show Time */}
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={config.showTime}
            onChange={handleShowTimeChange}
            className="w-4 h-4 text-primary rounded focus:ring-primary"
          />
          <span className="text-sm text-gray-600">Show time on events</span>
        </label>
      </div>
    </div>
  );
}
