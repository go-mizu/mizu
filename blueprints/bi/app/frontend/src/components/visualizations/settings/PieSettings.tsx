import {
  Stack, Switch, NumberInput, Text, Select, ColorSwatch, Group, Accordion
} from '@mantine/core'
import { IconEye, IconPalette } from '@tabler/icons-react'
import { chartColors } from '../../../theme'
import type { ChartSettingsProps } from './types'

interface PieSettingsProps extends ChartSettingsProps {
  isDonut?: boolean
}

export default function PieSettings({
  settings,
  onChange,
  columns = [],
  isDonut = false,
}: PieSettingsProps) {
  const sliceColors = settings.sliceColors || {}

  return (
    <Accordion variant="separated" defaultValue="display">
      <Accordion.Item value="data">
        <Accordion.Control icon={<IconPalette size={16} />}>
          Data
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="sm">
            <Select
              label="Labels column"
              description="Column for slice names"
              value={settings.labelColumn || ''}
              onChange={(v) => onChange('labelColumn', v)}
              data={columns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect (first column)"
              clearable
            />
            <Select
              label="Values column"
              description="Column for slice sizes"
              value={settings.valueColumn || ''}
              onChange={(v) => onChange('valueColumn', v)}
              data={columns
                .filter(c => ['number', 'integer', 'float'].includes(c.type))
                .map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect (first numeric)"
              clearable
            />
            <Text size="sm" fw={500} mt="sm">Slice colors</Text>
            <Text size="xs" c="dimmed">Click to change individual slice colors</Text>
            <Group gap="xs">
              {chartColors.slice(0, 10).map((color, i) => (
                <ColorSwatch
                  key={i}
                  color={sliceColors[i] || color}
                  size={28}
                  radius="xl"
                  style={{ cursor: 'pointer', border: '2px solid transparent' }}
                  onClick={() => {
                    // Cycle through palette colors
                    const nextColor = chartColors[(chartColors.indexOf(sliceColors[i] || color) + 1) % chartColors.length]
                    onChange('sliceColors', { ...sliceColors, [i]: nextColor })
                  }}
                />
              ))}
            </Group>
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
              label="Show legend"
              checked={settings.showLegend ?? true}
              onChange={(e) => onChange('showLegend', e.currentTarget.checked)}
            />
            <Switch
              label="Show slice labels"
              checked={settings.showLabels ?? true}
              onChange={(e) => onChange('showLabels', e.currentTarget.checked)}
            />
            <Switch
              label="Show percentages"
              checked={settings.showPercentages ?? true}
              onChange={(e) => onChange('showPercentages', e.currentTarget.checked)}
            />
            {isDonut && (
              <Switch
                label="Show total in center"
                checked={settings.showTotal ?? true}
                onChange={(e) => onChange('showTotal', e.currentTarget.checked)}
              />
            )}
            <NumberInput
              label="Min slice percentage"
              description="Slices below this threshold are grouped as 'Other'"
              value={settings.minSlicePercent ?? 0}
              onChange={(v) => onChange('minSlicePercent', v)}
              min={0}
              max={25}
              suffix="%"
            />
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}
