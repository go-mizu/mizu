import { useState, useEffect } from 'react'
import { Button, Modal, TextInput, Stack, Group, NumberInput } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconEye, IconTrash } from '@tabler/icons-react'
import { PageHeader, DataTable, type Column } from '../components/common'
import { api } from '../api/client'
import type { Queue } from '../types'

export function Queues() {
  const navigate = useNavigate()
  const [queues, setQueues] = useState<Queue[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const form = useForm({
    initialValues: {
      name: '',
      max_retries: 3,
      batch_size: 10,
    },
    validate: {
      name: (v) => (!v ? 'Name is required' : null),
    },
  })

  useEffect(() => {
    loadQueues()
  }, [])

  const loadQueues = async () => {
    try {
      const res = await api.queues.list()
      if (res.result) setQueues(res.result.queues)
    } catch (error) {
      console.error('Failed to load queues:', error)
      setQueues([
        { id: '1', name: 'order-events', created_at: new Date().toISOString(), message_count: 1234, ready_count: 1100, delayed_count: 134, failed_count: 12, settings: { max_retries: 3, max_batch_size: 10, max_batch_timeout: 30, message_ttl: 345600, delivery_delay: 0 }, consumers: [] },
        { id: '2', name: 'email-queue', created_at: new Date().toISOString(), message_count: 567, ready_count: 560, delayed_count: 7, failed_count: 0, settings: { max_retries: 3, max_batch_size: 10, max_batch_timeout: 30, message_ttl: 345600, delivery_delay: 0 }, consumers: [] },
        { id: '3', name: 'analytics', created_at: new Date().toISOString(), message_count: 8901, ready_count: 8500, delayed_count: 401, failed_count: 5, settings: { max_retries: 3, max_batch_size: 10, max_batch_timeout: 30, message_ttl: 345600, delivery_delay: 0 }, consumers: [] },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: typeof form.values) => {
    try {
      await api.queues.create({ name: values.name, settings: { max_retries: values.max_retries, max_batch_size: values.batch_size, max_batch_timeout: 30, message_ttl: 345600, delivery_delay: 0 } })
      notifications.show({ title: 'Success', message: 'Queue created', color: 'green' })
      setCreateModalOpen(false)
      form.reset()
      loadQueues()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create queue', color: 'red' })
    }
  }

  const handleDelete = async (queue: Queue) => {
    if (!confirm(`Delete queue "${queue.name}"?`)) return
    try {
      await api.queues.delete(queue.id)
      notifications.show({ title: 'Success', message: 'Queue deleted', color: 'green' })
      loadQueues()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete queue', color: 'red' })
    }
  }

  const columns: Column<Queue>[] = [
    { key: 'name', label: 'Name', sortable: true },
    { key: 'message_count', label: 'Messages', sortable: true, render: (row) => (row.message_count ?? 0).toLocaleString() },
    { key: 'ready_count', label: 'Ready', sortable: true, render: (row) => (row.ready_count ?? 0).toLocaleString() },
    { key: 'delayed_count', label: 'Delayed', sortable: true, render: (row) => (row.delayed_count ?? 0).toLocaleString() },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="Queues"
        subtitle="Manage message queues and monitor queue health"
        actions={
          <Button leftSection={<IconPlus size={16} />} onClick={() => setCreateModalOpen(true)}>
            Create Queue
          </Button>
        }
      />

      <DataTable
        data={queues}
        columns={columns}
        loading={loading}
        getRowKey={(row) => row.id}
        searchPlaceholder="Search queues..."
        onRowClick={(row) => navigate(`/queues/${row.id}`)}
        actions={[
          { label: 'View', icon: <IconEye size={14} />, onClick: (row) => navigate(`/queues/${row.id}`) },
          { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
        ]}
        emptyState={{
          title: 'No queues yet',
          description: 'Create your first queue to get started',
          action: { label: 'Create Queue', onClick: () => setCreateModalOpen(true) },
        }}
      />

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Create Queue" size="md">
        <form onSubmit={form.onSubmit(handleCreate)}>
          <Stack gap="md">
            <TextInput label="Queue Name" placeholder="my-queue" required {...form.getInputProps('name')} />
            <NumberInput label="Max Retries" min={0} max={10} {...form.getInputProps('max_retries')} />
            <NumberInput label="Batch Size" min={1} max={100} {...form.getInputProps('batch_size')} />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateModalOpen(false)}>Cancel</Button>
              <Button type="submit">Create Queue</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}
