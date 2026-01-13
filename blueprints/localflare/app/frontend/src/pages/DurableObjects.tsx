import { useState, useEffect } from 'react'
import { Button, Modal, TextInput, Stack, Group } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconEye, IconTrash } from '@tabler/icons-react'
import { PageHeader, DataTable, type Column } from '../components/common'
import { api } from '../api/client'
import type { DurableObjectNamespace } from '../types'

export function DurableObjects() {
  const navigate = useNavigate()
  const [namespaces, setNamespaces] = useState<DurableObjectNamespace[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const form = useForm({
    initialValues: {
      name: '',
      class_name: '',
      script_name: '',
    },
    validate: {
      name: (v) => (!v ? 'Name is required' : null),
      class_name: (v) => (!v ? 'Class name is required' : null),
    },
  })

  useEffect(() => {
    loadNamespaces()
  }, [])

  const loadNamespaces = async () => {
    try {
      const res = await api.durableObjects.listNamespaces()
      if (res.result) setNamespaces(res.result.namespaces ?? [])
    } catch (error) {
      console.error('Failed to load namespaces:', error)
      // Mock data
      setNamespaces([
        { id: '1', name: 'COUNTER', class_name: 'Counter', script_name: 'counter-worker', created_at: new Date(Date.now() - 172800000).toISOString(), object_count: 156 },
        { id: '2', name: 'USER_SESSION', class_name: 'Session', script_name: 'session-worker', created_at: new Date(Date.now() - 604800000).toISOString(), object_count: 1234 },
        { id: '3', name: 'RATE_LIMITER', class_name: 'Limiter', script_name: 'limiter-worker', created_at: new Date(Date.now() - 259200000).toISOString(), object_count: 89 },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: typeof form.values) => {
    try {
      await api.durableObjects.createNamespace(values)
      notifications.show({ title: 'Success', message: 'Namespace created', color: 'green' })
      setCreateModalOpen(false)
      form.reset()
      loadNamespaces()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create namespace', color: 'red' })
    }
  }

  const handleDelete = async (ns: DurableObjectNamespace) => {
    if (!confirm(`Delete namespace "${ns.name}"?`)) return
    try {
      await api.durableObjects.deleteNamespace(ns.id)
      notifications.show({ title: 'Success', message: 'Namespace deleted', color: 'green' })
      loadNamespaces()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete namespace', color: 'red' })
    }
  }

  const columns: Column<DurableObjectNamespace>[] = [
    { key: 'name', label: 'Name', sortable: true },
    { key: 'class_name', label: 'Class', sortable: true },
    {
      key: 'object_count',
      label: 'Objects',
      sortable: true,
      render: (row) => (row.object_count ?? 0).toLocaleString(),
    },
    {
      key: 'created_at',
      label: 'Created',
      sortable: true,
      render: (row) => formatDate(row.created_at),
    },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="Durable Objects"
        subtitle="Manage Durable Object namespaces and instances"
        actions={
          <Button leftSection={<IconPlus size={16} />} onClick={() => setCreateModalOpen(true)}>
            Create Namespace
          </Button>
        }
      />

      <DataTable
        data={namespaces}
        columns={columns}
        loading={loading}
        getRowKey={(row) => row.id}
        searchPlaceholder="Search namespaces..."
        onRowClick={(row) => navigate(`/durable-objects/${row.id}`)}
        actions={[
          { label: 'View', icon: <IconEye size={14} />, onClick: (row) => navigate(`/durable-objects/${row.id}`) },
          { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
        ]}
        emptyState={{
          title: 'No namespaces yet',
          description: 'Create your first Durable Object namespace to get started',
          action: { label: 'Create Namespace', onClick: () => setCreateModalOpen(true) },
        }}
      />

      {/* Create Modal */}
      <Modal
        opened={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        title="Create Durable Object Namespace"
        size="md"
      >
        <form onSubmit={form.onSubmit(handleCreate)}>
          <Stack gap="md">
            <TextInput
              label="Name"
              placeholder="MY_NAMESPACE"
              required
              {...form.getInputProps('name')}
            />
            <TextInput
              label="Class Name"
              placeholder="MyDurableObject"
              required
              {...form.getInputProps('class_name')}
            />
            <TextInput
              label="Script (optional)"
              placeholder="my-worker"
              {...form.getInputProps('script_name')}
            />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateModalOpen(false)}>
                Cancel
              </Button>
              <Button type="submit">Create Namespace</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const days = Math.floor(diff / 86400000)
  if (days === 0) return 'Today'
  if (days === 1) return 'Yesterday'
  if (days < 7) return `${days} days ago`
  if (days < 30) return `${Math.floor(days / 7)} weeks ago`
  return date.toLocaleDateString()
}
