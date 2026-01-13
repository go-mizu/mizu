import { useState, useEffect } from 'react'
import { Stack, SimpleGrid, Paper, Text, Group, Button, Select, Textarea, Code, Badge, ActionIcon, Tabs, SegmentedControl } from '@mantine/core'
import { IconRefresh, IconPlayerPlay, IconDownload } from '@tabler/icons-react'
import { PageHeader, StatCard, AreaChart, LoadingState, StatusBadge, DataTable, type Column } from '../components/common'
import { api } from '../api/client'
import type { LogEntry, Trace, TimeSeriesData } from '../types'

export function Observability() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [traces, setTraces] = useState<Trace[]>([])
  const [loading, setLoading] = useState(true)
  const [query, setQuery] = useState('')
  const [timeRange, setTimeRange] = useState<'1h' | '24h' | '7d' | '30d'>('1h')
  const [logLevel, setLogLevel] = useState<string | null>(null)
  const [workerFilter, setWorkerFilter] = useState<string | null>(null)
  const [timeSeries, setTimeSeries] = useState<TimeSeriesData[]>([])
  const [selectedTrace, setSelectedTrace] = useState<Trace | null>(null)

  useEffect(() => {
    loadData()
  }, [timeRange, logLevel, workerFilter])

  useEffect(() => {
    const now = Date.now()
    const points = timeRange === '1h' ? 60 : timeRange === '24h' ? 24 : 7
    const interval = timeRange === '1h' ? 60000 : timeRange === '24h' ? 3600000 : 86400000
    setTimeSeries(
      Array.from({ length: points }, (_, i) => ({
        timestamp: new Date(now - (points - i) * interval).toISOString(),
        value: Math.floor(Math.random() * 500) + 100,
      }))
    )
  }, [timeRange])

  const loadData = async () => {
    setLoading(true)
    try {
      const [logsRes, tracesRes] = await Promise.all([
        api.observability.getLogs({ level: logLevel || undefined, worker: workerFilter || undefined }),
        api.observability.getTraces(),
      ])
      if (logsRes.result) setLogs(logsRes.result.logs ?? [])
      if (tracesRes.result) setTraces(tracesRes.result.traces ?? [])
    } catch (error) {
      // Mock data
      setLogs([
        { timestamp: new Date(Date.now() - 1000).toISOString(), level: 'info', message: 'Request completed successfully', worker: 'api-gateway', request_id: 'req-123', duration_ms: 45 },
        { timestamp: new Date(Date.now() - 2000).toISOString(), level: 'info', message: 'Cache hit for /api/users', worker: 'api-gateway', request_id: 'req-124', duration_ms: 12 },
        { timestamp: new Date(Date.now() - 3000).toISOString(), level: 'warn', message: 'Rate limit approaching threshold', worker: 'auth-worker', request_id: 'req-125', duration_ms: 34 },
        { timestamp: new Date(Date.now() - 4000).toISOString(), level: 'error', message: 'Database connection timeout', worker: 'api-gateway', request_id: 'req-126', duration_ms: 5000 },
        { timestamp: new Date(Date.now() - 5000).toISOString(), level: 'info', message: 'User session created', worker: 'auth-worker', request_id: 'req-127', duration_ms: 89 },
        { timestamp: new Date(Date.now() - 6000).toISOString(), level: 'debug', message: 'Processing webhook payload', worker: 'webhook-handler', request_id: 'req-128', duration_ms: 156 },
        { timestamp: new Date(Date.now() - 7000).toISOString(), level: 'info', message: 'Image resized successfully', worker: 'image-processor', request_id: 'req-129', duration_ms: 234 },
        { timestamp: new Date(Date.now() - 8000).toISOString(), level: 'error', message: 'Invalid API key', worker: 'api-gateway', request_id: 'req-130', duration_ms: 5 },
      ])
      setTraces([
        {
          trace_id: 'trace-1',
          root_span: 'fetch',
          worker: 'api-gateway',
          timestamp: new Date(Date.now() - 1000).toISOString(),
          duration_ms: 245,
          status: 'ok',
          spans: [
            { name: 'fetch', duration_ms: 245, start_ms: 0 },
            { name: 'KV.get', duration_ms: 12, start_ms: 5 },
            { name: 'D1.query', duration_ms: 45, start_ms: 20 },
            { name: 'Response', duration_ms: 2, start_ms: 243 },
          ],
        },
        {
          trace_id: 'trace-2',
          root_span: 'fetch',
          worker: 'auth-worker',
          timestamp: new Date(Date.now() - 5000).toISOString(),
          duration_ms: 89,
          status: 'ok',
          spans: [
            { name: 'fetch', duration_ms: 89, start_ms: 0 },
            { name: 'Durable Object', duration_ms: 34, start_ms: 10 },
            { name: 'Response', duration_ms: 1, start_ms: 88 },
          ],
        },
        {
          trace_id: 'trace-3',
          root_span: 'fetch',
          worker: 'api-gateway',
          timestamp: new Date(Date.now() - 4000).toISOString(),
          duration_ms: 5000,
          status: 'error',
          spans: [
            { name: 'fetch', duration_ms: 5000, start_ms: 0 },
            { name: 'D1.query (timeout)', duration_ms: 5000, start_ms: 0 },
          ],
        },
      ])
    } finally {
      setLoading(false)
    }
  }

  const executeQuery = async () => {
    // Execute custom query
    notifications.show({ title: 'Query Executed', message: 'Results updated', color: 'green' })
  }

  const formatTime = (ts: string) => {
    const date = new Date(ts)
    return date.toLocaleTimeString()
  }

  const getLevelColor = (level: string) => {
    switch (level) {
      case 'error': return 'red'
      case 'warn': return 'yellow'
      case 'info': return 'blue'
      case 'debug': return 'gray'
      default: return 'gray'
    }
  }

  const logColumns: Column<LogEntry>[] = [
    { key: 'timestamp', label: 'Time', render: (row) => formatTime(row.timestamp) },
    { key: 'level', label: 'Level', render: (row) => <Badge size="xs" color={getLevelColor(row.level)}>{row.level}</Badge> },
    { key: 'worker', label: 'Worker', render: (row) => <Code>{row.worker}</Code> },
    { key: 'message', label: 'Message', render: (row) => <Text size="sm" truncate style={{ maxWidth: 400 }}>{row.message}</Text> },
    { key: 'duration_ms', label: 'Duration', render: (row) => `${row.duration_ms}ms` },
  ]

  const traceColumns: Column<Trace>[] = [
    { key: 'timestamp', label: 'Time', render: (row) => formatTime(row.timestamp) },
    { key: 'worker', label: 'Worker', render: (row) => <Code>{row.worker}</Code> },
    { key: 'root_span', label: 'Operation' },
    { key: 'duration_ms', label: 'Duration', render: (row) => `${row.duration_ms}ms` },
    { key: 'status', label: 'Status', render: (row) => <StatusBadge status={row.status} /> },
  ]

  if (loading && logs.length === 0) return <LoadingState />

  return (
    <Stack gap="lg">
      <PageHeader
        title="Observability"
        subtitle="Logs, metrics, and traces for all Workers"
        actions={
          <Group>
            <SegmentedControl
              value={timeRange}
              onChange={(v) => setTimeRange(v as typeof timeRange)}
              data={[
                { label: '1h', value: '1h' },
                { label: '24h', value: '24h' },
                { label: '7d', value: '7d' },
                { label: '30d', value: '30d' },
              ]}
              size="xs"
            />
            <Button variant="light" leftSection={<IconDownload size={16} />}>Export</Button>
          </Group>
        }
      />

      <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
        <StatCard icon={<Text size="sm" fw={700}>R</Text>} label="Requests" value="12.4K" description={timeRange} color="orange" />
        <StatCard icon={<Text size="sm" fw={700}>E</Text>} label="Errors" value="23" description={timeRange} color="error" />
        <StatCard icon={<Text size="sm" fw={700}>L</Text>} label="Avg Latency" value="45ms" />
        <StatCard icon={<Text size="sm" fw={700}>CPU</Text>} label="CPU Time" value="2.3ms" description="avg" />
      </SimpleGrid>

      <AreaChart
        data={timeSeries}
        title="Request Volume"
        timeRange={timeRange}
        onTimeRangeChange={setTimeRange}
        height={200}
      />

      <Tabs defaultValue="logs">
        <Tabs.List>
          <Tabs.Tab value="logs">Logs</Tabs.Tab>
          <Tabs.Tab value="traces">Traces</Tabs.Tab>
          <Tabs.Tab value="query">Query Builder</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="logs" pt="md">
          <Paper p="md" radius="md" withBorder>
            <Stack gap="md">
              <Group justify="space-between">
                <Text size="sm" fw={600}>Real-time Logs</Text>
                <Group>
                  <Select
                    placeholder="All levels"
                    data={[
                      { value: 'error', label: 'Error' },
                      { value: 'warn', label: 'Warning' },
                      { value: 'info', label: 'Info' },
                      { value: 'debug', label: 'Debug' },
                    ]}
                    value={logLevel}
                    onChange={setLogLevel}
                    clearable
                    size="xs"
                    w={120}
                  />
                  <Select
                    placeholder="All workers"
                    data={[
                      { value: 'api-gateway', label: 'api-gateway' },
                      { value: 'auth-worker', label: 'auth-worker' },
                      { value: 'image-processor', label: 'image-processor' },
                      { value: 'webhook-handler', label: 'webhook-handler' },
                    ]}
                    value={workerFilter}
                    onChange={setWorkerFilter}
                    clearable
                    size="xs"
                    w={150}
                  />
                  <ActionIcon variant="light" onClick={loadData}>
                    <IconRefresh size={14} />
                  </ActionIcon>
                </Group>
              </Group>

              <DataTable
                data={logs}
                columns={logColumns}
                getRowKey={(row) => row.request_id}
                searchable={false}
              />
            </Stack>
          </Paper>
        </Tabs.Panel>

        <Tabs.Panel value="traces" pt="md">
          <Paper p="md" radius="md" withBorder>
            <Stack gap="md">
              <Group justify="space-between">
                <Text size="sm" fw={600}>Distributed Traces</Text>
                <ActionIcon variant="light" onClick={loadData}>
                  <IconRefresh size={14} />
                </ActionIcon>
              </Group>

              <DataTable
                data={traces}
                columns={traceColumns}
                getRowKey={(row) => row.trace_id}
                searchable={false}
                onRowClick={(row) => setSelectedTrace(row)}
              />

              {selectedTrace && (
                <Paper p="md" radius="sm" bg="dark.7">
                  <Stack gap="md">
                    <Group justify="space-between">
                      <Text size="sm" fw={600}>Trace: {selectedTrace.trace_id}</Text>
                      <ActionIcon variant="subtle" onClick={() => setSelectedTrace(null)}>
                        <IconClock size={14} />
                      </ActionIcon>
                    </Group>
                    <Stack gap="xs">
                      {selectedTrace.spans?.map((span, i) => (
                        <Group key={i} gap="md">
                          <Text size="xs" c="dimmed" w={80}>{span.start_ms}ms</Text>
                          <Paper
                            p="xs"
                            radius="sm"
                            bg="orange.9"
                            style={{ width: `${Math.max(10, (span.duration_ms / selectedTrace.duration_ms) * 100)}%`, minWidth: 80 }}
                          >
                            <Text size="xs" truncate>{span.name} ({span.duration_ms}ms)</Text>
                          </Paper>
                        </Group>
                      ))}
                    </Stack>
                  </Stack>
                </Paper>
              )}
            </Stack>
          </Paper>
        </Tabs.Panel>

        <Tabs.Panel value="query" pt="md">
          <Paper p="md" radius="md" withBorder>
            <Stack gap="md">
              <Text size="sm" fw={600}>Query Builder</Text>
              <Text size="xs" c="dimmed">
                Write structured queries to search logs, extract metrics, and create visualizations.
              </Text>
              <Textarea
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={`SELECT timestamp, level, message, worker
FROM logs
WHERE timestamp > now() - interval '1 hour'
  AND level = 'error'
ORDER BY timestamp DESC
LIMIT 100`}
                minRows={6}
                styles={{ input: { fontFamily: 'monospace', fontSize: 12 } }}
              />
              <Group justify="flex-end">
                <Button leftSection={<IconPlayerPlay size={14} />} onClick={executeQuery}>
                  Run Query
                </Button>
              </Group>
            </Stack>
          </Paper>
        </Tabs.Panel>
      </Tabs>
    </Stack>
  )
}

const notifications = {
  show: (opts: { title: string; message: string; color: string }) => {
    // Mock notification
    console.log(`[${opts.color}] ${opts.title}: ${opts.message}`)
  },
}

const IconClock = ({ size }: { size: number }) => (
  <svg xmlns="http://www.w3.org/2000/svg" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="12" r="10" />
    <polyline points="12 6 12 12 16 14" />
  </svg>
)
