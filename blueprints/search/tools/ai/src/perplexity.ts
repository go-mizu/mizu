import { ENDPOINTS, CHROME_HEADERS, API_VERSION, MODE_PAYLOAD, MODEL_PREFERENCE, CACHE_TTL } from './config'
import { Cache } from './cache'
import type { SSEPayload, SessionState, SearchResult, Citation, WebResult } from './types'

function uuid(): string {
  return crypto.randomUUID()
}

function extractDomain(url: string): string {
  try { return new URL(url).hostname.replace(/^www\./, '') } catch { return url }
}

function favicon(url: string): string {
  return `https://www.google.com/s2/favicons?domain=${extractDomain(url)}&sz=32`
}

/** Extract cookies from Set-Cookie headers into a single Cookie header string. */
function extractCookies(resp: Response, existing: string = ''): string {
  const cookies = new Map<string, string>()
  // Parse existing cookies
  if (existing) {
    for (const part of existing.split('; ')) {
      const eq = part.indexOf('=')
      if (eq > 0) cookies.set(part.slice(0, eq), part.slice(eq + 1))
    }
  }
  // Parse Set-Cookie headers
  const setCookies = resp.headers.getAll?.('set-cookie') ?? []
  for (const sc of setCookies) {
    const nameVal = sc.split(';')[0]
    const eq = nameVal.indexOf('=')
    if (eq > 0) cookies.set(nameVal.slice(0, eq).trim(), nameVal.slice(eq + 1).trim())
  }
  return Array.from(cookies.entries()).map(([k, v]) => `${k}=${v}`).join('; ')
}

/** Extract CSRF token from cookies or response. */
function extractCSRF(cookies: string, responseBody?: string): string {
  // Try response body JSON first
  if (responseBody) {
    try {
      const json = JSON.parse(responseBody)
      if (json.csrfToken) return json.csrfToken
    } catch { /* not JSON */ }
  }
  // Fallback: cookie "next-auth.csrf-token"
  const match = cookies.match(/next-auth\.csrf-token=([^;]+)/)
  if (match) {
    const val = match[1]
    // Split on %7C (URL-encoded pipe) or percent
    const parts = val.split('%')
    if (parts.length > 1) return parts[0]
    // Try URL-decode then split on |
    try {
      const decoded = decodeURIComponent(val)
      const pipeParts = decoded.split('|')
      if (pipeParts.length > 1) return pipeParts[0]
      return decoded
    } catch { return val }
  }
  return ''
}

/** Initialize a session: get cookies + CSRF token. */
export async function initSession(kv: KVNamespace): Promise<SessionState> {
  const cache = new Cache(kv)

  // Check cached session
  const cached = await cache.get<SessionState>('session:anon')
  if (cached?.csrfToken) return cached

  let cookies = ''

  // Step 1: GET /api/auth/session → establish cookies
  const sessionResp = await fetch(ENDPOINTS.session, {
    headers: { ...CHROME_HEADERS },
    redirect: 'manual',
  })
  cookies = extractCookies(sessionResp, cookies)

  // Step 2: GET /api/auth/csrf → get CSRF token
  const csrfResp = await fetch(ENDPOINTS.csrf, {
    headers: { ...CHROME_HEADERS, Cookie: cookies },
    redirect: 'manual',
  })
  cookies = extractCookies(csrfResp, cookies)
  const csrfBody = await csrfResp.text()
  const csrfToken = extractCSRF(cookies, csrfBody)

  if (!csrfToken) {
    throw new Error('Failed to extract CSRF token')
  }

  const session: SessionState = {
    csrfToken,
    cookies,
    createdAt: new Date().toISOString(),
  }

  await cache.set('session:anon', session, CACHE_TTL.session)
  return session
}

/** Execute an SSE search against Perplexity. */
export async function search(
  kv: KVNamespace,
  query: string,
  mode: string = 'auto',
  model: string = '',
  followUpUUID: string | null = null,
  sessionOverride?: SessionState,
): Promise<SearchResult> {
  // Validate mode/model
  const modeMap = MODEL_PREFERENCE[mode]
  if (!modeMap) throw new Error(`Invalid mode: ${mode}`)
  const modelPref = modeMap[model]
  if (modelPref === undefined) throw new Error(`Invalid model "${model}" for mode "${mode}"`)

  // Get session
  const session = sessionOverride || await initSession(kv)

  // Build payload
  const payload: SSEPayload = {
    query_str: query,
    params: {
      attachments: [],
      frontend_context_uuid: uuid(),
      frontend_uuid: uuid(),
      is_incognito: false,
      language: 'en-US',
      last_backend_uuid: followUpUUID,
      mode: MODE_PAYLOAD[mode] || 'concise',
      model_preference: modelPref,
      source: 'default',
      sources: ['web'],
      version: API_VERSION,
    },
  }

  const start = Date.now()

  const resp = await fetch(ENDPOINTS.sseAsk, {
    method: 'POST',
    headers: {
      ...CHROME_HEADERS,
      'Content-Type': 'application/json',
      'Cookie': session.cookies,
      'Accept': '*/*',
      'Sec-Fetch-Dest': 'empty',
      'Sec-Fetch-Mode': 'cors',
      'Sec-Fetch-Site': 'same-origin',
      'Origin': 'https://www.perplexity.ai',
      'Referer': 'https://www.perplexity.ai/',
    },
    body: JSON.stringify(payload),
  })

  if (!resp.ok) {
    const text = await resp.text()
    throw new Error(`SSE request failed: HTTP ${resp.status} — ${text.slice(0, 300)}`)
  }

  // Parse SSE stream
  const body = await resp.text()
  const lastChunk = parseSSEStream(body)

  if (!lastChunk) throw new Error('Empty SSE response')

  const result = extractSearchResult(lastChunk, query, mode, model)
  result.durationMs = Date.now() - start
  result.createdAt = new Date().toISOString()

  return result
}

/** Parse SSE stream text, return the last data chunk. */
function parseSSEStream(text: string): Record<string, unknown> | null {
  const chunks = text.split('\r\n\r\n')
  let last: Record<string, unknown> | null = null

  for (const chunk of chunks) {
    if (!chunk.trim()) continue
    if (chunk.startsWith('event: end_of_stream')) break

    // Handle "event: message\r\ndata: <json>" or just "data: <json>"
    let dataStr = ''
    if (chunk.includes('event: message')) {
      const dataIdx = chunk.indexOf('data: ')
      if (dataIdx >= 0) dataStr = chunk.slice(dataIdx + 6)
    } else if (chunk.startsWith('data: ')) {
      dataStr = chunk.slice(6)
    } else {
      continue
    }

    dataStr = dataStr.replace(/\r?\n$/g, '').trim()
    if (!dataStr) continue

    try {
      last = JSON.parse(dataStr) as Record<string, unknown>
    } catch { /* skip malformed */ }
  }

  return last
}

/** Extract structured SearchResult from the final SSE chunk. */
function extractSearchResult(
  data: Record<string, unknown>,
  query: string,
  mode: string,
  model: string,
): SearchResult {
  const result: SearchResult = {
    query: (data.query_str as string) || query,
    answer: '',
    citations: [],
    webResults: [],
    relatedQueries: [],
    backendUUID: (data.backend_uuid as string) || '',
    mode,
    model,
    durationMs: 0,
    createdAt: '',
  }

  // Parse the `text` field — can be string (JSON), object, or boolean
  const textField = data.text
  if (typeof textField === 'string') {
    parseTextContent(textField, result)
  } else if (typeof textField === 'object' && textField !== null && typeof textField !== 'boolean') {
    extractFromObject(textField as Record<string, unknown>, result)
  }

  // Fallback: extract from top-level data
  if (!result.answer) {
    extractAnswer(data, result)
  }

  // Related queries from top-level
  if (!result.relatedQueries.length && Array.isArray(data.related_queries)) {
    result.relatedQueries = (data.related_queries as unknown[]).filter(q => typeof q === 'string') as string[]
  }

  // Web results from top-level
  if (!result.webResults.length && Array.isArray(data.web_results)) {
    result.webResults = parseWebResults(data.web_results as unknown[])
  }

  // Build citations from web results if empty
  if (!result.citations.length && result.webResults.length) {
    result.citations = result.webResults.map(w => ({
      url: w.url,
      title: w.name || extractDomain(w.url),
      snippet: w.snippet || '',
      date: w.date,
      domain: extractDomain(w.url),
      favicon: favicon(w.url),
    }))
  }

  return result
}

function parseTextContent(textStr: string, result: SearchResult): void {
  // Try parsing as JSON
  let parsed: unknown
  try {
    parsed = JSON.parse(textStr)
  } catch {
    result.answer = textStr
    return
  }

  if (Array.isArray(parsed)) {
    // Step array format
    parseSteps(parsed, result)
  } else if (typeof parsed === 'object' && parsed !== null) {
    extractFromObject(parsed as Record<string, unknown>, result)
  }
}

function parseSteps(steps: unknown[], result: SearchResult): void {
  for (const step of steps) {
    if (typeof step !== 'object' || step === null) continue
    const s = step as Record<string, unknown>
    if (s.step_type !== 'FINAL') continue
    const content = s.content
    if (typeof content !== 'object' || content === null) continue
    const c = content as Record<string, unknown>
    const answerRaw = c.answer
    if (answerRaw === undefined) continue

    if (typeof answerRaw === 'string') {
      try {
        const parsed = JSON.parse(answerRaw)
        if (typeof parsed === 'object' && parsed !== null) {
          extractFromObject(parsed as Record<string, unknown>, result)
          return
        }
      } catch {
        result.answer = answerRaw
      }
    } else if (typeof answerRaw === 'object' && answerRaw !== null) {
      extractFromObject(answerRaw as Record<string, unknown>, result)
    }
    break
  }
}

function extractFromObject(data: Record<string, unknown>, result: SearchResult): void {
  extractAnswer(data, result)

  if (!result.webResults.length && Array.isArray(data.web_results)) {
    result.webResults = parseWebResults(data.web_results as unknown[])
  }
  if (!result.relatedQueries.length && Array.isArray(data.related_queries)) {
    result.relatedQueries = (data.related_queries as unknown[]).filter(q => typeof q === 'string') as string[]
  }
}

function extractAnswer(data: Record<string, unknown>, result: SearchResult): void {
  // Priority 1: structured_answer[0].text
  if (Array.isArray(data.structured_answer) && data.structured_answer.length > 0) {
    const first = data.structured_answer[0] as Record<string, unknown> | undefined
    if (first && typeof first.text === 'string' && first.text) {
      result.answer = first.text
      return
    }
  }

  // Priority 2: answer as object
  if (typeof data.answer === 'object' && data.answer !== null && !Array.isArray(data.answer)) {
    const answerMap = data.answer as Record<string, unknown>
    if (typeof answerMap.answer === 'string' && answerMap.answer) {
      result.answer = answerMap.answer
    }
    if (Array.isArray(answerMap.web_results) && !result.webResults.length) {
      result.webResults = parseWebResults(answerMap.web_results as unknown[])
    }
    if (Array.isArray(answerMap.related_queries) && !result.relatedQueries.length) {
      result.relatedQueries = (answerMap.related_queries as unknown[]).filter(q => typeof q === 'string') as string[]
    }
    // Nested structured_answer
    if (Array.isArray(answerMap.structured_answer) && answerMap.structured_answer.length > 0) {
      const first = answerMap.structured_answer[0] as Record<string, unknown> | undefined
      if (first && typeof first.text === 'string' && first.text) {
        result.answer = first.text
      }
    }
    return
  }

  // Priority 3: answer as JSON string
  if (typeof data.answer === 'string' && data.answer) {
    try {
      const parsed = JSON.parse(data.answer)
      if (typeof parsed === 'object' && parsed !== null) {
        extractAnswer(parsed as Record<string, unknown>, result)
        if (Array.isArray(parsed.web_results) && !result.webResults.length) {
          result.webResults = parseWebResults(parsed.web_results)
        }
        return
      }
    } catch {
      // Not JSON — use as plain text
      if (!result.answer) result.answer = data.answer as string
    }
  }
}

function parseWebResults(raw: unknown[]): WebResult[] {
  return raw
    .filter((item): item is Record<string, unknown> => typeof item === 'object' && item !== null)
    .map(m => ({
      name: (m.name as string) || '',
      url: (m.url as string) || '',
      snippet: (m.snippet as string) || '',
      date: (m.timestamp as string) || (m.date as string) || undefined,
    }))
}
