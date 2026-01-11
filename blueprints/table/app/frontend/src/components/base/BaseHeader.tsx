import { useState, useRef, useEffect } from 'react';
import { useBaseStore } from '../../stores/baseStore';

const BASE_COLORS = [
  '#166ee1', '#20c933', '#8b46ff', '#f82b60', '#fcb400',
  '#18bfff', '#ff6f2c', '#ff08c2', '#20d9d2', '#6b7280',
];

export function BaseHeader() {
  const { currentBase, updateBase } = useBaseStore();
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(currentBase?.name || '');
  const [showColorPicker, setShowColorPicker] = useState(false);
  const colorPickerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Update name when base changes
  useEffect(() => {
    setName(currentBase?.name || '');
  }, [currentBase?.name]);

  // Close color picker on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (colorPickerRef.current && !colorPickerRef.current.contains(e.target as Node)) {
        setShowColorPicker(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Focus input when editing
  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [editing]);

  if (!currentBase) return null;

  const handleSave = async () => {
    if (name.trim() && name !== currentBase.name) {
      await updateBase(currentBase.id, { name: name.trim() });
    }
    setEditing(false);
  };

  const handleColorChange = async (color: string) => {
    await updateBase(currentBase.id, { color });
    setShowColorPicker(false);
  };

  return (
    <div className="bg-white border-b border-[var(--at-border)] px-5 py-3">
      <div className="flex items-center gap-4">
        {/* Color picker */}
        <div className="relative" ref={colorPickerRef}>
          <button
            onClick={() => setShowColorPicker(!showColorPicker)}
            className="w-9 h-9 rounded-lg flex items-center justify-center transition-all hover:scale-105 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
            style={{
              backgroundColor: currentBase.color || '#166ee1',
              boxShadow: 'var(--shadow-sm)',
            }}
            title="Change color"
          >
            <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
            </svg>
          </button>

          {showColorPicker && (
            <div
              className="popover mt-2 p-3"
              style={{ left: 0, minWidth: '200px' }}
            >
              <p className="text-xs font-medium text-[var(--at-muted)] mb-3">Base color</p>
              <div className="grid grid-cols-5 gap-2">
                {BASE_COLORS.map((color) => (
                  <button
                    key={color}
                    onClick={() => handleColorChange(color)}
                    className={`w-8 h-8 rounded-lg transition-all hover:scale-110 ${
                      currentBase.color === color ? 'ring-2 ring-offset-2 ring-primary' : ''
                    }`}
                    style={{ backgroundColor: color, boxShadow: 'var(--shadow-sm)' }}
                  />
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Base name */}
        <div className="flex-1 min-w-0">
          {editing ? (
            <input
              ref={inputRef}
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onBlur={handleSave}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSave();
                if (e.key === 'Escape') {
                  setName(currentBase.name);
                  setEditing(false);
                }
              }}
              className="text-xl font-semibold border-0 border-b-2 border-primary bg-transparent px-1 py-0.5 focus:outline-none w-full"
              style={{ maxWidth: '400px' }}
            />
          ) : (
            <button
              type="button"
              className="text-left group flex items-center gap-2"
              onClick={() => {
                setName(currentBase.name);
                setEditing(true);
              }}
            >
              <span className="text-xl font-semibold text-[var(--at-text)] group-hover:text-primary transition-colors truncate">
                {currentBase.name}
              </span>
              <svg
                className="w-4 h-4 text-[var(--at-muted)] opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
              </svg>
            </button>
          )}
        </div>

        {/* Right side actions */}
        <div className="flex items-center gap-2">
          <button className="toolbar-btn">
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z" />
            </svg>
            Share
          </button>
        </div>
      </div>
    </div>
  );
}
