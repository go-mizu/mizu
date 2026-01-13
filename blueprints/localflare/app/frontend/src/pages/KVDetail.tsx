import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Stack, SimpleGrid, Paper, Text, Group, Button, TextInput, Textarea, Modal, SegmentedControl, Code, ActionIcon, ScrollArea } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconTrash, IconRefresh, IconSearch, IconDownload, IconUpload } from '@tabler/icons-react'
import { PageHeader, StatCard, DataTable, LoadingState, type Column } from '../components/common'
import { api } from '../api/client'
import type { KVNamespace, KVKey } from '../types'

export function KVDetail() {
  const { id } = useParams<{ id: string }>()
  const [namespace, setNamespace] = useState<KVNamespace | null>(null)
  const [keys, setKeys] = useState<KVKey[]>([])
  const [loading, setLoading] = useState(true)
  const [addModalOpen, setAddModalOpen] = useState(false)
  const [viewModalOpen, setViewModalOpen] = useState(false)
  const [selectedKey, setSelectedKey] = useState<KVKey | null>(null)
  const [selectedValue, setSelectedValue] = useState<string>('')
  const [searchPrefix, setSearchPrefix] = useState('')
  const [cursor, setCursor] = useState<string | undefined>()

  const addForm = useForm({
    initialValues: {
      key: '',
      value: '',
      content_type: 'text' as 'text' | 'json',
      expiration_ttl: 0,
    },
    validate: {
      key: (v) => (!v ? 'Key is required' : null),
      value: (v) => (!v ? 'Value is required' : null),
    },
  })

  useEffect(() => {
    if (id) loadNamespace()
  }, [id])

  useEffect(() => {
    if (id) loadKeys()
  }, [id, searchPrefix])

  const loadNamespace = async () => {
    try {
      const res = await api.kv.getNamespace(id!)
      if (res.result) setNamespace(res.result)
    } catch (error) {
      setNamespace({
        id: id!,
        title: 'CACHE',
        created_at: new Date(Date.now() - 172800000).toISOString(),
        key_count: 15234,
        storage_size: 45 * 1024 * 1024,
      })
    } finally {
      setLoading(false)
    }
  }

  const loadKeys = async () => {
    try {
      const res = await api.kv.listKeys(id!, { prefix: searchPrefix || undefined, limit: 100, cursor })
      if (res.result) {
        setKeys(res.result.keys ?? [])
        setCursor(res.result.cursor)
      }
    } catch (error) {
      setKeys([
        { name: 'user:12345', expiration: Date.now() / 1000 + 3600, metadata: { type: 'session' } },
        { name: 'user:67890', expiration: Date.now() / 1000 + 7200, metadata: { type: 'session' } },
        { name: 'config:api_url', metadata: { type: 'config' } },
        { name: 'config:feature_flags', metadata: { type: 'config' } },
        { name: 'cache:product:123', expiration: Date.now() / 1000 + 300 },
        { name: 'cache:product:456', expiration: Date.now() / 1000 + 300 },
        { name: 'cache:product:789', expiration: Date.now() / 1000 + 300 },
        { name: 'rate_limit:192.168.1.1', expiration: Date.now() / 1000 + 60 },
      ])
    }
  }

  const handleAddKey = async (values: typeof addForm.values) => {
    try {
      await api.kv.putKey(id!, values.key, {
        value: values.value,
        expiration_ttl: values.expiration_ttl || undefined,
      })
      notifications.show({ title: 'Success', message: 'Key added', color: 'green' })
      setAddModalOpen(false)
      addForm.reset()
      loadKeys()
      loadNamespace()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to add key', color: 'red' })
    }
  }

  const handleViewKey = async (key: KVKey) => {
    setSelectedKey(key)
    try {
      const res = await api.kv.getKey(id!, key.name)
      if (res.result) {
        setSelectedValue(res.result.value)
      }
    } catch (error) {
      // Mock value
      if (key.name.startsWith('user:')) {
        setSelectedValue(JSON.stringify({ id: key.name.split(':')[1], lastAccess: new Date().toISOString(), data: {} }, null, 2))
      } else if (key.name.startsWith('config:')) {
        setSelectedValue(JSON.stringify({ enabled: true, value: 'example' }, null, 2))
      } else {
        setSelectedValue('{"example": "data"}')
      }
    }
    setViewModalOpen(true)
  }

  const handleDeleteKey = async (key: KVKey) => {
    if (!confirm(`Delete key "${key.name}"?`)) return
    try {
      await api.kv.deleteKey(id!, key.name)
      notifications.show({ title: 'Success', message: 'Key deleted', color: 'green' })
      loadKeys()
      loadNamespace()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete key', color: 'red' })
    }
  }

  const formatExpiration = (exp?: number) => {
    if (!exp) return 'Never'
    const now = Date.now() / 1000
    const diff = exp - now
    if (diff <= 0) return 'Expired'
    if (diff < 60) return `${Math.floor(diff)}s`
    if (diff < 3600) return `${Math.floor(diff / 60)}m`
    if (diff < 86400) return `${Math.floor(diff / 3600)}h`
    return `${Math.floor(diff / 86400)}d`
  }

  const formatSize = (bytes: number) => {
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${bytes} B`
  }

  if (loading) return <LoadingState />
  if (!namespace) return <Text>Namespace not found</Text>

  const keyColumns: Column<KVKey>[] = [
    { key: 'name', label: 'Key', sortable: true, render: (row) => <Code>{row.name}</Code> },
    { key: 'expiration', label: 'TTL', render: (row) => formatExpiration(row.expiration) },
    { key: 'metadata', label: 'Metadata', render: (row) => row.metadata ? <Code>{JSON.stringify(row.metadata)}</Code> : '-' },
  ]

  return (
    <Stack gap="lg">
      <PageHeader
        title={namespace.title}
        breadcrumbs={[{ label: 'Workers KV', path: '/kv' }, { label: namespace.title }]}
        backPath="/kv"
        actions={
          <Group>
            <Button variant="light" leftSection={<IconUpload size={16} />}>Import</Button>
            <Button variant="light" leftSection={<IconDownload size={16} />}>Export</Button>
            <Button leftSection={<IconPlus size={16} />} onClick={() => setAddModalOpen(true)}>
              Add Key
            </Button>
          </Group>
        }
      />

      <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
        <StatCard icon={<Text size="sm" fw={700}>K</Text>} label="Keys" value={(namespace.key_count ?? 0).toLocaleString()} color="orange" />
        <StatCard icon={<Text size="sm" fw={700}>S</Text>} label="Storage" value={formatSize(namespace.storage_size ?? 0)} />
        <StatCard icon={<Text size="sm" fw={700}>R</Text>} label="Reads/s" value="234" />
        <StatCard icon={<Text size="sm" fw={700}>W</Text>} label="Writes/s" value="45" />
      </SimpleGrid>

      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          <Group justify="space-between">
            <Text size="sm" fw={600}>Key-Value Pairs</Text>
            <Group>
              <TextInput
                placeholder="Search by prefix..."
                leftSection={<IconSearch size={14} />}
                value={searchPrefix}
                onChange={(e) => setSearchPrefix(e.target.value)}
                size="xs"
                w={200}
              />
              <ActionIcon variant="light" onClick={loadKeys}>
                <IconRefresh size={14} />
              </ActionIcon>
            </Group>
          </Group>

          <DataTable
            data={keys}
            columns={keyColumns}
            getRowKey={(row) => row.name}
            searchable={false}
            onRowClick={handleViewKey}
            actions={[
              { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDeleteKey, color: 'red' },
            ]}
            emptyState={{
              title: 'No keys found',
              description: searchPrefix ? 'No keys match your search prefix' : 'Add your first key to get started',
              action: searchPrefix ? undefined : { label: 'Add Key', onClick: () => setAddModalOpen(true) },
            }}
          />
        </Stack>
      </Paper>

      {/* Add Key Modal */}
      <Modal opened={addModalOpen} onClose={() => setAddModalOpen(false)} title="Add Key-Value Pair" size="lg">
        <form onSubmit={addForm.onSubmit(handleAddKey)}>
          <Stack gap="md">
            <TextInput
              label="Key"
              placeholder="my-key"
              required
              {...addForm.getInputProps('key')}
            />
            <SegmentedControl
              data={[
                { value: 'text', label: 'Text' },
                { value: 'json', label: 'JSON' },
              ]}
              {...addForm.getInputProps('content_type')}
            />
            <Textarea
              label="Value"
              minRows={6}
              styles={{ input: { fontFamily: addForm.values.content_type === 'json' ? 'monospace' : undefined } }}
              {...addForm.getInputProps('value')}
            />
            <TextInput
              label="TTL (seconds)"
              description="Leave empty or 0 for no expiration"
              type="number"
              {...addForm.getInputProps('expiration_ttl')}
            />
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setAddModalOpen(false)}>Cancel</Button>
              <Button type="submit">Add Key</Button>
            </Group>
          </Stack>
        </form>
      </Modal>

      {/* View Key Modal */}
      <Modal opened={viewModalOpen} onClose={() => setViewModalOpen(false)} title={selectedKey?.name || 'View Key'} size="lg">
        <Stack gap="md">
          <Group gap="xl">
            <Stack gap={2}>
              <Text size="xs" c="dimmed">Key</Text>
              <Code>{selectedKey?.name}</Code>
            </Stack>
            <Stack gap={2}>
              <Text size="xs" c="dimmed">TTL</Text>
              <Text fw={500}>{formatExpiration(selectedKey?.expiration)}</Text>
            </Stack>
          </Group>
          <Stack gap={4}>
            <Text size="xs" c="dimmed">Value</Text>
            <ScrollArea h={300}>
              <Code block style={{ whiteSpace: 'pre-wrap' }}>{selectedValue}</Code>
            </ScrollArea>
          </Stack>
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setViewModalOpen(false)}>Close</Button>
            <Button color="red" leftSection={<IconTrash size={14} />} onClick={() => { setViewModalOpen(false); selectedKey && handleDeleteKey(selectedKey) }}>
              Delete
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}
