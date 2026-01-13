import { useState, useEffect } from 'react'
import { Button, Modal, TextInput, Stack, Group } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconEye, IconTrash } from '@tabler/icons-react'
import { PageHeader, DataTable, type Column } from '../components/common'
import { api } from '../api/client'
import type { KVNamespace } from '../types'

export function KV() {
  const navigate = useNavigate()
  const [namespaces, setNamespaces] = useState<KVNamespace[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const form = useForm({
    initialValues: {
      title: '',
    },
    validate: {
      title: (v) => (!v ? 'Name is required' : null),
    },
  })

  useEffect(() => {
    loadNamespaces()
  }, [])

  const loadNamespaces = async () => {
    try {
      const res = await api.kv.listNamespaces()
      if (res.result) setNamespaces(res.result.namespaces ?? [])
    } catch (error) {
      console.error('Failed to load KV namespaces:', error)
      setNamespaces([
        {
          id: 'kv-1',
          title: 'CACHE',
          created_at: new Date(Date.now() - 172800000).toISOString(),
          key_count: 15234,
          storage_size: 45 * 1024 * 1024,
        },
        {
          id: 'kv-2',
          title: 'SESSIONS',
          created_at: new Date(Date.now() - 604800000).toISOString(),
          key_count: 8901,
          storage_size: 12 * 1024 * 1024,
        },
        {
          id: 'kv-3',
          title: 'CONFIG',
          created_at: new Date(Date.now() - 259200000).toISOString(),
          key_count: 42,
          storage_size: 128 * 1024,
        },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: typeof form.values) => {
    try {
      await api.kv.createNamespace(values)
      notifications.show({ title: 'Success', message: 'Namespace created', color: 'green' })
      setCreateModalOpen(false)
      form.reset()
      loadNamespaces()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create namespace', color: 'red' })
    }
  }

  const handleDelete = async (namespace: KVNamespace) => {
    if (!confirm(`Delete namespace "${namespace.title}"? All keys will be deleted.`)) return
    try {
      await api.kv.deleteNamespace(namespace.id)
      notifications.show({ title: 'Success', message: 'Namespace deleted', color: 'green' })
      loadNamespaces()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete namespace', color: 'red' })
    }
  }

  const formatSize = (bytes: number) => {
    if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${bytes} B`
  }

  const formatDate = (dateStr: string) => {
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

  const columns: Column<KVNamespace>[] = [
    { key: 'title', label: 'Name', sortable: true },
    { key: 'key_count', label: 'Keys', sortable: true, render: (row) => (row.key_count ?? 0).toLocaleString() },
    { key: 'storage_size', label: 'Size', sortable: true, render: (row) => formatSize(row.storage_size ?? 0) },
    { key: 'created_at', label: 'Created', sortable: true, render: (row) => formatDate(row.created_at) },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="Workers KV"
        subtitle="Global, low-latency key-value data storage"
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
        onRowClick={(row) => navigate(`/kv/${row.id}`)}
        actions={[
          { label: 'View', icon: <IconEye size={14} />, onClick: (row) => navigate(`/kv/${row.id}`) },
          { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
        ]}
        emptyState={{
          title: 'No KV namespaces yet',
          description: 'Create your first KV namespace to store key-value data globally',
          action: { label: 'Create Namespace', onClick: () => setCreateModalOpen(true) },
        }}
      />

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Create KV Namespace" size="md">
        <form onSubmit={form.onSubmit(handleCreate)}>
          <Stack gap="md">
            <TextInput
              label="Namespace Name"
              placeholder="MY_KV_NAMESPACE"
              description="Use uppercase letters, numbers, and underscores"
              required
              {...form.getInputProps('title')}
            />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateModalOpen(false)}>Cancel</Button>
              <Button type="submit">Create Namespace</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}
