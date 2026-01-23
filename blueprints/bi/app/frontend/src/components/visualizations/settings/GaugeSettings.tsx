import { Stack, Group, NumberInput, Divider, Text } from '@mantine/core'
import type { BaseSettingsProps } from './types'

export default function GaugeSettings({
  settings,
  onChange,
}: BaseSettingsProps) {
  return (
    <Stack gap="md">
      <Group grow>
        <NumberInput
          label="Min value"
          value={settings.min ?? 0}
          onChange={(v) => onChange('min', v)}
        />
        <NumberInput
          label="Max value"
          value={settings.max ?? 100}
          onChange={(v) => onChange('max', v)}
        />
      </Group>
      <Divider my="xs" />
      <Text size="sm" fw={500}>Color Ranges</Text>
      <Text size="xs" c="dimmed">Values below Warning show red, between Warning and Success show yellow, above Success show green.</Text>
      <Group grow>
        <NumberInput
          label="Warning threshold"
          value={settings.warningThreshold ?? 60}
          onChange={(v) => onChange('warningThreshold', v)}
        />
        <NumberInput
          label="Success threshold"
          value={settings.successThreshold ?? 80}
          onChange={(v) => onChange('successThreshold', v)}
        />
      </Group>
    </Stack>
  )
}
