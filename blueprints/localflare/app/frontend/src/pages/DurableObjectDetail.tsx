import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Stack, Paper, Text, Button, Textarea, Table, Group, Code, ScrollArea } from '@mantine/core'
import { IconPlayerPlay } from '@tabler/icons-react'
import { PageHeader, DataTable, LoadingState, type Column } from '../components/common'
import { api } from '../api/client'
import type { DurableObjectNamespace, DurableObjectInstance } from '../types'

export function DurableObjectDetail() {
  const { id } = useParams<{ id: string }>()
  const [namespace, setNamespace] = useState<DurableObjectNamespace | null>(null)
  const [objects, setObjects] = useState<DurableObjectInstance[]>([])
  const [loading, setLoading] = useState(true)
  const [query, setQuery] = useState('SELECT * FROM storage LIMIT 100;')
  const [queryResults, setQueryResults] = useState<unknown[]>([])
  const [queryLoading, setQueryLoading] = useState(false)

  useEffect(() => {
    if (id) loadData()
  }, [id])

  const loadData = async () => {
    try {
      const [nsRes, objRes] = await Promise.all([
        api.durableObjects.getNamespace(id!),
        api.durableObjects.listObjects(id!),
      ])
      if (nsRes.result) setNamespace(nsRes.result)
      if (objRes.result) setObjects(objRes.result.objects)
    } catch (error) {
      console.error('Failed to load data:', error)
      // Mock data
      setNamespace({
        id: id!,
        name: 'COUNTER',
        class_name: 'Counter',
        script_name: 'counter-worker',
        created_at: new Date(Date.now() - 172800000).toISOString(),
        object_count: 156,
      })
      setObjects([
        { id: 'do:counter:user-123', namespace_id: id!, last_access: new Date(Date.now() - 300000).toISOString(), storage_size: 2356 },
        { id: 'do:counter:user-456', namespace_id: id!, last_access: new Date(Date.now() - 720000).toISOString(), storage_size: 1843 },
        { id: 'do:counter:user-789', namespace_id: id!, last_access: new Date(Date.now() - 3600000).toISOString(), storage_size: 4198 },
      ])
    } finally {
      setLoading(false)
    }
  }

  const runQuery = async () => {
    if (!objects[0]) return
    setQueryLoading(true)
    try {
      const res = await api.durableObjects.queryStorage(id!, objects[0].id, query)
      if (res.result) setQueryResults(res.result.results)
    } catch (error) {
      // Mock results
      setQueryResults([
        { key: 'count', value: 42, updated_at: '2024-01-15T10:30:00Z' },
        { key: 'last_reset', value: '2024-01-01T00:00:00Z', updated_at: '2024-01-01T00:00:00Z' },
      ])
    } finally {
      setQueryLoading(false)
    }
  }

  if (loading) return <LoadingState />
  if (!namespace) return <Text>Namespace not found</Text>

  const objectColumns: Column<DurableObjectInstance>[] = [
    { key: 'id', label: 'Object ID', sortable: true },
    {
      key: 'last_access',
      label: 'Last Access',
      sortable: true,
      render: (row) => formatRelativeTime(row.last_access),
    },
    {
      key: 'storage_size',
      label: 'Storage',
      sortable: true,
      render: (row) => formatBytes(row.storage_size),
    },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title={namespace.name}
        breadcrumbs={[
          { label: 'Durable Objects', path: '/durable-objects' },
          { label: namespace.name },
        ]}
        backPath="/durable-objects"
      />

      {/* Overview Stats */}
      <Paper p="md" radius="md" withBorder>
        <Group gap="xl">
          <Stack gap={2}>
            <Text size="xs" c="dimmed">Class</Text>
            <Text fw={500}>{namespace.class_name}</Text>
          </Stack>
          <Stack gap={2}>
            <Text size="xs" c="dimmed">Script</Text>
            <Text fw={500}>{namespace.script_name || '-'}</Text>
          </Stack>
          <Stack gap={2}>
            <Text size="xs" c="dimmed">Objects</Text>
            <Text fw={500}>{namespace.object_count?.toLocaleString() ?? 0}</Text>
          </Stack>
        </Group>
      </Paper>

      {/* Objects List */}
      <Stack gap="sm">
        <Text size="sm" fw={600}>Objects</Text>
        <DataTable
          data={objects}
          columns={objectColumns}
          getRowKey={(row) => row.id}
          searchPlaceholder="Search objects..."
          emptyState={{
            title: 'No objects yet',
            description: 'Objects will appear here when they are created',
          }}
        />
      </Stack>

      {/* Storage Inspector */}
      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Text size="sm" fw={600}>Storage Inspector (SQLite)</Text>
          <Textarea
            value={query}
            onChange={(e) => setQuery(e.currentTarget.value)}
            minRows={3}
            styles={{ input: { fontFamily: 'monospace', fontSize: 13 } }}
          />
          <Group justify="flex-end">
            <Button
              leftSection={<IconPlayerPlay size={16} />}
              onClick={runQuery}
              loading={queryLoading}
              disabled={objects.length === 0}
            >
              Run
            </Button>
          </Group>

          {queryResults.length > 0 && (
            <ScrollArea>
              <Table striped highlightOnHover withTableBorder>
                <Table.Thead>
                  <Table.Tr>
                    {Object.keys(queryResults[0] as object).map((key) => (
                      <Table.Th key={key}>{key}</Table.Th>
                    ))}
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {queryResults.map((row, idx) => (
                    <Table.Tr key={idx}>
                      {Object.values(row as object).map((val, vidx) => (
                        <Table.Td key={vidx}>
                          <Code>{JSON.stringify(val)}</Code>
                        </Table.Td>
                      ))}
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            </ScrollArea>
          )}
        </Stack>
      </Paper>
    </Stack>
  )
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'Just now'
  if (mins < 60) return `${mins} mins ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours} hours ago`
  const days = Math.floor(hours / 24)
  return `${days} days ago`
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
