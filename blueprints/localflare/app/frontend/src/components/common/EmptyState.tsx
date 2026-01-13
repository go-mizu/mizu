import { Stack, Text, ThemeIcon, Button } from '@mantine/core'
import { IconInbox, IconPlus } from '@tabler/icons-react'
import type { ReactNode } from 'react'

interface EmptyStateProps {
  icon?: ReactNode
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
  }
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <Stack align="center" justify="center" py="xl" gap="md">
      <ThemeIcon size={64} radius="xl" variant="light" color="gray">
        {icon || <IconInbox size={32} stroke={1.5} />}
      </ThemeIcon>
      <Stack gap={4} align="center">
        <Text size="lg" fw={600}>
          {title}
        </Text>
        {description && (
          <Text size="sm" c="dimmed" ta="center" maw={400}>
            {description}
          </Text>
        )}
      </Stack>
      {action && (
        <Button
          leftSection={<IconPlus size={16} />}
          onClick={action.onClick}
          variant="light"
        >
          {action.label}
        </Button>
      )}
    </Stack>
  )
}
