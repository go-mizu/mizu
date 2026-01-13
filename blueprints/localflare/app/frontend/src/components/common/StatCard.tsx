import { Paper, Group, Text, ThemeIcon, Stack, Box, Tooltip, UnstyledButton } from '@mantine/core'
import { IconArrowUpRight, IconArrowDownRight, IconMinus } from '@tabler/icons-react'
import { type ReactNode } from 'react'
import { Area, AreaChart, ResponsiveContainer } from 'recharts'

interface StatCardProps {
  /** Icon displayed in the card */
  icon: ReactNode
  /** Label/title of the stat */
  label: string
  /** Main value to display */
  value: string | number
  /** Trend indicator showing percentage change */
  trend?: {
    value: number
    direction: 'up' | 'down' | 'neutral'
  }
  /** Color theme for the card */
  color?: 'default' | 'success' | 'warning' | 'error' | 'orange'
  /** Additional description text */
  description?: string
  /** Click handler for drill-down */
  onClick?: () => void
  /** Sparkline data for mini chart */
  sparklineData?: number[]
  /** Previous period value for comparison */
  previousValue?: number
  /** Help text shown on hover */
  helpText?: string
}

const colorMap = {
  default: 'gray',
  success: 'grass',
  warning: 'yellow',
  error: 'apple',
  orange: 'orange',
}

const sparklineColorMap = {
  default: '#6D7786',
  success: '#9BCA3E',
  warning: '#fab005',
  error: '#BD2527',
  orange: '#f6821f',
}

export function StatCard({
  icon,
  label,
  value,
  trend,
  color = 'default',
  description,
  onClick,
  sparklineData,
  previousValue,
  helpText,
}: StatCardProps) {
  const formattedValue = typeof value === 'number' ? value.toLocaleString() : value

  // Calculate trend from previous value if not provided
  const calculatedTrend = trend || (previousValue !== undefined && typeof value === 'number'
    ? {
        value: previousValue === 0 ? 100 : Math.round(((value - previousValue) / previousValue) * 100),
        direction: value > previousValue ? 'up' : value < previousValue ? 'down' : 'neutral',
      }
    : undefined) as typeof trend

  const TrendIcon = calculatedTrend?.direction === 'up'
    ? IconArrowUpRight
    : calculatedTrend?.direction === 'down'
    ? IconArrowDownRight
    : IconMinus

  const trendColor = calculatedTrend?.direction === 'up'
    ? 'var(--mantine-color-grass-5)'
    : calculatedTrend?.direction === 'down'
    ? 'var(--mantine-color-apple-5)'
    : 'var(--mantine-color-dimmed)'

  const content = (
    <Paper
      p="md"
      radius="md"
      withBorder
      style={{
        cursor: onClick ? 'pointer' : 'default',
        transition: 'all 150ms ease',
      }}
      styles={onClick ? {
        root: {
          '&:hover': {
            borderColor: 'var(--mantine-color-orange-5)',
            transform: 'translateY(-2px)',
          },
        },
      } : undefined}
    >
      <Group justify="space-between" align="flex-start" wrap="nowrap">
        <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
          <Text size="xs" c="dimmed" tt="uppercase" fw={600} truncate>
            {label}
          </Text>
          <Text size="xl" fw={700} truncate>
            {formattedValue}
          </Text>
          {description && (
            <Text size="xs" c="dimmed" truncate>
              {description}
            </Text>
          )}
          {calculatedTrend && (
            <Group gap={4} wrap="nowrap">
              <TrendIcon size={14} color={trendColor} />
              <Text
                size="xs"
                c={
                  calculatedTrend.direction === 'up'
                    ? 'grass'
                    : calculatedTrend.direction === 'down'
                    ? 'apple'
                    : 'dimmed'
                }
                fw={500}
              >
                {calculatedTrend.value > 0 ? '+' : ''}
                {calculatedTrend.value}%
              </Text>
              {previousValue !== undefined && (
                <Text size="xs" c="dimmed">
                  vs prev
                </Text>
              )}
            </Group>
          )}
        </Stack>

        <Stack gap="xs" align="flex-end">
          <ThemeIcon variant="light" size="lg" radius="md" color={colorMap[color]}>
            {icon}
          </ThemeIcon>

          {/* Sparkline Mini Chart */}
          {sparklineData && sparklineData.length > 1 && (
            <Box w={60} h={24}>
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={sparklineData.map((v, i) => ({ value: v, index: i }))}>
                  <defs>
                    <linearGradient id={`sparkline-${label}`} x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={sparklineColorMap[color]} stopOpacity={0.3} />
                      <stop offset="100%" stopColor={sparklineColorMap[color]} stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <Area
                    type="monotone"
                    dataKey="value"
                    stroke={sparklineColorMap[color]}
                    strokeWidth={1.5}
                    fill={`url(#sparkline-${label})`}
                    isAnimationActive={false}
                  />
                </AreaChart>
              </ResponsiveContainer>
            </Box>
          )}
        </Stack>
      </Group>
    </Paper>
  )

  // Wrap with button if clickable
  if (onClick) {
    return (
      <Tooltip label={helpText || 'Click to view details'} disabled={!helpText && !onClick}>
        <UnstyledButton onClick={onClick} w="100%">
          {content}
        </UnstyledButton>
      </Tooltip>
    )
  }

  if (helpText) {
    return (
      <Tooltip label={helpText}>
        {content}
      </Tooltip>
    )
  }

  return content
}

/** Compact stat card variant for dense layouts */
interface CompactStatCardProps {
  label: string
  value: string | number
  icon?: ReactNode
  color?: StatCardProps['color']
  trend?: StatCardProps['trend']
}

export function CompactStatCard({ label, value, icon, color = 'default', trend }: CompactStatCardProps) {
  const formattedValue = typeof value === 'number' ? value.toLocaleString() : value

  return (
    <Paper p="sm" radius="md" withBorder>
      <Group gap="xs" wrap="nowrap">
        {icon && (
          <ThemeIcon variant="light" size="sm" radius="sm" color={colorMap[color]}>
            {icon}
          </ThemeIcon>
        )}
        <Box style={{ flex: 1, minWidth: 0 }}>
          <Text size="xs" c="dimmed" truncate>
            {label}
          </Text>
          <Group gap={4} wrap="nowrap">
            <Text size="sm" fw={600} truncate>
              {formattedValue}
            </Text>
            {trend && (
              <Text
                size="xs"
                c={trend.direction === 'up' ? 'grass' : trend.direction === 'down' ? 'apple' : 'dimmed'}
              >
                {trend.direction === 'up' ? '+' : ''}{trend.value}%
              </Text>
            )}
          </Group>
        </Box>
      </Group>
    </Paper>
  )
}
