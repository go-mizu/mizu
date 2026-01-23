import {
  Stack, Group, Text, Switch, NumberInput, Select, TextInput,
  ColorInput, Divider, Accordion, Slider, SegmentedControl
} from '@mantine/core'
import {
  IconChartLine, IconPalette, IconTarget, IconEye, IconNumbers,
  IconAxisX, IconLayoutGrid, IconPercentage
} from '@tabler/icons-react'
import type { VisualizationSettings } from '../../api/types'
import { chartColors } from '../../theme'

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
        return <LineSettings settings={settings} onChange={updateSetting} />
      case 'area':
        return <AreaSettings settings={settings} onChange={updateSetting} />
      case 'bar':
      case 'row':
        return <BarSettings settings={settings} onChange={updateSetting} isRow={type === 'row'} />
      case 'pie':
      case 'donut':
        return <PieSettings settings={settings} onChange={updateSetting} isDonut={type === 'donut'} />
      case 'scatter':
        return <ScatterSettings settings={settings} onChange={updateSetting} />
      case 'bubble':
        return <BubbleSettings settings={settings} onChange={updateSetting} />
      case 'funnel':
        return <FunnelSettings settings={settings} onChange={updateSetting} />
      case 'waterfall':
        return <WaterfallSettings settings={settings} onChange={updateSetting} />
      case 'combo':
        return <ComboSettings settings={settings} onChange={updateSetting} columns={columns} />
      case 'pivot':
        return <PivotSettings settings={settings} onChange={updateSetting} />
      case 'sankey':
        return <SankeySettings settings={settings} onChange={updateSetting} />
      case 'map-pin':
      case 'map-grid':
      case 'map-region':
        return <MapSettings settings={settings} onChange={updateSetting} />
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

// General settings shared by many chart types
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

// Number visualization settings
function NumberSettings({
  settings,
  onChange,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}) {
  return (
    <Stack gap="md">
      <Group grow>
        <TextInput
          label="Prefix"
          placeholder="$"
          value={settings.prefix || ''}
          onChange={(e) => onChange('prefix', e.target.value)}
        />
        <TextInput
          label="Suffix"
          placeholder="%"
          value={settings.suffix || ''}
          onChange={(e) => onChange('suffix', e.target.value)}
        />
      </Group>
      <NumberInput
        label="Decimal places"
        value={settings.decimals ?? 0}
        onChange={(v) => onChange('decimals', v)}
        min={0}
        max={10}
      />
      <Select
        label="Number style"
        value={settings.style || 'decimal'}
        onChange={(v) => onChange('style', v)}
        data={[
          { value: 'decimal', label: 'Decimal (1,234.56)' },
          { value: 'percent', label: 'Percent (12.34%)' },
          { value: 'currency', label: 'Currency ($1,234.56)' },
          { value: 'scientific', label: 'Scientific (1.23e+3)' },
        ]}
      />
      {settings.style === 'currency' && (
        <Select
          label="Currency"
          value={settings.currency || 'USD'}
          onChange={(v) => onChange('currency', v)}
          data={[
            { value: 'USD', label: 'US Dollar ($)' },
            { value: 'EUR', label: 'Euro (\u20ac)' },
            { value: 'GBP', label: 'British Pound (\u00a3)' },
            { value: 'JPY', label: 'Japanese Yen (\u00a5)' },
            { value: 'CNY', label: 'Chinese Yuan (\u00a5)' },
          ]}
        />
      )}
      <Switch
        label="Compact notation (K, M, B)"
        checked={settings.compact ?? false}
        onChange={(e) => onChange('compact', e.currentTarget.checked)}
      />
    </Stack>
  )
}

// Trend visualization settings
function TrendSettings({
  settings,
  onChange,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}) {
  return (
    <Stack gap="md">
      <Text size="sm" fw={500}>Number Formatting</Text>
      <Group grow>
        <TextInput
          label="Prefix"
          placeholder="$"
          value={settings.prefix || ''}
          onChange={(e) => onChange('prefix', e.target.value)}
        />
        <TextInput
          label="Suffix"
          placeholder="%"
          value={settings.suffix || ''}
          onChange={(e) => onChange('suffix', e.target.value)}
        />
      </Group>
      <NumberInput
        label="Decimal places"
        value={settings.decimals ?? 0}
        onChange={(v) => onChange('decimals', v)}
        min={0}
        max={10}
      />
      <Divider my="xs" />
      <Text size="sm" fw={500}>Trend Behavior</Text>
      <Switch
        label="Reverse colors (up is bad)"
        description="Show increases in red and decreases in green"
        checked={settings.reverseColors ?? false}
        onChange={(e) => onChange('reverseColors', e.currentTarget.checked)}
      />
    </Stack>
  )
}

// Progress bar settings
function ProgressSettings({
  settings,
  onChange,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}) {
  return (
    <Stack gap="md">
      <NumberInput
        label="Goal"
        value={settings.goal || 100}
        onChange={(v) => onChange('goal', v)}
        min={0}
      />
      <ColorInput
        label="Bar color"
        value={settings.color || chartColors[0]}
        onChange={(v) => onChange('color', v)}
        format="hex"
        swatches={chartColors.slice(0, 8)}
      />
      <Switch
        label="Show percentage"
        checked={settings.showPercentage ?? true}
        onChange={(e) => onChange('showPercentage', e.currentTarget.checked)}
      />
    </Stack>
  )
}

// Gauge settings
function GaugeSettings({
  settings,
  onChange,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}) {
  return (
    <Stack gap="md">
      <Group grow>
        <NumberInput
          label="Min value"
          value={settings.min ?? 0}
          onChange={(v) => onChange('min', v)}
        />
        <NumberInput
          label="Max value"
          value={settings.max ?? 100}
          onChange={(v) => onChange('max', v)}
        />
      </Group>
      <Divider my="xs" />
      <Text size="sm" fw={500}>Color Ranges</Text>
      <Text size="xs" c="dimmed">Values below Warning show red, between Warning and Success show yellow, above Success show green.</Text>
      <Group grow>
        <NumberInput
          label="Warning threshold"
          value={settings.warningThreshold ?? 60}
          onChange={(v) => onChange('warningThreshold', v)}
        />
        <NumberInput
          label="Success threshold"
          value={settings.successThreshold ?? 80}
          onChange={(v) => onChange('successThreshold', v)}
        />
      </Group>
    </Stack>
  )
}

// Line chart settings
function LineSettings({
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

// Area chart settings
function AreaSettings({
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
              label="Show data points"
              checked={settings.showPoints ?? false}
              onChange={(e) => onChange('showPoints', e.currentTarget.checked)}
            />
            <Switch
              label="Show data labels"
              checked={settings.showLabels ?? false}
              onChange={(e) => onChange('showLabels', e.currentTarget.checked)}
            />
            <Select
              label="Curve type"
              value={settings.interpolation || 'monotone'}
              onChange={(v) => onChange('interpolation', v)}
              data={[
                { value: 'linear', label: 'Linear' },
                { value: 'monotone', label: 'Smooth' },
                { value: 'step', label: 'Step' },
              ]}
            />
            <Select
              label="Stacking"
              value={settings.stacking || 'stacked'}
              onChange={(v) => onChange('stacking', v)}
              data={[
                { value: 'none', label: 'None (overlapping)' },
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
              label="Y-axis label"
              value={settings.yAxisLabel || ''}
              onChange={(e) => onChange('yAxisLabel', e.target.value)}
            />
            <Group grow>
              <NumberInput
                label="Y-axis min"
                placeholder="Auto"
                value={settings.yAxisMin ?? ''}
                onChange={(v) => onChange('yAxisMin', v)}
              />
              <NumberInput
                label="Y-axis max"
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

// Bar chart settings
function BarSettings({
  settings,
  onChange,
  isRow,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
  isRow: boolean
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

// Pie/Donut settings
function PieSettings({
  settings,
  onChange,
  isDonut,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
  isDonut: boolean
}) {
  return (
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
  )
}

// Scatter plot settings
function ScatterSettings({
  settings,
  onChange,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}) {
  return (
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
      <Divider my="xs" />
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
  )
}

// Bubble chart settings
function BubbleSettings({
  settings,
  onChange,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}) {
  return (
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
  )
}

// Funnel chart settings
function FunnelSettings({
  settings,
  onChange,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}) {
  return (
    <Stack gap="md">
      <Switch
        label="Show step labels"
        checked={settings.showLabels ?? true}
        onChange={(e) => onChange('showLabels', e.currentTarget.checked)}
      />
      <Switch
        label="Show conversion percentages"
        description="Shows percentage of first step value"
        checked={settings.showPercentage ?? true}
        onChange={(e) => onChange('showPercentage', e.currentTarget.checked)}
      />
    </Stack>
  )
}

// Waterfall chart settings
function WaterfallSettings({
  settings,
  onChange,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}) {
  return (
    <Stack gap="md">
      <Switch
        label="Show value labels"
        checked={settings.showLabels ?? false}
        onChange={(e) => onChange('showLabels', e.currentTarget.checked)}
      />
      <Divider my="xs" />
      <Text size="sm" fw={500}>Colors</Text>
      <ColorInput
        label="Increase color"
        value={settings.increaseColor || 'var(--mantine-color-green-5)'}
        onChange={(v) => onChange('increaseColor', v)}
        format="hex"
        swatches={['#40c057', '#228be6', '#7950f2']}
      />
      <ColorInput
        label="Decrease color"
        value={settings.decreaseColor || 'var(--mantine-color-red-5)'}
        onChange={(v) => onChange('decreaseColor', v)}
        format="hex"
        swatches={['#fa5252', '#fd7e14', '#fab005']}
      />
      <ColorInput
        label="Total color"
        value={settings.totalColor || 'var(--mantine-color-brand-5)'}
        onChange={(v) => onChange('totalColor', v)}
        format="hex"
        swatches={chartColors.slice(0, 6)}
      />
    </Stack>
  )
}

// Combo chart settings
function ComboSettings({
  settings,
  onChange,
  columns,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
  columns: { name: string; type: string }[]
}) {
  const seriesTypes = settings.seriesTypes || {}
  const metricColumns = columns.filter(c => ['number', 'integer', 'float'].includes(c.type))

  return (
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
      {metricColumns.length > 0 && (
        <>
          <Divider my="xs" />
          <Text size="sm" fw={500}>Series Types</Text>
          <Text size="xs" c="dimmed">Choose how each metric is displayed</Text>
          {metricColumns.map(col => (
            <Select
              key={col.name}
              label={col.name}
              value={seriesTypes[col.name] || 'bar'}
              onChange={(v) => onChange('seriesTypes', { ...seriesTypes, [col.name]: v })}
              data={[
                { value: 'bar', label: 'Bar' },
                { value: 'line', label: 'Line' },
                { value: 'area', label: 'Area' },
              ]}
            />
          ))}
        </>
      )}
    </Stack>
  )
}

// Pivot table settings
function PivotSettings({
  settings,
  onChange,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}) {
  return (
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
  )
}

// Sankey chart settings
function SankeySettings({
  settings,
  onChange,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}) {
  return (
    <Stack gap="md">
      <NumberInput
        label="Node width"
        value={settings.nodeWidth ?? 20}
        onChange={(v) => onChange('nodeWidth', v)}
        min={10}
        max={50}
      />
      <NumberInput
        label="Node padding"
        value={settings.nodePadding ?? 10}
        onChange={(v) => onChange('nodePadding', v)}
        min={2}
        max={30}
      />
      <Text size="sm" fw={500}>Link opacity</Text>
      <Slider
        value={settings.linkOpacity ?? 0.4}
        onChange={(v) => onChange('linkOpacity', v)}
        min={0.1}
        max={0.8}
        step={0.1}
        marks={[
          { value: 0.2, label: '0.2' },
          { value: 0.5, label: '0.5' },
          { value: 0.8, label: '0.8' },
        ]}
      />
    </Stack>
  )
}

// Map visualization settings
function MapSettings({
  settings,
  onChange,
}: {
  settings: Record<string, any>
  onChange: (key: string, value: any) => void
}) {
  return (
    <Stack gap="md">
      <ColorInput
        label="Base color"
        description="Color gradient will be based on this color"
        value={settings.baseColor || '#509EE3'}
        onChange={(v) => onChange('baseColor', v)}
        format="hex"
        swatches={chartColors.slice(0, 8)}
      />
    </Stack>
  )
}

// Table settings with conditional formatting
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
