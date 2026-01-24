import { forwardRef, useState } from 'react'
import {
  Paper, Group, Text, Badge, ActionIcon, Menu, Tooltip, Box, rem
} from '@mantine/core'
import {
  IconDots, IconStar, IconStarFilled, IconBookmark, IconBookmarkFilled,
  IconLayoutDashboard, IconChartBar, IconFolder, IconDatabase
} from '@tabler/icons-react'
import { chartPalette } from '../../lib/tokens'

// =============================================================================
// DATA CARD - Modern card component (shadcn-inspired)
// =============================================================================

export type DataCardType = 'question' | 'dashboard' | 'collection' | 'database'

export interface DataCardProps {
  /** Unique identifier */
  id: string
  /** Card type for icon and styling */
  type: DataCardType
  /** Display name */
  name: string
  /** Optional description */
  description?: string
  /** Optional badge text (e.g., "5 cards", "table") */
  badge?: string
  /** Color index for icon background */
  colorIndex?: number
  /** Is the item pinned */
  pinned?: boolean
  /** Is the item bookmarked */
  bookmarked?: boolean
  /** Click handler */
  onClick?: () => void
  /** Pin toggle handler */
  onTogglePin?: () => void
  /** Bookmark toggle handler */
  onToggleBookmark?: () => void
  /** Additional menu items */
  menuItems?: React.ReactNode
  /** Compact mode for smaller cards */
  compact?: boolean
}

const typeIcons = {
  question: IconChartBar,
  dashboard: IconLayoutDashboard,
  collection: IconFolder,
  database: IconDatabase,
}

const typeColors = {
  question: 'var(--color-primary)',
  dashboard: 'var(--color-success)',
  collection: 'var(--color-info)',
  database: 'var(--color-warning)',
}

export const DataCard = forwardRef<HTMLDivElement, DataCardProps>(
  function DataCard(
    {
      id: _id,
      type,
      name,
      description,
      badge,
      colorIndex = 0,
      pinned,
      bookmarked,
      onClick,
      onTogglePin,
      onToggleBookmark,
      menuItems,
      compact = false,
    },
    ref
  ) {
    const [isHovered, setIsHovered] = useState(false)
    const Icon = typeIcons[type]
    const accentColor = colorIndex !== undefined
      ? chartPalette[colorIndex % chartPalette.length]
      : typeColors[type]

    return (
      <Paper
        ref={ref}
        p={compact ? rem(14) : rem(18)}
        radius="lg"
        style={{
          cursor: onClick ? 'pointer' : 'default',
          transition: 'all 150ms cubic-bezier(0.4, 0, 0.2, 1)',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          border: '1px solid var(--color-border)',
          backgroundColor: 'var(--color-background)',
          boxShadow: isHovered ? 'var(--shadow-md)' : 'var(--shadow-xs)',
          transform: isHovered && onClick ? 'translateY(-2px)' : 'none',
        }}
        onClick={onClick}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
      >
        <Group justify="space-between" mb={compact ? rem(10) : rem(14)} wrap="nowrap">
          {/* Modern icon container with gradient hint */}
          <Box
            style={{
              width: compact ? 36 : 44,
              height: compact ? 36 : 44,
              borderRadius: 'var(--radius-lg)',
              backgroundColor: `${accentColor}12`,
              border: `1px solid ${accentColor}20`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'all 150ms ease',
              transform: isHovered ? 'scale(1.05)' : 'none',
            }}
          >
            <Icon size={compact ? 18 : 22} color={accentColor} strokeWidth={1.75} />
          </Box>

          <Group gap={4} onClick={(e) => e.stopPropagation()} style={{ opacity: isHovered ? 1 : 0.6, transition: 'opacity 150ms ease' }}>
            {onTogglePin && (
              <Tooltip label={pinned ? 'Unpin' : 'Pin to home'} position="top" withArrow>
                <ActionIcon
                  variant="subtle"
                  color={pinned ? 'yellow' : 'gray'}
                  size="sm"
                  radius="md"
                  onClick={onTogglePin}
                >
                  {pinned ? <IconStarFilled size={14} /> : <IconStar size={14} strokeWidth={1.75} />}
                </ActionIcon>
              </Tooltip>
            )}

            {(onToggleBookmark || menuItems) && (
              <Menu position="bottom-end" withinPortal shadow="md" radius="md">
                <Menu.Target>
                  <ActionIcon variant="subtle" color="gray" size="sm" radius="md">
                    <IconDots size={14} />
                  </ActionIcon>
                </Menu.Target>
                <Menu.Dropdown>
                  {onToggleBookmark && (
                    <Menu.Item
                      leftSection={
                        bookmarked
                          ? <IconBookmarkFilled size={14} />
                          : <IconBookmark size={14} strokeWidth={1.75} />
                      }
                      onClick={onToggleBookmark}
                    >
                      {bookmarked ? 'Remove bookmark' : 'Bookmark'}
                    </Menu.Item>
                  )}
                  {onTogglePin && (
                    <Menu.Item
                      leftSection={
                        pinned
                          ? <IconStarFilled size={14} />
                          : <IconStar size={14} strokeWidth={1.75} />
                      }
                      onClick={onTogglePin}
                    >
                      {pinned ? 'Unpin from home' : 'Pin to home'}
                    </Menu.Item>
                  )}
                  {menuItems && (
                    <>
                      <Menu.Divider />
                      {menuItems}
                    </>
                  )}
                </Menu.Dropdown>
              </Menu>
            )}
          </Group>
        </Group>

        <Text
          fw={600}
          lineClamp={compact ? 1 : 2}
          size={compact ? 'sm' : 'md'}
          style={{
            color: 'var(--color-foreground)',
            flex: 1,
            lineHeight: 1.4,
            letterSpacing: '-0.01em',
          }}
        >
          {name}
        </Text>

        {!compact && description && (
          <Text
            size="sm"
            lineClamp={2}
            mt={rem(6)}
            style={{
              color: 'var(--color-foreground-muted)',
              lineHeight: 1.5,
            }}
          >
            {description}
          </Text>
        )}

        {badge && (
          <Badge
            size="sm"
            variant="light"
            color="gray"
            radius="md"
            mt={compact ? rem(10) : rem(14)}
            style={{
              fontWeight: 500,
              textTransform: 'none',
              alignSelf: 'flex-start',
            }}
          >
            {badge}
          </Badge>
        )}
      </Paper>
    )
  }
)

// =============================================================================
// DATA CARD SKELETON - Modern loading placeholder
// =============================================================================

export function DataCardSkeleton({ compact = false }: { compact?: boolean }) {
  return (
    <Paper
      p={compact ? rem(14) : rem(18)}
      radius="lg"
      style={{
        height: '100%',
        border: '1px solid var(--color-border)',
        backgroundColor: 'var(--color-background)',
      }}
    >
      <Group justify="space-between" mb={compact ? rem(10) : rem(14)}>
        <div
          style={{
            width: compact ? 36 : 44,
            height: compact ? 36 : 44,
            borderRadius: 'var(--radius-lg)',
            backgroundColor: 'var(--color-background-subtle)',
            animation: 'pulse 1.5s ease-in-out infinite',
          }}
        />
        <div
          style={{
            width: 24,
            height: 24,
            borderRadius: 'var(--radius-md)',
            backgroundColor: 'var(--color-background-subtle)',
            animation: 'pulse 1.5s ease-in-out infinite',
          }}
        />
      </Group>
      <div
        style={{
          height: compact ? 18 : 22,
          width: '75%',
          borderRadius: 'var(--radius)',
          backgroundColor: 'var(--color-background-subtle)',
          animation: 'pulse 1.5s ease-in-out infinite',
        }}
      />
      {!compact && (
        <div
          style={{
            height: 16,
            width: '55%',
            borderRadius: 'var(--radius)',
            backgroundColor: 'var(--color-background-subtle)',
            marginTop: rem(10),
            animation: 'pulse 1.5s ease-in-out infinite',
          }}
        />
      )}
    </Paper>
  )
}
