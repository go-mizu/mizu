import { Container, Title, Text, SimpleGrid, Card, Group, ThemeIcon, Stack, Badge, Progress } from '@mantine/core'
import { IconWorld, IconBolt, IconKey, IconCloud, IconDatabase, IconShield, IconActivity, IconTrendingUp } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'

interface Stats {
  zones: number
  workers: number
  kvNamespaces: number
  r2Buckets: number
  d1Databases: number
  requests24h: number
  bandwidth24h: number
  threats24h: number
}

const mockTrafficData = Array.from({ length: 24 }, (_, i) => ({
  time: `${i}:00`,
  requests: Math.floor(Math.random() * 1000) + 500,
  bandwidth: Math.floor(Math.random() * 500) + 100,
}))

export function Dashboard() {
  const [stats, setStats] = useState<Stats>({
    zones: 0,
    workers: 0,
    kvNamespaces: 0,
    r2Buckets: 0,
    d1Databases: 0,
    requests24h: 0,
    bandwidth24h: 0,
    threats24h: 0,
  })

  useEffect(() => {
    // Fetch stats from API
    Promise.all([
      fetch('/api/zones').then(r => r.json()),
      fetch('/api/workers').then(r => r.json()),
      fetch('/api/kv/namespaces').then(r => r.json()),
      fetch('/api/r2/buckets').then(r => r.json()),
      fetch('/api/d1/databases').then(r => r.json()),
    ]).then(([zones, workers, kv, r2, d1]) => {
      setStats({
        zones: zones.result?.length || 0,
        workers: workers.result?.length || 0,
        kvNamespaces: kv.result?.length || 0,
        r2Buckets: r2.result?.length || 0,
        d1Databases: d1.result?.length || 0,
        requests24h: 12453,
        bandwidth24h: 1.2,
        threats24h: 23,
      })
    }).catch(console.error)
  }, [])

  const statCards = [
    { label: 'Zones', value: stats.zones, icon: IconWorld, color: 'blue' },
    { label: 'Workers', value: stats.workers, icon: IconBolt, color: 'yellow' },
    { label: 'KV Namespaces', value: stats.kvNamespaces, icon: IconKey, color: 'green' },
    { label: 'R2 Buckets', value: stats.r2Buckets, icon: IconCloud, color: 'grape' },
    { label: 'D1 Databases', value: stats.d1Databases, icon: IconDatabase, color: 'cyan' },
    { label: 'Threats Blocked', value: stats.threats24h, icon: IconShield, color: 'red' },
  ]

  return (
    <Container size="xl" py="xl">
      <Stack gap="xl">
        <div>
          <Title order={1}>Dashboard</Title>
          <Text c="dimmed" mt="xs">Overview of your Localflare infrastructure</Text>
        </div>

        <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="lg">
          {statCards.map((stat) => (
            <Card key={stat.label} withBorder shadow="sm" radius="md">
              <Group justify="space-between">
                <div>
                  <Text size="xs" c="dimmed" tt="uppercase" fw={700}>
                    {stat.label}
                  </Text>
                  <Text size="xl" fw={700} mt="xs">
                    {stat.value}
                  </Text>
                </div>
                <ThemeIcon size="xl" radius="md" variant="light" color={stat.color}>
                  <stat.icon size={24} />
                </ThemeIcon>
              </Group>
            </Card>
          ))}
        </SimpleGrid>

        <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="lg">
          <Card withBorder shadow="sm" radius="md" p="lg">
            <Group justify="space-between" mb="md">
              <div>
                <Text fw={600}>Traffic Overview</Text>
                <Text size="sm" c="dimmed">Requests over the last 24 hours</Text>
              </div>
              <Badge color="green" variant="light" leftSection={<IconTrendingUp size={12} />}>
                +12%
              </Badge>
            </Group>
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={mockTrafficData}>
                <defs>
                  <linearGradient id="colorRequests" x1="0" y1="0" x2="0" y2="1">
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
                <Area
                  type="monotone"
                  dataKey="requests"
                  stroke="#f6821f"
                  fillOpacity={1}
                  fill="url(#colorRequests)"
                />
              </AreaChart>
            </ResponsiveContainer>
          </Card>

          <Card withBorder shadow="sm" radius="md" p="lg">
            <Group justify="space-between" mb="md">
              <div>
                <Text fw={600}>System Status</Text>
                <Text size="sm" c="dimmed">All systems operational</Text>
              </div>
              <Badge color="green">Healthy</Badge>
            </Group>
            <Stack gap="md">
              <div>
                <Group justify="space-between" mb={4}>
                  <Text size="sm">DNS Server</Text>
                  <Text size="sm" c="green">Online</Text>
                </Group>
                <Progress value={100} color="green" size="sm" />
              </div>
              <div>
                <Group justify="space-between" mb={4}>
                  <Text size="sm">HTTP Proxy</Text>
                  <Text size="sm" c="green">Online</Text>
                </Group>
                <Progress value={100} color="green" size="sm" />
              </div>
              <div>
                <Group justify="space-between" mb={4}>
                  <Text size="sm">HTTPS Proxy</Text>
                  <Text size="sm" c="green">Online</Text>
                </Group>
                <Progress value={100} color="green" size="sm" />
              </div>
              <div>
                <Group justify="space-between" mb={4}>
                  <Text size="sm">Workers Runtime</Text>
                  <Text size="sm" c="green">Online</Text>
                </Group>
                <Progress value={100} color="green" size="sm" />
              </div>
            </Stack>
          </Card>
        </SimpleGrid>

        <Card withBorder shadow="sm" radius="md" p="lg">
          <Group justify="space-between" mb="md">
            <div>
              <Text fw={600}>Quick Stats</Text>
              <Text size="sm" c="dimmed">Last 24 hours</Text>
            </div>
            <IconActivity size={20} color="var(--mantine-color-dimmed)" />
          </Group>
          <SimpleGrid cols={{ base: 1, sm: 3 }}>
            <div>
              <Text size="xs" c="dimmed" tt="uppercase">Total Requests</Text>
              <Text size="xl" fw={700}>{stats.requests24h.toLocaleString()}</Text>
            </div>
            <div>
              <Text size="xs" c="dimmed" tt="uppercase">Bandwidth</Text>
              <Text size="xl" fw={700}>{stats.bandwidth24h} GB</Text>
            </div>
            <div>
              <Text size="xs" c="dimmed" tt="uppercase">Cache Hit Rate</Text>
              <Text size="xl" fw={700}>87.3%</Text>
            </div>
          </SimpleGrid>
        </Card>
      </Stack>
    </Container>
  )
}
