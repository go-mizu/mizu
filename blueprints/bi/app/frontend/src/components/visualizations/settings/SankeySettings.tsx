import { Stack, NumberInput, Select, Accordion, Slider, Text } from '@mantine/core'
import { IconEye, IconPalette } from '@tabler/icons-react'
import type { ChartSettingsProps } from './types'

export default function SankeySettings({
  settings,
  onChange,
  columns = [],
}: ChartSettingsProps) {
  const numericColumns = columns.filter(c => ['number', 'integer', 'float'].includes(c.type))
  const textColumns = columns.filter(c => !['number', 'integer', 'float'].includes(c.type))

  return (
    <Accordion variant="separated" defaultValue="data">
      <Accordion.Item value="data">
        <Accordion.Control icon={<IconPalette size={16} />}>
          Data
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="md">
            <Select
              label="Source column"
              description="Column containing source nodes"
              value={settings.sourceColumn || ''}
              onChange={(v) => onChange('sourceColumn', v)}
              data={textColumns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect (first column)"
              clearable
            />
            <Select
              label="Target column"
              description="Column containing target nodes"
              value={settings.targetColumn || ''}
              onChange={(v) => onChange('targetColumn', v)}
              data={textColumns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect (second column)"
              clearable
            />
            <Select
              label="Value column"
              description="Column containing flow values"
              value={settings.valueColumn || ''}
              onChange={(v) => onChange('valueColumn', v)}
              data={numericColumns.map(c => ({ value: c.name, label: c.name }))}
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
            <NumberInput
              label="Node width"
              value={settings.nodeWidth ?? 20}
              onChange={(v) => onChange('nodeWidth', v)}
              min={5}
              max={50}
            />
            <NumberInput
              label="Node padding"
              value={settings.nodePadding ?? 10}
              onChange={(v) => onChange('nodePadding', v)}
              min={0}
              max={50}
            />
            <Text size="sm" fw={500}>Link opacity</Text>
            <Slider
              value={settings.linkOpacity ?? 0.5}
              onChange={(v) => onChange('linkOpacity', v)}
              min={0.1}
              max={1}
              step={0.1}
              marks={[
                { value: 0.3, label: '30%' },
                { value: 0.5, label: '50%' },
                { value: 0.7, label: '70%' },
              ]}
            />
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}
