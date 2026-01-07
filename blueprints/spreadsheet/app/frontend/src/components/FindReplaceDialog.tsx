import React, { useState, useEffect, useCallback, useRef } from 'react';
import type { CellPosition } from '../types';

export interface FindReplaceProps {
  isOpen: boolean;
  mode: 'find' | 'replace';
  onClose: () => void;
  onFind: (text: string, options: FindOptions) => CellPosition | null;
  onFindAll: (text: string, options: FindOptions) => CellPosition[];
  onReplace: (findText: string, replaceText: string, options: FindOptions) => Promise<boolean>;
  onReplaceAll: (findText: string, replaceText: string, options: FindOptions) => Promise<number>;
  onNavigateToResult: (position: CellPosition) => void;
}

export interface FindOptions {
  matchCase: boolean;
  matchEntireCell: boolean;
  searchIn: 'values' | 'formulas';
  useRegex: boolean;
}

export const FindReplaceDialog: React.FC<FindReplaceProps> = ({
  isOpen,
  mode: initialMode,
  onClose,
  onFindAll,
  onReplace,
  onReplaceAll,
  onNavigateToResult,
}) => {
  const [mode, setMode] = useState<'find' | 'replace'>(initialMode);
  const [searchText, setSearchText] = useState('');
  const [replaceText, setReplaceText] = useState('');
  const [options, setOptions] = useState<FindOptions>({
    matchCase: false,
    matchEntireCell: false,
    searchIn: 'values',
    useRegex: false,
  });
  const [results, setResults] = useState<CellPosition[]>([]);
  const [currentIndex, setCurrentIndex] = useState(-1);
  const [statusMessage, setStatusMessage] = useState('');

  const searchInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setMode(initialMode);
  }, [initialMode]);

  useEffect(() => {
    if (isOpen && searchInputRef.current) {
      searchInputRef.current.focus();
      searchInputRef.current.select();
    }
  }, [isOpen]);

  useEffect(() => {
    // Clear results when search text or options change
    setResults([]);
    setCurrentIndex(-1);
    setStatusMessage('');
  }, [searchText, options]);

  const handleFind = useCallback(() => {
    if (!searchText.trim()) {
      setStatusMessage('Please enter search text');
      return;
    }

    const allResults = onFindAll(searchText, options);
    setResults(allResults);

    if (allResults.length === 0) {
      setStatusMessage('No matches found');
      setCurrentIndex(-1);
    } else {
      setCurrentIndex(0);
      setStatusMessage(`1 of ${allResults.length}`);
      onNavigateToResult(allResults[0]);
    }
  }, [searchText, options, onFindAll, onNavigateToResult]);

  const handleFindNext = useCallback(() => {
    if (results.length === 0) {
      handleFind();
      return;
    }

    const nextIndex = (currentIndex + 1) % results.length;
    setCurrentIndex(nextIndex);
    setStatusMessage(`${nextIndex + 1} of ${results.length}`);
    onNavigateToResult(results[nextIndex]);
  }, [results, currentIndex, handleFind, onNavigateToResult]);

  const handleFindPrevious = useCallback(() => {
    if (results.length === 0) {
      handleFind();
      return;
    }

    const prevIndex = (currentIndex - 1 + results.length) % results.length;
    setCurrentIndex(prevIndex);
    setStatusMessage(`${prevIndex + 1} of ${results.length}`);
    onNavigateToResult(results[prevIndex]);
  }, [results, currentIndex, handleFind, onNavigateToResult]);

  const handleReplace = useCallback(async () => {
    if (!searchText.trim()) {
      setStatusMessage('Please enter search text');
      return;
    }

    const replaced = await onReplace(searchText, replaceText, options);
    if (replaced) {
      // Re-find to update results
      const allResults = onFindAll(searchText, options);
      setResults(allResults);
      if (allResults.length === 0) {
        setStatusMessage('All occurrences replaced');
        setCurrentIndex(-1);
      } else {
        const newIndex = Math.min(currentIndex, allResults.length - 1);
        setCurrentIndex(newIndex);
        setStatusMessage(`${newIndex + 1} of ${allResults.length}`);
        onNavigateToResult(allResults[newIndex]);
      }
    }
  }, [searchText, replaceText, options, currentIndex, onReplace, onFindAll, onNavigateToResult]);

  const handleReplaceAll = useCallback(async () => {
    if (!searchText.trim()) {
      setStatusMessage('Please enter search text');
      return;
    }

    const count = await onReplaceAll(searchText, replaceText, options);
    setResults([]);
    setCurrentIndex(-1);
    setStatusMessage(count > 0 ? `Replaced ${count} occurrence${count > 1 ? 's' : ''}` : 'No matches found');
  }, [searchText, replaceText, options, onReplaceAll]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (e.shiftKey) {
        handleFindPrevious();
      } else {
        handleFindNext();
      }
    } else if (e.key === 'Escape') {
      onClose();
    }
  }, [handleFindNext, handleFindPrevious, onClose]);

  if (!isOpen) return null;

  return (
    <div className="find-replace-overlay">
      <div className="find-replace-dialog" onKeyDown={handleKeyDown}>
        <div className="find-replace-header">
          <div className="find-replace-tabs">
            <button
              className={`find-replace-tab ${mode === 'find' ? 'active' : ''}`}
              onClick={() => setMode('find')}
            >
              Find
            </button>
            <button
              className={`find-replace-tab ${mode === 'replace' ? 'active' : ''}`}
              onClick={() => setMode('replace')}
            >
              Replace
            </button>
          </div>
          <button className="find-replace-close" onClick={onClose}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div className="find-replace-body">
          <div className="find-replace-row">
            <label className="find-replace-label">Find:</label>
            <input
              ref={searchInputRef}
              type="text"
              className="find-replace-input"
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              placeholder="Search..."
            />
          </div>

          {mode === 'replace' && (
            <div className="find-replace-row">
              <label className="find-replace-label">Replace:</label>
              <input
                type="text"
                className="find-replace-input"
                value={replaceText}
                onChange={(e) => setReplaceText(e.target.value)}
                placeholder="Replace with..."
              />
            </div>
          )}

          <div className="find-replace-options">
            <label className="find-replace-option">
              <input
                type="checkbox"
                checked={options.matchCase}
                onChange={(e) => setOptions({ ...options, matchCase: e.target.checked })}
              />
              Match case
            </label>
            <label className="find-replace-option">
              <input
                type="checkbox"
                checked={options.matchEntireCell}
                onChange={(e) => setOptions({ ...options, matchEntireCell: e.target.checked })}
              />
              Match entire cell
            </label>
            <label className="find-replace-option">
              <input
                type="checkbox"
                checked={options.useRegex}
                onChange={(e) => setOptions({ ...options, useRegex: e.target.checked })}
              />
              Regular expression
            </label>
          </div>

          <div className="find-replace-search-in">
            <span>Search in:</span>
            <label>
              <input
                type="radio"
                name="searchIn"
                checked={options.searchIn === 'values'}
                onChange={() => setOptions({ ...options, searchIn: 'values' })}
              />
              Values
            </label>
            <label>
              <input
                type="radio"
                name="searchIn"
                checked={options.searchIn === 'formulas'}
                onChange={() => setOptions({ ...options, searchIn: 'formulas' })}
              />
              Formulas
            </label>
          </div>
        </div>

        <div className="find-replace-footer">
          <div className="find-replace-status">{statusMessage}</div>
          <div className="find-replace-actions">
            {mode === 'replace' && (
              <>
                <button className="find-replace-btn" onClick={handleReplace}>
                  Replace
                </button>
                <button className="find-replace-btn" onClick={handleReplaceAll}>
                  Replace all
                </button>
              </>
            )}
            <button className="find-replace-btn" onClick={handleFindPrevious} disabled={results.length === 0}>
              Previous
            </button>
            <button className="find-replace-btn primary" onClick={handleFindNext}>
              {results.length === 0 ? 'Find' : 'Next'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default FindReplaceDialog;
