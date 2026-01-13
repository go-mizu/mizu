import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Stack, SimpleGrid, Paper, Text, Group } from '@mantine/core'
import { PageHeader, StatCard, AreaChart, LoadingState, StatusBadge } from '../components/common'
import { api } from '../api/client'
import type { HyperdriveConfig, HyperdriveStats, TimeSeriesData } from '../types'

export function HyperdriveDetail() {
  const { id } = useParams<{ id: string }>()
  const [config, setConfig] = useState<HyperdriveConfig | null>(null)
  const [stats, setStats] = useState<HyperdriveStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [timeSeries, setTimeSeries] = useState<TimeSeriesData[]>([])
  const [timeRange, setTimeRange] = useState<'1h' | '24h' | '7d' | '30d'>('1h')

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
        value: Math.floor(Math.random() * 300) + 100,
      }))
    )
  }, [timeRange])

  const loadData = async () => {
    try {
      const [configRes, statsRes] = await Promise.all([
        api.hyperdrive.get(id!),
        api.hyperdrive.getStats(id!),
      ])
      if (configRes.result) setConfig(configRes.result)
      if (statsRes.result) setStats(statsRes.result)
    } catch (error) {
      setConfig({
        id: id!,
        name: 'prod-postgres',
        created_at: new Date().toISOString(),
        origin: { scheme: 'postgres', host: 'db.example.com', port: 5432, database: 'app_db', user: 'app_user' },
        caching: { enabled: true, max_age: 60, stale_while_revalidate: 15 },
        status: 'connected',
      })
      setStats({
        active_connections: 12,
        idle_connections: 38,
        total_connections: 50,
        queries_per_second: 234,
        cache_hit_rate: 89,
      })
    } finally {
      setLoading(false)
    }
  }

  if (loading) return <LoadingState />
  if (!config) return <Text>Config not found</Text>

  return (
    <Stack gap="lg">
      <PageHeader
        title={config.name}
        breadcrumbs={[{ label: 'Hyperdrive', path: '/hyperdrive' }, { label: config.name }]}
        backPath="/hyperdrive"
      />

      <Text size="sm" fw={600}>Connection Pool Stats</Text>
      <SimpleGrid cols={{ base: 2, sm: 5 }} spacing="md">
        <StatCard icon={<Text size="sm" fw={700}>A</Text>} label="Active" value={stats?.active_connections ?? 0} color="orange" />
        <StatCard icon={<Text size="sm" fw={700}>I</Text>} label="Idle" value={stats?.idle_connections ?? 0} />
        <StatCard icon={<Text size="sm" fw={700}>T</Text>} label="Total" value={stats?.total_connections ?? 0} />
        <StatCard icon={<Text size="sm" fw={700}>Q</Text>} label="QPS" value={stats?.queries_per_second ?? 0} />
        <StatCard icon={<Text size="sm" fw={700}>%</Text>} label="Cache Hit" value={`${stats?.cache_hit_rate ?? 0}%`} color="success" />
      </SimpleGrid>

      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Text size="sm" fw={600}>Connection Details</Text>
          <Group gap="xl">
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Host</Text>
              <Text fw={500}>{config.origin.host}</Text>
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Port</Text>
              <Text fw={500}>{config.origin.port}</Text>
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Database</Text>
              <Text fw={500}>{config.origin.database}</Text>
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">User</Text>
              <Text fw={500}>{config.origin.user}</Text>
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Scheme</Text>
              <Text fw={500}>{config.origin.scheme}</Text>
            </Stack>
          </Group>
        </Stack>
      </Paper>

      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Text size="sm" fw={600}>Caching Settings</Text>
          <Group gap="xl">
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Caching</Text>
              <StatusBadge status={config.caching.enabled ? 'enabled' : 'disabled'} />
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Max Age</Text>
              <Text fw={500}>{config.caching.max_age} seconds</Text>
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Stale While Revalidate</Text>
              <Text fw={500}>{config.caching.stale_while_revalidate} seconds</Text>
            </Stack>
          </Group>
        </Stack>
      </Paper>

      <AreaChart
        data={timeSeries}
        title="Query Performance"
        timeRange={timeRange}
        onTimeRangeChange={setTimeRange}
        height={200}
      />
    </Stack>
  )
}
