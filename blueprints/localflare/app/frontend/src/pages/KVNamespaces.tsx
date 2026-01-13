import { Container, Title, Text, Card, Table, Button, Group, TextInput, Modal, Stack, ActionIcon, Menu } from '@mantine/core'
import { IconPlus, IconSearch, IconDotsVertical, IconTrash, IconKey } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface KVNamespace {
  id: string
  name: string
  created_at: string
}

export function KVNamespaces() {
  const navigate = useNavigate()
  const [namespaces, setNamespaces] = useState<KVNamespace[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [newName, setNewName] = useState('')

  useEffect(() => {
    fetchNamespaces()
  }, [])

  const fetchNamespaces = async () => {
    try {
      const res = await fetch('/api/kv/namespaces')
      const data = await res.json()
      setNamespaces(data.result || [])
    } catch (error) {
      console.error('Failed to fetch namespaces:', error)
    } finally {
      setLoading(false)
    }
  }

  const createNamespace = async () => {
    try {
      const res = await fetch('/api/kv/namespaces', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newName }),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Namespace created', color: 'green' })
        setModalOpen(false)
        setNewName('')
        fetchNamespaces()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create namespace', color: 'red' })
    }
  }

  const deleteNamespace = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      const res = await fetch(`/api/kv/namespaces/${id}`, { method: 'DELETE' })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Namespace deleted', color: 'green' })
        fetchNamespaces()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete namespace', color: 'red' })
    }
  }

  const filteredNamespaces = namespaces.filter(ns =>
    ns.name.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <Container size="xl" py="xl">
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={1}>KV Namespaces</Title>
          <Text c="dimmed" mt="xs">Global, low-latency key-value data storage</Text>
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={() => setModalOpen(true)}>
          Create Namespace
        </Button>
      </Group>

      <Card withBorder shadow="sm" radius="md">
        <Group mb="md">
          <TextInput
            placeholder="Search namespaces..."
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
              <Table.Th>ID</Table.Th>
              <Table.Th>Created</Table.Th>
              <Table.Th></Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {filteredNamespaces.map((ns) => (
              <Table.Tr key={ns.id} style={{ cursor: 'pointer' }} onClick={() => navigate(`/kv/${ns.id}`)}>
                <Table.Td>
                  <Group gap="xs">
                    <IconKey size={16} color="var(--mantine-color-green-6)" />
                    <Text fw={500}>{ns.name}</Text>
                  </Group>
                </Table.Td>
                <Table.Td>
                  <Text ff="monospace" size="sm" c="dimmed">{ns.id}</Text>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" c="dimmed">
                    {new Date(ns.created_at).toLocaleDateString()}
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
                      <Menu.Item leftSection={<IconTrash size={14} />} color="red" onClick={(e) => deleteNamespace(ns.id, e)}>
                        Delete
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>

        {filteredNamespaces.length === 0 && !loading && (
          <Text c="dimmed" ta="center" py="xl">
            No namespaces found. Click "Create Namespace" to add one.
          </Text>
        )}
      </Card>

      <Modal opened={modalOpen} onClose={() => setModalOpen(false)} title="Create KV Namespace">
        <Stack>
          <TextInput
            label="Namespace Name"
            placeholder="MY_KV_NAMESPACE"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
          />
          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button onClick={createNamespace}>Create</Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
