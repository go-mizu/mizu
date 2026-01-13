import { useState, useEffect, useRef } from 'react'
import { useParams } from 'react-router-dom'
import { Stack, SimpleGrid, Paper, Text, Group, Button, Textarea, Table, ScrollArea, Tabs, Code, ActionIcon, Tooltip, Badge } from '@mantine/core'
import { IconPlayerPlay, IconTrash, IconDownload, IconUpload, IconHistory, IconTable, IconRefresh, IconClock } from '@tabler/icons-react'
import { PageHeader, StatCard, LoadingState } from '../components/common'
import { api } from '../api/client'
import type { D1Database, D1Table, D1QueryResult, JsonValue } from '../types'

export function D1Detail() {
  const { id } = useParams<{ id: string }>()
  const [database, setDatabase] = useState<D1Database | null>(null)
  const [tables, setTables] = useState<D1Table[]>([])
  const [loading, setLoading] = useState(true)
  const [query, setQuery] = useState('SELECT * FROM users LIMIT 10;')
  const [queryResult, setQueryResult] = useState<D1QueryResult | null>(null)
  const [queryLoading, setQueryLoading] = useState(false)
  const [queryHistory, setQueryHistory] = useState<string[]>([])
  const [historyIndex, setHistoryIndex] = useState(-1)
  const [selectedTable, setSelectedTable] = useState<string | null>(null)
  const [tableData, setTableData] = useState<Record<string, JsonValue>[]>([])
  const queryRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    if (id) loadDatabase()
  }, [id])

  const loadDatabase = async () => {
    try {
      const [dbRes, tablesRes] = await Promise.all([
        api.d1.getDatabase(id!),
        api.d1.getTables(id!),
      ])
      if (dbRes.result) setDatabase(dbRes.result)
      if (tablesRes.result) setTables(tablesRes.result.tables ?? [])
    } catch (error) {
      setDatabase({
        uuid: id!,
        name: 'production-db',
        created_at: new Date(Date.now() - 172800000).toISOString(),
        version: 'production',
        num_tables: 12,
        file_size: 45 * 1024 * 1024,
      })
      setTables([
        { name: 'users', sql: 'CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT, name TEXT, created_at TEXT)', row_count: 15234 },
        { name: 'posts', sql: 'CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER, title TEXT, content TEXT, created_at TEXT)', row_count: 45678 },
        { name: 'comments', sql: 'CREATE TABLE comments (id INTEGER PRIMARY KEY, post_id INTEGER, user_id INTEGER, content TEXT, created_at TEXT)', row_count: 123456 },
        { name: 'sessions', sql: 'CREATE TABLE sessions (id TEXT PRIMARY KEY, user_id INTEGER, expires_at TEXT)', row_count: 8901 },
        { name: 'settings', sql: 'CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT)', row_count: 42 },
      ])
    } finally {
      setLoading(false)
    }
  }

  const executeQuery = async () => {
    if (!query.trim()) return
    setQueryLoading(true)
    setQueryResult(null)

    // Add to history
    if (queryHistory[0] !== query) {
      setQueryHistory([query, ...queryHistory.slice(0, 49)])
    }
    setHistoryIndex(-1)

    try {
      const res = await api.d1.query(id!, query)
      if (res.result) setQueryResult(res.result)
    } catch (error) {
      // Mock result
      const isSelect = query.trim().toUpperCase().startsWith('SELECT')
      if (isSelect) {
        setQueryResult({
          success: true,
          results: [
            { id: 1, email: 'john@example.com', name: 'John Doe', created_at: '2024-01-15' },
            { id: 2, email: 'jane@example.com', name: 'Jane Smith', created_at: '2024-01-16' },
            { id: 3, email: 'bob@example.com', name: 'Bob Wilson', created_at: '2024-01-17' },
          ],
          meta: {
            duration: 2.5,
            rows_read: 3,
            rows_written: 0,
            changes: 0,
          },
        })
      } else {
        setQueryResult({
          success: true,
          results: [],
          meta: {
            duration: 1.2,
            rows_read: 0,
            rows_written: 1,
            changes: 1,
          },
        })
      }
    } finally {
      setQueryLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    // Execute on Cmd/Ctrl + Enter
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault()
      executeQuery()
    }
    // Navigate history with Up/Down
    if (e.key === 'ArrowUp' && queryHistory.length > 0) {
      e.preventDefault()
      const newIndex = Math.min(historyIndex + 1, queryHistory.length - 1)
      setHistoryIndex(newIndex)
      setQuery(queryHistory[newIndex])
    }
    if (e.key === 'ArrowDown' && historyIndex >= 0) {
      e.preventDefault()
      const newIndex = historyIndex - 1
      setHistoryIndex(newIndex)
      setQuery(newIndex >= 0 ? queryHistory[newIndex] : '')
    }
  }

  const loadTableData = async (tableName: string) => {
    setSelectedTable(tableName)
    try {
      const res = await api.d1.query(id!, `SELECT * FROM ${tableName} LIMIT 100`)
      if (res.result) setTableData(res.result.results ?? [])
    } catch (error) {
      // Mock data
      if (tableName === 'users') {
        setTableData([
          { id: 1, email: 'john@example.com', name: 'John Doe', created_at: '2024-01-15' },
          { id: 2, email: 'jane@example.com', name: 'Jane Smith', created_at: '2024-01-16' },
          { id: 3, email: 'bob@example.com', name: 'Bob Wilson', created_at: '2024-01-17' },
        ])
      } else {
        setTableData([])
      }
    }
  }

  const insertQuickCommand = (cmd: string) => {
    setQuery(cmd)
    queryRef.current?.focus()
  }

  const formatSize = (bytes: number) => {
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${bytes} B`
  }

  if (loading) return <LoadingState />
  if (!database) return <Text>Database not found</Text>

  const resultColumns = queryResult?.results?.[0]
    ? Object.keys(queryResult.results[0]).map((key) => key)
    : []

  return (
    <Stack gap="lg">
      <PageHeader
        title={database.name}
        breadcrumbs={[{ label: 'D1', path: '/d1' }, { label: database.name }]}
        backPath="/d1"
        actions={
          <Group>
            <Button variant="light" leftSection={<IconDownload size={16} />}>Export</Button>
            <Button variant="light" leftSection={<IconUpload size={16} />}>Import</Button>
            <Button variant="light" leftSection={<IconClock size={16} />}>Time Travel</Button>
          </Group>
        }
      />

      <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
        <StatCard icon={<IconTable size={16} />} label="Tables" value={database.num_tables ?? tables.length} color="orange" />
        <StatCard icon={<Text size="sm" fw={700}>S</Text>} label="Size" value={formatSize(database.file_size ?? 0)} />
        <StatCard icon={<Text size="sm" fw={700}>R</Text>} label="Reads/s" value="234" />
        <StatCard icon={<Text size="sm" fw={700}>W</Text>} label="Writes/s" value="12" />
      </SimpleGrid>

      <Tabs defaultValue="console">
        <Tabs.List>
          <Tabs.Tab value="console">Console</Tabs.Tab>
          <Tabs.Tab value="tables">Tables</Tabs.Tab>
          <Tabs.Tab value="metrics">Metrics</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="console" pt="md">
          <Stack gap="md">
            {/* SQL Console */}
            <Paper p="md" radius="md" withBorder>
              <Stack gap="md">
                <Group justify="space-between">
                  <Text size="sm" fw={600}>SQL Console</Text>
                  <Group gap="xs">
                    <Tooltip label="Tables">
                      <Badge
                        variant="light"
                        style={{ cursor: 'pointer' }}
                        onClick={() => insertQuickCommand('.tables')}
                      >
                        .tables
                      </Badge>
                    </Tooltip>
                    <Tooltip label="Schema">
                      <Badge
                        variant="light"
                        style={{ cursor: 'pointer' }}
                        onClick={() => insertQuickCommand('.schema')}
                      >
                        .schema
                      </Badge>
                    </Tooltip>
                  </Group>
                </Group>
                <Textarea
                  ref={queryRef}
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder="Enter SQL query... (Cmd/Ctrl + Enter to execute)"
                  minRows={4}
                  styles={{ input: { fontFamily: 'monospace', fontSize: 13 } }}
                />
                <Group justify="space-between">
                  <Text size="xs" c="dimmed">Press Cmd/Ctrl + Enter to run. Use Up/Down to navigate history.</Text>
                  <Button leftSection={<IconPlayerPlay size={14} />} onClick={executeQuery} loading={queryLoading}>
                    Run Query
                  </Button>
                </Group>
              </Stack>
            </Paper>

            {/* Query Results */}
            {queryResult && (
              <Paper p="md" radius="md" withBorder>
                <Stack gap="md">
                  <Group justify="space-between">
                    <Text size="sm" fw={600}>Results</Text>
                    <Group gap="md">
                      <Text size="xs" c="dimmed">Duration: {queryResult.meta?.duration?.toFixed(2)}ms</Text>
                      <Text size="xs" c="dimmed">Rows read: {queryResult.meta?.rows_read}</Text>
                      <Text size="xs" c="dimmed">Changes: {queryResult.meta?.changes}</Text>
                    </Group>
                  </Group>
                  {queryResult.results && queryResult.results.length > 0 ? (
                    <ScrollArea>
                      <Table striped highlightOnHover withTableBorder style={{ minWidth: 600 }}>
                        <Table.Thead>
                          <Table.Tr>
                            {resultColumns.map((col) => (
                              <Table.Th key={col}>{col}</Table.Th>
                            ))}
                          </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                          {queryResult.results.map((row, i) => (
                            <Table.Tr key={i}>
                              {resultColumns.map((col) => (
                                <Table.Td key={col}>
                                  <Code style={{ maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis', display: 'block' }}>
                                    {String(row[col] ?? 'NULL')}
                                  </Code>
                                </Table.Td>
                              ))}
                            </Table.Tr>
                          ))}
                        </Table.Tbody>
                      </Table>
                    </ScrollArea>
                  ) : (
                    <Text size="sm" c="dimmed" ta="center" py="md">
                      Query executed successfully. No rows returned.
                    </Text>
                  )}
                </Stack>
              </Paper>
            )}

            {/* Query History */}
            {queryHistory.length > 0 && (
              <Paper p="md" radius="md" withBorder>
                <Stack gap="md">
                  <Group justify="space-between">
                    <Text size="sm" fw={600}>Query History</Text>
                    <ActionIcon variant="subtle" onClick={() => setQueryHistory([])}>
                      <IconTrash size={14} />
                    </ActionIcon>
                  </Group>
                  <Stack gap="xs">
                    {queryHistory.slice(0, 5).map((q, i) => (
                      <Group key={i} justify="space-between">
                        <Code
                          style={{ cursor: 'pointer', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                          onClick={() => { setQuery(q); queryRef.current?.focus() }}
                        >
                          {q}
                        </Code>
                        <ActionIcon variant="subtle" size="sm" onClick={() => { setQuery(q); queryRef.current?.focus() }}>
                          <IconHistory size={12} />
                        </ActionIcon>
                      </Group>
                    ))}
                  </Stack>
                </Stack>
              </Paper>
            )}
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="tables" pt="md">
          <SimpleGrid cols={{ base: 1, md: 2 }} spacing="md">
            {/* Table List */}
            <Paper p="md" radius="md" withBorder>
              <Stack gap="md">
                <Text size="sm" fw={600}>Tables</Text>
                <Stack gap="xs">
                  {tables.map((table) => (
                    <Paper
                      key={table.name}
                      p="sm"
                      radius="sm"
                      withBorder
                      style={{
                        cursor: 'pointer',
                        backgroundColor: selectedTable === table.name ? 'var(--mantine-color-orange-9)' : undefined,
                      }}
                      onClick={() => loadTableData(table.name)}
                    >
                      <Group justify="space-between">
                        <Group gap="xs">
                          <IconTable size={14} />
                          <Text size="sm" fw={500}>{table.name}</Text>
                        </Group>
                        <Badge size="xs" variant="light">{(table.row_count ?? 0).toLocaleString()} rows</Badge>
                      </Group>
                    </Paper>
                  ))}
                </Stack>
              </Stack>
            </Paper>

            {/* Table Preview */}
            <Paper p="md" radius="md" withBorder>
              <Stack gap="md">
                <Group justify="space-between">
                  <Text size="sm" fw={600}>{selectedTable || 'Select a table'}</Text>
                  {selectedTable && (
                    <ActionIcon variant="light" onClick={() => loadTableData(selectedTable)}>
                      <IconRefresh size={14} />
                    </ActionIcon>
                  )}
                </Group>
                {selectedTable && tables.find((t) => t.name === selectedTable) && (
                  <>
                    <Code block style={{ fontSize: 11 }}>
                      {tables.find((t) => t.name === selectedTable)?.sql}
                    </Code>
                    {tableData.length > 0 && (
                      <ScrollArea h={300}>
                        <Table striped highlightOnHover withTableBorder>
                          <Table.Thead>
                            <Table.Tr>
                              {Object.keys(tableData[0]).map((col) => (
                                <Table.Th key={col}>{col}</Table.Th>
                              ))}
                            </Table.Tr>
                          </Table.Thead>
                          <Table.Tbody>
                            {tableData.slice(0, 10).map((row, i) => (
                              <Table.Tr key={i}>
                                {Object.values(row).map((val, j) => (
                                  <Table.Td key={j}>{String(val)}</Table.Td>
                                ))}
                              </Table.Tr>
                            ))}
                          </Table.Tbody>
                        </Table>
                      </ScrollArea>
                    )}
                  </>
                )}
                {!selectedTable && (
                  <Text size="sm" c="dimmed" ta="center" py="xl">
                    Click on a table to view its schema and data
                  </Text>
                )}
              </Stack>
            </Paper>
          </SimpleGrid>
        </Tabs.Panel>

        <Tabs.Panel value="metrics" pt="md">
          <Paper p="md" radius="md" withBorder>
            <Text size="sm" c="dimmed" ta="center" py="xl">
              Database metrics and query performance charts will appear here
            </Text>
          </Paper>
        </Tabs.Panel>
      </Tabs>
    </Stack>
  )
}
