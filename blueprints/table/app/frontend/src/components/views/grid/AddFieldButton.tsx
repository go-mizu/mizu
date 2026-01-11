import { useState } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { FieldType } from '../../../types';

const FIELD_TYPES: { type: FieldType; label: string; description: string }[] = [
  { type: 'text', label: 'Single line text', description: 'Short text, tags, labels' },
  { type: 'long_text', label: 'Long text', description: 'Notes, descriptions' },
  { type: 'number', label: 'Number', description: 'Integers, decimals' },
  { type: 'currency', label: 'Currency', description: 'Dollar amounts' },
  { type: 'percent', label: 'Percent', description: 'Percentages' },
  { type: 'single_select', label: 'Single select', description: 'One option from a list' },
  { type: 'multi_select', label: 'Multiple select', description: 'Multiple options from a list' },
  { type: 'checkbox', label: 'Checkbox', description: 'Yes or no' },
  { type: 'date', label: 'Date', description: 'Calendar date' },
  { type: 'datetime', label: 'Date & time', description: 'Date with time' },
  { type: 'email', label: 'Email', description: 'Email address' },
  { type: 'url', label: 'URL', description: 'Web link' },
  { type: 'phone', label: 'Phone', description: 'Phone number' },
  { type: 'rating', label: 'Rating', description: 'Star rating' },
  { type: 'attachment', label: 'Attachment', description: 'Files and images' },
  { type: 'user', label: 'User', description: 'Assign to people' },
  { type: 'link', label: 'Link to another record', description: 'Link rows between tables' },
  { type: 'formula', label: 'Formula', description: 'Calculate values' },
  { type: 'rollup', label: 'Rollup', description: 'Summarize linked records' },
  { type: 'lookup', label: 'Lookup', description: 'See fields from linked records' },
  { type: 'count', label: 'Count', description: 'Count linked records' },
  { type: 'autonumber', label: 'Autonumber', description: 'Auto-incrementing number' },
  { type: 'created_time', label: 'Created time', description: 'When record was created' },
  { type: 'last_modified_time', label: 'Last modified time', description: 'When record was changed' },
  { type: 'created_by', label: 'Created by', description: 'Who created the record' },
  { type: 'last_modified_by', label: 'Last modified by', description: 'Who last changed it' },
];

export function AddFieldButton() {
  const { createField } = useBaseStore();
  const [showMenu, setShowMenu] = useState(false);
  const [showNameInput, setShowNameInput] = useState(false);
  const [selectedType, setSelectedType] = useState<FieldType | null>(null);
  const [fieldName, setFieldName] = useState('');

  const handleSelectType = (type: FieldType) => {
    setSelectedType(type);
    setShowNameInput(true);
    setFieldName('');
  };

  const handleCreate = async () => {
    if (!fieldName.trim() || !selectedType) return;

    let options = {};
    if (selectedType === 'single_select' || selectedType === 'multi_select') {
      options = { choices: [] };
    } else if (selectedType === 'rating') {
      options = { max: 5 };
    }

    await createField(fieldName.trim(), selectedType, options);
    setShowMenu(false);
    setShowNameInput(false);
    setSelectedType(null);
    setFieldName('');
  };

  const handleClose = () => {
    setShowMenu(false);
    setShowNameInput(false);
    setSelectedType(null);
    setFieldName('');
  };

  return (
    <div className="relative">
      <button
        onClick={() => setShowMenu(true)}
        className="h-9 px-3 text-sm text-slate-500 hover:text-slate-800 flex items-center gap-1 rounded-md hover:bg-slate-50"
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
        Add field
      </button>

      {showMenu && (
        <>
          <div className="fixed inset-0 z-40" onClick={handleClose} />
          <div className="absolute right-0 top-full mt-1 w-80 bg-white rounded-xl shadow-lg border border-gray-200 z-50 max-h-[400px] overflow-hidden flex flex-col">
            {showNameInput && selectedType ? (
              <>
                <div className="p-4 border-b border-gray-200">
                  <button
                    onClick={() => setShowNameInput(false)}
                    className="flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 mb-3"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                    </svg>
                    Back
                  </button>
                  <div className="space-y-3">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Field name</label>
                      <input
                        type="text"
                        value={fieldName}
                        onChange={(e) => setFieldName(e.target.value)}
                        placeholder="Enter field name"
                        className="input"
                        autoFocus
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') handleCreate();
                          if (e.key === 'Escape') handleClose();
                        }}
                      />
                    </div>
                    <div>
                      <span className="text-sm text-gray-500">
                        Type: {FIELD_TYPES.find(f => f.type === selectedType)?.label}
                      </span>
                    </div>
                  </div>
                </div>
                <div className="p-4 bg-slate-50 flex justify-end gap-2">
                  <button onClick={handleClose} className="btn btn-secondary">Cancel</button>
                  <button onClick={handleCreate} className="btn btn-primary" disabled={!fieldName.trim()}>
                    Create field
                  </button>
                </div>
              </>
            ) : (
              <>
                <div className="p-3 border-b border-gray-200">
                  <h3 className="font-medium text-gray-900">Add a field</h3>
                </div>
                <div className="overflow-y-auto flex-1">
                  <div className="p-2 space-y-1">
                    {FIELD_TYPES.map((fieldType) => (
                      <button
                        key={fieldType.type}
                        onClick={() => handleSelectType(fieldType.type)}
                        className="w-full text-left px-3 py-2 rounded-md hover:bg-gray-100 transition-colors"
                      >
                        <div className="text-sm font-medium text-gray-900">{fieldType.label}</div>
                        <div className="text-xs text-gray-500">{fieldType.description}</div>
                      </button>
                    ))}
                  </div>
                </div>
              </>
            )}
          </div>
        </>
      )}
    </div>
  );
}
