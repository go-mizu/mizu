import { useState, useMemo } from 'react'
import {
  Modal, Box, Text, Group, UnstyledButton, SimpleGrid, Paper, Tabs, Badge,
  Stack, ThemeIcon
} from '@mantine/core'
import {
  IconTable, IconHash, IconTrendingUp, IconProgress, IconGauge,
  IconChartLine, IconChartArea, IconChartBar, IconChartPie, IconChartDonut,
  IconChartScatter, IconChartArrows, IconMap, IconMapPin, IconTablePivot,
  IconArrowsSort, IconChartHistogram
} from '@tabler/icons-react'
import type { VisualizationType } from '../../api/types'
import { chartColors } from '../../theme'

interface VisualizationPickerProps {
  opened: boolean
  onClose: () => void
  value: VisualizationType
  onChange: (type: VisualizationType) => void
}

interface VizOption {
  value: VisualizationType
  label: string
  icon: typeof IconTable
  description: string
}

const vizOptions: Record<string, VizOption[]> = {
  'Table': [
    { value: 'table', label: 'Table', icon: IconTable, description: 'Display data in rows and columns' },
  ],
  'Numbers': [
    { value: 'number', label: 'Number', icon: IconHash, description: 'Show a single value' },
    { value: 'trend', label: 'Trend', icon: IconTrendingUp, description: 'Number with trend indicator' },
    { value: 'progress', label: 'Progress', icon: IconProgress, description: 'Progress towards a goal' },
    { value: 'gauge', label: 'Gauge', icon: IconGauge, description: 'Gauge chart with min/max' },
  ],
  'Time Series': [
    { value: 'line', label: 'Line', icon: IconChartLine, description: 'Show trends over time' },
    { value: 'area', label: 'Area', icon: IconChartArea, description: 'Filled area chart' },
  ],
  'Bar Charts': [
    { value: 'bar', label: 'Bar', icon: IconChartBar, description: 'Vertical bar chart' },
    { value: 'row', label: 'Row', icon: IconChartHistogram, description: 'Horizontal bar chart' },
    { value: 'combo', label: 'Combo', icon: IconChartArrows, description: 'Bar and line combined' },
  ],
  'Parts of Whole': [
    { value: 'pie', label: 'Pie', icon: IconChartPie, description: 'Pie chart' },
    { value: 'donut', label: 'Donut', icon: IconChartDonut, description: 'Donut chart' },
    { value: 'funnel', label: 'Funnel', icon: IconArrowsSort, description: 'Funnel chart' },
  ],
  'Distribution': [
    { value: 'scatter', label: 'Scatter', icon: IconChartScatter, description: 'Scatter plot' },
  ],
  'Maps': [
    { value: 'map-pin', label: 'Pin Map', icon: IconMapPin, description: 'Map with pin markers' },
    { value: 'map-region', label: 'Region Map', icon: IconMap, description: 'Colored regions map' },
  ],
  'Advanced': [
    { value: 'pivot', label: 'Pivot Table', icon: IconTablePivot, description: 'Cross-tabulation table' },
  ],
}

export default function VisualizationPicker({
  opened,
  onClose,
  value,
  onChange,
}: VisualizationPickerProps) {
  const [activeTab, setActiveTab] = useState<string | null>('Table')

  const handleSelect = (type: VisualizationType) => {
    onChange(type)
    onClose()
  }

  const currentCategory = useMemo(() => {
    for (const [category, options] of Object.entries(vizOptions)) {
      if (options.some(opt => opt.value === value)) {
        return category
      }
    }
    return 'Table'
  }, [value])

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title="Choose visualization type"
      size="lg"
    >
      <Tabs value={activeTab || currentCategory} onChange={setActiveTab}>
        <Tabs.List mb="md">
          {Object.keys(vizOptions).map(category => (
            <Tabs.Tab key={category} value={category}>
              {category}
            </Tabs.Tab>
          ))}
        </Tabs.List>

        {Object.entries(vizOptions).map(([category, options]) => (
          <Tabs.Panel key={category} value={category}>
            <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
              {options.map(option => {
                const Icon = option.icon
                const isSelected = value === option.value
                return (
                  <UnstyledButton
                    key={option.value}
                    onClick={() => handleSelect(option.value)}
                  >
                    <Paper
                      withBorder
                      p="md"
                      radius="md"
                      style={{
                        borderColor: isSelected ? 'var(--mantine-color-brand-5)' : undefined,
                        backgroundColor: isSelected ? 'var(--mantine-color-brand-0)' : undefined,
                        cursor: 'pointer',
                      }}
                    >
                      <Group gap="md">
                        <ThemeIcon
                          size={40}
                          radius="md"
                          variant={isSelected ? 'filled' : 'light'}
                          color="brand"
                        >
                          <Icon size={20} />
                        </ThemeIcon>
                        <div style={{ flex: 1 }}>
                          <Group gap="xs">
                            <Text fw={500}>{option.label}</Text>
                            {isSelected && (
                              <Badge size="xs" color="brand">Selected</Badge>
                            )}
                          </Group>
                          <Text size="sm" c="dimmed">{option.description}</Text>
                        </div>
                      </Group>
                    </Paper>
                  </UnstyledButton>
                )
              })}
            </SimpleGrid>
          </Tabs.Panel>
        ))}
      </Tabs>
    </Modal>
  )
}

// Quick visualization type selector (inline)
export function VisualizationTypeSelect({
  value,
  onChange,
}: {
  value: VisualizationType
  onChange: (type: VisualizationType) => void
}) {
  const quickOptions: { value: VisualizationType; icon: typeof IconTable }[] = [
    { value: 'table', icon: IconTable },
    { value: 'number', icon: IconHash },
    { value: 'line', icon: IconChartLine },
    { value: 'bar', icon: IconChartBar },
    { value: 'pie', icon: IconChartPie },
  ]

  return (
    <Group gap={4}>
      {quickOptions.map(option => {
        const Icon = option.icon
        const isSelected = value === option.value
        return (
          <UnstyledButton
            key={option.value}
            onClick={() => onChange(option.value)}
            style={{
              padding: '6px 10px',
              borderRadius: 6,
              backgroundColor: isSelected ? 'var(--mantine-color-brand-0)' : 'transparent',
              border: isSelected ? '1px solid var(--mantine-color-brand-5)' : '1px solid transparent',
            }}
          >
            <Icon
              size={18}
              color={isSelected ? 'var(--mantine-color-brand-5)' : 'var(--mantine-color-gray-6)'}
            />
          </UnstyledButton>
        )
      })}
    </Group>
  )
}
