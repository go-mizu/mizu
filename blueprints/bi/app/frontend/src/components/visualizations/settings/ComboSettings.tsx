import { Stack, Switch, Select, TextInput, Group, Text, Accordion, ColorInput, Paper } from '@mantine/core'
import { IconEye, IconAxisX, IconPalette } from '@tabler/icons-react'
import { chartColors } from '../../../theme'
import type { ChartSettingsProps } from './types'

export default function ComboSettings({
  settings,
  onChange,
  columns = [],
}: ChartSettingsProps) {
  const numericColumns = columns.filter(c => ['number', 'integer', 'float'].includes(c.type))
  const seriesColors = settings.seriesColors || {}
  const seriesTypes = settings.seriesTypes || {}

  return (
    <Accordion variant="separated" defaultValue="series">
      <Accordion.Item value="data">
        <Accordion.Control icon={<IconPalette size={16} />}>
          Data
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="sm">
            <Select
              label="X-axis column"
              value={settings.xColumn || ''}
              onChange={(v) => onChange('xColumn', v)}
              data={columns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect"
              clearable
            />
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>

      <Accordion.Item value="series">
        <Accordion.Control icon={<IconPalette size={16} />}>
          Series Configuration
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="sm">
            <Text size="sm" c="dimmed">Configure each series type and color</Text>
            {numericColumns.map((col, i) => (
              <Paper key={col.name} withBorder p="sm" radius="md">
                <Stack gap="xs">
                  <Text size="sm" fw={500}>{col.name}</Text>
                  <Group grow>
                    <Select
                      label="Chart type"
                      value={seriesTypes[col.name] || 'bar'}
                      onChange={(v) => onChange('seriesTypes', { ...seriesTypes, [col.name]: v })}
                      data={[
                        { value: 'bar', label: 'Bar' },
                        { value: 'line', label: 'Line' },
                        { value: 'area', label: 'Area' },
                      ]}
                      size="xs"
                    />
                    <ColorInput
                      label="Color"
                      value={seriesColors[col.name] || chartColors[i % chartColors.length]}
                      onChange={(v) => onChange('seriesColors', { ...seriesColors, [col.name]: v })}
                      format="hex"
                      swatches={chartColors.slice(0, 6)}
                      size="xs"
                    />
                  </Group>
                </Stack>
              </Paper>
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
              label="Show legend"
              checked={settings.showLegend ?? true}
              onChange={(e) => onChange('showLegend', e.currentTarget.checked)}
            />
            <Switch
              label="Show data labels"
              checked={settings.showLabels ?? false}
              onChange={(e) => onChange('showLabels', e.currentTarget.checked)}
            />
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
              label="X-axis label"
              value={settings.xAxisLabel || ''}
              onChange={(e) => onChange('xAxisLabel', e.target.value)}
            />
            <TextInput
              label="Y-axis label (left)"
              value={settings.yAxisLabel || ''}
              onChange={(e) => onChange('yAxisLabel', e.target.value)}
            />
            <TextInput
              label="Y-axis label (right)"
              value={settings.yAxisRightLabel || ''}
              onChange={(e) => onChange('yAxisRightLabel', e.target.value)}
            />
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}
