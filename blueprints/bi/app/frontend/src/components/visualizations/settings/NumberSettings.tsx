import { Stack, Group, TextInput, NumberInput, Select, Switch } from '@mantine/core'
import type { BaseSettingsProps } from './types'

export default function NumberSettings({
  settings,
  onChange,
}: BaseSettingsProps) {
  return (
    <Stack gap="md">
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
      <Select
        label="Number style"
        value={settings.style || 'decimal'}
        onChange={(v) => onChange('style', v)}
        data={[
          { value: 'decimal', label: 'Decimal (1,234.56)' },
          { value: 'percent', label: 'Percent (12.34%)' },
          { value: 'currency', label: 'Currency ($1,234.56)' },
          { value: 'scientific', label: 'Scientific (1.23e+3)' },
        ]}
      />
      {settings.style === 'currency' && (
        <Select
          label="Currency"
          value={settings.currency || 'USD'}
          onChange={(v) => onChange('currency', v)}
          data={[
            { value: 'USD', label: 'US Dollar ($)' },
            { value: 'EUR', label: 'Euro (\u20ac)' },
            { value: 'GBP', label: 'British Pound (\u00a3)' },
            { value: 'JPY', label: 'Japanese Yen (\u00a5)' },
            { value: 'CNY', label: 'Chinese Yuan (\u00a5)' },
          ]}
        />
      )}
      <Switch
        label="Compact notation (K, M, B)"
        checked={settings.compact ?? false}
        onChange={(e) => onChange('compact', e.currentTarget.checked)}
      />
    </Stack>
  )
}
