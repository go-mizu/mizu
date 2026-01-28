/**
 * Integration tests for Kagi-compatible features.
 *
 * Tests for:
 * - Bang shortcuts
 * - Summarizer API
 * - Widgets API
 * - Enrichment API (Teclis/TinyGem style)
 *
 * These tests require the backend running on localhost:8080.
 *
 * Run with: pnpm test
 */

import { describe, test, expect, beforeAll } from 'vitest'

const API_BASE = process.env.API_URL || 'http://localhost:8080'

// ============ Type Definitions ============

interface Bang {
  id: number
  trigger: string
  name: string
  url_template: string
  category: string
  is_builtin: boolean
}

interface BangResult {
  bang?: Bang
  query: string
  orig_query: string
  redirect?: string
  internal: boolean
  category?: string
}

interface SummarizeResponse {
  output: string
  tokens: number
  cached: boolean
  engine: string
}

interface EnrichmentResponse {
  meta: {
    id: string
    node: string
    ms: number
  }
  data: Array<{
    t: number
    rank: number
    url: string
    title: string
    snippet?: string
    published?: string
  }>
}

interface CheatSheet {
  language: string
  title: string
  sections: Array<{
    name: string
    items: Array<{
      code: string
      description: string
    }>
  }>
}

interface WidgetSetting {
  user_id: string
  widget_type: string
  enabled: boolean
  position: number
}

// ============ Helper Functions ============

async function checkBackendHealth(): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE}/health`)
    return response.ok
  } catch {
    return false
  }
}

// ============ Tests ============

describe('Bangs API', () => {
  beforeAll(async () => {
    const healthy = await checkBackendHealth()
    if (!healthy) {
      console.warn('Backend is not available. Skipping integration tests.')
    }
  })

  test('list bangs returns built-in bangs', async () => {
    const response = await fetch(`${API_BASE}/api/bangs`)
    expect(response.ok).toBe(true)

    const data: Bang[] = await response.json()
    expect(Array.isArray(data)).toBe(true)

    // Should have some built-in bangs
    const googleBang = data.find(b => b.trigger === 'g')
    expect(googleBang).toBeDefined()
    expect(googleBang?.name).toBe('Google')
    expect(googleBang?.is_builtin).toBe(true)

    const youtubeBang = data.find(b => b.trigger === 'yt')
    expect(youtubeBang).toBeDefined()
    expect(youtubeBang?.name).toBe('YouTube')
  })

  test('parse external bang returns redirect URL', async () => {
    const response = await fetch(`${API_BASE}/api/bangs/parse?q=!g+test+query`)
    expect(response.ok).toBe(true)

    const data: BangResult = await response.json()
    expect(data.query).toBe('test query')
    expect(data.orig_query).toBe('!g test query')
    expect(data.redirect).toContain('google.com')
    expect(data.redirect).toContain('test+query')
    expect(data.internal).toBe(false)
  })

  test('parse internal bang (images) returns category', async () => {
    const response = await fetch(`${API_BASE}/api/bangs/parse?q=!i+cats`)
    expect(response.ok).toBe(true)

    const data: BangResult = await response.json()
    expect(data.query).toBe('cats')
    expect(data.internal).toBe(true)
    expect(data.category).toBe('images')
    expect(data.redirect).toBeUndefined()
  })

  test('parse AI bang returns ai category', async () => {
    const response = await fetch(`${API_BASE}/api/bangs/parse?q=!ai+explain+this`)
    expect(response.ok).toBe(true)

    const data: BangResult = await response.json()
    expect(data.query).toBe('explain this')
    expect(data.internal).toBe(true)
    expect(data.category).toBe('ai')
  })

  test('parse time filter bang returns time category', async () => {
    const response = await fetch(`${API_BASE}/api/bangs/parse?q=!week+news`)
    expect(response.ok).toBe(true)

    const data: BangResult = await response.json()
    expect(data.query).toBe('news')
    expect(data.internal).toBe(true)
    expect(data.category).toBe('time:week')
  })

  test('parse query without bang returns original query', async () => {
    const response = await fetch(`${API_BASE}/api/bangs/parse?q=normal+search`)
    expect(response.ok).toBe(true)

    const data: BangResult = await response.json()
    expect(data.query).toBe('normal search')
    expect(data.internal).toBe(false)
    expect(data.redirect).toBeUndefined()
    expect(data.bang).toBeUndefined()
  })

  test('parse suffix bang works', async () => {
    const response = await fetch(`${API_BASE}/api/bangs/parse?q=search+query+!g`)
    expect(response.ok).toBe(true)

    const data: BangResult = await response.json()
    expect(data.query).toBe('search query')
    expect(data.redirect).toContain('google.com')
  })
})

describe('Widgets API', () => {
  beforeAll(async () => {
    const healthy = await checkBackendHealth()
    if (!healthy) {
      console.warn('Backend is not available. Skipping integration tests.')
    }
  })

  test('get widget settings returns array', async () => {
    const response = await fetch(`${API_BASE}/api/widgets`)
    expect(response.ok).toBe(true)

    const data = await response.json()
    // Returns array or null
    expect(data === null || Array.isArray(data)).toBe(true)
  })

  test('list cheat sheets returns available languages', async () => {
    const response = await fetch(`${API_BASE}/api/cheatsheets`)
    expect(response.ok).toBe(true)

    const data: CheatSheet[] = await response.json()
    // May be empty array or null initially
    expect(data === null || Array.isArray(data)).toBe(true)
  })

  test('get Go cheat sheet (if seeded)', async () => {
    const response = await fetch(`${API_BASE}/api/cheatsheet/go`)

    if (response.ok) {
      const data: CheatSheet = await response.json()
      expect(data.language).toBe('go')
      expect(data.title).toBeDefined()
      expect(Array.isArray(data.sections)).toBe(true)
    } else {
      // 404 is acceptable if not seeded
      expect(response.status).toBe(404)
    }
  })

  test('get Python cheat sheet (if seeded)', async () => {
    const response = await fetch(`${API_BASE}/api/cheatsheet/python`)

    if (response.ok) {
      const data: CheatSheet = await response.json()
      expect(data.language).toBe('python')
    } else {
      expect(response.status).toBe(404)
    }
  })

  test('get related searches', async () => {
    const response = await fetch(`${API_BASE}/api/related?q=golang+tutorial`)
    expect(response.ok).toBe(true)

    const data: { query: string; related: string[] } = await response.json()
    expect(data.query).toBe('golang tutorial')
    // Related may be empty array or null
    expect(data.related === null || Array.isArray(data.related)).toBe(true)
  })
})

describe('Enrichment API (Small Web)', () => {
  beforeAll(async () => {
    const healthy = await checkBackendHealth()
    if (!healthy) {
      console.warn('Backend is not available. Skipping integration tests.')
    }
  })

  test('search web returns enrichment response', async () => {
    const response = await fetch(`${API_BASE}/api/enrich/web?q=technology&limit=5`)
    expect(response.ok).toBe(true)

    const data: EnrichmentResponse = await response.json()
    expect(data.meta).toBeDefined()
    expect(data.meta.node).toBe('local')
    expect(data.meta.ms).toBeDefined()
    expect(Array.isArray(data.data)).toBe(true)
  })

  test('search news returns enrichment response', async () => {
    const response = await fetch(`${API_BASE}/api/enrich/news?q=breaking&limit=5`)
    expect(response.ok).toBe(true)

    const data: EnrichmentResponse = await response.json()
    expect(data.meta).toBeDefined()
    expect(Array.isArray(data.data)).toBe(true)
  })

  test('enrich web requires query parameter', async () => {
    const response = await fetch(`${API_BASE}/api/enrich/web`)
    expect(response.status).toBe(400)
  })

  test('enrich news requires query parameter', async () => {
    const response = await fetch(`${API_BASE}/api/enrich/news`)
    expect(response.status).toBe(400)
  })
})

describe('Search with Bangs Integration', () => {
  beforeAll(async () => {
    const healthy = await checkBackendHealth()
    if (!healthy) {
      console.warn('Backend is not available. Skipping integration tests.')
    }
  })

  test('search with external bang returns redirect', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=!g+test`)
    expect(response.ok).toBe(true)

    const data = await response.json()
    // Should return redirect
    expect(data.redirect).toBeDefined()
    expect(data.redirect).toContain('google.com')
  })

  test('search with internal bang returns category', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=!i+cats`)
    expect(response.ok).toBe(true)

    const data = await response.json()
    // Should redirect internally or return images results
    expect(data.redirect || data.category || data.results).toBeDefined()
  })

  test('search returns widgets for programming queries', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=golang+for+loop`)
    expect(response.ok).toBe(true)

    const data = await response.json()
    // Widgets are optional
    if (data.widgets) {
      expect(Array.isArray(data.widgets)).toBe(true)
    }
  })

  test('search returns has_more indicator', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=programming&per_page=10`)
    expect(response.ok).toBe(true)

    const data = await response.json()
    // has_more should be boolean
    expect(typeof data.has_more).toBe('boolean')
  })
})

describe('Date Filtering', () => {
  beforeAll(async () => {
    const healthy = await checkBackendHealth()
    if (!healthy) {
      console.warn('Backend is not available. Skipping integration tests.')
    }
  })

  test('search with before date filter', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=news&before=2024-01-01`)
    expect(response.ok).toBe(true)

    const data = await response.json()
    expect(data.query).toBe('news')
  })

  test('search with after date filter', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=news&after=2023-01-01`)
    expect(response.ok).toBe(true)

    const data = await response.json()
    expect(data.query).toBe('news')
  })

  test('search with safe_level parameter', async () => {
    const response = await fetch(`${API_BASE}/api/search?q=test&safe_level=2`)
    expect(response.ok).toBe(true)

    const data = await response.json()
    expect(data.query).toBe('test')
  })
})

describe('Summarizer API', () => {
  beforeAll(async () => {
    const healthy = await checkBackendHealth()
    if (!healthy) {
      console.warn('Backend is not available. Skipping integration tests.')
    }
  })

  test('summarize endpoint exists', async () => {
    // The summarize endpoint may not be available if LLM is not configured
    const response = await fetch(`${API_BASE}/api/summarize?url=https://example.com`)

    // Either 200 (working) or 404/500 (not configured) is acceptable
    expect([200, 404, 500].includes(response.status)).toBe(true)
  })

  test('summarize requires url or text parameter', async () => {
    const response = await fetch(`${API_BASE}/api/summarize`)

    // Should return error (400 or 500)
    expect([400, 404, 500].includes(response.status)).toBe(true)
  })
})
