import { Stack, Loader, Text } from '@mantine/core'

interface LoadingStateProps {
  message?: string
}

export function LoadingState({ message = 'Loading...' }: LoadingStateProps) {
  return (
    <Stack align="center" justify="center" py="xl" gap="md">
      <Loader size="lg" color="orange" />
      <Text size="sm" c="dimmed">
        {message}
      </Text>
    </Stack>
  )
}
