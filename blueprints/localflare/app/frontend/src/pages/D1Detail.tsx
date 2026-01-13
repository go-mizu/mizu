import { Container, Title, Text, Card, Table, Button, Group, Textarea, Stack, Code, Tabs, Badge, ScrollArea } from '@mantine/core'
import { IconDatabase, IconPlayerPlay, IconTable, IconCode } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface D1Database {
  id: string
  name: string
}

interface QueryResult {
  columns: string[]
  rows: unknown[][]
  rows_affected: number
  duration_ms: number
}

interface TableInfo {
  name: string
  type: string
}

export function D1Detail() {
  const { id } = useParams<{ id: string }>()
  const [database, setDatabase] = useState<D1Database | null>(null)
  const [query, setQuery] = useState('SELECT * FROM sqlite_master WHERE type="table";')
  const [result, setResult] = useState<QueryResult | null>(null)
  const [tables, setTables] = useState<TableInfo[]>([])
  const [executing, setExecuting] = useState(false)

  useEffect(() => {
    fetchDatabase()
    fetchTables()
  }, [id])

  const fetchDatabase = async () => {
    try {
      const res = await fetch(`/api/d1/databases/${id}`)
      const data = await res.json()
      setDatabase(data.result)
    } catch (error) {
      console.error('Failed to fetch database:', error)
    }
  }

  const fetchTables = async () => {
    try {
      const res = await fetch(`/api/d1/databases/${id}/query`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sql: 'SELECT name, type FROM sqlite_master WHERE type="table" ORDER BY name' }),
      })
      const data = await res.json()
      if (data.result?.rows) {
        setTables(data.result.rows.map((r: string[]) => ({ name: r[0], type: r[1] })))
      }
    } catch (error) {
      console.error('Failed to fetch tables:', error)
    }
  }

  const executeQuery = async () => {
    setExecuting(true)
    try {
      const res = await fetch(`/api/d1/databases/${id}/query`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sql: query }),
      })
      const data = await res.json()
      if (data.success) {
        setResult(data.result)
        notifications.show({ title: 'Success', message: `Query executed in ${data.result.duration_ms}ms`, color: 'green' })
        fetchTables() // Refresh tables in case schema changed
      } else {
        notifications.show({ title: 'Error', message: data.errors?.[0]?.message || 'Query failed', color: 'red' })
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to execute query', color: 'red' })
    } finally {
      setExecuting(false)
    }
  }

  const selectTable = (tableName: string) => {
    setQuery(`SELECT * FROM ${tableName} LIMIT 100;`)
  }

  if (!database) return <Container py="xl"><Text>Loading...</Text></Container>

  return (
    <Container size="xl" py="xl">
      <Stack gap="xl">
        <Group justify="space-between">
          <Group>
            <IconDatabase size={32} color="var(--mantine-color-cyan-6)" />
            <div>
              <Title order={1}>{database.name}</Title>
              <Text c="dimmed">D1 Database</Text>
            </div>
          </Group>
        </Group>

        <Tabs defaultValue="console">
          <Tabs.List>
            <Tabs.Tab value="console" leftSection={<IconCode size={16} />}>Console</Tabs.Tab>
            <Tabs.Tab value="tables" leftSection={<IconTable size={16} />}>Tables</Tabs.Tab>
          </Tabs.List>

          <Tabs.Panel value="console" pt="md">
            <Stack gap="md">
              <Card withBorder shadow="sm" radius="md" p="md">
                <Text size="sm" fw={500} mb="xs">SQL Query</Text>
                <Textarea
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  minRows={4}
                  ff="monospace"
                  placeholder="Enter SQL query..."
                />
                <Group justify="flex-end" mt="md">
                  <Button leftSection={<IconPlayerPlay size={16} />} onClick={executeQuery} loading={executing}>
                    Execute
                  </Button>
                </Group>
              </Card>

              {result && (
                <Card withBorder shadow="sm" radius="md" p="md">
                  <Group justify="space-between" mb="md">
                    <Text fw={600}>Results</Text>
                    <Group gap="xs">
                      <Badge variant="outline">{result.rows?.length || 0} rows</Badge>
                      <Badge variant="outline" color="green">{result.duration_ms}ms</Badge>
                    </Group>
                  </Group>
                  <ScrollArea>
                    <Table striped highlightOnHover>
                      <Table.Thead>
                        <Table.Tr>
                          {result.columns?.map((col, i) => (
                            <Table.Th key={i}>{col}</Table.Th>
                          ))}
                        </Table.Tr>
                      </Table.Thead>
                      <Table.Tbody>
                        {result.rows?.map((row, i) => (
                          <Table.Tr key={i}>
                            {(row as unknown[]).map((cell, j) => (
                              <Table.Td key={j}>
                                <Text size="sm" ff="monospace" lineClamp={1}>
                                  {cell === null ? <Text c="dimmed">NULL</Text> : String(cell)}
                                </Text>
                              </Table.Td>
                            ))}
                          </Table.Tr>
                        ))}
                      </Table.Tbody>
                    </Table>
                  </ScrollArea>
                </Card>
              )}
            </Stack>
          </Tabs.Panel>

          <Tabs.Panel value="tables" pt="md">
            <Card withBorder shadow="sm" radius="md" p="lg">
              <Text fw={600} mb="md">Database Tables</Text>
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>Name</Table.Th>
                    <Table.Th>Type</Table.Th>
                    <Table.Th>Actions</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {tables.map((table) => (
                    <Table.Tr key={table.name}>
                      <Table.Td>
                        <Group gap="xs">
                          <IconTable size={16} />
                          <Code>{table.name}</Code>
                        </Group>
                      </Table.Td>
                      <Table.Td>
                        <Badge variant="outline">{table.type}</Badge>
                      </Table.Td>
                      <Table.Td>
                        <Button size="xs" variant="light" onClick={() => selectTable(table.name)}>
                          Query
                        </Button>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
              {tables.length === 0 && (
                <Text c="dimmed" ta="center" py="md">
                  No tables found. Create tables using the SQL Console.
                </Text>
              )}
            </Card>
          </Tabs.Panel>
        </Tabs>
      </Stack>
    </Container>
  )
}
