import { useState } from 'react'
import {
  Menu, ActionIcon, Text, TextInput, Group, Stack, Button, Badge
} from '@mantine/core'
import {
  IconSortAscending, IconSortDescending, IconFilter, IconEyeOff,
  IconDotsVertical, IconCheck, IconArrowsSort
} from '@tabler/icons-react'
import type { ResultColumn } from '../../../api/types'
import type { SortState } from './types'

interface ColumnHeaderMenuProps {
  column: ResultColumn
  sortState: SortState
  onSort: (column: string, direction: 'asc' | 'desc') => void
  onHide: (column: string) => void
  onFilter?: (column: string, operator: string, value: any) => void
  activeFilter?: { operator: string; value: any }
}

export default function ColumnHeaderMenu({
  column,
  sortState,
  onSort,
  onHide,
  onFilter,
  activeFilter,
}: ColumnHeaderMenuProps) {
  const [opened, setOpened] = useState(false)
  const [filterValue, setFilterValue] = useState('')
  const [filterOperator, setFilterOperator] = useState('contains')

  const isSorted = sortState.column === column.name
  const isFiltered = !!activeFilter

  const handleApplyFilter = () => {
    if (onFilter && filterValue.trim()) {
      onFilter(column.name, filterOperator, filterValue)
      setOpened(false)
    }
  }

  const handleClearFilter = () => {
    if (onFilter) {
      onFilter(column.name, '', '')
      setFilterValue('')
    }
  }

  const getFilterOperators = () => {
    const type = column.type?.toLowerCase()
    if (type === 'number' || type === 'integer' || type === 'float') {
      return [
        { value: '=', label: 'Equals' },
        { value: '!=', label: 'Not equals' },
        { value: '>', label: 'Greater than' },
        { value: '>=', label: 'Greater or equal' },
        { value: '<', label: 'Less than' },
        { value: '<=', label: 'Less or equal' },
      ]
    }
    return [
      { value: 'contains', label: 'Contains' },
      { value: '=', label: 'Equals' },
      { value: '!=', label: 'Not equals' },
      { value: 'starts_with', label: 'Starts with' },
      { value: 'ends_with', label: 'Ends with' },
    ]
  }

  return (
    <Menu
      opened={opened}
      onChange={setOpened}
      position="bottom-start"
      withinPortal
      shadow="md"
      width={220}
    >
      <Menu.Target>
        <ActionIcon
          variant="subtle"
          size="xs"
          onClick={(e) => {
            e.stopPropagation()
            setOpened(!opened)
          }}
          style={{ opacity: opened || isSorted || isFiltered ? 1 : 0.3 }}
        >
          {isFiltered ? (
            <IconFilter size={14} color="var(--mantine-color-brand-6)" />
          ) : (
            <IconDotsVertical size={14} />
          )}
        </ActionIcon>
      </Menu.Target>

      <Menu.Dropdown onClick={(e) => e.stopPropagation()}>
        {/* Sort Options */}
        <Menu.Label>Sort</Menu.Label>
        <Menu.Item
          leftSection={<IconSortAscending size={14} />}
          rightSection={isSorted && sortState.direction === 'asc' ? <IconCheck size={14} /> : null}
          onClick={() => {
            onSort(column.name, 'asc')
            setOpened(false)
          }}
        >
          Sort A to Z
        </Menu.Item>
        <Menu.Item
          leftSection={<IconSortDescending size={14} />}
          rightSection={isSorted && sortState.direction === 'desc' ? <IconCheck size={14} /> : null}
          onClick={() => {
            onSort(column.name, 'desc')
            setOpened(false)
          }}
        >
          Sort Z to A
        </Menu.Item>
        {isSorted && (
          <Menu.Item
            leftSection={<IconArrowsSort size={14} />}
            onClick={() => {
              onSort('', 'asc')
              setOpened(false)
            }}
          >
            Clear sort
          </Menu.Item>
        )}

        <Menu.Divider />

        {/* Filter Options */}
        {onFilter && (
          <>
            <Menu.Label>
              <Group justify="space-between">
                <Text size="xs">Filter</Text>
                {isFiltered && (
                  <Badge size="xs" color="brand" variant="light">Active</Badge>
                )}
              </Group>
            </Menu.Label>
            <Stack gap="xs" p="xs">
              <select
                value={filterOperator}
                onChange={(e) => setFilterOperator(e.target.value)}
                style={{
                  width: '100%',
                  padding: '6px 8px',
                  borderRadius: 4,
                  border: '1px solid var(--mantine-color-gray-4)',
                  fontSize: 13,
                }}
              >
                {getFilterOperators().map(op => (
                  <option key={op.value} value={op.value}>{op.label}</option>
                ))}
              </select>
              <TextInput
                size="xs"
                placeholder="Filter value..."
                value={filterValue}
                onChange={(e) => setFilterValue(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleApplyFilter()
                }}
              />
              <Group gap="xs">
                <Button
                  size="xs"
                  variant="light"
                  onClick={handleClearFilter}
                  disabled={!isFiltered}
                  style={{ flex: 1 }}
                >
                  Clear
                </Button>
                <Button
                  size="xs"
                  onClick={handleApplyFilter}
                  disabled={!filterValue.trim()}
                  style={{ flex: 1 }}
                >
                  Apply
                </Button>
              </Group>
            </Stack>
            <Menu.Divider />
          </>
        )}

        {/* Column Options */}
        <Menu.Label>Column</Menu.Label>
        <Menu.Item
          leftSection={<IconEyeOff size={14} />}
          onClick={() => {
            onHide(column.name)
            setOpened(false)
          }}
        >
          Hide column
        </Menu.Item>
      </Menu.Dropdown>
    </Menu>
  )
}
