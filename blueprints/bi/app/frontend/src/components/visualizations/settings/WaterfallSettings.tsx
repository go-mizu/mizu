import { Stack, Switch, ColorInput, Accordion, Text, Select } from '@mantine/core'
import { IconEye, IconPalette } from '@tabler/icons-react'
import type { ChartSettingsProps } from './types'

export default function WaterfallSettings({
  settings,
  onChange,
  columns = [],
}: ChartSettingsProps) {
  return (
    <Accordion variant="separated" defaultValue="display">
      <Accordion.Item value="data">
        <Accordion.Control icon={<IconPalette size={16} />}>
          Data
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="sm">
            <Select
              label="Label column"
              description="Column containing category labels"
              value={settings.labelColumn || ''}
              onChange={(v) => onChange('labelColumn', v)}
              data={columns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect (first column)"
              clearable
            />
            <Select
              label="Value column"
              description="Column containing values"
              value={settings.valueColumn || ''}
              onChange={(v) => onChange('valueColumn', v)}
              data={columns
                .filter(c => ['number', 'integer', 'float'].includes(c.type))
                .map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect (first numeric)"
              clearable
            />
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>

      <Accordion.Item value="display">
        <Accordion.Control icon={<IconEye size={16} />}>
          Display
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="md">
            <Switch
              label="Show data labels"
              checked={settings.showLabels ?? true}
              onChange={(e) => onChange('showLabels', e.currentTarget.checked)}
            />
            <Text size="sm" fw={500}>Colors</Text>
            <ColorInput
              label="Increase color"
              description="Color for positive values"
              value={settings.increaseColor || '#84bb4c'}
              onChange={(v) => onChange('increaseColor', v)}
              format="hex"
            />
            <ColorInput
              label="Decrease color"
              description="Color for negative values"
              value={settings.decreaseColor || '#ed6e6e'}
              onChange={(v) => onChange('decreaseColor', v)}
              format="hex"
            />
            <ColorInput
              label="Total color"
              description="Color for total/subtotal bars"
              value={settings.totalColor || '#509ee3'}
              onChange={(v) => onChange('totalColor', v)}
              format="hex"
            />
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}
