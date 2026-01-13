import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Stack, SimpleGrid, Paper, Text, Group, Button } from '@mantine/core'
import { IconList } from '@tabler/icons-react'
import { PageHeader, StatCard, AreaChart, DataTable, LoadingState, StatusBadge, type Column } from '../components/common'
import { api } from '../api/client'
import type { AIGateway, AIGatewayLog, TimeSeriesData } from '../types'

export function AIGatewayDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [gateway, setGateway] = useState<AIGateway | null>(null)
  const [logs, setLogs] = useState<AIGatewayLog[]>([])
  const [loading, setLoading] = useState(true)
  const [timeSeries, setTimeSeries] = useState<TimeSeriesData[]>([])
  const [timeRange, setTimeRange] = useState<'1h' | '24h' | '7d' | '30d'>('24h')

  useEffect(() => {
    if (id) loadData()
  }, [id])

  useEffect(() => {
    // Generate mock time series
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
    try {
      const [gwRes, logsRes] = await Promise.all([
        api.aiGateway.get(id!),
        api.aiGateway.getLogs(id!, { limit: 5 }),
      ])
      if (gwRes.result) setGateway(gwRes.result)
      if (logsRes.result) setLogs(logsRes.result.logs ?? [])
    } catch (error) {
      setGateway({
        id: id!,
        name: 'prod-gateway',
        created_at: new Date().toISOString(),
        settings: { cache_enabled: true, cache_ttl: 300, rate_limit_enabled: true, rate_limit: 100, rate_limit_period: '1m', logging_enabled: true, retry_enabled: true, retry_count: 3 },
        stats: { total_requests: 45200, cached_requests: 35300, error_count: 234, total_tokens: 8900000, total_cost: 12.45 },
      })
      setLogs([
        { id: '1', gateway_id: id!, timestamp: new Date().toISOString(), provider: 'OpenAI', model: 'gpt-4', status: 200, latency_ms: 1200, tokens: 1234, cost: 0.04, cached: false },
        { id: '2', gateway_id: id!, timestamp: new Date(Date.now() - 2000).toISOString(), provider: 'Anthropic', model: 'claude-3', status: 200, latency_ms: 800, tokens: 890, cost: 0.02, cached: false },
        { id: '3', gateway_id: id!, timestamp: new Date(Date.now() - 4000).toISOString(), provider: 'OpenAI', model: 'gpt-4', status: 'CACHED', latency_ms: 12, tokens: 0, cost: 0, cached: true },
        { id: '4', gateway_id: id!, timestamp: new Date(Date.now() - 6000).toISOString(), provider: 'Cohere', model: 'command', status: 429, latency_ms: 0, tokens: 0, cost: 0, cached: false },
      ])
    } finally {
      setLoading(false)
    }
  }

  if (loading) return <LoadingState />
  if (!gateway) return <Text>Gateway not found</Text>

  const logColumns: Column<AIGatewayLog>[] = [
    { key: 'timestamp', label: 'Time', render: (row) => new Date(row.timestamp).toLocaleTimeString() },
    { key: 'provider', label: 'Provider' },
    { key: 'model', label: 'Model' },
    { key: 'status', label: 'Status', render: (row) => row.cached ? <StatusBadge status="cached" /> : <StatusBadge status={row.status === 200 ? 'success' : 'error'} label={String(row.status)} /> },
    { key: 'latency_ms', label: 'Latency', render: (row) => row.latency_ms > 0 ? `${row.latency_ms}ms` : '-' },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title={gateway.name}
        breadcrumbs={[{ label: 'AI Gateway', path: '/ai-gateway' }, { label: gateway.name }]}
        backPath="/ai-gateway"
        actions={
          <Button leftSection={<IconList size={16} />} onClick={() => navigate(`/ai-gateway/${id}/logs`)}>
            View All Logs
          </Button>
        }
      />

      <SimpleGrid cols={{ base: 2, sm: 5 }} spacing="md">
        <StatCard icon={<Text size="sm" fw={700}>R</Text>} label="Requests" value={(gateway.stats?.total_requests ?? 0).toLocaleString()} color="orange" />
        <StatCard icon={<Text size="sm" fw={700}>C</Text>} label="Cached" value={(gateway.stats?.cached_requests ?? 0).toLocaleString()} color="success" />
        <StatCard icon={<Text size="sm" fw={700}>E</Text>} label="Errors" value={gateway.stats?.error_count ?? 0} color="error" />
        <StatCard icon={<Text size="sm" fw={700}>T</Text>} label="Tokens" value={`${((gateway.stats?.total_tokens ?? 0) / 1000000).toFixed(1)}M`} />
        <StatCard icon={<Text size="sm" fw={700}>$</Text>} label="Cost" value={`$${(gateway.stats?.total_cost ?? 0).toFixed(2)}`} />
      </SimpleGrid>

      <AreaChart
        data={timeSeries}
        title="Request Volume"
        timeRange={timeRange}
        onTimeRangeChange={setTimeRange}
        height={200}
      />

      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Text size="sm" fw={600}>Configuration</Text>
          <Group gap="xl">
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Caching</Text>
              <StatusBadge status={(gateway.settings?.cache_enabled ?? gateway.cache_enabled) ? 'enabled' : 'disabled'} />
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">TTL</Text>
              <Text fw={500}>{gateway.settings?.cache_ttl ?? gateway.cache_ttl ?? 0}s</Text>
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Rate Limit</Text>
              <Text fw={500}>{(gateway.settings?.rate_limit_enabled ?? gateway.rate_limit_enabled) ? `${gateway.settings?.rate_limit ?? gateway.rate_limit_count ?? 0}/min` : 'Disabled'}</Text>
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Logging</Text>
              <StatusBadge status={(gateway.settings?.logging_enabled ?? gateway.collect_logs) ? 'enabled' : 'disabled'} />
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Retry</Text>
              <Text fw={500}>{gateway.settings?.retry_enabled ? `${gateway.settings.retry_count} attempts` : 'Disabled'}</Text>
            </Stack>
          </Group>
        </Stack>
      </Paper>

      <Stack gap="sm">
        <Group justify="space-between">
          <Text size="sm" fw={600}>Recent Logs</Text>
          <Button variant="subtle" size="xs" onClick={() => navigate(`/ai-gateway/${id}/logs`)}>
            View All
          </Button>
        </Group>
        <DataTable
          data={logs}
          columns={logColumns}
          getRowKey={(row) => row.id}
          searchable={false}
        />
      </Stack>
    </Stack>
  )
}
