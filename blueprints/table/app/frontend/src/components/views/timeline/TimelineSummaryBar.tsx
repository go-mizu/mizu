import { useMemo, useState } from 'react';
import type { TableRecord, Field } from '../../../types';

interface SummaryConfig {
  field_id: string;
  aggregation: 'count' | 'sum' | 'avg' | 'min' | 'max' | 'count_filled' | 'count_empty';
}

interface TimelineSummaryBarProps {
  records: TableRecord[];
  fields: Field[];
  summaryConfigs?: SummaryConfig[];
  columns: { date: Date; label: string }[];
  columnWidth: number;
  startField: Field;
  endField?: Field;
}

export function TimelineSummaryBar({
  records,
  fields,
  summaryConfigs,
  columns,
  columnWidth,
  startField,
  endField,
}: TimelineSummaryBarProps) {
  const [showConfigPanel, setShowConfigPanel] = useState(false);

  // Calculate basic summaries
  const summaries = useMemo(() => {
    const result: { label: string; value: string | number; color?: string }[] = [];

    // Always show record count
    result.push({
      label: 'Records',
      value: records.length,
      color: '#3B82F6',
    });

    // Calculate records per time period for utilization
    const recordsPerColumn = columns.map((col, idx) => {
      const colStart = col.date.getTime();
      const colEnd = idx < columns.length - 1
        ? columns[idx + 1].date.getTime()
        : colStart + 86400000;

      return records.filter(record => {
        const recordStart = new Date(record.values[startField.id] as string).getTime();
        const recordEnd = endField && record.values[endField.id]
          ? new Date(record.values[endField.id] as string).getTime()
          : recordStart + 86400000;

        return recordStart < colEnd && recordEnd > colStart;
      }).length;
    });

    const maxRecords = Math.max(...recordsPerColumn, 1);
    const avgRecords = recordsPerColumn.reduce((a, b) => a + b, 0) / recordsPerColumn.length;

    result.push({
      label: 'Avg/Period',
      value: avgRecords.toFixed(1),
      color: '#10B981',
    });

    result.push({
      label: 'Peak',
      value: maxRecords,
      color: '#F59E0B',
    });

    // Sum for numeric fields if configured
    if (summaryConfigs) {
      summaryConfigs.forEach(config => {
        const field = fields.find(f => f.id === config.field_id);
        if (!field) return;

        const numericValues = records
          .map(r => r.values[config.field_id])
          .filter(v => typeof v === 'number') as number[];

        if (numericValues.length === 0) return;

        let value: number;
        switch (config.aggregation) {
          case 'sum':
            value = numericValues.reduce((a, b) => a + b, 0);
            break;
          case 'avg':
            value = numericValues.reduce((a, b) => a + b, 0) / numericValues.length;
            break;
          case 'min':
            value = Math.min(...numericValues);
            break;
          case 'max':
            value = Math.max(...numericValues);
            break;
          case 'count_filled':
            value = records.filter(r => r.values[config.field_id] != null).length;
            break;
          case 'count_empty':
            value = records.filter(r => r.values[config.field_id] == null).length;
            break;
          default:
            value = numericValues.length;
        }

        result.push({
          label: `${field.name} (${config.aggregation})`,
          value: config.aggregation === 'avg' ? value.toFixed(2) : value,
          color: '#8B5CF6',
        });
      });
    }

    return result;
  }, [records, fields, summaryConfigs, columns, startField, endField]);

  // Calculate utilization per column (for the mini chart)
  const utilizationData = useMemo(() => {
    return columns.map((col, idx) => {
      const colStart = col.date.getTime();
      const colEnd = idx < columns.length - 1
        ? columns[idx + 1].date.getTime()
        : colStart + 86400000;

      const count = records.filter(record => {
        const recordStart = new Date(record.values[startField.id] as string).getTime();
        const recordEnd = endField && record.values[endField.id]
          ? new Date(record.values[endField.id] as string).getTime()
          : recordStart + 86400000;

        return recordStart < colEnd && recordEnd > colStart;
      }).length;

      return count;
    });
  }, [records, columns, startField, endField]);

  const maxUtilization = Math.max(...utilizationData, 1);

  return (
    <div className="border-t border-slate-200 bg-white sticky bottom-0 z-30">
      {/* Summary stats row */}
      <div className="flex items-center px-4 py-2 border-b border-slate-100 gap-6">
        <div className="flex items-center gap-1 text-sm text-slate-600">
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          <span className="font-medium">Summary</span>
        </div>

        {summaries.map((summary, idx) => (
          <div key={idx} className="flex items-center gap-2">
            <span
              className="w-2 h-2 rounded-full"
              style={{ backgroundColor: summary.color }}
            />
            <span className="text-sm text-slate-500">{summary.label}:</span>
            <span className="text-sm font-semibold text-slate-700">{summary.value}</span>
          </div>
        ))}

        <button
          onClick={() => setShowConfigPanel(!showConfigPanel)}
          className="ml-auto p-1 text-slate-400 hover:text-slate-600 rounded transition-colors"
          title="Configure summaries"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
          </svg>
        </button>
      </div>

      {/* Utilization chart row */}
      <div className="flex" style={{ height: 32 }}>
        <div className="w-56 flex-shrink-0 px-3 flex items-center border-r border-slate-200 bg-slate-50">
          <span className="text-xs text-slate-500">Utilization</span>
        </div>
        <div className="flex-1 flex items-end" style={{ width: columns.length * columnWidth }}>
          {utilizationData.map((count, idx) => (
            <div
              key={idx}
              className="border-r border-slate-100 flex items-end justify-center"
              style={{ width: columnWidth }}
            >
              <div
                className="w-full mx-0.5 rounded-t transition-all"
                style={{
                  height: `${(count / maxUtilization) * 24}px`,
                  backgroundColor: count > maxUtilization * 0.8
                    ? '#EF4444'
                    : count > maxUtilization * 0.5
                      ? '#F59E0B'
                      : '#10B981',
                  opacity: 0.7,
                }}
                title={`${count} record${count !== 1 ? 's' : ''}`}
              />
            </div>
          ))}
        </div>
      </div>

      {/* Config panel (expandable) */}
      {showConfigPanel && (
        <div className="px-4 py-3 border-t border-slate-200 bg-slate-50">
          <p className="text-xs text-slate-500 mb-2">
            Configure which fields to summarize and how to aggregate them.
          </p>
          <div className="flex flex-wrap gap-2">
            {fields
              .filter(f => ['number', 'currency', 'percent', 'duration'].includes(f.type))
              .map(field => (
                <div key={field.id} className="flex items-center gap-1 text-xs">
                  <span className="text-slate-600">{field.name}:</span>
                  <select className="text-xs border rounded px-1 py-0.5">
                    <option value="">None</option>
                    <option value="sum">Sum</option>
                    <option value="avg">Average</option>
                    <option value="min">Min</option>
                    <option value="max">Max</option>
                  </select>
                </div>
              ))}
          </div>
        </div>
      )}
    </div>
  );
}
