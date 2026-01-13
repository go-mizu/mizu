import { Container, Title, Text, Card, SimpleGrid, Group, ThemeIcon, Stack, Badge } from '@mantine/core'
import { IconWorld, IconLock, IconShield, IconBolt, IconDatabase, IconChartBar, IconArrowRight } from '@tabler/icons-react'
import { useParams, useNavigate } from 'react-router-dom'
import { useEffect, useState } from 'react'

interface Zone {
  id: string
  name: string
  status: string
  plan: string
  name_servers: string[]
  created_at: string
}

const features = [
  { label: 'DNS', description: 'Manage DNS records', icon: IconWorld, path: 'dns', color: 'blue' },
  { label: 'SSL/TLS', description: 'Certificates and encryption', icon: IconLock, path: 'ssl', color: 'green' },
  { label: 'Firewall', description: 'WAF and security rules', icon: IconShield, path: 'firewall', color: 'red' },
  { label: 'Speed', description: 'Performance optimization', icon: IconBolt, path: 'speed', color: 'yellow' },
  { label: 'Caching', description: 'Cache configuration', icon: IconDatabase, path: 'caching', color: 'grape' },
  { label: 'Analytics', description: 'Traffic and security stats', icon: IconChartBar, path: 'analytics', color: 'cyan' },
]

export function ZoneDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [zone, setZone] = useState<Zone | null>(null)

  useEffect(() => {
    fetch(`/api/zones/${id}`)
      .then(r => r.json())
      .then(data => setZone(data.result))
      .catch(console.error)
  }, [id])

  if (!zone) {
    return (
      <Container size="xl" py="xl">
        <Text>Loading...</Text>
      </Container>
    )
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="xl">
        <Group justify="space-between">
          <div>
            <Group gap="md">
              <ThemeIcon size="xl" radius="md" color="orange">
                <IconWorld size={24} />
              </ThemeIcon>
              <div>
                <Title order={1}>{zone.name}</Title>
                <Text c="dimmed">Zone Overview</Text>
              </div>
            </Group>
          </div>
          <Group>
            <Badge size="lg" color={zone.status === 'active' ? 'green' : 'yellow'}>
              {zone.status}
            </Badge>
            <Badge size="lg" variant="outline">{zone.plan}</Badge>
          </Group>
        </Group>

        <Card withBorder shadow="sm" radius="md" p="lg">
          <Text fw={600} mb="md">Name Servers</Text>
          <Text size="sm" c="dimmed" mb="sm">
            Update your domain's nameservers to point to Localflare
          </Text>
          <Stack gap="xs">
            {zone.name_servers.map((ns, i) => (
              <Text key={i} ff="monospace" size="sm">{ns}</Text>
            ))}
          </Stack>
        </Card>

        <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="lg">
          {features.map((feature) => (
            <Card
              key={feature.path}
              withBorder
              shadow="sm"
              radius="md"
              p="lg"
              style={{ cursor: 'pointer' }}
              onClick={() => navigate(`/zones/${id}/${feature.path}`)}
            >
              <Group justify="space-between">
                <Group>
                  <ThemeIcon size="lg" radius="md" variant="light" color={feature.color}>
                    <feature.icon size={20} />
                  </ThemeIcon>
                  <div>
                    <Text fw={600}>{feature.label}</Text>
                    <Text size="sm" c="dimmed">{feature.description}</Text>
                  </div>
                </Group>
                <IconArrowRight size={18} color="var(--mantine-color-dimmed)" />
              </Group>
            </Card>
          ))}
        </SimpleGrid>

        <Card withBorder shadow="sm" radius="md" p="lg">
          <Text fw={600} mb="md">Quick Stats (24h)</Text>
          <SimpleGrid cols={{ base: 2, md: 4 }}>
            <div>
              <Text size="xs" c="dimmed" tt="uppercase">Requests</Text>
              <Text size="xl" fw={700}>12.4K</Text>
            </div>
            <div>
              <Text size="xs" c="dimmed" tt="uppercase">Bandwidth</Text>
              <Text size="xl" fw={700}>1.2 GB</Text>
            </div>
            <div>
              <Text size="xs" c="dimmed" tt="uppercase">Threats Blocked</Text>
              <Text size="xl" fw={700}>23</Text>
            </div>
            <div>
              <Text size="xs" c="dimmed" tt="uppercase">Cache Hit Rate</Text>
              <Text size="xl" fw={700}>87%</Text>
            </div>
          </SimpleGrid>
        </Card>
      </Stack>
    </Container>
  )
}
