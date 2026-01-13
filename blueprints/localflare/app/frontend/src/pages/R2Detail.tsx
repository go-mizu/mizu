import { Container, Title, Text, Card, Table, Button, Group, TextInput, Modal, Stack, ActionIcon, Badge, FileInput } from '@mantine/core'
import { IconPlus, IconSearch, IconTrash, IconCloud, IconDownload, IconFile, IconFolder } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { notifications } from '@mantine/notifications'

interface R2Object {
  key: string
  size: number
  etag: string
  last_modified: string
}

interface R2Bucket {
  id: string
  name: string
}

export function R2Detail() {
  const { id } = useParams<{ id: string }>()
  const [bucket, setBucket] = useState<R2Bucket | null>(null)
  const [objects, setObjects] = useState<R2Object[]>([])
  const [loading, setLoading] = useState(true)
  const [uploadModalOpen, setUploadModalOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [prefix, setPrefix] = useState('')
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [uploadKey, setUploadKey] = useState('')

  useEffect(() => {
    fetchBucket()
    fetchObjects()
  }, [id, prefix])

  const fetchBucket = async () => {
    try {
      const res = await fetch(`/api/r2/buckets/${id}`)
      const data = await res.json()
      setBucket(data.result)
    } catch (error) {
      console.error('Failed to fetch bucket:', error)
    }
  }

  const fetchObjects = async () => {
    try {
      const url = prefix
        ? `/api/r2/buckets/${id}/objects?prefix=${encodeURIComponent(prefix)}`
        : `/api/r2/buckets/${id}/objects`
      const res = await fetch(url)
      const data = await res.json()
      setObjects(data.result || [])
    } catch (error) {
      console.error('Failed to fetch objects:', error)
    } finally {
      setLoading(false)
    }
  }

  const uploadObject = async () => {
    if (!uploadFile || !uploadKey) return
    try {
      const formData = new FormData()
      formData.append('file', uploadFile)

      const res = await fetch(`/api/r2/buckets/${id}/objects/${encodeURIComponent(uploadKey)}`, {
        method: 'PUT',
        body: formData,
      })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Object uploaded', color: 'green' })
        setUploadModalOpen(false)
        setUploadFile(null)
        setUploadKey('')
        fetchObjects()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to upload object', color: 'red' })
    }
  }

  const deleteObject = async (key: string) => {
    try {
      const res = await fetch(`/api/r2/buckets/${id}/objects/${encodeURIComponent(key)}`, { method: 'DELETE' })
      if (res.ok) {
        notifications.show({ title: 'Success', message: 'Object deleted', color: 'green' })
        fetchObjects()
      }
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to delete object', color: 'red' })
    }
  }

  const downloadObject = async (key: string) => {
    try {
      const res = await fetch(`/api/r2/buckets/${id}/objects/${encodeURIComponent(key)}`)
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = key.split('/').pop() || key
      a.click()
      URL.revokeObjectURL(url)
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to download object', color: 'red' })
    }
  }

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const filteredObjects = objects.filter(o =>
    o.key.toLowerCase().includes(search.toLowerCase())
  )

  if (!bucket) return <Container py="xl"><Text>Loading...</Text></Container>

  return (
    <Container size="xl" py="xl">
      <Group justify="space-between" mb="xl">
        <Group>
          <IconCloud size={32} color="var(--mantine-color-grape-6)" />
          <div>
            <Title order={1}>{bucket.name}</Title>
            <Text c="dimmed">R2 Bucket</Text>
          </div>
        </Group>
        <Button leftSection={<IconPlus size={16} />} onClick={() => setUploadModalOpen(true)}>
          Upload Object
        </Button>
      </Group>

      <Card withBorder shadow="sm" radius="md">
        <Group mb="md">
          <TextInput
            placeholder="Search objects..."
            leftSection={<IconSearch size={16} />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ flex: 1 }}
          />
          <TextInput
            placeholder="Prefix filter..."
            leftSection={<IconFolder size={16} />}
            value={prefix}
            onChange={(e) => setPrefix(e.target.value)}
            w={200}
          />
        </Group>

        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Key</Table.Th>
              <Table.Th>Size</Table.Th>
              <Table.Th>Last Modified</Table.Th>
              <Table.Th>Actions</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {filteredObjects.map((obj) => (
              <Table.Tr key={obj.key}>
                <Table.Td>
                  <Group gap="xs">
                    <IconFile size={16} />
                    <Text ff="monospace" size="sm">{obj.key}</Text>
                  </Group>
                </Table.Td>
                <Table.Td>
                  <Badge variant="outline">{formatSize(obj.size)}</Badge>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" c="dimmed">
                    {new Date(obj.last_modified).toLocaleString()}
                  </Text>
                </Table.Td>
                <Table.Td>
                  <Group gap="xs">
                    <ActionIcon variant="subtle" onClick={() => downloadObject(obj.key)}>
                      <IconDownload size={14} />
                    </ActionIcon>
                    <ActionIcon variant="subtle" color="red" onClick={() => deleteObject(obj.key)}>
                      <IconTrash size={14} />
                    </ActionIcon>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>

        {filteredObjects.length === 0 && !loading && (
          <Text c="dimmed" ta="center" py="xl">
            No objects found. Click "Upload Object" to add one.
          </Text>
        )}
      </Card>

      <Modal opened={uploadModalOpen} onClose={() => setUploadModalOpen(false)} title="Upload Object">
        <Stack>
          <TextInput
            label="Object Key"
            placeholder="path/to/file.txt"
            value={uploadKey}
            onChange={(e) => setUploadKey(e.target.value)}
          />
          <FileInput
            label="File"
            placeholder="Select file..."
            value={uploadFile}
            onChange={setUploadFile}
          />
          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => setUploadModalOpen(false)}>Cancel</Button>
            <Button onClick={uploadObject} disabled={!uploadFile || !uploadKey}>Upload</Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  )
}
