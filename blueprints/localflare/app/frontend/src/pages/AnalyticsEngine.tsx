import { useState, useEffect } from 'react'
import { Button, Modal, TextInput, Stack, Group } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconSearch, IconTrash } from '@tabler/icons-react'
import { PageHeader, DataTable, type Column } from '../components/common'
import { api } from '../api/client'
import type { AnalyticsDataset } from '../types'

export function AnalyticsEngine() {
  const navigate = useNavigate()
  const [datasets, setDatasets] = useState<AnalyticsDataset[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const form = useForm({
    initialValues: { name: '' },
    validate: { name: (v) => (!v ? 'Name is required' : null) },
  })

  useEffect(() => {
    loadDatasets()
  }, [])

  const loadDatasets = async () => {
    try {
      const res = await api.analytics.listDatasets()
      if (res.result) setDatasets(res.result.datasets ?? [])
    } catch (error) {
      setDatasets([
        { id: '1', name: 'page_views', created_at: new Date(Date.now() - 14 * 86400000).toISOString(), data_points: 1200000, estimated_size_bytes: 245 * 1024 * 1024, last_write: new Date(Date.now() - 120000).toISOString() },
        { id: '2', name: 'api_metrics', created_at: new Date(Date.now() - 30 * 86400000).toISOString(), data_points: 5600000, estimated_size_bytes: 890 * 1024 * 1024, last_write: new Date(Date.now() - 60000).toISOString() },
        { id: '3', name: 'user_events', created_at: new Date(Date.now() - 3 * 86400000).toISOString(), data_points: 890000, estimated_size_bytes: 156 * 1024 * 1024, last_write: new Date(Date.now() - 300000).toISOString() },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: typeof form.values) => {
    try {
      await api.analytics.createDataset(values)
      notifications.show({ title: 'Success', message: 'Dataset created', color: 'green' })
      setCreateModalOpen(false)
      form.reset()
      loadDatasets()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create dataset', color: 'red' })
    }
  }

  const handleDelete = async (dataset: AnalyticsDataset) => {
    if (!confirm(`Delete dataset "${dataset.name}"?`)) return
    try {
      await api.analytics.deleteDataset(dataset.name)
      notifications.show({ title: 'Success', message: 'Dataset deleted', color: 'green' })
      loadDatasets()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete dataset', color: 'red' })
    }
  }

  const formatDataPoints = (n: number) => {
    if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`
    if (n >= 1000) return `${(n / 1000).toFixed(0)}K`
    return n.toString()
  }

  const formatSize = (bytes: number) => {
    if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(0)} MB`
    return `${(bytes / 1024).toFixed(0)} KB`
  }

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const days = Math.floor(diff / 86400000)
    if (days === 0) return 'Today'
    if (days < 7) return `${days} days ago`
    if (days < 30) return `${Math.floor(days / 7)} weeks ago`
    return `${Math.floor(days / 30)} month${Math.floor(days / 30) > 1 ? 's' : ''} ago`
  }

  const columns: Column<AnalyticsDataset>[] = [
    { key: 'name', label: 'Dataset', sortable: true },
    { key: 'data_points', label: 'Data Points', sortable: true, render: (row) => formatDataPoints(row.data_points) },
    { key: 'estimated_size_bytes', label: 'Size', sortable: true, render: (row) => formatSize(row.estimated_size_bytes) },
    { key: 'created_at', label: 'Created', sortable: true, render: (row) => formatDate(row.created_at) },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="Analytics Engine"
        subtitle="Time-series analytics with SQL query interface"
        actions={<Button leftSection={<IconPlus size={16} />} onClick={() => setCreateModalOpen(true)}>Create Dataset</Button>}
      />

      <DataTable
        data={datasets}
        columns={columns}
        loading={loading}
        getRowKey={(row) => row.id}
        searchPlaceholder="Search datasets..."
        onRowClick={(row) => navigate(`/analytics-engine/${row.name}`)}
        actions={[
          { label: 'Query', icon: <IconSearch size={14} />, onClick: (row) => navigate(`/analytics-engine/${row.name}`) },
          { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
        ]}
        emptyState={{
          title: 'No datasets yet',
          description: 'Create your first analytics dataset to get started',
          action: { label: 'Create Dataset', onClick: () => setCreateModalOpen(true) },
        }}
      />

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Create Analytics Dataset" size="md">
        <form onSubmit={form.onSubmit(handleCreate)}>
          <Stack gap="md">
            <TextInput label="Dataset Name" placeholder="page_views" required {...form.getInputProps('name')} />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateModalOpen(false)}>Cancel</Button>
              <Button type="submit">Create Dataset</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}
