import { Container, Title, Text, Card, SimpleGrid, Group, Stack, Badge, Select, Switch, Button, Table, Modal, TextInput } from '@mantine/core'
import { IconLock, IconPlus } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface SSLSettings {
  mode: string
  always_https: boolean
  min_tls_version: string
  tls_1_3: boolean
  automatic_https_rewrites: boolean
}

interface Certificate {
  id: string
  type: string
  hosts: string[]
  status: string
  expires_at: string
}

export function SSLSettings() {
  const { id: zoneId } = useParams<{ id: string }>()
  const [settings, setSettings] = useState<SSLSettings | null>(null)
  const [certificates, setCertificates] = useState<Certificate[]>([])
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [newCertHosts, setNewCertHosts] = useState('')

  useEffect(() => {
    fetchSettings()
    fetchCertificates()
  }, [zoneId])

  const fetchSettings = async () => {
    try {
      const res = await fetch(`/api/zones/${zoneId}/ssl/settings`)
      const data = await res.json()
      setSettings(data.result)
    } catch (error) {
      console.error('Failed to fetch SSL settings:', error)
    }
  }

  const fetchCertificates = async () => {
    try {
      const res = await fetch(`/api/zones/${zoneId}/ssl/certificates`)
      const data = await res.json()
      setCertificates(data.result || [])
    } catch (error) {
      console.error('Failed to fetch certificates:', error)
    }
  }

  const updateSettings = async (updates: Partial<SSLSettings>) => {
    try {
      const res = await fetch(`/api/zones/${zoneId}/ssl/settings`, {
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

  const createCertificate = async () => {
    try {
      const hosts = newCertHosts.split(',').map(h => h.trim()).filter(Boolean)
      const res = await fetch(`/api/zones/${zoneId}/ssl/certificates`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: 'edge', hosts }),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Certificate created', color: 'green' })
        setCreateModalOpen(false)
        setNewCertHosts('')
        fetchCertificates()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create certificate', color: 'red' })
    }
  }

  if (!settings) return <Container py="xl"><Text>Loading...</Text></Container>

  return (
    <Container size="xl" py="xl">
      <Stack gap="xl">
        <div>
          <Title order={1}>SSL/TLS</Title>
          <Text c="dimmed" mt="xs">Manage encryption settings for your domain</Text>
        </div>

        <Card withBorder shadow="sm" radius="md" p="lg">
          <Group mb="lg">
            <IconLock size={24} />
            <div>
              <Text fw={600}>Encryption Mode</Text>
              <Text size="sm" c="dimmed">Configure how traffic is encrypted</Text>
            </div>
          </Group>

          <SimpleGrid cols={{ base: 2, md: 4 }} spacing="md">
            {['off', 'flexible', 'full', 'strict'].map((mode) => (
              <Card
                key={mode}
                withBorder
                p="md"
                radius="md"
                style={{
                  cursor: 'pointer',
                  borderColor: settings.mode === mode ? 'var(--mantine-color-orange-6)' : undefined,
                }}
                onClick={() => updateSettings({ mode })}
              >
                <Text fw={600} tt="capitalize">{mode}</Text>
                <Text size="xs" c="dimmed">
                  {mode === 'off' && 'No encryption'}
                  {mode === 'flexible' && 'Encrypt visitor traffic'}
                  {mode === 'full' && 'End-to-end (self-signed OK)'}
                  {mode === 'strict' && 'End-to-end (valid cert)'}
                </Text>
              </Card>
            ))}
          </SimpleGrid>
        </Card>

        <Card withBorder shadow="sm" radius="md" p="lg">
          <Text fw={600} mb="lg">SSL Settings</Text>
          <Stack gap="md">
            <Group justify="space-between">
              <div>
                <Text fw={500}>Always Use HTTPS</Text>
                <Text size="sm" c="dimmed">Redirect all HTTP requests to HTTPS</Text>
              </div>
              <Switch
                checked={settings.always_https}
                onChange={(e) => updateSettings({ always_https: e.target.checked })}
              />
            </Group>
            <Group justify="space-between">
              <div>
                <Text fw={500}>Minimum TLS Version</Text>
                <Text size="sm" c="dimmed">Minimum version of TLS to accept</Text>
              </div>
              <Select
                w={150}
                value={settings.min_tls_version}
                onChange={(v) => updateSettings({ min_tls_version: v || '1.2' })}
                data={['1.0', '1.1', '1.2', '1.3']}
              />
            </Group>
            <Group justify="space-between">
              <div>
                <Text fw={500}>TLS 1.3</Text>
                <Text size="sm" c="dimmed">Enable TLS 1.3 for improved security</Text>
              </div>
              <Switch
                checked={settings.tls_1_3}
                onChange={(e) => updateSettings({ tls_1_3: e.target.checked })}
              />
            </Group>
            <Group justify="space-between">
              <div>
                <Text fw={500}>Automatic HTTPS Rewrites</Text>
                <Text size="sm" c="dimmed">Rewrite HTTP links to HTTPS</Text>
              </div>
              <Switch
                checked={settings.automatic_https_rewrites}
                onChange={(e) => updateSettings({ automatic_https_rewrites: e.target.checked })}
              />
            </Group>
          </Stack>
        </Card>

        <Card withBorder shadow="sm" radius="md" p="lg">
          <Group justify="space-between" mb="lg">
            <div>
              <Text fw={600}>Edge Certificates</Text>
              <Text size="sm" c="dimmed">Certificates used to encrypt traffic</Text>
            </div>
            <Button leftSection={<IconPlus size={16} />} variant="light" onClick={() => setCreateModalOpen(true)}>
              Create Certificate
            </Button>
          </Group>

          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Hosts</Table.Th>
                <Table.Th>Type</Table.Th>
                <Table.Th>Status</Table.Th>
                <Table.Th>Expires</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {certificates.map((cert) => (
                <Table.Tr key={cert.id}>
                  <Table.Td>
                    <Text size="sm">{cert.hosts.join(', ')}</Text>
                  </Table.Td>
                  <Table.Td>
                    <Badge variant="outline">{cert.type}</Badge>
                  </Table.Td>
                  <Table.Td>
                    <Badge color={cert.status === 'active' ? 'green' : 'yellow'}>
                      {cert.status}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm">{new Date(cert.expires_at).toLocaleDateString()}</Text>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </Card>
      </Stack>

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Create Certificate">
        <Stack>
          <TextInput
            label="Hostnames"
            placeholder="example.com, *.example.com"
            description="Comma-separated list of hostnames"
            value={newCertHosts}
            onChange={(e) => setNewCertHosts(e.target.value)}
          />
          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => setCreateModalOpen(false)}>Cancel</Button>
            <Button onClick={createCertificate}>Create</Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
