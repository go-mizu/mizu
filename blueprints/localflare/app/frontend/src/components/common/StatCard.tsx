import { Paper, Group, Text, ThemeIcon, Stack } from '@mantine/core'
import { IconArrowUpRight, IconArrowDownRight } from '@tabler/icons-react'
import type { ReactNode } from 'react'

interface StatCardProps {
  icon: ReactNode
  label: string
  value: string | number
  trend?: {
    value: number
    direction: 'up' | 'down'
  }
  color?: 'default' | 'success' | 'warning' | 'error' | 'orange'
  description?: string
}

const colorMap = {
  default: 'gray',
  success: 'green',
  warning: 'yellow',
  error: 'red',
  orange: 'orange',
}

export function StatCard({ icon, label, value, trend, color = 'default', description }: StatCardProps) {
  return (
    <Paper p="md" radius="md" withBorder>
      <Group justify="space-between" align="flex-start">
        <Stack gap={4}>
          <Text size="xs" c="dimmed" tt="uppercase" fw={600}>
            {label}
          </Text>
          <Text size="xl" fw={700}>
            {typeof value === 'number' ? value.toLocaleString() : value}
          </Text>
          {description && (
            <Text size="xs" c="dimmed">
              {description}
            </Text>
          )}
          {trend && (
            <Group gap={4}>
              {trend.direction === 'up' ? (
                <IconArrowUpRight size={16} color="var(--mantine-color-green-6)" />
              ) : (
                <IconArrowDownRight size={16} color="var(--mantine-color-red-6)" />
              )}
              <Text size="xs" c={trend.direction === 'up' ? 'green' : 'red'} fw={500}>
                {trend.value}%
              </Text>
            </Group>
          )}
        </Stack>
        <ThemeIcon variant="light" size="lg" radius="md" color={colorMap[color]}>
          {icon}
        </ThemeIcon>
      </Group>
    </Paper>
  )
}
