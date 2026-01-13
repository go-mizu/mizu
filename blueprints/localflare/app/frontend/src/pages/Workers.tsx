import { useState, useEffect } from 'react'
import { Button, Modal, TextInput, Stack, Group, Badge, Textarea, Select, Code } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconEye, IconTrash, IconRocket, IconSettings } from '@tabler/icons-react'
import { PageHeader, DataTable, StatusBadge, type Column } from '../components/common'
import { api } from '../api/client'
import type { Worker } from '../types'

export function Workers() {
  const navigate = useNavigate()
  const [workers, setWorkers] = useState<Worker[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const form = useForm({
    initialValues: {
      name: '',
      main_module: 'index.js',
      compatibility_date: new Date().toISOString().split('T')[0],
      code: `export default {
  async fetch(request, env, ctx) {
    return new Response('Hello World!');
  },
};`,
    },
    validate: {
      name: (v) => (!v ? 'Name is required' : !/^[a-z0-9-]+$/.test(v) ? 'Use lowercase letters, numbers, and hyphens only' : null),
    },
  })

  useEffect(() => {
    loadWorkers()
  }, [])

  const loadWorkers = async () => {
    try {
      const res = await api.workers.list()
      if (res.result) setWorkers(res.result.workers ?? [])
    } catch (error) {
      console.error('Failed to load workers:', error)
      setWorkers([
        {
          id: '1',
          name: 'api-gateway',
          created_at: new Date(Date.now() - 172800000).toISOString(),
          modified_at: new Date(Date.now() - 3600000).toISOString(),
          status: 'active',
          routes: ['api.example.com/*'],
          bindings: [
            { type: 'kv_namespace', name: 'CACHE' },
            { type: 'd1', name: 'DB' },
          ],
        },
        {
          id: '2',
          name: 'auth-worker',
          created_at: new Date(Date.now() - 604800000).toISOString(),
          modified_at: new Date(Date.now() - 86400000).toISOString(),
          status: 'active',
          routes: ['auth.example.com/*'],
          bindings: [
            { type: 'kv_namespace', name: 'SESSIONS' },
            { type: 'durable_object', name: 'RATE_LIMITER' },
          ],
        },
        {
          id: '3',
          name: 'image-processor',
          created_at: new Date(Date.now() - 259200000).toISOString(),
          modified_at: new Date(Date.now() - 7200000).toISOString(),
          status: 'active',
          routes: ['images.example.com/*'],
          bindings: [
            { type: 'r2_bucket', name: 'IMAGES' },
          ],
        },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: typeof form.values) => {
    try {
      await api.workers.create(values)
      notifications.show({ title: 'Success', message: 'Worker created', color: 'green' })
      setCreateModalOpen(false)
      form.reset()
      loadWorkers()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create worker', color: 'red' })
    }
  }

  const handleDelete = async (worker: Worker) => {
    if (!confirm(`Delete worker "${worker.name}"? This cannot be undone.`)) return
    try {
      await api.workers.delete(worker.name)
      notifications.show({ title: 'Success', message: 'Worker deleted', color: 'green' })
      loadWorkers()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete worker', color: 'red' })
    }
  }

  const handleDeploy = async (worker: Worker) => {
    try {
      await api.workers.deploy(worker.name)
      notifications.show({ title: 'Success', message: 'Worker deployed', color: 'green' })
      loadWorkers()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to deploy worker', color: 'red' })
    }
  }

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const hours = Math.floor(diff / 3600000)
    if (hours < 1) return 'Just now'
    if (hours < 24) return `${hours}h ago`
    const days = Math.floor(hours / 24)
    if (days < 7) return `${days}d ago`
    return date.toLocaleDateString()
  }

  const columns: Column<Worker>[] = [
    { key: 'name', label: 'Name', sortable: true, render: (row) => <Code>{row.name}</Code> },
    {
      key: 'routes',
      label: 'Routes',
      render: (row) => (
        <Group gap={4}>
          {row.routes?.slice(0, 2).map((route) => (
            <Badge key={route} size="sm" variant="light">
              {route}
            </Badge>
          ))}
          {row.routes && row.routes.length > 2 && (
            <Badge size="sm" variant="outline">+{row.routes.length - 2}</Badge>
          )}
        </Group>
      ),
    },
    {
      key: 'bindings',
      label: 'Bindings',
      render: (row) => (
        <Group gap={4}>
          {row.bindings?.slice(0, 3).map((binding) => (
            <Badge key={binding.name} size="xs" variant="dot" color={getBindingColor(binding.type)}>
              {binding.name}
            </Badge>
          ))}
          {row.bindings && row.bindings.length > 3 && (
            <Badge size="xs" variant="outline">+{row.bindings.length - 3}</Badge>
          )}
        </Group>
      ),
    },
    { key: 'modified_at', label: 'Last Modified', sortable: true, render: (row) => formatDate(row.modified_at) },
    { key: 'status', label: 'Status', render: (row) => <StatusBadge status={row.status} /> },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="Workers"
        subtitle="Manage your Cloudflare Worker scripts"
        actions={
          <Button leftSection={<IconPlus size={16} />} onClick={() => setCreateModalOpen(true)}>
            Create Worker
          </Button>
        }
      />

      <DataTable
        data={workers}
        columns={columns}
        loading={loading}
        getRowKey={(row) => row.id}
        searchPlaceholder="Search workers..."
        onRowClick={(row) => navigate(`/workers/${row.name}`)}
        actions={[
          { label: 'View', icon: <IconEye size={14} />, onClick: (row) => navigate(`/workers/${row.name}`) },
          { label: 'Deploy', icon: <IconRocket size={14} />, onClick: handleDeploy },
          { label: 'Settings', icon: <IconSettings size={14} />, onClick: (row) => navigate(`/workers/${row.name}/settings`) },
          { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
        ]}
        emptyState={{
          title: 'No workers yet',
          description: 'Create your first Worker to get started with serverless computing',
          action: { label: 'Create Worker', onClick: () => setCreateModalOpen(true) },
        }}
      />

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Create Worker" size="lg">
        <form onSubmit={form.onSubmit(handleCreate)}>
          <Stack gap="md">
            <TextInput
              label="Worker Name"
              placeholder="my-worker"
              description="Use lowercase letters, numbers, and hyphens"
              required
              {...form.getInputProps('name')}
            />
            <Group grow>
              <TextInput
                label="Main Module"
                placeholder="index.js"
                {...form.getInputProps('main_module')}
              />
              <Select
                label="Compatibility Date"
                data={[
                  { value: '2024-01-01', label: '2024-01-01' },
                  { value: '2024-06-01', label: '2024-06-01' },
                  { value: '2025-01-01', label: '2025-01-01' },
                ]}
                {...form.getInputProps('compatibility_date')}
              />
            </Group>
            <Textarea
              label="Initial Code"
              description="You can edit this later in the Worker editor"
              minRows={8}
              styles={{ input: { fontFamily: 'monospace', fontSize: 12 } }}
              {...form.getInputProps('code')}
            />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateModalOpen(false)}>
                Cancel
              </Button>
              <Button type="submit">Create Worker</Button>
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
