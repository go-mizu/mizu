import { Stack, Switch, NumberInput, Text, Group, Select, ColorInput, Accordion } from '@mantine/core'
import { IconEye, IconPalette } from '@tabler/icons-react'
import { chartColors } from '../../../theme'
import type { ChartSettingsProps } from './types'

export default function BubbleSettings({
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
              placeholder="Auto-detect (first numeric)"
              clearable
            />
            <Select
              label="Y-axis column"
              value={settings.yColumn || ''}
              onChange={(v) => onChange('yColumn', v)}
              data={numericColumns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect (second numeric)"
              clearable
            />
            <Select
              label="Bubble size column"
              value={settings.sizeColumn || ''}
              onChange={(v) => onChange('sizeColumn', v)}
              data={numericColumns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect (third numeric)"
              clearable
            />
            <Select
              label="Color dimension"
              description="Group bubbles by this column"
              value={settings.colorColumn || ''}
              onChange={(v) => onChange('colorColumn', v)}
              data={columns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="None"
              clearable
            />
            <ColorInput
              label="Default bubble color"
              value={settings.bubbleColor || chartColors[0]}
              onChange={(v) => onChange('bubbleColor', v)}
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
            <Text size="sm" fw={500}>Bubble Size Range</Text>
            <Group grow>
              <NumberInput
                label="Min size"
                value={settings.minBubbleSize ?? 20}
                onChange={(v) => onChange('minBubbleSize', v)}
                min={5}
                max={100}
              />
              <NumberInput
                label="Max size"
                value={settings.maxBubbleSize ?? 400}
                onChange={(v) => onChange('maxBubbleSize', v)}
                min={50}
                max={1000}
              />
            </Group>
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}
