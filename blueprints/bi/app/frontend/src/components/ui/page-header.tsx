import { Box, Group, Title, Text, Breadcrumbs, Anchor, rem } from '@mantine/core'
import { IconChevronRight } from '@tabler/icons-react'

// =============================================================================
// PAGE HEADER - Modern, clean page title (shadcn-inspired)
// =============================================================================

export interface PageHeaderProps {
  /** Page title */
  title: string
  /** Optional subtitle/description */
  subtitle?: string
  /** Optional breadcrumb items */
  breadcrumbs?: Array<{ label: string; href?: string; onClick?: () => void }>
  /** Actions to display on the right */
  actions?: React.ReactNode
  /** Content below the header (tabs, filters, etc.) */
  children?: React.ReactNode
  /** Remove bottom border */
  borderless?: boolean
  /** Compact variant with less padding */
  compact?: boolean
}

export function PageHeader({
  title,
  subtitle,
  breadcrumbs,
  actions,
  children,
  borderless = false,
  compact = false,
}: PageHeaderProps) {
  return (
    <Box
      mb={compact ? 'md' : 'xl'}
      pb={borderless ? 0 : compact ? 'sm' : 'lg'}
      style={{
        borderBottom: borderless ? 'none' : '1px solid var(--color-border)',
      }}
    >
      {breadcrumbs && breadcrumbs.length > 0 && (
        <Breadcrumbs
          mb="sm"
          separator={<IconChevronRight size={12} color="var(--color-foreground-subtle)" strokeWidth={2} />}
          styles={{
            root: { flexWrap: 'wrap' },
            separator: { marginLeft: rem(6), marginRight: rem(6) },
          }}
        >
          {breadcrumbs.map((crumb, index) => (
            <Anchor
              key={index}
              onClick={crumb.onClick}
              href={crumb.href}
              size="sm"
              style={{
                color: index === breadcrumbs.length - 1 ? 'var(--color-foreground)' : 'var(--color-foreground-muted)',
                fontWeight: index === breadcrumbs.length - 1 ? 500 : 400,
                cursor: 'pointer',
                textDecoration: 'none',
              }}
            >
              {crumb.label}
            </Anchor>
          ))}
        </Breadcrumbs>
      )}

      <Group justify="space-between" align="flex-start" wrap="nowrap" gap="xl">
        <Box style={{ flex: 1, minWidth: 0 }}>
          <Title
            order={2}
            style={{
              fontSize: compact ? rem(20) : rem(28),
              fontWeight: 600,
              color: 'var(--color-foreground)',
              lineHeight: 1.2,
              letterSpacing: '-0.02em',
            }}
          >
            {title}
          </Title>
          {subtitle && (
            <Text
              size="sm"
              mt={6}
              style={{
                color: 'var(--color-foreground-muted)',
                lineHeight: 1.5,
              }}
            >
              {subtitle}
            </Text>
          )}
        </Box>

        {actions && (
          <Group gap="sm" wrap="nowrap">
            {actions}
          </Group>
        )}
      </Group>

      {children}
    </Box>
  )
}

// =============================================================================
// SECTION HEADER - Clean section dividers (shadcn-inspired)
// =============================================================================

export interface SectionHeaderProps {
  /** Section title */
  title: string
  /** Optional icon */
  icon?: React.ReactNode
  /** Optional count badge */
  count?: number
  /** Actions to display on the right */
  actions?: React.ReactNode
  /** Custom styles */
  style?: React.CSSProperties
  /** Size variant */
  size?: 'sm' | 'md' | 'lg'
}

export function SectionHeader({
  title,
  icon,
  count,
  actions,
  style,
  size = 'md',
}: SectionHeaderProps) {
  const fontSizes = {
    sm: rem(11),
    md: rem(12),
    lg: rem(13),
  }

  return (
    <Group
      justify="space-between"
      mb={size === 'sm' ? 'sm' : 'md'}
      style={style}
    >
      <Group gap={rem(8)} align="center">
        {icon && (
          <span style={{ color: 'var(--color-foreground-muted)', display: 'flex' }}>
            {icon}
          </span>
        )}
        <Text
          size={fontSizes[size]}
          fw={500}
          tt="uppercase"
          style={{
            color: 'var(--color-foreground-muted)',
            letterSpacing: '0.04em',
          }}
        >
          {title}
        </Text>
        {count !== undefined && (
          <Text
            size="xs"
            fw={500}
            style={{
              color: 'var(--color-foreground-subtle)',
              backgroundColor: 'var(--color-background-subtle)',
              padding: `${rem(2)} ${rem(10)}`,
              borderRadius: 'var(--radius-full)',
              fontSize: rem(11),
            }}
          >
            {count}
          </Text>
        )}
      </Group>
      {actions}
    </Group>
  )
}
