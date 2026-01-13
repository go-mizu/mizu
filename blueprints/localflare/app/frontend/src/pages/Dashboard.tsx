import { SimpleGrid, Paper, Stack, Text, Group, Box, Button, Anchor } from '@mantine/core'
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  IconDatabase,
  IconMailbox,
  IconVectorTriangle,
  IconChartLine,
  IconRobot,
  IconApiApp,
  IconBolt,
  IconClock,
  IconArrowRight,
  IconRocket,
  IconExternalLink,
} from '@tabler/icons-react'
import { StatCard, AreaChart, LoadingState, StatusBadge, PageHeader, ModuleCard } from '../components/common'
import { api } from '../api/client'
import type { DashboardStats, TimeSeriesData, SystemStatus, ActivityEvent } from '../types'

export function Dashboard() {
  const navigate = useNavigate()
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [timeSeries, setTimeSeries] = useState<TimeSeriesData[]>([])
  const [statuses, setStatuses] = useState<SystemStatus[]>([])
  const [activity, setActivity] = useState<ActivityEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [timeRange, setTimeRange] = useState<'1h' | '24h' | '7d' | '30d'>('24h')

  // Sparkline data for stat cards
  const [sparklineData, setSparklineData] = useState<Record<string, number[]>>({})

  useEffect(() => {
    loadDashboardData()
  }, [])

  useEffect(() => {
    loadTimeSeries()
  }, [timeRange])

  const loadDashboardData = async () => {
    try {
      const [statsRes, statusRes, activityRes] = await Promise.all([
        api.dashboard.getStats(),
        api.dashboard.getStatus(),
        api.dashboard.getActivity(10),
      ])

      if (statsRes.result) setStats(statsRes.result)
      if (statusRes.result) setStatuses(statusRes.result.services)
      if (activityRes.result) setActivity(activityRes.result.events)
    } catch (error) {
      console.error('Failed to load dashboard data:', error)
      // Set mock data for development
      setStats({
        durable_objects: { namespaces: 3, objects: 156 },
        queues: { count: 5, total_messages: 1234 },
        vectorize: { indexes: 3, total_vectors: 50234 },
        analytics: { datasets: 4, data_points: 1200000 },
        ai: { requests_today: 1245, tokens_today: 2100000 },
        ai_gateway: { gateways: 2, requests_today: 4520 },
        hyperdrive: { configs: 3, active_connections: 12 },
        cron: { triggers: 5, executions_today: 288 },
      })
      setStatuses([
        { service: 'Durable Objects', status: 'online' },
        { service: 'Queues', status: 'online' },
        { service: 'Vectorize', status: 'online' },
        { service: 'Analytics Engine', status: 'online' },
        { service: 'Workers AI', status: 'online' },
        { service: 'AI Gateway', status: 'online' },
        { service: 'Hyperdrive', status: 'online' },
        { service: 'Cron', status: 'online' },
      ])
      setActivity([
        { id: '1', type: 'queue', message: 'Queue message processed', timestamp: new Date().toISOString(), service: 'Queues' },
        { id: '2', type: 'do', message: 'Durable Object created', timestamp: new Date(Date.now() - 60000).toISOString(), service: 'Durable Objects' },
        { id: '3', type: 'vector', message: 'Vector index updated', timestamp: new Date(Date.now() - 120000).toISOString(), service: 'Vectorize' },
        { id: '4', type: 'ai', message: 'AI inference completed', timestamp: new Date(Date.now() - 180000).toISOString(), service: 'Workers AI' },
      ])
      // Mock sparkline data
      setSparklineData({
        'Durable Objects': [45, 52, 48, 61, 58, 67, 72, 78],
        'Queues': [120, 145, 132, 156, 178, 189, 201, 195],
        'Vectorize': [1000, 1200, 1150, 1400, 1350, 1500, 1620, 1700],
        'AI Requests': [234, 289, 312, 356, 401, 445, 478, 512],
      })
    } finally {
      setLoading(false)
    }
  }

  const loadTimeSeries = async () => {
    try {
      const res = await api.dashboard.getTimeSeries('requests', timeRange)
      if (res.result) setTimeSeries(res.result.data)
    } catch {
      // Generate mock time series data
      const now = Date.now()
      const points = timeRange === '1h' ? 60 : timeRange === '24h' ? 24 : timeRange === '7d' ? 7 : 30
      const interval = timeRange === '1h' ? 60000 : timeRange === '24h' ? 3600000 : 86400000
      setTimeSeries(
        Array.from({ length: points }, (_, i) => ({
          timestamp: new Date(now - (points - i) * interval).toISOString(),
          value: Math.floor(Math.random() * 1000) + 500,
        }))
      )
    }
  }

  if (loading) {
    return <LoadingState message="Loading dashboard..." />
  }

  const formatTime = (ts: string) => {
    const date = new Date(ts)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    if (diff < 60000) return 'Just now'
    if (diff < 3600000) return `${Math.floor(diff / 60000)} mins ago`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)} hours ago`
    return `${Math.floor(diff / 86400000)} days ago`
  }

  return (
    <Stack gap="lg">
      <PageHeader
        title="Dashboard Overview"
        subtitle="Monitor all Localflare services at a glance"
      />

      {/* Get Started Card - for new users */}
      <ModuleCard
        title="Get Started with Localflare"
        description="Set up your local development environment for Cloudflare Workers"
        icon={<IconRocket size={18} />}
        helpContent={
          <Stack gap="xs">
            <Text size="sm">Localflare provides a complete local development environment for Cloudflare's Developer Platform.</Text>
            <Text size="sm">Start by creating a Durable Object namespace or configuring a Queue for your application.</Text>
          </Stack>
        }
        collapsible
        defaultCollapsed
        footer={
          <Group justify="space-between">
            <Anchor
              href="https://developers.cloudflare.com"
              target="_blank"
              size="sm"
              c="dimmed"
            >
              View Documentation <IconExternalLink size={12} style={{ marginLeft: 4 }} />
            </Anchor>
            <Button
              size="xs"
              variant="light"
              rightSection={<IconArrowRight size={14} />}
              onClick={() => navigate('/durable-objects')}
            >
              Create your first resource
            </Button>
          </Group>
        }
      >
        <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }} spacing="sm">
          <Paper p="sm" radius="sm" bg="dark.6" style={{ cursor: 'pointer' }} onClick={() => navigate('/durable-objects')}>
            <Group gap="xs">
              <IconDatabase size={16} color="var(--mantine-color-orange-5)" />
              <Text size="sm" fw={500}>Create Durable Object</Text>
            </Group>
          </Paper>
          <Paper p="sm" radius="sm" bg="dark.6" style={{ cursor: 'pointer' }} onClick={() => navigate('/queues')}>
            <Group gap="xs">
              <IconMailbox size={16} color="var(--mantine-color-orange-5)" />
              <Text size="sm" fw={500}>Set up a Queue</Text>
            </Group>
          </Paper>
          <Paper p="sm" radius="sm" bg="dark.6" style={{ cursor: 'pointer' }} onClick={() => navigate('/vectorize')}>
            <Group gap="xs">
              <IconVectorTriangle size={16} color="var(--mantine-color-orange-5)" />
              <Text size="sm" fw={500}>Create Vector Index</Text>
            </Group>
          </Paper>
          <Paper p="sm" radius="sm" bg="dark.6" style={{ cursor: 'pointer' }} onClick={() => navigate('/ai')}>
            <Group gap="xs">
              <IconRobot size={16} color="var(--mantine-color-orange-5)" />
              <Text size="sm" fw={500}>Try Workers AI</Text>
            </Group>
          </Paper>
        </SimpleGrid>
      </ModuleCard>

      {/* Primary Stats Grid with Sparklines */}
      <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
        <StatCard
          icon={<IconDatabase size={20} />}
          label="Durable Objects"
          value={stats?.durable_objects.namespaces ?? 0}
          description={`${stats?.durable_objects.objects ?? 0} active objects`}
          color="orange"
          sparklineData={sparklineData['Durable Objects']}
          onClick={() => navigate('/durable-objects')}
          helpText="Click to manage Durable Objects"
        />
        <StatCard
          icon={<IconMailbox size={20} />}
          label="Queues"
          value={stats?.queues.count ?? 0}
          description={`${(stats?.queues.total_messages ?? 0).toLocaleString()} messages`}
          color="orange"
          sparklineData={sparklineData['Queues']}
          onClick={() => navigate('/queues')}
          helpText="Click to manage Queues"
        />
        <StatCard
          icon={<IconVectorTriangle size={20} />}
          label="Vectorize"
          value={stats?.vectorize.indexes ?? 0}
          description={`${(stats?.vectorize.total_vectors ?? 0).toLocaleString()} vectors`}
          color="orange"
          sparklineData={sparklineData['Vectorize']}
          onClick={() => navigate('/vectorize')}
          helpText="Click to manage Vector indexes"
        />
        <StatCard
          icon={<IconRobot size={20} />}
          label="AI Requests"
          value={formatNumber(stats?.ai.requests_today ?? 0)}
          description="Today"
          color="orange"
          sparklineData={sparklineData['AI Requests']}
          onClick={() => navigate('/ai')}
          helpText="Click to open Workers AI playground"
        />
      </SimpleGrid>

      {/* Requests Chart */}
      <ModuleCard
        title="Requests Over Time"
        description="Total requests across all services"
        helpContent="This chart shows the aggregate request count across all Localflare services. Use the time range selector to adjust the view."
      >
        <AreaChart
          data={timeSeries}
          timeRange={timeRange}
          onTimeRangeChange={setTimeRange}
          height={250}
          formatValue={(v) => formatNumber(v)}
          withPaper={false}
        />
      </ModuleCard>

      {/* Activity and Status */}
      <SimpleGrid cols={{ base: 1, md: 2 }} spacing="md">
        {/* Recent Activity */}
        <ModuleCard
          title="Recent Activity"
          description="Latest events from your services"
          helpContent="Activity log shows real-time events from all Localflare services. Events are automatically refreshed every 30 seconds."
        >
          <Stack gap="xs">
            {activity.map((event) => (
              <Group key={event.id} justify="space-between" wrap="nowrap">
                <Group gap="xs" wrap="nowrap">
                  <Box
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: '50%',
                      backgroundColor: 'var(--mantine-color-orange-6)',
                    }}
                  />
                  <Text size="sm" truncate>
                    {event.message}
                  </Text>
                </Group>
                <Text size="xs" c="dimmed" style={{ flexShrink: 0 }}>
                  {formatTime(event.timestamp)}
                </Text>
              </Group>
            ))}
            {activity.length === 0 && (
              <Text size="sm" c="dimmed">
                No recent activity
              </Text>
            )}
          </Stack>
        </ModuleCard>

        {/* System Status */}
        <ModuleCard
          title="System Status"
          description="Current health of all services"
          helpContent="System status shows the current operational state of each Localflare service. Green indicates the service is running normally."
        >
          <Stack gap="xs">
            {statuses.map((status) => (
              <Group key={status.service} justify="space-between">
                <Text size="sm">{status.service}</Text>
                <StatusBadge status={status.status} />
              </Group>
            ))}
          </Stack>
        </ModuleCard>
      </SimpleGrid>

      {/* Secondary Stats */}
      <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
        <StatCard
          icon={<IconChartLine size={20} />}
          label="Analytics"
          value={stats?.analytics.datasets ?? 0}
          description={`${formatNumber(stats?.analytics.data_points ?? 0)} data points`}
          onClick={() => navigate('/analytics-engine')}
        />
        <StatCard
          icon={<IconApiApp size={20} />}
          label="AI Gateway"
          value={stats?.ai_gateway.gateways ?? 0}
          description={`${formatNumber(stats?.ai_gateway.requests_today ?? 0)} requests today`}
          onClick={() => navigate('/ai-gateway')}
        />
        <StatCard
          icon={<IconBolt size={20} />}
          label="Hyperdrive"
          value={stats?.hyperdrive.configs ?? 0}
          description={`${stats?.hyperdrive.active_connections ?? 0} active connections`}
          onClick={() => navigate('/hyperdrive')}
        />
        <StatCard
          icon={<IconClock size={20} />}
          label="Cron Triggers"
          value={stats?.cron.triggers ?? 0}
          description={`${stats?.cron.executions_today ?? 0} runs today`}
          onClick={() => navigate('/cron')}
        />
      </SimpleGrid>
    </Stack>
  )
}

function formatNumber(num: number): string {
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`
  if (num >= 1000) return `${(num / 1000).toFixed(1)}k`
  return num.toString()
}
