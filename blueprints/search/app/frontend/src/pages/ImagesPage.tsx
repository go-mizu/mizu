import { useEffect, useState } from 'react'
import { useSearchParams, Link } from 'react-router-dom'
import { Settings, X } from 'lucide-react'
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
        setImages(response.results || [])
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
      <header className="sticky top-0 bg-white border-b border-[#dadce0] z-50">
        <div className="max-w-7xl mx-auto px-4 py-3">
          <div className="flex items-center gap-6">
            <Link to="/">
              <span
                className="text-3xl font-bold"
                style={{
                  background: 'linear-gradient(90deg, #4285F4, #EA4335, #FBBC05, #34A853)',
                  WebkitBackgroundClip: 'text',
                  WebkitTextFillColor: 'transparent',
                }}
              >
                Search
              </span>
            </Link>

            <div className="flex-1 max-w-xl">
              <SearchBox
                initialValue={query}
                size="sm"
                onSearch={handleSearch}
              />
            </div>

            <Link
              to="/settings"
              className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
            >
              <Settings size={20} />
            </Link>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main>
        <div className="max-w-7xl mx-auto px-4 py-4">
          {isLoading ? (
            <div className="flex justify-center py-12">
              <div className="w-8 h-8 border-4 border-[#1a73e8] border-t-transparent rounded-full animate-spin" />
            </div>
          ) : error ? (
            <div className="py-12 text-center">
              <p className="text-red-600">{error}</p>
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
              <p className="text-[#70757a]">No images found</p>
            </div>
          ) : (
            <div className="py-12 text-center">
              <p className="text-[#70757a]">Search for images</p>
            </div>
          )}
        </div>
      </main>

      {/* Image preview modal */}
      {selectedImage && (
        <div
          className="fixed inset-0 bg-black/80 z-50 flex items-center justify-center p-4"
          onClick={() => setSelectedImage(null)}
        >
          <div
            className="bg-white rounded-lg max-w-4xl max-h-[90vh] overflow-hidden"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="relative">
              <button
                type="button"
                onClick={() => setSelectedImage(null)}
                className="absolute top-2 right-2 z-10 p-2 bg-black/50 text-white rounded-full hover:bg-black/70 transition-colors"
              >
                <X size={16} />
              </button>
              <img
                src={selectedImage.url}
                alt={selectedImage.title}
                className="max-h-[70vh] w-auto mx-auto"
              />
              <div className="p-4 bg-[#f8f9fa]">
                <p className="font-medium text-[#202124]">{selectedImage.title}</p>
                <p className="text-sm text-[#70757a]">
                  {selectedImage.source_domain} - {selectedImage.width}x{selectedImage.height}
                </p>
                <a
                  href={selectedImage.source_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-[#1a73e8] hover:underline"
                >
                  Visit page
                </a>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
