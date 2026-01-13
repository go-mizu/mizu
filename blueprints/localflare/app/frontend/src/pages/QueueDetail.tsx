import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Stack, SimpleGrid, Paper, Text, Group, Button, Modal, Textarea, SegmentedControl, NumberInput } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconSend } from '@tabler/icons-react'
import { PageHeader, StatCard, AreaChart, DataTable, LoadingState, StatusBadge, type Column } from '../components/common'
import { api } from '../api/client'
import type { Queue, QueueConsumer, TimeSeriesData } from '../types'

export function QueueDetail() {
  const { id } = useParams<{ id: string }>()
  const [queue, setQueue] = useState<Queue | null>(null)
  const [loading, setLoading] = useState(true)
  const [sendModalOpen, setSendModalOpen] = useState(false)
  const [timeSeries, setTimeSeries] = useState<TimeSeriesData[]>([])
  const [timeRange, setTimeRange] = useState<'1h' | '24h' | '7d' | '30d'>('1h')

  const sendForm = useForm({
    initialValues: { body: '{\n  "key": "value"\n}', content_type: 'json' as const, delay_seconds: 0 },
  })

  useEffect(() => {
    if (id) loadQueue()
  }, [id])

  useEffect(() => {
    // Generate mock throughput data
    const now = Date.now()
    const points = timeRange === '1h' ? 60 : 24
    const interval = timeRange === '1h' ? 60000 : 3600000
    setTimeSeries(
      Array.from({ length: points }, (_, i) => ({
        timestamp: new Date(now - (points - i) * interval).toISOString(),
        value: Math.floor(Math.random() * 100) + 20,
      }))
    )
  }, [timeRange])

  const loadQueue = async () => {
    try {
      const res = await api.queues.get(id!)
      if (res.result) setQueue(res.result)
    } catch (error) {
      setQueue({
        id: id!,
        name: 'order-events',
        created_at: new Date().toISOString(),
        message_count: 1234,
        ready_count: 1100,
        delayed_count: 134,
        failed_count: 12,
        settings: { max_retries: 3, batch_size: 10, max_batch_timeout: 30, message_retention_seconds: 345600, delivery_delay: 0 },
        consumers: [
          { id: '1', queue_id: id!, script_name: 'order-processor', consumer_type: 'worker', batch_size: 10, max_retries: 3, status: 'active' },
          { id: '2', queue_id: id!, script_name: 'http-pull', consumer_type: 'http', batch_size: 100, max_retries: 3, status: 'active' },
        ],
      })
    } finally {
      setLoading(false)
    }
  }

  const handleSendMessage = async (values: typeof sendForm.values) => {
    try {
      await api.queues.sendMessage(id!, { id: '', body: values.body, content_type: values.content_type, delay_seconds: values.delay_seconds })
      notifications.show({ title: 'Success', message: 'Message sent', color: 'green' })
      setSendModalOpen(false)
      sendForm.reset()
      loadQueue()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to send message', color: 'red' })
    }
  }

  if (loading) return <LoadingState />
  if (!queue) return <Text>Queue not found</Text>

  const consumerColumns: Column<QueueConsumer>[] = [
    { key: 'script_name', label: 'Script', sortable: true },
    { key: 'consumer_type', label: 'Type', sortable: true, render: (row) => row.consumer_type.toUpperCase() },
    { key: 'batch_size', label: 'Batch Size', sortable: true },
    { key: 'status', label: 'Status', render: (row) => <StatusBadge status={row.status} /> },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title={queue.name}
        breadcrumbs={[{ label: 'Queues', path: '/queues' }, { label: queue.name }]}
        backPath="/queues"
        actions={
          <Button leftSection={<IconSend size={16} />} onClick={() => setSendModalOpen(true)}>
            Send Message
          </Button>
        }
      />

      <SimpleGrid cols={{ base: 2, sm: 5 }} spacing="md">
        <StatCard icon={<Text size="sm" fw={700}>#</Text>} label="Total" value={queue.message_count} color="orange" />
        <StatCard icon={<Text size="sm" fw={700}>R</Text>} label="Ready" value={queue.ready_count} color="success" />
        <StatCard icon={<Text size="sm" fw={700}>D</Text>} label="Delayed" value={queue.delayed_count} color="warning" />
        <StatCard icon={<Text size="sm" fw={700}>F</Text>} label="Failed" value={queue.failed_count} color="error" />
        <StatCard icon={<Text size="sm" fw={700}>/s</Text>} label="Rate" value="45/s" />
      </SimpleGrid>

      <AreaChart
        data={timeSeries}
        title="Message Throughput"
        timeRange={timeRange}
        onTimeRangeChange={setTimeRange}
        height={200}
      />

      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Text size="sm" fw={600}>Settings</Text>
          <Group gap="xl">
            <Stack gap={2}><Text size="xs" c="dimmed">Max Retries</Text><Text fw={500}>{queue.settings.max_retries}</Text></Stack>
            <Stack gap={2}><Text size="xs" c="dimmed">Batch Size</Text><Text fw={500}>{queue.settings.batch_size}</Text></Stack>
            <Stack gap={2}><Text size="xs" c="dimmed">Timeout</Text><Text fw={500}>{queue.settings.max_batch_timeout}s</Text></Stack>
            <Stack gap={2}><Text size="xs" c="dimmed">Message TTL</Text><Text fw={500}>{Math.round(queue.settings.message_retention_seconds / 86400)} days</Text></Stack>
            <Stack gap={2}><Text size="xs" c="dimmed">Delivery Delay</Text><Text fw={500}>{queue.settings.delivery_delay}s</Text></Stack>
          </Group>
        </Stack>
      </Paper>

      <Stack gap="sm">
        <Text size="sm" fw={600}>Consumers</Text>
        <DataTable
          data={queue.consumers}
          columns={consumerColumns}
          getRowKey={(row) => row.id}
          searchable={false}
          emptyState={{ title: 'No consumers', description: 'Add a consumer to process messages' }}
        />
      </Stack>

      <Modal opened={sendModalOpen} onClose={() => setSendModalOpen(false)} title={`Send Message to ${queue.name}`} size="lg">
        <form onSubmit={sendForm.onSubmit(handleSendMessage)}>
          <Stack gap="md">
            <SegmentedControl data={[{ value: 'json', label: 'JSON' }, { value: 'text', label: 'Text' }, { value: 'bytes', label: 'Bytes' }]} {...sendForm.getInputProps('content_type')} />
            <Textarea label="Message Body" minRows={6} styles={{ input: { fontFamily: 'monospace' } }} {...sendForm.getInputProps('body')} />
            <NumberInput label="Delay (seconds)" min={0} {...sendForm.getInputProps('delay_seconds')} />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setSendModalOpen(false)}>Cancel</Button>
              <Button type="submit">Send Message</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}
