import {
  Table,
  TextInput,
  Group,
  ActionIcon,
  Menu,
  Text,
  Paper,
  Stack,
  ScrollArea,
  Checkbox,
  Select,
  Pagination,
  Button,
  Tooltip,
} from '@mantine/core'
import {
  IconSearch,
  IconDotsVertical,
  IconChevronUp,
  IconChevronDown,
  IconDownload,
  IconRefresh,
} from '@tabler/icons-react'
import { useState, useMemo, type ReactNode } from 'react'
import { EmptyState } from './EmptyState'
import { LoadingState } from './LoadingState'

export interface Column<T> {
  key: string
  label: string
  render?: (row: T) => ReactNode
  sortable?: boolean
  width?: number | string
  hidden?: boolean
}

export interface RowAction<T> {
  label: string
  icon?: ReactNode
  onClick: (row: T) => void
  color?: string
  disabled?: (row: T) => boolean
}

export interface BulkAction<T> {
  label: string
  icon?: ReactNode
  onClick: (rows: T[]) => void
  color?: string
}

interface PaginationConfig {
  page: number
  pageSize: number
  total: number
  onPageChange: (page: number) => void
  onPageSizeChange?: (size: number) => void
  pageSizeOptions?: number[]
}

interface DataTableProps<T> {
  data: T[]
  columns: Column<T>[]
  searchable?: boolean
  searchPlaceholder?: string
  onRowClick?: (row: T) => void
  actions?: RowAction<T>[]
  bulkActions?: BulkAction<T>[]
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
  /** Pagination config - if provided, enables pagination */
  pagination?: PaginationConfig
  /** Enable export functionality */
  exportable?: boolean
  onExport?: () => void
  /** Enable refresh */
  onRefresh?: () => void
  /** Custom filter component */
  filterComponent?: ReactNode
  /** Striped rows */
  striped?: boolean
  /** Sticky header */
  stickyHeader?: boolean
}

export function DataTable<T extends object>({
  data,
  columns,
  searchable = true,
  searchPlaceholder = 'Search...',
  onRowClick,
  actions,
  bulkActions,
  emptyState,
  loading,
  getRowKey,
  pagination,
  exportable,
  onExport,
  onRefresh,
  filterComponent,
  striped = false,
  stickyHeader = false,
}: DataTableProps<T>) {
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<string | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set())

  const visibleColumns = columns.filter((col) => !col.hidden)

  const filteredData = useMemo(() => {
    let result = [...(data ?? [])]

    // Filter by search
    if (search) {
      const searchLower = search.toLowerCase()
      result = result.filter((row) =>
        visibleColumns.some((col) => {
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
  }, [data, search, sortKey, sortDirection, visibleColumns])

  // Apply client-side pagination if no server-side pagination
  const paginatedData = useMemo(() => {
    if (pagination) {
      const start = (pagination.page - 1) * pagination.pageSize
      return filteredData.slice(start, start + pagination.pageSize)
    }
    return filteredData
  }, [filteredData, pagination])

  const handleSort = (key: string) => {
    if (sortKey === key) {
      setSortDirection((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDirection('asc')
    }
  }

  const toggleRowSelection = (key: string) => {
    setSelectedRows((prev) => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }

  const toggleAllRows = () => {
    if (selectedRows.size === paginatedData.length) {
      setSelectedRows(new Set())
    } else {
      setSelectedRows(new Set(paginatedData.map(getRowKey)))
    }
  }

  const selectedData = paginatedData.filter((row) => selectedRows.has(getRowKey(row)))
  const hasSelection = bulkActions && selectedRows.size > 0

  if (loading) {
    return <LoadingState />
  }

  if ((!data || data.length === 0) && emptyState) {
    return <EmptyState {...emptyState} />
  }

  const totalPages = pagination
    ? Math.ceil((pagination.total || filteredData.length) / pagination.pageSize)
    : 1

  return (
    <Stack gap="md">
      {/* Toolbar */}
      <Group justify="space-between" wrap="wrap">
        <Group gap="sm">
          {searchable && (
            <TextInput
              placeholder={searchPlaceholder}
              leftSection={<IconSearch size={16} />}
              value={search}
              onChange={(e) => setSearch(e.currentTarget.value)}
              w={250}
              styles={{
                input: {
                  backgroundColor: 'var(--mantine-color-dark-7)',
                },
              }}
            />
          )}
          {filterComponent}
        </Group>

        <Group gap="xs">
          {onRefresh && (
            <Tooltip label="Refresh">
              <ActionIcon variant="subtle" onClick={onRefresh}>
                <IconRefresh size={18} />
              </ActionIcon>
            </Tooltip>
          )}
          {exportable && onExport && (
            <Tooltip label="Export">
              <ActionIcon variant="subtle" onClick={onExport}>
                <IconDownload size={18} />
              </ActionIcon>
            </Tooltip>
          )}
        </Group>
      </Group>

      {/* Bulk Actions Bar */}
      {hasSelection && (
        <Paper p="xs" radius="md" bg="orange.9">
          <Group justify="space-between">
            <Text size="sm" fw={500}>
              {selectedRows.size} item{selectedRows.size !== 1 ? 's' : ''} selected
            </Text>
            <Group gap="xs">
              {bulkActions?.map((action, idx) => (
                <Button
                  key={idx}
                  size="xs"
                  variant="subtle"
                  color={action.color || 'gray'}
                  leftSection={action.icon}
                  onClick={() => action.onClick(selectedData)}
                >
                  {action.label}
                </Button>
              ))}
              <Button size="xs" variant="subtle" onClick={() => setSelectedRows(new Set())}>
                Clear
              </Button>
            </Group>
          </Group>
        </Paper>
      )}

      {/* Table */}
      <Paper withBorder radius="md" style={{ overflow: 'hidden' }}>
        <ScrollArea>
          <Table highlightOnHover striped={striped} stickyHeader={stickyHeader}>
            <Table.Thead>
              <Table.Tr>
                {bulkActions && (
                  <Table.Th style={{ width: 40 }}>
                    <Checkbox
                      checked={selectedRows.size === paginatedData.length && paginatedData.length > 0}
                      indeterminate={selectedRows.size > 0 && selectedRows.size < paginatedData.length}
                      onChange={toggleAllRows}
                      aria-label="Select all"
                    />
                  </Table.Th>
                )}
                {visibleColumns.map((col) => (
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
                  <Table.Th style={{ width: 50 }} />
                )}
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {paginatedData.map((row) => {
                const rowKey = getRowKey(row)
                const isSelected = selectedRows.has(rowKey)

                return (
                  <Table.Tr
                    key={rowKey}
                    style={{
                      cursor: onRowClick ? 'pointer' : 'default',
                      backgroundColor: isSelected ? 'var(--mantine-color-orange-9)' : undefined,
                    }}
                    onClick={() => onRowClick?.(row)}
                  >
                    {bulkActions && (
                      <Table.Td onClick={(e) => e.stopPropagation()}>
                        <Checkbox
                          checked={isSelected}
                          onChange={() => toggleRowSelection(rowKey)}
                          aria-label={`Select row ${rowKey}`}
                        />
                      </Table.Td>
                    )}
                    {visibleColumns.map((col) => (
                      <Table.Td key={col.key}>
                        {col.render
                          ? col.render(row)
                          : String((row as Record<string, unknown>)[col.key] ?? '-')}
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
                                disabled={action.disabled?.(row)}
                              >
                                {action.label}
                              </Menu.Item>
                            ))}
                          </Menu.Dropdown>
                        </Menu>
                      </Table.Td>
                    )}
                  </Table.Tr>
                )
              })}
            </Table.Tbody>
          </Table>
        </ScrollArea>
      </Paper>

      {/* No Search Results */}
      {filteredData.length === 0 && data && data.length > 0 && (
        <Text size="sm" c="dimmed" ta="center" py="md">
          No results found for "{search}"
        </Text>
      )}

      {/* Pagination */}
      {pagination && totalPages > 1 && (
        <Group justify="space-between">
          <Group gap="xs">
            <Text size="sm" c="dimmed">
              Showing {((pagination.page - 1) * pagination.pageSize) + 1}-
              {Math.min(pagination.page * pagination.pageSize, pagination.total || filteredData.length)} of{' '}
              {pagination.total || filteredData.length}
            </Text>
            {pagination.onPageSizeChange && (
              <Select
                size="xs"
                w={80}
                value={String(pagination.pageSize)}
                onChange={(v) => v && pagination.onPageSizeChange?.(parseInt(v))}
                data={(pagination.pageSizeOptions || [10, 25, 50, 100]).map((n) => ({
                  value: String(n),
                  label: String(n),
                }))}
              />
            )}
          </Group>
          <Pagination
            total={totalPages}
            value={pagination.page}
            onChange={pagination.onPageChange}
            size="sm"
          />
        </Group>
      )}
    </Stack>
  )
}
