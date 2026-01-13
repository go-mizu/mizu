import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Stack, SimpleGrid, Paper, Text, Group, Button, Tabs, Badge, Code, Textarea, Table, ActionIcon, Modal, TextInput, Select } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconRocket, IconHistory, IconPlus, IconTrash, IconRefresh } from '@tabler/icons-react'
import { PageHeader, StatCard, AreaChart, DataTable, LoadingState, StatusBadge, type Column } from '../components/common'
import { api } from '../api/client'
import type { Worker, WorkerVersion, WorkerBinding, TimeSeriesData } from '../types'

export function WorkerDetail() {
  const { id } = useParams<{ id: string }>()
  const [worker, setWorker] = useState<Worker | null>(null)
  const [versions, setVersions] = useState<WorkerVersion[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<string | null>('overview')
  const [timeSeries, setTimeSeries] = useState<TimeSeriesData[]>([])
  const [timeRange, setTimeRange] = useState<'1h' | '24h' | '7d' | '30d'>('24h')
  const [bindingModalOpen, setBindingModalOpen] = useState(false)

  const bindingForm = useForm({
    initialValues: {
      type: 'kv_namespace',
      name: '',
      namespace_id: '',
    },
  })

  useEffect(() => {
    if (id) loadWorker()
  }, [id])

  useEffect(() => {
    const now = Date.now()
    const points = timeRange === '1h' ? 60 : timeRange === '24h' ? 24 : 7
    const interval = timeRange === '1h' ? 60000 : timeRange === '24h' ? 3600000 : 86400000
    setTimeSeries(
      Array.from({ length: points }, (_, i) => ({
        timestamp: new Date(now - (points - i) * interval).toISOString(),
        value: Math.floor(Math.random() * 500) + 100,
      }))
    )
  }, [timeRange])

  const loadWorker = async () => {
    try {
      const [workerRes, versionsRes] = await Promise.all([
        api.workers.get(id!),
        api.workers.getVersions(id!),
      ])
      if (workerRes.result) setWorker(workerRes.result)
      if (versionsRes.result) setVersions(versionsRes.result.versions ?? [])
    } catch (error) {
      setWorker({
        id: '1',
        name: id!,
        created_at: new Date(Date.now() - 172800000).toISOString(),
        modified_at: new Date(Date.now() - 3600000).toISOString(),
        status: 'active',
        routes: ['api.example.com/*', 'api.example.com/v2/*'],
        bindings: [
          { type: 'kv_namespace', name: 'CACHE', namespace_id: 'kv-123' },
          { type: 'd1', name: 'DB', database_id: 'd1-456' },
          { type: 'r2_bucket', name: 'STORAGE', bucket_name: 'my-bucket' },
        ],
        environment_variables: {
          API_KEY: '***hidden***',
          DEBUG: 'false',
        },
        compatibility_date: '2024-01-01',
        usage_model: 'bundled',
        code: `export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);

    // Check cache first
    const cached = await env.CACHE.get(url.pathname);
    if (cached) {
      return new Response(cached, {
        headers: { 'Content-Type': 'application/json' }
      });
    }

    // Query database
    const result = await env.DB.prepare(
      'SELECT * FROM items WHERE path = ?'
    ).bind(url.pathname).first();

    return Response.json(result || { error: 'Not found' });
  },
};`,
      })
      setVersions([
        { id: 'v3', version: 'v3', created_at: new Date(Date.now() - 3600000).toISOString(), status: 'active', message: 'Added caching layer' },
        { id: 'v2', version: 'v2', created_at: new Date(Date.now() - 86400000).toISOString(), status: 'inactive', message: 'Database integration' },
        { id: 'v1', version: 'v1', created_at: new Date(Date.now() - 172800000).toISOString(), status: 'inactive', message: 'Initial deployment' },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleDeploy = async () => {
    try {
      await api.workers.deploy(id!)
      notifications.show({ title: 'Success', message: 'Worker deployed', color: 'green' })
      loadWorker()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to deploy', color: 'red' })
    }
  }

  const handleRollback = async (version: WorkerVersion) => {
    if (!confirm(`Rollback to ${version.version}?`)) return
    try {
      await api.workers.rollback(id!, version.id)
      notifications.show({ title: 'Success', message: `Rolled back to ${version.version}`, color: 'green' })
      loadWorker()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to rollback', color: 'red' })
    }
  }

  const handleAddBinding = async (values: typeof bindingForm.values) => {
    try {
      await api.workers.addBinding(id!, { type: values.type as WorkerBinding['type'], name: values.name, namespace_id: values.namespace_id })
      notifications.show({ title: 'Success', message: 'Binding added', color: 'green' })
      setBindingModalOpen(false)
      bindingForm.reset()
      loadWorker()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to add binding', color: 'red' })
    }
  }

  const handleRemoveBinding = async (binding: WorkerBinding) => {
    if (!confirm(`Remove binding "${binding.name}"?`)) return
    try {
      await api.workers.removeBinding(id!, binding.name)
      notifications.show({ title: 'Success', message: 'Binding removed', color: 'green' })
      loadWorker()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to remove binding', color: 'red' })
    }
  }

  if (loading) return <LoadingState />
  if (!worker) return <Text>Worker not found</Text>

  const versionColumns: Column<WorkerVersion>[] = [
    { key: 'version', label: 'Version', render: (row) => <Code>{row.version}</Code> },
    { key: 'message', label: 'Message' },
    { key: 'created_at', label: 'Deployed', render: (row) => new Date(row.created_at).toLocaleString() },
    { key: 'status', label: 'Status', render: (row) => <StatusBadge status={row.status} /> },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title={worker.name}
        breadcrumbs={[{ label: 'Workers', path: '/workers' }, { label: worker.name }]}
        backPath="/workers"
        actions={
          <Group>
            <Button variant="light" leftSection={<IconHistory size={16} />} onClick={() => setActiveTab('versions')}>
              Versions
            </Button>
            <Button leftSection={<IconRocket size={16} />} onClick={handleDeploy}>
              Deploy
            </Button>
          </Group>
        }
      />

      <Tabs value={activeTab} onChange={setActiveTab}>
        <Tabs.List>
          <Tabs.Tab value="overview">Overview</Tabs.Tab>
          <Tabs.Tab value="code">Code</Tabs.Tab>
          <Tabs.Tab value="bindings">Bindings</Tabs.Tab>
          <Tabs.Tab value="versions">Versions</Tabs.Tab>
          <Tabs.Tab value="settings">Settings</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="overview" pt="md">
          <Stack gap="md">
            <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
              <StatCard icon={<Text size="sm" fw={700}>R</Text>} label="Requests" value="12.4K" description="Today" color="orange" />
              <StatCard icon={<Text size="sm" fw={700}>E</Text>} label="Errors" value="23" description="Today" color="error" />
              <StatCard icon={<Text size="sm" fw={700}>L</Text>} label="Avg Latency" value="45ms" />
              <StatCard icon={<Text size="sm" fw={700}>CPU</Text>} label="CPU Time" value="2.3ms" description="avg" />
            </SimpleGrid>

            <AreaChart
              data={timeSeries}
              title="Request Volume"
              timeRange={timeRange}
              onTimeRangeChange={setTimeRange}
              height={200}
            />

            <SimpleGrid cols={{ base: 1, md: 2 }} spacing="md">
              <Paper p="md" radius="md" withBorder>
                <Stack gap="md">
                  <Text size="sm" fw={600}>Routes</Text>
                  <Stack gap="xs">
                    {worker.routes?.map((route) => (
                      <Group key={route} justify="space-between">
                        <Code>{route}</Code>
                        <StatusBadge status="active" />
                      </Group>
                    ))}
                    {(!worker.routes || worker.routes.length === 0) && (
                      <Text size="sm" c="dimmed">No routes configured</Text>
                    )}
                  </Stack>
                </Stack>
              </Paper>

              <Paper p="md" radius="md" withBorder>
                <Stack gap="md">
                  <Group justify="space-between">
                    <Text size="sm" fw={600}>Bindings</Text>
                    <Button size="xs" variant="light" leftSection={<IconPlus size={12} />} onClick={() => setBindingModalOpen(true)}>
                      Add
                    </Button>
                  </Group>
                  <Stack gap="xs">
                    {worker.bindings?.map((binding) => (
                      <Group key={binding.name} justify="space-between">
                        <Group gap="xs">
                          <Badge size="sm" color={getBindingColor(binding.type)}>{binding.type}</Badge>
                          <Code>{binding.name}</Code>
                        </Group>
                        <ActionIcon size="sm" variant="subtle" color="red" onClick={() => handleRemoveBinding(binding)}>
                          <IconTrash size={12} />
                        </ActionIcon>
                      </Group>
                    ))}
                    {(!worker.bindings || worker.bindings.length === 0) && (
                      <Text size="sm" c="dimmed">No bindings configured</Text>
                    )}
                  </Stack>
                </Stack>
              </Paper>
            </SimpleGrid>
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="code" pt="md">
          <Paper p="md" radius="md" withBorder>
            <Stack gap="md">
              <Group justify="space-between">
                <Text size="sm" fw={600}>Worker Code</Text>
                <Button size="xs" variant="light" leftSection={<IconRefresh size={12} />}>
                  Save & Deploy
                </Button>
              </Group>
              <Textarea
                value={worker.code || '// No code available'}
                minRows={20}
                styles={{ input: { fontFamily: 'monospace', fontSize: 12 } }}
                readOnly
              />
            </Stack>
          </Paper>
        </Tabs.Panel>

        <Tabs.Panel value="bindings" pt="md">
          <Stack gap="md">
            <Group justify="space-between">
              <Text size="sm" fw={600}>Resource Bindings</Text>
              <Button size="sm" leftSection={<IconPlus size={14} />} onClick={() => setBindingModalOpen(true)}>
                Add Binding
              </Button>
            </Group>
            <Table striped highlightOnHover withTableBorder>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Type</Table.Th>
                  <Table.Th>Variable Name</Table.Th>
                  <Table.Th>Resource</Table.Th>
                  <Table.Th>Actions</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {worker.bindings?.map((binding) => (
                  <Table.Tr key={binding.name}>
                    <Table.Td><Badge color={getBindingColor(binding.type)}>{binding.type}</Badge></Table.Td>
                    <Table.Td><Code>{binding.name}</Code></Table.Td>
                    <Table.Td><Code>{binding.namespace_id || binding.database_id || binding.bucket_name || '-'}</Code></Table.Td>
                    <Table.Td>
                      <ActionIcon variant="subtle" color="red" onClick={() => handleRemoveBinding(binding)}>
                        <IconTrash size={14} />
                      </ActionIcon>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
            {(!worker.bindings || worker.bindings.length === 0) && (
              <Text size="sm" c="dimmed" ta="center" py="xl">No bindings configured. Add a binding to connect your Worker to other resources.</Text>
            )}
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="versions" pt="md">
          <DataTable
            data={versions}
            columns={versionColumns}
            getRowKey={(row) => row.id}
            searchable={false}
            actions={[
              { label: 'Rollback', icon: <IconHistory size={14} />, onClick: handleRollback },
            ]}
          />
        </Tabs.Panel>

        <Tabs.Panel value="settings" pt="md">
          <Stack gap="md">
            <Paper p="md" radius="md" withBorder>
              <Stack gap="md">
                <Text size="sm" fw={600}>Environment Variables</Text>
                <Table striped withTableBorder>
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>Name</Table.Th>
                      <Table.Th>Value</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {worker.environment_variables && Object.entries(worker.environment_variables).map(([key, value]) => (
                      <Table.Tr key={key}>
                        <Table.Td><Code>{key}</Code></Table.Td>
                        <Table.Td><Code>{value}</Code></Table.Td>
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
              </Stack>
            </Paper>

            <Paper p="md" radius="md" withBorder>
              <Stack gap="md">
                <Text size="sm" fw={600}>Compatibility Settings</Text>
                <Group gap="xl">
                  <Stack gap={2}>
                    <Text size="xs" c="dimmed">Compatibility Date</Text>
                    <Text fw={500}>{worker.compatibility_date}</Text>
                  </Stack>
                  <Stack gap={2}>
                    <Text size="xs" c="dimmed">Usage Model</Text>
                    <Text fw={500}>{worker.usage_model || 'bundled'}</Text>
                  </Stack>
                </Group>
              </Stack>
            </Paper>
          </Stack>
        </Tabs.Panel>
      </Tabs>

      <Modal opened={bindingModalOpen} onClose={() => setBindingModalOpen(false)} title="Add Binding">
        <form onSubmit={bindingForm.onSubmit(handleAddBinding)}>
          <Stack gap="md">
            <Select
              label="Binding Type"
              data={[
                { value: 'kv_namespace', label: 'KV Namespace' },
                { value: 'd1', label: 'D1 Database' },
                { value: 'r2_bucket', label: 'R2 Bucket' },
                { value: 'durable_object', label: 'Durable Object' },
                { value: 'queue', label: 'Queue' },
                { value: 'vectorize', label: 'Vectorize Index' },
                { value: 'ai', label: 'Workers AI' },
              ]}
              {...bindingForm.getInputProps('type')}
            />
            <TextInput
              label="Variable Name"
              placeholder="MY_BINDING"
              description="The name to access this binding in your Worker code"
              {...bindingForm.getInputProps('name')}
            />
            <TextInput
              label="Resource ID"
              placeholder="namespace-id or database-id"
              {...bindingForm.getInputProps('namespace_id')}
            />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setBindingModalOpen(false)}>Cancel</Button>
              <Button type="submit">Add Binding</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}

function getBindingColor(type: string): string {
  switch (type) {
    case 'kv_namespace': return 'blue'
    case 'd1': return 'grape'
    case 'r2_bucket': return 'orange'
    case 'durable_object': return 'teal'
    case 'queue': return 'pink'
    case 'vectorize': return 'cyan'
    case 'ai': return 'violet'
    default: return 'gray'
  }
}
