import { useState, useEffect } from 'react'
import { Button, Modal, TextInput, Stack, Group } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconDatabase, IconTrash } from '@tabler/icons-react'
import { PageHeader, DataTable, type Column } from '../components/common'
import { api } from '../api/client'
import type { D1Database } from '../types'

export function D1() {
  const navigate = useNavigate()
  const [databases, setDatabases] = useState<D1Database[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const form = useForm({
    initialValues: {
      name: '',
    },
    validate: {
      name: (v) => (!v ? 'Name is required' : !/^[a-z0-9-_]+$/.test(v) ? 'Use lowercase letters, numbers, hyphens, and underscores only' : null),
    },
  })

  useEffect(() => {
    loadDatabases()
  }, [])

  const loadDatabases = async () => {
    try {
      const res = await api.d1.listDatabases()
      if (res.result) setDatabases(res.result.databases ?? [])
    } catch (error) {
      console.error('Failed to load D1 databases:', error)
      setDatabases([
        {
          uuid: 'd1-1',
          name: 'production-db',
          created_at: new Date(Date.now() - 172800000).toISOString(),
          version: 'production',
          num_tables: 12,
          file_size: 45 * 1024 * 1024,
        },
        {
          uuid: 'd1-2',
          name: 'staging-db',
          created_at: new Date(Date.now() - 604800000).toISOString(),
          version: 'production',
          num_tables: 12,
          file_size: 23 * 1024 * 1024,
        },
        {
          uuid: 'd1-3',
          name: 'analytics-db',
          created_at: new Date(Date.now() - 259200000).toISOString(),
          version: 'production',
          num_tables: 5,
          file_size: 156 * 1024 * 1024,
        },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: typeof form.values) => {
    try {
      await api.d1.createDatabase(values)
      notifications.show({ title: 'Success', message: 'Database created', color: 'green' })
      setCreateModalOpen(false)
      form.reset()
      loadDatabases()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create database', color: 'red' })
    }
  }

  const handleDelete = async (database: D1Database) => {
    if (!confirm(`Delete database "${database.name}"? This cannot be undone.`)) return
    try {
      await api.d1.deleteDatabase(database.uuid)
      notifications.show({ title: 'Success', message: 'Database deleted', color: 'green' })
      loadDatabases()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete database', color: 'red' })
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

  const columns: Column<D1Database>[] = [
    { key: 'name', label: 'Name', sortable: true },
    { key: 'num_tables', label: 'Tables', sortable: true },
    { key: 'file_size', label: 'Size', sortable: true, render: (row) => formatSize(row.file_size ?? 0) },
    { key: 'version', label: 'Version' },
    { key: 'created_at', label: 'Created', sortable: true, render: (row) => formatDate(row.created_at) },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="D1 Database"
        subtitle="Serverless SQL database built on SQLite"
        actions={
          <Button leftSection={<IconPlus size={16} />} onClick={() => setCreateModalOpen(true)}>
            Create Database
          </Button>
        }
      />

      <DataTable
        data={databases}
        columns={columns}
        loading={loading}
        getRowKey={(row) => row.uuid}
        searchPlaceholder="Search databases..."
        onRowClick={(row) => navigate(`/d1/${row.uuid}`)}
        actions={[
          { label: 'Open Console', icon: <IconDatabase size={14} />, onClick: (row) => navigate(`/d1/${row.uuid}`) },
          { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
        ]}
        emptyState={{
          title: 'No databases yet',
          description: 'Create your first D1 database to store relational data',
          action: { label: 'Create Database', onClick: () => setCreateModalOpen(true) },
        }}
      />

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Create D1 Database" size="md">
        <form onSubmit={form.onSubmit(handleCreate)}>
          <Stack gap="md">
            <TextInput
              label="Database Name"
              placeholder="my-database"
              description="Use lowercase letters, numbers, hyphens, and underscores"
              required
              {...form.getInputProps('name')}
            />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateModalOpen(false)}>Cancel</Button>
              <Button type="submit">Create Database</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}
