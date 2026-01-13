import { Table, TextInput, Group, ActionIcon, Menu, Text, Paper, Stack, ScrollArea } from '@mantine/core'
import { IconSearch, IconDotsVertical, IconChevronUp, IconChevronDown } from '@tabler/icons-react'
import { useState, useMemo, type ReactNode } from 'react'
import { EmptyState } from './EmptyState'
import { LoadingState } from './LoadingState'

export interface Column<T> {
  key: string
  label: string
  render?: (row: T) => ReactNode
  sortable?: boolean
  width?: number | string
}

export interface RowAction<T> {
  label: string
  icon?: ReactNode
  onClick: (row: T) => void
  color?: string
}

interface DataTableProps<T> {
  data: T[]
  columns: Column<T>[]
  searchable?: boolean
  searchPlaceholder?: string
  onRowClick?: (row: T) => void
  actions?: RowAction<T>[]
  emptyState?: {
    title: string
    description?: string
    action?: {
      label: string
      onClick: () => void
    }
  }
  loading?: boolean
  getRowKey: (row: T) => string
}

export function DataTable<T extends object>({
  data,
  columns,
  searchable = true,
  searchPlaceholder = 'Search...',
  onRowClick,
  actions,
  emptyState,
  loading,
  getRowKey,
}: DataTableProps<T>) {
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<string | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')

  const filteredData = useMemo(() => {
    let result = [...data]

    // Filter by search
    if (search) {
      const searchLower = search.toLowerCase()
      result = result.filter((row) =>
        columns.some((col) => {
          const value = (row as Record<string, unknown>)[col.key]
          if (value == null) return false
          return String(value).toLowerCase().includes(searchLower)
        })
      )
    }

    // Sort
    if (sortKey) {
      result.sort((a, b) => {
        const aVal = (a as Record<string, unknown>)[sortKey]
        const bVal = (b as Record<string, unknown>)[sortKey]
        if (aVal == null) return 1
        if (bVal == null) return -1
        const comparison = String(aVal).localeCompare(String(bVal), undefined, { numeric: true })
        return sortDirection === 'asc' ? comparison : -comparison
      })
    }

    return result
  }, [data, search, sortKey, sortDirection, columns])

  const handleSort = (key: string) => {
    if (sortKey === key) {
      setSortDirection((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDirection('asc')
    }
  }

  if (loading) {
    return <LoadingState />
  }

  if (data.length === 0 && emptyState) {
    return <EmptyState {...emptyState} />
  }

  return (
    <Stack gap="md">
      {searchable && (
        <TextInput
          placeholder={searchPlaceholder}
          leftSection={<IconSearch size={16} />}
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
          styles={{
            input: {
              backgroundColor: 'var(--mantine-color-dark-7)',
            },
          }}
        />
      )}

      <Paper withBorder radius="md" style={{ overflow: 'hidden' }}>
        <ScrollArea>
          <Table highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                {columns.map((col) => (
                  <Table.Th
                    key={col.key}
                    style={{
                      width: col.width,
                      cursor: col.sortable ? 'pointer' : 'default',
                    }}
                    onClick={() => col.sortable && handleSort(col.key)}
                  >
                    <Group gap={4} wrap="nowrap">
                      <Text size="sm" fw={600}>
                        {col.label}
                      </Text>
                      {col.sortable && sortKey === col.key && (
                        sortDirection === 'asc' ? (
                          <IconChevronUp size={14} />
                        ) : (
                          <IconChevronDown size={14} />
                        )
                      )}
                    </Group>
                  </Table.Th>
                ))}
                {actions && actions.length > 0 && (
                  <Table.Th style={{ width: 50 }}>
                    <Text size="sm" fw={600}>
                      Actions
                    </Text>
                  </Table.Th>
                )}
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filteredData.map((row) => (
                <Table.Tr
                  key={getRowKey(row)}
                  style={{ cursor: onRowClick ? 'pointer' : 'default' }}
                  onClick={() => onRowClick?.(row)}
                >
                  {columns.map((col) => (
                    <Table.Td key={col.key}>
                      {col.render ? col.render(row) : String((row as Record<string, unknown>)[col.key] ?? '-')}
                    </Table.Td>
                  ))}
                  {actions && actions.length > 0 && (
                    <Table.Td onClick={(e) => e.stopPropagation()}>
                      <Menu shadow="md" width={150} position="bottom-end">
                        <Menu.Target>
                          <ActionIcon variant="subtle" size="sm">
                            <IconDotsVertical size={16} />
                          </ActionIcon>
                        </Menu.Target>
                        <Menu.Dropdown>
                          {actions.map((action, idx) => (
                            <Menu.Item
                              key={idx}
                              leftSection={action.icon}
                              color={action.color}
                              onClick={() => action.onClick(row)}
                            >
                              {action.label}
                            </Menu.Item>
                          ))}
                        </Menu.Dropdown>
                      </Menu>
                    </Table.Td>
                  )}
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </ScrollArea>
      </Paper>

      {filteredData.length === 0 && data.length > 0 && (
        <Text size="sm" c="dimmed" ta="center" py="md">
          No results found for "{search}"
        </Text>
      )}
    </Stack>
  )
}
