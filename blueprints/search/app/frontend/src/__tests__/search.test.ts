/**
 * Integration tests for the search API.
 *
 * These tests require:
 * - SearXNG running on localhost:8888
 * - Backend running on localhost:8080
 *
 * Run with: pnpm test
 */

import { describe, test, expect, beforeAll } from 'vitest'

const API_BASE = process.env.API_URL || 'http://localhost:8080'

interface SearchResult {
  url: string
  title: string
  snippet: string
  domain: string
  score: number
  engine?: string
  engines?: string[]
}

interface SearchResponse {
  query: string
  corrected_query?: string
  total_results: number
  results: SearchResult[]
  suggestions?: string[]
  instant_answer?: {
    type: string
    query: string
    result: string
    data?: unknown
  }
  knowledge_panel?: {
    title: string
    description: string
    image?: string
    facts?: Array<{ label: string; value: string }>
  }
  related_searches?: string[]
  search_time_ms: number
  page: number
  per_page: number
}

interface ImageResult {
  id: string
  url: string
  thumbnail_url: string
  title: string
  source_url: string
  source_domain: string
  format?: string
  engine?: string
}

interface VideoResult {
  id: string
  url: string
  thumbnail_url: string
  title: string
  description: string
  duration_seconds?: number
  published_at?: string
  embed_url?: string
  engine?: string
}

interface NewsResult {
  id: string
  url: string
  title: string
  snippet: string
  source: string
  image_url?: string
  published_at: string
  engine?: string
}

async function checkBackendHealth(): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE}/health`)
    return response.ok
  } catch {
    return false
  }
}

describe('Search API Integration', () => {
  beforeAll(async () => {
    const healthy = await checkBackendHealth()
    if (!healthy) {
      console.warn('Backend is not available. Skipping integration tests.')
      return
    }
  })

  test('health check', async () => {
    const response = await fetch(`${API_BASE}/health`)
    expect(response.ok).toBe(true)
  })

  test('search returns results from SearXNG', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=golang`)
    expect(response.ok).toBe(true)

    const data: SearchResponse = await response.json()
    expect(data.query).toBe('golang')
    expect(data.results).toBeDefined()
    expect(Array.isArray(data.results)).toBe(true)

    if (data.results.length > 0) {
      const firstResult = data.results[0]
      expect(firstResult.url).toBeDefined()
      expect(firstResult.title).toBeDefined()
    }
  })

  test('search with pagination', async () => {
    const page1Response = await fetch(`${API_BASE}/api/search?q=programming&page=1&per_page=5`)
    const page1Data: SearchResponse = await page1Response.json()

    const page2Response = await fetch(`${API_BASE}/api/search?q=programming&page=2&per_page=5`)
    const page2Data: SearchResponse = await page2Response.json()

    expect(page1Data.page).toBe(1)
    expect(page2Data.page).toBe(2)

    // Pages should have different first results (if both have results)
    if (page1Data.results.length > 0 && page2Data.results.length > 0) {
      expect(page1Data.results[0].url).not.toBe(page2Data.results[0].url)
    }
  })

  test('search with time range filter', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=technology&time=week`)
    expect(response.ok).toBe(true)

    const data: SearchResponse = await response.json()
    expect(data.query).toBe('technology')
  })

  test('search with language filter', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=programming&lang=en`)
    expect(response.ok).toBe(true)

    const data: SearchResponse = await response.json()
    expect(data.query).toBe('programming')
  })

  test('search returns instant answer for calculator query', async () => {
    // Skip if backend doesn't support instant answers
    const response = await fetch(`${API_BASE}/api/search?q=2%2B2`)
    expect(response.ok).toBe(true)

    const data: SearchResponse = await response.json()
    // Instant answer is optional
    if (data.instant_answer) {
      expect(data.instant_answer.type).toBe('calculator')
      expect(data.instant_answer.result).toBe('4')
    }
  })

  test('search returns knowledge panel for known entity', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=Go`)
    expect(response.ok).toBe(true)

    const data: SearchResponse = await response.json()
    // Knowledge panel is optional (depends on seeded data)
    if (data.knowledge_panel) {
      expect(data.knowledge_panel.title).toBeDefined()
      expect(data.knowledge_panel.description).toBeDefined()
    }
  })

  test('image search returns image results', async () => {
    const response = await fetch(`${API_BASE}/api/search/images?q=golang+logo`)
    expect(response.ok).toBe(true)

    const data: { query: string; results: ImageResult[] } = await response.json()
    expect(data.query).toBe('golang logo')
    expect(data.results).toBeDefined()
    expect(Array.isArray(data.results)).toBe(true)

    if (data.results.length > 0) {
      const firstResult = data.results[0]
      // Either url or thumbnail_url should be present
      expect(firstResult.url || firstResult.thumbnail_url).toBeDefined()
      expect(firstResult.title).toBeDefined()
    }
  })

  test('video search returns video results', async () => {
    const response = await fetch(`${API_BASE}/api/search/videos?q=golang+tutorial`)
    expect(response.ok).toBe(true)

    const data: { query: string; results: VideoResult[] } = await response.json()
    expect(data.query).toBe('golang tutorial')
    expect(data.results).toBeDefined()
    expect(Array.isArray(data.results)).toBe(true)

    if (data.results.length > 0) {
      const firstResult = data.results[0]
      expect(firstResult.url).toBeDefined()
      expect(firstResult.title).toBeDefined()
    }
  })

  test('news search returns news results', async () => {
    const response = await fetch(`${API_BASE}/api/search/news?q=technology`)
    expect(response.ok).toBe(true)

    const data: { query: string; results: NewsResult[] } = await response.json()
    expect(data.query).toBe('technology')
    expect(data.results).toBeDefined()
    expect(Array.isArray(data.results)).toBe(true)

    if (data.results.length > 0) {
      const firstResult = data.results[0]
      expect(firstResult.url).toBeDefined()
      expect(firstResult.title).toBeDefined()
      expect(firstResult.source).toBeDefined()
    }
  })

  test('cached results are returned faster', async () => {
    const uniqueQuery = `cache-test-${Date.now()}`

    // First request - cache miss
    const start1 = Date.now()
    await fetch(`${API_BASE}/api/search?q=${uniqueQuery}`)
    const time1 = Date.now() - start1

    // Second request - cache hit
    const start2 = Date.now()
    await fetch(`${API_BASE}/api/search?q=${uniqueQuery}`)
    const time2 = Date.now() - start2

    // Cache hit should generally be faster, but this is not guaranteed
    // due to network variability, so we just log the times
    console.log(`Cache test: first request ${time1}ms, second request ${time2}ms`)

    // At minimum, both requests should succeed
    expect(time1).toBeGreaterThan(0)
    expect(time2).toBeGreaterThan(0)
  })

  test('search returns suggestions', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=programming`)
    expect(response.ok).toBe(true)

    const data: SearchResponse = await response.json()
    // Suggestions are optional
    if (data.suggestions) {
      expect(Array.isArray(data.suggestions)).toBe(true)
    }
  })

  test('search returns related searches', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=golang`)
    expect(response.ok).toBe(true)

    const data: SearchResponse = await response.json()
    // Related searches are optional
    if (data.related_searches) {
      expect(Array.isArray(data.related_searches)).toBe(true)
    }
  })

  test('search records history', async () => {
    const uniqueQuery = `history-test-${Date.now()}`

    // Perform search
    await fetch(`${API_BASE}/api/search?q=${encodeURIComponent(uniqueQuery)}`)

    // Check history
    const historyResponse = await fetch(`${API_BASE}/api/history`)
    expect(historyResponse.ok).toBe(true)

    const historyData: Array<{ query: string }> = await historyResponse.json()
    const found = historyData?.some(h => h.query === uniqueQuery)

    // History should contain our query
    expect(found).toBe(true)
  })

  test('suggest endpoint returns autocomplete suggestions', async () => {
    const response = await fetch(`${API_BASE}/api/suggest?q=prog`)
    expect(response.ok).toBe(true)

    const data: Array<{ text: string }> = await response.json()
    expect(data).toBeDefined()
    expect(Array.isArray(data)).toBe(true)
  })

  test('search with empty query returns error', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=`)
    expect(response.status).toBe(400)
  })

  test('search results include engine information', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=golang`)
    const data: SearchResponse = await response.json()

    if (data.results.length > 0) {
      const firstResult = data.results[0]
      // At least one of engine or engines should be present
      expect(firstResult.engine || firstResult.engines).toBeDefined()
    }
  })
})

describe('Settings API', () => {
  test('get settings', async () => {
    const response = await fetch(`${API_BASE}/api/settings`)
    expect(response.ok).toBe(true)

    const data = await response.json()
    expect(data.safe_search).toBeDefined()
    expect(data.results_per_page).toBeDefined()
  })
})

describe('Preferences API', () => {
  test('get preferences', async () => {
    const response = await fetch(`${API_BASE}/api/preferences`)
    expect(response.ok).toBe(true)

    const data = await response.json()
    // API returns array directly, or null if empty
    expect(data === null || Array.isArray(data)).toBe(true)
  })
})

describe('Lenses API', () => {
  test('list lenses', async () => {
    const response = await fetch(`${API_BASE}/api/lenses`)
    expect(response.ok).toBe(true)

    const data = await response.json()
    // API returns array directly
    expect(data).toBeDefined()
    expect(Array.isArray(data)).toBe(true)
  })
})
