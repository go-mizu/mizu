import { useMemo } from 'react'
import { Box, Text, Paper, Table, Skeleton, Group, Badge } from '@mantine/core'
import {
  LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, AreaChart, Area,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
  ScatterChart, Scatter, ComposedChart, FunnelChart, Funnel, LabelList
} from 'recharts'
import type { QueryResult, VisualizationSettings, VisualizationType } from '../../api/types'
import { chartColors } from '../../theme'

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
    case 'pivot':
      return <PivotVisualization data={data} columns={columns} settings={settings} />
    case 'table':
    default:
      return <TableVisualization data={data} columns={columns} settings={settings} />
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

// Pivot table visualization
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
  return <TableVisualization data={data} columns={columns} settings={settings} />
}

// Table visualization
function TableVisualization({
  data,
  columns,
  settings,
}: {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
}) {
  const maxRows = settings?.maxRows || 100

  const formatValue = (value: any, type: string) => {
    if (value === null || value === undefined) return '-'
    if (type === 'number' && typeof value === 'number') {
      return value.toLocaleString()
    }
    if (type === 'datetime' || type === 'date') {
      return new Date(value).toLocaleString()
    }
    return String(value)
  }

  return (
    <Box style={{ overflow: 'auto', maxHeight: settings?.maxHeight || 500 }}>
      <Table striped highlightOnHover withTableBorder stickyHeader>
        <Table.Thead>
          <Table.Tr>
            {columns.map((col) => (
              <Table.Th key={col.name} style={{ whiteSpace: 'nowrap' }}>
                {col.display_name || col.name}
              </Table.Th>
            ))}
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {data.slice(0, maxRows).map((row, i) => (
            <Table.Tr key={i}>
              {columns.map((col) => (
                <Table.Td key={col.name} style={{ whiteSpace: 'nowrap' }}>
                  {formatValue(row[col.name], col.type)}
                </Table.Td>
              ))}
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
      {data.length > maxRows && (
        <Text size="sm" c="dimmed" ta="center" py="sm">
          Showing {maxRows} of {data.length} rows
        </Text>
      )}
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
