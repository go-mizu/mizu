import { Container, Title, Text, Card, Table, Button, Group, Badge, TextInput, Modal, Stack, Select, Switch, ActionIcon, Menu } from '@mantine/core'
import { IconPlus, IconSearch, IconDotsVertical, IconTrash, IconEdit } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface DNSRecord {
  id: string
  type: string
  name: string
  content: string
  ttl: number
  priority: number
  proxied: boolean
}

const recordTypes = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SRV', 'CAA', 'PTR']

export function DNSRecords() {
  const { id: zoneId } = useParams<{ id: string }>()
  const [records, setRecords] = useState<DNSRecord[]>([])
  const [, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [editRecord, setEditRecord] = useState<DNSRecord | null>(null)
  const [search, setSearch] = useState('')
  const [form, setForm] = useState({
    type: 'A',
    name: '',
    content: '',
    ttl: 300,
    priority: 10,
    proxied: true,
  })

  useEffect(() => {
    fetchRecords()
  }, [zoneId])

  const fetchRecords = async () => {
    try {
      const res = await fetch(`/api/zones/${zoneId}/dns/records`)
      const data = await res.json()
      setRecords(data.result || [])
    } catch (error) {
      console.error('Failed to fetch records:', error)
    } finally {
      setLoading(false)
    }
  }

  const saveRecord = async () => {
    const method = editRecord ? 'PUT' : 'POST'
    const url = editRecord
      ? `/api/zones/${zoneId}/dns/records/${editRecord.id}`
      : `/api/zones/${zoneId}/dns/records`

    try {
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Record saved', color: 'green' })
        setModalOpen(false)
        setEditRecord(null)
        setForm({ type: 'A', name: '', content: '', ttl: 300, priority: 10, proxied: true })
        fetchRecords()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to save record', color: 'red' })
    }
  }

  const deleteRecord = async (recordId: string) => {
    try {
      const res = await fetch(`/api/zones/${zoneId}/dns/records/${recordId}`, { method: 'DELETE' })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Record deleted', color: 'green' })
        fetchRecords()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete record', color: 'red' })
    }
  }

  const openEditModal = (record: DNSRecord) => {
    setEditRecord(record)
    setForm({
      type: record.type,
      name: record.name,
      content: record.content,
      ttl: record.ttl,
      priority: record.priority,
      proxied: record.proxied,
    })
    setModalOpen(true)
  }

  const filteredRecords = records.filter(r =>
    r.name.toLowerCase().includes(search.toLowerCase()) ||
    r.content.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <Container size="xl" py="xl">
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={1}>DNS Records</Title>
          <Text c="dimmed" mt="xs">Manage DNS records for your domain</Text>
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={() => {
          setEditRecord(null)
          setForm({ type: 'A', name: '', content: '', ttl: 300, priority: 10, proxied: true })
          setModalOpen(true)
        }}>
          Add Record
        </Button>
      </Group>

      <Card withBorder shadow="sm" radius="md">
        <Group mb="md">
          <TextInput
            placeholder="Search records..."
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ flex: 1 }}
          />
        </Group>

        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Type</Table.Th>
              <Table.Th>Name</Table.Th>
              <Table.Th>Content</Table.Th>
              <Table.Th>TTL</Table.Th>
              <Table.Th>Proxy</Table.Th>
              <Table.Th></Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {filteredRecords.map((record) => (
              <Table.Tr key={record.id}>
                <Table.Td>
                  <Badge variant="outline">{record.type}</Badge>
                </Table.Td>
                <Table.Td>
                  <Text ff="monospace" size="sm">{record.name}</Text>
                </Table.Td>
                <Table.Td>
                  <Text ff="monospace" size="sm" lineClamp={1}>{record.content}</Text>
                </Table.Td>
                <Table.Td>
                  <Text size="sm">{record.ttl === 1 ? 'Auto' : `${record.ttl}s`}</Text>
                </Table.Td>
                <Table.Td>
                  <Badge color={record.proxied ? 'orange' : 'gray'}>
                    {record.proxied ? 'Proxied' : 'DNS only'}
                  </Badge>
                </Table.Td>
                <Table.Td>
                  <Menu position="bottom-end">
                    <Menu.Target>
                      <ActionIcon variant="subtle">
                        <IconDotsVertical size={16} />
                      </ActionIcon>
                    </Menu.Target>
                    <Menu.Dropdown>
                      <Menu.Item leftSection={<IconEdit size={14} />} onClick={() => openEditModal(record)}>
                        Edit
                      </Menu.Item>
                      <Menu.Divider />
                      <Menu.Item leftSection={<IconTrash size={14} />} color="red" onClick={() => deleteRecord(record.id)}>
                        Delete
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      </Card>

      <Modal opened={modalOpen} onClose={() => setModalOpen(false)} title={editRecord ? 'Edit Record' : 'Add Record'} size="lg">
        <Stack>
          <Select
            label="Type"
            data={recordTypes}
            value={form.type}
            onChange={(v) => setForm({ ...form, type: v || 'A' })}
          />
          <TextInput
            label="Name"
            placeholder="@ or subdomain"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          <TextInput
            label="Content"
            placeholder={form.type === 'A' ? 'IPv4 address' : form.type === 'AAAA' ? 'IPv6 address' : 'Value'}
            value={form.content}
            onChange={(e) => setForm({ ...form, content: e.target.value })}
          />
          <Select
            label="TTL"
            data={[
              { value: '1', label: 'Auto' },
              { value: '60', label: '1 minute' },
              { value: '300', label: '5 minutes' },
              { value: '3600', label: '1 hour' },
              { value: '86400', label: '1 day' },
            ]}
            value={String(form.ttl)}
            onChange={(v) => setForm({ ...form, ttl: parseInt(v || '300') })}
          />
          {form.type === 'MX' && (
            <TextInput
              label="Priority"
              type="number"
              value={form.priority}
              onChange={(e) => setForm({ ...form, priority: parseInt(e.target.value) || 10 })}
            />
          )}
          {['A', 'AAAA', 'CNAME'].includes(form.type) && (
            <Switch
              label="Proxy through Localflare"
              checked={form.proxied}
              onChange={(e) => setForm({ ...form, proxied: e.target.checked })}
            />
          )}
          <Group justify="flex-end" mt="md">
            <Button variant="subtle" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button onClick={saveRecord}>Save</Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
