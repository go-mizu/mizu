import { Stack, Switch, Select, Accordion } from '@mantine/core'
import { IconEye, IconPalette } from '@tabler/icons-react'
import type { ChartSettingsProps } from './types'

export default function PivotSettings({
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
              label="Row dimension"
              description="Values for row headers"
              value={settings.rowDimension || ''}
              onChange={(v) => onChange('rowDimension', v)}
              data={textColumns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect"
              clearable
            />
            <Select
              label="Column dimension"
              description="Values for column headers"
              value={settings.columnDimension || ''}
              onChange={(v) => onChange('columnDimension', v)}
              data={textColumns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect"
              clearable
            />
            <Select
              label="Value column"
              description="Values to aggregate in cells"
              value={settings.valueColumn || ''}
              onChange={(v) => onChange('valueColumn', v)}
              data={numericColumns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect"
              clearable
            />
            <Select
              label="Aggregation"
              value={settings.aggregation || 'sum'}
              onChange={(v) => onChange('aggregation', v)}
              data={[
                { value: 'sum', label: 'Sum' },
                { value: 'avg', label: 'Average' },
                { value: 'count', label: 'Count' },
                { value: 'min', label: 'Minimum' },
                { value: 'max', label: 'Maximum' },
              ]}
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
              label="Show row totals"
              checked={settings.showRowTotals ?? true}
              onChange={(e) => onChange('showRowTotals', e.currentTarget.checked)}
            />
            <Switch
              label="Show column totals"
              checked={settings.showColumnTotals ?? true}
              onChange={(e) => onChange('showColumnTotals', e.currentTarget.checked)}
            />
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}
