import { Stack, Group, TextInput, NumberInput, Divider, Text, Switch } from '@mantine/core'
import type { BaseSettingsProps } from './types'

export default function TrendSettings({
  settings,
  onChange,
}: BaseSettingsProps) {
  return (
    <Stack gap="md">
      <Text size="sm" fw={500}>Number Formatting</Text>
      <Group grow>
        <TextInput
          label="Prefix"
          placeholder="$"
          value={settings.prefix || ''}
          onChange={(e) => onChange('prefix', e.target.value)}
        />
        <TextInput
          label="Suffix"
          placeholder="%"
          value={settings.suffix || ''}
          onChange={(e) => onChange('suffix', e.target.value)}
        />
      </Group>
      <NumberInput
        label="Decimal places"
        value={settings.decimals ?? 0}
        onChange={(v) => onChange('decimals', v)}
        min={0}
        max={10}
      />
      <Divider my="xs" />
      <Text size="sm" fw={500}>Trend Behavior</Text>
      <Switch
        label="Reverse colors (up is bad)"
        description="Show increases in red and decreases in green"
        checked={settings.reverseColors ?? false}
        onChange={(e) => onChange('reverseColors', e.currentTarget.checked)}
      />
    </Stack>
  )
}
