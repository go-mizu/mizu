import { useEffect, useState, useRef } from 'react'
import { useSearchParams, Link, useNavigate } from 'react-router-dom'
import { Settings, Image, Video, Newspaper, ChevronDown, Sparkles, ExternalLink } from 'lucide-react'
import { SearchBox } from '../components/SearchBox'
import { SearchResult } from '../components/SearchResult'
import { InstantAnswer } from '../components/InstantAnswer'
import { KnowledgePanel } from '../components/KnowledgePanel'
import { AISummary } from '../components/ai'
import { CheatSheetWidget, RelatedSearchesWidget } from '../components/widgets'
import { searchApi } from '../api/search'
import { useSearchStore } from '../stores/searchStore'
import { useAIStore } from '../stores/aiStore'
import type { SearchResponse, CheatSheet } from '../types'

type SearchTab = 'all' | 'ai' | 'images' | 'videos' | 'news'

const TIME_OPTIONS = [
  { value: '', label: 'Any time' },
  { value: 'day', label: 'Past 24 hours' },
  { value: 'week', label: 'Past week' },
  { value: 'month', label: 'Past month' },
  { value: 'year', label: 'Past year' },
]

export default function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()
  const query = searchParams.get('q') || ''
  const page = parseInt(searchParams.get('page') || '1', 10)
  const timeFilter = searchParams.get('time') || ''
  const [activeTab] = useState<SearchTab>('all')

  const [results, setResults] = useState<SearchResponse | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showTimeDropdown, setShowTimeDropdown] = useState(false)
  const [bangRedirect, setBangRedirect] = useState<{ url: string; name: string } | null>(null)
  const timeDropdownRef = useRef<HTMLDivElement>(null)

  const { settings, addRecentSearch } = useSearchStore()
  const { aiAvailable } = useAIStore()

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (timeDropdownRef.current && !timeDropdownRef.current.contains(e.target as Node)) {
        setShowTimeDropdown(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  useEffect(() => {
    if (!query) return

    const performSearch = async () => {
      setIsLoading(true)
      setError(null)
      setBangRedirect(null)

      try {
        const response = await searchApi.search(query, {
          page,
          per_page: settings.results_per_page,
          time: timeFilter,
          safe: settings.safe_search,
          region: settings.region,
          lang: settings.language,
        })

        // Check for bang redirect
        if (response.redirect) {
          // If it's an external redirect, show confirmation
          if (response.redirect.startsWith('http')) {
            setBangRedirect({
              url: response.redirect,
              name: response.bang?.name || 'External site'
            })
            setResults(null)
          } else {
            // Internal redirect (AI mode, images, etc.)
            navigate(response.redirect)
            return
          }
        } else if (response.category) {
          // Redirect to category page
          navigate(`/${response.category}?q=${encodeURIComponent(response.query)}`)
          return
        } else {
          setResults(response)
        }
        addRecentSearch(query)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Search failed')
      } finally {
        setIsLoading(false)
      }
    }

    performSearch()
  }, [query, page, timeFilter, settings])

  const handleSearch = (newQuery: string) => {
    setSearchParams({ q: newQuery })
  }

  const handlePageChange = (newPage: number) => {
    setSearchParams({ q: query, page: String(newPage) })
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const handleTimeFilter = (value: string) => {
    if (value) {
      setSearchParams({ q: query, time: value })
    } else {
      setSearchParams({ q: query })
    }
    setShowTimeDropdown(false)
  }

  const totalPages = results ? Math.ceil(results.total_results / settings.results_per_page) : 0
  const currentTimeLabel = TIME_OPTIONS.find(opt => opt.value === timeFilter)?.label || 'Any time'

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="sticky top-0 bg-white z-50">
        <div className="max-w-7xl mx-auto px-4 py-3">
          <div className="flex items-center gap-6">
            {/* Logo */}
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

            {/* Search box */}
            <div className="flex-1 max-w-xl">
              <SearchBox
                initialValue={query}
                size="sm"
                onSearch={handleSearch}
              />
            </div>

            {/* AI Sessions */}
            {aiAvailable && (
              <Link
                to="/ai/sessions"
                className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
                title="AI Research Sessions"
              >
                <Sparkles size={20} />
              </Link>
            )}

            {/* Settings */}
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
              className={`search-tab ${activeTab === 'all' ? 'active' : ''}`}
              onClick={() => navigate(`/search?q=${encodeURIComponent(query)}`)}
            >
              All
            </button>
            {aiAvailable && (
              <button
                type="button"
                className={`search-tab ${activeTab === 'ai' ? 'active' : ''}`}
                onClick={() => navigate(`/ai?q=${encodeURIComponent(query)}`)}
              >
                <Sparkles size={16} />
                AI
              </button>
            )}
            <button
              type="button"
              className={`search-tab ${activeTab === 'images' ? 'active' : ''}`}
              onClick={() => navigate(`/images?q=${encodeURIComponent(query)}`)}
            >
              <Image size={16} />
              Images
            </button>
            <button
              type="button"
              className={`search-tab ${activeTab === 'videos' ? 'active' : ''}`}
              onClick={() => navigate(`/videos?q=${encodeURIComponent(query)}`)}
            >
              <Video size={16} />
              Videos
            </button>
            <button
              type="button"
              className={`search-tab ${activeTab === 'news' ? 'active' : ''}`}
              onClick={() => navigate(`/news?q=${encodeURIComponent(query)}`)}
            >
              <Newspaper size={16} />
              News
            </button>

            {/* Time filter */}
            <div className="time-filter ml-auto" ref={timeDropdownRef}>
              <button
                type="button"
                className="time-filter-button"
                onClick={() => setShowTimeDropdown(!showTimeDropdown)}
              >
                {currentTimeLabel}
                <ChevronDown size={16} />
              </button>

              {showTimeDropdown && (
                <div className="time-filter-dropdown">
                  {TIME_OPTIONS.map(option => (
                    <button
                      key={option.value}
                      type="button"
                      className={`time-filter-option ${timeFilter === option.value ? 'active' : ''}`}
                      onClick={() => handleTimeFilter(option.value)}
                    >
                      {option.label}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main>
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex gap-8">
            {/* Results */}
            <div className="flex-1 max-w-2xl">
              {isLoading ? (
                <div className="flex justify-center py-12">
                  <div className="w-8 h-8 border-4 border-[#1a73e8] border-t-transparent rounded-full animate-spin" />
                </div>
              ) : bangRedirect ? (
                <div className="py-12">
                  <div className="bg-[#f8f9fa] border border-[#dadce0] rounded-lg p-6 text-center">
                    <ExternalLink size={32} className="mx-auto mb-4 text-[#1a73e8]" />
                    <h2 className="text-lg font-medium text-[#202124] mb-2">
                      Redirecting to {bangRedirect.name}
                    </h2>
                    <p className="text-sm text-[#5f6368] mb-4">
                      You're about to leave Mizu Search
                    </p>
                    <div className="flex justify-center gap-3">
                      <a
                        href={bangRedirect.url}
                        className="px-4 py-2 bg-[#1a73e8] text-white rounded-lg hover:bg-[#1557b0] transition-colors"
                        rel="noopener noreferrer"
                      >
                        Continue to {bangRedirect.name}
                      </a>
                      <button
                        type="button"
                        onClick={() => {
                          setBangRedirect(null)
                          navigate('/')
                        }}
                        className="px-4 py-2 border border-[#dadce0] rounded-lg hover:bg-[#f1f3f4] transition-colors text-[#5f6368]"
                      >
                        Go Back
                      </button>
                    </div>
                    <p className="text-xs text-[#9aa0a6] mt-4 break-all">
                      {bangRedirect.url}
                    </p>
                  </div>
                </div>
              ) : error ? (
                <div className="py-12 text-center">
                  <p className="text-red-600">{error}</p>
                </div>
              ) : results ? (
                <div>
                  {/* Stats */}
                  <p className="text-xs text-[#70757a] mb-4">
                    About {(results.total_results ?? 0).toLocaleString()} results ({(results.search_time_ms ?? 0).toFixed(2)} ms)
                  </p>

                  {/* Corrected query */}
                  {results.corrected_query && (
                    <p className="text-sm mb-4">
                      Showing results for{' '}
                      <button
                        type="button"
                        onClick={() => handleSearch(results.corrected_query!)}
                        className="font-medium text-[#1a73e8] hover:underline"
                      >
                        {results.corrected_query}
                      </button>
                    </p>
                  )}

                  {/* AI Summary */}
                  {aiAvailable && query && (
                    <div className="mb-6">
                      <AISummary
                        query={query}
                        onFollowUp={(q) => handleSearch(q)}
                      />
                    </div>
                  )}

                  {/* Widgets - Cheat sheets */}
                  {results.widgets?.filter(w => w.type === 'cheat_sheet').map((widget, index) => (
                    <CheatSheetWidget key={index} sheet={widget.content as CheatSheet} />
                  ))}

                  {/* Instant answer */}
                  {results.instant_answer && (
                    <div className="mb-4">
                      <InstantAnswer answer={results.instant_answer} />
                    </div>
                  )}

                  {/* Results list */}
                  {(results.results || []).map((result) => (
                    <SearchResult key={result.id} result={result} />
                  ))}

                  {/* Related searches - from widgets or response */}
                  {(() => {
                    const relatedWidget = results.widgets?.find(w => w.type === 'related_searches')
                    const relatedSearches = relatedWidget
                      ? (relatedWidget.content as string[])
                      : results.related_searches || []

                    if (relatedSearches.length > 0) {
                      return (
                        <RelatedSearchesWidget
                          searches={relatedSearches}
                          onSearch={handleSearch}
                        />
                      )
                    }
                    return null
                  })()}

                  {/* Pagination */}
                  {totalPages > 1 && (
                    <div className="pagination">
                      {page > 1 && (
                        <button
                          type="button"
                          className="pagination-item"
                          onClick={() => handlePageChange(page - 1)}
                        >
                          &lt;
                        </button>
                      )}
                      {(() => {
                        // Sliding window pagination
                        const windowSize = 10
                        let start = Math.max(1, page - Math.floor(windowSize / 2))
                        const end = Math.min(totalPages, start + windowSize - 1)
                        if (end - start + 1 < windowSize) {
                          start = Math.max(1, end - windowSize + 1)
                        }
                        return Array.from({ length: end - start + 1 }, (_, i) => start + i).map(pageNum => (
                          <button
                            key={pageNum}
                            type="button"
                            className={`pagination-item ${page === pageNum ? 'active' : ''}`}
                            onClick={() => handlePageChange(pageNum)}
                          >
                            {pageNum}
                          </button>
                        ))
                      })()}
                      {page < totalPages && (
                        <button
                          type="button"
                          className="pagination-item"
                          onClick={() => handlePageChange(page + 1)}
                        >
                          &gt;
                        </button>
                      )}
                    </div>
                  )}
                </div>
              ) : (
                <div className="py-12 text-center">
                  <p className="text-[#70757a]">Enter a search query</p>
                </div>
              )}
            </div>

            {/* Sidebar - Knowledge Panel */}
            {results?.knowledge_panel && (
              <div className="hidden lg:block w-80">
                <KnowledgePanel panel={results.knowledge_panel} />
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
