import { useState, useEffect } from 'react'
import { Button, Modal, TextInput, Stack, Group, Select, NumberInput, Switch, PasswordInput } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconEye, IconTrash } from '@tabler/icons-react'
import { PageHeader, DataTable, StatusBadge, type Column } from '../components/common'
import { api } from '../api/client'
import type { HyperdriveConfig } from '../types'

export function Hyperdrive() {
  const navigate = useNavigate()
  const [configs, setConfigs] = useState<HyperdriveConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const form = useForm({
    initialValues: {
      name: '',
      scheme: 'postgres' as const,
      host: '',
      port: 5432,
      database: '',
      user: '',
      password: '',
      cache_enabled: true,
      max_age: 60,
      stale_while_revalidate: 15,
    },
    validate: {
      name: (v) => (!v ? 'Name is required' : null),
      host: (v) => (!v ? 'Host is required' : null),
      database: (v) => (!v ? 'Database is required' : null),
      user: (v) => (!v ? 'User is required' : null),
      password: (v) => (!v ? 'Password is required' : null),
    },
  })

  useEffect(() => {
    loadConfigs()
  }, [])

  const loadConfigs = async () => {
    try {
      const res = await api.hyperdrive.list()
      if (res.result) setConfigs(res.result.configs)
    } catch (error) {
      setConfigs([
        { id: '1', name: 'prod-postgres', created_at: new Date().toISOString(), origin: { scheme: 'postgres', host: 'db.example.com', port: 5432, database: 'app_db', user: 'app_user' }, caching: { enabled: true, max_age: 60, stale_while_revalidate: 15 }, status: 'connected' },
        { id: '2', name: 'analytics-db', created_at: new Date().toISOString(), origin: { scheme: 'postgres', host: 'analytics.io', port: 5432, database: 'analytics', user: 'analytics_user' }, caching: { enabled: true, max_age: 120, stale_while_revalidate: 30 }, status: 'connected' },
        { id: '3', name: 'staging-db', created_at: new Date().toISOString(), origin: { scheme: 'postgres', host: 'staging.local', port: 5432, database: 'staging', user: 'staging_user' }, caching: { enabled: false, max_age: 0, stale_while_revalidate: 0 }, status: 'idle' },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: typeof form.values) => {
    try {
      await api.hyperdrive.create({
        name: values.name,
        origin: {
          scheme: values.scheme,
          host: values.host,
          port: values.port,
          database: values.database,
          user: values.user,
          password: values.password,
        },
        caching: {
          enabled: values.cache_enabled,
          max_age: values.max_age,
          stale_while_revalidate: values.stale_while_revalidate,
        },
      })
      notifications.show({ title: 'Success', message: 'Config created', color: 'green' })
      setCreateModalOpen(false)
      form.reset()
      loadConfigs()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create config', color: 'red' })
    }
  }

  const handleDelete = async (config: HyperdriveConfig) => {
    if (!confirm(`Delete config "${config.name}"?`)) return
    try {
      await api.hyperdrive.delete(config.id)
      notifications.show({ title: 'Success', message: 'Config deleted', color: 'green' })
      loadConfigs()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete config', color: 'red' })
    }
  }

  const columns: Column<HyperdriveConfig>[] = [
    { key: 'name', label: 'Name', sortable: true },
    { key: 'origin.database', label: 'Database', render: (row) => row.origin?.database ?? '-' },
    { key: 'origin.host', label: 'Host', render: (row) => row.origin?.host ?? '-' },
    { key: 'status', label: 'Status', render: (row) => <StatusBadge status={row.status ?? 'idle'} /> },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="Hyperdrive"
        subtitle="Database connection pooling configuration"
        actions={<Button leftSection={<IconPlus size={16} />} onClick={() => setCreateModalOpen(true)}>Create Config</Button>}
      />

      <DataTable
        data={configs}
        columns={columns}
        loading={loading}
        getRowKey={(row) => row.id}
        searchPlaceholder="Search configs..."
        onRowClick={(row) => navigate(`/hyperdrive/${row.id}`)}
        actions={[
          { label: 'View', icon: <IconEye size={14} />, onClick: (row) => navigate(`/hyperdrive/${row.id}`) },
          { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
        ]}
        emptyState={{
          title: 'No configs yet',
          description: 'Create your first Hyperdrive config to get started',
          action: { label: 'Create Config', onClick: () => setCreateModalOpen(true) },
        }}
      />

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Create Hyperdrive Config" size="lg">
        <form onSubmit={form.onSubmit(handleCreate)}>
          <Stack gap="md">
            <TextInput label="Config Name" placeholder="my-database" required {...form.getInputProps('name')} />

            <Group grow>
              <Select label="Scheme" data={[{ value: 'postgres', label: 'PostgreSQL' }, { value: 'mysql', label: 'MySQL' }]} {...form.getInputProps('scheme')} />
              <TextInput label="Host" placeholder="db.example.com" required {...form.getInputProps('host')} />
            </Group>

            <Group grow>
              <NumberInput label="Port" min={1} max={65535} {...form.getInputProps('port')} />
              <TextInput label="Database" placeholder="my_database" required {...form.getInputProps('database')} />
            </Group>

            <Group grow>
              <TextInput label="User" placeholder="db_user" required {...form.getInputProps('user')} />
              <PasswordInput label="Password" required {...form.getInputProps('password')} />
            </Group>

            <Switch label="Enable query caching" {...form.getInputProps('cache_enabled', { type: 'checkbox' })} />

            {form.values.cache_enabled && (
              <Group grow>
                <NumberInput label="Max Age (seconds)" min={0} {...form.getInputProps('max_age')} />
                <NumberInput label="Stale While Revalidate (seconds)" min={0} {...form.getInputProps('stale_while_revalidate')} />
              </Group>
            )}

            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateModalOpen(false)}>Cancel</Button>
              <Button type="submit">Create Config</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}
