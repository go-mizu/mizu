import { useState } from 'react';
import { useBaseStore } from '../../stores/baseStore';

export function BaseHeader() {
  const { currentBase, updateBase } = useBaseStore();
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(currentBase?.name || '');

  if (!currentBase) return null;

  const handleSave = async () => {
    if (name.trim() && name !== currentBase.name) {
      await updateBase(currentBase.id, { name: name.trim() });
    }
    setEditing(false);
  };

  return (
    <div className="bg-white border-b border-slate-200 px-4 py-3">
      <div className="flex items-center gap-3">
        <span
          className="w-7 h-7 rounded-md border border-white shadow-sm"
          style={{ backgroundColor: currentBase.color }}
        />
        <div className="flex flex-col">
          <span className="text-xs font-semibold uppercase tracking-wide text-slate-400">
            Base
          </span>
          {editing ? (
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onBlur={handleSave}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSave();
                if (e.key === 'Escape') setEditing(false);
              }}
              className="text-lg font-semibold border border-slate-200 rounded-md px-2 py-1 focus:outline-none focus:ring-2 focus:ring-primary"
              autoFocus
            />
          ) : (
            <button
              type="button"
              className="text-left text-lg font-semibold text-gray-900 hover:bg-slate-50 px-2 py-1 rounded-md"
              onClick={() => {
                setName(currentBase.name);
                setEditing(true);
              }}
            >
              {currentBase.name}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
