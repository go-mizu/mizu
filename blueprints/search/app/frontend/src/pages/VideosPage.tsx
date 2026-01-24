import { useEffect, useState } from 'react'
import { useSearchParams, Link, useNavigate } from 'react-router-dom'
import { Settings, Image, Video, Newspaper } from 'lucide-react'
import { SearchBox } from '../components/SearchBox'
import { searchApi } from '../api/search'
import type { VideoResult } from '../types'

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

export default function VideosPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()
  const query = searchParams.get('q') || ''

  const [videos, setVideos] = useState<VideoResult[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!query) return

    const searchVideos = async () => {
      setIsLoading(true)
      setError(null)

      try {
        const response = await searchApi.searchVideos(query, { per_page: 50 })
        setVideos(response.results || [])
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Search failed')
      } finally {
        setIsLoading(false)
      }
    }

    searchVideos()
  }, [query])

  const handleSearch = (newQuery: string) => {
    setSearchParams({ q: newQuery })
  }

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="sticky top-0 bg-white z-50">
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

          {/* Tabs */}
          <div className="search-tabs mt-2" style={{ paddingLeft: 0 }}>
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
          ) : videos.length > 0 ? (
            <div className="space-y-6">
              {videos.map((video) => (
                <a
                  key={video.id}
                  href={video.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex gap-4 p-3 rounded-lg hover:bg-[#f8f9fa] transition-colors"
                >
                  <div className="relative flex-shrink-0 w-48 h-28 bg-[#f1f3f4] rounded-lg overflow-hidden">
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
                        <Video size={32} className="text-[#9aa0a6]" />
                      </div>
                    )}
                    {(video.duration || video.duration_seconds) && (
                      <span className="absolute bottom-1 right-1 bg-black/80 text-white text-xs px-1 rounded">
                        {video.duration || formatDuration(video.duration_seconds)}
                      </span>
                    )}
                  </div>
                  <div className="flex-1 min-w-0">
                    <h3 className="text-[#1a0dab] text-lg font-medium line-clamp-2 hover:underline">
                      {video.title}
                    </h3>
                    <p className="text-sm text-[#70757a] mt-1">
                      {video.channel || video.source_domain || video.engine}
                      {video.published_at && ` - ${new Date(video.published_at).toLocaleDateString()}`}
                    </p>
                    {video.description && (
                      <p className="text-sm text-[#4d5156] mt-2 line-clamp-2">
                        {video.description}
                      </p>
                    )}
                  </div>
                </a>
              ))}
            </div>
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
    </div>
  )
}
