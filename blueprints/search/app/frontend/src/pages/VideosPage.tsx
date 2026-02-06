import { useEffect, useState, useCallback, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Video, ChevronDown, ArrowUp, Grid, List, Play, X, Check, ChevronLeft, ChevronRight, ExternalLink, Volume2, VolumeX } from 'lucide-react'
import { SearchHeader } from '../components/SearchHeader'
import { searchApi } from '../api/search'
import { useInfiniteScroll } from '../hooks/useInfiniteScroll'
import type { VideoResult } from '../types'
import type { VideoSearchOptions } from '../api/search'

// Helper to get autoplay embed URL
function getAutoplayEmbedUrl(embedUrl: string | undefined): string | null {
  if (!embedUrl) return null

  try {
    const url = new URL(embedUrl)

    // YouTube
    if (url.hostname.includes('youtube.com') || url.hostname.includes('youtube-nocookie.com')) {
      url.searchParams.set('autoplay', '1')
      url.searchParams.set('mute', '1')
      url.searchParams.set('controls', '0')
      url.searchParams.set('modestbranding', '1')
      url.searchParams.set('rel', '0')
      return url.toString()
    }

    // Vimeo
    if (url.hostname.includes('vimeo.com') || url.hostname.includes('player.vimeo.com')) {
      url.searchParams.set('autoplay', '1')
      url.searchParams.set('muted', '1')
      url.searchParams.set('controls', '0')
      url.searchParams.set('background', '1')
      return url.toString()
    }

    // Dailymotion
    if (url.hostname.includes('dailymotion.com')) {
      url.searchParams.set('autoplay', '1')
      url.searchParams.set('mute', '1')
      url.searchParams.set('controls', '0')
      return url.toString()
    }

    // Generic fallback - try adding autoplay params
    url.searchParams.set('autoplay', '1')
    url.searchParams.set('mute', '1')
    return url.toString()
  } catch {
    return null
  }
}

// Video Preview Component for hover autoplay
interface VideoPreviewProps {
  video: VideoResult
  isVisible: boolean
  onMuteToggle?: (muted: boolean) => void
  isMuted?: boolean
}

function VideoPreview({ video, isVisible, onMuteToggle, isMuted = true }: VideoPreviewProps) {
  const [showPreview, setShowPreview] = useState(false)
  const [iframeLoaded, setIframeLoaded] = useState(false)
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const iframeRef = useRef<HTMLIFrameElement>(null)

  const autoplayUrl = getAutoplayEmbedUrl(video.embed_url)
  const hasPreview = !!autoplayUrl

  // Debounced show preview
  useEffect(() => {
    if (isVisible && hasPreview) {
      timeoutRef.current = setTimeout(() => {
        setShowPreview(true)
      }, 400) // 400ms delay before showing preview
    } else {
      // Clear timeout and hide immediately on mouse leave
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
        timeoutRef.current = null
      }
      setShowPreview(false)
      setIframeLoaded(false)
    }

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [isVisible, hasPreview])

  // Don't render anything if no embed URL or not visible
  if (!hasPreview || !showPreview) {
    return null
  }

  // Build URL with current mute state
  const currentUrl = autoplayUrl ? (() => {
    try {
      const url = new URL(autoplayUrl)
      if (url.hostname.includes('youtube')) {
        url.searchParams.set('mute', isMuted ? '1' : '0')
      } else if (url.hostname.includes('vimeo')) {
        url.searchParams.set('muted', isMuted ? '1' : '0')
      } else if (url.hostname.includes('dailymotion')) {
        url.searchParams.set('mute', isMuted ? '1' : '0')
      }
      return url.toString()
    } catch {
      return autoplayUrl
    }
  })() : autoplayUrl

  return (
    <div className="absolute inset-0 z-10">
      {/* Loading state while iframe loads */}
      {!iframeLoaded && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/50">
          <div className="w-8 h-8 border-2 border-white border-t-transparent rounded-full animate-spin" />
        </div>
      )}

      <iframe
        ref={iframeRef}
        src={currentUrl || undefined}
        className={`w-full h-full ${iframeLoaded ? 'opacity-100' : 'opacity-0'}`}
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
        title={`Preview: ${video.title}`}
        onLoad={() => setIframeLoaded(true)}
      />

      {/* Mute/Unmute toggle button */}
      {iframeLoaded && onMuteToggle && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            onMuteToggle(!isMuted)
          }}
          className="absolute bottom-2 left-2 p-1.5 bg-black/70 hover:bg-black/90 rounded-full text-white transition-colors z-20"
          aria-label={isMuted ? 'Unmute' : 'Mute'}
        >
          {isMuted ? <VolumeX size={16} /> : <Volume2 size={16} />}
        </button>
      )}
    </div>
  )
}

// Filter options configuration
const DURATION_OPTIONS = [
  { value: '', label: 'Any duration' },
  { value: 'short', label: 'Short (< 4 min)' },
  { value: 'medium', label: 'Medium (4-20 min)' },
  { value: 'long', label: 'Long (> 20 min)' },
]

const TIME_OPTIONS = [
  { value: '', label: 'Any time' },
  { value: 'hour', label: 'Past hour' },
  { value: 'day', label: 'Past 24 hours' },
  { value: 'week', label: 'Past week' },
  { value: 'month', label: 'Past month' },
  { value: 'year', label: 'Past year' },
]

const SOURCE_OPTIONS = [
  { value: '', label: 'All sources' },
  { value: 'youtube', label: 'YouTube' },
  { value: 'vimeo', label: 'Vimeo' },
  { value: 'dailymotion', label: 'Dailymotion' },
  { value: 'google_videos', label: 'Google' },
  { value: 'bing_videos', label: 'Bing' },
  { value: 'peertube', label: 'PeerTube' },
]

const QUALITY_OPTIONS = [
  { value: '', label: 'Any quality' },
  { value: 'hd', label: 'HD' },
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

// Filter chip dropdown component
interface FilterChipProps {
  label: string
  value: string
  options: { value: string; label: string }[]
  onChange: (value: string) => void
  isActive: boolean
}

function FilterChip({ label, value, options, onChange, isActive }: FilterChipProps) {
  const [isOpen, setIsOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const selectedOption = options.find(opt => opt.value === value)
  const displayLabel = isActive && selectedOption ? selectedOption.label : label

  return (
    <div className="relative flex-shrink-0" ref={dropdownRef}>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className={`
          flex items-center gap-1 px-3 py-1.5 text-sm rounded-full border transition-colors whitespace-nowrap
          ${isActive
            ? 'bg-[#e8f0fe] border-[#e8f0fe] text-[#1a73e8]'
            : 'bg-white border-[#dadce0] text-[#3c4043] hover:bg-[#f1f3f4]'
          }
        `}
      >
        {displayLabel}
        <ChevronDown size={14} className={`transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {isOpen && (
        <div className="absolute top-full left-0 mt-1 bg-white rounded-lg shadow-lg border border-[#e8eaed] py-1 z-50 min-w-[160px]">
          {options.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => {
                onChange(option.value)
                setIsOpen(false)
              }}
              className={`
                w-full flex items-center gap-2 px-3 py-2 text-sm text-left transition-colors
                ${option.value === value ? 'bg-[#e8f0fe] text-[#1a73e8]' : 'text-[#3c4043] hover:bg-[#f1f3f4]'}
              `}
            >
              {option.value === value && <Check size={14} />}
              <span className={option.value === value ? '' : 'ml-[22px]'}>{option.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

// Toggle chip for boolean filters (like CC)
interface ToggleChipProps {
  label: string
  isActive: boolean
  onToggle: () => void
}

function ToggleChip({ label, isActive, onToggle }: ToggleChipProps) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className={`
        flex items-center gap-1 px-3 py-1.5 text-sm rounded-full border transition-colors whitespace-nowrap flex-shrink-0
        ${isActive
          ? 'bg-[#e8f0fe] border-[#e8f0fe] text-[#1a73e8]'
          : 'bg-white border-[#dadce0] text-[#3c4043] hover:bg-[#f1f3f4]'
        }
      `}
    >
      {isActive && <Check size={14} />}
      {label}
    </button>
  )
}

// Video Player Modal Component
interface VideoPlayerProps {
  video: VideoResult
  onClose: () => void
  onPrevious: () => void
  onNext: () => void
  hasPrevious: boolean
  hasNext: boolean
}

function getSourceName(video: VideoResult): string {
  if (video.source_domain) {
    // Extract readable name from domain
    const domain = video.source_domain.replace(/^www\./, '')
    if (domain.includes('youtube')) return 'YouTube'
    if (domain.includes('vimeo')) return 'Vimeo'
    if (domain.includes('dailymotion')) return 'Dailymotion'
    if (domain.includes('twitch')) return 'Twitch'
    if (domain.includes('peertube')) return 'PeerTube'
    return domain
  }
  if (video.engine) {
    return video.engine.charAt(0).toUpperCase() + video.engine.slice(1)
  }
  return 'source'
}

function VideoPlayer({ video, onClose, onPrevious, onNext, hasPrevious, hasNext }: VideoPlayerProps) {
  const modalRef = useRef<HTMLDivElement>(null)

  // Handle keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      } else if (e.key === 'ArrowLeft' && hasPrevious) {
        onPrevious()
      } else if (e.key === 'ArrowRight' && hasNext) {
        onNext()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    // Prevent body scroll when modal is open
    document.body.style.overflow = 'hidden'

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = ''
    }
  }, [onClose, onPrevious, onNext, hasPrevious, hasNext])

  // Close modal when clicking on backdrop
  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === modalRef.current) {
      onClose()
    }
  }

  const sourceName = getSourceName(video)

  return (
    <div
      ref={modalRef}
      onClick={handleBackdropClick}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80"
    >
      {/* Close button */}
      <button
        type="button"
        onClick={onClose}
        className="absolute top-4 right-4 p-2 text-white/80 hover:text-white hover:bg-white/10 rounded-full transition-colors z-10"
        aria-label="Close"
      >
        <X size={28} />
      </button>

      {/* Previous button */}
      {hasPrevious && (
        <button
          type="button"
          onClick={onPrevious}
          className="absolute left-4 top-1/2 -translate-y-1/2 p-3 text-white/80 hover:text-white hover:bg-white/10 rounded-full transition-colors z-10"
          aria-label="Previous video"
        >
          <ChevronLeft size={36} />
        </button>
      )}

      {/* Next button */}
      {hasNext && (
        <button
          type="button"
          onClick={onNext}
          className="absolute right-4 top-1/2 -translate-y-1/2 p-3 text-white/80 hover:text-white hover:bg-white/10 rounded-full transition-colors z-10"
          aria-label="Next video"
        >
          <ChevronRight size={36} />
        </button>
      )}

      {/* Modal content */}
      <div className="w-full max-w-4xl mx-4 max-h-[90vh] overflow-y-auto rounded-lg">
        {/* Video player area */}
        <div className="bg-black">
          {video.embed_url ? (
            <div className="relative w-full" style={{ paddingBottom: '56.25%' /* 16:9 aspect ratio */ }}>
              <iframe
                src={video.embed_url}
                className="absolute inset-0 w-full h-full"
                allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                allowFullScreen
                title={video.title}
              />
            </div>
          ) : (
            <div className="relative w-full" style={{ paddingBottom: '56.25%' }}>
              <div className="absolute inset-0 flex flex-col items-center justify-center bg-[#1a1a1a]">
                {video.thumbnail_url && (
                  <img
                    src={video.thumbnail_url}
                    alt={video.title}
                    className="absolute inset-0 w-full h-full object-cover opacity-30"
                  />
                )}
                <div className="relative z-10 text-center p-8">
                  <Video size={64} className="text-white/50 mx-auto mb-4" />
                  <p className="text-white/70 mb-4">This video cannot be embedded</p>
                  <a
                    href={video.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-2 px-6 py-3 bg-[#1a73e8] text-white rounded-full hover:bg-[#1557b0] transition-colors"
                  >
                    <ExternalLink size={18} />
                    Watch on {sourceName}
                  </a>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Metadata panel */}
        <div className="bg-white p-6">
          <h2 className="text-xl font-medium text-[#202124] leading-snug">
            {video.title}
          </h2>

          <div className="flex items-center gap-2 mt-3 text-sm text-[#5f6368]">
            {video.channel && (
              <span className="font-medium text-[#202124]">{video.channel}</span>
            )}
            {video.channel && (video.views || video.published_at) && (
              <span className="text-[#70757a]">-</span>
            )}
            {video.views && (
              <span>{formatViews(video.views)}</span>
            )}
            {video.views && video.published_at && (
              <span className="text-[#70757a]">-</span>
            )}
            {video.published_at && (
              <span>{formatDate(video.published_at)}</span>
            )}
          </div>

          {video.description && (
            <p className="mt-4 text-sm text-[#4d5156] leading-relaxed">
              {video.description}
            </p>
          )}

          {/* Source link */}
          <div className="mt-4 pt-4 border-t border-[#e8eaed]">
            <a
              href={video.url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 text-sm text-[#1a73e8] hover:underline"
            >
              <ExternalLink size={14} />
              Open on {sourceName}
            </a>
          </div>
        </div>
      </div>
    </div>
  )
}

export default function VideosPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const query = searchParams.get('q') || ''

  const [videos, setVideos] = useState<VideoResult[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [viewMode, setViewMode] = useState<'list' | 'grid'>(() => {
    return (localStorage.getItem('videos_view_mode') as 'list' | 'grid') || 'list'
  })

  // Filter states
  const [durationFilter, setDurationFilter] = useState('')
  const [timeFilter, setTimeFilter] = useState('')
  const [sourceFilter, setSourceFilter] = useState('')
  const [qualityFilter, setQualityFilter] = useState('')
  const [ccFilter, setCcFilter] = useState(false)

  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null)
  const [previewIndex, setPreviewIndex] = useState<number | null>(null) // Which video is showing preview (only one at a time)
  const [isPreviewMuted, setIsPreviewMuted] = useState(true)
  const [selectedVideoIndex, setSelectedVideoIndex] = useState<number | null>(null)
  const [showBackToTop, setShowBackToTop] = useState(false)
  const filterScrollRef = useRef<HTMLDivElement>(null)

  // Handle hover for preview - only show preview for hovered video
  const handleVideoHover = useCallback((index: number | null) => {
    setHoveredIndex(index)
    setPreviewIndex(index)
    // Reset mute state when changing preview
    if (index !== previewIndex) {
      setIsPreviewMuted(true)
    }
  }, [previewIndex])

  const PER_PAGE = 30

  // Build filter options for API
  const buildFilterOptions = useCallback((): VideoSearchOptions => {
    const options: VideoSearchOptions = { per_page: PER_PAGE }
    if (durationFilter) options.duration = durationFilter
    if (timeFilter) options.time = timeFilter
    if (sourceFilter) options.source = sourceFilter
    if (qualityFilter) options.quality = qualityFilter
    if (ccFilter) options.cc = true
    return options
  }, [durationFilter, timeFilter, sourceFilter, qualityFilter, ccFilter])

  // Check if any filters are active
  const hasActiveFilters = durationFilter || timeFilter || sourceFilter || qualityFilter || ccFilter

  // Clear all filters
  const clearAllFilters = () => {
    setDurationFilter('')
    setTimeFilter('')
    setSourceFilter('')
    setQualityFilter('')
    setCcFilter(false)
  }

  // Persist view mode
  useEffect(() => {
    localStorage.setItem('videos_view_mode', viewMode)
  }, [viewMode])

  // Infinite scroll
  const loadMore = useCallback(async () => {
    if (!query || isLoading) return
    const nextPage = page + 1
    try {
      const options = buildFilterOptions()
      options.page = nextPage
      const response = await searchApi.searchVideos(query, options)
      if (response.results.length === 0) {
        setHasMore(false)
      } else {
        setVideos(prev => [...prev, ...response.results])
        setPage(nextPage)
      }
    } catch {
      setHasMore(false)
    }
  }, [query, page, isLoading, buildFilterOptions])

  const { isLoading: scrollLoading, hasMore, setHasMore, reset } = useInfiniteScroll(loadMore, {
    threshold: 300,
    enabled: videos.length > 0 && !isLoading,
  })

  // Initial search and filter changes
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
        const options = buildFilterOptions()
        const response = await searchApi.searchVideos(query, options)
        setVideos(response.results || [])
        setHasMore((response.results?.length || 0) >= PER_PAGE)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Search failed')
      } finally {
        setIsLoading(false)
      }
    }

    searchVideos()
  }, [query, durationFilter, timeFilter, sourceFilter, qualityFilter, ccFilter])

  // Scroll position tracking
  useEffect(() => {
    const handleScroll = () => {
      setShowBackToTop(window.scrollY > 500)
    }
    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  const handleSearch = (newQuery: string) => {
    setSearchParams({ q: newQuery })
  }

  const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  return (
    <div className="min-h-screen bg-white">
      <SearchHeader
        query={query}
        activeTab="videos"
        onSearch={handleSearch}
        tabsRight={
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
        }
        belowTabs={
          <div className="border-t border-[#e8eaed] bg-[#f8f9fa]">
            <div className="max-w-7xl mx-auto px-4 py-2">
              <div
                ref={filterScrollRef}
                className="flex items-center gap-2 overflow-x-auto scrollbar-hide"
                style={{ scrollbarWidth: 'none', msOverflowStyle: 'none' }}
              >
                {/* Duration filter */}
                <FilterChip
                  label="Any duration"
                  value={durationFilter}
                  options={DURATION_OPTIONS}
                  onChange={setDurationFilter}
                  isActive={!!durationFilter}
                />

                {/* Time filter */}
                <FilterChip
                  label="Any time"
                  value={timeFilter}
                  options={TIME_OPTIONS}
                  onChange={setTimeFilter}
                  isActive={!!timeFilter}
                />

                {/* Source filter */}
                <FilterChip
                  label="All sources"
                  value={sourceFilter}
                  options={SOURCE_OPTIONS}
                  onChange={setSourceFilter}
                  isActive={!!sourceFilter}
                />

                {/* Quality filter */}
                <FilterChip
                  label="Any quality"
                  value={qualityFilter}
                  options={QUALITY_OPTIONS}
                  onChange={setQualityFilter}
                  isActive={!!qualityFilter}
                />

                {/* CC toggle */}
                <ToggleChip
                  label="CC"
                  isActive={ccFilter}
                  onToggle={() => setCcFilter(!ccFilter)}
                />

                {/* Clear all filters button */}
                {hasActiveFilters && (
                  <button
                    type="button"
                    onClick={clearAllFilters}
                    className="flex items-center gap-1 px-3 py-1.5 text-sm text-[#1a73e8] hover:bg-[#e8f0fe] rounded-full transition-colors whitespace-nowrap flex-shrink-0"
                  >
                    <X size={14} />
                    Clear all
                  </button>
                )}
              </div>
            </div>
          </div>
        }
      />

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
                    <button
                      key={`${video.id}-${index}`}
                      type="button"
                      onClick={() => setSelectedVideoIndex(index)}
                      className="flex gap-4 p-3 rounded-lg hover:bg-[#f8f9fa] transition-colors w-full text-left cursor-pointer"
                      onMouseEnter={() => handleVideoHover(index)}
                      onMouseLeave={() => handleVideoHover(null)}
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

                        {/* Video Preview on hover (with debounce) */}
                        <VideoPreview
                          video={video}
                          isVisible={previewIndex === index}
                          isMuted={isPreviewMuted}
                          onMuteToggle={setIsPreviewMuted}
                        />

                        {/* Play icon on hover - only show if no embed_url (preview not available) */}
                        {hoveredIndex === index && !video.embed_url && (
                          <div className="absolute inset-0 flex items-center justify-center bg-black/30 transition-opacity">
                            <div className="w-16 h-16 rounded-full bg-black/70 flex items-center justify-center">
                              <Play size={28} className="text-white ml-1" fill="white" />
                            </div>
                          </div>
                        )}
                        {(video.duration || video.duration_seconds) && previewIndex !== index && (
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
                              <span>-</span>
                              <span>{formatViews(video.views)}</span>
                            </>
                          )}
                          {video.published_at && (
                            <>
                              <span>-</span>
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
                    </button>
                  ))}
                </div>
              ) : (
                /* Grid View */
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                  {videos.map((video, index) => (
                    <button
                      key={`${video.id}-${index}`}
                      type="button"
                      onClick={() => setSelectedVideoIndex(index)}
                      className="block rounded-lg overflow-hidden hover:shadow-lg transition-shadow bg-white border border-[#e8eaed] text-left cursor-pointer"
                      onMouseEnter={() => handleVideoHover(index)}
                      onMouseLeave={() => handleVideoHover(null)}
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

                        {/* Video Preview on hover (with debounce) */}
                        <VideoPreview
                          video={video}
                          isVisible={previewIndex === index}
                          isMuted={isPreviewMuted}
                          onMuteToggle={setIsPreviewMuted}
                        />

                        {/* Play icon on hover - only show if no embed_url (preview not available) */}
                        {hoveredIndex === index && !video.embed_url && (
                          <div className="absolute inset-0 flex items-center justify-center bg-black/30 transition-opacity">
                            <div className="w-14 h-14 rounded-full bg-black/70 flex items-center justify-center">
                              <Play size={24} className="text-white ml-0.5" fill="white" />
                            </div>
                          </div>
                        )}
                        {(video.duration || video.duration_seconds) && previewIndex !== index && (
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
                            {video.views && video.published_at && <span>-</span>}
                            {video.published_at && <span>{formatDate(video.published_at)}</span>}
                          </div>
                        </div>
                      </div>
                    </button>
                  ))}
                </div>
              )}

              {/* Loading more indicator */}
              {scrollLoading && (
                <div className="flex justify-center py-8">
                  <div className="w-6 h-6 border-2 border-[#1a73e8] border-t-transparent rounded-full animate-spin" />
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

      {/* Video Player Modal */}
      {selectedVideoIndex !== null && videos[selectedVideoIndex] && (
        <VideoPlayer
          video={videos[selectedVideoIndex]}
          onClose={() => setSelectedVideoIndex(null)}
          onPrevious={() => setSelectedVideoIndex(prev => prev !== null && prev > 0 ? prev - 1 : prev)}
          onNext={() => setSelectedVideoIndex(prev => prev !== null && prev < videos.length - 1 ? prev + 1 : prev)}
          hasPrevious={selectedVideoIndex > 0}
          hasNext={selectedVideoIndex < videos.length - 1}
        />
      )}

      {/* Hide scrollbar CSS */}
      <style>{`
        .scrollbar-hide::-webkit-scrollbar {
          display: none;
        }
      `}</style>
    </div>
  )
}
