import { useEffect, useState } from 'react'
import { Container, Title, Text, Group, Stack, Button, TextInput, Select, Table, Paper, Loader, Badge, Modal, ActionIcon, Menu, Accordion } from '@mantine/core'
import { IconPlus, IconDatabase, IconTable, IconTrash, IconDotsVertical, IconRefresh, IconPlugConnected } from '@tabler/icons-react'
import { api } from '../../api/client'

interface DataSource {
  id: string
  name: string
  engine: string
  connection_string: string
  created_at: string
}

interface TableInfo {
  id: string
  datasource_id: string
  name: string
  schema_name: string
  display_name: string
}

interface ColumnInfo {
  id: string
  table_id: string
  name: string
  data_type: string
  display_name: string
  description: string
}

export default function DataModel() {
  const [datasources, setDatasources] = useState<DataSource[]>([])
  const [tables, setTables] = useState<Record<string, TableInfo[]>>({})
  const [columns, setColumns] = useState<Record<string, ColumnInfo[]>>({})
  const [loading, setLoading] = useState(true)
  const [addModalOpen, setAddModalOpen] = useState(false)
  const [testingConnection, setTestingConnection] = useState(false)

  // Form state
  const [formName, setFormName] = useState('')
  const [formEngine, setFormEngine] = useState('sqlite')
  const [formConnection, setFormConnection] = useState('')

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const res = await api.get<DataSource[]>('/datasources')
      setDatasources(res || [])

      // Load tables for each datasource
      for (const ds of res || []) {
        loadTables(ds.id)
      }
    } catch (error) {
      console.error('Failed to load datasources:', error)
    } finally {
      setLoading(false)
    }
  }

  const loadTables = async (datasourceId: string) => {
    try {
      const res = await api.get<TableInfo[]>(`/datasources/${datasourceId}/tables`)
      setTables(prev => ({ ...prev, [datasourceId]: res || [] }))

      // Load columns for each table
      for (const table of res || []) {
        loadColumns(table.id)
      }
    } catch (error) {
      console.error('Failed to load tables:', error)
    }
  }

  const loadColumns = async (tableId: string) => {
    try {
      const res = await api.get<ColumnInfo[]>(`/tables/${tableId}/columns`)
      setColumns(prev => ({ ...prev, [tableId]: res || [] }))
    } catch (error) {
      console.error('Failed to load columns:', error)
    }
  }

  const handleTestConnection = async () => {
    setTestingConnection(true)
    try {
      await api.post('/datasources/test', {
        engine: formEngine,
        connection_string: formConnection,
      })
      alert('Connection successful!')
    } catch (err) {
      alert('Connection failed: ' + (err instanceof Error ? err.message : 'Unknown error'))
    } finally {
      setTestingConnection(false)
    }
  }

  const handleAddDatasource = async () => {
    if (!formName || !formConnection) return
    try {
      const ds = await api.post<DataSource>('/datasources', {
        name: formName,
        engine: formEngine,
        connection_string: formConnection,
      })
      setDatasources(prev => [...prev, ds])
      setAddModalOpen(false)
      setFormName('')
      setFormEngine('sqlite')
      setFormConnection('')
      // Sync tables for new datasource
      handleSyncTables(ds.id)
    } catch (err) {
      alert('Failed to add datasource')
    }
  }

  const handleSyncTables = async (datasourceId: string) => {
    try {
      await api.post(`/datasources/${datasourceId}/sync`, {})
      loadTables(datasourceId)
    } catch (err) {
      alert('Failed to sync tables')
    }
  }

  const handleDeleteDatasource = async (id: string) => {
    if (!confirm('Are you sure you want to delete this data source?')) return
    try {
      await api.delete(`/datasources/${id}`)
      setDatasources(prev => prev.filter(ds => ds.id !== id))
    } catch (err) {
      alert('Failed to delete datasource')
    }
  }

  const getEngineIcon = (engine: string) => {
    switch (engine) {
      case 'sqlite':
        return 'SQLite'
      case 'postgres':
        return 'PostgreSQL'
      case 'mysql':
        return 'MySQL'
      default:
        return engine
    }
  }

  const getTypeColor = (type: string) => {
    if (type.includes('INT') || type.includes('REAL') || type.includes('NUMERIC')) {
      return 'blue'
    }
    if (type.includes('TEXT') || type.includes('VARCHAR') || type.includes('CHAR')) {
      return 'green'
    }
    if (type.includes('DATE') || type.includes('TIME')) {
      return 'orange'
    }
    if (type.includes('BOOL')) {
      return 'violet'
    }
    return 'gray'
  }

  if (loading) {
    return (
      <Container size="xl" py="lg">
        <Paper withBorder p="xl" ta="center">
          <Loader size="lg" />
          <Text mt="md">Loading data model...</Text>
        </Paper>
      </Container>
    )
  }

  return (
    <Container size="xl" py="lg">
      {/* Header */}
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={2}>Data Model</Title>
          <Text c="dimmed">Manage your data sources and tables</Text>
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={() => setAddModalOpen(true)}>
          Add Data Source
        </Button>
      </Group>

      {/* Data Sources */}
      {datasources.length === 0 ? (
        <Paper withBorder radius="md" p="xl" ta="center">
          <Stack align="center" gap="md">
            <IconDatabase size={48} color="var(--mantine-color-gray-5)" />
            <Title order={3}>No data sources</Title>
            <Text c="dimmed">
              Add a data source to start exploring your data
            </Text>
            <Button leftSection={<IconPlus size={16} />} onClick={() => setAddModalOpen(true)}>
              Add Data Source
            </Button>
          </Stack>
        </Paper>
      ) : (
        <Accordion variant="separated">
          {datasources.map((ds) => (
            <Accordion.Item key={ds.id} value={ds.id}>
              <Accordion.Control>
                <Group>
                  <IconDatabase size={20} color="var(--mantine-color-orange-6)" />
                  <div>
                    <Text fw={500}>{ds.name}</Text>
                    <Text size="sm" c="dimmed">
                      {getEngineIcon(ds.engine)} - {tables[ds.id]?.length || 0} tables
                    </Text>
                  </div>
                </Group>
              </Accordion.Control>
              <Accordion.Panel>
                <Stack>
                  <Group justify="space-between">
                    <Badge color="orange" variant="light">{ds.engine}</Badge>
                    <Group>
                      <Button
                        variant="light"
                        size="xs"
                        leftSection={<IconRefresh size={14} />}
                        onClick={() => handleSyncTables(ds.id)}
                      >
                        Sync Tables
                      </Button>
                      <Menu shadow="md" width={200}>
                        <Menu.Target>
                          <ActionIcon variant="subtle" size="sm">
                            <IconDotsVertical size={16} />
                          </ActionIcon>
                        </Menu.Target>
                        <Menu.Dropdown>
                          <Menu.Item
                            leftSection={<IconTrash size={14} />}
                            color="red"
                            onClick={() => handleDeleteDatasource(ds.id)}
                          >
                            Delete
                          </Menu.Item>
                        </Menu.Dropdown>
                      </Menu>
                    </Group>
                  </Group>

                  {/* Tables */}
                  {tables[ds.id]?.length > 0 ? (
                    <Accordion variant="contained">
                      {tables[ds.id].map((table) => (
                        <Accordion.Item key={table.id} value={table.id}>
                          <Accordion.Control>
                            <Group>
                              <IconTable size={16} color="var(--mantine-color-blue-6)" />
                              <Text size="sm" fw={500}>{table.display_name || table.name}</Text>
                              <Badge size="xs" variant="light">
                                {columns[table.id]?.length || 0} columns
                              </Badge>
                            </Group>
                          </Accordion.Control>
                          <Accordion.Panel>
                            {columns[table.id]?.length > 0 ? (
                              <Table>
                                <Table.Thead>
                                  <Table.Tr>
                                    <Table.Th>Column</Table.Th>
                                    <Table.Th>Type</Table.Th>
                                    <Table.Th>Description</Table.Th>
                                  </Table.Tr>
                                </Table.Thead>
                                <Table.Tbody>
                                  {columns[table.id].map((col) => (
                                    <Table.Tr key={col.id}>
                                      <Table.Td>
                                        <Text size="sm" fw={500}>
                                          {col.display_name || col.name}
                                        </Text>
                                      </Table.Td>
                                      <Table.Td>
                                        <Badge size="xs" color={getTypeColor(col.data_type)} variant="light">
                                          {col.data_type}
                                        </Badge>
                                      </Table.Td>
                                      <Table.Td>
                                        <Text size="sm" c="dimmed">
                                          {col.description || '-'}
                                        </Text>
                                      </Table.Td>
                                    </Table.Tr>
                                  ))}
                                </Table.Tbody>
                              </Table>
                            ) : (
                              <Text c="dimmed" size="sm">No columns found</Text>
                            )}
                          </Accordion.Panel>
                        </Accordion.Item>
                      ))}
                    </Accordion>
                  ) : (
                    <Text c="dimmed" size="sm">No tables found. Click "Sync Tables" to discover tables.</Text>
                  )}
                </Stack>
              </Accordion.Panel>
            </Accordion.Item>
          ))}
        </Accordion>
      )}

      {/* Add Data Source Modal */}
      <Modal opened={addModalOpen} onClose={() => setAddModalOpen(false)} title="Add Data Source" size="lg">
        <Stack>
          <TextInput
            label="Name"
            placeholder="My Database"
            value={formName}
            onChange={(e) => setFormName(e.target.value)}
            required
          />
          <Select
            label="Database Engine"
            data={[
              { value: 'sqlite', label: 'SQLite' },
              { value: 'postgres', label: 'PostgreSQL' },
              { value: 'mysql', label: 'MySQL' },
            ]}
            value={formEngine}
            onChange={(v) => setFormEngine(v || 'sqlite')}
          />
          <TextInput
            label="Connection String"
            placeholder={formEngine === 'sqlite' ? '/path/to/database.db' : 'host=localhost user=admin password=secret dbname=mydb'}
            value={formConnection}
            onChange={(e) => setFormConnection(e.target.value)}
            required
          />
          <Group justify="space-between">
            <Button
              variant="light"
              leftSection={<IconPlugConnected size={16} />}
              onClick={handleTestConnection}
              loading={testingConnection}
            >
              Test Connection
            </Button>
            <Group>
              <Button variant="light" onClick={() => setAddModalOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleAddDatasource} disabled={!formName || !formConnection}>
                Add
              </Button>
            </Group>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
