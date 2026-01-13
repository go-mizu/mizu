import { useState, useEffect } from 'react'
import { Button, Modal, TextInput, Stack, Group, Select, Switch, Paper, Text, SimpleGrid } from '@mantine/core'
import { useNavigate } from 'react-router-dom'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconEye, IconTrash } from '@tabler/icons-react'
import { PageHeader, DataTable, StatusBadge, type Column } from '../components/common'
import { api } from '../api/client'
import type { CronTrigger } from '../types'

const cronPresets = [
  { value: '* * * * *', label: 'Every minute' },
  { value: '*/5 * * * *', label: 'Every 5 minutes' },
  { value: '0 * * * *', label: 'Hourly' },
  { value: '0 0 * * *', label: 'Daily at midnight' },
  { value: '0 0 * * 0', label: 'Weekly on Sunday' },
]

export function Cron() {
  const navigate = useNavigate()
  const [triggers, setTriggers] = useState<CronTrigger[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const form = useForm({
    initialValues: {
      cron: '*/5 * * * *',
      script_name: '',
      enabled: true,
    },
    validate: {
      cron: (v) => (!v ? 'Cron expression is required' : null),
      script_name: (v) => (!v ? 'Script is required' : null),
    },
  })

  useEffect(() => {
    loadTriggers()
  }, [])

  const loadTriggers = async () => {
    try {
      const res = await api.cron.list()
      if (res.result) setTriggers(res.result.triggers ?? [])
    } catch (error) {
      setTriggers([
        { id: '1', cron: '*/5 * * * *', script_name: 'cleanup-worker', enabled: true, created_at: new Date().toISOString(), last_run: new Date(Date.now() - 120000).toISOString(), next_run: new Date(Date.now() + 180000).toISOString() },
        { id: '2', cron: '0 * * * *', script_name: 'hourly-report', enabled: true, created_at: new Date().toISOString(), last_run: new Date(Date.now() - 2700000).toISOString(), next_run: new Date(Date.now() + 900000).toISOString() },
        { id: '3', cron: '0 0 * * *', script_name: 'daily-backup', enabled: true, created_at: new Date().toISOString(), last_run: new Date(Date.now() - 28800000).toISOString(), next_run: new Date(Date.now() + 57600000).toISOString() },
        { id: '4', cron: '0 0 * * 0', script_name: 'weekly-digest', enabled: false, created_at: new Date().toISOString(), last_run: new Date(Date.now() - 259200000).toISOString() },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: typeof form.values) => {
    try {
      await api.cron.create(values)
      notifications.show({ title: 'Success', message: 'Trigger created', color: 'green' })
      setCreateModalOpen(false)
      form.reset()
      loadTriggers()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create trigger', color: 'red' })
    }
  }

  const handleDelete = async (trigger: CronTrigger) => {
    if (!confirm(`Delete trigger "${trigger.cron} -> ${trigger.script_name}"?`)) return
    try {
      await api.cron.delete(trigger.id)
      notifications.show({ title: 'Success', message: 'Trigger deleted', color: 'green' })
      loadTriggers()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete trigger', color: 'red' })
    }
  }

  const formatRelativeTime = (dateStr?: string) => {
    if (!dateStr) return '-'
    const date = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const future = diff < 0
    const absDiff = Math.abs(diff)
    const mins = Math.floor(absDiff / 60000)
    if (mins < 1) return future ? 'in a moment' : 'Just now'
    if (mins < 60) return future ? `in ${mins} mins` : `${mins} mins ago`
    const hours = Math.floor(mins / 60)
    if (hours < 24) return future ? `in ${hours} hours` : `${hours} hours ago`
    const days = Math.floor(hours / 24)
    return future ? `in ${days} days` : `${days} days ago`
  }

  const columns: Column<CronTrigger>[] = [
    { key: 'cron', label: 'Schedule', sortable: true },
    { key: 'script_name', label: 'Script', sortable: true },
    { key: 'last_run', label: 'Last Run', render: (row) => formatRelativeTime(row.last_run) },
    { key: 'enabled', label: 'Status', render: (row) => <StatusBadge status={row.enabled ? 'enabled' : 'disabled'} /> },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title="Cron Triggers"
        subtitle="Manage scheduled task triggers and view executions"
        actions={<Button leftSection={<IconPlus size={16} />} onClick={() => setCreateModalOpen(true)}>Create Trigger</Button>}
      />

      <DataTable
        data={triggers}
        columns={columns}
        loading={loading}
        getRowKey={(row) => row.id}
        searchPlaceholder="Search triggers..."
        onRowClick={(row) => navigate(`/cron/${row.id}`)}
        actions={[
          { label: 'View', icon: <IconEye size={14} />, onClick: (row) => navigate(`/cron/${row.id}`) },
          { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
        ]}
        emptyState={{
          title: 'No triggers yet',
          description: 'Create your first cron trigger to get started',
          action: { label: 'Create Trigger', onClick: () => setCreateModalOpen(true) },
        }}
      />

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Create Cron Trigger" size="md">
        <form onSubmit={form.onSubmit(handleCreate)}>
          <Stack gap="md">
            <TextInput
              label="Cron Schedule"
              placeholder="*/5 * * * *"
              required
              {...form.getInputProps('cron')}
            />
            <Text size="xs" c="dimmed">
              {describeCron(form.values.cron)}
            </Text>

            <Text size="xs" fw={600}>Quick Presets</Text>
            <SimpleGrid cols={3} spacing="xs">
              {cronPresets.map((preset) => (
                <Paper
                  key={preset.value}
                  p="xs"
                  radius="sm"
                  withBorder
                  style={{ cursor: 'pointer' }}
                  onClick={() => form.setFieldValue('cron', preset.value)}
                >
                  <Text size="xs">{preset.label}</Text>
                </Paper>
              ))}
            </SimpleGrid>

            <Select
              label="Worker Script"
              placeholder="Select a worker..."
              data={[
                { value: 'cleanup-worker', label: 'cleanup-worker' },
                { value: 'report-worker', label: 'report-worker' },
                { value: 'backup-worker', label: 'backup-worker' },
                { value: 'digest-worker', label: 'digest-worker' },
              ]}
              required
              {...form.getInputProps('script_name')}
            />

            <Switch label="Enable trigger immediately" {...form.getInputProps('enabled', { type: 'checkbox' })} />

            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateModalOpen(false)}>Cancel</Button>
              <Button type="submit">Create Trigger</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}

function describeCron(cron: string): string {
  const parts = cron.split(' ')
  if (parts.length !== 5) return 'Invalid cron expression'

  const [min, hour, day, month, weekday] = parts

  if (min === '*' && hour === '*' && day === '*' && month === '*' && weekday === '*') {
    return 'Every minute'
  }
  if (min.startsWith('*/') && hour === '*' && day === '*' && month === '*' && weekday === '*') {
    return `Every ${min.slice(2)} minutes`
  }
  if (min === '0' && hour === '*' && day === '*' && month === '*' && weekday === '*') {
    return 'Every hour'
  }
  if (min === '0' && hour === '0' && day === '*' && month === '*' && weekday === '*') {
    return 'Every day at midnight'
  }
  if (min === '0' && hour === '0' && day === '*' && month === '*' && weekday === '0') {
    return 'Every Sunday at midnight'
  }

  return `${cron}`
}
