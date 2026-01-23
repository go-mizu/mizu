import { useMemo } from 'react'
import { Box, Text, Paper, Skeleton, Group, Badge } from '@mantine/core'
import {
  LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, AreaChart, Area,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
  ScatterChart, Scatter, ComposedChart, FunnelChart, Funnel, LabelList
} from 'recharts'
import type { QueryResult, VisualizationSettings, VisualizationType } from '../../api/types'
import { chartColors } from '../../theme'
import { TableVisualization } from './table'
import type { TableSettings } from './table'

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
  const { type, settings } = visualization

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
      return <TrendVisualization data={data} columns={columns} />
    case 'progress':
      return <ProgressVisualization data={data} columns={columns} settings={settings} />
    case 'gauge':
      return <GaugeVisualization data={data} columns={columns} settings={settings} />
    case 'line':
      return <LineVisualization data={data} columns={columns} height={height} showLegend={showLegend} />
    case 'area':
      return <AreaVisualization data={data} columns={columns} settings={settings} height={height} showLegend={showLegend} />
    case 'bar':
      return <BarVisualization data={data} columns={columns} settings={settings} height={height} showLegend={showLegend} />
    case 'row':
      return <RowVisualization data={data} columns={columns} height={height} showLegend={showLegend} />
    case 'pie':
    case 'donut':
      return <PieVisualization data={data} columns={columns} height={height} showLegend={showLegend} donut={type === 'donut'} />
    case 'scatter':
      return <ScatterVisualization data={data} columns={columns} height={height} showLegend={showLegend} />
    case 'funnel':
      return <FunnelVisualization data={data} columns={columns} height={height} />
    case 'combo':
      return <ComboVisualization data={data} columns={columns} height={height} showLegend={showLegend} />
    case 'waterfall':
      return <WaterfallVisualization data={data} columns={columns} height={height} />
    case 'bubble':
      return <BubbleVisualization data={data} columns={columns} height={height} showLegend={showLegend} />
    case 'map-pin':
    case 'map-grid':
    case 'map-region':
      return <MapVisualization data={data} columns={columns} height={height} type={type} />
    case 'pivot':
      return <PivotVisualization data={data} columns={columns} settings={settings} />
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

// Number / Scalar visualization
function NumberVisualization({
  data,
  columns,
  settings,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
}) {
  const value = data[0]?.[columns[0]?.name]
  const prefix = settings?.prefix || ''
  const suffix = settings?.suffix || ''
  const decimals = settings?.decimals ?? 0

  const formattedValue = useMemo(() => {
    if (typeof value === 'number') {
      return value.toLocaleString(undefined, {
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      })
    }
    return String(value ?? '-')
  }, [value, decimals])

  return (
    <Paper p="xl" ta="center">
      <Text size="3rem" fw={700} c="brand.5">
        {prefix}{formattedValue}{suffix}
      </Text>
      <Text c="dimmed" mt="xs">{columns[0]?.display_name || columns[0]?.name}</Text>
    </Paper>
  )
}

// Trend visualization (number with trend indicator)
function TrendVisualization({
  data,
  columns,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
}) {
  const currentValue = data[0]?.[columns[0]?.name]
  const previousValue = data[1]?.[columns[0]?.name] || data[0]?.[columns[1]?.name]

  const change = useMemo(() => {
    if (typeof currentValue === 'number' && typeof previousValue === 'number' && previousValue !== 0) {
      return ((currentValue - previousValue) / previousValue) * 100
    }
    return null
  }, [currentValue, previousValue])

  return (
    <Paper p="xl" ta="center">
      <Text size="3rem" fw={700} c="brand.5">
        {typeof currentValue === 'number' ? currentValue.toLocaleString() : String(currentValue)}
      </Text>
      {change !== null && (
        <Badge
          size="lg"
          mt="sm"
          color={change >= 0 ? 'green' : 'red'}
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
  settings,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
}) {
  const value = data[0]?.[columns[0]?.name]
  const goal = settings?.goal || 100
  const percentage = typeof value === 'number' ? Math.min(100, (value / goal) * 100) : 0

  return (
    <Paper p="xl">
      <Group justify="space-between" mb="sm">
        <Text fw={500}>{columns[0]?.display_name || columns[0]?.name}</Text>
        <Text fw={700} c="brand.5">
          {typeof value === 'number' ? value.toLocaleString() : String(value)}
        </Text>
      </Group>
      <Box
        style={{
          height: 8,
          backgroundColor: 'var(--mantine-color-gray-2)',
          borderRadius: 4,
          overflow: 'hidden',
        }}
      >
        <Box
          style={{
            height: '100%',
            width: `${percentage}%`,
            backgroundColor: 'var(--mantine-color-brand-5)',
            borderRadius: 4,
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

// Gauge visualization
function GaugeVisualization({
  data,
  columns,
  settings,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
}) {
  const value = data[0]?.[columns[0]?.name]
  const min = settings?.min || 0
  const max = settings?.max || 100
  const percentage = typeof value === 'number' ? Math.min(100, Math.max(0, ((value - min) / (max - min)) * 100)) : 0

  return (
    <Paper p="xl" ta="center">
      <Box style={{ position: 'relative', width: 200, height: 100, margin: '0 auto' }}>
        <svg viewBox="0 0 200 100" width="200" height="100">
          {/* Background arc */}
          <path
            d="M 20 100 A 80 80 0 0 1 180 100"
            fill="none"
            stroke="var(--mantine-color-gray-2)"
            strokeWidth="16"
            strokeLinecap="round"
          />
          {/* Value arc */}
          <path
            d="M 20 100 A 80 80 0 0 1 180 100"
            fill="none"
            stroke="var(--mantine-color-brand-5)"
            strokeWidth="16"
            strokeLinecap="round"
            strokeDasharray={`${percentage * 2.51} 251`}
          />
        </svg>
        <Text
          size="xl"
          fw={700}
          style={{
            position: 'absolute',
            bottom: 0,
            left: '50%',
            transform: 'translateX(-50%)',
          }}
        >
          {typeof value === 'number' ? value.toLocaleString() : String(value)}
        </Text>
      </Box>
      <Text c="dimmed" mt="md">{columns[0]?.display_name || columns[0]?.name}</Text>
    </Paper>
  )
}

// Line chart visualization
function LineVisualization({
  data,
  columns,
  height,
  showLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  height: number
  showLegend: boolean
}) {
  const xKey = columns[0]?.name
  const yKeys = columns.slice(1).map(c => c.name)

  return (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={data} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
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
        {yKeys.map((key, i) => (
          <Line
            key={key}
            type="monotone"
            dataKey={key}
            stroke={chartColors[i % chartColors.length]}
            strokeWidth={2}
            dot={{ r: 3 }}
            activeDot={{ r: 5 }}
          />
        ))}
      </LineChart>
    </ResponsiveContainer>
  )
}

// Area chart visualization
function AreaVisualization({
  data,
  columns,
  settings,
  height,
  showLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
  showLegend: boolean
}) {
  const xKey = columns[0]?.name
  const yKeys = columns.slice(1).map(c => c.name)
  const stacked = settings?.stacked ?? false

  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
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
        {yKeys.map((key, i) => (
          <Area
            key={key}
            type="monotone"
            dataKey={key}
            stackId={stacked ? '1' : undefined}
            stroke={chartColors[i % chartColors.length]}
            fill={chartColors[i % chartColors.length]}
            fillOpacity={0.3}
          />
        ))}
      </AreaChart>
    </ResponsiveContainer>
  )
}

// Bar chart visualization
function BarVisualization({
  data,
  columns,
  settings,
  height,
  showLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
  showLegend: boolean
}) {
  const xKey = columns[0]?.name
  const yKeys = columns.slice(1).map(c => c.name)
  const stacked = settings?.stacked ?? false

  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={data} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
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
        {yKeys.map((key, i) => (
          <Bar
            key={key}
            dataKey={key}
            stackId={stacked ? '1' : undefined}
            fill={chartColors[i % chartColors.length]}
            radius={[4, 4, 0, 0]}
          />
        ))}
      </BarChart>
    </ResponsiveContainer>
  )
}

// Horizontal bar (row) chart visualization
function RowVisualization({
  data,
  columns,
  height,
  showLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  height: number
  showLegend: boolean
}) {
  const yKey = columns[0]?.name
  const xKeys = columns.slice(1).map(c => c.name)

  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={data} layout="vertical" margin={{ top: 5, right: 20, left: 80, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
        <XAxis type="number" tick={{ fontSize: 12 }} stroke="var(--mantine-color-gray-5)" />
        <YAxis dataKey={yKey} type="category" tick={{ fontSize: 12 }} stroke="var(--mantine-color-gray-5)" width={70} />
        <Tooltip
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
        />
        {showLegend && <Legend />}
        {xKeys.map((key, i) => (
          <Bar
            key={key}
            dataKey={key}
            fill={chartColors[i % chartColors.length]}
            radius={[0, 4, 4, 0]}
          />
        ))}
      </BarChart>
    </ResponsiveContainer>
  )
}

// Pie/Donut chart visualization
function PieVisualization({
  data,
  columns,
  height,
  showLegend,
  donut,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  height: number
  showLegend: boolean
  donut: boolean
}) {
  const nameKey = columns[0]?.name
  const valueKey = columns[1]?.name

  return (
    <ResponsiveContainer width="100%" height={height}>
      <PieChart>
        <Pie
          data={data}
          dataKey={valueKey}
          nameKey={nameKey}
          cx="50%"
          cy="50%"
          outerRadius={height / 3}
          innerRadius={donut ? height / 5 : 0}
          paddingAngle={2}
          label={({ name, percent }) => `${name}: ${((percent ?? 0) * 100).toFixed(0)}%`}
          labelLine={{ stroke: 'var(--mantine-color-gray-5)' }}
        >
          {data.map((_, index) => (
            <Cell key={`cell-${index}`} fill={chartColors[index % chartColors.length]} />
          ))}
        </Pie>
        <Tooltip
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
        />
        {showLegend && <Legend />}
      </PieChart>
    </ResponsiveContainer>
  )
}

// Scatter chart visualization
function ScatterVisualization({
  data,
  columns,
  height,
  showLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  height: number
  showLegend: boolean
}) {
  const xKey = columns[0]?.name
  const yKey = columns[1]?.name

  return (
    <ResponsiveContainer width="100%" height={height}>
      <ScatterChart margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
        <XAxis dataKey={xKey} name={xKey} tick={{ fontSize: 12 }} stroke="var(--mantine-color-gray-5)" />
        <YAxis dataKey={yKey} name={yKey} tick={{ fontSize: 12 }} stroke="var(--mantine-color-gray-5)" />
        <Tooltip
          cursor={{ strokeDasharray: '3 3' }}
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
        />
        {showLegend && <Legend />}
        <Scatter name={yKey} data={data} fill={chartColors[0]} />
      </ScatterChart>
    </ResponsiveContainer>
  )
}

// Funnel chart visualization
function FunnelVisualization({
  data,
  columns,
  height,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  height: number
}) {
  const nameKey = columns[0]?.name
  const valueKey = columns[1]?.name

  const funnelData = data.map((item, index) => ({
    name: item[nameKey],
    value: item[valueKey],
    fill: chartColors[index % chartColors.length],
  }))

  return (
    <ResponsiveContainer width="100%" height={height}>
      <FunnelChart>
        <Tooltip
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
        />
        <Funnel
          dataKey="value"
          data={funnelData}
          isAnimationActive
        >
          <LabelList position="right" fill="#000" stroke="none" dataKey="name" />
        </Funnel>
      </FunnelChart>
    </ResponsiveContainer>
  )
}

// Combo chart visualization (bar + line)
function ComboVisualization({
  data,
  columns,
  height,
  showLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  height: number
  showLegend: boolean
}) {
  const xKey = columns[0]?.name
  const barKeys = columns.slice(1, -1).map(c => c.name)
  const lineKey = columns[columns.length - 1]?.name

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
            fill={chartColors[i % chartColors.length]}
            radius={[4, 4, 0, 0]}
          />
        ))}
        <Line
          type="monotone"
          dataKey={lineKey}
          stroke={chartColors[(barKeys.length) % chartColors.length]}
          strokeWidth={2}
        />
      </ComposedChart>
    </ResponsiveContainer>
  )
}

// Waterfall chart visualization
function WaterfallVisualization({
  data,
  columns,
  height,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  height: number
}) {
  const xKey = columns[0]?.name
  const valueKey = columns[1]?.name

  // Transform data for waterfall: calculate running total and invisible base
  const waterfallData = useMemo(() => {
    let cumulative = 0
    return data.map((item, index) => {
      const value = Number(item[valueKey]) || 0
      const isTotal = index === data.length - 1 && item[xKey]?.toLowerCase().includes('total')
      const base = isTotal ? 0 : cumulative
      cumulative += value

      return {
        ...item,
        _base: isTotal ? 0 : Math.min(base, base + value),
        _value: Math.abs(value),
        _positive: value >= 0,
        _cumulative: cumulative,
      }
    })
  }, [data, xKey, valueKey])

  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={waterfallData} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
        <XAxis dataKey={xKey} tick={{ fontSize: 12 }} stroke="var(--mantine-color-gray-5)" />
        <YAxis tick={{ fontSize: 12 }} stroke="var(--mantine-color-gray-5)" />
        <Tooltip
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid var(--mantine-color-gray-3)',
            borderRadius: 6,
          }}
          formatter={(value: any, name: string | undefined) => {
            if (name === '_base') return null
            return [value, valueKey || name || '']
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
              fill={entry._positive ? 'var(--color-success)' : 'var(--color-error)'}
            />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  )
}

// Bubble chart visualization (scatter with variable size)
function BubbleVisualization({
  data,
  columns,
  height,
  showLegend,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  height: number
  showLegend: boolean
}) {
  const xKey = columns[0]?.name
  const yKey = columns[1]?.name
  const sizeKey = columns[2]?.name // Third column for bubble size

  // Calculate size scale
  const { minSize, maxSize } = useMemo(() => {
    if (!sizeKey) return { minSize: 1, maxSize: 1 }
    const values = data.map(d => Number(d[sizeKey]) || 0).filter(v => v > 0)
    return {
      minSize: Math.min(...values),
      maxSize: Math.max(...values),
    }
  }, [data, sizeKey])

  // Scale bubble size between 5 and 30 pixels
  const getSize = (value: number) => {
    if (!sizeKey || maxSize === minSize) return 10
    const normalized = (value - minSize) / (maxSize - minSize)
    return 5 + normalized * 25
  }

  return (
    <ResponsiveContainer width="100%" height={height}>
      <ScatterChart margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-gray-2)" />
        <XAxis dataKey={xKey} name={xKey} tick={{ fontSize: 12 }} stroke="var(--mantine-color-gray-5)" type="number" />
        <YAxis dataKey={yKey} name={yKey} tick={{ fontSize: 12 }} stroke="var(--mantine-color-gray-5)" type="number" />
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
            />
          ))}
          {sizeKey && data.map((item, index) => {
            const size = getSize(Number(item[sizeKey]) || 0)
            return (
              <Cell
                key={`size-${index}`}
                // @ts-ignore - recharts doesn't have proper types for this
                r={size}
              />
            )
          })}
        </Scatter>
      </ScatterChart>
    </ResponsiveContainer>
  )
}

// Map visualization - displays geographic data in a tile/grid format
function MapVisualization({
  data,
  columns,
  height,
  type,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  height: number
  type: 'map-pin' | 'map-grid' | 'map-region'
}) {
  const locationKey = columns[0]?.name
  const valueKey = columns[1]?.name

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
    if (maxValue === minValue) return chartColors[0]
    const normalized = (value - minValue) / (maxValue - minValue)
    // Color gradient from light blue to dark blue
    const hue = 210 // Blue
    const saturation = 70
    const lightness = 85 - (normalized * 50) // 85% to 35%
    return `hsl(${hue}, ${saturation}%, ${lightness}%)`
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
              <Text size="xs" fw={600} style={{ color: value > (maxValue - minValue) / 2 + minValue ? 'white' : 'black' }}>
                {location}
              </Text>
              <Text size="lg" fw={700} style={{ color: value > (maxValue - minValue) / 2 + minValue ? 'white' : 'black' }}>
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
            background: 'linear-gradient(to right, hsl(210, 70%, 85%), hsl(210, 70%, 35%))',
            borderRadius: 2,
          }}
        />
        <Text size="xs" c="dimmed">High</Text>
      </Group>
    </Box>
  )
}

// Pivot table visualization - Uses enhanced TableVisualization
function PivotVisualization({
  data,
  columns,
  settings,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
}) {
  // Simple pivot implementation - shows grouped data
  return (
    <TableVisualization
      result={{ columns, rows: data, row_count: data.length, duration_ms: 0 }}
      settings={settings as TableSettings}
    />
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
