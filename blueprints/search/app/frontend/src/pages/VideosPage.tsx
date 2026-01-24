import { useEffect, useState, useCallback, useRef } from 'react'
import { useSearchParams, Link, useNavigate } from 'react-router-dom'
import { Settings, Image, Video, Newspaper, ChevronDown, ArrowUp, Grid, List, Play } from 'lucide-react'
import { SearchBox } from '../components/SearchBox'
import { searchApi } from '../api/search'
import { useInfiniteScroll } from '../hooks/useInfiniteScroll'
import type { VideoResult } from '../types'

const DURATION_OPTIONS = [
  { value: '', label: 'Any duration' },
  { value: 'short', label: 'Short (< 4 min)' },
  { value: 'medium', label: 'Medium (4-20 min)' },
  { value: 'long', label: 'Long (> 20 min)' },
]

const SORT_OPTIONS = [
  { value: '', label: 'Relevance' },
  { value: 'date', label: 'Date' },
  { value: 'views', label: 'Views' },
]

function formatDuration(seconds: number | undefined): string {
  if (!seconds) return ''
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  if (mins >= 60) {
    const hours = Math.floor(mins / 60)
    const remainingMins = mins % 60
    return `${hours}:${remainingMins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

function formatViews(views: number | undefined): string {
  if (!views) return ''
  if (views >= 1000000) return `${(views / 1000000).toFixed(1)}M views`
  if (views >= 1000) return `${(views / 1000).toFixed(1)}K views`
  return `${views} views`
}

function formatDate(dateString: string | undefined): string {
  if (!dateString) return ''
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  if (diffDays < 1) return 'Today'
  if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`
  if (diffDays < 30) return `${Math.floor(diffDays / 7)} week${Math.floor(diffDays / 7) > 1 ? 's' : ''} ago`
  if (diffDays < 365) return `${Math.floor(diffDays / 30)} month${Math.floor(diffDays / 30) > 1 ? 's' : ''} ago`
  return `${Math.floor(diffDays / 365)} year${Math.floor(diffDays / 365) > 1 ? 's' : ''} ago`
}

export default function VideosPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()
  const query = searchParams.get('q') || ''

  const [videos, setVideos] = useState<VideoResult[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [viewMode, setViewMode] = useState<'list' | 'grid'>(() => {
    return (localStorage.getItem('videos_view_mode') as 'list' | 'grid') || 'list'
  })
  const [showFilters, setShowFilters] = useState(false)
  const [durationFilter, setDurationFilter] = useState('')
  const [sortFilter, setSortFilter] = useState('')
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null)
  const [showBackToTop, setShowBackToTop] = useState(false)
  const filterRef = useRef<HTMLDivElement>(null)

  const PER_PAGE = 30

  // Persist view mode
  useEffect(() => {
    localStorage.setItem('videos_view_mode', viewMode)
  }, [viewMode])

  // Infinite scroll
  const loadMore = useCallback(async () => {
    if (!query || isLoading) return
    const nextPage = page + 1
    try {
      const response = await searchApi.searchVideos(query, { page: nextPage, per_page: PER_PAGE })
      if (response.results.length === 0) {
        setHasMore(false)
      } else {
        setVideos(prev => [...prev, ...response.results])
        setPage(nextPage)
      }
    } catch {
      setHasMore(false)
    }
  }, [query, page, isLoading])

  const { isLoading: scrollLoading, hasMore, setHasMore, reset } = useInfiniteScroll(loadMore, {
    threshold: 300,
    enabled: videos.length > 0 && !isLoading,
  })

  // Initial search
  useEffect(() => {
    if (!query) {
      setVideos([])
      return
    }

    const searchVideos = async () => {
      setIsLoading(true)
      setError(null)
      reset()
      setPage(1)

      try {
        const response = await searchApi.searchVideos(query, { per_page: PER_PAGE })
        setVideos(response.results || [])
        setHasMore((response.results?.length || 0) >= PER_PAGE)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Search failed')
      } finally {
        setIsLoading(false)
      }
    }

    searchVideos()
  }, [query])

  // Scroll position tracking
  useEffect(() => {
    const handleScroll = () => {
      setShowBackToTop(window.scrollY > 500)
    }
    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  // Close filter dropdown
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

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="sticky top-0 bg-white z-40 border-b border-[#e8eaed]">
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

          {/* Tabs and Controls */}
          <div className="flex items-center justify-between mt-2">
            <div className="search-tabs" style={{ paddingLeft: 0 }}>
              <button
                type="button"
                className="search-tab"
                onClick={() => navigate(`/search?q=${encodeURIComponent(query)}`)}
              >
                All
              </button>
              <button
                type="button"
                className="search-tab"
                onClick={() => navigate(`/images?q=${encodeURIComponent(query)}`)}
              >
                <Image size={16} />
                Images
              </button>
              <button
                type="button"
                className="search-tab active"
              >
                <Video size={16} />
                Videos
              </button>
              <button
                type="button"
                className="search-tab"
                onClick={() => navigate(`/news?q=${encodeURIComponent(query)}`)}
              >
                <Newspaper size={16} />
                News
              </button>
            </div>

            <div className="flex items-center gap-2">
              {/* View toggle */}
              <div className="flex items-center border border-[#dadce0] rounded-full overflow-hidden">
                <button
                  type="button"
                  onClick={() => setViewMode('list')}
                  className={`p-2 transition-colors ${viewMode === 'list' ? 'bg-[#e8f0fe] text-[#1a73e8]' : 'text-[#5f6368] hover:bg-[#f1f3f4]'}`}
                  title="List view"
                >
                  <List size={18} />
                </button>
                <button
                  type="button"
                  onClick={() => setViewMode('grid')}
                  className={`p-2 transition-colors ${viewMode === 'grid' ? 'bg-[#e8f0fe] text-[#1a73e8]' : 'text-[#5f6368] hover:bg-[#f1f3f4]'}`}
                  title="Grid view"
                >
                  <Grid size={18} />
                </button>
              </div>

              {/* Filters */}
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
                        <label className="block text-sm font-medium text-[#202124] mb-2">Duration</label>
                        <select
                          value={durationFilter}
                          onChange={(e) => setDurationFilter(e.target.value)}
                          className="w-full px-3 py-2 border border-[#dadce0] rounded-lg text-sm focus:outline-none focus:border-[#1a73e8]"
                        >
                          {DURATION_OPTIONS.map(opt => (
                            <option key={opt.value} value={opt.value}>{opt.label}</option>
                          ))}
                        </select>
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-[#202124] mb-2">Sort by</label>
                        <select
                          value={sortFilter}
                          onChange={(e) => setSortFilter(e.target.value)}
                          className="w-full px-3 py-2 border border-[#dadce0] rounded-lg text-sm focus:outline-none focus:border-[#1a73e8]"
                        >
                          {SORT_OPTIONS.map(opt => (
                            <option key={opt.value} value={opt.value}>{opt.label}</option>
                          ))}
                        </select>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main>
        <div className="max-w-7xl mx-auto px-4 py-4">
          {isLoading && videos.length === 0 ? (
            <div className="flex justify-center py-12">
              <div className="w-8 h-8 border-4 border-[#1a73e8] border-t-transparent rounded-full animate-spin" />
            </div>
          ) : error ? (
            <div className="py-12 text-center">
              <p className="text-red-600">{error}</p>
            </div>
          ) : videos.length > 0 ? (
            <>
              {viewMode === 'list' ? (
                /* List View */
                <div className="space-y-6">
                  {videos.map((video, index) => (
                    <a
                      key={`${video.id}-${index}`}
                      href={video.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex gap-4 p-3 rounded-lg hover:bg-[#f8f9fa] transition-colors"
                      onMouseEnter={() => setHoveredIndex(index)}
                      onMouseLeave={() => setHoveredIndex(null)}
                    >
                      <div className="relative flex-shrink-0 w-80 h-44 bg-[#f1f3f4] rounded-lg overflow-hidden">
                        {video.thumbnail_url ? (
                          <img
                            src={video.thumbnail_url}
                            alt={video.title}
                            className="w-full h-full object-cover"
                            onError={(e) => {
                              e.currentTarget.style.display = 'none'
                            }}
                          />
                        ) : (
                          <div className="w-full h-full flex items-center justify-center">
                            <Video size={48} className="text-[#9aa0a6]" />
                          </div>
                        )}
                        {/* Play icon on hover */}
                        {hoveredIndex === index && (
                          <div className="absolute inset-0 flex items-center justify-center bg-black/30 transition-opacity">
                            <div className="w-16 h-16 rounded-full bg-black/70 flex items-center justify-center">
                              <Play size={28} className="text-white ml-1" fill="white" />
                            </div>
                          </div>
                        )}
                        {(video.duration || video.duration_seconds) && (
                          <span className="absolute bottom-2 right-2 bg-black/80 text-white text-xs px-1.5 py-0.5 rounded">
                            {video.duration || formatDuration(video.duration_seconds)}
                          </span>
                        )}
                      </div>
                      <div className="flex-1 min-w-0">
                        <h3 className="text-[#1a0dab] text-lg font-medium line-clamp-2 hover:underline">
                          {video.title}
                        </h3>
                        <div className="text-sm text-[#70757a] mt-2 flex items-center gap-2">
                          <span>{video.channel || video.source_domain || video.engine}</span>
                          {video.views && (
                            <>
                              <span>•</span>
                              <span>{formatViews(video.views)}</span>
                            </>
                          )}
                          {video.published_at && (
                            <>
                              <span>•</span>
                              <span>{formatDate(video.published_at)}</span>
                            </>
                          )}
                        </div>
                        {video.description && (
                          <p className="text-sm text-[#4d5156] mt-3 line-clamp-2">
                            {video.description}
                          </p>
                        )}
                      </div>
                    </a>
                  ))}
                </div>
              ) : (
                /* Grid View */
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                  {videos.map((video, index) => (
                    <a
                      key={`${video.id}-${index}`}
                      href={video.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="block rounded-lg overflow-hidden hover:shadow-lg transition-shadow bg-white border border-[#e8eaed]"
                      onMouseEnter={() => setHoveredIndex(index)}
                      onMouseLeave={() => setHoveredIndex(null)}
                    >
                      <div className="relative aspect-video bg-[#f1f3f4]">
                        {video.thumbnail_url ? (
                          <img
                            src={video.thumbnail_url}
                            alt={video.title}
                            className="w-full h-full object-cover"
                            onError={(e) => {
                              e.currentTarget.style.display = 'none'
                            }}
                          />
                        ) : (
                          <div className="w-full h-full flex items-center justify-center">
                            <Video size={48} className="text-[#9aa0a6]" />
                          </div>
                        )}
                        {/* Play icon on hover */}
                        {hoveredIndex === index && (
                          <div className="absolute inset-0 flex items-center justify-center bg-black/30 transition-opacity">
                            <div className="w-14 h-14 rounded-full bg-black/70 flex items-center justify-center">
                              <Play size={24} className="text-white ml-0.5" fill="white" />
                            </div>
                          </div>
                        )}
                        {(video.duration || video.duration_seconds) && (
                          <span className="absolute bottom-2 right-2 bg-black/80 text-white text-xs px-1.5 py-0.5 rounded">
                            {video.duration || formatDuration(video.duration_seconds)}
                          </span>
                        )}
                      </div>
                      <div className="p-3">
                        <h3 className="text-[#202124] text-sm font-medium line-clamp-2">
                          {video.title}
                        </h3>
                        <div className="text-xs text-[#70757a] mt-2">
                          <p>{video.channel || video.source_domain || video.engine}</p>
                          <div className="flex items-center gap-1 mt-1">
                            {video.views && <span>{formatViews(video.views)}</span>}
                            {video.views && video.published_at && <span>•</span>}
                            {video.published_at && <span>{formatDate(video.published_at)}</span>}
                          </div>
                        </div>
                      </div>
                    </a>
                  ))}
                </div>
              )}

              {/* Loading more indicator */}
              {scrollLoading && (
                <div className="flex justify-center py-8">
                  <div className="w-6 h-6 border-3 border-[#1a73e8] border-t-transparent rounded-full animate-spin" />
                </div>
              )}

              {/* End of results */}
              {!hasMore && videos.length > 0 && (
                <div className="text-center py-8 text-[#70757a]">
                  No more videos
                </div>
              )}
            </>
          ) : query ? (
            <div className="py-12 text-center">
              <p className="text-[#70757a]">No videos found</p>
            </div>
          ) : (
            <div className="py-12 text-center">
              <p className="text-[#70757a]">Search for videos</p>
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
    </div>
  )
}
