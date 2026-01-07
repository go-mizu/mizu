import React, { useMemo } from 'react';
import type { Cell, Selection } from '../types';

export interface StatusBarProps {
  cells: Map<string, Cell>;
  selection: Selection | null;
  zoom: number;
  onZoomChange: (zoom: number) => void;
}

interface SelectionStats {
  sum: number | null;
  average: number | null;
  count: number;
  countNumbers: number;
  min: number | null;
  max: number | null;
}

export const StatusBar: React.FC<StatusBarProps> = ({
  cells,
  selection,
  zoom,
  onZoomChange,
}) => {
  const stats = useMemo((): SelectionStats => {
    if (!selection) {
      return { sum: null, average: null, count: 0, countNumbers: 0, min: null, max: null };
    }

    const numbers: number[] = [];
    let count = 0;

    for (let row = selection.startRow; row <= selection.endRow; row++) {
      for (let col = selection.startCol; col <= selection.endCol; col++) {
        const cell = cells.get(`${row}:${col}`);
        if (cell && cell.value !== null && cell.value !== '') {
          count++;
          const numValue = typeof cell.value === 'number'
            ? cell.value
            : parseFloat(String(cell.value));
          if (!isNaN(numValue) && isFinite(numValue)) {
            numbers.push(numValue);
          }
        }
      }
    }

    if (numbers.length === 0) {
      return { sum: null, average: null, count, countNumbers: 0, min: null, max: null };
    }

    const sum = numbers.reduce((a, b) => a + b, 0);
    const average = sum / numbers.length;
    const min = Math.min(...numbers);
    const max = Math.max(...numbers);

    return {
      sum,
      average,
      count,
      countNumbers: numbers.length,
      min,
      max,
    };
  }, [cells, selection]);

  const formatNumber = (value: number | null): string => {
    if (value === null) return '-';
    if (Math.abs(value) >= 1000000) {
      return value.toExponential(2);
    }
    return value.toLocaleString(undefined, {
      minimumFractionDigits: 0,
      maximumFractionDigits: 2,
    });
  };

  const hasSelection = selection &&
    (selection.startRow !== selection.endRow || selection.startCol !== selection.endCol);

  return (
    <div className="status-bar">
      <div className="status-bar-left">
        {hasSelection && stats.countNumbers > 0 && (
          <>
            <span className="status-stat" title="Sum of selected numbers">
              <span className="status-stat-label">Sum:</span>
              <span className="status-stat-value">{formatNumber(stats.sum)}</span>
            </span>
            <span className="status-stat" title="Average of selected numbers">
              <span className="status-stat-label">Average:</span>
              <span className="status-stat-value">{formatNumber(stats.average)}</span>
            </span>
            <span className="status-stat" title="Count of non-empty cells">
              <span className="status-stat-label">Count:</span>
              <span className="status-stat-value">{stats.count}</span>
            </span>
            {stats.countNumbers > 1 && (
              <>
                <span className="status-stat" title="Minimum value">
                  <span className="status-stat-label">Min:</span>
                  <span className="status-stat-value">{formatNumber(stats.min)}</span>
                </span>
                <span className="status-stat" title="Maximum value">
                  <span className="status-stat-label">Max:</span>
                  <span className="status-stat-value">{formatNumber(stats.max)}</span>
                </span>
              </>
            )}
          </>
        )}
        {hasSelection && stats.countNumbers === 0 && stats.count > 0 && (
          <span className="status-stat" title="Count of non-empty cells">
            <span className="status-stat-label">Count:</span>
            <span className="status-stat-value">{stats.count}</span>
          </span>
        )}
      </div>
      <div className="status-bar-right">
        <div className="status-zoom">
          <button
            className="status-zoom-btn"
            onClick={() => onZoomChange(Math.max(50, zoom - 25))}
            disabled={zoom <= 50}
            title="Zoom out"
          >
            <MinusIcon />
          </button>
          <span className="status-zoom-value" title="Current zoom level">{zoom}%</span>
          <button
            className="status-zoom-btn"
            onClick={() => onZoomChange(Math.min(200, zoom + 25))}
            disabled={zoom >= 200}
            title="Zoom in"
          >
            <PlusIcon />
          </button>
        </div>
      </div>
    </div>
  );
};

const MinusIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="5" y1="12" x2="19" y2="12" />
  </svg>
);

const PlusIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="12" y1="5" x2="12" y2="19" />
    <line x1="5" y1="12" x2="19" y2="12" />
  </svg>
);

export default StatusBar;
