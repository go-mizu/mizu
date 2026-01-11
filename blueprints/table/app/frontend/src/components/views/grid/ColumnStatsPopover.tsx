import { useMemo } from 'react';
import type { TableRecord, Field } from '../../../types';

interface ColumnStatsPopoverProps {
  field: Field;
  records: TableRecord[];
  position: { x: number; y: number };
  onClose: () => void;
}

interface ColumnStats {
  total: number;
  filled: number;
  empty: number;
  unique: number;
  // Numeric stats
  sum?: number;
  avg?: number;
  min?: number | string;
  max?: number | string;
  median?: number;
  // Text stats
  shortest?: string;
  longest?: string;
  avgLength?: number;
  // Boolean stats
  checked?: number;
  unchecked?: number;
}

export function ColumnStatsPopover({
  field,
  records,
  position,
  onClose,
}: ColumnStatsPopoverProps) {
  const stats = useMemo<ColumnStats>(() => {
    const values = records.map((r) => r.values[field.id]);
    const filled = values.filter((v) => v !== null && v !== undefined && v !== '');
    const empty = values.length - filled.length;

    // Get unique values
    const uniqueSet = new Set(
      filled.map((v) => (typeof v === 'object' ? JSON.stringify(v) : String(v)))
    );

    const baseStats: ColumnStats = {
      total: values.length,
      filled: filled.length,
      empty,
      unique: uniqueSet.size,
    };

    // Numeric fields
    if (['number', 'currency', 'percent', 'rating', 'duration'].includes(field.type)) {
      const numbers = filled
        .filter((v): v is number => typeof v === 'number')
        .sort((a, b) => a - b);

      if (numbers.length > 0) {
        const sum = numbers.reduce((a, b) => a + b, 0);
        baseStats.sum = sum;
        baseStats.avg = sum / numbers.length;
        baseStats.min = numbers[0];
        baseStats.max = numbers[numbers.length - 1];
        baseStats.median =
          numbers.length % 2 === 0
            ? (numbers[numbers.length / 2 - 1] + numbers[numbers.length / 2]) / 2
            : numbers[Math.floor(numbers.length / 2)];
      }
    }

    // Text fields
    if (['text', 'single_line_text', 'long_text', 'email', 'url', 'phone'].includes(field.type)) {
      const strings = filled.filter((v): v is string => typeof v === 'string');

      if (strings.length > 0) {
        const sorted = [...strings].sort((a, b) => a.length - b.length);
        baseStats.shortest = sorted[0];
        baseStats.longest = sorted[sorted.length - 1];
        baseStats.avgLength = strings.reduce((a, b) => a + b.length, 0) / strings.length;
      }
    }

    // Date fields
    if (['date', 'date_time', 'created_time', 'last_modified_time'].includes(field.type)) {
      const dates = filled
        .map((v) => (typeof v === 'string' ? new Date(v) : null))
        .filter((d): d is Date => d !== null && !isNaN(d.getTime()))
        .sort((a, b) => a.getTime() - b.getTime());

      if (dates.length > 0) {
        baseStats.min = dates[0].toLocaleDateString();
        baseStats.max = dates[dates.length - 1].toLocaleDateString();
      }
    }

    // Checkbox fields
    if (field.type === 'checkbox') {
      baseStats.checked = filled.filter((v) => v === true).length;
      baseStats.unchecked = values.length - (baseStats.checked || 0);
    }

    return baseStats;
  }, [field, records]);

  // Format number for display
  const formatNumber = (num: number | undefined, decimals = 2): string => {
    if (num === undefined) return '-';
    return num.toLocaleString(undefined, {
      minimumFractionDigits: 0,
      maximumFractionDigits: decimals,
    });
  };

  // Format percentage
  const formatPercent = (value: number, total: number): string => {
    if (total === 0) return '0%';
    return `${((value / total) * 100).toFixed(1)}%`;
  };

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 z-40" onClick={onClose} />

      {/* Popover */}
      <div
        className="fixed z-50 bg-white rounded-lg shadow-xl border border-slate-200 w-64 py-2 animate-slide-in"
        style={{ left: position.x, top: position.y }}
      >
        {/* Header */}
        <div className="px-3 pb-2 mb-2 border-b border-slate-200">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-slate-700 truncate">
              {field.name}
            </span>
            <span className="text-xs text-slate-400 capitalize">
              {field.type.replace(/_/g, ' ')}
            </span>
          </div>
        </div>

        {/* Basic stats */}
        <div className="px-3 py-1 grid grid-cols-2 gap-2 text-sm">
          <div>
            <span className="text-slate-500">Total</span>
            <span className="float-right text-slate-700">{stats.total}</span>
          </div>
          <div>
            <span className="text-slate-500">Unique</span>
            <span className="float-right text-slate-700">{stats.unique}</span>
          </div>
          <div>
            <span className="text-slate-500">Filled</span>
            <span className="float-right text-slate-700">
              {stats.filled} <span className="text-slate-400 text-xs">({formatPercent(stats.filled, stats.total)})</span>
            </span>
          </div>
          <div>
            <span className="text-slate-500">Empty</span>
            <span className="float-right text-slate-700">
              {stats.empty} <span className="text-slate-400 text-xs">({formatPercent(stats.empty, stats.total)})</span>
            </span>
          </div>
        </div>

        {/* Numeric stats */}
        {stats.sum !== undefined && (
          <>
            <hr className="my-2 border-slate-200" />
            <div className="px-3 py-1 grid grid-cols-2 gap-2 text-sm">
              <div>
                <span className="text-slate-500">Sum</span>
                <span className="float-right text-slate-700">{formatNumber(stats.sum)}</span>
              </div>
              <div>
                <span className="text-slate-500">Average</span>
                <span className="float-right text-slate-700">{formatNumber(stats.avg)}</span>
              </div>
              <div>
                <span className="text-slate-500">Min</span>
                <span className="float-right text-slate-700">{formatNumber(stats.min as number)}</span>
              </div>
              <div>
                <span className="text-slate-500">Max</span>
                <span className="float-right text-slate-700">{formatNumber(stats.max as number)}</span>
              </div>
              <div className="col-span-2">
                <span className="text-slate-500">Median</span>
                <span className="float-right text-slate-700">{formatNumber(stats.median)}</span>
              </div>
            </div>
          </>
        )}

        {/* Date stats */}
        {(field.type.includes('date') || field.type.includes('time')) && stats.min && (
          <>
            <hr className="my-2 border-slate-200" />
            <div className="px-3 py-1 grid grid-cols-1 gap-2 text-sm">
              <div>
                <span className="text-slate-500">Earliest</span>
                <span className="float-right text-slate-700">{stats.min}</span>
              </div>
              <div>
                <span className="text-slate-500">Latest</span>
                <span className="float-right text-slate-700">{stats.max}</span>
              </div>
            </div>
          </>
        )}

        {/* Text stats */}
        {stats.avgLength !== undefined && (
          <>
            <hr className="my-2 border-slate-200" />
            <div className="px-3 py-1 grid grid-cols-1 gap-2 text-sm">
              <div>
                <span className="text-slate-500">Avg length</span>
                <span className="float-right text-slate-700">{formatNumber(stats.avgLength, 0)} chars</span>
              </div>
              <div>
                <span className="text-slate-500">Shortest</span>
                <span className="float-right text-slate-700 truncate max-w-[120px]" title={stats.shortest}>
                  {stats.shortest}
                </span>
              </div>
              <div>
                <span className="text-slate-500">Longest</span>
                <span className="float-right text-slate-700 truncate max-w-[120px]" title={stats.longest}>
                  {stats.longest}
                </span>
              </div>
            </div>
          </>
        )}

        {/* Checkbox stats */}
        {field.type === 'checkbox' && (
          <>
            <hr className="my-2 border-slate-200" />
            <div className="px-3 py-1 grid grid-cols-2 gap-2 text-sm">
              <div>
                <span className="text-slate-500">Checked</span>
                <span className="float-right text-slate-700">
                  {stats.checked} <span className="text-slate-400 text-xs">({formatPercent(stats.checked || 0, stats.total)})</span>
                </span>
              </div>
              <div>
                <span className="text-slate-500">Unchecked</span>
                <span className="float-right text-slate-700">
                  {stats.unchecked} <span className="text-slate-400 text-xs">({formatPercent(stats.unchecked || 0, stats.total)})</span>
                </span>
              </div>
            </div>
          </>
        )}

        {/* Distribution bar */}
        {stats.total > 0 && (
          <div className="px-3 pt-2 mt-2 border-t border-slate-200">
            <div className="flex h-2 rounded-full overflow-hidden bg-slate-100">
              <div
                className="bg-primary"
                style={{ width: formatPercent(stats.filled, stats.total) }}
                title={`Filled: ${stats.filled}`}
              />
              <div
                className="bg-slate-300"
                style={{ width: formatPercent(stats.empty, stats.total) }}
                title={`Empty: ${stats.empty}`}
              />
            </div>
          </div>
        )}
      </div>
    </>
  );
}
