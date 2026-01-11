import type { DashboardWidget, WidgetData, NumberData } from '../../../../types';

interface NumberWidgetProps {
  widget: DashboardWidget;
  data: WidgetData | undefined;
  isLoading: boolean;
}

export function NumberWidget({ widget, data, isLoading }: NumberWidgetProps) {
  if (isLoading) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-4 h-full flex flex-col justify-center">
        <div className="animate-pulse">
          <div className="h-3 bg-gray-200 rounded w-20 mb-2"></div>
          <div className="h-8 bg-gray-200 rounded w-32"></div>
        </div>
      </div>
    );
  }

  const numberData = data?.data as NumberData | undefined;

  // Determine the color based on aggregation type or custom config
  const getAccentColor = () => {
    switch (widget.config.aggregation) {
      case 'count':
        return 'text-blue-600';
      case 'sum':
        return 'text-green-600';
      case 'avg':
        return 'text-purple-600';
      case 'min':
        return 'text-orange-600';
      case 'max':
        return 'text-red-600';
      case 'percent_filled':
        return 'text-teal-600';
      case 'percent_empty':
        return 'text-gray-600';
      default:
        return 'text-blue-600';
    }
  };

  // Get an icon based on aggregation type
  const getIcon = () => {
    switch (widget.config.aggregation) {
      case 'count':
        return (
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 20l4-16m2 16l4-16M6 9h14M4 15h14" />
          </svg>
        );
      case 'sum':
        return (
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
          </svg>
        );
      case 'avg':
        return (
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
        );
      case 'percent_filled':
      case 'percent_empty':
        return (
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 3.055A9.001 9.001 0 1020.945 13H11V3.055z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.488 9H15V3.512A9.025 9.025 0 0120.488 9z" />
          </svg>
        );
      default:
        return (
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 12l3-3 3 3 4-4M8 21l4-4 4 4M3 4h18M4 4h16v12a1 1 0 01-1 1H5a1 1 0 01-1-1V4z" />
          </svg>
        );
    }
  };

  // Format the value for display
  const displayValue = numberData?.formatted || formatNumber(numberData?.value || 0, widget.config);

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4 h-full flex flex-col justify-center">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm font-medium text-gray-500 mb-1">{widget.title}</p>
          <p className={`text-3xl font-bold ${getAccentColor()}`}>
            {displayValue}
          </p>
          {widget.config.aggregation && (
            <p className="text-xs text-gray-400 mt-1 capitalize">
              {widget.config.aggregation.replace(/_/g, ' ')}
            </p>
          )}
        </div>
        <div className={`${getAccentColor()} p-2 rounded-lg`} style={{ backgroundColor: 'currentColor', opacity: 0.1 }}>
          {getIcon()}
        </div>
      </div>
    </div>
  );
}

function formatNumber(value: number, config: { prefix?: string; suffix?: string; precision?: number }): string {
  let formatted: string;

  // Handle large numbers with abbreviations
  if (Math.abs(value) >= 1000000) {
    formatted = (value / 1000000).toFixed(1) + 'M';
  } else if (Math.abs(value) >= 1000) {
    formatted = (value / 1000).toFixed(1) + 'K';
  } else if (Number.isInteger(value)) {
    formatted = value.toLocaleString();
  } else {
    formatted = value.toFixed(config.precision ?? 2);
  }

  if (config.prefix) {
    formatted = config.prefix + formatted;
  }
  if (config.suffix) {
    formatted = formatted + config.suffix;
  }

  return formatted;
}
