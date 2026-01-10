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
    color_field_id: currentView?.settings?.color_field_id,
    label_field_ids: currentView?.settings?.label_field_ids || [],
    timeline_row_height: currentView?.settings?.timeline_row_height || 'medium',
    show_dependencies: currentView?.settings?.show_dependencies ?? true,
    show_today_marker: currentView?.settings?.show_today_marker ?? true,
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
    updateViewConfig({
      ...currentView.config,
      dateField: localSettings.start_field_id,
      endDateField: localSettings.end_field_id,
      ...localSettings,
    });
    onClose();
  };

  const updateSetting = <K extends keyof ViewSettings>(key: K, value: ViewSettings[K]) => {
    setLocalSettings(prev => ({ ...prev, [key]: value }));
  };

  if (!isOpen) return null;

  return (
    <div className="absolute right-0 top-full mt-1 w-80 bg-white rounded-lg shadow-xl border border-slate-200 z-50">
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

      <div className="p-4 space-y-4 max-h-[400px] overflow-y-auto">
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

        {/* Grouping */}
        <div className="space-y-3">
          <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Grouping</h4>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Group By</label>
            <select
              value={localSettings.group_field_id || ''}
              onChange={(e) => updateSetting('group_field_id', e.target.value || undefined)}
              className="input w-full"
            >
              <option value="">No grouping</option>
              {groupableFields.map(field => (
                <option key={field.id} value={field.id}>{field.name}</option>
              ))}
            </select>
          </div>
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
              <option value="compact">Compact</option>
              <option value="medium">Medium</option>
              <option value="tall">Tall</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Bar Labels</label>
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

        {/* Features */}
        <div className="space-y-3">
          <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Features</h4>

          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.show_today_marker ?? true}
              onChange={(e) => updateSetting('show_today_marker', e.target.checked)}
              className="rounded border-slate-300"
            />
            <span className="text-sm text-slate-700">Show today marker</span>
          </label>

          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={localSettings.show_dependencies ?? true}
              onChange={(e) => updateSetting('show_dependencies', e.target.checked)}
              className="rounded border-slate-300"
            />
            <span className="text-sm text-slate-700">Show dependencies</span>
          </label>
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
