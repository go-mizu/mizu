import { useState, useMemo, useEffect } from 'react';
import type { Field, TableRecord, CellValue } from '../../../types';

interface SummaryBarProps {
  records: TableRecord[];
  fields: Field[];
  columnWidths: Record<string, number>;
  savedConfig?: Record<string, SummaryFunction>;
  onConfigChange?: (config: Record<string, SummaryFunction>) => void;
}

export type SummaryFunction =
  | 'none'
  | 'count_all'
  | 'count_empty'
  | 'count_filled'
  | 'count_unique'
  | 'percent_empty'
  | 'percent_filled'
  | 'sum'
  | 'average'
  | 'min'
  | 'max'
  | 'median'
  | 'range'
  | 'checked'
  | 'unchecked'
  | 'percent_checked'
  | 'earliest'
  | 'latest'
  | 'date_range';

const SUMMARY_OPTIONS: Record<string, { label: string; types: string[] }> = {
  none: { label: 'None', types: ['all'] },
  count_all: { label: 'Count all', types: ['all'] },
  count_empty: { label: 'Empty', types: ['all'] },
  count_filled: { label: 'Filled', types: ['all'] },
  count_unique: { label: 'Unique', types: ['all'] },
  percent_empty: { label: '% Empty', types: ['all'] },
  percent_filled: { label: '% Filled', types: ['all'] },
  sum: { label: 'Sum', types: ['number', 'currency', 'percent', 'rating', 'duration'] },
  average: { label: 'Average', types: ['number', 'currency', 'percent', 'rating', 'duration'] },
  min: { label: 'Min', types: ['number', 'currency', 'percent', 'rating', 'duration'] },
  max: { label: 'Max', types: ['number', 'currency', 'percent', 'rating', 'duration'] },
  median: { label: 'Median', types: ['number', 'currency', 'percent', 'rating', 'duration'] },
  range: { label: 'Range', types: ['number', 'currency', 'percent', 'rating', 'duration'] },
  checked: { label: 'Checked', types: ['checkbox'] },
  unchecked: { label: 'Unchecked', types: ['checkbox'] },
  percent_checked: { label: '% Checked', types: ['checkbox'] },
  earliest: { label: 'Earliest', types: ['date', 'datetime', 'created_time', 'last_modified_time'] },
  latest: { label: 'Latest', types: ['date', 'datetime', 'created_time', 'last_modified_time'] },
  date_range: { label: 'Date range', types: ['date', 'datetime', 'created_time', 'last_modified_time'] },
};

function getOptionsForField(field: Field): { value: SummaryFunction; label: string }[] {
  return Object.entries(SUMMARY_OPTIONS)
    .filter(([_, opt]) => opt.types.includes('all') || opt.types.includes(field.type))
    .map(([value, opt]) => ({ value: value as SummaryFunction, label: opt.label }));
}

function calculateSummary(
  records: TableRecord[],
  field: Field,
  fn: SummaryFunction
): string {
  const values = records.map(r => r.values[field.id]);
  const total = values.length;

  if (total === 0) return '—';

  const isEmpty = (v: CellValue) => v === null || v === undefined || v === '' || (Array.isArray(v) && v.length === 0);
  const emptyCount = values.filter(isEmpty).length;
  const filledCount = total - emptyCount;

  switch (fn) {
    case 'none':
      return '';
    case 'count_all':
      return String(total);
    case 'count_empty':
      return String(emptyCount);
    case 'count_filled':
      return String(filledCount);
    case 'count_unique':
      const unique = new Set(values.filter(v => !isEmpty(v)).map(v => JSON.stringify(v)));
      return String(unique.size);
    case 'percent_empty':
      return `${Math.round((emptyCount / total) * 100)}%`;
    case 'percent_filled':
      return `${Math.round((filledCount / total) * 100)}%`;
    case 'sum': {
      const nums = values.filter(v => typeof v === 'number') as number[];
      if (nums.length === 0) return '—';
      const sum = nums.reduce((a, b) => a + b, 0);
      return formatNumber(sum, field.type);
    }
    case 'average': {
      const nums = values.filter(v => typeof v === 'number') as number[];
      if (nums.length === 0) return '—';
      const avg = nums.reduce((a, b) => a + b, 0) / nums.length;
      return formatNumber(avg, field.type);
    }
    case 'min': {
      const nums = values.filter(v => typeof v === 'number') as number[];
      if (nums.length === 0) return '—';
      return formatNumber(Math.min(...nums), field.type);
    }
    case 'max': {
      const nums = values.filter(v => typeof v === 'number') as number[];
      if (nums.length === 0) return '—';
      return formatNumber(Math.max(...nums), field.type);
    }
    case 'median': {
      const nums = values.filter(v => typeof v === 'number') as number[];
      if (nums.length === 0) return '—';
      nums.sort((a, b) => a - b);
      const mid = Math.floor(nums.length / 2);
      const median = nums.length % 2 ? nums[mid] : (nums[mid - 1] + nums[mid]) / 2;
      return formatNumber(median, field.type);
    }
    case 'range': {
      const nums = values.filter(v => typeof v === 'number') as number[];
      if (nums.length === 0) return '—';
      const range = Math.max(...nums) - Math.min(...nums);
      return formatNumber(range, field.type);
    }
    case 'checked': {
      const checked = values.filter(v => v === true).length;
      return String(checked);
    }
    case 'unchecked': {
      const unchecked = values.filter(v => v !== true).length;
      return String(unchecked);
    }
    case 'percent_checked': {
      const checked = values.filter(v => v === true).length;
      return `${Math.round((checked / total) * 100)}%`;
    }
    case 'earliest': {
      const dates = values
        .filter(v => v && typeof v === 'string')
        .map(v => new Date(v as string))
        .filter(d => !isNaN(d.getTime()));
      if (dates.length === 0) return '—';
      return new Date(Math.min(...dates.map(d => d.getTime()))).toLocaleDateString();
    }
    case 'latest': {
      const dates = values
        .filter(v => v && typeof v === 'string')
        .map(v => new Date(v as string))
        .filter(d => !isNaN(d.getTime()));
      if (dates.length === 0) return '—';
      return new Date(Math.max(...dates.map(d => d.getTime()))).toLocaleDateString();
    }
    case 'date_range': {
      const dates = values
        .filter(v => v && typeof v === 'string')
        .map(v => new Date(v as string))
        .filter(d => !isNaN(d.getTime()));
      if (dates.length === 0) return '—';
      const earliest = new Date(Math.min(...dates.map(d => d.getTime())));
      const latest = new Date(Math.max(...dates.map(d => d.getTime())));
      const days = Math.round((latest.getTime() - earliest.getTime()) / (1000 * 60 * 60 * 24));
      return `${days} days`;
    }
    default:
      return '—';
  }
}

function formatNumber(num: number, type: string): string {
  if (type === 'currency') {
    return `$${num.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
  }
  if (type === 'percent') {
    return `${num.toLocaleString(undefined, { maximumFractionDigits: 1 })}%`;
  }
  return num.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

function getDefaultSummary(field: Field): SummaryFunction {
  if (['number', 'currency', 'percent'].includes(field.type)) {
    return 'sum';
  }
  return 'none';
}

export function SummaryBar({ records, fields, columnWidths, savedConfig, onConfigChange }: SummaryBarProps) {
  const [summaryFunctions, setSummaryFunctions] = useState<Record<string, SummaryFunction>>({});
  const [activeMenu, setActiveMenu] = useState<string | null>(null);

  // Load saved config on mount or when it changes
  useEffect(() => {
    if (savedConfig) {
      setSummaryFunctions(savedConfig);
    }
  }, [savedConfig]);

  const getSummaryFn = (fieldId: string, field: Field): SummaryFunction => {
    return summaryFunctions[fieldId] ?? getDefaultSummary(field);
  };

  const handleSummaryChange = (fieldId: string, fn: SummaryFunction) => {
    const next = { ...summaryFunctions, [fieldId]: fn };
    setSummaryFunctions(next);
    onConfigChange?.(next);
    setActiveMenu(null);
  };

  const summaries = useMemo(() => {
    const result: Record<string, string> = {};
    fields.forEach(field => {
      const fn = getSummaryFn(field.id, field);
      result[field.id] = calculateSummary(records, field, fn);
    });
    return result;
  }, [records, fields, summaryFunctions]);

  return (
    <tr className="bg-slate-50 border-t-2 border-slate-300">
      {/* Row number column */}
      <td className="border-b border-r border-slate-200 p-0">
        <div className="flex items-center justify-center h-8 text-xs text-gray-400">
          Σ
        </div>
      </td>

      {/* Summary cells */}
      {fields.map(field => {
        const fn = getSummaryFn(field.id, field);
        const value = summaries[field.id];
        const options = getOptionsForField(field);

        return (
          <td
            key={field.id}
            className="border-b border-r border-slate-200 p-0 relative"
            style={{ width: columnWidths[field.id] || field.width || 200 }}
          >
            <button
              onClick={() => setActiveMenu(activeMenu === field.id ? null : field.id)}
              className="w-full h-8 px-2 text-left text-xs text-gray-600 hover:bg-slate-100 flex items-center justify-between group"
            >
              <span className="truncate">
                {fn === 'none' ? (
                  <span className="text-gray-400 opacity-0 group-hover:opacity-100">Calculate</span>
                ) : (
                  <>
                    <span className="text-gray-400">{SUMMARY_OPTIONS[fn].label}: </span>
                    <span className="font-medium text-gray-700">{value}</span>
                  </>
                )}
              </span>
              <svg
                className="w-3 h-3 text-gray-400 opacity-0 group-hover:opacity-100 flex-shrink-0"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
            </button>

            {activeMenu === field.id && (
              <>
                <div className="fixed inset-0 z-40" onClick={() => setActiveMenu(null)} />
                <div className="absolute bottom-full left-0 mb-1 bg-white rounded-lg shadow-xl border border-gray-200 z-50 animate-slide-in min-w-[140px]">
                  {options.map(opt => (
                    <button
                      key={opt.value}
                      onClick={() => handleSummaryChange(field.id, opt.value)}
                      className={`w-full px-3 py-1.5 text-left text-sm hover:bg-slate-50 ${
                        fn === opt.value ? 'text-primary font-medium' : 'text-gray-700'
                      }`}
                    >
                      {opt.label}
                    </button>
                  ))}
                </div>
              </>
            )}
          </td>
        );
      })}

      {/* Add field column */}
      <td className="border-b border-slate-200" />
    </tr>
  );
}
