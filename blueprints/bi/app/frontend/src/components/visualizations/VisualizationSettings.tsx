import { useState } from 'react'
import {
  Box, Stack, Group, Text, Switch, NumberInput, Select, TextInput,
  ColorInput, Divider, Accordion, Tooltip, ActionIcon, Paper, Tabs,
  SegmentedControl, Slider
} from '@mantine/core'
import {
  IconChartLine, IconPalette, IconTarget, IconEye, IconNumbers,
  IconAxisX, IconAxisY, IconInfoCircle
} from '@tabler/icons-react'
import type { VisualizationSettings, VisualizationType } from '../../api/types'
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
      case 'trend':
        return <NumberSettings settings={settings} onChange={updateSetting} />
      case 'progress':
        return <ProgressSettings settings={settings} onChange={updateSetting} />
      case 'gauge':
        return <GaugeSettings settings={settings} onChange={updateSetting} />
      case 'line':
      case 'area':
        return <LineAreaSettings settings={settings} onChange={updateSetting} />
      case 'bar':
      case 'row':
        return <BarSettings settings={settings} onChange={updateSetting} />
      case 'pie':
      case 'donut':
        return <PieSettings settings={settings} onChange={updateSetting} />
      case 'scatter':
        return <ScatterSettings settings={settings} onChange={updateSetting} />
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

// Number/Trend settings
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
          { value: 'decimal', label: 'Decimal' },
          { value: 'percent', label: 'Percent' },
          { value: 'currency', label: 'Currency' },
          { value: 'scientific', label: 'Scientific' },
        ]}
      />
      <Switch
        label="Compact notation (K, M, B)"
        checked={settings.compact ?? false}
        onChange={(e) => onChange('compact', e.currentTarget.checked)}
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
      <Stack gap="xs">
        <Text size="sm" fw={500}>Ranges</Text>
        <Group grow>
          <NumberInput
            label="Warning"
            value={settings.warningThreshold ?? 60}
            onChange={(v) => onChange('warningThreshold', v)}
          />
          <NumberInput
            label="Success"
            value={settings.successThreshold ?? 80}
            onChange={(v) => onChange('successThreshold', v)}
          />
        </Group>
      </Stack>
    </Stack>
  )
}

// Line/Area chart settings
function LineAreaSettings({
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
              label="Interpolation"
              value={settings.interpolation || 'monotone'}
              onChange={(v) => onChange('interpolation', v)}
              data={[
                { value: 'linear', label: 'Linear' },
                { value: 'monotone', label: 'Smooth' },
                { value: 'step', label: 'Step' },
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
              checked={settings.showTrend ?? false}
              onChange={(e) => onChange('showTrend', e.currentTarget.checked)}
            />
            {settings.showTrend && (
              <Select
                label="Trend type"
                value={settings.trendType || 'linear'}
                onChange={(v) => onChange('trendType', v)}
                data={[
                  { value: 'linear', label: 'Linear' },
                  { value: 'exponential', label: 'Exponential' },
                  { value: 'polynomial', label: 'Polynomial' },
                ]}
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

// Bar chart settings
function BarSettings({
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
              label="Show data labels"
              checked={settings.showLabels ?? false}
              onChange={(e) => onChange('showLabels', e.currentTarget.checked)}
            />
            <Select
              label="Stacking"
              value={settings.stacking || 'none'}
              onChange={(v) => onChange('stacking', v)}
              data={[
                { value: 'none', label: 'None' },
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
    </Accordion>
  )
}

// Pie/Donut settings
function PieSettings({
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
      <Switch
        label="Show labels"
        checked={settings.showLabels ?? true}
        onChange={(e) => onChange('showLabels', e.currentTarget.checked)}
      />
      <Switch
        label="Show percentages"
        checked={settings.showPercentages ?? true}
        onChange={(e) => onChange('showPercentages', e.currentTarget.checked)}
      />
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
      <Switch
        label="Show trend line"
        checked={settings.showTrend ?? false}
        onChange={(e) => onChange('showTrend', e.currentTarget.checked)}
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
                    .filter(c => c.type === 'number')
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
            Configure individual column display options like width, alignment, and number formatting.
          </Text>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  )
}
