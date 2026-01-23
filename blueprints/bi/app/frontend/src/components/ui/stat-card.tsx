import { forwardRef } from 'react'
import { Paper, Group, Text, ThemeIcon, rem, UnstyledButton, Skeleton } from '@mantine/core'
import { IconTrendingUp, IconTrendingDown, IconMinus } from '@tabler/icons-react'
import { formatNumber, formatPercent } from '../../lib/utils'

// =============================================================================
// STAT CARD - Display a metric with optional trend
// =============================================================================

export interface StatCardProps {
  /** Metric label */
  label: string
  /** Numeric value */
  value: number | string
  /** Icon to display */
  icon?: React.ReactNode
  /** Icon color (hex or CSS variable) */
  iconColor?: string
  /** Format type for the value */
  format?: 'number' | 'currency' | 'percent' | 'custom'
  /** Currency code if format is currency */
  currency?: string
  /** Trend percentage (positive or negative) */
  trend?: number
  /** Trend label (e.g., "vs last month") */
  trendLabel?: string
  /** Click handler */
  onClick?: () => void
  /** Custom value formatter */
  formatValue?: (value: number | string) => string
  /** Loading state */
  loading?: boolean
}

export const StatCard = forwardRef<HTMLDivElement, StatCardProps>(
  function StatCard(
    {
      label,
      value,
      icon,
      iconColor = 'var(--color-primary)',
      format = 'number',
      currency = 'USD',
      trend,
      trendLabel,
      onClick,
      formatValue,
      loading = false,
    },
    ref
  ) {
    // Format the value
    const displayValue = (() => {
      if (formatValue) return formatValue(value)
      if (typeof value === 'string') return value

      switch (format) {
        case 'number':
          return formatNumber(value)
        case 'currency':
          return new Intl.NumberFormat(undefined, {
            style: 'currency',
            currency,
            minimumFractionDigits: 0,
            maximumFractionDigits: 0,
          }).format(value)
        case 'percent':
          return formatPercent(value / 100, 1)
        default:
          return String(value)
      }
    })()

    // Determine trend direction and color
    const trendDirection = trend ? (trend > 0 ? 'up' : trend < 0 ? 'down' : 'neutral') : null
    const trendColor = trendDirection === 'up'
      ? 'var(--color-success)'
      : trendDirection === 'down'
        ? 'var(--color-error)'
        : 'var(--color-foreground-muted)'
    const TrendIcon = trendDirection === 'up'
      ? IconTrendingUp
      : trendDirection === 'down'
        ? IconTrendingDown
        : IconMinus

    const content = (
      <Paper
        ref={ref}
        withBorder
        p="md"
        radius="md"
        style={{
          cursor: onClick ? 'pointer' : 'default',
          transition: 'all var(--transition-fast)',
          height: '100%',
        }}
        onMouseEnter={(e) => {
          if (onClick) {
            e.currentTarget.style.boxShadow = 'var(--shadow-card-hover)'
          }
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.boxShadow = 'none'
        }}
      >
        <Group gap="md" wrap="nowrap">
          {icon && (
            <ThemeIcon
              size={44}
              radius="md"
              variant="light"
              style={{
                backgroundColor: `${iconColor}15`,
                color: iconColor,
              }}
            >
              {icon}
            </ThemeIcon>
          )}

          <div style={{ flex: 1, minWidth: 0 }}>
            {loading ? (
              <>
                <Skeleton height={28} width={80} mb={4} />
                <Skeleton height={14} width={60} />
              </>
            ) : (
              <>
                <Text
                  size="xl"
                  fw={700}
                  style={{
                    color: 'var(--color-foreground)',
                    lineHeight: 1.2,
                    fontSize: rem(24),
                  }}
                >
                  {displayValue}
                </Text>
                <Text
                  size="sm"
                  style={{ color: 'var(--color-foreground-muted)' }}
                  mt={2}
                >
                  {label}
                </Text>
              </>
            )}
          </div>

          {trend !== undefined && !loading && (
            <div style={{ textAlign: 'right' }}>
              <Group gap={4} justify="flex-end">
                <TrendIcon size={14} color={trendColor} strokeWidth={2} />
                <Text
                  size="sm"
                  fw={600}
                  style={{ color: trendColor }}
                >
                  {trend > 0 ? '+' : ''}{formatPercent(Math.abs(trend) / 100, 1)}
                </Text>
              </Group>
              {trendLabel && (
                <Text
                  size="xs"
                  style={{ color: 'var(--color-foreground-subtle)' }}
                  mt={2}
                >
                  {trendLabel}
                </Text>
              )}
            </div>
          )}
        </Group>
      </Paper>
    )

    if (onClick) {
      return (
        <UnstyledButton onClick={onClick} style={{ display: 'block', width: '100%' }}>
          {content}
        </UnstyledButton>
      )
    }

    return content
  }
)

// =============================================================================
// STAT CARD SKELETON - Loading placeholder
// =============================================================================

export function StatCardSkeleton() {
  return (
    <Paper withBorder p="md" radius="md">
      <Group gap="md" wrap="nowrap">
        <Skeleton height={44} width={44} radius="md" />
        <div style={{ flex: 1 }}>
          <Skeleton height={28} width={80} mb={4} />
          <Skeleton height={14} width={60} />
        </div>
      </Group>
    </Paper>
  )
}

// =============================================================================
// MINI STAT - Compact inline stat display
// =============================================================================

export interface MiniStatProps {
  label: string
  value: string | number
  color?: string
}

export function MiniStat({ label, value, color }: MiniStatProps) {
  return (
    <Group gap="xs">
      {color && (
        <div
          style={{
            width: 8,
            height: 8,
            borderRadius: '50%',
            backgroundColor: color,
          }}
        />
      )}
      <Text size="sm" c="dimmed">{label}:</Text>
      <Text size="sm" fw={600}>{value}</Text>
    </Group>
  )
}
