import { useState, useEffect, useRef, useCallback } from 'react';
import type { Field, TableRecord } from '../../../types';
import { cellValueToString } from './clipboardUtils';

interface SearchMatch {
  recordId: string;
  fieldId: string;
  rowIndex: number;
  colIndex: number;
}

interface SearchBarProps {
  records: TableRecord[];
  fields: Field[];
  isOpen: boolean;
  onClose: () => void;
  onNavigate: (recordId: string, fieldId: string) => void;
}

export function SearchBar({ records, fields, isOpen, onClose, onNavigate }: SearchBarProps) {
  const [query, setQuery] = useState('');
  const [matches, setMatches] = useState<SearchMatch[]>([]);
  const [currentIndex, setCurrentIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  // Focus input when opened
  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [isOpen]);

  // Search when query changes
  useEffect(() => {
    if (!query.trim()) {
      setMatches([]);
      setCurrentIndex(0);
      return;
    }

    const searchTerm = query.toLowerCase();
    const newMatches: SearchMatch[] = [];

    records.forEach((record, rowIndex) => {
      fields.forEach((field, colIndex) => {
        const value = record.values[field.id];
        const stringValue = cellValueToString(value, field).toLowerCase();
        if (stringValue.includes(searchTerm)) {
          newMatches.push({
            recordId: record.id,
            fieldId: field.id,
            rowIndex,
            colIndex,
          });
        }
      });
    });

    setMatches(newMatches);
    setCurrentIndex(0);

    // Navigate to first match
    if (newMatches.length > 0) {
      onNavigate(newMatches[0].recordId, newMatches[0].fieldId);
    }
  }, [query, records, fields, onNavigate]);

  const goToNext = useCallback(() => {
    if (matches.length === 0) return;
    const nextIndex = (currentIndex + 1) % matches.length;
    setCurrentIndex(nextIndex);
    onNavigate(matches[nextIndex].recordId, matches[nextIndex].fieldId);
  }, [matches, currentIndex, onNavigate]);

  const goToPrev = useCallback(() => {
    if (matches.length === 0) return;
    const prevIndex = (currentIndex - 1 + matches.length) % matches.length;
    setCurrentIndex(prevIndex);
    onNavigate(matches[prevIndex].recordId, matches[prevIndex].fieldId);
  }, [matches, currentIndex, onNavigate]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClose();
    } else if (e.key === 'Enter') {
      if (e.shiftKey) {
        goToPrev();
      } else {
        goToNext();
      }
    }
  };

  if (!isOpen) return null;

  return (
    <div className="absolute top-0 right-0 mt-2 mr-4 bg-white rounded-lg shadow-xl border border-gray-200 z-50 flex items-center gap-2 p-2">
      <svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>

      <input
        ref={inputRef}
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Find in view..."
        className="w-48 text-sm border-0 focus:outline-none focus:ring-0"
      />

      {matches.length > 0 && (
        <span className="text-xs text-gray-500 whitespace-nowrap">
          {currentIndex + 1} of {matches.length}
        </span>
      )}

      {query && matches.length === 0 && (
        <span className="text-xs text-gray-400">No results</span>
      )}

      <div className="flex items-center gap-1 border-l border-gray-200 pl-2">
        <button
          onClick={goToPrev}
          disabled={matches.length === 0}
          className="p-1 text-gray-400 hover:text-gray-600 disabled:opacity-50"
          title="Previous (Shift+Enter)"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
          </svg>
        </button>
        <button
          onClick={goToNext}
          disabled={matches.length === 0}
          className="p-1 text-gray-400 hover:text-gray-600 disabled:opacity-50"
          title="Next (Enter)"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>
        <button
          onClick={onClose}
          className="p-1 text-gray-400 hover:text-gray-600"
          title="Close (Escape)"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>
  );
}
