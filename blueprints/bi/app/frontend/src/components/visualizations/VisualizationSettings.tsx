import { Stack, Text, Accordion, Switch, NumberInput, Select, ColorInput, Group } from '@mantine/core'
import { IconEye, IconNumbers, IconPalette } from '@tabler/icons-react'
import type { VisualizationSettings } from '../../api/types'

// Import modular settings components
import {
  NumberSettings,
  TrendSettings,
  ProgressSettings,
  GaugeSettings,
  LineSettings,
  AreaSettings,
  BarSettings,
  PieSettings,
  ScatterSettings,
  BubbleSettings,
  FunnelSettings,
  WaterfallSettings,
  ComboSettings,
  PivotSettings,
  SankeySettings,
  MapSettings,
} from './settings'

interface VisualizationSettingsEditorProps {
  visualization: VisualizationSettings
  onChange: (settings: VisualizationSettings) => void
  columns?: { name: string; type: string }[]
}

export default function VisualizationSettingsEditor({
  visualization,
  onChange,
  columns = [],
}: VisualizationSettingsEditorProps) {
  const { type, settings = {} } = visualization

  const updateSetting = (key: string, value: any) => {
    onChange({
      ...visualization,
      settings: {
        ...settings,
        [key]: value,
      },
    })
  }

  const getSettingsForType = () => {
    switch (type) {
      case 'number':
        return <NumberSettings settings={settings} onChange={updateSetting} />
      case 'trend':
        return <TrendSettings settings={settings} onChange={updateSetting} />
      case 'progress':
        return <ProgressSettings settings={settings} onChange={updateSetting} />
      case 'gauge':
        return <GaugeSettings settings={settings} onChange={updateSetting} />
      case 'line':
        return <LineSettings settings={settings} onChange={updateSetting} columns={columns} />
      case 'area':
        return <AreaSettings settings={settings} onChange={updateSetting} columns={columns} />
      case 'bar':
        return <BarSettings settings={settings} onChange={updateSetting} columns={columns} isRow={false} />
      case 'row':
        return <BarSettings settings={settings} onChange={updateSetting} columns={columns} isRow={true} />
      case 'pie':
        return <PieSettings settings={settings} onChange={updateSetting} columns={columns} isDonut={false} />
      case 'donut':
        return <PieSettings settings={settings} onChange={updateSetting} columns={columns} isDonut={true} />
      case 'scatter':
        return <ScatterSettings settings={settings} onChange={updateSetting} columns={columns} />
      case 'bubble':
        return <BubbleSettings settings={settings} onChange={updateSetting} columns={columns} />
      case 'funnel':
        return <FunnelSettings settings={settings} onChange={updateSetting} columns={columns} />
      case 'waterfall':
        return <WaterfallSettings settings={settings} onChange={updateSetting} columns={columns} />
      case 'combo':
        return <ComboSettings settings={settings} onChange={updateSetting} columns={columns} />
      case 'pivot':
        return <PivotSettings settings={settings} onChange={updateSetting} columns={columns} />
      case 'sankey':
        return <SankeySettings settings={settings} onChange={updateSetting} columns={columns} />
      case 'map-pin':
        return <MapSettings settings={settings} onChange={updateSetting} columns={columns} mapType="pin" />
      case 'map-grid':
        return <MapSettings settings={settings} onChange={updateSetting} columns={columns} mapType="grid" />
      case 'map-region':
        return <MapSettings settings={settings} onChange={updateSetting} columns={columns} mapType="region" />
      case 'table':
        return <TableSettings settings={settings} onChange={updateSetting} columns={columns} />
      default:
        return <GeneralSettings settings={settings} onChange={updateSetting} />
    }
  }

  return (
    <Stack gap="md">
      {getSettingsForType()}
    </Stack>
  )
}

// General settings shared by many chart types (fallback)
function GeneralSettings({
  settings,
  onChange,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}) {
  return (
    <Accordion variant="separated" defaultValue="display">
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
              label="Show values"
              checked={settings.showValues ?? false}
              onChange={(e) => onChange('showValues', e.currentTarget.checked)}
            />
            <Switch
              label="Show grid lines"
              checked={settings.showGrid ?? true}
              onChange={(e) => onChange('showGrid', e.currentTarget.checked)}
            />
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}

// Table settings with conditional formatting (keeping inline as it uses columns prop heavily)
function TableSettings({
  settings,
  onChange,
  columns,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
  columns: { name: string; type: string }[]
}) {
  return (
    <Accordion variant="separated" defaultValue="display">
      <Accordion.Item value="display">
        <Accordion.Control icon={<IconEye size={16} />}>
          Display
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="sm">
            <NumberInput
              label="Row limit"
              value={settings.maxRows ?? 100}
              onChange={(v) => onChange('maxRows', v)}
              min={1}
              max={10000}
            />
            <Switch
              label="Striped rows"
              checked={settings.striped ?? true}
              onChange={(e) => onChange('striped', e.currentTarget.checked)}
            />
            <Switch
              label="Highlight on hover"
              checked={settings.highlightOnHover ?? true}
              onChange={(e) => onChange('highlightOnHover', e.currentTarget.checked)}
            />
            <Switch
              label="Sticky header"
              checked={settings.stickyHeader ?? true}
              onChange={(e) => onChange('stickyHeader', e.currentTarget.checked)}
            />
            <Switch
              label="Show row numbers"
              checked={settings.showRowIndex ?? false}
              onChange={(e) => onChange('showRowIndex', e.currentTarget.checked)}
            />
            <Switch
              label="Paginate results"
              checked={settings.paginateResults ?? false}
              onChange={(e) => onChange('paginateResults', e.currentTarget.checked)}
            />
            {settings.paginateResults && (
              <Select
                label="Page size"
                value={String(settings.pageSize || 25)}
                onChange={(v) => onChange('pageSize', Number(v))}
                data={['10', '25', '50', '100']}
              />
            )}
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>

      <Accordion.Item value="formatting">
        <Accordion.Control icon={<IconNumbers size={16} />}>
          Conditional Formatting
        </Accordion.Control>
        <Accordion.Panel>
          <Stack gap="sm">
            <Switch
              label="Enable conditional formatting"
              checked={settings.conditionalFormatting ?? false}
              onChange={(e) => onChange('conditionalFormatting', e.currentTarget.checked)}
            />
            {settings.conditionalFormatting && (
              <>
                <Select
                  label="Apply to column"
                  placeholder="Select column"
                  value={settings.formatColumn || ''}
                  onChange={(v) => onChange('formatColumn', v)}
                  data={columns
                    .filter(c => ['number', 'integer', 'float'].includes(c.type))
                    .map(c => ({ value: c.name, label: c.name }))}
                />
                <Select
                  label="Format type"
                  value={settings.formatType || 'background'}
                  onChange={(v) => onChange('formatType', v)}
                  data={[
                    { value: 'background', label: 'Background color' },
                    { value: 'bar', label: 'Mini bar' },
                    { value: 'text', label: 'Text color' },
                  ]}
                />
                <Group grow>
                  <ColorInput
                    label="Min color"
                    value={settings.minColor || '#ff6b6b'}
                    onChange={(v) => onChange('minColor', v)}
                  />
                  <ColorInput
                    label="Max color"
                    value={settings.maxColor || '#51cf66'}
                    onChange={(v) => onChange('maxColor', v)}
                  />
                </Group>
              </>
            )}
          </Stack>
        </Accordion.Panel>
      </Accordion.Item>

      <Accordion.Item value="columns">
        <Accordion.Control icon={<IconPalette size={16} />}>
          Column Settings
        </Accordion.Control>
        <Accordion.Panel>
          <Text size="sm" c="dimmed">
            Configure individual column display options like width, alignment, and number formatting in the table header menu.
          </Text>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}
