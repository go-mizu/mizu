import { Container, Title, Text, Card, SimpleGrid, Group, Stack, SegmentedControl } from '@mantine/core'
import { IconChartBar, IconShield, IconWorld, IconTrendingUp } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar, PieChart, Pie, Cell } from 'recharts'

interface AnalyticsData {
  requests: { time: string; value: number }[]
  bandwidth: { time: string; value: number }[]
  threats: { time: string; value: number }[]
  countries: { country: string; requests: number }[]
  status_codes: { code: string; count: number }[]
  content_types: { type: string; count: number }[]
  totals: {
    requests: number
    bandwidth: number
    threats: number
    cache_hits: number
    unique_visitors: number
  }
}

const COLORS = ['#f6821f', '#2196F3', '#4CAF50', '#FF9800', '#9C27B0', '#00BCD4']

export function Analytics() {
  const { id: zoneId } = useParams<{ id: string }>()
  const [data, setData] = useState<AnalyticsData | null>(null)
  const [timeRange, setTimeRange] = useState('24h')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchAnalytics()
  }, [zoneId, timeRange])

  const fetchAnalytics = async () => {
    setLoading(true)
    try {
      const url = zoneId
        ? `/api/zones/${zoneId}/analytics?range=${timeRange}`
        : `/api/analytics?range=${timeRange}`
      const res = await fetch(url)
      const result = await res.json()
      setData(result.result)
    } catch (error) {
      console.error('Failed to fetch analytics:', error)
    } finally {
      setLoading(false)
    }
  }

  const formatNumber = (num: number) => {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
    return num.toString()
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  if (loading || !data) {
    return <Container size="xl" py="xl"><Text>Loading analytics...</Text></Container>
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="xl">
        <Group justify="space-between">
          <div>
            <Title order={1}>Analytics</Title>
            <Text c="dimmed" mt="xs">Traffic and security insights</Text>
          </div>
          <SegmentedControl
            value={timeRange}
            onChange={setTimeRange}
            data={[
              { label: '24h', value: '24h' },
              { label: '7d', value: '7d' },
              { label: '30d', value: '30d' },
            ]}
          />
        </Group>

        <SimpleGrid cols={{ base: 1, sm: 2, lg: 5 }} spacing="lg">
          <Card withBorder shadow="sm" radius="md" p="lg">
            <Group justify="space-between">
              <div>
                <Text size="xs" c="dimmed" tt="uppercase">Requests</Text>
                <Text size="xl" fw={700}>{formatNumber(data.totals.requests)}</Text>
              </div>
              <IconWorld size={24} color="var(--mantine-color-blue-6)" />
            </Group>
          </Card>
          <Card withBorder shadow="sm" radius="md" p="lg">
            <Group justify="space-between">
              <div>
                <Text size="xs" c="dimmed" tt="uppercase">Bandwidth</Text>
                <Text size="xl" fw={700}>{formatBytes(data.totals.bandwidth)}</Text>
              </div>
              <IconTrendingUp size={24} color="var(--mantine-color-green-6)" />
            </Group>
          </Card>
          <Card withBorder shadow="sm" radius="md" p="lg">
            <Group justify="space-between">
              <div>
                <Text size="xs" c="dimmed" tt="uppercase">Threats Blocked</Text>
                <Text size="xl" fw={700}>{formatNumber(data.totals.threats)}</Text>
              </div>
              <IconShield size={24} color="var(--mantine-color-red-6)" />
            </Group>
          </Card>
          <Card withBorder shadow="sm" radius="md" p="lg">
            <Group justify="space-between">
              <div>
                <Text size="xs" c="dimmed" tt="uppercase">Cache Hit Rate</Text>
                <Text size="xl" fw={700}>
                  {data.totals.requests > 0
                    ? Math.round((data.totals.cache_hits / data.totals.requests) * 100)
                    : 0}%
                </Text>
              </div>
              <IconChartBar size={24} color="var(--mantine-color-grape-6)" />
            </Group>
          </Card>
          <Card withBorder shadow="sm" radius="md" p="lg">
            <Group justify="space-between">
              <div>
                <Text size="xs" c="dimmed" tt="uppercase">Unique Visitors</Text>
                <Text size="xl" fw={700}>{formatNumber(data.totals.unique_visitors)}</Text>
              </div>
              <IconWorld size={24} color="var(--mantine-color-cyan-6)" />
            </Group>
          </Card>
        </SimpleGrid>

        <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="lg">
          <Card withBorder shadow="sm" radius="md" p="lg">
            <Text fw={600} mb="md">Requests Over Time</Text>
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={data.requests}>
                <defs>
                  <linearGradient id="colorReq" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#f6821f" stopOpacity={0.8}/>
                    <stop offset="95%" stopColor="#f6821f" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                <XAxis dataKey="time" stroke="#666" fontSize={12} />
                <YAxis stroke="#666" fontSize={12} />
                <Tooltip
                  contentStyle={{ background: '#1a1a2e', border: '1px solid #333' }}
                  labelStyle={{ color: '#fff' }}
                />
                <Area type="monotone" dataKey="value" stroke="#f6821f" fillOpacity={1} fill="url(#colorReq)" />
              </AreaChart>
            </ResponsiveContainer>
          </Card>

          <Card withBorder shadow="sm" radius="md" p="lg">
            <Text fw={600} mb="md">Bandwidth Over Time</Text>
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={data.bandwidth}>
                <defs>
                  <linearGradient id="colorBw" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#2196F3" stopOpacity={0.8}/>
                    <stop offset="95%" stopColor="#2196F3" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                <XAxis dataKey="time" stroke="#666" fontSize={12} />
                <YAxis stroke="#666" fontSize={12} tickFormatter={(v) => formatBytes(v)} />
                <Tooltip
                  contentStyle={{ background: '#1a1a2e', border: '1px solid #333' }}
                  labelStyle={{ color: '#fff' }}
                  formatter={(value) => formatBytes(value as number)}
                />
                <Area type="monotone" dataKey="value" stroke="#2196F3" fillOpacity={1} fill="url(#colorBw)" />
              </AreaChart>
            </ResponsiveContainer>
          </Card>
        </SimpleGrid>

        <SimpleGrid cols={{ base: 1, lg: 3 }} spacing="lg">
          <Card withBorder shadow="sm" radius="md" p="lg">
            <Text fw={600} mb="md">Top Countries</Text>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={data.countries.slice(0, 5)} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                <XAxis type="number" stroke="#666" fontSize={12} />
                <YAxis dataKey="country" type="category" stroke="#666" fontSize={12} width={60} />
                <Tooltip
                  contentStyle={{ background: '#1a1a2e', border: '1px solid #333' }}
                />
                <Bar dataKey="requests" fill="#f6821f" />
              </BarChart>
            </ResponsiveContainer>
          </Card>

          <Card withBorder shadow="sm" radius="md" p="lg">
            <Text fw={600} mb="md">Status Codes</Text>
            <ResponsiveContainer width="100%" height={200}>
              <PieChart>
                <Pie
                  data={data.status_codes}
                  dataKey="count"
                  nameKey="code"
                  cx="50%"
                  cy="50%"
                  outerRadius={70}
                  label={({ name }) => name}
                >
                  {data.status_codes.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{ background: '#1a1a2e', border: '1px solid #333' }}
                />
              </PieChart>
            </ResponsiveContainer>
          </Card>

          <Card withBorder shadow="sm" radius="md" p="lg">
            <Text fw={600} mb="md">Content Types</Text>
            <ResponsiveContainer width="100%" height={200}>
              <PieChart>
                <Pie
                  data={data.content_types}
                  dataKey="count"
                  nameKey="type"
                  cx="50%"
                  cy="50%"
                  outerRadius={70}
                  label={({ name }) => name}
                >
                  {data.content_types.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{ background: '#1a1a2e', border: '1px solid #333' }}
                />
              </PieChart>
            </ResponsiveContainer>
          </Card>
        </SimpleGrid>

        <Card withBorder shadow="sm" radius="md" p="lg">
          <Text fw={600} mb="md">Threats Blocked Over Time</Text>
          <ResponsiveContainer width="100%" height={200}>
            <AreaChart data={data.threats}>
              <defs>
                <linearGradient id="colorThreats" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#f44336" stopOpacity={0.8}/>
                  <stop offset="95%" stopColor="#f44336" stopOpacity={0}/>
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#333" />
              <XAxis dataKey="time" stroke="#666" fontSize={12} />
              <YAxis stroke="#666" fontSize={12} />
              <Tooltip
                contentStyle={{ background: '#1a1a2e', border: '1px solid #333' }}
                labelStyle={{ color: '#fff' }}
              />
              <Area type="monotone" dataKey="value" stroke="#f44336" fillOpacity={1} fill="url(#colorThreats)" />
            </AreaChart>
          </ResponsiveContainer>
        </Card>
      </Stack>
    </Container>
  )
}
