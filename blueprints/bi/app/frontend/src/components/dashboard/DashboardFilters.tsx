import { useState, useMemo } from 'react'
import {
  Box, Group, Paper, Text, Button, ActionIcon, Modal, Stack,
  Select, TextInput, Tooltip, Badge, Collapse, Menu, NumberInput
} from '@mantine/core'
import { DatePickerInput, DateInput } from '@mantine/dates'
import { useDisclosure } from '@mantine/hooks'
import {
  IconPlus, IconFilter, IconX, IconChevronDown, IconCalendar,
  IconHash, IconLetterCase, IconList, IconSearch, IconAdjustments
} from '@tabler/icons-react'
import type { DashboardFilter, Column } from '../../api/types'

interface DashboardFiltersProps {
  filters: DashboardFilter[]
  filterValues: Record<string, any>
  onFilterChange: (filterId: string, value: any) => void
  onAddFilter?: (filter: Omit<DashboardFilter, 'id'>) => void
  onRemoveFilter?: (filterId: string) => void
  editMode?: boolean
  availableColumns?: Column[]
}

export default function DashboardFilters({
  filters,
  filterValues,
  onFilterChange,
  onAddFilter,
  onRemoveFilter,
  editMode = false,
  availableColumns = [],
}: DashboardFiltersProps) {
  const [addFilterModalOpen, { open: openAddFilter, close: closeAddFilter }] = useDisclosure(false)
  const [expanded, setExpanded] = useState(true)

  const activeFilterCount = useMemo(() => {
    return Object.values(filterValues).filter(v => v !== null && v !== undefined && v !== '').length
  }, [filterValues])

  if (filters.length === 0 && !editMode) {
    return null
  }

  return (
    <Paper
      withBorder
      radius="sm"
      mb="md"
      style={{
        backgroundColor: 'var(--mantine-color-gray-0)',
        borderColor: 'var(--mantine-color-gray-2)',
      }}
    >
      {/* Filter Bar Header */}
      <Group
        justify="space-between"
        px="md"
        py="xs"
        style={{
          cursor: 'pointer',
          borderBottom: expanded ? '1px solid var(--mantine-color-gray-2)' : 'none',
        }}
        onClick={() => setExpanded(!expanded)}
      >
        <Group gap="sm">
          <IconFilter size={16} color="var(--mantine-color-filter-5)" />
          <Text size="sm" fw={600} c="dimmed">Filters</Text>
          {activeFilterCount > 0 && (
            <Badge size="sm" variant="filled" color="filter">
              {activeFilterCount} active
            </Badge>
          )}
        </Group>
        <Group gap="xs">
          {editMode && (
            <Tooltip label="Add filter">
              <ActionIcon
                variant="subtle"
                color="filter"
                size="sm"
                onClick={(e) => {
                  e.stopPropagation()
                  openAddFilter()
                }}
              >
                <IconPlus size={14} />
              </ActionIcon>
            </Tooltip>
          )}
          <ActionIcon variant="subtle" color="gray" size="sm">
            <IconChevronDown
              size={14}
              style={{
                transform: expanded ? 'rotate(180deg)' : 'rotate(0deg)',
                transition: 'transform 0.2s ease',
              }}
            />
          </ActionIcon>
        </Group>
      </Group>

      {/* Filter Widgets */}
      <Collapse in={expanded}>
        <Group gap="md" p="md" wrap="wrap">
          {filters.map((filter) => (
            <FilterWidget
              key={filter.id}
              filter={filter}
              value={filterValues[filter.id]}
              onChange={(value) => onFilterChange(filter.id, value)}
              onRemove={editMode && onRemoveFilter ? () => onRemoveFilter(filter.id) : undefined}
            />
          ))}
          {filters.length === 0 && editMode && (
            <Text size="sm" c="dimmed">
              No filters yet. Click + to add a filter widget.
            </Text>
          )}
        </Group>
      </Collapse>

      {/* Add Filter Modal */}
      <AddFilterModal
        opened={addFilterModalOpen}
        onClose={closeAddFilter}
        onAdd={(filter) => {
          onAddFilter?.(filter)
          closeAddFilter()
        }}
        availableColumns={availableColumns}
      />
    </Paper>
  )
}

// Individual Filter Widget
function FilterWidget({
  filter,
  value,
  onChange,
  onRemove,
}: {
  filter: DashboardFilter
  value: any
  onChange: (value: any) => void
  onRemove?: () => void
}) {
  const [focused, setFocused] = useState(false)

  const getIcon = () => {
    switch (filter.type) {
      case 'time':
        return <IconCalendar size={14} />
      case 'number':
        return <IconHash size={14} />
      case 'text':
        return <IconLetterCase size={14} />
      case 'category':
        return <IconList size={14} />
      default:
        return <IconSearch size={14} />
    }
  }

  const renderInput = () => {
    switch (filter.display_type || filter.type) {
      case 'date':
      case 'time':
        return (
          <DatePickerInput
            size="xs"
            placeholder={filter.name}
            value={value ? new Date(value) : null}
            onChange={(date) => onChange(date?.toISOString() || null)}
            leftSection={getIcon()}
            clearable
            style={{ minWidth: 180 }}
          />
        )
      case 'dropdown':
      case 'category':
        return (
          <Select
            size="xs"
            placeholder={filter.name}
            value={value || null}
            onChange={onChange}
            data={[
              // TODO: Populate from card column values
              { value: 'option1', label: 'Option 1' },
              { value: 'option2', label: 'Option 2' },
            ]}
            clearable
            searchable
            leftSection={getIcon()}
            style={{ minWidth: 150 }}
          />
        )
      case 'number':
        return (
          <NumberInput
            size="xs"
            placeholder={filter.name}
            value={value ?? ''}
            onChange={onChange}
            leftSection={getIcon()}
            style={{ minWidth: 120 }}
          />
        )
      case 'search':
      case 'text':
      default:
        return (
          <TextInput
            size="xs"
            placeholder={filter.name}
            value={value || ''}
            onChange={(e) => onChange(e.target.value)}
            leftSection={getIcon()}
            rightSection={
              value ? (
                <ActionIcon size="xs" variant="subtle" onClick={() => onChange('')}>
                  <IconX size={12} />
                </ActionIcon>
              ) : null
            }
            style={{ minWidth: 150 }}
          />
        )
    }
  }

  return (
    <Box
      style={{
        position: 'relative',
        borderRadius: 4,
        outline: focused ? '2px solid var(--mantine-color-filter-5)' : 'none',
        outlineOffset: 2,
      }}
      onFocus={() => setFocused(true)}
      onBlur={() => setFocused(false)}
    >
      <Group gap={4}>
        <Text size="xs" fw={600} c="dimmed" mb={2}>
          {filter.name}
          {filter.required && <span style={{ color: 'var(--mantine-color-error-5)' }}> *</span>}
        </Text>
        {onRemove && (
          <ActionIcon size="xs" variant="subtle" color="gray" onClick={onRemove}>
            <IconX size={12} />
          </ActionIcon>
        )}
      </Group>
      {renderInput()}
    </Box>
  )
}

// Add Filter Modal
function AddFilterModal({
  opened,
  onClose,
  onAdd,
  availableColumns,
}: {
  opened: boolean
  onClose: () => void
  onAdd: (filter: Omit<DashboardFilter, 'id'>) => void
  availableColumns: Column[]
}) {
  const [name, setName] = useState('')
  const [type, setType] = useState<DashboardFilter['type']>('text')
  const [displayType, setDisplayType] = useState<DashboardFilter['display_type']>('search')
  const [required, setRequired] = useState(false)

  const handleAdd = () => {
    if (!name.trim()) return
    onAdd({
      dashboard_id: '',
      name,
      type,
      display_type: displayType,
      required,
      targets: [],
    })
    setName('')
    setType('text')
    setDisplayType('search')
    setRequired(false)
  }

  return (
    <Modal opened={opened} onClose={onClose} title="Add Dashboard Filter">
      <Stack gap="md">
        <TextInput
          label="Filter Name"
          placeholder="e.g., Date Range, Status, Category"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
        <Select
          label="Filter Type"
          value={type}
          onChange={(v) => setType(v as DashboardFilter['type'])}
          data={[
            { value: 'time', label: 'Time/Date' },
            { value: 'text', label: 'Text' },
            { value: 'number', label: 'Number' },
            { value: 'category', label: 'Category' },
            { value: 'id', label: 'ID' },
            { value: 'location', label: 'Location' },
          ]}
        />
        <Select
          label="Display Type"
          value={displayType || 'search'}
          onChange={(v) => setDisplayType(v as DashboardFilter['display_type'])}
          data={[
            { value: 'search', label: 'Search box' },
            { value: 'dropdown', label: 'Dropdown list' },
            { value: 'date', label: 'Date picker' },
            { value: 'input', label: 'Input field' },
          ]}
        />
        <Group justify="flex-end" mt="md">
          <Button variant="subtle" onClick={onClose}>Cancel</Button>
          <Button onClick={handleAdd} disabled={!name.trim()}>
            Add Filter
          </Button>
        </Group>
      </Stack>
    </Modal>
  )
}

// Compact date range filter
export function DateRangeFilter({
  value,
  onChange,
}: {
  value: { start: Date | null; end: Date | null }
  onChange: (range: { start: Date | null; end: Date | null }) => void
}) {
  return (
    <Group gap="xs">
      <DateInput
        size="xs"
        placeholder="Start date"
        value={value.start}
        onChange={(date) => onChange({ ...value, start: date })}
        clearable
        style={{ width: 130 }}
      />
      <Text size="xs" c="dimmed">to</Text>
      <DateInput
        size="xs"
        placeholder="End date"
        value={value.end}
        onChange={(date) => onChange({ ...value, end: date })}
        clearable
        style={{ width: 130 }}
      />
    </Group>
  )
}

// Quick date presets
export function QuickDateFilter({
  value,
  onChange,
}: {
  value: string
  onChange: (value: string) => void
}) {
  return (
    <Menu position="bottom-start">
      <Menu.Target>
        <Button
          variant="subtle"
          size="xs"
          leftSection={<IconCalendar size={14} />}
          rightSection={<IconChevronDown size={12} />}
        >
          {value || 'All time'}
        </Button>
      </Menu.Target>
      <Menu.Dropdown>
        <Menu.Item onClick={() => onChange('')}>All time</Menu.Item>
        <Menu.Divider />
        <Menu.Item onClick={() => onChange('today')}>Today</Menu.Item>
        <Menu.Item onClick={() => onChange('yesterday')}>Yesterday</Menu.Item>
        <Menu.Item onClick={() => onChange('last7days')}>Last 7 days</Menu.Item>
        <Menu.Item onClick={() => onChange('last30days')}>Last 30 days</Menu.Item>
        <Menu.Item onClick={() => onChange('lastMonth')}>Last month</Menu.Item>
        <Menu.Item onClick={() => onChange('lastQuarter')}>Last quarter</Menu.Item>
        <Menu.Item onClick={() => onChange('lastYear')}>Last year</Menu.Item>
        <Menu.Divider />
        <Menu.Item onClick={() => onChange('thisWeek')}>This week</Menu.Item>
        <Menu.Item onClick={() => onChange('thisMonth')}>This month</Menu.Item>
        <Menu.Item onClick={() => onChange('thisQuarter')}>This quarter</Menu.Item>
        <Menu.Item onClick={() => onChange('thisYear')}>This year</Menu.Item>
      </Menu.Dropdown>
    </Menu>
  )
}
