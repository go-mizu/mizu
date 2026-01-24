import { useEffect, useState } from 'react'
import { useSearchParams, Link } from 'react-router-dom'
import { Container, Group, Text, Loader, Modal, Image, ActionIcon } from '@mantine/core'
import { IconSettings, IconX } from '@tabler/icons-react'
import { SearchBox } from '../components/SearchBox'
import { searchApi } from '../api/search'
import type { ImageResult } from '../types'

export default function ImagesPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const query = searchParams.get('q') || ''

  const [images, setImages] = useState<ImageResult[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedImage, setSelectedImage] = useState<ImageResult | null>(null)

  useEffect(() => {
    if (!query) return

    const searchImages = async () => {
      setIsLoading(true)
      setError(null)

      try {
        const response = await searchApi.searchImages(query, { per_page: 50 })
        setImages(response.results)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Search failed')
      } finally {
        setIsLoading(false)
      }
    }

    searchImages()
  }, [query])

  const handleSearch = (newQuery: string) => {
    setSearchParams({ q: newQuery })
  }

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="sticky top-0 bg-white border-b z-50">
        <Container size="xl" className="py-3">
          <Group>
            <Link to="/">
              <Text
                size="28px"
                fw={700}
                style={{
                  background: 'linear-gradient(90deg, #4285F4, #EA4335, #FBBC05, #34A853)',
                  WebkitBackgroundClip: 'text',
                  WebkitTextFillColor: 'transparent',
                }}
              >
                Search
              </Text>
            </Link>

            <div className="flex-1 max-w-xl">
              <SearchBox
                initialValue={query}
                size="sm"
                onSearch={handleSearch}
              />
            </div>

            <Link to="/settings">
              <ActionIcon variant="subtle" color="gray" size="lg">
                <IconSettings size={20} />
              </ActionIcon>
            </Link>
          </Group>
        </Container>
      </header>

      {/* Main content */}
      <main>
        <Container size="xl" className="py-4">
          {isLoading ? (
            <div className="flex justify-center py-12">
              <Loader />
            </div>
          ) : error ? (
            <div className="py-12 text-center">
              <Text c="red">{error}</Text>
            </div>
          ) : images.length > 0 ? (
            <div className="image-grid">
              {images.map((image) => (
                <div
                  key={image.id}
                  className="image-grid-item"
                  onClick={() => setSelectedImage(image)}
                >
                  <img
                    src={image.thumbnail_url || image.url}
                    alt={image.title}
                    loading="lazy"
                  />
                </div>
              ))}
            </div>
          ) : query ? (
            <div className="py-12 text-center">
              <Text c="dimmed">No images found</Text>
            </div>
          ) : (
            <div className="py-12 text-center">
              <Text c="dimmed">Search for images</Text>
            </div>
          )}
        </Container>
      </main>

      {/* Image preview modal */}
      <Modal
        opened={!!selectedImage}
        onClose={() => setSelectedImage(null)}
        size="xl"
        withCloseButton={false}
        padding={0}
      >
        {selectedImage && (
          <div className="relative">
            <ActionIcon
              variant="filled"
              color="dark"
              className="absolute top-2 right-2 z-10"
              onClick={() => setSelectedImage(null)}
            >
              <IconX size={16} />
            </ActionIcon>
            <Image
              src={selectedImage.url}
              alt={selectedImage.title}
              fit="contain"
              mah="80vh"
            />
            <div className="p-4 bg-gray-50">
              <Text fw={500}>{selectedImage.title}</Text>
              <Text size="sm" c="dimmed">
                {selectedImage.source_domain} - {selectedImage.width}x{selectedImage.height}
              </Text>
              <a
                href={selectedImage.source_url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-blue-600 hover:underline"
              >
                Visit page
              </a>
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}
