import { useEffect, useState, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Newspaper, ChevronLeft, ChevronRight, Clock, ExternalLink } from 'lucide-react'
import { SearchHeader } from '../components/SearchHeader'
import { Pagination } from '../components/Pagination'
import { searchApi } from '../api/search'
import type { NewsResult } from '../types'

const PER_PAGE = 20

function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / (1000 * 60))
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins} min${diffMins > 1 ? 's' : ''} ago`
  if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`
  if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`
  return date.toLocaleDateString()
}

function getSourceFavicon(source: string | undefined, url: string): string {
  if (!source && !url) return ''
  try {
    const domain = source || new URL(url).hostname
    return `https://www.google.com/s2/favicons?domain=${domain}&sz=32`
  } catch {
    return ''
  }
}

function getSourceInitial(source: string | undefined): string {
  if (!source) return '?'
  return source.charAt(0).toUpperCase()
}

export default function NewsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const query = searchParams.get('q') || ''
  const page = parseInt(searchParams.get('page') || '1', 10)

  const [news, setNews] = useState<NewsResult[]>([])
  const [topStories, setTopStories] = useState<NewsResult[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [totalResults, setTotalResults] = useState(0)
  const carouselRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!query) {
      setNews([])
      setTopStories([])
      return
    }

    const searchNews = async () => {
      setIsLoading(true)
      setError(null)

      try {
        const response = await searchApi.searchNews(query, { page, per_page: PER_PAGE })
        const results = response.results || []

        // First 5 results are "top stories" (only on page 1)
        if (page === 1 && results.length > 5) {
          setTopStories(results.slice(0, 5))
          setNews(results.slice(5))
        } else {
          setTopStories([])
          setNews(results)
        }

        // Estimate total (SearXNG doesn't always provide this)
        setTotalResults(response.total_results || results.length * 10)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Search failed')
      } finally {
        setIsLoading(false)
      }
    }

    searchNews()
  }, [query, page])

  const handleSearch = (newQuery: string) => {
    setSearchParams({ q: newQuery })
  }

  const handlePageChange = (newPage: number) => {
    setSearchParams({ q: query, page: String(newPage) })
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const scrollCarousel = (direction: 'left' | 'right') => {
    if (carouselRef.current) {
      const scrollAmount = direction === 'left' ? -400 : 400
      carouselRef.current.scrollBy({ left: scrollAmount, behavior: 'smooth' })
    }
  }

  const totalPages = Math.ceil(totalResults / PER_PAGE)

  return (
    <div className="min-h-screen bg-white">
      <SearchHeader query={query} activeTab="news" onSearch={handleSearch} />

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
          ) : news.length > 0 || topStories.length > 0 ? (
            <>
              {/* Top Stories Carousel */}
              {topStories.length > 0 && (
                <div className="mb-8">
                  <h2 className="text-lg font-medium text-[#202124] mb-4 flex items-center gap-2">
                    <Newspaper size={20} className="text-[#1a73e8]" />
                    Top Stories
                  </h2>
                  <div className="relative">
                    {/* Carousel navigation */}
                    <button
                      type="button"
                      onClick={() => scrollCarousel('left')}
                      className="absolute left-0 top-1/2 -translate-y-1/2 -translate-x-4 z-10 w-10 h-10 bg-white rounded-full shadow-lg flex items-center justify-center text-[#5f6368] hover:text-[#202124] transition-colors"
                    >
                      <ChevronLeft size={24} />
                    </button>
                    <button
                      type="button"
                      onClick={() => scrollCarousel('right')}
                      className="absolute right-0 top-1/2 -translate-y-1/2 translate-x-4 z-10 w-10 h-10 bg-white rounded-full shadow-lg flex items-center justify-center text-[#5f6368] hover:text-[#202124] transition-colors"
                    >
                      <ChevronRight size={24} />
                    </button>

                    {/* Carousel container */}
                    <div
                      ref={carouselRef}
                      className="flex gap-4 overflow-x-auto scrollbar-hide scroll-smooth pb-2"
                      style={{ scrollbarWidth: 'none', msOverflowStyle: 'none' }}
                    >
                      {topStories.map((article, index) => (
                        <a
                          key={`top-${article.id}-${index}`}
                          href={article.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="flex-shrink-0 w-80 bg-white rounded-lg shadow-sm border border-[#e8eaed] overflow-hidden hover:shadow-md transition-shadow"
                        >
                          {(article.image_url || article.thumbnail_url) && (
                            <div className="h-40 bg-[#f1f3f4]">
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
                          <div className="p-4">
                            <div className="flex items-center gap-2 mb-2">
                              <img
                                src={getSourceFavicon(article.source || article.source_name, article.url)}
                                alt=""
                                className="w-4 h-4 rounded"
                                onError={(e) => {
                                  e.currentTarget.style.display = 'none'
                                }}
                              />
                              <span className="text-xs text-[#70757a]">
                                {article.source || article.source_name || article.source_domain}
                              </span>
                            </div>
                            <h3 className="text-sm font-medium text-[#202124] line-clamp-2">
                              {article.title}
                            </h3>
                            <div className="flex items-center gap-1 mt-2 text-xs text-[#70757a]">
                              <Clock size={12} />
                              <span>{article.published_at ? formatRelativeTime(article.published_at) : ''}</span>
                            </div>
                          </div>
                        </a>
                      ))}
                    </div>
                  </div>
                </div>
              )}

              {/* News Cards */}
              <div className="space-y-4">
                {news.map((article, index) => (
                  <a
                    key={`news-${article.id}-${index}`}
                    href={article.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="block bg-white rounded-lg shadow-sm border border-[#e8eaed] overflow-hidden hover:shadow-md transition-shadow"
                  >
                    <div className="flex">
                      <div className="flex-1 p-5">
                        {/* Source with favicon */}
                        <div className="flex items-center gap-2 mb-3">
                          <div className="w-6 h-6 rounded-full bg-[#f1f3f4] flex items-center justify-center overflow-hidden">
                            <img
                              src={getSourceFavicon(article.source || article.source_name, article.url)}
                              alt=""
                              className="w-5 h-5"
                              onError={(e) => {
                                e.currentTarget.style.display = 'none'
                                const parent = e.currentTarget.parentElement
                                if (parent) {
                                  parent.innerHTML = `<span class="text-xs font-medium text-[#5f6368]">${getSourceInitial(article.source || article.source_name)}</span>`
                                }
                              }}
                            />
                          </div>
                          <span className="text-sm font-medium text-[#202124]">
                            {article.source || article.source_name || article.source_domain}
                          </span>
                          <span className="text-[#70757a]">•</span>
                          <span className={`text-sm flex items-center gap-1 ${
                            article.published_at && new Date(article.published_at).getTime() > Date.now() - 3600000
                              ? 'text-[#1a73e8] font-medium'
                              : 'text-[#70757a]'
                          }`}>
                            <Clock size={14} />
                            {article.published_at ? formatRelativeTime(article.published_at) : ''}
                          </span>
                        </div>

                        {/* Title */}
                        <h3 className="text-lg font-medium text-[#202124] line-clamp-2 hover:text-[#1a73e8] transition-colors">
                          {article.title}
                        </h3>

                        {/* Snippet */}
                        {article.snippet && (
                          <p className="text-sm text-[#4d5156] mt-2 line-clamp-2">
                            {article.snippet}
                          </p>
                        )}

                        {/* External link indicator */}
                        <div className="mt-3 flex items-center gap-1 text-xs text-[#70757a]">
                          <ExternalLink size={12} />
                          <span>Opens in new tab</span>
                        </div>
                      </div>

                      {/* Image */}
                      {(article.image_url || article.thumbnail_url) && (
                        <div className="flex-shrink-0 w-48 bg-[#f1f3f4]">
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

              <Pagination page={page} totalPages={totalPages} onPageChange={handlePageChange} />
            </>
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
