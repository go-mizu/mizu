import { Paper, Stack, Text, Title, Box, Group, rem } from '@mantine/core'
import { IconInbox } from '@tabler/icons-react'

// =============================================================================
// EMPTY STATE - Modern empty/no-data display (shadcn-inspired)
// =============================================================================

export interface EmptyStateProps {
  /** Icon to display */
  icon?: React.ReactNode
  /** Icon color */
  iconColor?: string
  /** Main title */
  title: string
  /** Description text or element */
  description?: React.ReactNode
  /** Action buttons */
  action?: React.ReactNode
  /** Additional actions */
  secondaryAction?: React.ReactNode
  /** Size variant */
  size?: 'sm' | 'md' | 'lg'
  /** Whether to show border */
  withBorder?: boolean
}

export function EmptyState({
  icon,
  iconColor = 'var(--color-primary)',
  title,
  description,
  action,
  secondaryAction,
  size = 'md',
  withBorder = true,
}: EmptyStateProps) {
  const iconSizes = {
    sm: 56,
    md: 72,
    lg: 88,
  }

  const innerIconSizes = {
    sm: 26,
    md: 34,
    lg: 42,
  }

  const paddings = {
    sm: rem(32),
    md: rem(48),
    lg: rem(64),
  }

  const Container = withBorder ? Paper : 'div'
  const containerProps = withBorder
    ? {
        radius: 'xl',
        p: paddings[size],
        style: {
          border: '1px solid var(--color-border)',
          backgroundColor: 'var(--color-background)',
        }
      }
    : { style: { padding: paddings[size] } }

  return (
    <Container {...containerProps as any}>
      <Stack align="center" gap="xl" ta="center">
        {/* Modern icon container with subtle gradient */}
        <Box
          style={{
            width: iconSizes[size],
            height: iconSizes[size],
            borderRadius: 'var(--radius-2xl)',
            backgroundColor: `${iconColor}08`,
            border: `1px solid ${iconColor}15`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <span style={{ color: iconColor, display: 'flex' }}>
            {icon || <IconInbox size={innerIconSizes[size]} strokeWidth={1.5} />}
          </span>
        </Box>

        <div>
          <Title
            order={size === 'sm' ? 5 : size === 'md' ? 4 : 3}
            mb={rem(8)}
            style={{
              color: 'var(--color-foreground)',
              letterSpacing: '-0.02em',
            }}
          >
            {title}
          </Title>
          {description && (
            <Text
              size={size === 'sm' ? 'sm' : 'md'}
              style={{
                color: 'var(--color-foreground-muted)',
                lineHeight: 1.6,
              }}
              maw={420}
              mx="auto"
            >
              {description}
            </Text>
          )}
        </div>

        {(action || secondaryAction) && (
          <Group justify="center" gap="sm" mt={rem(8)}>
            {action}
            {secondaryAction}
          </Group>
        )}
      </Stack>
    </Container>
  )
}

// =============================================================================
// INLINE EMPTY STATE - Smaller, inline version for tables/lists
// =============================================================================

export interface InlineEmptyStateProps {
  /** Icon to display */
  icon?: React.ReactNode
  /** Message text */
  message: string
  /** Action button */
  action?: React.ReactNode
}

export function InlineEmptyState({
  icon,
  message,
  action,
}: InlineEmptyStateProps) {
  return (
    <Group
      justify="center"
      py="xl"
      style={{ color: 'var(--color-foreground-muted)' }}
    >
      {icon && <span style={{ opacity: 0.5 }}>{icon}</span>}
      <Text size="sm">{message}</Text>
      {action}
    </Group>
  )
}

// =============================================================================
// LOADING STATE - Centered loading indicator
// =============================================================================

import { Loader } from '@mantine/core'

export interface LoadingStateProps {
  /** Loading message */
  message?: string
  /** Size variant */
  size?: 'sm' | 'md' | 'lg'
}

export function LoadingState({
  message = 'Loading...',
  size = 'md',
}: LoadingStateProps) {
  const loaderSizes = {
    sm: 'sm',
    md: 'md',
    lg: 'lg',
  } as const

  return (
    <Stack align="center" justify="center" py="xl" gap="md">
      <Loader size={loaderSizes[size]} color="var(--color-primary)" />
      {message && (
        <Text
          size="sm"
          style={{ color: 'var(--color-foreground-muted)' }}
        >
          {message}
        </Text>
      )}
    </Stack>
  )
}

// =============================================================================
// ERROR STATE - Display an error message
// =============================================================================

import { IconAlertCircle } from '@tabler/icons-react'
import { Button } from '@mantine/core'

export interface ErrorStateProps {
  /** Error title */
  title?: string
  /** Error message */
  message: string
  /** Retry action */
  onRetry?: () => void
  /** Size variant */
  size?: 'sm' | 'md' | 'lg'
}

export function ErrorState({
  title = 'Something went wrong',
  message,
  onRetry,
  size = 'md',
}: ErrorStateProps) {
  return (
    <EmptyState
      icon={<IconAlertCircle size={size === 'sm' ? 24 : 32} strokeWidth={1.5} />}
      iconColor="var(--color-error)"
      title={title}
      description={message}
      size={size}
      action={
        onRetry && (
          <Button variant="light" color="red" onClick={onRetry}>
            Try again
          </Button>
        )
      }
    />
  )
}
