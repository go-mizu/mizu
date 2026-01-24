import { useEffect, useState } from 'react'
import { useSearchParams, Link, useNavigate } from 'react-router-dom'
import { Settings, Image, Video, Newspaper } from 'lucide-react'
import { SearchBox } from '../components/SearchBox'
import { searchApi } from '../api/search'
import type { NewsResult } from '../types'

export default function NewsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()
  const query = searchParams.get('q') || ''

  const [news, setNews] = useState<NewsResult[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!query) return

    const searchNews = async () => {
      setIsLoading(true)
      setError(null)

      try {
        const response = await searchApi.searchNews(query, { per_page: 50 })
        setNews(response.results || [])
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Search failed')
      } finally {
        setIsLoading(false)
      }
    }

    searchNews()
  }, [query])

  const handleSearch = (newQuery: string) => {
    setSearchParams({ q: newQuery })
  }

  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

    if (diffHours < 1) return 'Just now'
    if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`
    if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`
    return date.toLocaleDateString()
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
              className="search-tab"
              onClick={() => navigate(`/videos?q=${encodeURIComponent(query)}`)}
            >
              <Video size={16} />
              Videos
            </button>
            <button
              type="button"
              className="search-tab active"
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
          ) : news.length > 0 ? (
            <div className="space-y-6">
              {news.map((article) => (
                <a
                  key={article.id}
                  href={article.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="block p-4 rounded-lg hover:bg-[#f8f9fa] transition-colors"
                >
                  <div className="flex gap-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 text-sm text-[#70757a] mb-1">
                        <span>{article.source || article.source_name || article.source_domain}</span>
                        {article.published_at && (
                          <>
                            <span>-</span>
                            <span>{formatDate(article.published_at)}</span>
                          </>
                        )}
                      </div>
                      <h3 className="text-[#1a0dab] text-lg font-medium line-clamp-2 hover:underline">
                        {article.title}
                      </h3>
                      {article.snippet && (
                        <p className="text-sm text-[#4d5156] mt-2 line-clamp-2">
                          {article.snippet}
                        </p>
                      )}
                    </div>
                    {(article.image_url || article.thumbnail_url) && (
                      <div className="flex-shrink-0 w-32 h-20 bg-[#f1f3f4] rounded-lg overflow-hidden">
                        <img
                          src={article.image_url || article.thumbnail_url}
                          alt=""
                          className="w-full h-full object-cover"
                          onError={(e) => {
                            e.currentTarget.parentElement!.style.display = 'none'
                          }}
                        />
                      </div>
                    )}
                  </div>
                </a>
              ))}
            </div>
          ) : query ? (
            <div className="py-12 text-center">
              <p className="text-[#70757a]">No news found</p>
            </div>
          ) : (
            <div className="py-12 text-center">
              <p className="text-[#70757a]">Search for news</p>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
