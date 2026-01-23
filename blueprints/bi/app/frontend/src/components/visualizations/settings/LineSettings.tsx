import {
  Stack, Group, Switch, Select, TextInput, NumberInput, Accordion, Text, Divider, ColorInput
} from '@mantine/core'
import { IconEye, IconTarget, IconChartLine, IconAxisX, IconPalette } from '@tabler/icons-react'
import { chartColors } from '../../../theme'
import type { ChartSettingsProps } from './types'

export default function LineSettings({
  settings,
  onChange,
  columns = [],
}: ChartSettingsProps) {
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
              label="X-axis column"
              description="Column for the horizontal axis"
              value={settings.xColumn || ''}
              onChange={(v) => onChange('xColumn', v)}
              data={columns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect"
              clearable
            />
            <Text size="sm" fw={500} mt="sm">Y-axis column(s)</Text>
            <Text size="xs" c="dimmed">Select columns to display as lines</Text>
            {numericColumns.map((col, i) => (
              <Group key={col.name} gap="sm">
                <Switch
                  label={col.name}
                  checked={!settings.hiddenSeries?.[col.name]}
                  onChange={(e) => onChange('hiddenSeries', {
                    ...settings.hiddenSeries,
                    [col.name]: !e.currentTarget.checked,
                  })}
                  style={{ flex: 1 }}
                />
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
              label="Show data points"
              checked={settings.showPoints ?? true}
              onChange={(e) => onChange('showPoints', e.currentTarget.checked)}
            />
            <Switch
              label="Show data labels"
              checked={settings.showLabels ?? false}
              onChange={(e) => onChange('showLabels', e.currentTarget.checked)}
            />
            <Select
              label="Line style"
              value={settings.lineStyle || 'solid'}
              onChange={(v) => onChange('lineStyle', v)}
              data={[
                { value: 'solid', label: 'Solid' },
                { value: 'dashed', label: 'Dashed' },
                { value: 'dotted', label: 'Dotted' },
              ]}
            />
            <Select
              label="Curve type"
              value={settings.interpolation || 'monotone'}
              onChange={(v) => onChange('interpolation', v)}
              data={[
                { value: 'linear', label: 'Linear (straight lines)' },
                { value: 'monotone', label: 'Smooth (curved)' },
                { value: 'step', label: 'Step (stepped lines)' },
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
              <>
                <NumberInput
                  label="Goal value"
                  value={settings.goalValue ?? 0}
                  onChange={(v) => onChange('goalValue', v)}
                />
                <TextInput
                  label="Goal label"
                  placeholder="Target"
                  value={settings.goalLabel || ''}
                  onChange={(e) => onChange('goalLabel', e.target.value)}
                />
              </>
            )}
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>

      <Accordion.Item value="trend">
        <Accordion.Control icon={<IconChartLine size={16} />}>
          Trend Line
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="sm">
            <Switch
              label="Show trend line"
              description="Adds a linear regression trend line to the chart"
              checked={settings.showTrend ?? false}
              onChange={(e) => onChange('showTrend', e.currentTarget.checked)}
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
            <Text size="sm" fw={500}>X-Axis</Text>
            <TextInput
              label="Label"
              placeholder="X-axis label"
              value={settings.xAxisLabel || ''}
              onChange={(e) => onChange('xAxisLabel', e.target.value)}
            />
            <Switch
              label="Show grid"
              checked={settings.xAxisGrid ?? true}
              onChange={(e) => onChange('xAxisGrid', e.currentTarget.checked)}
            />
            <Divider my="xs" />
            <Text size="sm" fw={500}>Y-Axis</Text>
            <TextInput
              label="Label"
              placeholder="Y-axis label"
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
            <Select
              label="Scale"
              value={settings.yAxisScale || 'linear'}
              onChange={(v) => onChange('yAxisScale', v)}
              data={[
                { value: 'linear', label: 'Linear' },
                { value: 'log', label: 'Logarithmic' },
              ]}
            />
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}
