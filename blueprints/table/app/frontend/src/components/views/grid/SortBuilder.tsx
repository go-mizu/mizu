import { useState, useRef, useEffect } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { Sort } from '../../../types';

interface SortBuilderProps {
  isOpen: boolean;
  onClose: () => void;
  anchorRef?: React.RefObject<HTMLElement | null>;
}

export function SortBuilder({ isOpen, onClose, anchorRef }: SortBuilderProps) {
  const { fields, currentView, setSorts } = useBaseStore();
  const [localSorts, setLocalSorts] = useState<Sort[]>([]);
  const panelRef = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState({ top: 0, left: 0 });
  const [dragIndex, setDragIndex] = useState<number | null>(null);

  // Initialize sorts from view
  useEffect(() => {
    if (currentView?.sorts) {
      setLocalSorts(currentView.sorts.length > 0 ? currentView.sorts : []);
    } else {
      setLocalSorts([]);
    }
  }, [currentView?.sorts, isOpen]);

  // Position panel below anchor
  useEffect(() => {
    if (isOpen && anchorRef?.current) {
      const rect = anchorRef.current.getBoundingClientRect();
      setPosition({
        top: rect.bottom + 4,
        left: Math.max(8, rect.left),
      });
    }
  }, [isOpen, anchorRef]);

  // Close on outside click
  useEffect(() => {
    if (!isOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node) &&
          anchorRef?.current && !anchorRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [isOpen, onClose, anchorRef]);

  const sortableFields = fields.filter(f =>
    !['button', 'attachment', 'link'].includes(f.type)
  );

  const addSort = () => {
    // Find first field not already used in sorts
    const usedFieldIds = new Set(localSorts.map(s => s.field_id));
    const availableField = sortableFields.find(f => !usedFieldIds.has(f.id));

    if (!availableField) {
      // All fields used, just use first field
      if (sortableFields.length === 0) return;
      setLocalSorts([
        ...localSorts,
        { field_id: sortableFields[0].id, direction: 'asc' },
      ]);
      return;
    }

    setLocalSorts([
      ...localSorts,
      { field_id: availableField.id, direction: 'asc' },
    ]);
  };

  const updateSort = (index: number, updates: Partial<Sort>) => {
    const newSorts = [...localSorts];
    newSorts[index] = { ...newSorts[index], ...updates };
    setLocalSorts(newSorts);
  };

  const removeSort = (index: number) => {
    setLocalSorts(localSorts.filter((_, i) => i !== index));
  };

  const applySorts = () => {
    setSorts(localSorts);
    onClose();
  };

  const clearSorts = () => {
    setLocalSorts([]);
    setSorts([]);
  };

  const handleDragStart = (e: React.DragEvent, index: number) => {
    setDragIndex(index);
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    if (dragIndex === null || dragIndex === index) return;

    const newSorts = [...localSorts];
    const [removed] = newSorts.splice(dragIndex, 1);
    newSorts.splice(index, 0, removed);
    setLocalSorts(newSorts);
    setDragIndex(index);
  };

  const handleDragEnd = () => {
    setDragIndex(null);
  };

  if (!isOpen) return null;

  return (
    <div
      ref={panelRef}
      className="fixed z-50 bg-white rounded-lg shadow-xl border border-slate-200 min-w-[360px]"
      style={{ top: position.top, left: position.left }}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200">
        <div className="flex items-center gap-2">
          <svg className="w-4 h-4 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12" />
          </svg>
          <span className="font-medium text-slate-700">Sort</span>
          {localSorts.length > 0 && (
            <span className="px-1.5 py-0.5 text-xs bg-primary-100 text-primary-700 rounded-full">
              {localSorts.length}
            </span>
          )}
        </div>
        <button onClick={onClose} className="text-slate-400 hover:text-slate-600">
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Sort rows */}
      <div className="p-4 space-y-2 max-h-[300px] overflow-y-auto">
        {localSorts.length === 0 ? (
          <div className="text-sm text-slate-500 text-center py-4">
            No sorts applied. Add a sort to order records.
          </div>
        ) : (
          localSorts.map((sort, index) => {
            const _field = fields.find(f => f.id === sort.field_id);
            void _field; // Used for future field-type specific UI

            return (
              <div
                key={index}
                draggable
                onDragStart={(e) => handleDragStart(e, index)}
                onDragOver={(e) => handleDragOver(e, index)}
                onDragEnd={handleDragEnd}
                className={`flex items-center gap-2 p-2 rounded-md border border-slate-200 bg-white ${
                  dragIndex === index ? 'opacity-50' : ''
                }`}
              >
                {/* Drag handle */}
                <div className="cursor-move text-slate-400 hover:text-slate-600">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
                  </svg>
                </div>

                {/* Sort number */}
                <span className="w-5 h-5 flex items-center justify-center text-xs font-medium bg-slate-100 text-slate-600 rounded">
                  {index + 1}
                </span>

                {/* Field selector */}
                <select
                  value={sort.field_id}
                  onChange={(e) => updateSort(index, { field_id: e.target.value })}
                  className="flex-1 min-w-[120px] px-2 py-1.5 text-sm border border-slate-200 rounded-md focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  {sortableFields.map(f => (
                    <option key={f.id} value={f.id}>{f.name}</option>
                  ))}
                </select>

                {/* Direction toggle */}
                <button
                  onClick={() => updateSort(index, { direction: sort.direction === 'asc' ? 'desc' : 'asc' })}
                  className={`flex items-center gap-1 px-2 py-1.5 text-sm rounded-md border transition-colors ${
                    sort.direction === 'asc'
                      ? 'border-primary-200 bg-primary-50 text-primary-700'
                      : 'border-slate-200 bg-slate-50 text-slate-700'
                  }`}
                >
                  {sort.direction === 'asc' ? (
                    <>
                      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
                      </svg>
                      <span>A-Z</span>
                    </>
                  ) : (
                    <>
                      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                      </svg>
                      <span>Z-A</span>
                    </>
                  )}
                </button>

                {/* Remove button */}
                <button
                  onClick={() => removeSort(index)}
                  className="p-1.5 text-slate-400 hover:text-red-500 rounded hover:bg-red-50"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            );
          })
        )}

        {/* Add sort button */}
        {localSorts.length < 3 && (
          <button
            onClick={addSort}
            className="flex items-center gap-2 text-sm text-primary hover:text-primary-dark"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add sort
          </button>
        )}

        {localSorts.length >= 3 && (
          <p className="text-xs text-slate-500 italic">Maximum 3 sort levels</p>
        )}
      </div>

      {/* Footer */}
      <div className="flex items-center justify-between px-4 py-3 border-t border-slate-200 bg-slate-50">
        <button
          onClick={clearSorts}
          className="text-sm text-slate-600 hover:text-slate-800"
        >
          Clear all
        </button>
        <div className="flex items-center gap-2">
          <button
            onClick={onClose}
            className="px-3 py-1.5 text-sm text-slate-600 hover:bg-slate-100 rounded-md"
          >
            Cancel
          </button>
          <button
            onClick={applySorts}
            className="px-3 py-1.5 text-sm text-white bg-primary hover:bg-primary-dark rounded-md"
          >
            Apply sort
          </button>
        </div>
      </div>
    </div>
  );
}
