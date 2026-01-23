import { Stack, ColorInput, Select, Accordion, Text } from '@mantine/core'
import { IconEye, IconPalette } from '@tabler/icons-react'
import { chartColors } from '../../../theme'
import type { ChartSettingsProps } from './types'

interface MapSettingsProps extends ChartSettingsProps {
  mapType: 'pin' | 'grid' | 'region'
}

export default function MapSettings({
  settings,
  onChange,
  columns = [],
  mapType,
}: MapSettingsProps) {
  const numericColumns = columns.filter(c => ['number', 'integer', 'float'].includes(c.type))

  return (
    <Accordion variant="separated" defaultValue="data">
      <Accordion.Item value="data">
        <Accordion.Control icon={<IconPalette size={16} />}>
          Data
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="sm">
            {mapType === 'pin' && (
              <>
                <Select
                  label="Latitude column"
                  value={settings.latColumn || ''}
                  onChange={(v) => onChange('latColumn', v)}
                  data={numericColumns.map(c => ({ value: c.name, label: c.name }))}
                  placeholder="Auto-detect"
                  clearable
                />
                <Select
                  label="Longitude column"
                  value={settings.lngColumn || ''}
                  onChange={(v) => onChange('lngColumn', v)}
                  data={numericColumns.map(c => ({ value: c.name, label: c.name }))}
                  placeholder="Auto-detect"
                  clearable
                />
              </>
            )}
            {mapType === 'region' && (
              <Select
                label="Region column"
                description="Column containing region names or codes"
                value={settings.regionColumn || ''}
                onChange={(v) => onChange('regionColumn', v)}
                data={columns.map(c => ({ value: c.name, label: c.name }))}
                placeholder="Auto-detect"
                clearable
              />
            )}
            <Select
              label="Value column"
              description="Column for sizing/coloring"
              value={settings.valueColumn || ''}
              onChange={(v) => onChange('valueColumn', v)}
              data={numericColumns.map(c => ({ value: c.name, label: c.name }))}
              placeholder="Auto-detect"
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
            <Text size="sm" c="dimmed">
              Map visualizations use a simplified tile-based display.
              Full geographic maps are coming in a future update.
            </Text>
            <ColorInput
              label="Base color"
              value={settings.baseColor || chartColors[0]}
              onChange={(v) => onChange('baseColor', v)}
              format="hex"
              swatches={chartColors.slice(0, 8)}
            />
            {mapType === 'grid' && (
              <Select
                label="Color scale"
                value={settings.colorScale || 'linear'}
                onChange={(v) => onChange('colorScale', v)}
                data={[
                  { value: 'linear', label: 'Linear' },
                  { value: 'quantile', label: 'Quantile' },
                  { value: 'log', label: 'Logarithmic' },
                ]}
              />
            )}
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}
