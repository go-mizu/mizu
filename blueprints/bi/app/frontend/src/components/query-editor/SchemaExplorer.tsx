import { useState, useMemo } from 'react'
import {
  Stack, Text, TextInput, ScrollArea, UnstyledButton, Group, Box,
  Collapse, Skeleton, Badge
} from '@mantine/core'
import {
  IconTable, IconChevronRight, IconChevronDown, IconSearch,
  IconHash, IconAbc, IconCalendar, IconToggleLeft, IconBraces
} from '@tabler/icons-react'
import { useTables, useColumns } from '../../api/hooks'
import type { Table, Column } from '../../api/types'

interface SchemaExplorerProps {
  datasourceId: string | null
  onInsertText: (text: string) => void
}

// Map column types to icons
const typeIcons: Record<string, typeof IconAbc> = {
  string: IconAbc,
  number: IconHash,
  boolean: IconToggleLeft,
  datetime: IconCalendar,
  date: IconCalendar,
  json: IconBraces,
}

export default function SchemaExplorer({ datasourceId, onInsertText }: SchemaExplorerProps) {
  const [search, setSearch] = useState('')
  const [expandedTables, setExpandedTables] = useState<Set<string>>(new Set())

  const { data: tables, isLoading: tablesLoading } = useTables(datasourceId || '')

  const filteredTables = useMemo(() => {
    if (!tables) return []
    if (!search) return tables
    const lowerSearch = search.toLowerCase()
    return tables.filter(t =>
      t.name.toLowerCase().includes(lowerSearch) ||
      (t.display_name && t.display_name.toLowerCase().includes(lowerSearch))
    )
  }, [tables, search])

  const toggleTable = (tableId: string) => {
    setExpandedTables(prev => {
      const next = new Set(prev)
      if (next.has(tableId)) {
        next.delete(tableId)
      } else {
        next.add(tableId)
      }
      return next
    })
  }

  const handleDoubleClick = (text: string) => {
    onInsertText(text)
  }

  if (!datasourceId) {
    return (
      <Box p="md">
        <Text size="sm" c="dimmed">Select a data source to explore schema</Text>
      </Box>
    )
  }

  return (
    <Stack gap={0} h="100%">
      <Box p="xs" style={{ borderBottom: '1px solid var(--mantine-color-gray-2)' }}>
        <TextInput
          size="xs"
          placeholder="Search tables..."
          leftSection={<IconSearch size={14} />}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </Box>

      <ScrollArea flex={1} p="xs">
        {tablesLoading ? (
          <Stack gap="xs">
            {[1, 2, 3, 4, 5].map(i => (
              <Skeleton key={i} height={28} />
            ))}
          </Stack>
        ) : filteredTables.length === 0 ? (
          <Text size="sm" c="dimmed" ta="center" py="md">
            {search ? 'No tables match your search' : 'No tables found'}
          </Text>
        ) : (
          <Stack gap={2}>
            {filteredTables.map(table => (
              <TableItem
                key={table.id}
                table={table}
                datasourceId={datasourceId!}
                expanded={expandedTables.has(table.id)}
                onToggle={() => toggleTable(table.id)}
                onDoubleClick={handleDoubleClick}
              />
            ))}
          </Stack>
        )}
      </ScrollArea>

      <Box p="xs" style={{ borderTop: '1px solid var(--mantine-color-gray-2)' }}>
        <Text size="xs" c="dimmed">
          {filteredTables.length} table{filteredTables.length !== 1 ? 's' : ''}
        </Text>
      </Box>
    </Stack>
  )
}

interface TableItemProps {
  table: Table
  datasourceId: string
  expanded: boolean
  onToggle: () => void
  onDoubleClick: (text: string) => void
}

function TableItem({ table, datasourceId, expanded, onToggle, onDoubleClick }: TableItemProps) {
  const { data: columns, isLoading } = useColumns(expanded ? datasourceId : '', expanded ? table.id : '')

  return (
    <Box>
      <UnstyledButton
        onClick={onToggle}
        onDoubleClick={() => onDoubleClick(table.name)}
        w="100%"
        p={6}
        style={{
          borderRadius: 4,
          backgroundColor: expanded ? 'var(--mantine-color-blue-0)' : 'transparent',
        }}
        styles={{
          root: {
            '&:hover': {
              backgroundColor: expanded
                ? 'var(--mantine-color-blue-1)'
                : 'var(--mantine-color-gray-0)',
            },
          },
        }}
      >
        <Group gap="xs" wrap="nowrap">
          {expanded ? (
            <IconChevronDown size={14} color="var(--mantine-color-gray-6)" />
          ) : (
            <IconChevronRight size={14} color="var(--mantine-color-gray-6)" />
          )}
          <IconTable size={16} color="var(--mantine-color-blue-6)" />
          <Text size="sm" fw={500} truncate flex={1}>
            {table.display_name || table.name}
          </Text>
          {table.row_count !== undefined && (
            <Badge size="xs" variant="light" color="gray">
              {formatRowCount(table.row_count)}
            </Badge>
          )}
        </Group>
      </UnstyledButton>

      <Collapse in={expanded}>
        <Box pl={32} pr={8} pb={4}>
          {isLoading ? (
            <Stack gap={4} py={4}>
              {[1, 2, 3].map(i => (
                <Skeleton key={i} height={24} />
              ))}
            </Stack>
          ) : columns && columns.length > 0 ? (
            <Stack gap={2}>
              {columns.map(col => (
                <ColumnItem
                  key={col.id}
                  column={col}
                  tableName={table.name}
                  onDoubleClick={onDoubleClick}
                />
              ))}
            </Stack>
          ) : (
            <Text size="xs" c="dimmed" py={4}>
              No columns found
            </Text>
          )}
        </Box>
      </Collapse>
    </Box>
  )
}

interface ColumnItemProps {
  column: Column
  tableName: string
  onDoubleClick: (text: string) => void
}

function ColumnItem({ column, tableName, onDoubleClick }: ColumnItemProps) {
  const TypeIcon = typeIcons[column.mapped_type || 'string'] || IconAbc
  const isPrimaryKey = column.semantic === 'type/PK'
  const isForeignKey = column.semantic === 'type/FK'

  return (
    <UnstyledButton
      onDoubleClick={() => onDoubleClick(column.name)}
      w="100%"
      p={4}
      style={{ borderRadius: 4 }}
      styles={{
        root: {
          '&:hover': {
            backgroundColor: 'var(--mantine-color-gray-0)',
          },
        },
      }}
      title={`Double-click to insert "${tableName}.${column.name}"`}
    >
      <Group gap="xs" wrap="nowrap">
        <TypeIcon
          size={14}
          color={
            isPrimaryKey ? 'var(--mantine-color-yellow-6)' :
            isForeignKey ? 'var(--mantine-color-grape-6)' :
            'var(--mantine-color-gray-5)'
          }
        />
        <Text size="xs" truncate flex={1}>
          {column.display_name || column.name}
        </Text>
        <Text size="xs" c="dimmed">
          {column.mapped_type || 'unknown'}
        </Text>
        {isPrimaryKey && (
          <Badge size="xs" variant="light" color="yellow">
            PK
          </Badge>
        )}
        {isForeignKey && (
          <Badge size="xs" variant="light" color="grape">
            FK
          </Badge>
        )}
      </Group>
    </UnstyledButton>
  )
}

function formatRowCount(count: number): string {
  if (count >= 1000000) {
    return `${(count / 1000000).toFixed(1)}M`
  }
  if (count >= 1000) {
    return `${(count / 1000).toFixed(1)}K`
  }
  return count.toString()
}
