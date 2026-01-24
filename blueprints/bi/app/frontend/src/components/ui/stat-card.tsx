import { forwardRef, useState } from 'react'
import { Paper, Group, Text, Box, rem, UnstyledButton, Skeleton } from '@mantine/core'
import { IconTrendingUp, IconTrendingDown, IconMinus } from '@tabler/icons-react'
import { formatNumber, formatPercent } from '../../lib/utils'

// =============================================================================
// STAT CARD - Modern metric display (shadcn-inspired)
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
  /** Size variant */
  size?: 'sm' | 'md' | 'lg'
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
      size = 'md',
    },
    ref
  ) {
    const [isHovered, setIsHovered] = useState(false)

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

    const padding = { sm: rem(14), md: rem(18), lg: rem(24) }
    const valueFontSize = { sm: rem(22), md: rem(28), lg: rem(36) }
    const iconSize = { sm: 40, md: 48, lg: 56 }

    const content = (
      <Paper
        ref={ref}
        p={padding[size]}
        radius="lg"
        style={{
          cursor: onClick ? 'pointer' : 'default',
          transition: 'all 150ms cubic-bezier(0.4, 0, 0.2, 1)',
          height: '100%',
          border: '1px solid var(--color-border)',
          backgroundColor: 'var(--color-background)',
          boxShadow: isHovered && onClick ? 'var(--shadow-md)' : 'var(--shadow-xs)',
          transform: isHovered && onClick ? 'translateY(-2px)' : 'none',
        }}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
      >
        <Group gap="lg" wrap="nowrap" align="flex-start">
          {icon && (
            <Box
              style={{
                width: iconSize[size],
                height: iconSize[size],
                borderRadius: 'var(--radius-lg)',
                backgroundColor: `${iconColor}10`,
                border: `1px solid ${iconColor}18`,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flexShrink: 0,
                transition: 'transform 150ms ease',
                transform: isHovered ? 'scale(1.05)' : 'none',
              }}
            >
              <span style={{ color: iconColor, display: 'flex' }}>
                {icon}
              </span>
            </Box>
          )}

          <div style={{ flex: 1, minWidth: 0 }}>
            {loading ? (
              <>
                <Skeleton height={32} width={100} mb={6} radius="md" />
                <Skeleton height={16} width={70} radius="md" />
              </>
            ) : (
              <>
                <Text
                  fw={700}
                  style={{
                    color: 'var(--color-foreground)',
                    lineHeight: 1.1,
                    fontSize: valueFontSize[size],
                    letterSpacing: '-0.02em',
                  }}
                >
                  {displayValue}
                </Text>
                <Text
                  size="sm"
                  style={{ color: 'var(--color-foreground-muted)' }}
                  mt={rem(6)}
                >
                  {label}
                </Text>
              </>
            )}
          </div>

          {trend !== undefined && !loading && (
            <Box style={{ textAlign: 'right', flexShrink: 0 }}>
              <Group gap={rem(4)} justify="flex-end">
                <Box
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: rem(4),
                    backgroundColor: trendDirection === 'up' ? 'var(--color-success-light)' : trendDirection === 'down' ? 'var(--color-error-light)' : 'var(--color-background-subtle)',
                    padding: `${rem(4)} ${rem(8)}`,
                    borderRadius: 'var(--radius-full)',
                  }}
                >
                  <TrendIcon size={14} color={trendColor} strokeWidth={2.5} />
                  <Text
                    size="sm"
                    fw={600}
                    style={{ color: trendColor }}
                  >
                    {trend > 0 ? '+' : ''}{formatPercent(Math.abs(trend) / 100, 1)}
                  </Text>
                </Box>
              </Group>
              {trendLabel && (
                <Text
                  size="xs"
                  style={{ color: 'var(--color-foreground-subtle)' }}
                  mt={rem(6)}
                >
                  {trendLabel}
                </Text>
              )}
            </Box>
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
// STAT CARD SKELETON - Modern loading placeholder
// =============================================================================

export function StatCardSkeleton({ size = 'md' }: { size?: 'sm' | 'md' | 'lg' }) {
  const padding = { sm: rem(14), md: rem(18), lg: rem(24) }
  const iconSize = { sm: 40, md: 48, lg: 56 }

  return (
    <Paper
      p={padding[size]}
      radius="lg"
      style={{
        border: '1px solid var(--color-border)',
        backgroundColor: 'var(--color-background)',
      }}
    >
      <Group gap="lg" wrap="nowrap">
        <Skeleton height={iconSize[size]} width={iconSize[size]} radius="lg" />
        <div style={{ flex: 1 }}>
          <Skeleton height={32} width={100} mb={6} radius="md" />
          <Skeleton height={16} width={70} radius="md" />
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
