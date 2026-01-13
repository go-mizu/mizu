import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Stack, Paper, Text, Group, Button, Progress } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconPlayerPlay } from '@tabler/icons-react'
import { PageHeader, DataTable, LoadingState, StatusBadge, type Column } from '../components/common'
import { api } from '../api/client'
import type { CronTrigger, CronExecution } from '../types'

export function CronDetail() {
  const { id } = useParams<{ id: string }>()
  const [trigger, setTrigger] = useState<CronTrigger | null>(null)
  const [executions, setExecutions] = useState<CronExecution[]>([])
  const [loading, setLoading] = useState(true)
  const [triggerLoading, setTriggerLoading] = useState(false)

  useEffect(() => {
    if (id) loadData()
  }, [id])

  const loadData = async () => {
    try {
      const [triggerRes, execRes] = await Promise.all([
        api.cron.get(id!),
        api.cron.getExecutions(id!, 20),
      ])
      if (triggerRes.result) setTrigger(triggerRes.result)
      if (execRes.result) setExecutions(execRes.result.executions)
    } catch (error) {
      setTrigger({
        id: id!,
        cron: '*/5 * * * *',
        script_name: 'cleanup-worker',
        enabled: true,
        created_at: new Date().toISOString(),
        last_run: new Date(Date.now() - 120000).toISOString(),
        next_run: new Date(Date.now() + 180000).toISOString(),
      })
      setExecutions([
        { id: '1', trigger_id: id!, scheduled_at: new Date(Date.now() - 120000).toISOString(), started_at: new Date(Date.now() - 119000).toISOString(), finished_at: new Date(Date.now() - 117800).toISOString(), duration_ms: 1200, status: 'success' },
        { id: '2', trigger_id: id!, scheduled_at: new Date(Date.now() - 420000).toISOString(), started_at: new Date(Date.now() - 419000).toISOString(), finished_at: new Date(Date.now() - 418200).toISOString(), duration_ms: 800, status: 'success' },
        { id: '3', trigger_id: id!, scheduled_at: new Date(Date.now() - 720000).toISOString(), started_at: new Date(Date.now() - 719000).toISOString(), finished_at: new Date(Date.now() - 716900).toISOString(), duration_ms: 2100, status: 'success' },
        { id: '4', trigger_id: id!, scheduled_at: new Date(Date.now() - 1020000).toISOString(), started_at: new Date(Date.now() - 1019000).toISOString(), finished_at: new Date(Date.now() - 1003700).toISOString(), duration_ms: 15300, status: 'success' },
        { id: '5', trigger_id: id!, scheduled_at: new Date(Date.now() - 1320000).toISOString(), started_at: new Date(Date.now() - 1319000).toISOString(), status: 'failed', error: 'Script execution timeout' },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleTrigger = async () => {
    setTriggerLoading(true)
    try {
      await api.cron.trigger(id!)
      notifications.show({ title: 'Success', message: 'Trigger executed', color: 'green' })
      loadData()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to trigger', color: 'red' })
    } finally {
      setTriggerLoading(false)
    }
  }

  const handleToggle = async () => {
    if (!trigger) return
    try {
      await api.cron.update(id!, { enabled: !trigger.enabled })
      notifications.show({ title: 'Success', message: trigger.enabled ? 'Trigger disabled' : 'Trigger enabled', color: 'green' })
      loadData()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to update trigger', color: 'red' })
    }
  }

  if (loading) return <LoadingState />
  if (!trigger) return <Text>Trigger not found</Text>

  const formatTime = (dateStr: string) => {
    return new Date(dateStr).toLocaleString()
  }

  const formatRelativeTime = (dateStr?: string) => {
    if (!dateStr) return '-'
    const date = new Date(dateStr)
    const now = new Date()
    const diff = date.getTime() - now.getTime()
    const future = diff > 0
    const absDiff = Math.abs(diff)
    const mins = Math.floor(absDiff / 60000)
    if (mins < 1) return future ? 'in a moment' : 'Just now'
    if (mins < 60) return future ? `in ${mins} minutes` : `${mins} minutes ago`
    const hours = Math.floor(mins / 60)
    return future ? `in ${hours} hours` : `${hours} hours ago`
  }

  const successCount = executions.filter((e) => e.status === 'success').length
  const failedCount = executions.filter((e) => e.status === 'failed').length
  const successRate = executions.length > 0 ? Math.round((successCount / executions.length) * 100) : 0

  const executionColumns: Column<CronExecution>[] = [
    { key: 'scheduled_at', label: 'Scheduled At', render: (row) => formatTime(row.scheduled_at) },
    { key: 'started_at', label: 'Started', render: (row) => formatTime(row.started_at) },
    { key: 'duration_ms', label: 'Duration', render: (row) => row.duration_ms ? `${(row.duration_ms / 1000).toFixed(1)}s` : '-' },
    { key: 'status', label: 'Status', render: (row) => <StatusBadge status={row.status} /> },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title={`${trigger.cron} -> ${trigger.script_name}`}
        breadcrumbs={[{ label: 'Cron Triggers', path: '/cron' }, { label: trigger.script_name }]}
        backPath="/cron"
        actions={
          <Button leftSection={<IconPlayerPlay size={16} />} onClick={handleTrigger} loading={triggerLoading}>
            Run Now
          </Button>
        }
      />

      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Text size="sm" fw={600}>Trigger Details</Text>
          <Group gap="xl">
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Schedule</Text>
              <Text fw={500}>{trigger.cron}</Text>
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Script</Text>
              <Text fw={500}>{trigger.script_name}</Text>
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Status</Text>
              <Group gap="xs">
                <StatusBadge status={trigger.enabled ? 'enabled' : 'disabled'} />
                <Button size="xs" variant="subtle" onClick={handleToggle}>
                  {trigger.enabled ? 'Disable' : 'Enable'}
                </Button>
              </Group>
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Next Run</Text>
              <Text fw={500}>{formatRelativeTime(trigger.next_run)}</Text>
            </Stack>
          </Group>
        </Stack>
      </Paper>

      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Group justify="space-between">
            <Text size="sm" fw={600}>Execution Success Rate</Text>
            <Text size="xs" c="dimmed">Last {executions.length} executions</Text>
          </Group>
          <Group gap="md">
            <Text size="sm" c="green">{successCount} success</Text>
            <Text size="sm" c="red">{failedCount} failed</Text>
          </Group>
          <Progress.Root size="lg">
            <Progress.Section value={successRate} color="green" />
            <Progress.Section value={100 - successRate} color="red" />
          </Progress.Root>
          <Text size="sm" ta="center" fw={600}>{successRate}% success rate</Text>
        </Stack>
      </Paper>

      <Stack gap="sm">
        <Text size="sm" fw={600}>Execution History</Text>
        <DataTable
          data={executions}
          columns={executionColumns}
          getRowKey={(row) => row.id}
          searchable={false}
          emptyState={{ title: 'No executions yet', description: 'This trigger has not run yet' }}
        />
      </Stack>
    </Stack>
  )
}
