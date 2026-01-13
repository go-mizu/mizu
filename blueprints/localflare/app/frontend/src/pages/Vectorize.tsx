import { useState, useEffect } from 'react'
import { Button, Modal, TextInput, Stack, Group, NumberInput, Select, Textarea } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconSearch, IconTrash } from '@tabler/icons-react'
import { PageHeader, DataTable, type Column } from '../components/common'
import { api } from '../api/client'
import type { VectorIndex } from '../types'

export function Vectorize() {
  const navigate = useNavigate()
  const [indexes, setIndexes] = useState<VectorIndex[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const form = useForm({
    initialValues: { name: '', dimensions: 768, metric: 'cosine', description: '' },
    validate: {
      name: (v) => (!v ? 'Name is required' : null),
      dimensions: (v) => (v < 1 || v > 4096 ? 'Dimensions must be 1-4096' : null),
    },
  })

  useEffect(() => {
    loadIndexes()
  }, [])

  const loadIndexes = async () => {
    try {
      const res = await api.vectorize.listIndexes()
      if (res.result) setIndexes(res.result.indexes)
    } catch (error) {
      setIndexes([
        { id: '1', name: 'product-emb', dimensions: 768, metric: 'cosine', created_at: new Date().toISOString(), vector_count: 50234, namespace_count: 3 },
        { id: '2', name: 'doc-search', dimensions: 1536, metric: 'euclidean', created_at: new Date().toISOString(), vector_count: 12456, namespace_count: 1 },
        { id: '3', name: 'image-vectors', dimensions: 512, metric: 'dot-product', created_at: new Date().toISOString(), vector_count: 8901, namespace_count: 2 },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: typeof form.values) => {
    try {
      await api.vectorize.createIndex(values)
      notifications.show({ title: 'Success', message: 'Index created', color: 'green' })
      setCreateModalOpen(false)
      form.reset()
      loadIndexes()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create index', color: 'red' })
    }
  }

  const handleDelete = async (index: VectorIndex) => {
    if (!confirm(`Delete index "${index.name}"?`)) return
    try {
      await api.vectorize.deleteIndex(index.name)
      notifications.show({ title: 'Success', message: 'Index deleted', color: 'green' })
      loadIndexes()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete index', color: 'red' })
    }
  }

  const columns: Column<VectorIndex>[] = [
    { key: 'name', label: 'Name', sortable: true },
    { key: 'dimensions', label: 'Dimensions', sortable: true },
    { key: 'metric', label: 'Metric', sortable: true },
    { key: 'vector_count', label: 'Vectors', sortable: true, render: (row) => (row.vector_count ?? 0).toLocaleString() },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="Vectorize"
        subtitle="Manage vector indexes and perform similarity searches"
        actions={<Button leftSection={<IconPlus size={16} />} onClick={() => setCreateModalOpen(true)}>Create Index</Button>}
      />

      <DataTable
        data={indexes}
        columns={columns}
        loading={loading}
        getRowKey={(row) => row.id}
        searchPlaceholder="Search indexes..."
        onRowClick={(row) => navigate(`/vectorize/${row.name}`)}
        actions={[
          { label: 'Query', icon: <IconSearch size={14} />, onClick: (row) => navigate(`/vectorize/${row.name}`) },
          { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
        ]}
        emptyState={{
          title: 'No indexes yet',
          description: 'Create your first vector index to get started',
          action: { label: 'Create Index', onClick: () => setCreateModalOpen(true) },
        }}
      />

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Create Vector Index" size="md">
        <form onSubmit={form.onSubmit(handleCreate)}>
          <Stack gap="md">
            <TextInput label="Index Name" placeholder="my-embeddings" required {...form.getInputProps('name')} />
            <NumberInput label="Dimensions" min={1} max={4096} required {...form.getInputProps('dimensions')} />
            <Select
              label="Distance Metric"
              data={[
                { value: 'cosine', label: 'Cosine (recommended)' },
                { value: 'euclidean', label: 'Euclidean' },
                { value: 'dot-product', label: 'Dot Product' },
              ]}
              {...form.getInputProps('metric')}
            />
            <Textarea label="Description (optional)" placeholder="Product embeddings for similarity search" {...form.getInputProps('description')} />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateModalOpen(false)}>Cancel</Button>
              <Button type="submit">Create Index</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}
