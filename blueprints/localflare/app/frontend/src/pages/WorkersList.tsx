import { Container, Title, Text, Card, Table, Button, Group, Badge, TextInput, Modal, Stack, ActionIcon, Menu, Textarea } from '@mantine/core'
import { IconPlus, IconSearch, IconDotsVertical, IconTrash, IconEdit, IconBolt, IconPlayerPlay } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface Worker {
  id: string
  name: string
  routes: string[]
  created_at: string
  updated_at: string
}

export function WorkersList() {
  const navigate = useNavigate()
  const [workers, setWorkers] = useState<Worker[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [form, setForm] = useState({
    name: '',
    script: `export default {
  async fetch(request, env, ctx) {
    return new Response('Hello World!');
  },
};`,
  })

  useEffect(() => {
    fetchWorkers()
  }, [])

  const fetchWorkers = async () => {
    try {
      const res = await fetch('/api/workers')
      const data = await res.json()
      setWorkers(data.result || [])
    } catch (error) {
      console.error('Failed to fetch workers:', error)
    } finally {
      setLoading(false)
    }
  }

  const createWorker = async () => {
    try {
      const res = await fetch('/api/workers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Worker created', color: 'green' })
        setModalOpen(false)
        setForm({ name: '', script: form.script })
        fetchWorkers()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create worker', color: 'red' })
    }
  }

  const deleteWorker = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      const res = await fetch(`/api/workers/${id}`, { method: 'DELETE' })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Worker deleted', color: 'green' })
        fetchWorkers()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete worker', color: 'red' })
    }
  }

  const filteredWorkers = workers.filter(w =>
    w.name.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <Container size="xl" py="xl">
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={1}>Workers</Title>
          <Text c="dimmed" mt="xs">Deploy serverless JavaScript at the edge</Text>
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={() => setModalOpen(true)}>
          Create Worker
        </Button>
      </Group>

      <Card withBorder shadow="sm" radius="md">
        <Group mb="md">
          <TextInput
            placeholder="Search workers..."
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ flex: 1 }}
          />
        </Group>

        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Name</Table.Th>
              <Table.Th>Routes</Table.Th>
              <Table.Th>Last Updated</Table.Th>
              <Table.Th></Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {filteredWorkers.map((worker) => (
              <Table.Tr key={worker.id} style={{ cursor: 'pointer' }} onClick={() => navigate(`/workers/${worker.id}`)}>
                <Table.Td>
                  <Group gap="xs">
                    <IconBolt size={16} color="var(--mantine-color-yellow-6)" />
                    <Text fw={500}>{worker.name}</Text>
                  </Group>
                </Table.Td>
                <Table.Td>
                  <Group gap="xs">
                    {worker.routes?.slice(0, 2).map((route, i) => (
                      <Badge key={i} variant="outline" size="sm">{route}</Badge>
                    ))}
                    {worker.routes?.length > 2 && (
                      <Badge variant="outline" size="sm">+{worker.routes.length - 2}</Badge>
                    )}
                  </Group>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" c="dimmed">
                    {new Date(worker.updated_at).toLocaleDateString()}
                  </Text>
                </Table.Td>
                <Table.Td onClick={(e) => e.stopPropagation()}>
                  <Menu position="bottom-end">
                    <Menu.Target>
                      <ActionIcon variant="subtle">
                        <IconDotsVertical size={16} />
                      </ActionIcon>
                    </Menu.Target>
                    <Menu.Dropdown>
                      <Menu.Item leftSection={<IconEdit size={14} />} onClick={() => navigate(`/workers/${worker.id}`)}>
                        Edit
                      </Menu.Item>
                      <Menu.Item leftSection={<IconPlayerPlay size={14} />} onClick={() => navigate(`/workers/${worker.id}`)}>
                        Test
                      </Menu.Item>
                      <Menu.Divider />
                      <Menu.Item leftSection={<IconTrash size={14} />} color="red" onClick={(e) => deleteWorker(worker.id, e)}>
                        Delete
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>

        {filteredWorkers.length === 0 && !loading && (
          <Text c="dimmed" ta="center" py="xl">
            No workers found. Click "Create Worker" to deploy one.
          </Text>
        )}
      </Card>

      <Modal opened={modalOpen} onClose={() => setModalOpen(false)} title="Create Worker" size="xl">
        <Stack>
          <TextInput
            label="Worker Name"
            placeholder="my-worker"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          <div>
            <Text size="sm" fw={500} mb={4}>Script</Text>
            <Textarea
              placeholder="Worker script..."
              minRows={10}
              ff="monospace"
              value={form.script}
              onChange={(e) => setForm({ ...form, script: e.target.value })}
            />
          </div>
          <Group justify="flex-end" mt="md">
            <Button variant="subtle" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button onClick={createWorker}>Deploy</Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
