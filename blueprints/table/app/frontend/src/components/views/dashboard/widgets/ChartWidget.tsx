import {
  BarChart, Bar,
  LineChart, Line,
  PieChart, Pie, Cell,
  AreaChart, Area,
  ScatterChart, Scatter,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, ZAxis
} from 'recharts';
import type { DashboardWidget, WidgetData, ChartData, StackingType } from '../../../../types';

interface ChartWidgetProps {
  widget: DashboardWidget;
  data: WidgetData | undefined;
  isLoading: boolean;
}

export function ChartWidget({ widget, data, isLoading }: ChartWidgetProps) {
  if (isLoading) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-4 h-full flex flex-col">
        <h3 className="text-sm font-medium text-gray-700 mb-3">{widget.title}</h3>
        <div className="flex-1 flex items-center justify-center">
          <div className="animate-pulse flex flex-col items-center">
            <div className="w-32 h-32 bg-gray-200 rounded-full"></div>
            <div className="mt-4 h-4 bg-gray-200 rounded w-24"></div>
          </div>
        </div>
      </div>
    );
  }

  const chartData = data?.data as ChartData | undefined;
  if (!chartData || !chartData.labels || chartData.labels.length === 0) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-4 h-full flex flex-col">
        <h3 className="text-sm font-medium text-gray-700 mb-3">{widget.title}</h3>
        <div className="flex-1 flex items-center justify-center text-gray-400">
          No data available
        </div>
      </div>
    );
  }

  // Transform data for recharts
  const transformedData = chartData.labels.map((label, index) => ({
    name: label,
    value: chartData.values[index],
    color: chartData.colors?.[index] || getDefaultColor(index),
  }));

  const chartType = widget.config.chart_type || 'bar';
  const showLegend = widget.config.show_legend !== false;
  const stacking = widget.config.stacking || 'none';
  const hasSeries = chartData.series && chartData.series.length > 0;

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4 h-full flex flex-col">
      <h3 className="text-sm font-medium text-gray-700 mb-3">{widget.title}</h3>
      <div className="flex-1 min-h-0">
        <ResponsiveContainer width="100%" height="100%">
          {chartType === 'bar' && hasSeries && stacking !== 'none'
            ? renderStackedBarChart(chartData, stacking, showLegend)
            : renderChart(chartType, transformedData, showLegend)}
        </ResponsiveContainer>
      </div>
    </div>
  );
}

function renderStackedBarChart(
  chartData: ChartData,
  stacking: StackingType,
  showLegend: boolean
) {
  if (!chartData.series || chartData.series.length === 0) {
    return null;
  }

  // Transform data for recharts stacked bar format
  // Each data point needs all series values as separate keys
  const stackedData = chartData.labels.map((label, labelIndex) => {
    const dataPoint: Record<string, string | number> = { name: label };
    chartData.series!.forEach((series) => {
      dataPoint[series.name] = series.values[labelIndex] || 0;
    });
    return dataPoint;
  });

  // For percent stacking, normalize values to 100%
  if (stacking === 'percent') {
    stackedData.forEach((dataPoint) => {
      const total = chartData.series!.reduce(
        (sum, series) => sum + (dataPoint[series.name] as number),
        0
      );
      if (total > 0) {
        chartData.series!.forEach((series) => {
          dataPoint[series.name] = ((dataPoint[series.name] as number) / total) * 100;
        });
      }
    });
  }

  return (
    <BarChart data={stackedData} layout="vertical">
      <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
      <XAxis
        type="number"
        tick={{ fontSize: 12 }}
        domain={stacking === 'percent' ? [0, 100] : undefined}
        tickFormatter={stacking === 'percent' ? (value) => `${value}%` : undefined}
      />
      <YAxis type="category" dataKey="name" tick={{ fontSize: 12 }} width={100} />
      <Tooltip
        formatter={(value) =>
          stacking === 'percent'
            ? `${(value as number).toFixed(1)}%`
            : (value as number).toLocaleString()
        }
      />
      {showLegend && <Legend />}
      {chartData.series.map((series, index) => (
        <Bar
          key={series.name}
          dataKey={series.name}
          stackId="stack"
          fill={series.color || getDefaultColor(index)}
          radius={index === chartData.series!.length - 1 ? [0, 4, 4, 0] : undefined}
        />
      ))}
    </BarChart>
  );
}

function renderChart(
  chartType: string,
  data: { name: string; value: number; color: string }[],
  showLegend: boolean
) {
  switch (chartType) {
    case 'pie':
    case 'donut':
      return (
        <PieChart>
          <Pie
            data={data}
            dataKey="value"
            nameKey="name"
            cx="50%"
            cy="50%"
            innerRadius={chartType === 'donut' ? '50%' : 0}
            outerRadius="80%"
            label={({ name, percent }) => `${name || ''} (${((percent || 0) * 100).toFixed(0)}%)`}
            labelLine={false}
          >
            {data.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.color} />
            ))}
          </Pie>
          <Tooltip formatter={(value) => (value as number).toLocaleString()} />
          {showLegend && <Legend />}
        </PieChart>
      );

    case 'line':
      return (
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
          <XAxis dataKey="name" tick={{ fontSize: 12 }} />
          <YAxis tick={{ fontSize: 12 }} />
          <Tooltip formatter={(value) => (value as number).toLocaleString()} />
          {showLegend && <Legend />}
          <Line
            type="monotone"
            dataKey="value"
            stroke="#3B82F6"
            strokeWidth={2}
            dot={{ r: 4 }}
            activeDot={{ r: 6 }}
          />
        </LineChart>
      );

    case 'area':
      return (
        <AreaChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
          <XAxis dataKey="name" tick={{ fontSize: 12 }} />
          <YAxis tick={{ fontSize: 12 }} />
          <Tooltip formatter={(value) => (value as number).toLocaleString()} />
          {showLegend && <Legend />}
          <Area
            type="monotone"
            dataKey="value"
            stroke="#3B82F6"
            fill="#3B82F6"
            fillOpacity={0.3}
          />
        </AreaChart>
      );

    case 'scatter':
      // For scatter charts, transform data to x/y format
      const scatterData = data.map((entry, index) => ({
        x: index,
        y: entry.value,
        name: entry.name,
        color: entry.color,
      }));
      return (
        <ScatterChart>
          <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
          <XAxis type="number" dataKey="x" name="Index" tick={{ fontSize: 12 }} />
          <YAxis type="number" dataKey="y" name="Value" tick={{ fontSize: 12 }} />
          <ZAxis range={[60, 400]} />
          <Tooltip
            cursor={{ strokeDasharray: '3 3' }}
            content={({ active, payload }) => {
              if (active && payload && payload.length) {
                const data = payload[0].payload;
                return (
                  <div className="bg-white border border-gray-200 rounded shadow-sm p-2 text-sm">
                    <p className="font-medium">{data.name}</p>
                    <p className="text-gray-600">Value: {data.y.toLocaleString()}</p>
                  </div>
                );
              }
              return null;
            }}
          />
          {showLegend && <Legend />}
          <Scatter name="Data" data={scatterData} fill="#3B82F6">
            {scatterData.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.color} />
            ))}
          </Scatter>
        </ScatterChart>
      );

    case 'bar':
    default:
      return (
        <BarChart data={data} layout="vertical">
          <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
          <XAxis type="number" tick={{ fontSize: 12 }} />
          <YAxis type="category" dataKey="name" tick={{ fontSize: 12 }} width={100} />
          <Tooltip formatter={(value) => (value as number).toLocaleString()} />
          {showLegend && <Legend />}
          <Bar dataKey="value" radius={[0, 4, 4, 0]}>
            {data.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.color} />
            ))}
          </Bar>
        </BarChart>
      );
  }
}

function getDefaultColor(index: number): string {
  const palette = [
    '#3B82F6', // Blue
    '#10B981', // Green
    '#F59E0B', // Amber
    '#EF4444', // Red
    '#8B5CF6', // Purple
    '#EC4899', // Pink
    '#14B8A6', // Teal
    '#F97316', // Orange
  ];
  return palette[index % palette.length];
}
