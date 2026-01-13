import { useState, useEffect } from 'react'
import { Button, Modal, Stack, Group, SimpleGrid, Paper, Text, ActionIcon, TextInput, Code, Progress, Tabs } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconUpload, IconTrash, IconSearch, IconLink, IconPlayerPlay, IconRefresh, IconBroadcast, IconVideo, IconCopy } from '@tabler/icons-react'
import { PageHeader, StatCard, LoadingState, StatusBadge, DataTable, type Column } from '../components/common'
import { api } from '../api/client'
import type { StreamVideo, LiveInput } from '../types'

export function Stream() {
  const [videos, setVideos] = useState<StreamVideo[]>([])
  const [liveInputs, setLiveInputs] = useState<LiveInput[]>([])
  const [loading, setLoading] = useState(true)
  const [uploadProgress, setUploadProgress] = useState<number | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedVideo, setSelectedVideo] = useState<StreamVideo | null>(null)
  const [viewModalOpen, setViewModalOpen] = useState(false)
  const [createLiveModalOpen, setCreateLiveModalOpen] = useState(false)

  const liveForm = useForm({
    initialValues: { name: '' },
    validate: { name: (v) => (!v ? 'Name is required' : null) },
  })

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [videosRes, liveRes] = await Promise.all([
        api.stream.listVideos(),
        api.stream.listLiveInputs(),
      ])
      if (videosRes.result) setVideos(videosRes.result.videos ?? [])
      if (liveRes.result) setLiveInputs(liveRes.result.live_inputs ?? [])
    } catch (error) {
      // Mock data
      setVideos([
        {
          uid: 'vid-1',
          name: 'Product Demo',
          created: new Date(Date.now() - 3600000).toISOString(),
          duration: 245,
          size: 45 * 1024 * 1024,
          status: { state: 'ready' },
          thumbnail: 'https://example.com/thumb1.jpg',
          playback: { hls: 'https://customer-xxx.cloudflarestream.com/vid-1/manifest/video.m3u8' },
        },
        {
          uid: 'vid-2',
          name: 'Getting Started Tutorial',
          created: new Date(Date.now() - 86400000).toISOString(),
          duration: 1234,
          size: 234 * 1024 * 1024,
          status: { state: 'ready' },
          thumbnail: 'https://example.com/thumb2.jpg',
          playback: { hls: 'https://customer-xxx.cloudflarestream.com/vid-2/manifest/video.m3u8' },
        },
        {
          uid: 'vid-3',
          name: 'Conference Recording',
          created: new Date(Date.now() - 172800000).toISOString(),
          duration: 5678,
          size: 1.2 * 1024 * 1024 * 1024,
          status: { state: 'ready' },
          thumbnail: 'https://example.com/thumb3.jpg',
          playback: { hls: 'https://customer-xxx.cloudflarestream.com/vid-3/manifest/video.m3u8' },
        },
        {
          uid: 'vid-4',
          name: 'Marketing Video',
          created: new Date(Date.now() - 7200000).toISOString(),
          duration: 0,
          size: 0,
          status: { state: 'pendingupload' },
          playback: {},
        },
      ])
      setLiveInputs([
        {
          uid: 'live-1',
          name: 'Main Studio',
          created: new Date(Date.now() - 604800000).toISOString(),
          status: 'connected',
          rtmps: { url: 'rtmps://live.cloudflare.com:443/live', streamKey: 'xxx-yyy-zzz' },
        },
        {
          uid: 'live-2',
          name: 'Backup Stream',
          created: new Date(Date.now() - 259200000).toISOString(),
          status: 'disconnected',
          rtmps: { url: 'rtmps://live.cloudflare.com:443/live', streamKey: 'aaa-bbb-ccc' },
        },
      ])
    } finally {
      setLoading(false)
    }
  }

  const handleUpload = async (file: File) => {
    setUploadProgress(0)
    try {
      // Simulate upload progress
      const interval = setInterval(() => {
        setUploadProgress((prev) => {
          if (prev === null || prev >= 95) {
            clearInterval(interval)
            return prev
          }
          return prev + 5
        })
      }, 200)
      await api.stream.upload(file)
      setUploadProgress(100)
      notifications.show({ title: 'Success', message: 'Video uploaded', color: 'green' })
      loadData()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Upload failed', color: 'red' })
    } finally {
      setTimeout(() => setUploadProgress(null), 1000)
    }
  }

  const handleDelete = async (video: StreamVideo) => {
    if (!confirm(`Delete "${video.name}"?`)) return
    try {
      await api.stream.deleteVideo(video.uid)
      notifications.show({ title: 'Success', message: 'Video deleted', color: 'green' })
      loadData()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Delete failed', color: 'red' })
    }
  }

  const handleCreateLiveInput = async (values: typeof liveForm.values) => {
    try {
      await api.stream.createLiveInput(values)
      notifications.show({ title: 'Success', message: 'Live input created', color: 'green' })
      setCreateLiveModalOpen(false)
      liveForm.reset()
      loadData()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Failed to create live input', color: 'red' })
    }
  }

  const handleCopyUrl = async (url: string) => {
    await navigator.clipboard.writeText(url)
    notifications.show({ title: 'Copied', message: 'URL copied to clipboard', color: 'green' })
  }

  const formatDuration = (seconds: number) => {
    if (!seconds) return '-'
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const s = seconds % 60
    if (h > 0) return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
    return `${m}:${s.toString().padStart(2, '0')}`
  }

  const formatSize = (bytes: number) => {
    if (bytes >= 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
    if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${bytes} B`
  }

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const hours = Math.floor(diff / 3600000)
    if (hours < 1) return 'Just now'
    if (hours < 24) return `${hours}h ago`
    const days = Math.floor(hours / 24)
    if (days < 7) return `${days}d ago`
    return date.toLocaleDateString()
  }

  const filteredVideos = videos.filter((v) =>
    v.name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const videoColumns: Column<StreamVideo>[] = [
    { key: 'name', label: 'Name', sortable: true },
    { key: 'duration', label: 'Duration', render: (row) => formatDuration(row.duration) },
    { key: 'size', label: 'Size', render: (row) => formatSize(row.size) },
    { key: 'status.state', label: 'Status', render: (row) => <StatusBadge status={row.status.state} /> },
    { key: 'created', label: 'Uploaded', render: (row) => formatDate(row.created) },
  ]

  const liveColumns: Column<LiveInput>[] = [
    { key: 'name', label: 'Name', sortable: true },
    { key: 'status', label: 'Status', render: (row) => <StatusBadge status={row.status} /> },
    { key: 'created', label: 'Created', render: (row) => formatDate(row.created) },
  ]

  if (loading) return <LoadingState />

  return (
    <Stack gap="lg">
      <PageHeader
        title="Stream"
        subtitle="Video encoding, storage, and delivery"
        actions={
          <Group>
            <Button variant="light" leftSection={<IconBroadcast size={16} />} onClick={() => setCreateLiveModalOpen(true)}>
              Create Live Input
            </Button>
            <Button leftSection={<IconUpload size={16} />} component="label">
              Upload Video
              <input type="file" accept="video/*" hidden onChange={(e) => e.target.files?.[0] && handleUpload(e.target.files[0])} />
            </Button>
          </Group>
        }
      />

      <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
        <StatCard icon={<IconVideo size={16} />} label="Videos" value={videos.length} color="orange" />
        <StatCard icon={<IconBroadcast size={16} />} label="Live Inputs" value={liveInputs.length} />
        <StatCard icon={<Text size="sm" fw={700}>V</Text>} label="Views" value="12.4K" description="This month" />
        <StatCard icon={<Text size="sm" fw={700}>M</Text>} label="Minutes Watched" value="45.2K" description="This month" />
      </SimpleGrid>

      {uploadProgress !== null && (
        <Paper p="md" radius="md" withBorder>
          <Stack gap="xs">
            <Text size="sm" fw={500}>Uploading video...</Text>
            <Progress value={uploadProgress} size="lg" animated />
          </Stack>
        </Paper>
      )}

      <Tabs defaultValue="videos">
        <Tabs.List>
          <Tabs.Tab value="videos" leftSection={<IconVideo size={14} />}>Videos</Tabs.Tab>
          <Tabs.Tab value="live" leftSection={<IconBroadcast size={14} />}>Live Inputs</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="videos" pt="md">
          <Paper p="md" radius="md" withBorder>
            <Stack gap="md">
              <Group justify="space-between">
                <Text size="sm" fw={600}>Video Library</Text>
                <Group>
                  <TextInput
                    placeholder="Search videos..."
                    leftSection={<IconSearch size={14} />}
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    size="xs"
                    w={200}
                  />
                  <ActionIcon variant="light" onClick={loadData}>
                    <IconRefresh size={14} />
                  </ActionIcon>
                </Group>
              </Group>

              <DataTable
                data={filteredVideos}
                columns={videoColumns}
                getRowKey={(row) => row.uid}
                searchable={false}
                onRowClick={(row) => { setSelectedVideo(row); setViewModalOpen(true) }}
                actions={[
                  { label: 'View', icon: <IconPlayerPlay size={14} />, onClick: (row) => { setSelectedVideo(row); setViewModalOpen(true) } },
                  { label: 'Copy URL', icon: <IconLink size={14} />, onClick: (row) => row.playback?.hls && handleCopyUrl(row.playback.hls) },
                  { label: 'Delete', icon: <IconTrash size={14} />, onClick: handleDelete, color: 'red' },
                ]}
                emptyState={{
                  title: 'No videos yet',
                  description: 'Upload your first video to get started',
                }}
              />
            </Stack>
          </Paper>
        </Tabs.Panel>

        <Tabs.Panel value="live" pt="md">
          <Paper p="md" radius="md" withBorder>
            <Stack gap="md">
              <Group justify="space-between">
                <Text size="sm" fw={600}>Live Streaming Inputs</Text>
                <Button size="xs" leftSection={<IconBroadcast size={12} />} onClick={() => setCreateLiveModalOpen(true)}>
                  Create Live Input
                </Button>
              </Group>

              <DataTable
                data={liveInputs}
                columns={liveColumns}
                getRowKey={(row) => row.uid}
                searchable={false}
                actions={[
                  { label: 'Copy RTMPS URL', icon: <IconCopy size={14} />, onClick: (row) => handleCopyUrl(row.rtmps.url) },
                  { label: 'Copy Stream Key', icon: <IconCopy size={14} />, onClick: (row) => handleCopyUrl(row.rtmps.streamKey) },
                  { label: 'Delete', icon: <IconTrash size={14} />, onClick: () => {}, color: 'red' },
                ]}
                emptyState={{
                  title: 'No live inputs yet',
                  description: 'Create a live input to start streaming',
                  action: { label: 'Create Live Input', onClick: () => setCreateLiveModalOpen(true) },
                }}
              />
            </Stack>
          </Paper>
        </Tabs.Panel>
      </Tabs>

      {/* View Video Modal */}
      <Modal opened={viewModalOpen} onClose={() => setViewModalOpen(false)} title={selectedVideo?.name} size="lg">
        {selectedVideo && (
          <Stack gap="md">
            <Paper radius="md" bg="dark.6" p="xl" style={{ aspectRatio: '16/9', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <IconVideo size={64} color="var(--mantine-color-dimmed)" />
            </Paper>
            <Group gap="xl">
              <Stack gap={2}>
                <Text size="xs" c="dimmed">Duration</Text>
                <Text fw={500}>{formatDuration(selectedVideo.duration)}</Text>
              </Stack>
              <Stack gap={2}>
                <Text size="xs" c="dimmed">Size</Text>
                <Text fw={500}>{formatSize(selectedVideo.size)}</Text>
              </Stack>
              <Stack gap={2}>
                <Text size="xs" c="dimmed">Status</Text>
                <StatusBadge status={selectedVideo.status.state} />
              </Stack>
              <Stack gap={2}>
                <Text size="xs" c="dimmed">Uploaded</Text>
                <Text fw={500}>{new Date(selectedVideo.created).toLocaleString()}</Text>
              </Stack>
            </Group>
            {selectedVideo.playback?.hls && (
              <Stack gap="xs">
                <Text size="sm" fw={600}>Playback URL (HLS)</Text>
                <Group>
                  <Code style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis' }}>{selectedVideo.playback.hls}</Code>
                  <Button size="xs" variant="light" leftSection={<IconCopy size={12} />} onClick={() => handleCopyUrl(selectedVideo.playback!.hls!)}>
                    Copy
                  </Button>
                </Group>
              </Stack>
            )}
            <Stack gap="xs">
              <Text size="sm" fw={600}>Embed Code</Text>
              <Code block style={{ fontSize: 11 }}>
{`<iframe
  src="https://customer-xxx.cloudflarestream.com/${selectedVideo.uid}/iframe"
  style="border: none"
  height="720"
  width="1280"
  allow="accelerometer; gyroscope; autoplay; encrypted-media; picture-in-picture;"
  allowfullscreen="true"
></iframe>`}
              </Code>
            </Stack>
            <Group justify="flex-end">
              <Button variant="default" onClick={() => setViewModalOpen(false)}>Close</Button>
              <Button color="red" leftSection={<IconTrash size={14} />} onClick={() => { setViewModalOpen(false); handleDelete(selectedVideo) }}>
                Delete
              </Button>
            </Group>
          </Stack>
        )}
      </Modal>

      {/* Create Live Input Modal */}
      <Modal opened={createLiveModalOpen} onClose={() => setCreateLiveModalOpen(false)} title="Create Live Input" size="md">
        <form onSubmit={liveForm.onSubmit(handleCreateLiveInput)}>
          <Stack gap="md">
            <TextInput
              label="Name"
              placeholder="My Live Stream"
              required
              {...liveForm.getInputProps('name')}
            />
            <Text size="xs" c="dimmed">
              After creation, you'll receive RTMPS credentials to use with your streaming software (OBS, etc.)
            </Text>
            <Group justify="flex-end" mt="md">
              <Button variant="default" onClick={() => setCreateLiveModalOpen(false)}>Cancel</Button>
              <Button type="submit">Create Live Input</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Stack>
  )
}
