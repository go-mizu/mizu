import { Container, Title, Text, Card, Group, Stack, Select, Switch, Button, SimpleGrid, Progress } from '@mantine/core'
import { IconDatabase, IconRefresh, IconTrash } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface CacheSettings {
  level: string
  browser_ttl: number
  always_online: boolean
  development_mode: boolean
  cache_deception_armor: boolean
}

interface CacheStats {
  hits: number
  misses: number
  bandwidth_saved: number
  size_bytes: number
}

export function CacheSettings() {
  const { id: zoneId } = useParams<{ id: string }>()
  const [settings, setSettings] = useState<CacheSettings | null>(null)
  const [stats, setStats] = useState<CacheStats>({ hits: 0, misses: 0, bandwidth_saved: 0, size_bytes: 0 })

  useEffect(() => {
    fetchSettings()
    fetchStats()
  }, [zoneId])

  const fetchSettings = async () => {
    try {
      const res = await fetch(`/api/zones/${zoneId}/cache/settings`)
      const data = await res.json()
      setSettings(data.result)
    } catch (error) {
      console.error('Failed to fetch cache settings:', error)
    }
  }

  const fetchStats = async () => {
    try {
      const res = await fetch(`/api/zones/${zoneId}/cache/stats`)
      const data = await res.json()
      setStats(data.result || { hits: 0, misses: 0, bandwidth_saved: 0, size_bytes: 0 })
    } catch (error) {
      console.error('Failed to fetch cache stats:', error)
    }
  }

  const updateSettings = async (updates: Partial<CacheSettings>) => {
    try {
      const res = await fetch(`/api/zones/${zoneId}/cache/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...settings, ...updates }),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Settings updated', color: 'green' })
        fetchSettings()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to update settings', color: 'red' })
    }
  }

  const purgeCache = async (purgeAll: boolean) => {
    try {
      const res = await fetch(`/api/zones/${zoneId}/cache/purge`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ purge_everything: purgeAll }),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Cache purged', color: 'green' })
        fetchStats()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to purge cache', color: 'red' })
    }
  }

  if (!settings) return <Container py="xl"><Text>Loading...</Text></Container>

  const hitRate = stats.hits + stats.misses > 0
    ? Math.round((stats.hits / (stats.hits + stats.misses)) * 100)
    : 0

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="xl">
        <div>
          <Title order={1}>Caching</Title>
          <Text c="dimmed" mt="xs">Configure caching behavior for your domain</Text>
        </div>

        <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="lg">
          <Card withBorder shadow="sm" radius="md" p="lg">
            <Group justify="space-between">
              <div>
                <Text size="xs" c="dimmed" tt="uppercase">Cache Hits</Text>
                <Text size="xl" fw={700}>{stats.hits.toLocaleString()}</Text>
              </div>
              <IconDatabase size={24} color="var(--mantine-color-green-6)" />
            </Group>
          </Card>
          <Card withBorder shadow="sm" radius="md" p="lg">
            <Group justify="space-between">
              <div>
                <Text size="xs" c="dimmed" tt="uppercase">Cache Misses</Text>
                <Text size="xl" fw={700}>{stats.misses.toLocaleString()}</Text>
              </div>
              <IconDatabase size={24} color="var(--mantine-color-red-6)" />
            </Group>
          </Card>
          <Card withBorder shadow="sm" radius="md" p="lg">
            <Group justify="space-between">
              <div>
                <Text size="xs" c="dimmed" tt="uppercase">Hit Rate</Text>
                <Text size="xl" fw={700}>{hitRate}%</Text>
              </div>
            </Group>
            <Progress value={hitRate} color="green" size="sm" mt="md" />
          </Card>
          <Card withBorder shadow="sm" radius="md" p="lg">
            <Group justify="space-between">
              <div>
                <Text size="xs" c="dimmed" tt="uppercase">Bandwidth Saved</Text>
                <Text size="xl" fw={700}>{formatBytes(stats.bandwidth_saved)}</Text>
              </div>
            </Group>
          </Card>
        </SimpleGrid>

        <Card withBorder shadow="sm" radius="md" p="lg">
          <Group justify="space-between" mb="lg">
            <div>
              <Text fw={600}>Purge Cache</Text>
              <Text size="sm" c="dimmed">Remove cached content</Text>
            </div>
            <Group>
              <Button variant="light" leftSection={<IconRefresh size={16} />} onClick={() => purgeCache(false)}>
                Purge by URL
              </Button>
              <Button color="red" leftSection={<IconTrash size={16} />} onClick={() => purgeCache(true)}>
                Purge Everything
              </Button>
            </Group>
          </Group>
        </Card>

        <Card withBorder shadow="sm" radius="md" p="lg">
          <Text fw={600} mb="lg">Cache Settings</Text>
          <Stack gap="md">
            <Group justify="space-between">
              <div>
                <Text fw={500}>Caching Level</Text>
                <Text size="sm" c="dimmed">How aggressively to cache content</Text>
              </div>
              <Select
                w={200}
                value={settings.level}
                onChange={(v) => updateSettings({ level: v || 'standard' })}
                data={[
                  { value: 'bypass', label: 'No Query String' },
                  { value: 'basic', label: 'Basic' },
                  { value: 'simplified', label: 'Simplified' },
                  { value: 'aggressive', label: 'Aggressive' },
                  { value: 'standard', label: 'Standard' },
                ]}
              />
            </Group>
            <Group justify="space-between">
              <div>
                <Text fw={500}>Browser Cache TTL</Text>
                <Text size="sm" c="dimmed">How long browsers should cache files</Text>
              </div>
              <Select
                w={200}
                value={String(settings.browser_ttl)}
                onChange={(v) => updateSettings({ browser_ttl: parseInt(v || '14400') })}
                data={[
                  { value: '0', label: 'Respect Existing Headers' },
                  { value: '1800', label: '30 minutes' },
                  { value: '3600', label: '1 hour' },
                  { value: '14400', label: '4 hours' },
                  { value: '86400', label: '1 day' },
                  { value: '604800', label: '1 week' },
                  { value: '2592000', label: '1 month' },
                  { value: '31536000', label: '1 year' },
                ]}
              />
            </Group>
            <Group justify="space-between">
              <div>
                <Text fw={500}>Always Online</Text>
                <Text size="sm" c="dimmed">Serve cached content when origin is down</Text>
              </div>
              <Switch
                checked={settings.always_online}
                onChange={(e) => updateSettings({ always_online: e.target.checked })}
              />
            </Group>
            <Group justify="space-between">
              <div>
                <Text fw={500}>Development Mode</Text>
                <Text size="sm" c="dimmed">Temporarily bypass cache for development</Text>
              </div>
              <Switch
                checked={settings.development_mode}
                onChange={(e) => updateSettings({ development_mode: e.target.checked })}
              />
            </Group>
            <Group justify="space-between">
              <div>
                <Text fw={500}>Cache Deception Armor</Text>
                <Text size="sm" c="dimmed">Protect against web cache deception attacks</Text>
              </div>
              <Switch
                checked={settings.cache_deception_armor}
                onChange={(e) => updateSettings({ cache_deception_armor: e.target.checked })}
              />
            </Group>
          </Stack>
        </Card>
      </Stack>
    </Container>
  )
}
