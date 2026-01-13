import { Paper, Group, Text, SegmentedControl, Box, Stack } from '@mantine/core'
import {
  AreaChart as RechartsAreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'

interface AreaChartProps {
  data: Array<{ timestamp: string; value: number }>
  xKey?: string
  yKey?: string
  color?: string
  gradient?: boolean
  title?: string
  timeRange?: '1h' | '24h' | '7d' | '30d'
  onTimeRangeChange?: (range: '1h' | '24h' | '7d' | '30d') => void
  height?: number
  formatValue?: (value: number) => string
  formatLabel?: (timestamp: string) => string
}

const timeRangeOptions = [
  { value: '1h', label: '1h' },
  { value: '24h', label: '24h' },
  { value: '7d', label: '7d' },
  { value: '30d', label: '30d' },
]

export function AreaChart({
  data,
  xKey = 'timestamp',
  yKey = 'value',
  color = '#f6821f',
  gradient = true,
  title,
  timeRange = '24h',
  onTimeRangeChange,
  height = 200,
  formatValue = (v) => v.toLocaleString(),
  formatLabel = (t) => {
    const date = new Date(t)
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  },
}: AreaChartProps) {
  return (
    <Paper p="md" radius="md" withBorder>
      <Stack gap="md">
        {(title || onTimeRangeChange) && (
          <Group justify="space-between">
            {title && (
              <Text size="sm" fw={600}>
                {title}
              </Text>
            )}
            {onTimeRangeChange && (
              <SegmentedControl
                size="xs"
                value={timeRange}
                onChange={(v) => onTimeRangeChange(v as '1h' | '24h' | '7d' | '30d')}
                data={timeRangeOptions}
              />
            )}
          </Group>
        )}

        <Box h={height}>
          <ResponsiveContainer width="100%" height="100%">
            <RechartsAreaChart data={data} margin={{ top: 5, right: 5, left: 0, bottom: 5 }}>
              {gradient && (
                <defs>
                  <linearGradient id={`gradient-${color.replace('#', '')}`} x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor={color} stopOpacity={0.3} />
                    <stop offset="95%" stopColor={color} stopOpacity={0} />
                  </linearGradient>
                </defs>
              )}
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.1)" vertical={false} />
              <XAxis
                dataKey={xKey}
                tickFormatter={formatLabel}
                tick={{ fontSize: 11, fill: 'rgba(255,255,255,0.5)' }}
                axisLine={{ stroke: 'rgba(255,255,255,0.1)' }}
                tickLine={false}
              />
              <YAxis
                tickFormatter={formatValue}
                tick={{ fontSize: 11, fill: 'rgba(255,255,255,0.5)' }}
                axisLine={false}
                tickLine={false}
                width={50}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'var(--mantine-color-dark-6)',
                  border: '1px solid var(--mantine-color-dark-4)',
                  borderRadius: 6,
                  fontSize: 12,
                }}
                labelFormatter={formatLabel}
                formatter={(value) => [formatValue(value as number), '']}
              />
              <Area
                type="monotone"
                dataKey={yKey}
                stroke={color}
                strokeWidth={2}
                fill={gradient ? `url(#gradient-${color.replace('#', '')})` : color}
                fillOpacity={gradient ? 1 : 0.1}
              />
            </RechartsAreaChart>
          </ResponsiveContainer>
        </Box>
      </Stack>
    </Paper>
  )
}
