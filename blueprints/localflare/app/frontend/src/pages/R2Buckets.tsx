import { Container, Title, Text, Card, Table, Button, Group, Badge, TextInput, Modal, Stack, ActionIcon, Menu } from '@mantine/core'
import { IconPlus, IconSearch, IconDotsVertical, IconTrash, IconCloud } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface R2Bucket {
  id: string
  name: string
  location: string
  created_at: string
}

export function R2Buckets() {
  const navigate = useNavigate()
  const [buckets, setBuckets] = useState<R2Bucket[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [newName, setNewName] = useState('')

  useEffect(() => {
    fetchBuckets()
  }, [])

  const fetchBuckets = async () => {
    try {
      const res = await fetch('/api/r2/buckets')
      const data = await res.json()
      setBuckets(data.result || [])
    } catch (error) {
      console.error('Failed to fetch buckets:', error)
    } finally {
      setLoading(false)
    }
  }

  const createBucket = async () => {
    try {
      const res = await fetch('/api/r2/buckets', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newName }),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Bucket created', color: 'green' })
        setModalOpen(false)
        setNewName('')
        fetchBuckets()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create bucket', color: 'red' })
    }
  }

  const deleteBucket = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      const res = await fetch(`/api/r2/buckets/${id}`, { method: 'DELETE' })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Bucket deleted', color: 'green' })
        fetchBuckets()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete bucket', color: 'red' })
    }
  }

  const filteredBuckets = buckets.filter(b =>
    b.name.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <Container size="xl" py="xl">
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={1}>R2 Buckets</Title>
          <Text c="dimmed" mt="xs">S3-compatible object storage</Text>
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={() => setModalOpen(true)}>
          Create Bucket
        </Button>
      </Group>

      <Card withBorder shadow="sm" radius="md">
        <Group mb="md">
          <TextInput
            placeholder="Search buckets..."
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
              <Table.Th>Location</Table.Th>
              <Table.Th>Created</Table.Th>
              <Table.Th></Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {filteredBuckets.map((bucket) => (
              <Table.Tr key={bucket.id} style={{ cursor: 'pointer' }} onClick={() => navigate(`/r2/${bucket.id}`)}>
                <Table.Td>
                  <Group gap="xs">
                    <IconCloud size={16} color="var(--mantine-color-grape-6)" />
                    <Text fw={500}>{bucket.name}</Text>
                  </Group>
                </Table.Td>
                <Table.Td>
                  <Badge variant="outline">{bucket.location || 'auto'}</Badge>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" c="dimmed">
                    {new Date(bucket.created_at).toLocaleDateString()}
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
                      <Menu.Item leftSection={<IconTrash size={14} />} color="red" onClick={(e) => deleteBucket(bucket.id, e)}>
                        Delete
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>

        {filteredBuckets.length === 0 && !loading && (
          <Text c="dimmed" ta="center" py="xl">
            No buckets found. Click "Create Bucket" to add one.
          </Text>
        )}
      </Card>

      <Modal opened={modalOpen} onClose={() => setModalOpen(false)} title="Create R2 Bucket">
        <Stack>
          <TextInput
            label="Bucket Name"
            placeholder="my-bucket"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
          />
          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button onClick={createBucket}>Create</Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
