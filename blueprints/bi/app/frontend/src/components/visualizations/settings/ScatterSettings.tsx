import { Stack, Switch, NumberInput, Divider, Select, Accordion, ColorInput } from '@mantine/core'
import { IconEye, IconTarget, IconPalette } from '@tabler/icons-react'
import { chartColors } from '../../../theme'
import type { ChartSettingsProps } from './types'

export default function ScatterSettings({
  settings,
  onChange,
  columns = [],
}: ChartSettingsProps) {
  const numericColumns = columns.filter(c => ['number', 'integer', 'float'].includes(c.type))

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
              value={settings.xColumn || ''}
              onChange={(v) => onChange('xColumn', v)}
              data={numericColumns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect"
              clearable
            />
            <Select
              label="Y-axis column"
              value={settings.yColumn || ''}
              onChange={(v) => onChange('yColumn', v)}
              data={numericColumns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect"
              clearable
            />
            <Select
              label="Color dimension"
              description="Group points by this column"
              value={settings.colorColumn || ''}
              onChange={(v) => onChange('colorColumn', v)}
              data={columns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="None"
              clearable
            />
            <ColorInput
              label="Point color"
              value={settings.pointColor || chartColors[0]}
              onChange={(v) => onChange('pointColor', v)}
              format="hex"
              swatches={chartColors.slice(0, 8)}
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
              label="Show legend"
              checked={settings.showLegend ?? true}
              onChange={(e) => onChange('showLegend', e.currentTarget.checked)}
            />
            <NumberInput
              label="Dot size"
              value={settings.dotSize ?? 8}
              onChange={(v) => onChange('dotSize', v)}
              min={2}
              max={20}
            />
            <Divider my="xs" />
            <Switch
              label="Show trend line"
              description="Adds a linear regression trend line"
              checked={settings.showTrend ?? false}
              onChange={(e) => onChange('showTrend', e.currentTarget.checked)}
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
    </Accordion>
  )
}
