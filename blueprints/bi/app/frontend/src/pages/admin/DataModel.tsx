import { useState, useMemo } from 'react'
import {
  Container, Title, Text, Group, Stack, Button, TextInput, Select, Table, Paper,
  Badge, Modal, ActionIcon, Menu, Accordion, Tabs, Loader, ThemeIcon, Tooltip,
  PasswordInput, NumberInput, Switch, Box, SimpleGrid, Card, Divider,
  UnstyledButton, ScrollArea
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import {
  IconPlus, IconDatabase, IconTable, IconTrash, IconDotsVertical, IconRefresh,
  IconPlugConnected, IconCheck, IconX, IconColumns, IconEdit,
  IconHash, IconLetterCase, IconCalendar,
  IconToggleLeft, IconSettings, IconSql,
  IconBrandMysql, IconFileDatabase, IconServer
} from '@tabler/icons-react'
import {
  useDataSources, useCreateDataSource, useDeleteDataSource, useSyncDataSource,
  useTestDataSourceConnection, useTables, useColumns, useUpdateColumn
} from '../../api/hooks'
import type { DataSource, Table as TableType, Column } from '../../api/types'

const ENGINE_ICONS: Record<string, React.ComponentType<any>> = {
  sqlite: IconFileDatabase,
  postgres: IconSql,
  mysql: IconBrandMysql,
}

const ENGINE_LABELS: Record<string, string> = {
  sqlite: 'SQLite',
  postgres: 'PostgreSQL',
  mysql: 'MySQL',
}

const TYPE_ICONS: Record<string, React.ComponentType<any>> = {
  string: IconLetterCase,
  number: IconHash,
  boolean: IconToggleLeft,
  datetime: IconCalendar,
  date: IconCalendar,
}

const TYPE_COLORS: Record<string, string> = {
  string: 'green',
  number: 'blue',
  boolean: 'violet',
  datetime: 'orange',
  date: 'orange',
}

const SEMANTIC_OPTIONS = [
  { value: '', label: 'None' },
  { value: 'pk', label: 'Primary Key' },
  { value: 'fk', label: 'Foreign Key' },
  { value: 'name', label: 'Name/Title' },
  { value: 'category', label: 'Category' },
  { value: 'quantity', label: 'Quantity' },
  { value: 'price', label: 'Price/Currency' },
  { value: 'percentage', label: 'Percentage' },
  { value: 'latitude', label: 'Latitude' },
  { value: 'longitude', label: 'Longitude' },
  { value: 'email', label: 'Email' },
  { value: 'url', label: 'URL' },
  { value: 'image', label: 'Image URL' },
  { value: 'created_at', label: 'Creation Date' },
  { value: 'updated_at', label: 'Update Date' },
]

export default function DataModel() {
  const [activeTab, setActiveTab] = useState<string | null>('sources')
  const [addModalOpened, { open: openAddModal, close: closeAddModal }] = useDisclosure(false)
  const [selectedDatasource, setSelectedDatasource] = useState<string | null>(null)
  const [selectedTable, setSelectedTable] = useState<string | null>(null)
  const [editingColumn, setEditingColumn] = useState<Column | null>(null)

  // Data fetching
  const { data: datasources, isLoading: loadingDatasources } = useDataSources()
  const createDatasource = useCreateDataSource()
  const deleteDatasource = useDeleteDataSource()
  const syncDatasource = useSyncDataSource()
  const testConnection = useTestDataSourceConnection()
  const updateColumn = useUpdateColumn()

  const handleSyncTables = async (dsId: string) => {
    try {
      const result = await syncDatasource.mutateAsync({ id: dsId })
      notifications.show({
        title: 'Tables synced',
        message: `Synced ${result.tables_synced} tables from database`,
        color: 'green',
      })
    } catch (err: any) {
      notifications.show({
        title: 'Sync failed',
        message: err.message || 'Failed to sync tables',
        color: 'red',
      })
    }
  }

  const handleDeleteDatasource = async (ds: DataSource) => {
    if (!confirm(`Are you sure you want to delete "${ds.name}"? This cannot be undone.`)) return
    try {
      await deleteDatasource.mutateAsync(ds.id)
      notifications.show({
        title: 'Data source deleted',
        message: `${ds.name} has been removed`,
        color: 'green',
      })
      if (selectedDatasource === ds.id) {
        setSelectedDatasource(null)
        setSelectedTable(null)
      }
    } catch (err: any) {
      notifications.show({
        title: 'Delete failed',
        message: err.message || 'Failed to delete data source',
        color: 'red',
      })
    }
  }

  if (loadingDatasources) {
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
          <Text c="dimmed">Manage your data sources, tables, and metadata</Text>
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={openAddModal}>
          Add Data Source
        </Button>
      </Group>

      <Tabs value={activeTab} onChange={setActiveTab}>
        <Tabs.List mb="lg">
          <Tabs.Tab value="sources" leftSection={<IconDatabase size={16} />}>
            Data Sources
          </Tabs.Tab>
          <Tabs.Tab value="schema" leftSection={<IconTable size={16} />}>
            Schema Browser
          </Tabs.Tab>
          <Tabs.Tab value="metadata" leftSection={<IconColumns size={16} />}>
            Column Metadata
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="sources">
          <DataSourcesPanel
            datasources={datasources || []}
            onSync={handleSyncTables}
            onDelete={handleDeleteDatasource}
            syncingId={syncDatasource.isPending ? syncDatasource.variables?.id ?? null : null}
          />
        </Tabs.Panel>

        <Tabs.Panel value="schema">
          <SchemaBrowserPanel
            datasources={datasources || []}
            selectedDatasource={selectedDatasource}
            selectedTable={selectedTable}
            onSelectDatasource={setSelectedDatasource}
            onSelectTable={setSelectedTable}
          />
        </Tabs.Panel>

        <Tabs.Panel value="metadata">
          <MetadataPanel
            datasources={datasources || []}
            selectedDatasource={selectedDatasource}
            selectedTable={selectedTable}
            onSelectDatasource={setSelectedDatasource}
            onSelectTable={setSelectedTable}
            editingColumn={editingColumn}
            onEditColumn={setEditingColumn}
            onUpdateColumn={updateColumn}
          />
        </Tabs.Panel>
      </Tabs>

      {/* Add Data Source Modal */}
      <AddDataSourceModal
        opened={addModalOpened}
        onClose={closeAddModal}
        onCreate={createDatasource}
        onTest={testConnection}
      />
    </Container>
  )
}

// Data Sources Panel
function DataSourcesPanel({
  datasources,
  onSync,
  onDelete,
  syncingId,
}: {
  datasources: DataSource[]
  onSync: (id: string) => void
  onDelete: (ds: DataSource) => void
  syncingId: string | null
}) {
  if (datasources.length === 0) {
    return (
      <Paper withBorder radius="md" p="xl" ta="center">
        <Stack align="center" gap="md">
          <ThemeIcon size={60} radius="xl" variant="light" color="orange">
            <IconDatabase size={30} />
          </ThemeIcon>
          <Title order={3}>No data sources</Title>
          <Text c="dimmed" maw={400}>
            Add a data source to start exploring your data. You can connect to SQLite, PostgreSQL, or MySQL databases.
          </Text>
        </Stack>
      </Paper>
    )
  }

  return (
    <SimpleGrid cols={{ base: 1, md: 2, lg: 3 }} spacing="md">
      {datasources.map((ds) => (
        <DataSourceCard
          key={ds.id}
          datasource={ds}
          onSync={() => onSync(ds.id)}
          onDelete={() => onDelete(ds)}
          isSyncing={syncingId === ds.id}
        />
      ))}
    </SimpleGrid>
  )
}

function DataSourceCard({
  datasource,
  onSync,
  onDelete,
  isSyncing,
}: {
  datasource: DataSource
  onSync: () => void
  onDelete: () => void
  isSyncing: boolean
}) {
  const EngineIcon = ENGINE_ICONS[datasource.engine] || IconDatabase
  const { data: tables } = useTables(datasource.id)

  return (
    <Card withBorder radius="md" padding="lg">
      <Group justify="space-between" mb="md">
        <Group gap="sm">
          <ThemeIcon size="lg" variant="light" color="orange">
            <EngineIcon size={20} />
          </ThemeIcon>
          <div>
            <Text fw={600}>{datasource.name}</Text>
            <Text size="xs" c="dimmed">{ENGINE_LABELS[datasource.engine]}</Text>
          </div>
        </Group>
        <Menu shadow="md" position="bottom-end">
          <Menu.Target>
            <ActionIcon variant="subtle" size="sm">
              <IconDotsVertical size={16} />
            </ActionIcon>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Item leftSection={<IconRefresh size={14} />} onClick={onSync}>
              Sync tables
            </Menu.Item>
            <Menu.Item leftSection={<IconSettings size={14} />}>
              Settings
            </Menu.Item>
            <Menu.Divider />
            <Menu.Item leftSection={<IconTrash size={14} />} color="red" onClick={onDelete}>
              Delete
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Group>

      <Stack gap="xs">
        <Group gap="xs">
          <Badge size="sm" variant="light" color="blue">
            {tables?.length || 0} tables
          </Badge>
          {datasource.ssl && (
            <Badge size="sm" variant="light" color="green">
              SSL
            </Badge>
          )}
        </Group>

        <Text size="sm" c="dimmed" lineClamp={1}>
          {datasource.host ? `${datasource.host}:${datasource.port}/${datasource.database}` : datasource.database}
        </Text>

        <Group gap="xs" mt="xs">
          <Button
            size="xs"
            variant="light"
            leftSection={<IconRefresh size={14} />}
            onClick={onSync}
            loading={isSyncing}
            style={{ flex: 1 }}
          >
            Sync
          </Button>
        </Group>
      </Stack>
    </Card>
  )
}

// Schema Browser Panel
function SchemaBrowserPanel({
  datasources,
  selectedDatasource,
  selectedTable,
  onSelectDatasource,
  onSelectTable,
}: {
  datasources: DataSource[]
  selectedDatasource: string | null
  selectedTable: string | null
  onSelectDatasource: (id: string | null) => void
  onSelectTable: (id: string | null) => void
}) {
  const { data: tables, isLoading: loadingTables } = useTables(selectedDatasource || '')
  const { data: columns, isLoading: loadingColumns } = useColumns(selectedTable || '')

  // Group tables by schema
  const tablesBySchema = useMemo(() => {
    if (!tables) return {}
    return tables.reduce((acc, table) => {
      const schema = table.schema || 'default'
      if (!acc[schema]) acc[schema] = []
      acc[schema].push(table)
      return acc
    }, {} as Record<string, TableType[]>)
  }, [tables])

  return (
    <Box style={{ display: 'flex', gap: 16, height: 'calc(100vh - 280px)' }}>
      {/* Sidebar - Datasources & Tables */}
      <Paper withBorder radius="md" style={{ width: 300, display: 'flex', flexDirection: 'column' }}>
        <Box p="sm" style={{ borderBottom: '1px solid var(--mantine-color-gray-3)' }}>
          <Select
            placeholder="Select data source"
            data={datasources.map(ds => ({
              value: ds.id,
              label: ds.name,
            }))}
            value={selectedDatasource}
            onChange={(value) => {
              onSelectDatasource(value)
              onSelectTable(null)
            }}
            leftSection={<IconDatabase size={16} />}
            size="sm"
          />
        </Box>

        <ScrollArea style={{ flex: 1 }} p="sm">
          {!selectedDatasource ? (
            <Text c="dimmed" size="sm" ta="center" py="xl">
              Select a data source to browse tables
            </Text>
          ) : loadingTables ? (
            <Stack align="center" py="xl">
              <Loader size="sm" />
              <Text size="sm" c="dimmed">Loading tables...</Text>
            </Stack>
          ) : tables?.length === 0 ? (
            <Text c="dimmed" size="sm" ta="center" py="xl">
              No tables found. Click "Sync" to discover tables.
            </Text>
          ) : (
            <Accordion variant="filled" multiple>
              {Object.entries(tablesBySchema).map(([schema, schemaTables]) => (
                <Accordion.Item key={schema} value={schema}>
                  <Accordion.Control>
                    <Group gap="xs">
                      <IconServer size={14} />
                      <Text size="sm" fw={500}>{schema}</Text>
                      <Badge size="xs" variant="light">{schemaTables.length}</Badge>
                    </Group>
                  </Accordion.Control>
                  <Accordion.Panel>
                    <Stack gap={4}>
                      {schemaTables.map((table) => (
                        <UnstyledButton
                          key={table.id}
                          onClick={() => onSelectTable(table.id)}
                          p="xs"
                          style={{
                            borderRadius: 4,
                            backgroundColor: selectedTable === table.id
                              ? 'var(--mantine-color-brand-0)'
                              : 'transparent',
                          }}
                        >
                          <Group gap="xs">
                            <IconTable size={14} color="var(--mantine-color-blue-6)" />
                            <div style={{ flex: 1 }}>
                              <Text size="sm">{table.display_name || table.name}</Text>
                              <Text size="xs" c="dimmed">{table.row_count?.toLocaleString()} rows</Text>
                            </div>
                          </Group>
                        </UnstyledButton>
                      ))}
                    </Stack>
                  </Accordion.Panel>
                </Accordion.Item>
              ))}
            </Accordion>
          )}
        </ScrollArea>
      </Paper>

      {/* Main - Columns */}
      <Paper withBorder radius="md" style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {!selectedTable ? (
          <Stack align="center" justify="center" style={{ flex: 1 }} gap="md">
            <ThemeIcon size={60} radius="xl" variant="light" color="gray">
              <IconColumns size={30} />
            </ThemeIcon>
            <Text c="dimmed">Select a table to view its columns</Text>
          </Stack>
        ) : loadingColumns ? (
          <Stack align="center" justify="center" style={{ flex: 1 }}>
            <Loader size="lg" />
            <Text c="dimmed">Loading columns...</Text>
          </Stack>
        ) : (
          <>
            <Box p="md" style={{ borderBottom: '1px solid var(--mantine-color-gray-3)' }}>
              <Group justify="space-between">
                <Text fw={600}>
                  {tables?.find(t => t.id === selectedTable)?.display_name || 'Table'}
                </Text>
                <Badge variant="light">{columns?.length || 0} columns</Badge>
              </Group>
            </Box>
            <ScrollArea style={{ flex: 1 }}>
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>Column</Table.Th>
                    <Table.Th>Type</Table.Th>
                    <Table.Th>Semantic</Table.Th>
                    <Table.Th>Description</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {columns?.map((col) => {
                    const TypeIcon = TYPE_ICONS[col.type] || IconHash
                    return (
                      <Table.Tr key={col.id}>
                        <Table.Td>
                          <Group gap="xs">
                            <TypeIcon size={14} color={`var(--mantine-color-${TYPE_COLORS[col.type] || 'gray'}-6)`} />
                            <Text size="sm" fw={500}>{col.display_name || col.name}</Text>
                            {col.name !== col.display_name && (
                              <Text size="xs" c="dimmed">({col.name})</Text>
                            )}
                          </Group>
                        </Table.Td>
                        <Table.Td>
                          <Badge size="sm" variant="light" color={TYPE_COLORS[col.type] || 'gray'}>
                            {col.type}
                          </Badge>
                        </Table.Td>
                        <Table.Td>
                          {col.semantic ? (
                            <Badge size="sm" variant="dot" color="brand">
                              {SEMANTIC_OPTIONS.find(s => s.value === col.semantic)?.label || col.semantic}
                            </Badge>
                          ) : (
                            <Text size="sm" c="dimmed">—</Text>
                          )}
                        </Table.Td>
                        <Table.Td>
                          <Text size="sm" c="dimmed" lineClamp={1}>
                            {col.description || '—'}
                          </Text>
                        </Table.Td>
                      </Table.Tr>
                    )
                  })}
                </Table.Tbody>
              </Table>
            </ScrollArea>
          </>
        )}
      </Paper>
    </Box>
  )
}

// Metadata Panel
function MetadataPanel({
  datasources,
  selectedDatasource,
  selectedTable,
  onSelectDatasource,
  onSelectTable,
  editingColumn,
  onEditColumn,
  onUpdateColumn,
}: {
  datasources: DataSource[]
  selectedDatasource: string | null
  selectedTable: string | null
  onSelectDatasource: (id: string | null) => void
  onSelectTable: (id: string | null) => void
  editingColumn: Column | null
  onEditColumn: (col: Column | null) => void
  onUpdateColumn: ReturnType<typeof useUpdateColumn>
}) {
  const { data: tables } = useTables(selectedDatasource || '')
  const { data: columns, isLoading: loadingColumns } = useColumns(selectedTable || '')

  const [editDisplayName, setEditDisplayName] = useState('')
  const [editDescription, setEditDescription] = useState('')
  const [editSemantic, setEditSemantic] = useState('')

  const handleStartEdit = (col: Column) => {
    onEditColumn(col)
    setEditDisplayName(col.display_name || col.name)
    setEditDescription(col.description || '')
    setEditSemantic(col.semantic || '')
  }

  const handleSaveColumn = async () => {
    if (!editingColumn || !selectedTable) return
    try {
      await onUpdateColumn.mutateAsync({
        tableId: selectedTable,
        columnId: editingColumn.id,
        display_name: editDisplayName,
        description: editDescription,
        semantic: editSemantic || undefined,
      })
      notifications.show({
        title: 'Column updated',
        message: 'Column metadata has been saved',
        color: 'green',
      })
      onEditColumn(null)
    } catch (err: any) {
      notifications.show({
        title: 'Update failed',
        message: err.message || 'Failed to update column',
        color: 'red',
      })
    }
  }

  return (
    <Box>
      <Paper withBorder radius="md" p="md" mb="md">
        <Group gap="md">
          <Select
            placeholder="Select data source"
            data={datasources.map(ds => ({ value: ds.id, label: ds.name }))}
            value={selectedDatasource}
            onChange={(value) => {
              onSelectDatasource(value)
              onSelectTable(null)
            }}
            leftSection={<IconDatabase size={16} />}
            style={{ width: 250 }}
          />
          <Select
            placeholder="Select table"
            data={tables?.map(t => ({ value: t.id, label: t.display_name || t.name })) || []}
            value={selectedTable}
            onChange={onSelectTable}
            leftSection={<IconTable size={16} />}
            disabled={!selectedDatasource}
            style={{ width: 250 }}
          />
        </Group>
      </Paper>

      {!selectedTable ? (
        <Paper withBorder radius="md" p="xl" ta="center">
          <Stack align="center" gap="md">
            <ThemeIcon size={60} radius="xl" variant="light" color="gray">
              <IconEdit size={30} />
            </ThemeIcon>
            <Text c="dimmed">Select a table to edit column metadata</Text>
            <Text size="sm" c="dimmed" maw={400}>
              Column metadata helps BI tools understand your data better. Set semantic types, display names, and descriptions.
            </Text>
          </Stack>
        </Paper>
      ) : loadingColumns ? (
        <Paper withBorder radius="md" p="xl" ta="center">
          <Loader size="lg" />
          <Text mt="md" c="dimmed">Loading columns...</Text>
        </Paper>
      ) : (
        <Paper withBorder radius="md">
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th style={{ width: 200 }}>Column Name</Table.Th>
                <Table.Th style={{ width: 200 }}>Display Name</Table.Th>
                <Table.Th style={{ width: 100 }}>Type</Table.Th>
                <Table.Th style={{ width: 150 }}>Semantic Type</Table.Th>
                <Table.Th>Description</Table.Th>
                <Table.Th style={{ width: 80 }}></Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {columns?.map((col) => {
                const isEditing = editingColumn?.id === col.id
                const TypeIcon = TYPE_ICONS[col.type] || IconHash

                if (isEditing) {
                  return (
                    <Table.Tr key={col.id} bg="brand.0">
                      <Table.Td>
                        <Group gap="xs">
                          <TypeIcon size={14} />
                          <Text size="sm" fw={500}>{col.name}</Text>
                        </Group>
                      </Table.Td>
                      <Table.Td>
                        <TextInput
                          size="xs"
                          value={editDisplayName}
                          onChange={(e) => setEditDisplayName(e.target.value)}
                        />
                      </Table.Td>
                      <Table.Td>
                        <Badge size="sm" variant="light" color={TYPE_COLORS[col.type]}>
                          {col.type}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        <Select
                          size="xs"
                          data={SEMANTIC_OPTIONS}
                          value={editSemantic}
                          onChange={(v) => setEditSemantic(v || '')}
                        />
                      </Table.Td>
                      <Table.Td>
                        <TextInput
                          size="xs"
                          value={editDescription}
                          onChange={(e) => setEditDescription(e.target.value)}
                          placeholder="Add description..."
                        />
                      </Table.Td>
                      <Table.Td>
                        <Group gap={4}>
                          <ActionIcon
                            size="sm"
                            variant="filled"
                            color="green"
                            onClick={handleSaveColumn}
                            loading={onUpdateColumn.isPending}
                          >
                            <IconCheck size={14} />
                          </ActionIcon>
                          <ActionIcon
                            size="sm"
                            variant="subtle"
                            color="gray"
                            onClick={() => onEditColumn(null)}
                          >
                            <IconX size={14} />
                          </ActionIcon>
                        </Group>
                      </Table.Td>
                    </Table.Tr>
                  )
                }

                return (
                  <Table.Tr key={col.id}>
                    <Table.Td>
                      <Group gap="xs">
                        <TypeIcon size={14} color={`var(--mantine-color-${TYPE_COLORS[col.type] || 'gray'}-6)`} />
                        <Text size="sm">{col.name}</Text>
                      </Group>
                    </Table.Td>
                    <Table.Td>
                      <Text size="sm" fw={500}>{col.display_name || col.name}</Text>
                    </Table.Td>
                    <Table.Td>
                      <Badge size="sm" variant="light" color={TYPE_COLORS[col.type] || 'gray'}>
                        {col.type}
                      </Badge>
                    </Table.Td>
                    <Table.Td>
                      {col.semantic ? (
                        <Badge size="sm" variant="dot" color="brand">
                          {SEMANTIC_OPTIONS.find(s => s.value === col.semantic)?.label || col.semantic}
                        </Badge>
                      ) : (
                        <Text size="xs" c="dimmed">—</Text>
                      )}
                    </Table.Td>
                    <Table.Td>
                      <Text size="sm" c="dimmed" lineClamp={1}>
                        {col.description || '—'}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Tooltip label="Edit metadata">
                        <ActionIcon variant="subtle" size="sm" onClick={() => handleStartEdit(col)}>
                          <IconEdit size={14} />
                        </ActionIcon>
                      </Tooltip>
                    </Table.Td>
                  </Table.Tr>
                )
              })}
            </Table.Tbody>
          </Table>
        </Paper>
      )}
    </Box>
  )
}

// Add Data Source Modal
function AddDataSourceModal({
  opened,
  onClose,
  onCreate,
  onTest,
}: {
  opened: boolean
  onClose: () => void
  onCreate: ReturnType<typeof useCreateDataSource>
  onTest: ReturnType<typeof useTestDataSourceConnection>
}) {
  const [engine, setEngine] = useState<string>('sqlite')
  const [name, setName] = useState('')
  const [host, setHost] = useState('')
  const [port, setPort] = useState<number | ''>(5432)
  const [database, setDatabase] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [ssl, setSsl] = useState(false)
  const [testStatus, setTestStatus] = useState<'idle' | 'success' | 'error'>('idle')
  const [testError, setTestError] = useState('')

  const resetForm = () => {
    setEngine('sqlite')
    setName('')
    setHost('')
    setPort(5432)
    setDatabase('')
    setUsername('')
    setPassword('')
    setSsl(false)
    setTestStatus('idle')
    setTestError('')
  }

  const handleClose = () => {
    resetForm()
    onClose()
  }

  const handleTestConnection = async () => {
    setTestStatus('idle')
    setTestError('')
    try {
      const result = await onTest.mutateAsync({
        engine,
        host: engine !== 'sqlite' ? host : undefined,
        port: engine !== 'sqlite' && port ? port : undefined,
        database,
        username: engine !== 'sqlite' ? username : undefined,
        password: engine !== 'sqlite' ? password : undefined,
      })
      if (result.success) {
        setTestStatus('success')
      } else {
        setTestStatus('error')
        setTestError(result.error || 'Connection failed')
      }
    } catch (err: any) {
      setTestStatus('error')
      setTestError(err.message || 'Connection test failed')
    }
  }

  const handleCreate = async () => {
    if (!name || !database) return
    try {
      await onCreate.mutateAsync({
        name,
        engine: engine as 'sqlite' | 'postgres' | 'mysql',
        host: engine !== 'sqlite' ? host : undefined,
        port: engine !== 'sqlite' && port ? port : undefined,
        database,
        username: engine !== 'sqlite' ? username : undefined,
        ssl,
      })
      notifications.show({
        title: 'Data source created',
        message: `${name} has been added. Syncing tables...`,
        color: 'green',
      })
      handleClose()
    } catch (err: any) {
      notifications.show({
        title: 'Creation failed',
        message: err.message || 'Failed to create data source',
        color: 'red',
      })
    }
  }

  return (
    <Modal opened={opened} onClose={handleClose} title="Add Data Source" size="lg">
      <Stack gap="md">
        <TextInput
          label="Display Name"
          placeholder="My Database"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />

        <Select
          label="Database Engine"
          data={[
            { value: 'sqlite', label: 'SQLite' },
            { value: 'postgres', label: 'PostgreSQL' },
            { value: 'mysql', label: 'MySQL' },
          ]}
          value={engine}
          onChange={(v) => {
            setEngine(v || 'sqlite')
            if (v === 'postgres') setPort(5432)
            else if (v === 'mysql') setPort(3306)
          }}
          leftSection={
            engine === 'sqlite' ? <IconFileDatabase size={16} /> :
            engine === 'postgres' ? <IconSql size={16} /> :
            <IconBrandMysql size={16} />
          }
        />

        {engine === 'sqlite' ? (
          <TextInput
            label="Database Path"
            placeholder="/path/to/database.db"
            value={database}
            onChange={(e) => setDatabase(e.target.value)}
            required
          />
        ) : (
          <>
            <Group grow>
              <TextInput
                label="Host"
                placeholder="localhost"
                value={host}
                onChange={(e) => setHost(e.target.value)}
                required
              />
              <NumberInput
                label="Port"
                placeholder={engine === 'postgres' ? '5432' : '3306'}
                value={port}
                onChange={(v) => setPort(v as number | '')}
                min={1}
                max={65535}
                style={{ width: 120 }}
              />
            </Group>

            <TextInput
              label="Database"
              placeholder="mydb"
              value={database}
              onChange={(e) => setDatabase(e.target.value)}
              required
            />

            <Group grow>
              <TextInput
                label="Username"
                placeholder="admin"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
              />
              <PasswordInput
                label="Password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </Group>

            <Switch
              label="Use SSL"
              description="Encrypt connection to database"
              checked={ssl}
              onChange={(e) => setSsl(e.currentTarget.checked)}
            />
          </>
        )}

        {/* Test connection result */}
        {testStatus !== 'idle' && (
          <Paper
            withBorder
            p="sm"
            radius="md"
            bg={testStatus === 'success' ? 'green.0' : 'red.0'}
          >
            <Group gap="sm">
              <ThemeIcon
                color={testStatus === 'success' ? 'green' : 'red'}
                variant="light"
              >
                {testStatus === 'success' ? <IconCheck size={16} /> : <IconX size={16} />}
              </ThemeIcon>
              <Text size="sm" c={testStatus === 'success' ? 'green.7' : 'red.7'}>
                {testStatus === 'success' ? 'Connection successful!' : testError}
              </Text>
            </Group>
          </Paper>
        )}

        <Divider />

        <Group justify="space-between">
          <Button
            variant="light"
            leftSection={<IconPlugConnected size={16} />}
            onClick={handleTestConnection}
            loading={onTest.isPending}
          >
            Test Connection
          </Button>
          <Group gap="sm">
            <Button variant="subtle" onClick={handleClose}>
              Cancel
            </Button>
            <Button
              onClick={handleCreate}
              loading={onCreate.isPending}
              disabled={!name || !database}
            >
              Add Data Source
            </Button>
          </Group>
        </Group>
      </Stack>
    </Modal>
  )
}
