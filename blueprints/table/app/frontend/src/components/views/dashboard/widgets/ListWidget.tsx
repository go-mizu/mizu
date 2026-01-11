import type { DashboardWidget, WidgetData, ListData } from '../../../../types';

interface ListWidgetProps {
  widget: DashboardWidget;
  data: WidgetData | undefined;
  isLoading: boolean;
}

export function ListWidget({ widget, data, isLoading }: ListWidgetProps) {
  if (isLoading) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-4 h-full flex flex-col">
        <h3 className="text-sm font-medium text-gray-700 mb-3">{widget.title}</h3>
        <div className="flex-1 space-y-2">
          {[1, 2, 3].map(i => (
            <div key={i} className="animate-pulse flex items-center gap-3 p-2 bg-gray-50 rounded">
              <div className="w-8 h-8 bg-gray-200 rounded-full"></div>
              <div className="flex-1">
                <div className="h-3 bg-gray-200 rounded w-3/4 mb-1"></div>
                <div className="h-2 bg-gray-200 rounded w-1/2"></div>
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  const listData = data?.data as ListData | undefined;
  const records = listData?.records || [];

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4 h-full flex flex-col">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium text-gray-700">{widget.title}</h3>
        {listData && (
          <span className="text-xs text-gray-400">
            {records.length} of {listData.total}
          </span>
        )}
      </div>

      {records.length === 0 ? (
        <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">
          No records found
        </div>
      ) : (
        <div className="flex-1 overflow-auto space-y-2">
          {records.map((record, index) => (
            <RecordItem key={record.id as string || index} record={record} />
          ))}
        </div>
      )}
    </div>
  );
}

function RecordItem({ record }: { record: Record<string, unknown> }) {
  // Get the primary field (first non-id field) for display
  const entries = Object.entries(record).filter(([key]) => key !== 'id');
  const primaryValue = entries[0]?.[1];
  const secondaryEntries = entries.slice(1, 3);

  return (
    <div className="p-2 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer">
      <div className="font-medium text-sm text-gray-900 truncate">
        {formatValue(primaryValue)}
      </div>
      {secondaryEntries.length > 0 && (
        <div className="flex items-center gap-2 mt-1">
          {secondaryEntries.map(([key, value], index) => (
            <span
              key={key}
              className="text-xs text-gray-500 truncate"
            >
              {index > 0 && <span className="mx-1">·</span>}
              {formatValue(value)}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) {
    return '-';
  }

  if (typeof value === 'boolean') {
    return value ? 'Yes' : 'No';
  }

  if (typeof value === 'number') {
    return value.toLocaleString();
  }

  if (value instanceof Date) {
    return value.toLocaleDateString();
  }

  if (Array.isArray(value)) {
    return value.map(v => formatValue(v)).join(', ');
  }

  if (typeof value === 'object') {
    return JSON.stringify(value);
  }

  return String(value);
}
