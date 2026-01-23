import { forwardRef } from 'react'
import {
  Paper, Group, Text, Badge, ActionIcon, Menu, Tooltip, ThemeIcon, rem
} from '@mantine/core'
import {
  IconDots, IconStar, IconStarFilled, IconBookmark, IconBookmarkFilled,
  IconLayoutDashboard, IconChartBar, IconFolder, IconDatabase
} from '@tabler/icons-react'
import { chartPalette } from '../../lib/tokens'

// =============================================================================
// DATA CARD - Unified card for questions, dashboards, collections
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
    const Icon = typeIcons[type]
    const accentColor = colorIndex !== undefined
      ? chartPalette[colorIndex % chartPalette.length]
      : typeColors[type]

    return (
      <Paper
        ref={ref}
        withBorder
        p={compact ? 'sm' : 'md'}
        radius="md"
        style={{
          cursor: onClick ? 'pointer' : 'default',
          transition: 'all var(--transition-fast)',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
        }}
        onClick={onClick}
        onMouseEnter={(e) => {
          if (onClick) {
            e.currentTarget.style.boxShadow = 'var(--shadow-card-hover)'
            e.currentTarget.style.borderColor = 'var(--color-border-strong)'
          }
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.boxShadow = 'none'
          e.currentTarget.style.borderColor = ''
        }}
      >
        <Group justify="space-between" mb={compact ? 'xs' : 'sm'} wrap="nowrap">
          <ThemeIcon
            size={compact ? 32 : 40}
            radius="md"
            variant="light"
            style={{
              backgroundColor: `${accentColor}15`,
              color: accentColor,
            }}
          >
            <Icon size={compact ? 16 : 20} strokeWidth={1.75} />
          </ThemeIcon>

          <Group gap={4} onClick={(e) => e.stopPropagation()}>
            {onTogglePin && (
              <Tooltip label={pinned ? 'Unpin' : 'Pin to home'} position="top">
                <ActionIcon
                  variant="subtle"
                  color={pinned ? 'yellow' : 'gray'}
                  size="sm"
                  onClick={onTogglePin}
                >
                  {pinned ? <IconStarFilled size={14} /> : <IconStar size={14} strokeWidth={1.75} />}
                </ActionIcon>
              </Tooltip>
            )}

            {(onToggleBookmark || menuItems) && (
              <Menu position="bottom-end" withinPortal>
                <Menu.Target>
                  <ActionIcon variant="subtle" color="gray" size="sm">
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
          }}
        >
          {name}
        </Text>

        {!compact && description && (
          <Text
            size="sm"
            c="dimmed"
            lineClamp={1}
            mt={4}
            style={{ color: 'var(--color-foreground-muted)' }}
          >
            {description}
          </Text>
        )}

        {badge && (
          <Badge
            size="xs"
            variant="light"
            color="gray"
            radius="sm"
            mt={compact ? 'xs' : 'sm'}
          >
            {badge}
          </Badge>
        )}
      </Paper>
    )
  }
)

// =============================================================================
// DATA CARD SKELETON - Loading placeholder
// =============================================================================

export function DataCardSkeleton({ compact = false }: { compact?: boolean }) {
  return (
    <Paper
      withBorder
      p={compact ? 'sm' : 'md'}
      radius="md"
      style={{ height: '100%' }}
    >
      <Group justify="space-between" mb={compact ? 'xs' : 'sm'}>
        <div
          style={{
            width: compact ? 32 : 40,
            height: compact ? 32 : 40,
            borderRadius: 'var(--radius-md)',
            backgroundColor: 'var(--color-background-subtle)',
            animation: 'pulse 2s infinite',
          }}
        />
        <div
          style={{
            width: 24,
            height: 24,
            borderRadius: 'var(--radius)',
            backgroundColor: 'var(--color-background-subtle)',
            animation: 'pulse 2s infinite',
          }}
        />
      </Group>
      <div
        style={{
          height: compact ? 16 : 20,
          width: '80%',
          borderRadius: 'var(--radius-sm)',
          backgroundColor: 'var(--color-background-subtle)',
          animation: 'pulse 2s infinite',
        }}
      />
      {!compact && (
        <div
          style={{
            height: 14,
            width: '60%',
            borderRadius: 'var(--radius-sm)',
            backgroundColor: 'var(--color-background-subtle)',
            marginTop: rem(8),
            animation: 'pulse 2s infinite',
          }}
        />
      )}
    </Paper>
  )
}
