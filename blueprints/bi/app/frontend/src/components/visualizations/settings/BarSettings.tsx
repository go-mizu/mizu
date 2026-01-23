import {
  Stack, Switch, Select, TextInput, NumberInput, Accordion, Text, ColorInput, Group
} from '@mantine/core'
import { IconEye, IconTarget, IconAxisX, IconPalette } from '@tabler/icons-react'
import { chartColors } from '../../../theme'
import type { ChartSettingsProps } from './types'

interface BarSettingsProps extends ChartSettingsProps {
  isRow?: boolean
}

export default function BarSettings({
  settings,
  onChange,
  columns = [],
  isRow = false,
}: BarSettingsProps) {
  const numericColumns = columns.filter(c => ['number', 'integer', 'float'].includes(c.type))
  const seriesColors = settings.seriesColors || {}

  return (
    <Accordion variant="separated" defaultValue="display">
      <Accordion.Item value="data">
        <Accordion.Control icon={<IconPalette size={16} />}>
          Data
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="sm">
            <Select
              label={isRow ? "Y-axis column (categories)" : "X-axis column (categories)"}
              value={settings.xColumn || ''}
              onChange={(v) => onChange('xColumn', v)}
              data={columns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect"
              clearable
            />
            <Text size="sm" fw={500} mt="sm">Series colors</Text>
            {numericColumns.map((col, i) => (
              <Group key={col.name} gap="sm">
                <Text size="sm" style={{ flex: 1 }}>{col.name}</Text>
                <ColorInput
                  value={seriesColors[col.name] || chartColors[i % chartColors.length]}
                  onChange={(v) => onChange('seriesColors', { ...seriesColors, [col.name]: v })}
                  format="hex"
                  swatches={chartColors.slice(0, 6)}
                  size="xs"
                  style={{ width: 100 }}
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
          <Stack gap="sm">
            <Switch
              label="Show legend"
              checked={settings.showLegend ?? true}
              onChange={(e) => onChange('showLegend', e.currentTarget.checked)}
            />
            <Switch
              label="Show data labels"
              checked={settings.showLabels ?? false}
              onChange={(e) => onChange('showLabels', e.currentTarget.checked)}
            />
            <Select
              label="Stacking"
              value={settings.stacking || 'none'}
              onChange={(v) => onChange('stacking', v)}
              data={[
                { value: 'none', label: 'None (side by side)' },
                { value: 'stacked', label: 'Stacked' },
                { value: 'normalized', label: 'Stacked 100%' },
              ]}
            />
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>

      <Accordion.Item value="goal">
        <Accordion.Control icon={<IconTarget size={16} />}>
          Goal Line
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="sm">
            <Switch
              label="Show goal line"
              checked={settings.showGoal ?? false}
              onChange={(e) => onChange('showGoal', e.currentTarget.checked)}
            />
            {settings.showGoal && (
              <NumberInput
                label="Goal value"
                value={settings.goalValue ?? 0}
                onChange={(v) => onChange('goalValue', v)}
              />
            )}
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>

      <Accordion.Item value="axes">
        <Accordion.Control icon={<IconAxisX size={16} />}>
          Axes
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="sm">
            <TextInput
              label={isRow ? "Y-axis label" : "X-axis label"}
              value={settings.xAxisLabel || ''}
              onChange={(e) => onChange('xAxisLabel', e.target.value)}
            />
            <TextInput
              label={isRow ? "X-axis label" : "Y-axis label"}
              value={settings.yAxisLabel || ''}
              onChange={(e) => onChange('yAxisLabel', e.target.value)}
            />
            <Group grow>
              <NumberInput
                label="Min"
                placeholder="Auto"
                value={settings.yAxisMin ?? ''}
                onChange={(v) => onChange('yAxisMin', v)}
              />
              <NumberInput
                label="Max"
                placeholder="Auto"
                value={settings.yAxisMax ?? ''}
                onChange={(v) => onChange('yAxisMax', v)}
              />
            </Group>
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}
