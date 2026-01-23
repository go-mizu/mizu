import { useState, useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Card, Group, SimpleGrid, Badge, TextInput, Text, Title,
  Paper, Tabs, Table, ActionIcon, Button, Breadcrumbs, Anchor, ThemeIcon,
  Loader, Pagination, Select, Tooltip, Box, ScrollArea
} from '@mantine/core'
import {
  IconSearch, IconTable, IconDatabase, IconColumns, IconArrowLeft,
  IconRefresh, IconChartBar, IconSortAscending, IconSortDescending,
  IconEye, IconEyeOff
} from '@tabler/icons-react'
import {
  useDataSource, useTables, useColumns, useTablePreview, useSyncTable
} from '../../api/hooks'
import { PageContainer, LoadingState, EmptyState } from '../../components/ui'

export default function DatabaseBrowser() {
  const navigate = useNavigate()
  const { datasourceId, tableId } = useParams()
  const [search, setSearch] = useState('')
  const [selectedSchema, setSelectedSchema] = useState<string | null>(null)

  // Fetch data
  const { data: datasource, isLoading: loadingDs } = useDataSource(datasourceId || '')
  const { data: tables, isLoading: loadingTables } = useTables(datasourceId || '')

  // Group tables by schema
  const { schemas, filteredTables } = useMemo(() => {
    if (!tables) return { schemas: [], filteredTables: [] }

    const schemaSet = new Set<string>()
    const searchLower = search.toLowerCase()

    tables.forEach(table => {
      const schema = table.schema || 'default'
      schemaSet.add(schema)
    })

    let filtered = tables
    if (search) {
      filtered = tables.filter(t =>
        t.name.toLowerCase().includes(searchLower) ||
        t.display_name?.toLowerCase().includes(searchLower) ||
        t.schema?.toLowerCase().includes(searchLower)
      )
    }
    if (selectedSchema) {
      filtered = filtered.filter(t => (t.schema || 'default') === selectedSchema)
    }

    return {
      schemas: Array.from(schemaSet).sort(),
      filteredTables: filtered
    }
  }, [tables, search, selectedSchema])

  if (loadingDs || loadingTables) {
    return (
      <PageContainer>
        <LoadingState message="Loading database..." />
      </PageContainer>
    )
  }

  if (!datasource) {
    return (
      <PageContainer>
        <EmptyState
          icon={<IconDatabase size={32} strokeWidth={1.5} />}
          iconColor="var(--color-warning)"
          title="Database not found"
          description="The requested database could not be found."
          action={
            <Button onClick={() => navigate('/browse/databases')}>
              Back to Databases
            </Button>
          }
        />
      </PageContainer>
    )
  }

  // If a table is selected, show table preview
  if (tableId) {
    return (
      <TablePreviewPage
        datasourceId={datasourceId!}
        tableId={tableId}
        datasourceName={datasource.name}
        onBack={() => navigate(`/browse/database/${datasourceId}`)}
      />
    )
  }

  return (
    <PageContainer>
      {/* Header */}
      <Group justify="space-between" mb="xl">
        <div>
          <Breadcrumbs mb="xs">
            <Anchor onClick={() => navigate('/browse/databases')} style={{ color: 'var(--color-foreground-muted)' }}>
              Databases
            </Anchor>
            <Text fw={600} style={{ color: 'var(--color-foreground)' }}>{datasource.name}</Text>
          </Breadcrumbs>
          <Group gap="sm">
            <ThemeIcon size={40} radius="md" variant="light" style={{ backgroundColor: 'var(--color-warning)15', color: 'var(--color-warning)' }}>
              <IconDatabase size={20} />
            </ThemeIcon>
            <div>
              <Title order={2} style={{ color: 'var(--color-foreground)' }}>{datasource.name}</Title>
              <Text size="sm" style={{ color: 'var(--color-foreground-muted)' }}>{datasource.engine} - {datasource.database}</Text>
            </div>
          </Group>
        </div>
        <Group gap="sm">
          <TextInput
            placeholder="Search tables..."
            leftSection={<IconSearch size={16} style={{ color: 'var(--color-foreground-subtle)' }} />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            w={250}
          />
          {schemas.length > 1 && (
            <Select
              placeholder="All schemas"
              data={[
                { value: '', label: 'All schemas' },
                ...schemas.map(s => ({ value: s, label: s }))
              ]}
              value={selectedSchema || ''}
              onChange={(v) => setSelectedSchema(v || null)}
              w={150}
              clearable
            />
          )}
        </Group>
      </Group>

      {/* Stats */}
      <SimpleGrid cols={{ base: 2, sm: 4 }} mb="xl">
        <Card withBorder p="md">
          <Text size="sm" c="dimmed">Tables</Text>
          <Text size="xl" fw={700}>{tables?.length || 0}</Text>
        </Card>
        <Card withBorder p="md">
          <Text size="sm" c="dimmed">Schemas</Text>
          <Text size="xl" fw={700}>{schemas.length}</Text>
        </Card>
        <Card withBorder p="md">
          <Text size="sm" c="dimmed">Status</Text>
          <Badge color="green" size="lg" mt={4}>Connected</Badge>
        </Card>
        <Card withBorder p="md">
          <Text size="sm" c="dimmed">Last Sync</Text>
          <Text size="sm" mt={4}>
            {datasource.last_sync_at
              ? new Date(datasource.last_sync_at).toLocaleDateString()
              : 'Never'}
          </Text>
        </Card>
      </SimpleGrid>

      {/* Tables List */}
      {filteredTables.length > 0 ? (
        <Paper withBorder radius="md">
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Table</Table.Th>
                <Table.Th>Schema</Table.Th>
                <Table.Th>Rows</Table.Th>
                <Table.Th>Visibility</Table.Th>
                <Table.Th w={100}></Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filteredTables.map((table) => (
                <Table.Tr
                  key={table.id}
                  style={{ cursor: 'pointer' }}
                  onClick={() => navigate(`/browse/database/${datasourceId}/table/${table.id}`)}
                >
                  <Table.Td>
                    <Group gap="sm">
                      <ThemeIcon size="sm" variant="light" color="brand">
                        <IconTable size={14} />
                      </ThemeIcon>
                      <div>
                        <Text fw={500}>{table.display_name || table.name}</Text>
                        {table.display_name && table.display_name !== table.name && (
                          <Text size="xs" c="dimmed">{table.name}</Text>
                        )}
                      </div>
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Badge variant="light" size="sm">{table.schema || 'default'}</Badge>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm">{table.row_count?.toLocaleString() || '-'}</Text>
                  </Table.Td>
                  <Table.Td>
                    {table.visible !== false ? (
                      <Badge color="green" variant="light" size="sm">Visible</Badge>
                    ) : (
                      <Badge color="gray" variant="light" size="sm">Hidden</Badge>
                    )}
                  </Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      <Tooltip label="Preview data">
                        <ActionIcon
                          variant="subtle"
                          onClick={(e) => {
                            e.stopPropagation()
                            navigate(`/browse/database/${datasourceId}/table/${table.id}`)
                          }}
                        >
                          <IconEye size={16} />
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label="Create question">
                        <ActionIcon
                          variant="subtle"
                          onClick={(e) => {
                            e.stopPropagation()
                            navigate(`/question/new?datasource=${datasourceId}&table=${table.id}`)
                          }}
                        >
                          <IconChartBar size={16} />
                        </ActionIcon>
                      </Tooltip>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Paper>
      ) : (
        <EmptyState
          icon={<IconTable size={32} strokeWidth={1.5} />}
          iconColor="var(--color-foreground-muted)"
          title={search ? 'No tables found' : 'No tables in this database'}
          description={
            search
              ? 'Try adjusting your search terms'
              : 'Sync the database to discover tables'
          }
        />
      )}
    </PageContainer>
  )
}

// Table Preview Page Component
function TablePreviewPage({
  datasourceId,
  tableId,
  datasourceName,
  onBack
}: {
  datasourceId: string
  tableId: string
  datasourceName: string
  onBack: () => void
}) {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(25)
  const [sortColumn, setSortColumn] = useState<string | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')
  const [activeTab, setActiveTab] = useState<string | null>('data')

  // Fetch table metadata
  const { data: tables } = useTables(datasourceId)
  const table = tables?.find(t => t.id === tableId)
  const { data: columns, isLoading: loadingColumns } = useColumns(datasourceId, tableId)
  const syncTable = useSyncTable()

  // Fetch preview data
  const orderBy = sortColumn ? [{ column: sortColumn, direction: sortDirection }] : []
  const { data: previewData, isLoading: loadingPreview, refetch } = useTablePreview({
    datasourceId,
    tableId,
    page,
    pageSize,
    orderBy
  })

  const handleSort = (column: string) => {
    if (sortColumn === column) {
      setSortDirection(d => d === 'asc' ? 'desc' : 'asc')
    } else {
      setSortColumn(column)
      setSortDirection('asc')
    }
    setPage(1)
  }

  const handleSync = async () => {
    await syncTable.mutateAsync({ datasourceId, tableId })
    refetch()
  }

  if (!table) {
    return (
      <PageContainer>
        <EmptyState
          icon={<IconTable size={32} strokeWidth={1.5} />}
          iconColor="var(--color-foreground-muted)"
          title="Table not found"
          description="The requested table could not be found."
          action={<Button onClick={onBack}>Back</Button>}
        />
      </PageContainer>
    )
  }

  return (
    <PageContainer>
      {/* Header */}
      <Group justify="space-between" mb="lg">
        <div>
          <Breadcrumbs mb="xs">
            <Anchor onClick={() => navigate('/browse/databases')}>Databases</Anchor>
            <Anchor onClick={onBack}>{datasourceName}</Anchor>
            <Text fw={600}>{table.display_name || table.name}</Text>
          </Breadcrumbs>
          <Group gap="sm">
            <ActionIcon variant="subtle" onClick={onBack}>
              <IconArrowLeft size={20} />
            </ActionIcon>
            <ThemeIcon size={40} radius="md" variant="light" color="brand">
              <IconTable size={20} />
            </ThemeIcon>
            <div>
              <Title order={2}>{table.display_name || table.name}</Title>
              <Group gap="xs">
                <Badge variant="light" size="sm">{table.schema || 'default'}</Badge>
                <Text size="sm" c="dimmed">{table.row_count?.toLocaleString() || 0} rows</Text>
              </Group>
            </div>
          </Group>
        </div>
        <Group gap="sm">
          <Button
            variant="light"
            leftSection={<IconRefresh size={16} />}
            onClick={handleSync}
            loading={syncTable.isPending}
          >
            Sync
          </Button>
          <Button
            leftSection={<IconChartBar size={16} />}
            onClick={() => navigate(`/question/new?datasource=${datasourceId}&table=${tableId}`)}
          >
            New Question
          </Button>
        </Group>
      </Group>

      {/* Tabs */}
      <Tabs value={activeTab} onChange={setActiveTab} mb="lg">
        <Tabs.List>
          <Tabs.Tab value="data" leftSection={<IconTable size={14} />}>
            Data Preview
          </Tabs.Tab>
          <Tabs.Tab value="columns" leftSection={<IconColumns size={14} />}>
            Columns ({columns?.length || 0})
          </Tabs.Tab>
        </Tabs.List>
      </Tabs>

      {activeTab === 'data' && (
        <Paper withBorder radius="md">
          {/* Toolbar */}
          <Group justify="space-between" p="md" style={{ borderBottom: '1px solid var(--mantine-color-gray-3)' }}>
            <Text size="sm" c="dimmed">
              {previewData ? (
                <>
                  Showing {((page - 1) * pageSize) + 1}-{Math.min(page * pageSize, previewData.total_rows || previewData.row_count)} of {previewData.total_rows || previewData.row_count} rows
                </>
              ) : (
                'Loading...'
              )}
            </Text>
            <Group gap="sm">
              <Select
                size="xs"
                value={String(pageSize)}
                onChange={(v) => { setPageSize(Number(v)); setPage(1) }}
                data={[
                  { value: '10', label: '10 rows' },
                  { value: '25', label: '25 rows' },
                  { value: '50', label: '50 rows' },
                  { value: '100', label: '100 rows' },
                ]}
                w={100}
              />
            </Group>
          </Group>

          {/* Data Table */}
          <ScrollArea>
            {loadingPreview ? (
              <Box p="xl" ta="center">
                <Loader size="lg" />
              </Box>
            ) : previewData && previewData.rows.length > 0 ? (
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    {previewData.columns.map((col) => (
                      <Table.Th
                        key={col.name}
                        style={{ cursor: 'pointer', whiteSpace: 'nowrap' }}
                        onClick={() => handleSort(col.name)}
                      >
                        <Group gap={4} wrap="nowrap">
                          <Text size="sm" fw={500}>{col.display_name || col.name}</Text>
                          {sortColumn === col.name && (
                            sortDirection === 'asc'
                              ? <IconSortAscending size={14} />
                              : <IconSortDescending size={14} />
                          )}
                        </Group>
                      </Table.Th>
                    ))}
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {previewData.rows.map((row, i) => (
                    <Table.Tr key={i}>
                      {previewData.columns.map((col) => (
                        <Table.Td key={col.name}>
                          <Text size="sm" lineClamp={1}>
                            {row[col.name] === null ? (
                              <Text span c="dimmed" fs="italic">null</Text>
                            ) : (
                              String(row[col.name])
                            )}
                          </Text>
                        </Table.Td>
                      ))}
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            ) : (
              <Box p="xl" ta="center">
                <Text c="dimmed">No data in this table</Text>
              </Box>
            )}
          </ScrollArea>

          {/* Pagination */}
          {previewData && (previewData.total_pages || 1) > 1 && (
            <Group justify="center" p="md" style={{ borderTop: '1px solid var(--mantine-color-gray-3)' }}>
              <Pagination
                value={page}
                onChange={setPage}
                total={previewData.total_pages || 1}
                size="sm"
              />
            </Group>
          )}
        </Paper>
      )}

      {activeTab === 'columns' && (
        <Paper withBorder radius="md">
          {loadingColumns ? (
            <Box p="xl" ta="center">
              <Loader size="lg" />
            </Box>
          ) : columns && columns.length > 0 ? (
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Column</Table.Th>
                  <Table.Th>Type</Table.Th>
                  <Table.Th>Semantic</Table.Th>
                  <Table.Th>Nullable</Table.Th>
                  <Table.Th>Visibility</Table.Th>
                  <Table.Th>Stats</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {columns.map((col) => (
                  <Table.Tr key={col.id}>
                    <Table.Td>
                      <Group gap="xs">
                        {col.primary_key && (
                          <Badge size="xs" color="yellow" variant="filled">PK</Badge>
                        )}
                        {col.foreign_key && (
                          <Badge size="xs" color="blue" variant="filled">FK</Badge>
                        )}
                        <div>
                          <Text fw={500}>{col.display_name || col.name}</Text>
                          {col.display_name && col.display_name !== col.name && (
                            <Text size="xs" c="dimmed">{col.name}</Text>
                          )}
                        </div>
                      </Group>
                    </Table.Td>
                    <Table.Td>
                      <Badge variant="light" size="sm">{col.type}</Badge>
                    </Table.Td>
                    <Table.Td>
                      {col.semantic ? (
                        <Badge variant="outline" size="sm">{col.semantic.replace('type/', '')}</Badge>
                      ) : (
                        <Text size="sm" c="dimmed">-</Text>
                      )}
                    </Table.Td>
                    <Table.Td>
                      {col.nullable ? (
                        <Badge color="gray" variant="light" size="sm">Yes</Badge>
                      ) : (
                        <Badge color="blue" variant="light" size="sm">No</Badge>
                      )}
                    </Table.Td>
                    <Table.Td>
                      {col.visibility === 'hidden' ? (
                        <Group gap={4}>
                          <IconEyeOff size={14} />
                          <Text size="sm">Hidden</Text>
                        </Group>
                      ) : (
                        <Group gap={4}>
                          <IconEye size={14} />
                          <Text size="sm">Visible</Text>
                        </Group>
                      )}
                    </Table.Td>
                    <Table.Td>
                      <Group gap="xs">
                        {col.distinct_count != null && (
                          <Tooltip label="Distinct values">
                            <Badge variant="light" size="xs">{col.distinct_count} unique</Badge>
                          </Tooltip>
                        )}
                        {col.null_count != null && col.null_count > 0 && (
                          <Tooltip label="Null values">
                            <Badge variant="light" color="orange" size="xs">{col.null_count} nulls</Badge>
                          </Tooltip>
                        )}
                      </Group>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          ) : (
            <Box p="xl" ta="center">
              <Text c="dimmed">No column information available</Text>
            </Box>
          )}
        </Paper>
      )}
    </PageContainer>
  )
}
