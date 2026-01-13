import { useState, useEffect } from 'react'
import { Button, Modal, TextInput, Stack, Group, Select } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconEye, IconTrash, IconWorld, IconLock } from '@tabler/icons-react'
import { PageHeader, DataTable, StatusBadge, type Column } from '../components/common'
import { api } from '../api/client'
import type { R2Bucket } from '../types'

export function R2() {
  const navigate = useNavigate()
  const [buckets, setBuckets] = useState<R2Bucket[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const form = useForm({
    initialValues: {
      name: '',
      location_hint: 'auto',
    },
    validate: {
      name: (v) => (!v ? 'Name is required' : !/^[a-z0-9-]+$/.test(v) ? 'Use lowercase letters, numbers, and hyphens only' : null),
    },
  })

  useEffect(() => {
    loadBuckets()
  }, [])

  const loadBuckets = async () => {
    try {
      const res = await api.r2.listBuckets()
      if (res.result) setBuckets(res.result.buckets ?? [])
    } catch (error) {
      console.error('Failed to load R2 buckets:', error)
      setBuckets([
        {
          name: 'media-assets',
          created_at: new Date(Date.now() - 172800000).toISOString(),
          location: 'WNAM',
          object_count: 12456,
          storage_size: 2.5 * 1024 * 1024 * 1024,
          public_access: true,
        },
        {
          name: 'user-uploads',
          created_at: new Date(Date.now() - 604800000).toISOString(),
          location: 'WNAM',
          object_count: 45678,
          storage_size: 8.9 * 1024 * 1024 * 1024,
          public_access: false,
        },
        {
          name: 'backups',
          created_at: new Date(Date.now() - 259200000).toISOString(),
          location: 'ENAM',
          object_count: 234,
          storage_size: 15.2 * 1024 * 1024 * 1024,
          public_access: false,
        },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: typeof form.values) => {
    try {
      await api.r2.createBucket(values)
      notifications.show({ title: 'Success', message: 'Bucket created', color: 'green' })
      setCreateModalOpen(false)
      form.reset()
      loadBuckets()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create bucket', color: 'red' })
    }
  }

  const handleDelete = async (bucket: R2Bucket) => {
    if (!confirm(`Delete bucket "${bucket.name}"? This will delete all objects.`)) return
    try {
      await api.r2.deleteBucket(bucket.name)
      notifications.show({ title: 'Success', message: 'Bucket deleted', color: 'green' })
      loadBuckets()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete bucket', color: 'red' })
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

  const columns: Column<R2Bucket>[] = [
    { key: 'name', label: 'Bucket Name', sortable: true },
    { key: 'object_count', label: 'Objects', sortable: true, render: (row) => (row.object_count ?? 0).toLocaleString() },
    { key: 'storage_size', label: 'Size', sortable: true, render: (row) => formatSize(row.storage_size ?? 0) },
    { key: 'location', label: 'Location', render: (row) => row.location || 'Auto' },
    {
      key: 'public_access',
      label: 'Access',
      render: (row) => row.public_access ? (
        <Group gap={4}><IconWorld size={14} color="var(--mantine-color-green-6)" /><StatusBadge status="public" /></Group>
      ) : (
        <Group gap={4}><IconLock size={14} color="var(--mantine-color-gray-6)" /><StatusBadge status="private" /></Group>
      ),
    },
    { key: 'created_at', label: 'Created', sortable: true, render: (row) => formatDate(row.created_at) },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="R2 Object Storage"
        subtitle="S3-compatible object storage with zero egress fees"
        actions={
          <Button leftSection={<IconPlus size={16} />} onClick={() => setCreateModalOpen(true)}>
            Create Bucket
          </Button>
        }
      />

      <DataTable
        data={buckets}
        columns={columns}
        loading={loading}
        getRowKey={(row) => row.name}
        searchPlaceholder="Search buckets..."
        onRowClick={(row) => navigate(`/r2/${row.name}`)}
        actions={[
          { label: 'View', icon: <IconEye size={14} />, onClick: (row) => navigate(`/r2/${row.name}`) },
          { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
        ]}
        emptyState={{
          title: 'No buckets yet',
          description: 'Create your first R2 bucket to store objects',
          action: { label: 'Create Bucket', onClick: () => setCreateModalOpen(true) },
        }}
      />

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Create R2 Bucket" size="md">
        <form onSubmit={form.onSubmit(handleCreate)}>
          <Stack gap="md">
            <TextInput
              label="Bucket Name"
              placeholder="my-bucket"
              description="Use lowercase letters, numbers, and hyphens. Must be globally unique."
              required
              {...form.getInputProps('name')}
            />
            <Select
              label="Location Hint"
              description="Suggest a location for your bucket. Data is stored closest to initial access."
              data={[
                { value: 'auto', label: 'Automatic (Recommended)' },
                { value: 'wnam', label: 'Western North America' },
                { value: 'enam', label: 'Eastern North America' },
                { value: 'weur', label: 'Western Europe' },
                { value: 'eeur', label: 'Eastern Europe' },
                { value: 'apac', label: 'Asia Pacific' },
              ]}
              {...form.getInputProps('location_hint')}
            />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateModalOpen(false)}>Cancel</Button>
              <Button type="submit">Create Bucket</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}
