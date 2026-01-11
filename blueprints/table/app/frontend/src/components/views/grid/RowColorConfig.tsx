import { useState } from 'react';
import { useBaseStore } from '../../../stores/baseStore';

interface RowColorConfigProps {
  onClose: () => void;
}

export function RowColorConfig({ onClose }: RowColorConfigProps) {
  const { fields, currentView, updateViewConfig } = useBaseStore();

  // Get current config
  const config = currentView?.config && typeof currentView.config === 'object'
    ? currentView.config as Record<string, unknown>
    : {};

  const [selectedFieldId, setSelectedFieldId] = useState<string | null>(
    (config.row_color_field_id as string) || null
  );

  // Filter to only single_select fields
  const selectFields = fields.filter(f => f.type === 'single_select');

  const handleApply = () => {
    updateViewConfig({ ...config, row_color_field_id: selectedFieldId });
    onClose();
  };

  const handleClear = () => {
    setSelectedFieldId(null);
    updateViewConfig({ ...config, row_color_field_id: null });
    onClose();
  };

  return (
    <div className="p-4 min-w-[280px]">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-semibold text-gray-900">Color records by</h3>
        <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <p className="text-sm text-gray-500 mb-4">
        Select a single select field to color rows based on the selected option.
      </p>

      {selectFields.length === 0 ? (
        <div className="text-sm text-gray-500 py-4 text-center">
          No single select fields available.
          <br />
          Create a single select field to use row coloring.
        </div>
      ) : (
        <div className="space-y-2">
          {/* No color option */}
          <label className="flex items-center gap-3 p-2 rounded-md hover:bg-slate-50 cursor-pointer">
            <input
              type="radio"
              name="colorField"
              checked={selectedFieldId === null}
              onChange={() => setSelectedFieldId(null)}
              className="w-4 h-4 text-primary"
            />
            <span className="text-sm text-gray-600">No color</span>
          </label>

          {/* Field options */}
          {selectFields.map(field => {
            const options = field.options?.choices || [];
            return (
              <label
                key={field.id}
                className="flex items-center gap-3 p-2 rounded-md hover:bg-slate-50 cursor-pointer"
              >
                <input
                  type="radio"
                  name="colorField"
                  checked={selectedFieldId === field.id}
                  onChange={() => setSelectedFieldId(field.id)}
                  className="w-4 h-4 text-primary"
                />
                <div className="flex-1">
                  <div className="text-sm font-medium text-gray-700">{field.name}</div>
                  {options.length > 0 && (
                    <div className="flex gap-1 mt-1 flex-wrap">
                      {options.slice(0, 5).map((opt: { id: string; name: string; color: string }) => (
                        <span
                          key={opt.id}
                          className="w-3 h-3 rounded-full"
                          style={{ backgroundColor: opt.color }}
                          title={opt.name}
                        />
                      ))}
                      {options.length > 5 && (
                        <span className="text-xs text-gray-400">+{options.length - 5}</span>
                      )}
                    </div>
                  )}
                </div>
              </label>
            );
          })}
        </div>
      )}

      {/* Action buttons */}
      <div className="mt-4 pt-3 border-t border-slate-200 flex items-center justify-between">
        <button
          onClick={handleClear}
          className="text-sm text-slate-600 hover:text-slate-800"
        >
          Clear
        </button>
        <button
          onClick={handleApply}
          className="btn btn-primary py-1.5 px-4 text-sm"
        >
          Apply
        </button>
      </div>
    </div>
  );
}
