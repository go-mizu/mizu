import { useEffect, useState, useCallback, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { X, ChevronLeft, ChevronRight, ChevronDown, ArrowUp } from 'lucide-react'
import { SearchHeader } from '../components/SearchHeader'
import { searchApi } from '../api/search'
import { useInfiniteScroll } from '../hooks/useInfiniteScroll'
import type { ImageResult } from '../types'

const SIZE_OPTIONS = [
  { value: '', label: 'Any size' },
  { value: 'large', label: 'Large' },
  { value: 'medium', label: 'Medium' },
  { value: 'small', label: 'Small' },
]

const COLOR_OPTIONS = [
  { value: '', label: 'Any color' },
  { value: 'color', label: 'Full color' },
  { value: 'mono', label: 'Black & white' },
]

const TYPE_OPTIONS = [
  { value: '', label: 'Any type' },
  { value: 'photo', label: 'Photo' },
  { value: 'clipart', label: 'Clipart' },
  { value: 'gif', label: 'GIF' },
]

export default function ImagesPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const query = searchParams.get('q') || ''

  const [images, setImages] = useState<ImageResult[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null)
  const [page, setPage] = useState(1)
  const [showFilters, setShowFilters] = useState(false)
  const [sizeFilter, setSizeFilter] = useState('')
  const [colorFilter, setColorFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null)
  const [showBackToTop, setShowBackToTop] = useState(false)
  const filterRef = useRef<HTMLDivElement>(null)

  const PER_PAGE = 30

  // Infinite scroll
  const loadMore = useCallback(async () => {
    if (!query || isLoading) return
    const nextPage = page + 1
    try {
      const response = await searchApi.searchImages(query, {
        page: nextPage,
        per_page: PER_PAGE,
        ...(sizeFilter && { size: sizeFilter }),
        ...(colorFilter && { color: colorFilter }),
        ...(typeFilter && { type: typeFilter }),
      })
      if (response.results.length === 0) {
        setHasMore(false)
      } else {
        setImages(prev => [...prev, ...response.results])
        setPage(nextPage)
      }
    } catch {
      setHasMore(false)
    }
  }, [query, page, isLoading])

  const { isLoading: scrollLoading, hasMore, setHasMore, reset } = useInfiniteScroll(loadMore, {
    threshold: 300,
    enabled: images.length > 0 && !isLoading,
  })

  // Initial search
  useEffect(() => {
    if (!query) {
      setImages([])
      return
    }

    const searchImages = async () => {
      setIsLoading(true)
      setError(null)
      reset()
      setPage(1)

      try {
        const response = await searchApi.searchImages(query, {
          per_page: PER_PAGE,
          ...(sizeFilter && { size: sizeFilter }),
          ...(colorFilter && { color: colorFilter }),
          ...(typeFilter && { type: typeFilter }),
        })
        setImages(response.results || [])
        setHasMore((response.results?.length || 0) >= PER_PAGE)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Search failed')
      } finally {
        setIsLoading(false)
      }
    }

    searchImages()
  }, [query, sizeFilter, colorFilter, typeFilter])

  // Scroll position tracking for back to top button
  useEffect(() => {
    const handleScroll = () => {
      setShowBackToTop(window.scrollY > 500)
    }
    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  // Keyboard navigation for lightbox
  useEffect(() => {
    if (selectedIndex === null) return

    const handleKeyDown = (e: KeyboardEvent) => {
      switch (e.key) {
        case 'Escape':
          setSelectedIndex(null)
          break
        case 'ArrowLeft':
          setSelectedIndex(prev => (prev !== null && prev > 0 ? prev - 1 : prev))
          break
        case 'ArrowRight':
          setSelectedIndex(prev => (prev !== null && prev < images.length - 1 ? prev + 1 : prev))
          break
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [selectedIndex, images.length])

  // Close filter dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (filterRef.current && !filterRef.current.contains(e.target as Node)) {
        setShowFilters(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleSearch = (newQuery: string) => {
    setSearchParams({ q: newQuery })
  }

  const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const selectedImage = selectedIndex !== null ? images[selectedIndex] : null

  return (
    <div className="min-h-screen bg-white">
      <SearchHeader
        query={query}
        activeTab="images"
        onSearch={handleSearch}
        tabsRight={
          <div className="relative" ref={filterRef}>
            <button
              type="button"
              className="flex items-center gap-1 px-3 py-1.5 text-sm text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
              onClick={() => setShowFilters(!showFilters)}
            >
              Filters
              <ChevronDown size={16} />
            </button>
            {showFilters && (
              <div className="absolute right-0 mt-2 w-64 bg-white rounded-lg shadow-lg border border-[#e8eaed] p-4 z-50">
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-[#202124] mb-2">Size</label>
                    <select
                      value={sizeFilter}
                      onChange={(e) => setSizeFilter(e.target.value)}
                      className="w-full px-3 py-2 border border-[#dadce0] rounded-lg text-sm focus:outline-none focus:border-[#1a73e8]"
                    >
                      {SIZE_OPTIONS.map(opt => (
                        <option key={opt.value} value={opt.value}>{opt.label}</option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-[#202124] mb-2">Color</label>
                    <select
                      value={colorFilter}
                      onChange={(e) => setColorFilter(e.target.value)}
                      className="w-full px-3 py-2 border border-[#dadce0] rounded-lg text-sm focus:outline-none focus:border-[#1a73e8]"
                    >
                      {COLOR_OPTIONS.map(opt => (
                        <option key={opt.value} value={opt.value}>{opt.label}</option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-[#202124] mb-2">Type</label>
                    <select
                      value={typeFilter}
                      onChange={(e) => setTypeFilter(e.target.value)}
                      className="w-full px-3 py-2 border border-[#dadce0] rounded-lg text-sm focus:outline-none focus:border-[#1a73e8]"
                    >
                      {TYPE_OPTIONS.map(opt => (
                        <option key={opt.value} value={opt.value}>{opt.label}</option>
                      ))}
                    </select>
                  </div>
                </div>
              </div>
            )}
          </div>
        }
      />

      {/* Main content */}
      <main>
        <div className="mx-auto px-4 py-4">
          {isLoading && images.length === 0 ? (
            <div className="flex justify-center py-12">
              <div className="w-8 h-8 border-4 border-[#1a73e8] border-t-transparent rounded-full animate-spin" />
            </div>
          ) : error ? (
            <div className="py-12 text-center">
              <p className="text-red-600">{error}</p>
            </div>
          ) : images.length > 0 ? (
            <>
              {/* Masonry Grid */}
              <div className="columns-2 sm:columns-3 md:columns-4 lg:columns-5 gap-4">
                {images.map((image, index) => (
                  <div
                    key={`${image.id}-${index}`}
                    className="break-inside-avoid mb-4 relative group cursor-pointer"
                    onClick={() => setSelectedIndex(index)}
                    onMouseEnter={() => setHoveredIndex(index)}
                    onMouseLeave={() => setHoveredIndex(null)}
                  >
                    <div className="relative overflow-hidden rounded-lg bg-[#f1f3f4]">
                      <img
                        src={image.thumbnail_url || image.url}
                        alt={image.title}
                        loading="lazy"
                        className="w-full h-auto transition-transform duration-200 group-hover:scale-105"
                        onError={(e) => {
                          const parent = e.currentTarget.parentElement?.parentElement
                          if (parent) parent.style.display = 'none'
                        }}
                      />
                      {/* Hover overlay */}
                      {hoveredIndex === index && (
                        <div className="absolute inset-0 bg-black/40 flex items-end p-3 transition-opacity">
                          <div className="text-white">
                            <p className="text-sm font-medium line-clamp-2">{image.title}</p>
                            <p className="text-xs opacity-80 mt-1">
                              {image.source_domain}
                              {image.width && image.height && ` • ${image.width}×${image.height}`}
                            </p>
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>

              {/* Loading more indicator */}
              {scrollLoading && (
                <div className="flex justify-center py-8">
                  <div className="w-6 h-6 border-2 border-[#1a73e8] border-t-transparent rounded-full animate-spin" />
                </div>
              )}

              {/* End of results */}
              {!hasMore && images.length > 0 && (
                <div className="text-center py-8 text-[#70757a]">
                  No more images
                </div>
              )}
            </>
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

      {/* Back to top button */}
      {showBackToTop && (
        <button
          type="button"
          onClick={scrollToTop}
          className="fixed bottom-6 right-6 p-3 bg-[#1a73e8] text-white rounded-full shadow-lg hover:bg-[#1557b0] transition-colors z-40"
        >
          <ArrowUp size={20} />
        </button>
      )}

      {/* Lightbox */}
      {selectedImage && selectedIndex !== null && (
        <div
          className="fixed inset-0 bg-black/90 z-50 flex items-center justify-center"
          onClick={() => setSelectedIndex(null)}
        >
          {/* Close button */}
          <button
            type="button"
            onClick={() => setSelectedIndex(null)}
            className="absolute top-4 right-4 p-2 text-white/80 hover:text-white transition-colors"
          >
            <X size={24} />
          </button>

          {/* Previous button */}
          {selectedIndex > 0 && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                setSelectedIndex(selectedIndex - 1)
              }}
              className="absolute left-4 p-2 text-white/80 hover:text-white transition-colors"
            >
              <ChevronLeft size={32} />
            </button>
          )}

          {/* Next button */}
          {selectedIndex < images.length - 1 && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                setSelectedIndex(selectedIndex + 1)
              }}
              className="absolute right-4 p-2 text-white/80 hover:text-white transition-colors"
            >
              <ChevronRight size={32} />
            </button>
          )}

          {/* Image container */}
          <div
            className="max-w-[90vw] max-h-[90vh] flex flex-col"
            onClick={(e) => e.stopPropagation()}
          >
            <img
              src={selectedImage.url}
              alt={selectedImage.title}
              className="max-h-[80vh] w-auto object-contain rounded-lg"
              onError={(e) => {
                e.currentTarget.src = selectedImage.thumbnail_url || ''
              }}
            />
            <div className="mt-4 text-white text-center">
              <p className="font-medium">{selectedImage.title}</p>
              <div className="text-sm text-white/70 mt-1 flex items-center justify-center gap-4">
                <span>{selectedImage.source_domain}</span>
                {selectedImage.width && selectedImage.height && (
                  <span>{selectedImage.width} × {selectedImage.height}</span>
                )}
                {selectedImage.format && (
                  <span className="uppercase">{selectedImage.format}</span>
                )}
              </div>
              <a
                href={selectedImage.source_url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-block mt-3 px-4 py-2 bg-[#1a73e8] text-white rounded-full text-sm hover:bg-[#1557b0] transition-colors"
              >
                Visit page
              </a>
            </div>
          </div>

          {/* Image counter */}
          <div className="absolute bottom-4 left-1/2 -translate-x-1/2 text-white/70 text-sm">
            {selectedIndex + 1} / {images.length}
          </div>
        </div>
      )}
    </div>
  )
}
