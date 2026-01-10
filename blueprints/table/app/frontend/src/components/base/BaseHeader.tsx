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
    <div className="bg-white border-b border-gray-200 px-4 py-3">
      <div className="flex items-center gap-3">
        <span
          className="w-6 h-6 rounded"
          style={{ backgroundColor: currentBase.color }}
        />
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
            className="text-lg font-semibold border border-gray-300 rounded px-2 py-1 focus:outline-none focus:ring-2 focus:ring-primary"
            autoFocus
          />
        ) : (
          <h2
            className="text-lg font-semibold text-gray-900 cursor-pointer hover:bg-gray-100 px-2 py-1 rounded"
            onClick={() => {
              setName(currentBase.name);
              setEditing(true);
            }}
          >
            {currentBase.name}
          </h2>
        )}
      </div>
    </div>
  );
}
