import { useMemo } from 'react'
import {
  Paper, Text, Group, Stack, Button, Select, TextInput, NumberInput,
  ActionIcon, Badge, Divider
} from '@mantine/core'
import { DatePickerInput } from '@mantine/dates'
import { IconPlus, IconTrash, IconFilter } from '@tabler/icons-react'
import { useColumns, useTables } from '../../api/hooks'
import type { Filter, FilterOperator } from '../../api/types'

interface FilterBuilderProps {
  datasourceId: string | null
  sourceTable: string | null
  filters: Filter[]
  onAddFilter: (filter: Filter) => void
  onRemoveFilter: (id: string) => void
  onUpdateFilter: (id: string, updates: Partial<Filter>) => void
  onClearFilters: () => void
}

const operatorLabels: Record<FilterOperator, string> = {
  'equals': 'is',
  'not-equals': 'is not',
  'contains': 'contains',
  'not-contains': 'does not contain',
  'starts-with': 'starts with',
  'ends-with': 'ends with',
  'is-null': 'is empty',
  'is-not-null': 'is not empty',
  'greater-than': '>',
  'less-than': '<',
  'greater-or-equal': '>=',
  'less-or-equal': '<=',
  'between': 'between',
  'is-in-previous': 'in previous',
  'is-in-next': 'in next',
  'is-in-current': 'in current',
}

const operatorsByType: Record<string, FilterOperator[]> = {
  string: ['equals', 'not-equals', 'contains', 'not-contains', 'starts-with', 'ends-with', 'is-null', 'is-not-null'],
  number: ['equals', 'not-equals', 'greater-than', 'less-than', 'greater-or-equal', 'less-or-equal', 'between', 'is-null', 'is-not-null'],
  boolean: ['equals', 'not-equals', 'is-null', 'is-not-null'],
  datetime: ['equals', 'not-equals', 'greater-than', 'less-than', 'between', 'is-in-previous', 'is-in-next', 'is-in-current', 'is-null', 'is-not-null'],
  date: ['equals', 'not-equals', 'greater-than', 'less-than', 'between', 'is-in-previous', 'is-in-next', 'is-in-current', 'is-null', 'is-not-null'],
}

export default function FilterBuilder({
  datasourceId,
  sourceTable,
  filters,
  onAddFilter,
  onRemoveFilter,
  onUpdateFilter,
  onClearFilters,
}: FilterBuilderProps) {
  const { data: tables } = useTables(datasourceId || '')
  const selectedTableId = useMemo(() => {
    const table = tables?.find(t => t.name === sourceTable || t.id === sourceTable)
    return table?.id || ''
  }, [tables, sourceTable])
  const { data: columns } = useColumns(selectedTableId)

  const columnOptions = useMemo(() => {
    return (columns || []).map(c => ({
      value: c.name,
      label: c.display_name || c.name,
      type: c.type,
    }))
  }, [columns])

  const getColumnType = (columnName: string) => {
    const col = columns?.find(c => c.name === columnName)
    return col?.type || 'string'
  }

  const handleAddFilter = () => {
    if (columnOptions.length > 0) {
      const defaultColumn = columnOptions[0]
      onAddFilter({
        id: Math.random().toString(36).substring(2, 9),
        column: defaultColumn.value,
        operator: 'equals',
        value: '',
      })
    }
  }

  if (!sourceTable) {
    return null
  }

  return (
    <Paper withBorder radius="md" p={0} style={{ overflow: 'hidden' }}>
      <Group justify="space-between" p="sm" bg="filter.0">
        <Group gap="xs">
          <IconFilter size={18} color="var(--mantine-color-filter-5)" />
          <Text fw={500} size="sm" c="filter.7">Filter</Text>
          {filters.length > 0 && (
            <Badge size="sm" variant="filled" color="filter">
              {filters.length}
            </Badge>
          )}
        </Group>
        <Group gap="xs">
          {filters.length > 0 && (
            <Button variant="subtle" size="xs" color="gray" onClick={onClearFilters}>
              Clear all
            </Button>
          )}
          <Button
            size="xs"
            variant="light"
            color="filter"
            leftSection={<IconPlus size={14} />}
            onClick={handleAddFilter}
          >
            Add filter
          </Button>
        </Group>
      </Group>

      {filters.length > 0 && (
        <>
          <Divider />
          <Stack gap="sm" p="sm">
            {filters.map((filter, index) => (
              <FilterRow
                key={filter.id}
                filter={filter}
                columns={columnOptions}
                columnType={getColumnType(filter.column)}
                onUpdate={(updates) => onUpdateFilter(filter.id, updates)}
                onRemove={() => onRemoveFilter(filter.id)}
                showAnd={index > 0}
              />
            ))}
          </Stack>
        </>
      )}
    </Paper>
  )
}

function FilterRow({
  filter,
  columns,
  columnType,
  onUpdate,
  onRemove,
  showAnd,
}: {
  filter: Filter
  columns: { value: string; label: string; type: string }[]
  columnType: string
  onUpdate: (updates: Partial<Filter>) => void
  onRemove: () => void
  showAnd: boolean
}) {
  const availableOperators = operatorsByType[columnType] || operatorsByType.string

  const operatorOptions = availableOperators.map(op => ({
    value: op,
    label: operatorLabels[op],
  }))

  const needsValue = !['is-null', 'is-not-null'].includes(filter.operator)
  const needsSecondValue = filter.operator === 'between'

  return (
    <Group gap="xs" align="flex-end">
      {showAnd && (
        <Badge size="sm" variant="outline" color="gray" style={{ minWidth: 40 }}>
          AND
        </Badge>
      )}

      <Select
        size="xs"
        data={columns}
        value={filter.column}
        onChange={(value) => onUpdate({ column: value || '' })}
        style={{ minWidth: 150 }}
        searchable
      />

      <Select
        size="xs"
        data={operatorOptions}
        value={filter.operator}
        onChange={(value) => onUpdate({ operator: value as FilterOperator })}
        style={{ minWidth: 120 }}
      />

      {needsValue && (
        <FilterValueInput
          type={columnType}
          operator={filter.operator}
          value={filter.value}
          onChange={(value) => onUpdate({ value })}
        />
      )}

      {needsSecondValue && (
        <>
          <Text size="xs" c="dimmed">and</Text>
          <FilterValueInput
            type={columnType}
            operator={filter.operator}
            value={(filter.value as any)?.end || ''}
            onChange={(value) => onUpdate({
              value: { start: (filter.value as any)?.start || '', end: value }
            })}
          />
        </>
      )}

      <ActionIcon variant="subtle" color="gray" size="sm" onClick={onRemove}>
        <IconTrash size={14} />
      </ActionIcon>
    </Group>
  )
}

function FilterValueInput({
  type,
  operator,
  value,
  onChange,
}: {
  type: string
  operator: FilterOperator
  value: any
  onChange: (value: any) => void
}) {
  // Time unit for relative date filters
  if (['is-in-previous', 'is-in-next', 'is-in-current'].includes(operator)) {
    return (
      <Group gap="xs">
        {operator !== 'is-in-current' && (
          <NumberInput
            size="xs"
            value={value?.count || 1}
            onChange={(v) => onChange({ ...value, count: v })}
            min={1}
            style={{ width: 60 }}
          />
        )}
        <Select
          size="xs"
          data={[
            { value: 'day', label: 'days' },
            { value: 'week', label: 'weeks' },
            { value: 'month', label: 'months' },
            { value: 'quarter', label: 'quarters' },
            { value: 'year', label: 'years' },
          ]}
          value={value?.unit || 'day'}
          onChange={(v) => onChange({ ...value, unit: v })}
          style={{ width: 100 }}
        />
      </Group>
    )
  }

  switch (type) {
    case 'number':
      return (
        <NumberInput
          size="xs"
          value={typeof value === 'number' ? value : ''}
          onChange={(v) => onChange(v)}
          style={{ width: 100 }}
        />
      )

    case 'boolean':
      return (
        <Select
          size="xs"
          data={[
            { value: 'true', label: 'True' },
            { value: 'false', label: 'False' },
          ]}
          value={String(value)}
          onChange={(v) => onChange(v === 'true')}
          style={{ width: 80 }}
        />
      )

    case 'datetime':
    case 'date':
      return (
        <DatePickerInput
          size="xs"
          value={value || null}
          onChange={(d) => onChange(d || null)}
          style={{ width: 140 }}
        />
      )

    default:
      return (
        <TextInput
          size="xs"
          value={value || ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder="Enter value..."
          style={{ width: 150 }}
        />
      )
  }
}

// Compact filter display
export function FilterBadges({
  filters,
  onRemove,
}: {
  filters: Filter[]
  onRemove: (id: string) => void
}) {
  if (filters.length === 0) return null

  return (
    <Group gap="xs">
      {filters.map(filter => (
        <Badge
          key={filter.id}
          variant="light"
          color="filter"
          rightSection={
            <ActionIcon
              size="xs"
              variant="transparent"
              color="filter"
              onClick={() => onRemove(filter.id)}
            >
              <IconTrash size={10} />
            </ActionIcon>
          }
        >
          {filter.column} {operatorLabels[filter.operator]} {formatFilterValue(filter.value, filter.operator)}
        </Badge>
      ))}
    </Group>
  )
}

function formatFilterValue(value: any, operator: FilterOperator): string {
  if (['is-null', 'is-not-null'].includes(operator)) return ''
  if (['is-in-previous', 'is-in-next', 'is-in-current'].includes(operator)) {
    return `${value?.count || ''} ${value?.unit || ''}`
  }
  if (typeof value === 'object' && value?.start !== undefined) {
    return `${value.start} - ${value.end}`
  }
  return String(value || '')
}
