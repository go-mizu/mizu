import { useEffect, useState } from 'react'
import { useSearchParams, Link } from 'react-router-dom'
import {
  Container,
  Group,
  Text,
  Loader,
  Stack,
  Pagination,
  Select,
  Badge,
  Anchor,
  ActionIcon,
} from '@mantine/core'
import { IconSettings, IconPhoto, IconVideo, IconNews } from '@tabler/icons-react'
import { SearchBox } from '../components/SearchBox'
import { SearchResult } from '../components/SearchResult'
import { InstantAnswer } from '../components/InstantAnswer'
import { KnowledgePanel } from '../components/KnowledgePanel'
import { searchApi } from '../api/search'
import { useSearchStore } from '../stores/searchStore'
import type { SearchResponse } from '../types'

type SearchTab = 'all' | 'images' | 'videos' | 'news'

export default function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const query = searchParams.get('q') || ''
  const page = parseInt(searchParams.get('page') || '1', 10)
  const timeFilter = searchParams.get('time') || ''
  const [activeTab, setActiveTab] = useState<SearchTab>('all')

  const [results, setResults] = useState<SearchResponse | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const { settings, addRecentSearch } = useSearchStore()

  useEffect(() => {
    if (!query) return

    const performSearch = async () => {
      setIsLoading(true)
      setError(null)

      try {
        const response = await searchApi.search(query, {
          page,
          per_page: settings.results_per_page,
          time: timeFilter,
          safe: settings.safe_search,
          region: settings.region,
          lang: settings.language,
        })
        setResults(response)
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

  const handleTimeFilter = (value: string | null) => {
    if (value) {
      setSearchParams({ q: query, time: value })
    } else {
      setSearchParams({ q: query })
    }
  }

  const totalPages = results ? Math.ceil(results.total_results / settings.results_per_page) : 0

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="sticky top-0 bg-white border-b z-50">
        <Container size="xl" className="py-3">
          <Group>
            {/* Logo */}
            <Link to="/">
              <Text
                size="28px"
                fw={700}
                style={{
                  background: 'linear-gradient(90deg, #4285F4, #EA4335, #FBBC05, #34A853)',
                  WebkitBackgroundClip: 'text',
                  WebkitTextFillColor: 'transparent',
                }}
              >
                Search
              </Text>
            </Link>

            {/* Search box */}
            <div className="flex-1 max-w-xl">
              <SearchBox
                initialValue={query}
                size="sm"
                onSearch={handleSearch}
              />
            </div>

            {/* Settings */}
            <Link to="/settings">
              <ActionIcon variant="subtle" color="gray" size="lg">
                <IconSettings size={20} />
              </ActionIcon>
            </Link>
          </Group>

          {/* Tabs */}
          <Group gap="xs" mt="md">
            <button
              className={`search-tab ${activeTab === 'all' ? 'active' : ''}`}
              onClick={() => setActiveTab('all')}
            >
              All
            </button>
            <button
              className={`search-tab ${activeTab === 'images' ? 'active' : ''}`}
              onClick={() => setActiveTab('images')}
            >
              <Group gap={4}>
                <IconPhoto size={16} />
                Images
              </Group>
            </button>
            <button
              className={`search-tab ${activeTab === 'videos' ? 'active' : ''}`}
              onClick={() => setActiveTab('videos')}
            >
              <Group gap={4}>
                <IconVideo size={16} />
                Videos
              </Group>
            </button>
            <button
              className={`search-tab ${activeTab === 'news' ? 'active' : ''}`}
              onClick={() => setActiveTab('news')}
            >
              <Group gap={4}>
                <IconNews size={16} />
                News
              </Group>
            </button>

            <div className="ml-auto">
              <Select
                placeholder="Any time"
                size="xs"
                value={timeFilter || null}
                onChange={handleTimeFilter}
                data={[
                  { value: 'day', label: 'Past 24 hours' },
                  { value: 'week', label: 'Past week' },
                  { value: 'month', label: 'Past month' },
                  { value: 'year', label: 'Past year' },
                ]}
                clearable
                w={140}
              />
            </div>
          </Group>
        </Container>
      </header>

      {/* Main content */}
      <main>
        <Container size="xl" className="py-4">
          <div className="flex gap-8">
            {/* Results */}
            <div className="flex-1 max-w-2xl">
              {isLoading ? (
                <div className="flex justify-center py-12">
                  <Loader />
                </div>
              ) : error ? (
                <div className="py-12 text-center">
                  <Text c="red">{error}</Text>
                </div>
              ) : results ? (
                <Stack gap={0}>
                  {/* Stats */}
                  <Text size="xs" c="dimmed" mb="md">
                    About {results.total_results.toLocaleString()} results ({results.search_time_ms.toFixed(2)} ms)
                  </Text>

                  {/* Corrected query */}
                  {results.corrected_query && (
                    <Text size="sm" mb="md">
                      Showing results for{' '}
                      <Anchor
                        onClick={() => handleSearch(results.corrected_query!)}
                        fw={500}
                      >
                        {results.corrected_query}
                      </Anchor>
                    </Text>
                  )}

                  {/* Instant answer */}
                  {results.instant_answer && (
                    <div className="mb-4">
                      <InstantAnswer answer={results.instant_answer} />
                    </div>
                  )}

                  {/* Results list */}
                  {results.results.map((result) => (
                    <SearchResult key={result.id} result={result} />
                  ))}

                  {/* Related searches */}
                  {results.related_searches && results.related_searches.length > 0 && (
                    <div className="mt-8 pt-4 border-t">
                      <Text size="sm" fw={500} mb="sm">
                        Related searches
                      </Text>
                      <Group gap="xs">
                        {results.related_searches.map((search) => (
                          <Badge
                            key={search}
                            variant="light"
                            className="cursor-pointer"
                            onClick={() => handleSearch(search)}
                          >
                            {search}
                          </Badge>
                        ))}
                      </Group>
                    </div>
                  )}

                  {/* Pagination */}
                  {totalPages > 1 && (
                    <div className="flex justify-center mt-8 pt-4 border-t">
                      <Pagination
                        total={totalPages}
                        value={page}
                        onChange={handlePageChange}
                      />
                    </div>
                  )}
                </Stack>
              ) : (
                <div className="py-12 text-center">
                  <Text c="dimmed">Enter a search query</Text>
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
        </Container>
      </main>
    </div>
  )
}
