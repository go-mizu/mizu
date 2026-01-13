import { Container, Title, Text, Card, Table, Button, Group, Badge, TextInput, Modal, Stack, ActionIcon, Menu } from '@mantine/core'
import { IconPlus, IconSearch, IconDotsVertical, IconTrash, IconSettings, IconWorld } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface Zone {
  id: string
  name: string
  status: string
  plan: string
  created_at: string
}

export function ZonesList() {
  const navigate = useNavigate()
  const [zones, setZones] = useState<Zone[]>([])
  const [loading, setLoading] = useState(true)
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [newZoneName, setNewZoneName] = useState('')
  const [search, setSearch] = useState('')

  useEffect(() => {
    fetchZones()
  }, [])

  const fetchZones = async () => {
    try {
      const res = await fetch('/api/zones')
      const data = await res.json()
      setZones(data.result || [])
    } catch (error) {
      console.error('Failed to fetch zones:', error)
    } finally {
      setLoading(false)
    }
  }

  const createZone = async () => {
    if (!newZoneName) return
    try {
      const res = await fetch('/api/zones', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newZoneName }),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Zone created successfully', color: 'green' })
        setCreateModalOpen(false)
        setNewZoneName('')
        fetchZones()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create zone', color: 'red' })
    }
  }

  const deleteZone = async (id: string) => {
    try {
      const res = await fetch(`/api/zones/${id}`, { method: 'DELETE' })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Zone deleted', color: 'green' })
        fetchZones()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete zone', color: 'red' })
    }
  }

  const filteredZones = zones.filter(z =>
    z.name.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <Container size="xl" py="xl">
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={1}>Zones</Title>
          <Text c="dimmed" mt="xs">Manage your domains and DNS settings</Text>
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={() => setCreateModalOpen(true)}>
          Add Zone
        </Button>
      </Group>

      <Card withBorder shadow="sm" radius="md">
        <Group mb="md">
          <TextInput
            placeholder="Search zones..."
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ flex: 1 }}
          />
        </Group>

        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Domain</Table.Th>
              <Table.Th>Status</Table.Th>
              <Table.Th>Plan</Table.Th>
              <Table.Th>Created</Table.Th>
              <Table.Th></Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {filteredZones.map((zone) => (
              <Table.Tr key={zone.id} style={{ cursor: 'pointer' }} onClick={() => navigate(`/zones/${zone.id}`)}>
                <Table.Td>
                  <Group gap="xs">
                    <IconWorld size={16} />
                    <Text fw={500}>{zone.name}</Text>
                  </Group>
                </Table.Td>
                <Table.Td>
                  <Badge color={zone.status === 'active' ? 'green' : 'yellow'}>
                    {zone.status}
                  </Badge>
                </Table.Td>
                <Table.Td>
                  <Badge variant="outline">{zone.plan}</Badge>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" c="dimmed">
                    {new Date(zone.created_at).toLocaleDateString()}
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
                      <Menu.Item leftSection={<IconSettings size={14} />} onClick={() => navigate(`/zones/${zone.id}`)}>
                        Settings
                      </Menu.Item>
                      <Menu.Divider />
                      <Menu.Item leftSection={<IconTrash size={14} />} color="red" onClick={() => deleteZone(zone.id)}>
                        Delete
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>

        {filteredZones.length === 0 && !loading && (
          <Text c="dimmed" ta="center" py="xl">
            No zones found. Click "Add Zone" to create one.
          </Text>
        )}
      </Card>

      <Modal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Add Zone">
        <Stack>
          <TextInput
            label="Domain Name"
            placeholder="example.com"
            value={newZoneName}
            onChange={(e) => setNewZoneName(e.target.value)}
          />
          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => setCreateModalOpen(false)}>Cancel</Button>
            <Button onClick={createZone}>Add Zone</Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
