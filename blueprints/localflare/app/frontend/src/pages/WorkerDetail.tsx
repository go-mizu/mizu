import { Container, Title, Text, Card, Button, Group, Stack, Textarea, TextInput, Badge, Tabs, Code, Table, ActionIcon, Modal } from '@mantine/core'
import { IconBolt, IconPlayerPlay, IconDeviceFloppy, IconPlus, IconTrash, IconTerminal } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface Worker {
  id: string
  name: string
  script: string
  routes: string[]
  env_vars: Record<string, string>
  kv_bindings: { name: string; namespace_id: string }[]
  created_at: string
  updated_at: string
}

interface TestResult {
  status: number
  headers: Record<string, string>
  body: string
  duration_ms: number
}

export function WorkerDetail() {
  const { id } = useParams<{ id: string }>()
  const [worker, setWorker] = useState<Worker | null>(null)
  const [script, setScript] = useState('')
  const [testUrl, setTestUrl] = useState('https://example.com/')
  const [testResult, setTestResult] = useState<TestResult | null>(null)
  const [routeModalOpen, setRouteModalOpen] = useState(false)
  const [newRoute, setNewRoute] = useState('')
  const [testing, setTesting] = useState(false)

  useEffect(() => {
    fetchWorker()
  }, [id])

  const fetchWorker = async () => {
    try {
      const res = await fetch(`/api/workers/${id}`)
      const data = await res.json()
      setWorker(data.result)
      setScript(data.result?.script || '')
    } catch (error) {
      console.error('Failed to fetch worker:', error)
    }
  }

  const saveWorker = async () => {
    try {
      const res = await fetch(`/api/workers/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...worker, script }),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Worker saved', color: 'green' })
        fetchWorker()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to save worker', color: 'red' })
    }
  }

  const testWorker = async () => {
    setTesting(true)
    try {
      const res = await fetch(`/api/workers/${id}/test`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url: testUrl, method: 'GET' }),
      })
      const data = await res.json()
      setTestResult(data.result)
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to test worker', color: 'red' })
    } finally {
      setTesting(false)
    }
  }

  const addRoute = async () => {
    if (!newRoute || !worker) return
    try {
      const routes = [...(worker.routes || []), newRoute]
      const res = await fetch(`/api/workers/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...worker, routes }),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Route added', color: 'green' })
        setRouteModalOpen(false)
        setNewRoute('')
        fetchWorker()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to add route', color: 'red' })
    }
  }

  const removeRoute = async (route: string) => {
    if (!worker) return
    try {
      const routes = worker.routes.filter(r => r !== route)
      const res = await fetch(`/api/workers/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...worker, routes }),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Route removed', color: 'green' })
        fetchWorker()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to remove route', color: 'red' })
    }
  }

  if (!worker) return <Container py="xl"><Text>Loading...</Text></Container>

  return (
    <Container size="xl" py="xl">
      <Stack gap="xl">
        <Group justify="space-between">
          <Group>
            <IconBolt size={32} color="var(--mantine-color-yellow-6)" />
            <div>
              <Title order={1}>{worker.name}</Title>
              <Text c="dimmed">Worker Script</Text>
            </div>
          </Group>
          <Group>
            <Button variant="light" leftSection={<IconPlayerPlay size={16} />} onClick={testWorker} loading={testing}>
              Quick Test
            </Button>
            <Button leftSection={<IconDeviceFloppy size={16} />} onClick={saveWorker}>
              Save
            </Button>
          </Group>
        </Group>

        <Tabs defaultValue="code">
          <Tabs.List>
            <Tabs.Tab value="code">Code</Tabs.Tab>
            <Tabs.Tab value="routes">Routes</Tabs.Tab>
            <Tabs.Tab value="settings">Settings</Tabs.Tab>
            <Tabs.Tab value="test">Test</Tabs.Tab>
          </Tabs.List>

          <Tabs.Panel value="code" pt="md">
            <Card withBorder shadow="sm" radius="md" p={0}>
              <Textarea
                value={script}
                onChange={(e) => setScript(e.target.value)}
                minRows={20}
                styles={{
                  input: {
                    fontFamily: 'monospace',
                    fontSize: '14px',
                    border: 'none',
                    borderRadius: 'var(--mantine-radius-md)',
                  },
                }}
              />
            </Card>
          </Tabs.Panel>

          <Tabs.Panel value="routes" pt="md">
            <Card withBorder shadow="sm" radius="md" p="lg">
              <Group justify="space-between" mb="lg">
                <div>
                  <Text fw={600}>Routes</Text>
                  <Text size="sm" c="dimmed">URL patterns that trigger this worker</Text>
                </div>
                <Button size="sm" leftSection={<IconPlus size={14} />} onClick={() => setRouteModalOpen(true)}>
                  Add Route
                </Button>
              </Group>
              <Table>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>Pattern</Table.Th>
                    <Table.Th></Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {worker.routes?.map((route) => (
                    <Table.Tr key={route}>
                      <Table.Td>
                        <Code>{route}</Code>
                      </Table.Td>
                      <Table.Td>
                        <ActionIcon variant="subtle" color="red" onClick={() => removeRoute(route)}>
                          <IconTrash size={14} />
                        </ActionIcon>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
              {(!worker.routes || worker.routes.length === 0) && (
                <Text c="dimmed" ta="center" py="md">No routes configured</Text>
              )}
            </Card>
          </Tabs.Panel>

          <Tabs.Panel value="settings" pt="md">
            <Stack gap="md">
              <Card withBorder shadow="sm" radius="md" p="lg">
                <Text fw={600} mb="md">Environment Variables</Text>
                <Text size="sm" c="dimmed">Configure environment variables for your worker</Text>
                {/* Environment variables would go here */}
              </Card>
              <Card withBorder shadow="sm" radius="md" p="lg">
                <Text fw={600} mb="md">KV Namespace Bindings</Text>
                <Text size="sm" c="dimmed">Bind KV namespaces to your worker</Text>
                {/* KV bindings would go here */}
              </Card>
            </Stack>
          </Tabs.Panel>

          <Tabs.Panel value="test" pt="md">
            <Card withBorder shadow="sm" radius="md" p="lg">
              <Stack gap="md">
                <Group>
                  <TextInput
                    placeholder="https://example.com/path"
                    value={testUrl}
                    onChange={(e) => setTestUrl(e.target.value)}
                    style={{ flex: 1 }}
                  />
                  <Button leftSection={<IconTerminal size={16} />} onClick={testWorker} loading={testing}>
                    Send Request
                  </Button>
                </Group>

                {testResult && (
                  <Card withBorder p="md" bg="dark.8">
                    <Group mb="md">
                      <Badge color={testResult.status < 400 ? 'green' : 'red'}>
                        {testResult.status}
                      </Badge>
                      <Text size="sm" c="dimmed">{testResult.duration_ms}ms</Text>
                    </Group>
                    <Text size="sm" fw={500} mb="xs">Response Headers</Text>
                    <Code block mb="md">
                      {JSON.stringify(testResult.headers, null, 2)}
                    </Code>
                    <Text size="sm" fw={500} mb="xs">Response Body</Text>
                    <Code block>
                      {testResult.body}
                    </Code>
                  </Card>
                )}
              </Stack>
            </Card>
          </Tabs.Panel>
        </Tabs>
      </Stack>

      <Modal opened={routeModalOpen} onClose={() => setRouteModalOpen(false)} title="Add Route">
        <Stack>
          <TextInput
            label="Route Pattern"
            placeholder="*example.com/*"
            value={newRoute}
            onChange={(e) => setNewRoute(e.target.value)}
          />
          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => setRouteModalOpen(false)}>Cancel</Button>
            <Button onClick={addRoute}>Add</Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
