import { Box, Group, Title, Text, Breadcrumbs, Anchor, rem } from '@mantine/core'
import { IconChevronRight } from '@tabler/icons-react'

// =============================================================================
// PAGE HEADER - Consistent page title with optional breadcrumbs and actions
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
}

export function PageHeader({
  title,
  subtitle,
  breadcrumbs,
  actions,
  children,
}: PageHeaderProps) {
  return (
    <Box
      style={{
        padding: `${rem(16)} ${rem(24)}`,
        backgroundColor: 'var(--color-background)',
        borderBottom: '1px solid var(--color-border)',
      }}
    >
      {breadcrumbs && breadcrumbs.length > 0 && (
        <Breadcrumbs
          mb="xs"
          separator={<IconChevronRight size={14} color="var(--color-foreground-subtle)" />}
          styles={{
            root: { flexWrap: 'wrap' },
            separator: { marginLeft: rem(4), marginRight: rem(4) },
          }}
        >
          {breadcrumbs.map((crumb, index) => (
            <Anchor
              key={index}
              onClick={crumb.onClick}
              href={crumb.href}
              size="sm"
              c={index === breadcrumbs.length - 1 ? 'var(--color-foreground)' : 'var(--color-foreground-muted)'}
              fw={index === breadcrumbs.length - 1 ? 500 : 400}
              style={{ cursor: 'pointer' }}
            >
              {crumb.label}
            </Anchor>
          ))}
        </Breadcrumbs>
      )}

      <Group justify="space-between" align="flex-start" wrap="nowrap">
        <Box style={{ flex: 1, minWidth: 0 }}>
          <Title
            order={2}
            style={{
              fontSize: rem(24),
              fontWeight: 600,
              color: 'var(--color-foreground)',
              lineHeight: 1.3,
            }}
          >
            {title}
          </Title>
          {subtitle && (
            <Text
              size="sm"
              c="dimmed"
              mt={4}
              style={{ color: 'var(--color-foreground-muted)' }}
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
// SECTION HEADER - For content sections within a page
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
}

export function SectionHeader({
  title,
  icon,
  count,
  actions,
  style,
}: SectionHeaderProps) {
  return (
    <Group
      justify="space-between"
      mb="md"
      style={style}
    >
      <Group gap="xs" align="center">
        {icon}
        <Text
          size="xs"
          fw={600}
          tt="uppercase"
          style={{
            color: 'var(--color-foreground-muted)',
            letterSpacing: '0.05em',
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
              padding: `${rem(2)} ${rem(8)}`,
              borderRadius: 'var(--radius-full)',
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
