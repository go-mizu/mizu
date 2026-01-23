import { Stack, Switch, Select, ColorInput, Accordion, Group, Text } from '@mantine/core'
import { IconEye, IconPalette } from '@tabler/icons-react'
import { chartColors } from '../../../theme'
import type { ChartSettingsProps } from './types'

export default function FunnelSettings({
  settings,
  onChange,
  columns = [],
}: ChartSettingsProps) {
  const stepColors = settings.stepColors || {}

  return (
    <Accordion variant="separated" defaultValue="display">
      <Accordion.Item value="data">
        <Accordion.Control icon={<IconPalette size={16} />}>
          Data
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="sm">
            <Select
              label="Stage/Step column"
              description="Column containing funnel step names"
              value={settings.labelColumn || ''}
              onChange={(v) => onChange('labelColumn', v)}
              data={columns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect (first column)"
              clearable
            />
            <Select
              label="Value column"
              description="Column containing step values"
              value={settings.valueColumn || ''}
              onChange={(v) => onChange('valueColumn', v)}
              data={columns
                .filter(c => ['number', 'integer', 'float'].includes(c.type))
                .map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect (first numeric)"
              clearable
            />
            <Text size="sm" fw={500} mt="sm">Step colors</Text>
            {[0, 1, 2, 3, 4].map(i => (
              <Group key={i} gap="sm">
                <Text size="sm" c="dimmed" style={{ width: 80 }}>Step {i + 1}</Text>
                <ColorInput
                  value={stepColors[i] || chartColors[i % chartColors.length]}
                  onChange={(v) => onChange('stepColors', { ...stepColors, [i]: v })}
                  format="hex"
                  swatches={chartColors.slice(0, 6)}
                  size="xs"
                  style={{ flex: 1 }}
                />
              </Group>
            ))}
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
              label="Show labels"
              checked={settings.showLabels ?? true}
              onChange={(e) => onChange('showLabels', e.currentTarget.checked)}
            />
            <Switch
              label="Show percentages"
              description="Show percentage value for each step"
              checked={settings.showPercentage ?? true}
              onChange={(e) => onChange('showPercentage', e.currentTarget.checked)}
            />
            <Switch
              label="Show conversion rates"
              description="Show conversion rate between steps"
              checked={settings.showConversion ?? true}
              onChange={(e) => onChange('showConversion', e.currentTarget.checked)}
            />
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}
