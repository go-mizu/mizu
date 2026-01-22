import { useState, useMemo } from 'react'
import {
  Modal, TextInput, Stack, Group, Text, UnstyledButton, Badge, Loader,
  ScrollArea, Paper, Select
} from '@mantine/core'
import { IconSearch, IconTable, IconDatabase } from '@tabler/icons-react'
import { useTables } from '../../api/hooks'
import type { Table } from '../../api/types'

interface TablePickerProps {
  datasourceId: string | null
  value: string | null
  onChange: (tableId: string | null, tableName: string | null) => void
}

export default function TablePicker({ datasourceId, value, onChange }: TablePickerProps) {
  const [opened, setOpened] = useState(false)
  const [search, setSearch] = useState('')
  const { data: tables, isLoading } = useTables(datasourceId || '')

  const selectedTable = useMemo(() => {
    return tables?.find(t => t.id === value || t.name === value)
  }, [tables, value])

  const filteredTables = useMemo(() => {
    if (!search.trim()) return tables || []
    const searchLower = search.toLowerCase()
    return (tables || []).filter(t =>
      t.name.toLowerCase().includes(searchLower) ||
      t.display_name.toLowerCase().includes(searchLower) ||
      t.schema.toLowerCase().includes(searchLower)
    )
  }, [tables, search])

  // Group tables by schema
  const groupedTables = useMemo(() => {
    const groups: Record<string, Table[]> = {}
    filteredTables.forEach(table => {
      const schema = table.schema || 'default'
      if (!groups[schema]) groups[schema] = []
      groups[schema].push(table)
    })
    return groups
  }, [filteredTables])

  const handleSelect = (table: Table) => {
    onChange(table.id, table.name)
    setOpened(false)
    setSearch('')
  }

  if (!datasourceId) {
    return (
      <Paper withBorder p="md" radius="md" bg="gray.0">
        <Group gap="sm">
          <IconDatabase size={20} color="var(--mantine-color-gray-5)" />
          <Text c="dimmed" size="sm">Select a data source first</Text>
        </Group>
      </Paper>
    )
  }

  return (
    <>
      <UnstyledButton onClick={() => setOpened(true)} style={{ width: '100%' }} data-testid="table-picker">
        <Paper withBorder p="md" radius="md" style={{ cursor: 'pointer' }}>
          {selectedTable ? (
            <Group gap="sm">
              <IconTable size={20} color="var(--mantine-color-brand-5)" />
              <div>
                <Text fw={500}>{selectedTable.display_name || selectedTable.name}</Text>
                <Text size="xs" c="dimmed">{selectedTable.schema}.{selectedTable.name}</Text>
              </div>
              {selectedTable.row_count > 0 && (
                <Badge size="sm" variant="light" color="gray" ml="auto">
                  {selectedTable.row_count.toLocaleString()} rows
                </Badge>
              )}
            </Group>
          ) : (
            <Group gap="sm">
              <IconTable size={20} color="var(--mantine-color-gray-5)" />
              <Text c="dimmed">Select a table</Text>
            </Group>
          )}
        </Paper>
      </UnstyledButton>

      <Modal
        opened={opened}
        onClose={() => setOpened(false)}
        title="Select a Table"
        size="lg"
        data-testid="modal-table-picker"
      >
        <Stack gap="md">
          <TextInput
            placeholder="Search tables..."
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            autoFocus
          />

          {isLoading ? (
            <Group justify="center" py="xl">
              <Loader size="sm" />
              <Text c="dimmed">Loading tables...</Text>
            </Group>
          ) : filteredTables.length === 0 ? (
            <Paper p="xl" bg="gray.0" ta="center" radius="md">
              <Text c="dimmed">
                {search ? 'No tables found matching your search' : 'No tables available'}
              </Text>
            </Paper>
          ) : (
            <ScrollArea h={400}>
              <Stack gap="xs">
                {Object.entries(groupedTables).map(([schema, schemaTables]) => (
                  <div key={schema}>
                    <Text size="xs" fw={600} c="dimmed" px="sm" py="xs">
                      {schema}
                    </Text>
                    {schemaTables.map(table => (
                      <UnstyledButton
                        key={table.id}
                        onClick={() => handleSelect(table)}
                        style={{ display: 'block', width: '100%' }}
                      >
                        <Paper
                          p="sm"
                          radius="md"
                          style={{
                            backgroundColor: value === table.id || value === table.name
                              ? 'var(--mantine-color-brand-0)'
                              : 'transparent',
                            cursor: 'pointer',
                          }}
                        >
                          <Group justify="space-between">
                            <Group gap="sm">
                              <IconTable size={18} color="var(--mantine-color-gray-6)" />
                              <div>
                                <Text size="sm" fw={500}>
                                  {table.display_name || table.name}
                                </Text>
                                {table.description && (
                                  <Text size="xs" c="dimmed" lineClamp={1}>
                                    {table.description}
                                  </Text>
                                )}
                              </div>
                            </Group>
                            {table.row_count > 0 && (
                              <Badge size="xs" variant="light" color="gray">
                                {table.row_count.toLocaleString()} rows
                              </Badge>
                            )}
                          </Group>
                        </Paper>
                      </UnstyledButton>
                    ))}
                  </div>
                ))}
              </Stack>
            </ScrollArea>
          )}
        </Stack>
      </Modal>
    </>
  )
}

// Inline table selector (for compact view)
export function TableSelect({
  datasourceId,
  value,
  onChange,
}: {
  datasourceId: string | null
  value: string | null
  onChange: (value: string | null) => void
}) {
  const { data: tables, isLoading } = useTables(datasourceId || '')

  const options = (tables || []).map(t => ({
    value: t.id,
    label: t.display_name || t.name,
    schema: t.schema,
  }))

  return (
    <Select
      placeholder="Select a table"
      data={options}
      value={value}
      onChange={onChange}
      disabled={!datasourceId || isLoading}
      searchable
      leftSection={<IconTable size={16} />}
    />
  )
}
