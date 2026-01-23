import { Paper, Stack, Text, Title, ThemeIcon, Group } from '@mantine/core'
import { IconInbox } from '@tabler/icons-react'

// =============================================================================
// EMPTY STATE - Consistent empty/no-data display
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
    sm: 48,
    md: 64,
    lg: 80,
  }

  const innerIconSizes = {
    sm: 24,
    md: 32,
    lg: 40,
  }

  const paddings = {
    sm: 'lg',
    md: 'xl',
    lg: '48px',
  }

  const Container = withBorder ? Paper : 'div'
  const containerProps = withBorder
    ? { withBorder: true, radius: 'md', p: paddings[size], bg: 'var(--color-background)' }
    : { style: { padding: paddings[size] } }

  return (
    <Container {...containerProps as any}>
      <Stack align="center" gap="lg" ta="center">
        <ThemeIcon
          size={iconSizes[size]}
          radius="xl"
          variant="light"
          style={{
            backgroundColor: `${iconColor}10`,
            color: iconColor,
          }}
        >
          {icon || <IconInbox size={innerIconSizes[size]} strokeWidth={1.5} />}
        </ThemeIcon>

        <div>
          <Title
            order={size === 'sm' ? 5 : size === 'md' ? 4 : 3}
            mb="xs"
            style={{ color: 'var(--color-foreground)' }}
          >
            {title}
          </Title>
          {description && (
            <Text
              size={size === 'sm' ? 'sm' : 'md'}
              style={{ color: 'var(--color-foreground-muted)' }}
              maw={400}
              mx="auto"
            >
              {description}
            </Text>
          )}
        </div>

        {(action || secondaryAction) && (
          <Group justify="center" gap="sm">
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
