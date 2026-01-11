import { useState } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { ViewSettings } from '../../../types';

interface TimelineSettingsProps {
  isOpen: boolean;
  onClose: () => void;
}

export function TimelineSettings({ isOpen, onClose }: TimelineSettingsProps) {
  const { currentView, fields, updateViewConfig } = useBaseStore();
  const [localSettings, setLocalSettings] = useState<Partial<ViewSettings>>(() => ({
    start_field_id: currentView?.settings?.start_field_id || (currentView?.config?.dateField as string | undefined),
    end_field_id: currentView?.settings?.end_field_id || (currentView?.config?.endDateField as string | undefined),
    group_field_id: currentView?.settings?.group_field_id,
    group_field_ids: currentView?.settings?.group_field_ids || [],
    color_field_id: currentView?.settings?.color_field_id,
    label_field_ids: currentView?.settings?.label_field_ids || [],
    timeline_row_height: currentView?.settings?.timeline_row_height || 'medium',
    show_dependencies: currentView?.settings?.show_dependencies ?? true,
    show_today_marker: currentView?.settings?.show_today_marker ?? true,
    show_weekends: currentView?.settings?.show_weekends ?? true,
  }));

  const dateFields = fields.filter(f => ['date', 'datetime'].includes(f.type));
  const groupableFields = fields.filter(f =>
    ['single_select', 'multi_select', 'text', 'single_line_text', 'checkbox', 'rating', 'collaborator', 'user'].includes(f.type)
  );
  const colorFields = fields.filter(f =>
    ['single_select', 'multi_select'].includes(f.type)
  );

  const handleSave = () => {
    if (!currentView) return;

    // Build group_field_ids from the multi-level grouping
    const groupFieldIds = [
      localSettings.group_field_id,
      ...(localSettings.group_field_ids?.slice(1) || []),
    ].filter(Boolean) as string[];

    updateViewConfig({
      ...currentView.config,
      dateField: localSettings.start_field_id,
      endDateField: localSettings.end_field_id,
      ...localSettings,
      group_field_ids: groupFieldIds,
    });
    onClose();
  };

  const updateSetting = <K extends keyof ViewSettings>(key: K, value: ViewSettings[K]) => {
    setLocalSettings(prev => ({ ...prev, [key]: value }));
  };

  // Multi-level grouping management
  const groupLevels = [
    localSettings.group_field_id,
    ...(localSettings.group_field_ids?.slice(1) || []),
  ].filter(Boolean);

  const addGroupLevel = () => {
    if (groupLevels.length >= 3) return;
    const availableFields = groupableFields.filter(f => !groupLevels.includes(f.id));
    if (availableFields.length === 0) return;

    const newGroupIds = [...groupLevels.filter((id): id is string => typeof id === 'string'), ''];
    updateSetting('group_field_ids', newGroupIds);
  };

  const updateGroupLevel = (index: number, fieldId: string) => {
    if (index === 0) {
      updateSetting('group_field_id', fieldId || undefined);
    } else {
      const newGroupIds = [...groupLevels];
      newGroupIds[index] = fieldId;
      updateSetting('group_field_ids', newGroupIds.filter(Boolean) as string[]);
    }
  };

  const removeGroupLevel = (index: number) => {
    if (index === 0) {
      updateSetting('group_field_id', undefined);
      updateSetting('group_field_ids', []);
    } else {
      const newGroupIds = groupLevels.filter((_, i) => i !== index);
      updateSetting('group_field_ids', newGroupIds as string[]);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="absolute right-0 top-full mt-1 w-96 bg-white rounded-lg shadow-xl border border-slate-200 z-50">
      <div className="p-4 border-b border-slate-200">
        <div className="flex items-center justify-between">
          <h3 className="font-semibold text-slate-900">Timeline Settings</h3>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-600">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      <div className="p-4 space-y-4 max-h-[500px] overflow-y-auto">
        {/* Date Fields */}
        <div className="space-y-3">
          <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Date Fields</h4>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Start Date</label>
            <select
              value={localSettings.start_field_id || ''}
              onChange={(e) => updateSetting('start_field_id', e.target.value || undefined)}
              className="input w-full"
            >
              <option value="">Select field...</option>
              {dateFields.map(field => (
                <option key={field.id} value={field.id}>{field.name}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">End Date (optional)</label>
            <select
              value={localSettings.end_field_id || ''}
              onChange={(e) => updateSetting('end_field_id', e.target.value || undefined)}
              className="input w-full"
            >
              <option value="">Same as start date</option>
              {dateFields.map(field => (
                <option key={field.id} value={field.id}>{field.name}</option>
              ))}
            </select>
          </div>
        </div>

        {/* Multi-level Grouping */}
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wide">
              Grouping (Swimlanes)
            </h4>
            {groupLevels.length < 3 && groupableFields.length > groupLevels.length && (
              <button
                onClick={addGroupLevel}
                className="text-xs text-primary hover:text-primary/80 font-medium"
              >
                + Add level
              </button>
            )}
          </div>

          {groupLevels.length === 0 ? (
            <div>
              <select
                value=""
                onChange={(e) => {
                  if (e.target.value) {
                    updateSetting('group_field_id', e.target.value);
                  }
                }}
                className="input w-full"
              >
                <option value="">No grouping</option>
                {groupableFields.map(field => (
                  <option key={field.id} value={field.id}>{field.name}</option>
                ))}
              </select>
            </div>
          ) : (
            groupLevels.map((fieldId, index) => (
              <div key={index} className="flex items-center gap-2">
                <span className="text-xs text-slate-400 w-4">{index + 1}.</span>
                <select
                  value={fieldId || ''}
                  onChange={(e) => updateGroupLevel(index, e.target.value)}
                  className="input flex-1"
                >
                  <option value="">Select field...</option>
                  {groupableFields
                    .filter(f => f.id === fieldId || !groupLevels.includes(f.id))
                    .map(field => (
                      <option key={field.id} value={field.id}>{field.name}</option>
                    ))}
                </select>
                <button
                  onClick={() => removeGroupLevel(index)}
                  className="p-1 text-slate-400 hover:text-slate-600"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            ))
          )}
          <p className="text-xs text-slate-400">
            Group records into swimlanes (up to 3 levels)
          </p>
        </div>

        {/* Appearance */}
        <div className="space-y-3">
          <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Appearance</h4>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Color By</label>
            <select
              value={localSettings.color_field_id || ''}
              onChange={(e) => updateSetting('color_field_id', e.target.value || undefined)}
              className="input w-full"
            >
              <option value="">Auto (first select field)</option>
              {colorFields.map(field => (
                <option key={field.id} value={field.id}>{field.name}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Row Height</label>
            <select
              value={localSettings.timeline_row_height || 'medium'}
              onChange={(e) => updateSetting('timeline_row_height', e.target.value as 'compact' | 'medium' | 'tall')}
              className="input w-full"
            >
              <option value="compact">Compact (32px)</option>
              <option value="medium">Medium (40px)</option>
              <option value="tall">Tall (48px)</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Bar Labels</label>
            <p className="text-xs text-slate-400 mb-2">Select fields to display on timeline bars</p>
            <div className="space-y-1 max-h-32 overflow-y-auto border rounded-md p-2">
              {fields.filter(f => !f.is_hidden).slice(0, 10).map(field => (
                <label key={field.id} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={localSettings.label_field_ids?.includes(field.id) || false}
                    onChange={(e) => {
                      const current = localSettings.label_field_ids || [];
                      if (e.target.checked) {
                        updateSetting('label_field_ids', [...current, field.id]);
                      } else {
                        updateSetting('label_field_ids', current.filter(id => id !== field.id));
                      }
                    }}
                    className="rounded border-slate-300"
                  />
                  <span className="text-slate-700">{field.name}</span>
                </label>
              ))}
            </div>
          </div>
        </div>

        {/* Display Options */}
        <div className="space-y-3">
          <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Display Options</h4>

          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.show_today_marker ?? true}
              onChange={(e) => updateSetting('show_today_marker', e.target.checked)}
              className="rounded border-slate-300"
            />
            <div>
              <span className="text-sm text-slate-700">Show today marker</span>
              <p className="text-xs text-slate-400">Red line indicating current date</p>
            </div>
          </label>

          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.show_weekends ?? true}
              onChange={(e) => updateSetting('show_weekends', e.target.checked)}
              className="rounded border-slate-300"
            />
            <div>
              <span className="text-sm text-slate-700">Show weekends</span>
              <p className="text-xs text-slate-400">Uncheck to show weekdays only (Mon-Fri)</p>
            </div>
          </label>

          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.show_dependencies ?? true}
              onChange={(e) => updateSetting('show_dependencies', e.target.checked)}
              className="rounded border-slate-300"
            />
            <div>
              <span className="text-sm text-slate-700">Show dependencies</span>
              <p className="text-xs text-slate-400">Display arrows connecting related records</p>
            </div>
          </label>
        </div>

        {/* Tips */}
        <div className="bg-slate-50 rounded-lg p-3">
          <h4 className="text-xs font-semibold text-slate-600 mb-2">Tips</h4>
          <ul className="text-xs text-slate-500 space-y-1">
            <li>Drag on empty space to create new records</li>
            <li>Drag bar edges to resize duration</li>
            <li>Drag bars to change dates</li>
            <li>Click a bar to edit the record</li>
          </ul>
        </div>
      </div>

      <div className="p-4 border-t border-slate-200 flex justify-end gap-2">
        <button onClick={onClose} className="btn btn-secondary">
          Cancel
        </button>
        <button onClick={handleSave} className="btn btn-primary">
          Apply
        </button>
      </div>
    </div>
  );
}
