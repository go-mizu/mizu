import { useEffect, useState, useMemo, useCallback } from 'react';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  RadialLinearScale,
  Title,
  Tooltip,
  Legend,
  Filler,
  ChartOptions,
  ChartData as ChartJSData,
} from 'chart.js';
import { Line, Bar, Pie, Doughnut, Scatter, Radar, Bubble } from 'react-chartjs-2';
import type { Chart, ChartData, ChartType } from '../../types';
import { api } from '../../utils/api';

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  RadialLinearScale,
  Title,
  Tooltip,
  Legend,
  Filler
);

interface SpreadsheetChartProps {
  chart: Chart;
  onEdit?: () => void;
  selected?: boolean;
  onSelect?: () => void;
}

export function SpreadsheetChart({
  chart,
  onEdit,
  selected,
  onSelect,
}: SpreadsheetChartProps) {
  const [chartData, setChartData] = useState<ChartData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch chart data
  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);
        const data = await api.getChartData(chart.id);
        setChartData(data);
      } catch (err) {
        setError('Failed to load chart data');
        console.error('Failed to fetch chart data:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [chart.id, chart.dataRanges]);

  // Build Chart.js options
  const chartOptions = useMemo<ChartOptions<'line' | 'bar' | 'pie' | 'doughnut' | 'scatter' | 'radar' | 'bubble'>>(() => {
    const options: ChartOptions<'line' | 'bar' | 'pie' | 'doughnut' | 'scatter' | 'radar' | 'bubble'> = {
      responsive: true,
      maintainAspectRatio: false,
      animation: chart.options?.animated !== false ? {
        duration: chart.options?.animationDuration || 750,
      } : false,
      plugins: {
        title: chart.title ? {
          display: true,
          text: chart.title.text,
          font: {
            size: chart.title.fontSize || 16,
            weight: chart.title.bold ? 'bold' : 'normal',
            style: chart.title.italic ? 'italic' : 'normal',
          },
          color: chart.title.fontColor || '#333',
        } : { display: false },
        legend: chart.legend ? {
          display: chart.legend.enabled,
          position: (chart.legend.position || 'bottom') as 'top' | 'bottom' | 'left' | 'right',
        } : { display: true, position: 'bottom' },
        tooltip: {
          enabled: chart.options?.tooltipEnabled !== false,
        },
      },
      interaction: {
        mode: (chart.options?.hoverMode || 'nearest') as 'nearest' | 'point' | 'index' | 'dataset',
        intersect: true,
      },
    };

    // Add axes config for non-circular charts
    if (!['pie', 'doughnut', 'radar'].includes(chart.chartType)) {
      const scales: Record<string, unknown> = {};

      if (chart.axes?.xAxis) {
        scales.x = {
          title: chart.axes.xAxis.title ? {
            display: true,
            text: chart.axes.xAxis.title.text,
          } : { display: false },
          grid: {
            display: chart.axes.xAxis.gridLines,
            color: chart.axes.xAxis.gridColor,
          },
          min: chart.axes.xAxis.min,
          max: chart.axes.xAxis.max,
          reverse: chart.axes.xAxis.reversed,
        };
      }

      if (chart.axes?.yAxis) {
        scales.y = {
          title: chart.axes.yAxis.title ? {
            display: true,
            text: chart.axes.yAxis.title.text,
          } : { display: false },
          grid: {
            display: chart.axes.yAxis.gridLines !== false,
            color: chart.axes.yAxis.gridColor,
          },
          min: chart.axes.yAxis.min,
          max: chart.axes.yAxis.max,
          reverse: chart.axes.yAxis.reversed,
          type: chart.axes.yAxis.logarithmic ? 'logarithmic' : 'linear',
        };
      }

      if (chart.axes?.y2Axis) {
        scales.y1 = {
          position: 'right',
          title: chart.axes.y2Axis.title ? {
            display: true,
            text: chart.axes.y2Axis.title.text,
          } : { display: false },
          grid: {
            display: false,
          },
        };
      }

      (options as Record<string, unknown>).scales = scales;
    }

    return options;
  }, [chart]);

  // Build Chart.js data
  const chartJsData = useMemo<ChartJSData<'line' | 'bar' | 'pie' | 'doughnut' | 'scatter' | 'radar' | 'bubble'> | null>(() => {
    if (!chartData) return null;

    return {
      labels: chartData.labels,
      datasets: chartData.datasets.map((ds, idx) => ({
        label: ds.label,
        data: ds.data,
        backgroundColor: ds.backgroundColor || getDefaultColor(idx, 0.6),
        borderColor: ds.borderColor || getDefaultColor(idx),
        borderWidth: ds.borderWidth || 2,
        fill: ds.fill,
        tension: ds.tension || 0,
        pointRadius: ds.pointRadius || 3,
        pointStyle: ds.pointStyle,
      })),
    } as ChartJSData<'line' | 'bar' | 'pie' | 'doughnut' | 'scatter' | 'radar' | 'bubble'>;
  }, [chartData]);

  const handleContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  if (loading) {
    return (
      <div
        className="chart-container"
        style={{
          width: chart.size.width,
          height: chart.size.height,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          backgroundColor: '#f9f9f9',
          border: '1px solid #e0e0e0',
          borderRadius: 4,
        }}
      >
        <span style={{ color: '#666' }}>Loading chart...</span>
      </div>
    );
  }

  if (error || !chartJsData) {
    return (
      <div
        className="chart-container"
        style={{
          width: chart.size.width,
          height: chart.size.height,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          backgroundColor: '#fff5f5',
          border: '1px solid #fecaca',
          borderRadius: 4,
        }}
      >
        <span style={{ color: '#dc2626' }}>{error || 'No data'}</span>
      </div>
    );
  }

  const ChartComponent = getChartComponent(chart.chartType);

  return (
    <div
      className="chart-container"
      style={{
        width: chart.size.width,
        height: chart.size.height,
        backgroundColor: chart.options?.backgroundColor || '#fff',
        border: selected ? '2px solid #2196F3' : '1px solid #e0e0e0',
        borderRadius: chart.options?.borderRadius || 4,
        cursor: 'pointer',
        position: 'relative',
        padding: 8,
        boxSizing: 'border-box',
      }}
      onClick={onSelect}
      onDoubleClick={onEdit}
      onContextMenu={handleContextMenu}
    >
      <ChartComponent
        data={chartJsData as never}
        options={chartOptions as never}
      />

      {/* Resize handles when selected */}
      {selected && (
        <>
          <div
            style={{
              position: 'absolute',
              top: -4,
              left: -4,
              width: 8,
              height: 8,
              backgroundColor: '#2196F3',
              borderRadius: '50%',
              cursor: 'nw-resize',
            }}
          />
          <div
            style={{
              position: 'absolute',
              top: -4,
              right: -4,
              width: 8,
              height: 8,
              backgroundColor: '#2196F3',
              borderRadius: '50%',
              cursor: 'ne-resize',
            }}
          />
          <div
            style={{
              position: 'absolute',
              bottom: -4,
              left: -4,
              width: 8,
              height: 8,
              backgroundColor: '#2196F3',
              borderRadius: '50%',
              cursor: 'sw-resize',
            }}
          />
          <div
            style={{
              position: 'absolute',
              bottom: -4,
              right: -4,
              width: 8,
              height: 8,
              backgroundColor: '#2196F3',
              borderRadius: '50%',
              cursor: 'se-resize',
            }}
          />
        </>
      )}
    </div>
  );
}

// Get the appropriate Chart.js component
function getChartComponent(chartType: ChartType) {
  switch (chartType) {
    case 'line':
    case 'area':
    case 'stacked_area':
      return Line;
    case 'bar':
    case 'column':
    case 'stacked_bar':
    case 'stacked_column':
    case 'histogram':
    case 'waterfall':
      return Bar;
    case 'pie':
      return Pie;
    case 'doughnut':
      return Doughnut;
    case 'scatter':
      return Scatter;
    case 'radar':
      return Radar;
    case 'bubble':
      return Bubble;
    case 'combo':
      return Bar; // Combo uses Bar with mixed types
    default:
      return Line;
  }
}

// Default colors
const DEFAULT_COLORS = [
  '#4CAF50', // Green
  '#2196F3', // Blue
  '#FF9800', // Orange
  '#E91E63', // Pink
  '#9C27B0', // Purple
  '#00BCD4', // Cyan
  '#FFC107', // Amber
  '#795548', // Brown
  '#607D8B', // Blue Grey
  '#F44336', // Red
];

function getDefaultColor(index: number, alpha = 1): string {
  const color = DEFAULT_COLORS[index % DEFAULT_COLORS.length];
  if (alpha === 1) return color;

  // Convert hex to rgba
  const r = parseInt(color.slice(1, 3), 16);
  const g = parseInt(color.slice(3, 5), 16);
  const b = parseInt(color.slice(5, 7), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

export default SpreadsheetChart;
