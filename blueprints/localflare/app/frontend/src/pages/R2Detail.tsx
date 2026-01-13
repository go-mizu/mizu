import { useState, useEffect } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { Stack, SimpleGrid, Paper, Text, Group, Button, TextInput, Modal, ActionIcon, Breadcrumbs, Anchor, Badge, Table, FileButton, Progress, Tabs, Switch, ScrollArea } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconUpload, IconTrash, IconRefresh, IconSearch, IconFolder, IconFile, IconDownload, IconLink, IconChevronRight, IconArrowLeft, IconSettings, IconWorld, IconLock, IconPlus } from '@tabler/icons-react'
import { PageHeader, StatCard, LoadingState } from '../components/common'
import { api } from '../api/client'
import type { R2Bucket, R2Object } from '../types'

export function R2Detail() {
  const { name } = useParams<{ name: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const prefix = searchParams.get('prefix') || ''

  const [bucket, setBucket] = useState<R2Bucket | null>(null)
  const [objects, setObjects] = useState<R2Object[]>([])
  const [folders, setFolders] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedObjects, setSelectedObjects] = useState<string[]>([])
  const [uploadProgress, setUploadProgress] = useState<number | null>(null)
  const [settingsOpen, setSettingsOpen] = useState(false)

  useEffect(() => {
    if (name) {
      loadBucket()
      loadObjects()
    }
  }, [name, prefix])

  const loadBucket = async () => {
    try {
      const res = await api.r2.getBucket(name!)
      if (res.result) setBucket(res.result)
    } catch (error) {
      setBucket({
        name: name!,
        created_at: new Date(Date.now() - 172800000).toISOString(),
        location: 'WNAM',
        object_count: 12456,
        storage_size: 2.5 * 1024 * 1024 * 1024,
        public_access: true,
      })
    }
  }

  const loadObjects = async () => {
    setLoading(true)
    try {
      const res = await api.r2.listObjects(name!, { prefix, delimiter: '/' })
      if (res.result) {
        setObjects(res.result.objects ?? [])
        setFolders(res.result.common_prefixes ?? [])
      }
    } catch (error) {
      // Mock data
      if (prefix === '') {
        setFolders(['images/', 'documents/', 'videos/'])
        setObjects([
          { key: 'index.html', size: 4567, last_modified: new Date().toISOString(), etag: 'abc123' },
          { key: 'style.css', size: 12345, last_modified: new Date().toISOString(), etag: 'def456' },
          { key: 'favicon.ico', size: 1024, last_modified: new Date().toISOString(), etag: 'ghi789' },
        ])
      } else if (prefix === 'images/') {
        setFolders(['images/thumbnails/', 'images/originals/'])
        setObjects([
          { key: 'images/hero.jpg', size: 245678, last_modified: new Date().toISOString(), etag: 'img001' },
          { key: 'images/logo.png', size: 34567, last_modified: new Date().toISOString(), etag: 'img002' },
          { key: 'images/banner.webp', size: 156789, last_modified: new Date().toISOString(), etag: 'img003' },
        ])
      } else {
        setFolders([])
        setObjects([])
      }
    } finally {
      setLoading(false)
    }
  }

  const handleUpload = async (files: File[]) => {
    if (!files.length) return
    setUploadProgress(0)
    try {
      for (let i = 0; i < files.length; i++) {
        const file = files[i]
        const key = prefix + file.name
        await api.r2.putObject(name!, key, file)
        setUploadProgress(((i + 1) / files.length) * 100)
      }
      notifications.show({ title: 'Success', message: `${files.length} file(s) uploaded`, color: 'green' })
      loadObjects()
      loadBucket()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Upload failed', color: 'red' })
    } finally {
      setUploadProgress(null)
    }
  }

  const handleDelete = async (object: R2Object) => {
    if (!confirm(`Delete "${object.key}"?`)) return
    try {
      await api.r2.deleteObject(name!, object.key)
      notifications.show({ title: 'Success', message: 'Object deleted', color: 'green' })
      loadObjects()
      loadBucket()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Delete failed', color: 'red' })
    }
  }

  const handleDeleteSelected = async () => {
    if (!selectedObjects.length) return
    if (!confirm(`Delete ${selectedObjects.length} object(s)?`)) return
    try {
      for (const key of selectedObjects) {
        await api.r2.deleteObject(name!, key)
      }
      notifications.show({ title: 'Success', message: `${selectedObjects.length} object(s) deleted`, color: 'green' })
      setSelectedObjects([])
      loadObjects()
      loadBucket()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Delete failed', color: 'red' })
    }
  }

  const handleCopyUrl = async (object: R2Object) => {
    const url = `https://${name}.r2.cloudflarestorage.com/${object.key}`
    await navigator.clipboard.writeText(url)
    notifications.show({ title: 'Copied', message: 'URL copied to clipboard', color: 'green' })
  }

  const navigateToFolder = (folderPath: string) => {
    setSearchParams({ prefix: folderPath })
  }

  const navigateUp = () => {
    const parts = prefix.split('/').filter(Boolean)
    parts.pop()
    const newPrefix = parts.length ? parts.join('/') + '/' : ''
    setSearchParams(newPrefix ? { prefix: newPrefix } : {})
  }

  const formatSize = (bytes: number) => {
    if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${bytes} B`
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString()
  }

  const getFileName = (key: string) => {
    const parts = key.split('/')
    return parts[parts.length - 1]
  }

  const getBreadcrumbs = () => {
    const parts = prefix.split('/').filter(Boolean)
    return [
      { label: name!, path: '' },
      ...parts.map((part, i) => ({
        label: part,
        path: parts.slice(0, i + 1).join('/') + '/',
      })),
    ]
  }

  const filteredObjects = objects.filter((obj) =>
    getFileName(obj.key).toLowerCase().includes(searchQuery.toLowerCase())
  )

  if (loading && !bucket) return <LoadingState />
  if (!bucket) return <Text>Bucket not found</Text>

  return (
    <Stack gap="lg">
      <PageHeader
        title={bucket.name}
        breadcrumbs={[{ label: 'R2', path: '/r2' }, { label: bucket.name }]}
        backPath="/r2"
        actions={
          <Group>
            <Button variant="light" leftSection={<IconSettings size={16} />} onClick={() => setSettingsOpen(true)}>
              Settings
            </Button>
            <FileButton onChange={(files) => handleUpload(files ? [files] : [])} accept="*/*">
              {(props) => (
                <Button {...props} leftSection={<IconUpload size={16} />}>
                  Upload
                </Button>
              )}
            </FileButton>
          </Group>
        }
      />

      <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
        <StatCard icon={<Text size="sm" fw={700}>O</Text>} label="Objects" value={(bucket.object_count ?? 0).toLocaleString()} color="orange" />
        <StatCard icon={<Text size="sm" fw={700}>S</Text>} label="Storage" value={formatSize(bucket.storage_size ?? 0)} />
        <StatCard icon={<Text size="sm" fw={700}>L</Text>} label="Location" value={bucket.location || 'Auto'} />
        <StatCard
          icon={bucket.public_access ? <IconWorld size={16} /> : <IconLock size={16} />}
          label="Access"
          value={bucket.public_access ? 'Public' : 'Private'}
          color={bucket.public_access ? 'success' : 'default'}
        />
      </SimpleGrid>

      {uploadProgress !== null && (
        <Paper p="md" radius="md" withBorder>
          <Stack gap="xs">
            <Text size="sm" fw={500}>Uploading...</Text>
            <Progress value={uploadProgress} size="lg" animated />
          </Stack>
        </Paper>
      )}

      <Paper p="md" radius="md" withBorder>
        <Stack gap="md">
          {/* Breadcrumb Navigation */}
          <Group justify="space-between">
            <Group gap="xs">
              {prefix && (
                <ActionIcon variant="light" onClick={navigateUp}>
                  <IconArrowLeft size={16} />
                </ActionIcon>
              )}
              <Breadcrumbs separator={<IconChevronRight size={14} />}>
                {getBreadcrumbs().map((crumb, i) => (
                  <Anchor
                    key={crumb.path}
                    onClick={() => crumb.path === '' ? setSearchParams({}) : navigateToFolder(crumb.path)}
                    fw={i === getBreadcrumbs().length - 1 ? 600 : 400}
                  >
                    {crumb.label}
                  </Anchor>
                ))}
              </Breadcrumbs>
            </Group>
            <Group>
              <TextInput
                placeholder="Search objects..."
                leftSection={<IconSearch size={14} />}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                size="xs"
                w={200}
              />
              <ActionIcon variant="light" onClick={loadObjects}>
                <IconRefresh size={14} />
              </ActionIcon>
            </Group>
          </Group>

          {/* Selected Actions */}
          {selectedObjects.length > 0 && (
            <Group>
              <Badge>{selectedObjects.length} selected</Badge>
              <Button size="xs" variant="light" color="red" leftSection={<IconTrash size={12} />} onClick={handleDeleteSelected}>
                Delete Selected
              </Button>
            </Group>
          )}

          {/* File Browser */}
          <ScrollArea>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th w={40}></Table.Th>
                  <Table.Th>Name</Table.Th>
                  <Table.Th>Size</Table.Th>
                  <Table.Th>Last Modified</Table.Th>
                  <Table.Th w={100}>Actions</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {/* Folders */}
                {folders.map((folder) => (
                  <Table.Tr key={folder} style={{ cursor: 'pointer' }} onClick={() => navigateToFolder(folder)}>
                    <Table.Td></Table.Td>
                    <Table.Td>
                      <Group gap="xs">
                        <IconFolder size={16} color="var(--mantine-color-orange-6)" />
                        <Text size="sm" fw={500}>{folder.replace(prefix, '').replace('/', '')}</Text>
                      </Group>
                    </Table.Td>
                    <Table.Td>-</Table.Td>
                    <Table.Td>-</Table.Td>
                    <Table.Td></Table.Td>
                  </Table.Tr>
                ))}
                {/* Objects */}
                {filteredObjects.map((obj) => (
                  <Table.Tr key={obj.key}>
                    <Table.Td>
                      <input
                        type="checkbox"
                        checked={selectedObjects.includes(obj.key)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedObjects([...selectedObjects, obj.key])
                          } else {
                            setSelectedObjects(selectedObjects.filter((k) => k !== obj.key))
                          }
                        }}
                        onClick={(e) => e.stopPropagation()}
                      />
                    </Table.Td>
                    <Table.Td>
                      <Group gap="xs">
                        <IconFile size={16} color="var(--mantine-color-gray-6)" />
                        <Text size="sm">{getFileName(obj.key)}</Text>
                      </Group>
                    </Table.Td>
                    <Table.Td>{formatSize(obj.size)}</Table.Td>
                    <Table.Td>{formatDate(obj.last_modified)}</Table.Td>
                    <Table.Td>
                      <Group gap={4}>
                        <ActionIcon variant="subtle" size="sm" onClick={() => handleCopyUrl(obj)}>
                          <IconLink size={14} />
                        </ActionIcon>
                        <ActionIcon variant="subtle" size="sm">
                          <IconDownload size={14} />
                        </ActionIcon>
                        <ActionIcon variant="subtle" size="sm" color="red" onClick={() => handleDelete(obj)}>
                          <IconTrash size={14} />
                        </ActionIcon>
                      </Group>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </ScrollArea>

          {folders.length === 0 && filteredObjects.length === 0 && !loading && (
            <Text size="sm" c="dimmed" ta="center" py="xl">
              {searchQuery ? 'No objects match your search' : 'This folder is empty'}
            </Text>
          )}
        </Stack>
      </Paper>

      {/* Settings Modal */}
      <Modal opened={settingsOpen} onClose={() => setSettingsOpen(false)} title="Bucket Settings" size="lg">
        <Tabs defaultValue="general">
          <Tabs.List>
            <Tabs.Tab value="general">General</Tabs.Tab>
            <Tabs.Tab value="cors">CORS</Tabs.Tab>
            <Tabs.Tab value="lifecycle">Lifecycle</Tabs.Tab>
          </Tabs.List>

          <Tabs.Panel value="general" pt="md">
            <Stack gap="md">
              <Switch
                label="Public Access"
                description="Allow public access to objects via custom domain or r2.dev URL"
                checked={bucket.public_access}
              />
              <TextInput
                label="Custom Domain"
                placeholder="cdn.example.com"
                description="Connect a custom domain to serve objects"
              />
              <TextInput
                label="r2.dev URL"
                value={`https://pub-${bucket.name}.r2.dev`}
                readOnly
                description="Development URL for public buckets"
              />
            </Stack>
          </Tabs.Panel>

          <Tabs.Panel value="cors" pt="md">
            <Stack gap="md">
              <Text size="sm" c="dimmed">Configure Cross-Origin Resource Sharing (CORS) rules for this bucket.</Text>
              <Button variant="light" leftSection={<IconPlus size={14} />}>Add CORS Rule</Button>
            </Stack>
          </Tabs.Panel>

          <Tabs.Panel value="lifecycle" pt="md">
            <Stack gap="md">
              <Text size="sm" c="dimmed">Configure lifecycle rules to automatically delete or transition objects.</Text>
              <Button variant="light" leftSection={<IconPlus size={14} />}>Add Lifecycle Rule</Button>
            </Stack>
          </Tabs.Panel>
        </Tabs>

        <Group justify="flex-end" mt="xl">
          <Button variant="default" onClick={() => setSettingsOpen(false)}>Close</Button>
        </Group>
      </Modal>
    </Stack>
  )
}
