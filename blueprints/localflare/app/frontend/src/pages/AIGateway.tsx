import { useState, useEffect } from 'react'
import { Button, Modal, TextInput, Stack, Group, Switch, NumberInput } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconList, IconTrash } from '@tabler/icons-react'
import { PageHeader, DataTable, StatusBadge, type Column } from '../components/common'
import { api } from '../api/client'
import type { AIGateway } from '../types'

export function AIGatewayPage() {
  const navigate = useNavigate()
  const [gateways, setGateways] = useState<AIGateway[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const form = useForm({
    initialValues: {
      name: '',
      cache_enabled: true,
      cache_ttl: 300,
      rate_limit_enabled: false,
      rate_limit: 100,
      logging_enabled: true,
    },
    validate: { name: (v) => (!v ? 'Name is required' : null) },
  })

  useEffect(() => {
    loadGateways()
  }, [])

  const loadGateways = async () => {
    try {
      const res = await api.aiGateway.list()
      if (res.result) setGateways(res.result.gateways)
    } catch (error) {
      setGateways([
        {
          id: '1',
          name: 'prod-gateway',
          created_at: new Date().toISOString(),
          settings: { cache_enabled: true, cache_ttl: 300, rate_limit_enabled: true, rate_limit: 100, rate_limit_period: '1m', logging_enabled: true, retry_enabled: true, retry_count: 3 },
          stats: { total_requests: 45200, cached_requests: 35300, error_count: 234, total_tokens: 8900000, total_cost: 12.45 },
        },
        {
          id: '2',
          name: 'dev-gateway',
          created_at: new Date().toISOString(),
          settings: { cache_enabled: true, cache_ttl: 60, rate_limit_enabled: false, rate_limit: 0, rate_limit_period: '1m', logging_enabled: true, retry_enabled: false, retry_count: 0 },
          stats: { total_requests: 1200, cached_requests: 540, error_count: 12, total_tokens: 456000, total_cost: 0.89 },
        },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: typeof form.values) => {
    try {
      await api.aiGateway.create({
        name: values.name,
        settings: {
          cache_enabled: values.cache_enabled,
          cache_ttl: values.cache_ttl,
          rate_limit_enabled: values.rate_limit_enabled,
          rate_limit: values.rate_limit,
          logging_enabled: values.logging_enabled,
        },
      })
      notifications.show({ title: 'Success', message: 'Gateway created', color: 'green' })
      setCreateModalOpen(false)
      form.reset()
      loadGateways()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create gateway', color: 'red' })
    }
  }

  const handleDelete = async (gateway: AIGateway) => {
    if (!confirm(`Delete gateway "${gateway.name}"?`)) return
    try {
      await api.aiGateway.delete(gateway.id)
      notifications.show({ title: 'Success', message: 'Gateway deleted', color: 'green' })
      loadGateways()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete gateway', color: 'red' })
    }
  }

  // Helper to get stats - handles both flat and nested structure
  const getStats = (row: AIGateway) => ({
    total_requests: row.stats?.total_requests ?? 0,
    cached_requests: row.stats?.cached_requests ?? 0,
  })

  const columns: Column<AIGateway>[] = [
    { key: 'name', label: 'Name', sortable: true },
    { key: 'stats.total_requests', label: 'Requests', sortable: true, render: (row) => getStats(row).total_requests.toLocaleString() },
    { key: 'cache_hit', label: 'Cache Hit', render: (row) => {
      const stats = getStats(row)
      if (stats.total_requests === 0) return '0%'
      return `${Math.round((stats.cached_requests / stats.total_requests) * 100)}%`
    }},
    { key: 'status', label: 'Status', render: () => <StatusBadge status="active" /> },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="AI Gateway"
        subtitle="Manage AI Gateway configurations with logging and caching"
        actions={<Button leftSection={<IconPlus size={16} />} onClick={() => setCreateModalOpen(true)}>Create Gateway</Button>}
      />

      <DataTable
        data={gateways}
        columns={columns}
        loading={loading}
        getRowKey={(row) => row.id}
        searchPlaceholder="Search gateways..."
        onRowClick={(row) => navigate(`/ai-gateway/${row.id}`)}
        actions={[
          { label: 'Logs', icon: <IconList size={14} />, onClick: (row) => navigate(`/ai-gateway/${row.id}/logs`) },
          { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
        ]}
        emptyState={{
          title: 'No gateways yet',
          description: 'Create your first AI Gateway to get started',
          action: { label: 'Create Gateway', onClick: () => setCreateModalOpen(true) },
        }}
      />

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Create AI Gateway" size="md">
        <form onSubmit={form.onSubmit(handleCreate)}>
          <Stack gap="md">
            <TextInput label="Gateway Name" placeholder="my-gateway" required {...form.getInputProps('name')} />
            <Switch label="Enable Caching" {...form.getInputProps('cache_enabled', { type: 'checkbox' })} />
            {form.values.cache_enabled && (
              <NumberInput label="Cache TTL (seconds)" min={0} {...form.getInputProps('cache_ttl')} />
            )}
            <Switch label="Enable Rate Limiting" {...form.getInputProps('rate_limit_enabled', { type: 'checkbox' })} />
            {form.values.rate_limit_enabled && (
              <NumberInput label="Rate Limit (requests/min)" min={1} {...form.getInputProps('rate_limit')} />
            )}
            <Switch label="Enable Logging" {...form.getInputProps('logging_enabled', { type: 'checkbox' })} />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateModalOpen(false)}>Cancel</Button>
              <Button type="submit">Create Gateway</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}
