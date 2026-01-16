import { useState, useEffect } from 'react'
import { Button, Modal, Stack, Group, SimpleGrid, Paper, Text, Badge, ActionIcon, FileButton, TextInput, Table, Code, Progress } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconUpload, IconTrash, IconSearch, IconLink, IconSettings, IconPhoto, IconRefresh } from '@tabler/icons-react'
import { PageHeader, StatCard, LoadingState } from '../components/common'
import { api } from '../api/client'
import type { CloudflareImage, ImageVariant } from '../types'

export function Images() {
  const [images, setImages] = useState<CloudflareImage[]>([])
  const [variants, setVariants] = useState<ImageVariant[]>([])
  const [loading, setLoading] = useState(true)
  const [uploadProgress, setUploadProgress] = useState<number | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedImage, setSelectedImage] = useState<CloudflareImage | null>(null)
  const [viewModalOpen, setViewModalOpen] = useState(false)
  const [variantModalOpen, setVariantModalOpen] = useState(false)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [imagesRes, variantsRes] = await Promise.all([
        api.images.list(),
        api.images.listVariants(),
      ])
      if (imagesRes.result) setImages(imagesRes.result.images ?? [])
      if (variantsRes.result) setVariants(variantsRes.result.variants ?? [])
    } catch (error) {
      console.error('Failed to load images:', error)
      notifications.show({ title: 'Error', message: 'Failed to load images', color: 'red' })
      setImages([])
      setVariants([])
    } finally {
      setLoading(false)
    }
  }

  const handleUpload = async (files: File[]) => {
    if (!files.length) return
    setUploadProgress(0)
    try {
      for (let i = 0; i < files.length; i++) {
        await api.images.upload(files[i])
        setUploadProgress(((i + 1) / files.length) * 100)
      }
      notifications.show({ title: 'Success', message: `${files.length} image(s) uploaded`, color: 'green' })
      loadData()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Upload failed', color: 'red' })
    } finally {
      setUploadProgress(null)
    }
  }

  const handleDelete = async (image: CloudflareImage) => {
    if (!confirm(`Delete "${image.filename}"?`)) return
    try {
      await api.images.delete(image.id)
      notifications.show({ title: 'Success', message: 'Image deleted', color: 'green' })
      loadData()
    } catch (error) {
      notifications.show({ title: 'Error', message: 'Delete failed', color: 'red' })
    }
  }

  const handleCopyUrl = async (image: CloudflareImage, variant: string) => {
    const url = `https://imagedelivery.net/account-hash/${image.id}/${variant}`
    await navigator.clipboard.writeText(url)
    notifications.show({ title: 'Copied', message: 'URL copied to clipboard', color: 'green' })
  }

  const viewImage = (image: CloudflareImage) => {
    setSelectedImage(image)
    setViewModalOpen(true)
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

  const filteredImages = images.filter((img) =>
    img.filename.toLowerCase().includes(searchQuery.toLowerCase())
  )

  if (loading) return <LoadingState />

  return (
    <Stack gap="lg">
      <PageHeader
        title="Images"
        subtitle="Store, resize, and optimize images at the edge"
        actions={
          <Group>
            <Button variant="light" leftSection={<IconSettings size={16} />} onClick={() => setVariantModalOpen(true)}>
              Variants
            </Button>
            <FileButton onChange={(files) => handleUpload(files ? (Array.isArray(files) ? files : [files]) : [])} accept="image/*" multiple>
              {(props) => (
                <Button {...props} leftSection={<IconUpload size={16} />}>
                  Upload Images
                </Button>
              )}
            </FileButton>
          </Group>
        }
      />

      <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
        <StatCard icon={<IconPhoto size={16} />} label="Images" value={images.length.toLocaleString()} color="orange" />
        <StatCard icon={<Text size="sm" fw={700}>V</Text>} label="Variants" value={variants.length} />
        <StatCard icon={<Text size="sm" fw={700}>R</Text>} label="Requests" value="45.2K" description="Today" />
        <StatCard icon={<Text size="sm" fw={700}>B</Text>} label="Bandwidth" value="1.2 GB" description="Today" />
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
          <Group justify="space-between">
            <Text size="sm" fw={600}>Image Gallery</Text>
            <Group>
              <TextInput
                placeholder="Search images..."
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

          <SimpleGrid cols={{ base: 2, sm: 3, md: 4, lg: 6 }} spacing="md">
            {filteredImages.map((image) => (
              <Paper
                key={image.id}
                p="xs"
                radius="md"
                withBorder
                style={{ cursor: 'pointer' }}
                onClick={() => viewImage(image)}
              >
                <Stack gap="xs">
                  <Paper
                    radius="sm"
                    bg="dark.6"
                    style={{ aspectRatio: '1', display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden' }}
                  >
                    <IconPhoto size={32} color="var(--mantine-color-dimmed)" />
                  </Paper>
                  <Text size="xs" truncate fw={500}>{image.filename}</Text>
                  <Group justify="space-between">
                    <Text size="xs" c="dimmed">{formatDate(image.uploaded)}</Text>
                    <Group gap={4}>
                      <ActionIcon size="xs" variant="subtle" onClick={(e) => { e.stopPropagation(); handleCopyUrl(image, 'public') }}>
                        <IconLink size={12} />
                      </ActionIcon>
                      <ActionIcon size="xs" variant="subtle" color="red" onClick={(e) => { e.stopPropagation(); handleDelete(image) }}>
                        <IconTrash size={12} />
                      </ActionIcon>
                    </Group>
                  </Group>
                </Stack>
              </Paper>
            ))}
          </SimpleGrid>

          {filteredImages.length === 0 && (
            <Text size="sm" c="dimmed" ta="center" py="xl">
              {searchQuery ? 'No images match your search' : 'No images uploaded yet'}
            </Text>
          )}
        </Stack>
      </Paper>

      {/* View Image Modal */}
      <Modal opened={viewModalOpen} onClose={() => setViewModalOpen(false)} title={selectedImage?.filename} size="lg">
        {selectedImage && (
          <Stack gap="md">
            <Paper radius="md" bg="dark.6" p="xl" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <IconPhoto size={64} color="var(--mantine-color-dimmed)" />
            </Paper>
            <Group gap="xl">
              <Stack gap={2}>
                <Text size="xs" c="dimmed">Dimensions</Text>
                <Text fw={500}>{selectedImage.meta?.width} x {selectedImage.meta?.height}</Text>
              </Stack>
              <Stack gap={2}>
                <Text size="xs" c="dimmed">Uploaded</Text>
                <Text fw={500}>{new Date(selectedImage.uploaded).toLocaleString()}</Text>
              </Stack>
              <Stack gap={2}>
                <Text size="xs" c="dimmed">Variants</Text>
                <Group gap={4}>
                  {selectedImage.variants?.map((v) => (
                    <Badge key={v} size="xs">{v}</Badge>
                  ))}
                </Group>
              </Stack>
            </Group>
            <Stack gap="xs">
              <Text size="sm" fw={600}>Variant URLs</Text>
              {selectedImage.variants?.map((variant) => (
                <Group key={variant} justify="space-between">
                  <Code style={{ flex: 1 }}>/{selectedImage.id}/{variant}</Code>
                  <Button size="xs" variant="light" leftSection={<IconLink size={12} />} onClick={() => handleCopyUrl(selectedImage, variant)}>
                    Copy
                  </Button>
                </Group>
              ))}
            </Stack>
            <Group justify="flex-end">
              <Button variant="default" onClick={() => setViewModalOpen(false)}>Close</Button>
              <Button color="red" leftSection={<IconTrash size={14} />} onClick={() => { setViewModalOpen(false); handleDelete(selectedImage) }}>
                Delete
              </Button>
            </Group>
          </Stack>
        )}
      </Modal>

      {/* Variants Modal */}
      <Modal opened={variantModalOpen} onClose={() => setVariantModalOpen(false)} title="Image Variants" size="lg">
        <Stack gap="md">
          <Text size="sm" c="dimmed">
            Variants define how images are transformed when delivered. Each variant can specify resize options, format, and quality settings.
          </Text>
          <Table striped withTableBorder>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Name</Table.Th>
                <Table.Th>Fit</Table.Th>
                <Table.Th>Dimensions</Table.Th>
                <Table.Th>Signed URLs</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {variants.map((variant) => (
                <Table.Tr key={variant.id}>
                  <Table.Td><Code>{variant.name}</Code></Table.Td>
                  <Table.Td>{variant.options.fit}</Table.Td>
                  <Table.Td>{variant.options.width} x {variant.options.height}</Table.Td>
                  <Table.Td>{variant.never_require_signed_urls ? 'No' : 'Yes'}</Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setVariantModalOpen(false)}>Close</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}
