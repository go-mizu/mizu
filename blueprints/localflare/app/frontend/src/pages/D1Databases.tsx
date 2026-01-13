import { Container, Title, Text, Card, Table, Button, Group, TextInput, Modal, Stack, ActionIcon, Menu } from '@mantine/core'
import { IconPlus, IconSearch, IconDotsVertical, IconTrash, IconDatabase } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface D1Database {
  id: string
  name: string
  created_at: string
}

export function D1Databases() {
  const navigate = useNavigate()
  const [databases, setDatabases] = useState<D1Database[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [newName, setNewName] = useState('')

  useEffect(() => {
    fetchDatabases()
  }, [])

  const fetchDatabases = async () => {
    try {
      const res = await fetch('/api/d1/databases')
      const data = await res.json()
      setDatabases(data.result || [])
    } catch (error) {
      console.error('Failed to fetch databases:', error)
    } finally {
      setLoading(false)
    }
  }

  const createDatabase = async () => {
    try {
      const res = await fetch('/api/d1/databases', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newName }),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Database created', color: 'green' })
        setModalOpen(false)
        setNewName('')
        fetchDatabases()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create database', color: 'red' })
    }
  }

  const deleteDatabase = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      const res = await fetch(`/api/d1/databases/${id}`, { method: 'DELETE' })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Database deleted', color: 'green' })
        fetchDatabases()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete database', color: 'red' })
    }
  }

  const filteredDatabases = databases.filter(db =>
    db.name.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <Container size="xl" py="xl">
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={1}>D1 Databases</Title>
          <Text c="dimmed" mt="xs">Serverless SQL databases at the edge</Text>
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={() => setModalOpen(true)}>
          Create Database
        </Button>
      </Group>

      <Card withBorder shadow="sm" radius="md">
        <Group mb="md">
          <TextInput
            placeholder="Search databases..."
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
            {filteredDatabases.map((db) => (
              <Table.Tr key={db.id} style={{ cursor: 'pointer' }} onClick={() => navigate(`/d1/${db.id}`)}>
                <Table.Td>
                  <Group gap="xs">
                    <IconDatabase size={16} color="var(--mantine-color-cyan-6)" />
                    <Text fw={500}>{db.name}</Text>
                  </Group>
                </Table.Td>
                <Table.Td>
                  <Text ff="monospace" size="sm" c="dimmed">{db.id}</Text>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" c="dimmed">
                    {new Date(db.created_at).toLocaleDateString()}
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
                      <Menu.Item leftSection={<IconTrash size={14} />} color="red" onClick={(e) => deleteDatabase(db.id, e)}>
                        Delete
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>

        {filteredDatabases.length === 0 && !loading && (
          <Text c="dimmed" ta="center" py="xl">
            No databases found. Click "Create Database" to add one.
          </Text>
        )}
      </Card>

      <Modal opened={modalOpen} onClose={() => setModalOpen(false)} title="Create D1 Database">
        <Stack>
          <TextInput
            label="Database Name"
            placeholder="my-database"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
          />
          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button onClick={createDatabase}>Create</Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
