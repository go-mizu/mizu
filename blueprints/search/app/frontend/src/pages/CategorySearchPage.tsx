import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { RefreshCw } from 'lucide-react'
import { SearchHeader } from '../components/SearchHeader'
import { Pagination } from '../components/Pagination'
import { SearchResult } from '../components/SearchResult'
import { ReaderView } from '../components/ReaderView'
import { searchApi } from '../api/search'
import { useSearchStore } from '../stores/searchStore'
import type { SearchResponse } from '../types'
import type { SearchTab } from '../components/SearchHeader'

interface CategorySearchPageProps {
  category: string
  tab: SearchTab
  searchFn: (query: string, options: Record<string, unknown>) => Promise<SearchResponse>
}

export default function CategorySearchPage({ category, tab, searchFn }: CategorySearchPageProps) {
  const [searchParams, setSearchParams] = useSearchParams()
  const query = searchParams.get('q') || ''
  const page = parseInt(searchParams.get('page') || '1', 10)
  const [results, setResults] = useState<SearchResponse | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [readerUrl, setReaderUrl] = useState<string | null>(null)
  const [isRefetching, setIsRefetching] = useState(false)

  const { settings, addRecentSearch } = useSearchStore()

  const doSearch = async (refetch = false) => {
    if (!query) return
    setIsLoading(!refetch)
    if (refetch) setIsRefetching(true)
    setError(null)

    try {
      const response = await searchFn(query, {
        page,
        per_page: settings.results_per_page,
        safe: settings.safe_search,
        region: settings.region,
        lang: settings.language,
        refetch,
      })
      setResults(response)
      addRecentSearch(query)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed')
    } finally {
      setIsLoading(false)
      setIsRefetching(false)
    }
  }

  useEffect(() => {
    if (!query) return
    doSearch()
  }, [query, page, settings])

  const handleSearch = (newQuery: string) => {
    setSearchParams({ q: newQuery })
  }

  const handlePageChange = (newPage: number) => {
    setSearchParams({ q: query, page: String(newPage) })
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const totalPages = results ? Math.ceil(results.total_results / settings.results_per_page) : 0

  return (
    <div className="min-h-screen bg-white">
      <SearchHeader
        query={query}
        activeTab={tab}
        onSearch={handleSearch}
      />

      <main>
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex-1 max-w-2xl">
            {isLoading ? (
              <div className="flex justify-center py-12">
                <div className="w-8 h-8 border-4 border-[#1a73e8] border-t-transparent rounded-full animate-spin" />
              </div>
            ) : error ? (
              <div className="py-12 text-center">
                <p className="text-red-600">{error}</p>
              </div>
            ) : results ? (
              <div>
                <div className="flex items-center gap-2 text-xs text-[#70757a] mb-4">
                  <span>
                    About {(results.total_results ?? 0).toLocaleString()} {category} results ({(results.search_time_ms ?? 0).toFixed(2)} ms)
                    {results.cached && (
                      <span className="ml-1 text-[#188038]" title="Served from cache">
                        - cached
                      </span>
                    )}
                  </span>
                  <button
                    type="button"
                    onClick={() => doSearch(true)}
                    disabled={isRefetching}
                    className="p-1 text-[#5f6368] hover:text-[#1a73e8] hover:bg-[#f1f3f4] rounded transition-colors disabled:opacity-50"
                    title="Refresh results (bypass cache)"
                  >
                    <RefreshCw size={14} className={isRefetching ? 'animate-spin' : ''} />
                  </button>
                </div>

                {(results.results || []).map((result) => (
                  <SearchResult key={result.id} result={result} onRead={setReaderUrl} />
                ))}

                <Pagination page={page} totalPages={totalPages} onPageChange={handlePageChange} />
              </div>
            ) : (
              <div className="py-12 text-center">
                <p className="text-[#70757a]">Search for {category} results</p>
              </div>
            )}
          </div>
        </div>
      </main>

      {readerUrl && (
        <ReaderView url={readerUrl} onClose={() => setReaderUrl(null)} />
      )}
    </div>
  )
}
