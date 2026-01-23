import { useState, useMemo } from 'react'
import {
  Paper, Text, Group, Stack, UnstyledButton, Checkbox, Badge, TextInput,
  ActionIcon, Tooltip, Box, Divider, ScrollArea, Collapse
} from '@mantine/core'
import {
  IconSearch, IconColumns, IconHash, IconCalendar, IconToggleLeft,
  IconLetterCase, IconChevronDown, IconChevronRight, IconGripVertical,
  IconX
} from '@tabler/icons-react'
import { useColumns, useTables } from '../../api/hooks'
import type { Column, Table } from '../../api/types'

interface SelectedColumn {
  id: string
  table: string
  column: string
  alias?: string
}

interface ColumnSelectorProps {
  datasourceId: string | null
  sourceTable: string | null
  joinedTables?: string[]
  selectedColumns: SelectedColumn[]
  onToggleColumn: (column: SelectedColumn) => void
  onClearColumns: () => void
}

// Column type icons
const typeIcons: Record<string, typeof IconHash> = {
  number: IconHash,
  string: IconLetterCase,
  boolean: IconToggleLeft,
  datetime: IconCalendar,
  date: IconCalendar,
}

export default function ColumnSelector({
  datasourceId,
  sourceTable,
  joinedTables = [],
  selectedColumns,
  onToggleColumn,
  onClearColumns,
}: ColumnSelectorProps) {
  const [search, setSearch] = useState('')
  const [expandedTables, setExpandedTables] = useState<Record<string, boolean>>({
    [sourceTable || '']: true,
  })

  const { data: tables } = useTables(datasourceId || '')

  const allTables = useMemo(() => {
    if (!tables) return []
    const tableNames = [sourceTable, ...joinedTables].filter(Boolean) as string[]
    return tables.filter(t => tableNames.includes(t.id) || tableNames.includes(t.name))
  }, [tables, sourceTable, joinedTables])

  const toggleTable = (tableId: string) => {
    setExpandedTables(prev => ({
      ...prev,
      [tableId]: !prev[tableId],
    }))
  }

  const isColumnSelected = (table: string, column: string) => {
    return selectedColumns.some(c => c.table === table && c.column === column)
  }

  if (!sourceTable) {
    return (
      <Paper withBorder p="md" radius="md" bg="gray.0">
        <Group gap="sm">
          <IconColumns size={20} color="var(--mantine-color-gray-5)" />
          <Text c="dimmed" size="sm">Select a table to see columns</Text>
        </Group>
      </Paper>
    )
  }

  return (
    <Paper withBorder radius="md" p={0} style={{ overflow: 'hidden' }}>
      <Group justify="space-between" p="sm" bg="gray.0">
        <Group gap="xs">
          <IconColumns size={18} />
          <Text fw={500} size="sm">Columns</Text>
          {selectedColumns.length > 0 && (
            <Badge size="sm" variant="filled" color="brand">
              {selectedColumns.length}
            </Badge>
          )}
        </Group>
        {selectedColumns.length > 0 && (
          <Tooltip label="Clear all columns">
            <ActionIcon variant="subtle" size="sm" onClick={onClearColumns}>
              <IconX size={14} />
            </ActionIcon>
          </Tooltip>
        )}
      </Group>

      <Divider />

      <Box p="sm">
        <TextInput
          placeholder="Search columns..."
          leftSection={<IconSearch size={14} />}
          size="xs"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </Box>

      <ScrollArea h={300}>
        <Stack gap={0}>
          {allTables.map(table => (
            <TableColumns
              key={table.id}
              table={table}
              datasourceId={datasourceId!}
              search={search}
              expanded={expandedTables[table.id] ?? expandedTables[table.name] ?? true}
              onToggle={() => toggleTable(table.id)}
              isColumnSelected={(col) => isColumnSelected(table.name, col)}
              onToggleColumn={(col) => onToggleColumn({
                id: `${table.name}.${col}`,
                table: table.name,
                column: col,
              })}
            />
          ))}
        </Stack>
      </ScrollArea>
    </Paper>
  )
}

function TableColumns({
  table,
  datasourceId,
  search,
  expanded,
  onToggle,
  isColumnSelected,
  onToggleColumn,
}: {
  table: Table
  datasourceId: string
  search: string
  expanded: boolean
  onToggle: () => void
  isColumnSelected: (column: string) => boolean
  onToggleColumn: (column: string) => void
}) {
  const { data: columns, isLoading } = useColumns(datasourceId, table.id)

  const filteredColumns = useMemo(() => {
    if (!search.trim()) return columns || []
    const searchLower = search.toLowerCase()
    return (columns || []).filter(c =>
      c.name.toLowerCase().includes(searchLower) ||
      c.display_name.toLowerCase().includes(searchLower)
    )
  }, [columns, search])

  const selectedCount = useMemo(() => {
    return filteredColumns.filter(c => isColumnSelected(c.name)).length
  }, [filteredColumns, isColumnSelected])

  return (
    <Box>
      <UnstyledButton
        onClick={onToggle}
        style={{ display: 'block', width: '100%' }}
      >
        <Group
          gap="sm"
          px="sm"
          py="xs"
          bg={expanded ? 'gray.0' : 'transparent'}
          style={{ borderBottom: '1px solid var(--mantine-color-gray-2)' }}
        >
          {expanded ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
          <Text size="sm" fw={500}>{table.display_name || table.name}</Text>
          {selectedCount > 0 && (
            <Badge size="xs" variant="light" color="brand">
              {selectedCount}
            </Badge>
          )}
        </Group>
      </UnstyledButton>

      <Collapse in={expanded}>
        {isLoading ? (
          <Text size="sm" c="dimmed" p="sm">Loading columns...</Text>
        ) : filteredColumns.length === 0 ? (
          <Text size="sm" c="dimmed" p="sm">No columns found</Text>
        ) : (
          <Stack gap={0} p="xs">
            {filteredColumns.map(column => (
              <ColumnRow
                key={column.id}
                column={column}
                selected={isColumnSelected(column.name)}
                onToggle={() => onToggleColumn(column.name)}
              />
            ))}
          </Stack>
        )}
      </Collapse>
    </Box>
  )
}

function ColumnRow({
  column,
  selected,
  onToggle,
}: {
  column: Column
  selected: boolean
  onToggle: () => void
}) {
  const Icon = typeIcons[column.type] || IconLetterCase

  return (
    <UnstyledButton onClick={onToggle} style={{ display: 'block', width: '100%' }}>
      <Group
        gap="sm"
        px="sm"
        py={6}
        style={{
          borderRadius: 6,
          backgroundColor: selected ? 'var(--mantine-color-brand-0)' : 'transparent',
        }}
      >
        <Checkbox checked={selected} onChange={() => {}} size="xs" />
        <Icon size={14} color="var(--mantine-color-gray-6)" />
        <div style={{ flex: 1 }}>
          <Text size="sm">{column.display_name || column.name}</Text>
        </div>
        <Text size="xs" c="dimmed">{column.type}</Text>
        {column.semantic && (
          <Badge size="xs" variant="outline" color="gray">
            {column.semantic}
          </Badge>
        )}
      </Group>
    </UnstyledButton>
  )
}

// Selected columns list (draggable)
export function SelectedColumnsList({
  columns,
  onRemove,
}: {
  columns: SelectedColumn[]
  onRemove: (id: string) => void
}) {
  if (columns.length === 0) {
    return (
      <Paper withBorder p="md" radius="md" bg="gray.0" ta="center">
        <Text c="dimmed" size="sm">No columns selected</Text>
        <Text c="dimmed" size="xs">Click columns to add them</Text>
      </Paper>
    )
  }

  return (
    <Stack gap="xs">
      {columns.map((col) => (
        <Paper
          key={col.id}
          withBorder
          p="xs"
          radius="md"
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
          }}
        >
          <IconGripVertical size={14} color="var(--mantine-color-gray-5)" style={{ cursor: 'grab' }} />
          <div style={{ flex: 1 }}>
            <Text size="sm" fw={500}>{col.column}</Text>
            <Text size="xs" c="dimmed">{col.table}</Text>
          </div>
          <ActionIcon variant="subtle" size="sm" color="gray" onClick={() => onRemove(col.id)}>
            <IconX size={14} />
          </ActionIcon>
        </Paper>
      ))}
    </Stack>
  )
}
