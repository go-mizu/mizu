import { useMemo } from 'react'
import { Box, Text, Paper, Skeleton, Group, Badge } from '@mantine/core'
import {
  LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, AreaChart, Area,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
  ScatterChart, Scatter, ComposedChart, FunnelChart, Funnel, LabelList,
  ReferenceLine, ZAxis
} from 'recharts'
import type { QueryResult, VisualizationSettings, VisualizationType } from '../../api/types'
import { chartColors } from '../../theme'
import { TableVisualization } from './table'
import type { TableSettings } from './table'
import SankeyVisualization from './SankeyVisualization'

interface VisualizationProps {
  result: QueryResult
  visualization: VisualizationSettings
  height?: number
  showLegend?: boolean
}

export default function Visualization({
  result,
  visualization,
  height = 400,
  showLegend = true,
}: VisualizationProps) {
  const { type, settings = {} } = visualization

  if (!result || !result.rows || result.rows.length === 0) {
    return (
      <Paper p="xl" ta="center" bg="gray.0" radius="md">
        <Text c="dimmed">No data to display</Text>
      </Paper>
    )
  }

  const data = result.rows
  const columns = result.columns

  switch (type) {
    case 'number':
      return <NumberVisualization data={data} columns={columns} settings={settings} />
    case 'trend':
      return <TrendVisualization data={data} columns={columns} settings={settings} />
    case 'progress':
      return <ProgressVisualization data={data} columns={columns} settings={settings} />
    case 'gauge':
      return <GaugeVisualization data={data} columns={columns} settings={settings} />
    case 'line':
      return <LineVisualization data={data} columns={columns} settings={settings} height={height} showLegend={showLegend} />
    case 'area':
      return <AreaVisualization data={data} columns={columns} settings={settings} height={height} showLegend={showLegend} />
    case 'bar':
      return <BarVisualization data={data} columns={columns} settings={settings} height={height} showLegend={showLegend} />
    case 'row':
      return <RowVisualization data={data} columns={columns} settings={settings} height={height} showLegend={showLegend} />
    case 'pie':
    case 'donut':
      return <PieVisualization data={data} columns={columns} settings={settings} height={height} showLegend={showLegend} donut={type === 'donut'} />
    case 'scatter':
      return <ScatterVisualization data={data} columns={columns} settings={settings} height={height} showLegend={showLegend} />
    case 'funnel':
      return <FunnelVisualization data={data} columns={columns} settings={settings} height={height} />
    case 'combo':
      return <ComboVisualization data={data} columns={columns} settings={settings} height={height} showLegend={showLegend} />
    case 'waterfall':
      return <WaterfallVisualization data={data} columns={columns} settings={settings} height={height} />
    case 'bubble':
      return <BubbleVisualization data={data} columns={columns} settings={settings} height={height} showLegend={showLegend} />
    case 'map-pin':
    case 'map-grid':
    case 'map-region':
      return <MapVisualization data={data} columns={columns} settings={settings} height={height} type={type} />
    case 'pivot':
      return <PivotVisualization data={data} columns={columns} settings={settings} />
    case 'sankey':
      return <SankeyVisualization data={data} columns={columns} settings={settings} height={height} />
    case 'table':
    default:
      return (
        <TableVisualization
          result={result}
          settings={settings as TableSettings}
          height={height}
        />
      )
  }
}

// Loading skeleton for visualizations
export function VisualizationSkeleton({ height = 400 }: { height?: number }) {
  return <Skeleton height={height} radius="md" />
}

// Format number with compact notation and style
function formatNumber(value: number, settings: Record<string, any>): string {
  const prefix = settings.prefix || ''
  const suffix = settings.suffix || ''
  const decimals = settings.decimals ?? 0
  const compact = settings.compact ?? false
  const style = settings.style || 'decimal'

  let formatted: string

  if (compact && Math.abs(value) >= 1000) {
    const absValue = Math.abs(value)
    if (absValue >= 1e9) {
      formatted = (value / 1e9).toFixed(decimals) + 'B'
    } else if (absValue >= 1e6) {
      formatted = (value / 1e6).toFixed(decimals) + 'M'
    } else if (absValue >= 1e3) {
      formatted = (value / 1e3).toFixed(decimals) + 'K'
    } else {
      formatted = value.toFixed(decimals)
    }
  } else {
    const options: Intl.NumberFormatOptions = {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    }

    if (style === 'percent') {
      options.style = 'percent'
    } else if (style === 'currency') {
      options.style = 'currency'
      options.currency = settings.currency || 'USD'
    } else if (style === 'scientific') {
      formatted = value.toExponential(decimals)
      return prefix + formatted + suffix
    }

    formatted = value.toLocaleString(undefined, options)
  }

  return prefix + formatted + suffix
}

// Number / Scalar visualization
function NumberVisualization({
  data,
  columns,
  settings = {},
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
}) {
  const value = data[0]?.[columns[0]?.name]

  const formattedValue = useMemo(() => {
    if (typeof value === 'number') {
      return formatNumber(value, settings)
    }
    return String(value ?? '-')
  }, [value, settings])

  return (
    <Paper p="xl" ta="center">
      <Text size="3rem" fw={700} c="brand.5">
        {formattedValue}
      </Text>
      <Text c="dimmed" mt="xs">{columns[0]?.display_name || columns[0]?.name}</Text>
    </Paper>
  )
}

// Trend visualization (number with trend indicator)
function TrendVisualization({
  data,
  columns,
  settings = {},
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
}) {
  const currentValue = data[0]?.[columns[0]?.name]
  const previousValue = data[1]?.[columns[0]?.name] || data[0]?.[columns[1]?.name]
  const reverseColors = settings.reverseColors ?? false

  const { formattedValue, change } = useMemo(() => {
    const formatted = typeof currentValue === 'number'
      ? formatNumber(currentValue, settings)
      : String(currentValue)

    let changePercent = null
    if (typeof currentValue === 'number' && typeof previousValue === 'number' && previousValue !== 0) {
      changePercent = ((currentValue - previousValue) / previousValue) * 100
    }

    return { formattedValue: formatted, change: changePercent }
  }, [currentValue, previousValue, settings])

  // Determine colors based on reverseColors setting
  const isPositiveGood = !reverseColors
  const positiveColor = isPositiveGood ? 'green' : 'red'
  const negativeColor = isPositiveGood ? 'red' : 'green'

  return (
    <Paper p="xl" ta="center">
      <Text size="3rem" fw={700} c="brand.5">
        {formattedValue}
      </Text>
      {change !== null && (
        <Badge
          size="lg"
          mt="sm"
          color={change >= 0 ? positiveColor : negativeColor}
          variant="light"
        >
          {change >= 0 ? '+' : ''}{change.toFixed(1)}%
        </Badge>
      )}
      <Text c="dimmed" mt="xs">{columns[0]?.display_name || columns[0]?.name}</Text>
    </Paper>
  )
}

// Progress bar visualization
function ProgressVisualization({
  data,
  columns,
  settings = {},
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
}) {
  const value = data[0]?.[columns[0]?.name]
  const goal = settings.goal || 100
  const color = settings.color || 'var(--mantine-color-brand-5)'
  const showPercentage = settings.showPercentage ?? true
  const percentage = typeof value === 'number' ? Math.min(100, (value / goal) * 100) : 0

  return (
    <Paper p="xl">
      <Group justify="space-between" mb="sm">
        <Text fw={500}>{columns[0]?.display_name || columns[0]?.name}</Text>
        <Group gap="xs">
          <Text fw={700} c="brand.5">
            {typeof value === 'number' ? value.toLocaleString() : String(value)}
          </Text>
          {showPercentage && (
            <Text size="sm" c="dimmed">
              ({percentage.toFixed(0)}%)
            </Text>
          )}
        </Group>
      </Group>
      <Box
        style={{
          height: 12,
          backgroundColor: 'var(--mantine-color-gray-2)',
          borderRadius: 6,
          overflow: 'hidden',
        }}
      >
        <Box
          style={{
            height: '100%',
            width: `${percentage}%`,
            backgroundColor: color,
            borderRadius: 6,
            transition: 'width 0.3s ease',
          }}
        />
      </Box>
      <Group justify="space-between" mt="xs">
        <Text size="sm" c="dimmed">0</Text>
        <Text size="sm" c="dimmed">Goal: {goal.toLocaleString()}</Text>
      </Group>
    </Paper>
  )
}

// Gauge visualization with colored segments
function GaugeVisualization({
  data,
  columns,
  settings = {},
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
}) {
  const value = data[0]?.[columns[0]?.name]
  const min = settings.min ?? 0
  const max = settings.max ?? 100
  const warningThreshold = settings.warningThreshold ?? 60
  const successThreshold = settings.successThreshold ?? 80

  const percentage = typeof value === 'number'
    ? Math.min(100, Math.max(0, ((value - min) / (max - min)) * 100))
    : 0

  // Determine color based on thresholds
  const getColor = () => {
    const normalizedValue = (percentage / 100) * (max - min) + min
    if (normalizedValue >= successThreshold) return 'var(--mantine-color-green-6)'
    if (normalizedValue >= warningThreshold) return 'var(--mantine-color-yellow-6)'
    return 'var(--mantine-color-red-6)'
  }

  // Calculate segment percentages for the arc
  const warningPercent = ((warningThreshold - min) / (max - min)) * 100
  const successPercent = ((successThreshold - min) / (max - min)) * 100

  return (
    <Paper p="xl" ta="center">
      <Box style={{ position: 'relative', width: 200, height: 120, margin: '0 auto' }}>
        <svg viewBox="0 0 200 120" width="200" height="120">
          {/* Background segments */}
          {/* Red segment (0 to warning) */}
          <path
            d={describeArc(100, 100, 80, 180, 180 + (warningPercent * 1.8))}
            fill="none"
            stroke="var(--mantine-color-red-2)"
            strokeWidth="16"
            strokeLinecap="butt"
          />
          {/* Yellow segment (warning to success) */}
          <path
            d={describeArc(100, 100, 80, 180 + (warningPercent * 1.8), 180 + (successPercent * 1.8))}
            fill="none"
            stroke="var(--mantine-color-yellow-2)"
            strokeWidth="16"
            strokeLinecap="butt"
          />
          {/* Green segment (success to max) */}
          <path
            d={describeArc(100, 100, 80, 180 + (successPercent * 1.8), 360)}
            fill="none"
            stroke="var(--mantine-color-green-2)"
            strokeWidth="16"
            strokeLinecap="butt"
          />
          {/* Value arc */}
          <path
            d={describeArc(100, 100, 80, 180, 180 + (percentage * 1.8))}
            fill="none"
            stroke={getColor()}
            strokeWidth="16"
            strokeLinecap="round"
          />
        </svg>
        <Text
          size="xl"
          fw={700}
          style={{
            position: 'absolute',
            bottom: 10,
            left: '50%',
            transform: 'translateX(-50%)',
            color: getColor(),
          }}
        >
          {typeof value === 'number' ? value.toLocaleString() : String(value)}
        </Text>
      </Box>
      <Group justify="center" gap="xl" mt="xs">
        <Text size="xs" c="dimmed">{min}</Text>
        <Text size="xs" c="dimmed">{max}</Text>
      </Group>
      <Text c="dimmed" mt="md">{columns[0]?.display_name || columns[0]?.name}</Text>
    </Paper>
  )
}

// Helper function to describe SVG arc path
function describeArc(x: number, y: number, radius: number, startAngle: number, endAngle: number): string {
  const start = polarToCartesian(x, y, radius, endAngle)
  const end = polarToCartesian(x, y, radius, startAngle)
  const largeArcFlag = endAngle - startAngle <= 180 ? '0' : '1'
  return `M ${start.x} ${start.y} A ${radius} ${radius} 0 ${largeArcFlag} 0 ${end.x} ${end.y}`
}

function polarToCartesian(cx: number, cy: number, r: number, angle: number) {
  const rad = (angle - 90) * Math.PI / 180
  return {
    x: cx + r * Math.cos(rad),
    y: cy + r * Math.sin(rad),
  }
}

// Line chart visualization with full settings support
function LineVisualization({
  data,
  columns,
  settings = {},
  height,
  showLegend: defaultShowLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
  showLegend: boolean
}) {
  const xKey = columns[0]?.name
  const yKeys = columns.slice(1).map(c => c.name)

  // Apply settings
  const showLegend = settings.showLegend ?? defaultShowLegend
  const showPoints = settings.showPoints ?? true
  const showLabels = settings.showLabels ?? false
  const lineStyle = settings.lineStyle || 'solid'
  const interpolation = settings.interpolation || 'monotone'
  const showGoal = settings.showGoal ?? false
  const goalValue = settings.goalValue
  const goalLabel = settings.goalLabel || 'Goal'
  const showTrend = settings.showTrend ?? false
  const xAxisLabel = settings.xAxisLabel
  const yAxisLabel = settings.yAxisLabel
  const xAxisGrid = settings.xAxisGrid ?? true
  const yAxisMin = settings.yAxisMin
  const yAxisMax = settings.yAxisMax
  const yAxisScale = settings.yAxisScale || 'linear'

  // Calculate trend line if needed
  const trendData = useMemo(() => {
    if (!showTrend || yKeys.length === 0) return null

    const yKey = yKeys[0]
    const points = data.map((d, i) => ({ x: i, y: Number(d[yKey]) || 0 }))
    const n = points.length
    if (n < 2) return null

    // Linear regression
    const sumX = points.reduce((acc, p) => acc + p.x, 0)
    const sumY = points.reduce((acc, p) => acc + p.y, 0)
    const sumXY = points.reduce((acc, p) => acc + p.x * p.y, 0)
    const sumX2 = points.reduce((acc, p) => acc + p.x * p.x, 0)

    const slope = (n * sumXY - sumX * sumY) / (n * sumX2 - sumX * sumX)
    const intercept = (sumY - slope * sumX) / n

    return data.map((d, i) => ({
      ...d,
      _trend: intercept + slope * i,
    }))
  }, [data, showTrend, yKeys])

  const chartData = trendData || data

  // Line stroke dasharray based on style
  const getStrokeDasharray = (style: string) => {
    switch (style) {
      case 'dashed': return '8 4'
      case 'dotted': return '2 2'
      default: return undefined
    }
  }

  return (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={chartData} margin={{ top: 5, right: 20, left: 10, bottom: xAxisLabel ? 30 : 5 }}>
        {xAxisGrid && <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />}
        <XAxis
          dataKey={xKey}
          tick={{ fontSize: 12 }}
          stroke="var(--mantine-color-gray-5)"
          label={xAxisLabel ? { value: xAxisLabel, position: 'bottom', offset: 10 } : undefined}
        />
        <YAxis
          tick={{ fontSize: 12 }}
          stroke="var(--mantine-color-gray-5)"
          domain={[yAxisMin ?? 'auto', yAxisMax ?? 'auto']}
          scale={yAxisScale === 'log' ? 'log' : 'linear'}
          label={yAxisLabel ? { value: yAxisLabel, angle: -90, position: 'insideLeft' } : undefined}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
        />
        {showLegend && <Legend />}
        {showGoal && goalValue !== undefined && (
          <ReferenceLine
            y={goalValue}
            stroke="var(--mantine-color-red-5)"
            strokeDasharray="5 5"
            label={{ value: goalLabel, position: 'right', fill: 'var(--mantine-color-red-5)' }}
          />
        )}
        {yKeys.map((key, i) => (
          <Line
            key={key}
            type={interpolation as 'linear' | 'monotone' | 'step'}
            dataKey={key}
            stroke={chartColors[i % chartColors.length]}
            strokeWidth={2}
            strokeDasharray={getStrokeDasharray(lineStyle)}
            dot={showPoints ? { r: 3 } : false}
            activeDot={{ r: 5 }}
          >
            {showLabels && <LabelList dataKey={key} position="top" fontSize={10} />}
          </Line>
        ))}
        {showTrend && (
          <Line
            type="linear"
            dataKey="_trend"
            stroke="var(--mantine-color-gray-5)"
            strokeWidth={1}
            strokeDasharray="5 5"
            dot={false}
            name="Trend"
          />
        )}
      </LineChart>
    </ResponsiveContainer>
  )
}

// Area chart visualization with full settings support
function AreaVisualization({
  data,
  columns,
  settings = {},
  height,
  showLegend: defaultShowLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
  showLegend: boolean
}) {
  const xKey = columns[0]?.name
  const yKeys = columns.slice(1).map(c => c.name)

  // Apply settings
  const showLegend = settings.showLegend ?? defaultShowLegend
  const showPoints = settings.showPoints ?? false
  const showLabels = settings.showLabels ?? false
  const interpolation = settings.interpolation || 'monotone'
  const stacked = settings.stacked ?? settings.stacking === 'stacked' ?? true
  const normalized = settings.stacking === 'normalized'
  const showGoal = settings.showGoal ?? false
  const goalValue = settings.goalValue
  const goalLabel = settings.goalLabel || 'Goal'
  const xAxisLabel = settings.xAxisLabel
  const yAxisLabel = settings.yAxisLabel
  const xAxisGrid = settings.xAxisGrid ?? true
  const yAxisMin = settings.yAxisMin
  const yAxisMax = settings.yAxisMax

  // Normalize data if needed (100% stacking)
  const chartData = useMemo(() => {
    if (!normalized) return data

    return data.map(row => {
      const total = yKeys.reduce((acc, key) => acc + (Number(row[key]) || 0), 0)
      if (total === 0) return row

      const normalized: Record<string, any> = { ...row }
      yKeys.forEach(key => {
        normalized[key] = ((Number(row[key]) || 0) / total) * 100
      })
      return normalized
    })
  }, [data, yKeys, normalized])

  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={chartData} margin={{ top: 5, right: 20, left: 10, bottom: xAxisLabel ? 30 : 5 }}>
        {xAxisGrid && <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />}
        <XAxis
          dataKey={xKey}
          tick={{ fontSize: 12 }}
          stroke="var(--mantine-color-gray-5)"
          label={xAxisLabel ? { value: xAxisLabel, position: 'bottom', offset: 10 } : undefined}
        />
        <YAxis
          tick={{ fontSize: 12 }}
          stroke="var(--mantine-color-gray-5)"
          domain={normalized ? [0, 100] : [yAxisMin ?? 'auto', yAxisMax ?? 'auto']}
          tickFormatter={normalized ? (v) => `${v}%` : undefined}
          label={yAxisLabel ? { value: yAxisLabel, angle: -90, position: 'insideLeft' } : undefined}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
          formatter={normalized ? (value: number) => `${value.toFixed(1)}%` : undefined}
        />
        {showLegend && <Legend />}
        {showGoal && goalValue !== undefined && !normalized && (
          <ReferenceLine
            y={goalValue}
            stroke="var(--mantine-color-red-5)"
            strokeDasharray="5 5"
            label={{ value: goalLabel, position: 'right', fill: 'var(--mantine-color-red-5)' }}
          />
        )}
        {yKeys.map((key, i) => (
          <Area
            key={key}
            type={interpolation as 'linear' | 'monotone' | 'step'}
            dataKey={key}
            stackId={stacked || normalized ? '1' : undefined}
            stroke={chartColors[i % chartColors.length]}
            fill={chartColors[i % chartColors.length]}
            fillOpacity={0.3}
            dot={showPoints ? { r: 3 } : false}
          >
            {showLabels && <LabelList dataKey={key} position="top" fontSize={10} />}
          </Area>
        ))}
      </AreaChart>
    </ResponsiveContainer>
  )
}

// Bar chart visualization with full settings support
function BarVisualization({
  data,
  columns,
  settings = {},
  height,
  showLegend: defaultShowLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
  showLegend: boolean
}) {
  const xKey = columns[0]?.name
  const yKeys = columns.slice(1).map(c => c.name)

  // Apply settings
  const showLegend = settings.showLegend ?? defaultShowLegend
  const showLabels = settings.showLabels ?? false
  const stacking = settings.stacking || 'none'
  const stacked = stacking === 'stacked' || settings.stacked
  const normalized = stacking === 'normalized'
  const showGoal = settings.showGoal ?? false
  const goalValue = settings.goalValue
  const xAxisLabel = settings.xAxisLabel
  const yAxisLabel = settings.yAxisLabel
  const yAxisMin = settings.yAxisMin
  const yAxisMax = settings.yAxisMax

  // Normalize data if needed (100% stacking)
  const chartData = useMemo(() => {
    if (!normalized) return data

    return data.map(row => {
      const total = yKeys.reduce((acc, key) => acc + (Number(row[key]) || 0), 0)
      if (total === 0) return row

      const normalized: Record<string, any> = { ...row }
      yKeys.forEach(key => {
        normalized[key] = ((Number(row[key]) || 0) / total) * 100
      })
      return normalized
    })
  }, [data, yKeys, normalized])

  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={chartData} margin={{ top: 5, right: 20, left: 10, bottom: xAxisLabel ? 30 : 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
        <XAxis
          dataKey={xKey}
          tick={{ fontSize: 12 }}
          stroke="var(--mantine-color-gray-5)"
          label={xAxisLabel ? { value: xAxisLabel, position: 'bottom', offset: 10 } : undefined}
        />
        <YAxis
          tick={{ fontSize: 12 }}
          stroke="var(--mantine-color-gray-5)"
          domain={normalized ? [0, 100] : [yAxisMin ?? 'auto', yAxisMax ?? 'auto']}
          tickFormatter={normalized ? (v) => `${v}%` : undefined}
          label={yAxisLabel ? { value: yAxisLabel, angle: -90, position: 'insideLeft' } : undefined}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
          formatter={normalized ? (value: number) => `${value.toFixed(1)}%` : undefined}
        />
        {showLegend && <Legend />}
        {showGoal && goalValue !== undefined && !normalized && (
          <ReferenceLine
            y={goalValue}
            stroke="var(--mantine-color-red-5)"
            strokeDasharray="5 5"
            label={{ value: 'Goal', position: 'right', fill: 'var(--mantine-color-red-5)' }}
          />
        )}
        {yKeys.map((key, i) => (
          <Bar
            key={key}
            dataKey={key}
            stackId={stacked || normalized ? '1' : undefined}
            fill={chartColors[i % chartColors.length]}
            radius={stacked || normalized ? (i === yKeys.length - 1 ? [4, 4, 0, 0] : 0) : [4, 4, 0, 0]}
          >
            {showLabels && <LabelList dataKey={key} position="top" fontSize={10} />}
          </Bar>
        ))}
      </BarChart>
    </ResponsiveContainer>
  )
}

// Horizontal bar (row) chart visualization
function RowVisualization({
  data,
  columns,
  settings = {},
  height,
  showLegend: defaultShowLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
  showLegend: boolean
}) {
  const yKey = columns[0]?.name
  const xKeys = columns.slice(1).map(c => c.name)

  // Apply settings
  const showLegend = settings.showLegend ?? defaultShowLegend
  const showLabels = settings.showLabels ?? false
  const stacking = settings.stacking || 'none'
  const stacked = stacking === 'stacked' || settings.stacked
  const normalized = stacking === 'normalized'
  const showGoal = settings.showGoal ?? false
  const goalValue = settings.goalValue

  // Normalize data if needed
  const chartData = useMemo(() => {
    if (!normalized) return data

    return data.map(row => {
      const total = xKeys.reduce((acc, key) => acc + (Number(row[key]) || 0), 0)
      if (total === 0) return row

      const normalized: Record<string, any> = { ...row }
      xKeys.forEach(key => {
        normalized[key] = ((Number(row[key]) || 0) / total) * 100
      })
      return normalized
    })
  }, [data, xKeys, normalized])

  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={chartData} layout="vertical" margin={{ top: 5, right: 20, left: 80, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
        <XAxis
          type="number"
          tick={{ fontSize: 12 }}
          stroke="var(--mantine-color-gray-5)"
          domain={normalized ? [0, 100] : undefined}
          tickFormatter={normalized ? (v) => `${v}%` : undefined}
        />
        <YAxis
          dataKey={yKey}
          type="category"
          tick={{ fontSize: 12 }}
          stroke="var(--mantine-color-gray-5)"
          width={70}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
          formatter={normalized ? (value: number) => `${value.toFixed(1)}%` : undefined}
        />
        {showLegend && <Legend />}
        {showGoal && goalValue !== undefined && !normalized && (
          <ReferenceLine
            x={goalValue}
            stroke="var(--mantine-color-red-5)"
            strokeDasharray="5 5"
          />
        )}
        {xKeys.map((key, i) => (
          <Bar
            key={key}
            dataKey={key}
            stackId={stacked || normalized ? '1' : undefined}
            fill={chartColors[i % chartColors.length]}
            radius={[0, 4, 4, 0]}
          >
            {showLabels && <LabelList dataKey={key} position="right" fontSize={10} />}
          </Bar>
        ))}
      </BarChart>
    </ResponsiveContainer>
  )
}

// Pie/Donut chart visualization with full settings support
function PieVisualization({
  data,
  columns,
  settings = {},
  height,
  showLegend: defaultShowLegend,
  donut,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
  showLegend: boolean
  donut: boolean
}) {
  const nameKey = columns[0]?.name
  const valueKey = columns[1]?.name

  // Apply settings
  const showLegend = settings.showLegend ?? defaultShowLegend
  const showLabels = settings.showLabels ?? true
  const showPercentages = settings.showPercentages ?? true
  const minSlicePercent = settings.minSlicePercent ?? 0
  const showTotal = settings.showTotal ?? donut

  // Calculate total and group small slices
  const { chartData, total } = useMemo(() => {
    const total = data.reduce((acc, row) => acc + (Number(row[valueKey]) || 0), 0)

    if (minSlicePercent <= 0 || total === 0) {
      return { chartData: data, total }
    }

    const threshold = (minSlicePercent / 100) * total
    const largeSlices: Record<string, any>[] = []
    let otherTotal = 0

    data.forEach(row => {
      const value = Number(row[valueKey]) || 0
      if (value < threshold) {
        otherTotal += value
      } else {
        largeSlices.push(row)
      }
    })

    if (otherTotal > 0) {
      largeSlices.push({ [nameKey]: 'Other', [valueKey]: otherTotal })
    }

    return { chartData: largeSlices, total }
  }, [data, nameKey, valueKey, minSlicePercent])

  // Custom label renderer
  const renderLabel = ({ name, percent }: { name: string; percent: number }) => {
    if (!showLabels) return null
    if (showPercentages) {
      return `${name}: ${(percent * 100).toFixed(0)}%`
    }
    return name
  }

  return (
    <ResponsiveContainer width="100%" height={height}>
      <PieChart>
        <Pie
          data={chartData}
          dataKey={valueKey}
          nameKey={nameKey}
          cx="50%"
          cy="50%"
          outerRadius={height / 3}
          innerRadius={donut ? height / 5 : 0}
          paddingAngle={2}
          label={showLabels ? renderLabel : false}
          labelLine={showLabels ? { stroke: 'var(--mantine-color-gray-5)' } : false}
        >
          {chartData.map((_, index) => (
            <Cell key={`cell-${index}`} fill={chartColors[index % chartColors.length]} />
          ))}
        </Pie>
        <Tooltip
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
          formatter={(value: number) => [
            showPercentages ? `${value.toLocaleString()} (${((value / total) * 100).toFixed(1)}%)` : value.toLocaleString(),
            valueKey
          ]}
        />
        {showLegend && <Legend />}
        {showTotal && donut && (
          <text x="50%" y="50%" textAnchor="middle" dominantBaseline="middle">
            <tspan x="50%" dy="-0.5em" fontSize="14" fill="var(--mantine-color-gray-6)">Total</tspan>
            <tspan x="50%" dy="1.4em" fontSize="18" fontWeight="bold" fill="var(--mantine-color-gray-8)">{total.toLocaleString()}</tspan>
          </text>
        )}
      </PieChart>
    </ResponsiveContainer>
  )
}

// Scatter chart visualization with full settings support
function ScatterVisualization({
  data,
  columns,
  settings = {},
  height,
  showLegend: defaultShowLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
  showLegend: boolean
}) {
  const xKey = columns[0]?.name
  const yKey = columns[1]?.name

  // Apply settings
  const showLegend = settings.showLegend ?? defaultShowLegend
  const dotSize = settings.dotSize ?? 8
  const showTrend = settings.showTrend ?? false
  const showGoal = settings.showGoal ?? false
  const goalValue = settings.goalValue

  // Calculate trend line if needed
  const trendLine = useMemo(() => {
    if (!showTrend) return null

    const points = data.map(d => ({
      x: Number(d[xKey]) || 0,
      y: Number(d[yKey]) || 0,
    }))
    const n = points.length
    if (n < 2) return null

    // Linear regression
    const sumX = points.reduce((acc, p) => acc + p.x, 0)
    const sumY = points.reduce((acc, p) => acc + p.y, 0)
    const sumXY = points.reduce((acc, p) => acc + p.x * p.y, 0)
    const sumX2 = points.reduce((acc, p) => acc + p.x * p.x, 0)

    const slope = (n * sumXY - sumX * sumY) / (n * sumX2 - sumX * sumX)
    const intercept = (sumY - slope * sumX) / n

    const minX = Math.min(...points.map(p => p.x))
    const maxX = Math.max(...points.map(p => p.x))

    return [
      { x: minX, y: intercept + slope * minX },
      { x: maxX, y: intercept + slope * maxX },
    ]
  }, [data, xKey, yKey, showTrend])

  return (
    <ResponsiveContainer width="100%" height={height}>
      <ScatterChart margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
        <XAxis
          dataKey="x"
          name={xKey}
          tick={{ fontSize: 12 }}
          stroke="var(--mantine-color-gray-5)"
          type="number"
          domain={['auto', 'auto']}
        />
        <YAxis
          dataKey="y"
          name={yKey}
          tick={{ fontSize: 12 }}
          stroke="var(--mantine-color-gray-5)"
          type="number"
          domain={['auto', 'auto']}
        />
        <Tooltip
          cursor={{ strokeDasharray: '3 3' }}
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
        />
        {showLegend && <Legend />}
        {showGoal && goalValue !== undefined && (
          <ReferenceLine
            y={goalValue}
            stroke="var(--mantine-color-red-5)"
            strokeDasharray="5 5"
          />
        )}
        <Scatter
          name={yKey}
          data={data.map(d => ({ x: d[xKey], y: d[yKey], ...d }))}
          fill={chartColors[0]}
        >
          {data.map((_, index) => (
            <Cell
              key={`cell-${index}`}
              fill={chartColors[0]}
              // @ts-ignore
              r={dotSize / 2}
            />
          ))}
        </Scatter>
        {showTrend && trendLine && (
          <Scatter
            name="Trend"
            data={trendLine}
            fill="transparent"
            line={{ stroke: 'var(--mantine-color-gray-5)', strokeDasharray: '5 5' }}
            shape={() => null}
          />
        )}
      </ScatterChart>
    </ResponsiveContainer>
  )
}

// Funnel chart visualization with settings support
function FunnelVisualization({
  data,
  columns,
  settings = {},
  height,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
}) {
  const nameKey = columns[0]?.name
  const valueKey = columns[1]?.name

  const showLabels = settings.showLabels ?? true
  const showPercentage = settings.showPercentage ?? true

  const funnelData = useMemo(() => {
    const firstValue = Number(data[0]?.[valueKey]) || 1
    return data.map((item, index) => {
      const value = Number(item[valueKey]) || 0
      const percentage = ((value / firstValue) * 100).toFixed(1)
      return {
        name: item[nameKey],
        value,
        fill: chartColors[index % chartColors.length],
        percentage,
      }
    })
  }, [data, nameKey, valueKey])

  return (
    <ResponsiveContainer width="100%" height={height}>
      <FunnelChart>
        <Tooltip
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
          formatter={(value: number, name: string, props: any) => {
            if (showPercentage) {
              return [`${value.toLocaleString()} (${props.payload.percentage}%)`, name]
            }
            return [value.toLocaleString(), name]
          }}
        />
        <Funnel
          dataKey="value"
          data={funnelData}
          isAnimationActive
        >
          {showLabels && (
            <LabelList
              position="right"
              fill="#000"
              stroke="none"
              dataKey="name"
              formatter={(name: string, entry: any) => {
                if (showPercentage && entry?.percentage) {
                  return `${name} (${entry.percentage}%)`
                }
                return name
              }}
            />
          )}
        </Funnel>
      </FunnelChart>
    </ResponsiveContainer>
  )
}

// Combo chart visualization (bar + line)
function ComboVisualization({
  data,
  columns,
  settings = {},
  height,
  showLegend: defaultShowLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
  showLegend: boolean
}) {
  const xKey = columns[0]?.name
  // By default, all but last column are bars, last is line
  // Can be customized via settings.seriesTypes
  const seriesTypes = settings.seriesTypes || {}
  const allKeys = columns.slice(1).map(c => c.name)

  const showLegend = settings.showLegend ?? defaultShowLegend
  const showLabels = settings.showLabels ?? false

  // Determine which series are bars, lines, or areas
  const barKeys: string[] = []
  const lineKeys: string[] = []
  const areaKeys: string[] = []

  allKeys.forEach((key, index) => {
    const type = seriesTypes[key] || (index === allKeys.length - 1 ? 'line' : 'bar')
    if (type === 'line') lineKeys.push(key)
    else if (type === 'area') areaKeys.push(key)
    else barKeys.push(key)
  })

  return (
    <ResponsiveContainer width="100%" height={height}>
      <ComposedChart data={data} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
        <XAxis dataKey={xKey} tick={{ fontSize: 12 }} stroke="var(--mantine-color-gray-5)" />
        <YAxis tick={{ fontSize: 12 }} stroke="var(--mantine-color-gray-5)" />
        <Tooltip
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
        />
        {showLegend && <Legend />}
        {barKeys.map((key, i) => (
          <Bar
            key={key}
            dataKey={key}
            fill={chartColors[allKeys.indexOf(key) % chartColors.length]}
            radius={[4, 4, 0, 0]}
          >
            {showLabels && <LabelList dataKey={key} position="top" fontSize={10} />}
          </Bar>
        ))}
        {areaKeys.map((key) => (
          <Area
            key={key}
            type="monotone"
            dataKey={key}
            fill={chartColors[allKeys.indexOf(key) % chartColors.length]}
            stroke={chartColors[allKeys.indexOf(key) % chartColors.length]}
            fillOpacity={0.3}
          />
        ))}
        {lineKeys.map((key) => (
          <Line
            key={key}
            type="monotone"
            dataKey={key}
            stroke={chartColors[allKeys.indexOf(key) % chartColors.length]}
            strokeWidth={2}
            dot={{ r: 3 }}
          />
        ))}
      </ComposedChart>
    </ResponsiveContainer>
  )
}

// Waterfall chart visualization with settings support
function WaterfallVisualization({
  data,
  columns,
  settings = {},
  height,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
}) {
  const xKey = columns[0]?.name
  const valueKey = columns[1]?.name

  const showLabels = settings.showLabels ?? false
  const increaseColor = settings.increaseColor || 'var(--mantine-color-green-5)'
  const decreaseColor = settings.decreaseColor || 'var(--mantine-color-red-5)'
  const totalColor = settings.totalColor || 'var(--mantine-color-brand-5)'

  // Transform data for waterfall: calculate running total and invisible base
  const waterfallData = useMemo(() => {
    let cumulative = 0
    return data.map((item, index) => {
      const value = Number(item[valueKey]) || 0
      const name = String(item[xKey] || '')
      const isTotal = index === data.length - 1 && name.toLowerCase().includes('total')
      const base = isTotal ? 0 : cumulative
      cumulative += value

      return {
        ...item,
        _base: isTotal ? 0 : Math.min(base, base + value),
        _value: Math.abs(value),
        _positive: value >= 0,
        _isTotal: isTotal,
        _cumulative: cumulative,
        _displayValue: value,
      }
    })
  }, [data, xKey, valueKey])

  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={waterfallData} margin={{ top: 20, right: 20, left: 10, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
        <XAxis dataKey={xKey} tick={{ fontSize: 12 }} stroke="var(--mantine-color-gray-5)" />
        <YAxis tick={{ fontSize: 12 }} stroke="var(--mantine-color-gray-5)" />
        <Tooltip
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
          formatter={(value: any, name: string, props: any) => {
            if (name === '_base') return null
            const displayValue = props.payload._displayValue
            return [displayValue >= 0 ? `+${displayValue.toLocaleString()}` : displayValue.toLocaleString(), valueKey]
          }}
        />
        {/* Invisible base bar for positioning */}
        <Bar dataKey="_base" stackId="waterfall" fill="transparent" />
        {/* Value bars with conditional coloring */}
        <Bar
          dataKey="_value"
          stackId="waterfall"
          radius={[4, 4, 0, 0]}
        >
          {waterfallData.map((entry, index) => (
            <Cell
              key={`cell-${index}`}
              fill={entry._isTotal ? totalColor : entry._positive ? increaseColor : decreaseColor}
            />
          ))}
          {showLabels && (
            <LabelList
              dataKey="_displayValue"
              position="top"
              fontSize={10}
              formatter={(value: number) => value >= 0 ? `+${value}` : value}
            />
          )}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  )
}

// Bubble chart visualization (scatter with variable size)
function BubbleVisualization({
  data,
  columns,
  settings = {},
  height,
  showLegend: defaultShowLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
  showLegend: boolean
}) {
  const xKey = columns[0]?.name
  const yKey = columns[1]?.name
  const sizeKey = columns[2]?.name // Third column for bubble size

  const showLegend = settings.showLegend ?? defaultShowLegend
  const minBubbleSize = settings.minBubbleSize ?? 20
  const maxBubbleSize = settings.maxBubbleSize ?? 400

  // Calculate size scale
  const { minSize, maxSize } = useMemo(() => {
    if (!sizeKey) return { minSize: 1, maxSize: 1 }
    const values = data.map(d => Number(d[sizeKey]) || 0).filter(v => v > 0)
    return {
      minSize: Math.min(...values),
      maxSize: Math.max(...values),
    }
  }, [data, sizeKey])

  return (
    <ResponsiveContainer width="100%" height={height}>
      <ScatterChart margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
        <XAxis
          dataKey={xKey}
          name={xKey}
          tick={{ fontSize: 12 }}
          stroke="var(--mantine-color-gray-5)"
          type="number"
        />
        <YAxis
          dataKey={yKey}
          name={yKey}
          tick={{ fontSize: 12 }}
          stroke="var(--mantine-color-gray-5)"
          type="number"
        />
        <ZAxis
          dataKey={sizeKey}
          range={[minBubbleSize, maxBubbleSize]}
          domain={[minSize, maxSize]}
        />
        <Tooltip
          cursor={{ strokeDasharray: '3 3' }}
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
        />
        {showLegend && <Legend />}
        <Scatter name={yKey} data={data} fill={chartColors[0]}>
          {data.map((_, index) => (
            <Cell
              key={`cell-${index}`}
              fill={chartColors[index % chartColors.length]}
              fillOpacity={0.7}
            />
          ))}
        </Scatter>
      </ScatterChart>
    </ResponsiveContainer>
  )
}

// Map visualization - displays geographic data in a tile/grid format
function MapVisualization({
  data,
  columns,
  settings = {},
  height,
  type,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
  type: 'map-pin' | 'map-grid' | 'map-region'
}) {
  const locationKey = columns[0]?.name
  const valueKey = columns[1]?.name

  const baseColor = settings.baseColor || '#509EE3'

  // Calculate value range for color scaling
  const { minValue, maxValue } = useMemo(() => {
    const values = data.map(d => Number(d[valueKey]) || 0)
    return {
      minValue: Math.min(...values),
      maxValue: Math.max(...values),
    }
  }, [data, valueKey])

  // Get color based on value (heat map style)
  const getColor = (value: number) => {
    if (maxValue === minValue) return baseColor
    const normalized = (value - minValue) / (maxValue - minValue)
    // Extract base color components and vary lightness
    const r = parseInt(baseColor.slice(1, 3), 16)
    const g = parseInt(baseColor.slice(3, 5), 16)
    const b = parseInt(baseColor.slice(5, 7), 16)

    // Lighten for low values, darken for high values
    const lightness = 1 - normalized * 0.6
    return `rgb(${Math.round(r * lightness + 255 * (1 - lightness) * 0.7)}, ${Math.round(g * lightness + 255 * (1 - lightness) * 0.7)}, ${Math.round(b * lightness + 255 * (1 - lightness) * 0.7)})`
  }

  // Format value for display
  const formatValue = (value: number) => {
    if (value >= 1000000) return `${(value / 1000000).toFixed(1)}M`
    if (value >= 1000) return `${(value / 1000).toFixed(1)}K`
    return value.toLocaleString()
  }

  return (
    <Box style={{ height, overflow: 'auto', padding: 8 }}>
      <Text size="xs" c="dimmed" mb="sm">
        Geographic Distribution ({type === 'map-pin' ? 'Pin' : type === 'map-grid' ? 'Grid' : 'Region'} Map)
      </Text>
      <Box
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(120px, 1fr))',
          gap: 8,
        }}
      >
        {data.map((item, index) => {
          const location = item[locationKey]
          const value = Number(item[valueKey]) || 0
          const color = getColor(value)
          const isDark = (value - minValue) / (maxValue - minValue) > 0.5

          return (
            <Paper
              key={index}
              p="xs"
              radius="sm"
              style={{
                backgroundColor: color,
                border: '1px solid var(--mantine-color-gray-3)',
                cursor: 'default',
              }}
            >
              <Text size="xs" fw={600} style={{ color: isDark ? 'white' : 'black' }}>
                {location}
              </Text>
              <Text size="lg" fw={700} style={{ color: isDark ? 'white' : 'black' }}>
                {formatValue(value)}
              </Text>
            </Paper>
          )
        })}
      </Box>
      {/* Legend */}
      <Group mt="md" gap="xs">
        <Text size="xs" c="dimmed">Low</Text>
        <Box
          style={{
            width: 100,
            height: 10,
            background: `linear-gradient(to right, ${getColor(minValue)}, ${getColor(maxValue)})`,
            borderRadius: 2,
          }}
        />
        <Text size="xs" c="dimmed">High</Text>
      </Group>
    </Box>
  )
}

// Pivot table visualization - Enhanced with row/column structure
function PivotVisualization({
  data,
  columns,
  settings = {},
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
}) {
  const showRowTotals = settings.showRowTotals ?? true
  const showColumnTotals = settings.showColumnTotals ?? true

  // For a basic pivot, we need at least 3 columns: row dimension, column dimension, and value
  // If we have fewer, fall back to table view
  if (columns.length < 3) {
    return (
      <TableVisualization
        result={{ columns, rows: data, row_count: data.length, duration_ms: 0 }}
        settings={settings as TableSettings}
      />
    )
  }

  const rowKey = columns[0]?.name
  const colKey = columns[1]?.name
  const valueKey = columns[2]?.name

  // Build pivot structure
  const { pivotData, rowValues, colValues, rowTotals, colTotals, grandTotal } = useMemo(() => {
    const rowSet = new Set<string>()
    const colSet = new Set<string>()
    const pivotMap: Record<string, Record<string, number>> = {}
    const rowTotals: Record<string, number> = {}
    const colTotals: Record<string, number> = {}
    let grandTotal = 0

    data.forEach(row => {
      const rowVal = String(row[rowKey] ?? '')
      const colVal = String(row[colKey] ?? '')
      const value = Number(row[valueKey]) || 0

      rowSet.add(rowVal)
      colSet.add(colVal)

      if (!pivotMap[rowVal]) pivotMap[rowVal] = {}
      pivotMap[rowVal][colVal] = (pivotMap[rowVal][colVal] || 0) + value

      rowTotals[rowVal] = (rowTotals[rowVal] || 0) + value
      colTotals[colVal] = (colTotals[colVal] || 0) + value
      grandTotal += value
    })

    return {
      pivotData: pivotMap,
      rowValues: Array.from(rowSet).sort(),
      colValues: Array.from(colSet).sort(),
      rowTotals,
      colTotals,
      grandTotal,
    }
  }, [data, rowKey, colKey, valueKey])

  return (
    <Box style={{ overflow: 'auto', maxHeight: 500 }}>
      <table style={{ borderCollapse: 'collapse', width: '100%', fontSize: 13 }}>
        <thead>
          <tr>
            <th style={{ border: '1px solid var(--mantine-color-gray-3)', padding: '8px 12px', background: 'var(--mantine-color-gray-0)', textAlign: 'left' }}>
              {columns[0]?.display_name || rowKey} / {columns[1]?.display_name || colKey}
            </th>
            {colValues.map(col => (
              <th key={col} style={{ border: '1px solid var(--mantine-color-gray-3)', padding: '8px 12px', background: 'var(--mantine-color-gray-0)', textAlign: 'right' }}>
                {col}
              </th>
            ))}
            {showRowTotals && (
              <th style={{ border: '1px solid var(--mantine-color-gray-3)', padding: '8px 12px', background: 'var(--mantine-color-gray-1)', textAlign: 'right', fontWeight: 600 }}>
                Total
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {rowValues.map(row => (
            <tr key={row}>
              <td style={{ border: '1px solid var(--mantine-color-gray-3)', padding: '8px 12px', fontWeight: 500 }}>
                {row}
              </td>
              {colValues.map(col => (
                <td key={col} style={{ border: '1px solid var(--mantine-color-gray-3)', padding: '8px 12px', textAlign: 'right' }}>
                  {(pivotData[row]?.[col] || 0).toLocaleString()}
                </td>
              ))}
              {showRowTotals && (
                <td style={{ border: '1px solid var(--mantine-color-gray-3)', padding: '8px 12px', textAlign: 'right', background: 'var(--mantine-color-gray-0)', fontWeight: 500 }}>
                  {(rowTotals[row] || 0).toLocaleString()}
                </td>
              )}
            </tr>
          ))}
          {showColumnTotals && (
            <tr>
              <td style={{ border: '1px solid var(--mantine-color-gray-3)', padding: '8px 12px', background: 'var(--mantine-color-gray-1)', fontWeight: 600 }}>
                Total
              </td>
              {colValues.map(col => (
                <td key={col} style={{ border: '1px solid var(--mantine-color-gray-3)', padding: '8px 12px', textAlign: 'right', background: 'var(--mantine-color-gray-0)', fontWeight: 500 }}>
                  {(colTotals[col] || 0).toLocaleString()}
                </td>
              ))}
              {showRowTotals && (
                <td style={{ border: '1px solid var(--mantine-color-gray-3)', padding: '8px 12px', textAlign: 'right', background: 'var(--mantine-color-gray-1)', fontWeight: 600 }}>
                  {grandTotal.toLocaleString()}
                </td>
              )}
            </tr>
          )}
        </tbody>
      </table>
    </Box>
  )
}

// Visualization type selector
export function VisualizationTypeIcon({ type }: { type: VisualizationType }) {
  const icons: Record<VisualizationType, string> = {
    table: '📋',
    number: '#',
    trend: '📈',
    progress: '📊',
    gauge: '🎯',
    line: '📉',
    area: '📊',
    bar: '📊',
    row: '📊',
    combo: '📊',
    waterfall: '📊',
    funnel: '🔽',
    pie: '🥧',
    donut: '🍩',
    scatter: '⚬',
    bubble: '⚬',
    'map-pin': '📍',
    'map-grid': '🗺️',
    'map-region': '🗺️',
    pivot: '📊',
    sankey: '🔀',
  }
  return <span>{icons[type] || '📊'}</span>
}

// Export visualization types for use elsewhere
export const visualizationTypes: { value: VisualizationType; label: string; category: string }[] = [
  // Table & Numbers
  { value: 'table', label: 'Table', category: 'Table' },
  { value: 'number', label: 'Number', category: 'Numbers' },
  { value: 'trend', label: 'Trend', category: 'Numbers' },
  { value: 'progress', label: 'Progress', category: 'Numbers' },
  { value: 'gauge', label: 'Gauge', category: 'Numbers' },
  // Line & Area
  { value: 'line', label: 'Line', category: 'Time Series' },
  { value: 'area', label: 'Area', category: 'Time Series' },
  // Bar charts
  { value: 'bar', label: 'Bar', category: 'Bar Charts' },
  { value: 'row', label: 'Row', category: 'Bar Charts' },
  { value: 'combo', label: 'Combo', category: 'Bar Charts' },
  { value: 'waterfall', label: 'Waterfall', category: 'Bar Charts' },
  // Parts of whole
  { value: 'pie', label: 'Pie', category: 'Parts of Whole' },
  { value: 'donut', label: 'Donut', category: 'Parts of Whole' },
  { value: 'funnel', label: 'Funnel', category: 'Parts of Whole' },
  // Scatter & Distribution
  { value: 'scatter', label: 'Scatter', category: 'Distribution' },
  { value: 'bubble', label: 'Bubble', category: 'Distribution' },
  // Maps
  { value: 'map-pin', label: 'Pin Map', category: 'Maps' },
  { value: 'map-grid', label: 'Grid Map', category: 'Maps' },
  { value: 'map-region', label: 'Region Map', category: 'Maps' },
  // Advanced
  { value: 'pivot', label: 'Pivot Table', category: 'Advanced' },
  { value: 'sankey', label: 'Sankey', category: 'Advanced' },
]
