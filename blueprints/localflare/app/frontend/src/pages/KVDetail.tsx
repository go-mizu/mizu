import { Container, Title, Text, Card, Table, Button, Group, TextInput, Modal, Stack, ActionIcon, Textarea, Code } from '@mantine/core'
import { IconPlus, IconSearch, IconTrash, IconKey, IconEdit } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface KVEntry {
  key: string
  value: string
  metadata?: Record<string, unknown>
}

interface KVNamespace {
  id: string
  name: string
}

export function KVDetail() {
  const { id } = useParams<{ id: string }>()
  const [namespace, setNamespace] = useState<KVNamespace | null>(null)
  const [entries, setEntries] = useState<KVEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [editEntry, setEditEntry] = useState<KVEntry | null>(null)
  const [search, setSearch] = useState('')
  const [form, setForm] = useState({ key: '', value: '' })

  useEffect(() => {
    fetchNamespace()
    fetchEntries()
  }, [id])

  const fetchNamespace = async () => {
    try {
      const res = await fetch(`/api/kv/namespaces/${id}`)
      const data = await res.json()
      setNamespace(data.result)
    } catch (error) {
      console.error('Failed to fetch namespace:', error)
    }
  }

  const fetchEntries = async () => {
    try {
      const res = await fetch(`/api/kv/namespaces/${id}/keys`)
      const data = await res.json()
      setEntries(data.result || [])
    } catch (error) {
      console.error('Failed to fetch entries:', error)
    } finally {
      setLoading(false)
    }
  }

  const saveEntry = async () => {
    try {
      const res = await fetch(`/api/kv/namespaces/${id}/values/${encodeURIComponent(form.key)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ value: form.value }),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Entry saved', color: 'green' })
        setModalOpen(false)
        setEditEntry(null)
        setForm({ key: '', value: '' })
        fetchEntries()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to save entry', color: 'red' })
    }
  }

  const deleteEntry = async (key: string) => {
    try {
      const res = await fetch(`/api/kv/namespaces/${id}/values/${encodeURIComponent(key)}`, { method: 'DELETE' })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Entry deleted', color: 'green' })
        fetchEntries()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete entry', color: 'red' })
    }
  }

  const openEditModal = async (entry: KVEntry) => {
    try {
      const res = await fetch(`/api/kv/namespaces/${id}/values/${encodeURIComponent(entry.key)}`)
      const data = await res.json()
      setEditEntry(entry)
      setForm({ key: entry.key, value: data.result || '' })
      setModalOpen(true)
    } catch (error) {
      console.error('Failed to fetch value:', error)
    }
  }

  const filteredEntries = entries.filter(e =>
    e.key.toLowerCase().includes(search.toLowerCase())
  )

  if (!namespace) return <Container py="xl"><Text>Loading...</Text></Container>

  return (
    <Container size="xl" py="xl">
      <Group justify="space-between" mb="xl">
        <Group>
          <IconKey size={32} color="var(--mantine-color-green-6)" />
          <div>
            <Title order={1}>{namespace.name}</Title>
            <Text c="dimmed">KV Namespace</Text>
          </div>
        </Group>
        <Button leftSection={<IconPlus size={16} />} onClick={() => {
          setEditEntry(null)
          setForm({ key: '', value: '' })
          setModalOpen(true)
        }}>
          Add Entry
        </Button>
      </Group>

      <Card withBorder shadow="sm" radius="md">
        <Group mb="md">
          <TextInput
            placeholder="Search keys..."
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ flex: 1 }}
          />
        </Group>

        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Key</Table.Th>
              <Table.Th>Actions</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {filteredEntries.map((entry) => (
              <Table.Tr key={entry.key}>
                <Table.Td>
                  <Code>{entry.key}</Code>
                </Table.Td>
                <Table.Td>
                  <Group gap="xs">
                    <ActionIcon variant="subtle" onClick={() => openEditModal(entry)}>
                      <IconEdit size={14} />
                    </ActionIcon>
                    <ActionIcon variant="subtle" color="red" onClick={() => deleteEntry(entry.key)}>
                      <IconTrash size={14} />
                    </ActionIcon>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>

        {filteredEntries.length === 0 && !loading && (
          <Text c="dimmed" ta="center" py="xl">
            No entries found. Click "Add Entry" to create one.
          </Text>
        )}
      </Card>

      <Modal opened={modalOpen} onClose={() => setModalOpen(false)} title={editEntry ? 'Edit Entry' : 'Add Entry'} size="lg">
        <Stack>
          <TextInput
            label="Key"
            placeholder="my-key"
            value={form.key}
            onChange={(e) => setForm({ ...form, key: e.target.value })}
            disabled={!!editEntry}
          />
          <Textarea
            label="Value"
            placeholder="Enter value..."
            minRows={5}
            value={form.value}
            onChange={(e) => setForm({ ...form, value: e.target.value })}
          />
          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button onClick={saveEntry}>Save</Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
