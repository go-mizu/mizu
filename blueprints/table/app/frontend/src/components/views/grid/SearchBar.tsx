import { useState, useEffect, useRef, useCallback } from 'react';
import type { Field, TableRecord, CellValue } from '../../../types';
import { cellValueToString, stringToCellValue } from './clipboardUtils';

interface SearchMatch {
  recordId: string;
  fieldId: string;
  rowIndex: number;
  colIndex: number;
  value: CellValue;
}

interface SearchBarProps {
  records: TableRecord[];
  fields: Field[];
  isOpen: boolean;
  onClose: () => void;
  onNavigate: (recordId: string, fieldId: string) => void;
  onReplace?: (recordId: string, fieldId: string, newValue: CellValue) => void;
}

export function SearchBar({ records, fields, isOpen, onClose, onNavigate, onReplace }: SearchBarProps) {
  const [query, setQuery] = useState('');
  const [replaceText, setReplaceText] = useState('');
  const [showReplace, setShowReplace] = useState(false);
  const [matches, setMatches] = useState<SearchMatch[]>([]);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [caseSensitive, setCaseSensitive] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const replaceInputRef = useRef<HTMLInputElement>(null);

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

    const searchTerm = caseSensitive ? query : query.toLowerCase();
    const newMatches: SearchMatch[] = [];

    records.forEach((record, rowIndex) => {
      fields.forEach((field, colIndex) => {
        const value = record.values[field.id];
        const stringValue = cellValueToString(value, field);
        const compareValue = caseSensitive ? stringValue : stringValue.toLowerCase();
        if (compareValue.includes(searchTerm)) {
          newMatches.push({
            recordId: record.id,
            fieldId: field.id,
            rowIndex,
            colIndex,
            value,
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
  }, [query, records, fields, onNavigate, caseSensitive]);

  // Handle replace single
  const handleReplace = useCallback(() => {
    if (matches.length === 0 || !onReplace) return;
    const match = matches[currentIndex];
    const field = fields.find((f) => f.id === match.fieldId);
    if (!field) return;

    const currentValue = cellValueToString(match.value, field);
    const regex = new RegExp(query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), caseSensitive ? 'g' : 'gi');
    const newValue = currentValue.replace(regex, replaceText);
    const convertedValue = stringToCellValue(newValue, field);

    onReplace(match.recordId, match.fieldId, convertedValue);

    // Move to next match after replace
    if (matches.length > 1) {
      goToNext();
    }
  }, [matches, currentIndex, query, replaceText, caseSensitive, onReplace, fields]);

  // Handle replace all
  const handleReplaceAll = useCallback(() => {
    if (matches.length === 0 || !onReplace) return;

    const count = matches.length;
    if (!window.confirm(`Replace ${count} occurrence${count > 1 ? 's' : ''}?`)) return;

    const regex = new RegExp(query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), caseSensitive ? 'g' : 'gi');

    matches.forEach((match) => {
      const field = fields.find((f) => f.id === match.fieldId);
      if (!field) return;

      const currentValue = cellValueToString(match.value, field);
      const newValue = currentValue.replace(regex, replaceText);
      const convertedValue = stringToCellValue(newValue, field);

      onReplace(match.recordId, match.fieldId, convertedValue);
    });

    // Clear matches after replace all
    setMatches([]);
    setCurrentIndex(0);
  }, [matches, query, replaceText, caseSensitive, onReplace, fields]);

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
    <div className="absolute top-0 right-0 mt-2 mr-4 bg-white rounded-lg shadow-xl border border-gray-200 z-50 p-2">
      {/* Search row */}
      <div className="flex items-center gap-2">
        <svg className="w-4 h-4 text-gray-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
          {/* Case sensitive toggle */}
          <button
            onClick={() => setCaseSensitive(!caseSensitive)}
            className={`p-1 text-xs font-semibold rounded ${
              caseSensitive
                ? 'bg-primary-100 text-primary'
                : 'text-gray-400 hover:text-gray-600'
            }`}
            title="Match case"
          >
            Aa
          </button>

          {/* Toggle replace */}
          {onReplace && (
            <button
              onClick={() => setShowReplace(!showReplace)}
              className={`p-1 rounded ${
                showReplace
                  ? 'bg-primary-100 text-primary'
                  : 'text-gray-400 hover:text-gray-600'
              }`}
              title="Find and Replace"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
              </svg>
            </button>
          )}

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

      {/* Replace row */}
      {showReplace && onReplace && (
        <div className="flex items-center gap-2 mt-2 pt-2 border-t border-gray-200">
          <svg className="w-4 h-4 text-gray-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
          </svg>

          <input
            ref={replaceInputRef}
            type="text"
            value={replaceText}
            onChange={(e) => setReplaceText(e.target.value)}
            placeholder="Replace with..."
            className="w-48 text-sm border-0 focus:outline-none focus:ring-0"
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                handleReplace();
              }
            }}
          />

          <div className="flex items-center gap-1 border-l border-gray-200 pl-2">
            <button
              onClick={handleReplace}
              disabled={matches.length === 0}
              className="px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 rounded disabled:opacity-50"
              title="Replace current match"
            >
              Replace
            </button>
            <button
              onClick={handleReplaceAll}
              disabled={matches.length === 0}
              className="px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 rounded disabled:opacity-50"
              title="Replace all matches"
            >
              Replace All
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
