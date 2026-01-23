import {
  Modal, Text, Group, UnstyledButton, SimpleGrid, Box, rem
} from '@mantine/core'
import {
  IconTable, IconHash, IconTrendingUp, IconProgress, IconGauge,
  IconChartLine, IconChartArea, IconChartBar, IconChartPie,
  IconChartScatter, IconChartArrows, IconMapPin, IconTableRow,
  IconArrowsSort, IconChartHistogram, IconChartCandle, IconRoute
} from '@tabler/icons-react'
import type { VisualizationType } from '../../api/types'

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
}

// Metabase-style grid layout
const primaryOptions: VizOption[] = [
  { value: 'table', label: 'Table', icon: IconTable },
  { value: 'bar', label: 'Bar', icon: IconChartBar },
  { value: 'line', label: 'Line', icon: IconChartLine },
  { value: 'pie', label: 'Pie', icon: IconChartPie },
  { value: 'row', label: 'Row', icon: IconChartHistogram },
  { value: 'area', label: 'Area', icon: IconChartArea },
  { value: 'combo', label: 'Combo', icon: IconChartArrows },
  { value: 'pivot', label: 'Pivot Table', icon: IconTableRow },
  { value: 'trend', label: 'Trend', icon: IconTrendingUp },
  { value: 'map-pin', label: 'Map', icon: IconMapPin },
  { value: 'scatter', label: 'Scatter', icon: IconChartScatter },
  { value: 'waterfall', label: 'Waterfall', icon: IconChartCandle },
]

const otherOptions: VizOption[] = [
  { value: 'number', label: 'Number', icon: IconHash },
  { value: 'gauge', label: 'Gauge', icon: IconGauge },
  { value: 'progress', label: 'Progress', icon: IconProgress },
  { value: 'funnel', label: 'Funnel', icon: IconArrowsSort },
  { value: 'sankey', label: 'Sankey', icon: IconRoute },
]

const styles = {
  optionButton: {
    display: 'flex',
    flexDirection: 'column' as const,
    alignItems: 'center',
    gap: rem(8),
    padding: rem(12),
    borderRadius: rem(8),
    cursor: 'pointer',
    transition: 'all 0.15s ease',
  },
  iconBox: {
    width: 48,
    height: 48,
    borderRadius: rem(8),
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    transition: 'all 0.15s ease',
  },
}

export default function VisualizationPicker({
  opened,
  onClose,
  value,
  onChange,
}: VisualizationPickerProps) {
  const handleSelect = (type: VisualizationType) => {
    onChange(type)
    onClose()
  }

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title="Choose visualization type"
      size="lg"
    >
      {/* Primary Chart Types - 4 column grid */}
      <SimpleGrid cols={4} spacing="xs" mb="lg">
        {primaryOptions.map(option => {
          const Icon = option.icon
          const isSelected = value === option.value
          return (
            <UnstyledButton
              key={option.value}
              onClick={() => handleSelect(option.value)}
              style={styles.optionButton}
              onMouseEnter={(e) => {
                if (!isSelected) {
                  e.currentTarget.style.backgroundColor = '#F9FBFC'
                }
              }}
              onMouseLeave={(e) => {
                if (!isSelected) {
                  e.currentTarget.style.backgroundColor = 'transparent'
                }
              }}
            >
              <Box
                style={{
                  ...styles.iconBox,
                  backgroundColor: isSelected ? '#509EE3' : '#F0F0F0',
                  border: isSelected ? 'none' : '1px solid #EEECEC',
                }}
              >
                <Icon
                  size={24}
                  color={isSelected ? '#ffffff' : '#4C5773'}
                  strokeWidth={1.5}
                />
              </Box>
              <Text
                size="xs"
                fw={isSelected ? 600 : 500}
                style={{ color: isSelected ? '#509EE3' : '#4C5773' }}
              >
                {option.label}
              </Text>
            </UnstyledButton>
          )
        })}
      </SimpleGrid>

      {/* Other Charts Section */}
      <Text
        size="xs"
        fw={700}
        tt="uppercase"
        c="dimmed"
        mb="sm"
        style={{ letterSpacing: '0.05em' }}
      >
        Other Charts
      </Text>
      <SimpleGrid cols={4} spacing="xs">
        {otherOptions.map(option => {
          const Icon = option.icon
          const isSelected = value === option.value
          return (
            <UnstyledButton
              key={option.value}
              onClick={() => handleSelect(option.value)}
              style={styles.optionButton}
              onMouseEnter={(e) => {
                if (!isSelected) {
                  e.currentTarget.style.backgroundColor = '#F9FBFC'
                }
              }}
              onMouseLeave={(e) => {
                if (!isSelected) {
                  e.currentTarget.style.backgroundColor = 'transparent'
                }
              }}
            >
              <Box
                style={{
                  ...styles.iconBox,
                  backgroundColor: isSelected ? '#509EE3' : '#F0F0F0',
                  border: isSelected ? 'none' : '1px solid #EEECEC',
                }}
              >
                <Icon
                  size={24}
                  color={isSelected ? '#ffffff' : '#4C5773'}
                  strokeWidth={1.5}
                />
              </Box>
              <Text
                size="xs"
                fw={isSelected ? 600 : 500}
                style={{ color: isSelected ? '#509EE3' : '#4C5773' }}
              >
                {option.label}
              </Text>
            </UnstyledButton>
          )
        })}
      </SimpleGrid>
    </Modal>
  )
}

// Quick visualization type selector (inline) - Metabase style
export function VisualizationTypeSelect({
  value,
  onChange,
}: {
  value: VisualizationType
  onChange: (type: VisualizationType) => void
}) {
  const quickOptions: { value: VisualizationType; icon: typeof IconTable }[] = [
    { value: 'table', icon: IconTable },
    { value: 'bar', icon: IconChartBar },
    { value: 'line', icon: IconChartLine },
    { value: 'pie', icon: IconChartPie },
    { value: 'number', icon: IconHash },
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
              width: 40,
              height: 40,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              borderRadius: 8,
              backgroundColor: isSelected ? '#509EE3' : '#F0F0F0',
              border: isSelected ? 'none' : '1px solid #EEECEC',
              cursor: 'pointer',
              transition: 'all 0.15s ease',
            }}
          >
            <Icon
              size={20}
              color={isSelected ? '#ffffff' : '#4C5773'}
              strokeWidth={1.5}
            />
          </UnstyledButton>
        )
      })}
    </Group>
  )
}
