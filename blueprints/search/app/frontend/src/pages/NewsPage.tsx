import { useEffect, useState, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Newspaper, ChevronLeft, ChevronRight, Clock, User } from 'lucide-react'
import { SearchHeader } from '../components/SearchHeader'
import { Pagination } from '../components/Pagination'
import { searchApi } from '../api/search'
import type { NewsResult } from '../types'

const PER_PAGE = 20

function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString)
  if (isNaN(date.getTime())) return ''
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / (1000 * 60))
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`
  if (diffDays < 30) return `${Math.floor(diffDays / 7)}w ago`
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined })
}

function isBreaking(dateString: string): boolean {
  const date = new Date(dateString)
  if (isNaN(date.getTime())) return false
  return Date.now() - date.getTime() < 3600000
}

function getSourceFavicon(source: string | undefined, url: string): string {
  if (!source && !url) return ''
  try {
    const domain = new URL(url).hostname
    return `https://www.google.com/s2/favicons?domain=${domain}&sz=32`
  } catch {
    if (source) return `https://www.google.com/s2/favicons?domain=${source}&sz=32`
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

        if (page === 1 && results.length > 5) {
          setTopStories(results.slice(0, 5))
          setNews(results.slice(5))
        } else {
          setTopStories([])
          setNews(results)
        }

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

      <main>
        <div className="max-w-5xl mx-auto px-4 py-4">
          {isLoading ? (
            <div className="flex justify-center py-12">
              <div className="w-8 h-8 border-[3px] border-[#1a73e8] border-t-transparent rounded-full animate-spin" />
            </div>
          ) : error ? (
            <div className="py-12 text-center">
              <p className="text-red-600">{error}</p>
            </div>
          ) : news.length > 0 || topStories.length > 0 ? (
            <>
              {/* Top Stories Carousel */}
              {topStories.length > 0 && (
                <div className="mb-6">
                  <h2 className="text-base font-medium text-[#202124] mb-3 flex items-center gap-2">
                    <Newspaper size={18} className="text-[#1a73e8]" />
                    Top Stories
                  </h2>
                  <div className="relative">
                    <button
                      type="button"
                      onClick={() => scrollCarousel('left')}
                      className="absolute left-0 top-1/2 -translate-y-1/2 -translate-x-3 z-10 w-8 h-8 bg-white rounded-full shadow-md flex items-center justify-center text-[#5f6368] hover:text-[#202124] transition-colors border border-[#dadce0]"
                    >
                      <ChevronLeft size={18} />
                    </button>
                    <button
                      type="button"
                      onClick={() => scrollCarousel('right')}
                      className="absolute right-0 top-1/2 -translate-y-1/2 translate-x-3 z-10 w-8 h-8 bg-white rounded-full shadow-md flex items-center justify-center text-[#5f6368] hover:text-[#202124] transition-colors border border-[#dadce0]"
                    >
                      <ChevronRight size={18} />
                    </button>

                    <div
                      ref={carouselRef}
                      className="flex gap-3 overflow-x-auto scroll-smooth pb-1"
                      style={{ scrollbarWidth: 'none', msOverflowStyle: 'none' }}
                    >
                      {topStories.map((article, index) => (
                        <a
                          key={`top-${article.id}-${index}`}
                          href={article.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="flex-shrink-0 w-[260px] bg-white rounded-lg border border-[#dadce0] overflow-hidden hover:shadow-md transition-shadow group"
                        >
                          {(article.image_url || article.thumbnail_url) ? (
                            <div className="h-[140px] bg-[#f8f9fa] overflow-hidden">
                              <img
                                src={article.image_url || article.thumbnail_url}
                                alt=""
                                className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                                loading="lazy"
                                onError={(e) => {
                                  e.currentTarget.parentElement!.style.display = 'none'
                                }}
                              />
                            </div>
                          ) : (
                            <div className="h-[140px] bg-gradient-to-br from-[#e8f0fe] to-[#f8f9fa] flex items-center justify-center">
                              <Newspaper size={32} className="text-[#dadce0]" />
                            </div>
                          )}
                          <div className="p-3">
                            <div className="flex items-center gap-1.5 mb-1.5">
                              <img
                                src={getSourceFavicon(article.source, article.url)}
                                alt=""
                                className="w-4 h-4 rounded-sm"
                                loading="lazy"
                                onError={(e) => {
                                  e.currentTarget.style.display = 'none'
                                  const parent = e.currentTarget.parentElement
                                  if (parent) {
                                    const span = document.createElement('span')
                                    span.className = 'w-4 h-4 rounded-sm bg-[#e8eaed] flex items-center justify-center text-[8px] font-bold text-[#5f6368]'
                                    span.textContent = getSourceInitial(article.source)
                                    parent.insertBefore(span, parent.firstChild)
                                  }
                                }}
                              />
                              <span className="text-xs text-[#70757a] truncate">
                                {article.source || article.source_domain}
                              </span>
                            </div>
                            <h3 className="text-[13px] font-medium text-[#202124] line-clamp-2 leading-tight">
                              {article.title}
                            </h3>
                            <div className="flex items-center gap-1 mt-2 text-[11px] text-[#70757a]">
                              <Clock size={10} />
                              <span className={isBreaking(article.published_at) ? 'text-[#1a73e8] font-medium' : ''}>
                                {formatRelativeTime(article.published_at)}
                              </span>
                            </div>
                          </div>
                        </a>
                      ))}
                    </div>
                  </div>
                </div>
              )}

              {/* News Cards List */}
              <div className="space-y-0 divide-y divide-[#e8eaed]">
                {news.map((article, index) => (
                  <a
                    key={`news-${article.id}-${index}`}
                    href={article.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex gap-4 py-5 group"
                  >
                    {/* Text content */}
                    <div className="flex-1 min-w-0">
                      {/* Source line: favicon + source + author + time */}
                      <div className="flex items-center gap-1.5 mb-2 text-xs">
                        <div className="w-5 h-5 rounded-full bg-[#f1f3f4] flex items-center justify-center overflow-hidden flex-shrink-0">
                          <img
                            src={getSourceFavicon(article.source, article.url)}
                            alt=""
                            className="w-4 h-4"
                            loading="lazy"
                            onError={(e) => {
                              e.currentTarget.style.display = 'none'
                              const parent = e.currentTarget.parentElement
                              if (parent) {
                                parent.innerHTML = `<span class="text-[10px] font-bold text-[#5f6368]">${getSourceInitial(article.source)}</span>`
                              }
                            }}
                          />
                        </div>
                        <span className="font-medium text-[#202124] truncate">
                          {article.source || article.source_domain}
                        </span>
                        {article.author && (
                          <>
                            <span className="text-[#bdc1c6]">/</span>
                            <span className="text-[#5f6368] truncate flex items-center gap-0.5">
                              <User size={10} className="flex-shrink-0" />
                              {article.author}
                            </span>
                          </>
                        )}
                        <span className="text-[#bdc1c6]">&middot;</span>
                        <span className={`flex-shrink-0 ${
                          isBreaking(article.published_at)
                            ? 'text-[#1a73e8] font-medium'
                            : 'text-[#70757a]'
                        }`}>
                          {formatRelativeTime(article.published_at)}
                        </span>
                      </div>

                      {/* Title */}
                      <h3 className="text-base font-medium text-[#202124] line-clamp-2 leading-snug group-hover:text-[#1a73e8] transition-colors">
                        {article.title}
                      </h3>

                      {/* Snippet */}
                      {article.snippet && (
                        <p className="text-sm text-[#4d5156] mt-1.5 line-clamp-2 leading-relaxed">
                          {article.snippet}
                        </p>
                      )}
                    </div>

                    {/* Thumbnail */}
                    {(article.image_url || article.thumbnail_url) && (
                      <div className="flex-shrink-0 w-[120px] h-[120px] rounded-lg overflow-hidden bg-[#f8f9fa] self-center">
                        <img
                          src={article.image_url || article.thumbnail_url}
                          alt=""
                          className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                          loading="lazy"
                          onError={(e) => {
                            e.currentTarget.parentElement!.style.display = 'none'
                          }}
                        />
                      </div>
                    )}
                  </a>
                ))}
              </div>

              <Pagination page={page} totalPages={totalPages} onPageChange={handlePageChange} />
            </>
          ) : query ? (
            <div className="py-16 text-center">
              <Newspaper size={48} className="mx-auto text-[#dadce0] mb-4" />
              <p className="text-[#5f6368] text-lg">No news articles found</p>
              <p className="text-[#70757a] text-sm mt-1">Try a different search term</p>
            </div>
          ) : (
            <div className="py-16 text-center">
              <Newspaper size={48} className="mx-auto text-[#dadce0] mb-4" />
              <p className="text-[#5f6368] text-lg">Search for news</p>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
