import { ENDPOINTS, CHROME_HEADERS, API_VERSION, MODE_PAYLOAD, MODEL_PREFERENCE, CACHE_TTL } from './config'
import { Cache } from './cache'
import type { SSEPayload, SessionState, SearchResult, Citation, WebResult, MediaItem } from './types'

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
  if (existing) {
    for (const part of existing.split('; ')) {
      const eq = part.indexOf('=')
      if (eq > 0) cookies.set(part.slice(0, eq), part.slice(eq + 1))
    }
  }
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
  if (responseBody) {
    try {
      const json = JSON.parse(responseBody)
      if (json.csrfToken) return json.csrfToken
    } catch { /* not JSON */ }
  }
  const match = cookies.match(/next-auth\.csrf-token=([^;]+)/)
  if (match) {
    const val = match[1]
    const parts = val.split('%')
    if (parts.length > 1) return parts[0]
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
  const cached = await cache.get<SessionState>('session:anon')
  if (cached?.csrfToken) return cached

  let cookies = ''

  const sessionResp = await fetch(ENDPOINTS.session, {
    headers: { ...CHROME_HEADERS },
    redirect: 'manual',
  })
  cookies = extractCookies(sessionResp, cookies)

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

function buildPayload(
  query: string,
  mode: string,
  model: string,
  followUpUUID: string | null,
): SSEPayload {
  const modeMap = MODEL_PREFERENCE[mode]
  if (!modeMap) throw new Error(`Invalid mode: ${mode}`)
  const modelPref = modeMap[model]
  if (modelPref === undefined) throw new Error(`Invalid model "${model}" for mode "${mode}"`)

  return {
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
}

function buildHeaders(session: SessionState): Record<string, string> {
  return {
    ...CHROME_HEADERS,
    'Content-Type': 'application/json',
    'Cookie': session.cookies,
    'Accept': '*/*',
    'Sec-Fetch-Dest': 'empty',
    'Sec-Fetch-Mode': 'cors',
    'Sec-Fetch-Site': 'same-origin',
    'Origin': 'https://www.perplexity.ai',
    'Referer': 'https://www.perplexity.ai/',
  }
}

/** Execute a search against Perplexity (non-streaming, reads full response). */
export async function search(
  kv: KVNamespace,
  query: string,
  mode: string = 'auto',
  model: string = '',
  followUpUUID: string | null = null,
  sessionOverride?: SessionState,
): Promise<SearchResult> {
  const payload = buildPayload(query, mode, model, followUpUUID)
  const session = sessionOverride || await initSession(kv)
  const start = Date.now()

  const resp = await fetch(ENDPOINTS.sseAsk, {
    method: 'POST',
    headers: buildHeaders(session),
    body: JSON.stringify(payload),
  })

  if (!resp.ok) {
    const text = await resp.text()
    throw new Error(`SSE request failed: HTTP ${resp.status} — ${text.slice(0, 300)}`)
  }

  const body = await resp.text()
  const lastChunk = parseSSEStream(body)

  if (!lastChunk) throw new Error('Empty SSE response')

  const result = extractSearchResult(lastChunk, query, mode, model)
  result.durationMs = Date.now() - start
  result.createdAt = new Date().toISOString()

  return result
}

/** Stream search: returns a ReadableStream of SSE events for client consumption. */
export function streamSearch(
  kv: KVNamespace,
  query: string,
  mode: string = 'auto',
  model: string = '',
  followUpUUID: string | null = null,
): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()

  function sendEvent(controller: ReadableStreamDefaultController, event: string, data: unknown): void {
    const json = JSON.stringify(data)
    controller.enqueue(encoder.encode(`event: ${event}\ndata: ${json}\n\n`))
  }

  return new ReadableStream({
    async start(controller) {
      try {
        const payload = buildPayload(query, mode, model, followUpUUID)
        const session = await initSession(kv)
        const start = Date.now()

        // Emit progress immediately so the client shows a searching indicator
        sendEvent(controller, 'progress', { status: 'searching', message: 'Searching the web...' })

        const resp = await fetch(ENDPOINTS.sseAsk, {
          method: 'POST',
          headers: buildHeaders(session),
          body: JSON.stringify(payload),
        })

        if (!resp.ok) {
          const text = await resp.text()
          sendEvent(controller, 'error', { message: `HTTP ${resp.status}: ${text.slice(0, 300)}` })
          controller.close()
          return
        }

        if (!resp.body) {
          sendEvent(controller, 'error', { message: 'No response body from upstream' })
          controller.close()
          return
        }

        // TRUE STREAMING: read Perplexity response body incrementally
        const reader = resp.body.getReader()
        const decoder = new TextDecoder()
        let sseBuffer = ''
        let lastData: Record<string, unknown> | null = null
        let sentSources = false
        let lastAnswerLen = 0

        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          sseBuffer += decoder.decode(value, { stream: true })

          // Perplexity SSE chunks are delimited by \r\n\r\n
          const parts = sseBuffer.split('\r\n\r\n')
          sseBuffer = parts.pop() || '' // Keep incomplete chunk in buffer

          for (const chunk of parts) {
            if (!chunk.trim()) continue
            if (chunk.startsWith('event: end_of_stream')) continue

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

            let data: Record<string, unknown>
            try {
              data = JSON.parse(dataStr)
            } catch { continue }

            lastData = data

            // Emit sources on first chunk that has web_results
            if (!sentSources) {
              const webResults = extractWebResultsFromData(data)
              if (webResults.length > 0) {
                const citations = webResults.map(w => ({
                  url: w.url,
                  title: w.name || extractDomain(w.url),
                  snippet: w.snippet || '',
                  date: w.date,
                  domain: extractDomain(w.url),
                  favicon: favicon(w.url),
                }))
                sendEvent(controller, 'sources', { citations, webResults })
                sentSources = true
              }
            }

            // Emit answer chunk deltas
            const answer = extractAnswerText(data)
            if (answer && answer.length > lastAnswerLen) {
              const delta = answer.slice(lastAnswerLen)
              sendEvent(controller, 'chunk', { delta, full: answer })
              lastAnswerLen = answer.length
            }
          }
        }

        // Final result
        if (lastData) {
          const result = extractSearchResult(lastData, query, mode, model)
          result.durationMs = Date.now() - start
          result.createdAt = new Date().toISOString()

          // Emit media
          if (result.images.length > 0 || result.videos.length > 0) {
            sendEvent(controller, 'media', { images: result.images, videos: result.videos })
          }

          // Emit related
          if (result.relatedQueries.length > 0) {
            sendEvent(controller, 'related', { queries: result.relatedQueries })
          }

          // Emit done
          sendEvent(controller, 'done', { result })
        } else {
          sendEvent(controller, 'error', { message: 'Empty SSE response' })
        }

        controller.close()
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e)
        sendEvent(controller, 'error', { message: msg })
        controller.close()
      }
    },
  })
}

/** Extract just the answer text from a data chunk (for streaming deltas). */
function extractAnswerText(data: Record<string, unknown>): string {
  const textField = data.text
  if (typeof textField === 'string') {
    try {
      const parsed = JSON.parse(textField)
      if (Array.isArray(parsed)) {
        for (const step of parsed) {
          if (typeof step === 'object' && step !== null) {
            const s = step as Record<string, unknown>
            if (s.step_type === 'FINAL') {
              const content = s.content as Record<string, unknown> | undefined
              if (content) {
                const a = content.answer
                if (typeof a === 'string') return a
              }
            }
          }
        }
        return ''
      }
      if (typeof parsed === 'object' && parsed !== null) {
        const obj = parsed as Record<string, unknown>
        if (typeof obj.answer === 'string') return obj.answer
        if (Array.isArray(obj.structured_answer) && obj.structured_answer.length > 0) {
          const first = obj.structured_answer[0] as Record<string, unknown>
          if (typeof first?.text === 'string') return first.text
        }
      }
      return ''
    } catch {
      return textField
    }
  }
  // Fallback: check answer field directly
  if (typeof data.answer === 'string') {
    try {
      const parsed = JSON.parse(data.answer)
      if (typeof parsed === 'object' && parsed !== null && typeof parsed.answer === 'string') return parsed.answer
    } catch {
      return data.answer as string
    }
  }
  if (typeof data.answer === 'object' && data.answer !== null) {
    const a = data.answer as Record<string, unknown>
    if (typeof a.answer === 'string') return a.answer
  }
  return ''
}

/** Extract web results from any level of a data chunk. */
function extractWebResultsFromData(data: Record<string, unknown>): WebResult[] {
  if (Array.isArray(data.web_results)) {
    return parseWebResults(data.web_results as unknown[])
  }
  // Check nested answer object
  if (typeof data.answer === 'object' && data.answer !== null) {
    const a = data.answer as Record<string, unknown>
    if (Array.isArray(a.web_results)) {
      return parseWebResults(a.web_results as unknown[])
    }
  }
  // Check text field parsed as JSON
  if (typeof data.text === 'string') {
    try {
      const parsed = JSON.parse(data.text)
      if (typeof parsed === 'object' && parsed !== null && Array.isArray(parsed.web_results)) {
        return parseWebResults(parsed.web_results)
      }
    } catch { /* not JSON */ }
  }
  return []
}

/** Parse SSE stream text, return the last data chunk. */
function parseSSEStream(text: string): Record<string, unknown> | null {
  const chunks = text.split('\r\n\r\n')
  let last: Record<string, unknown> | null = null

  for (const chunk of chunks) {
    if (!chunk.trim()) continue
    if (chunk.startsWith('event: end_of_stream')) break

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
    images: [],
    videos: [],
    backendUUID: (data.backend_uuid as string) || '',
    mode,
    model,
    durationMs: 0,
    createdAt: '',
  }

  // Parse the `text` field
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

  // Extract media
  extractMedia(data, result)

  return result
}

/** Extract images and videos from the SSE data. */
function extractMedia(data: Record<string, unknown>, result: SearchResult): void {
  // Images from media_items or image_results
  const imageArrays = [data.media_items, data.image_results, data.images]
  for (const arr of imageArrays) {
    if (Array.isArray(arr) && arr.length > 0) {
      for (const item of arr) {
        if (typeof item !== 'object' || item === null) continue
        const m = item as Record<string, unknown>
        const url = (m.url as string) || (m.image_url as string) || (m.src as string) || ''
        if (!url) continue
        result.images.push({
          type: 'image',
          url,
          title: (m.title as string) || (m.alt as string) || '',
          sourceUrl: (m.source_url as string) || (m.page_url as string) || '',
          width: (m.width as number) || undefined,
          height: (m.height as number) || undefined,
        })
      }
      break
    }
  }

  // Videos from video_results
  const videoArrays = [data.video_results, data.videos]
  for (const arr of videoArrays) {
    if (Array.isArray(arr) && arr.length > 0) {
      for (const item of arr) {
        if (typeof item !== 'object' || item === null) continue
        const m = item as Record<string, unknown>
        const url = (m.url as string) || (m.video_url as string) || ''
        if (!url) continue
        result.videos.push({
          type: 'video',
          url,
          thumbnail: (m.thumbnail as string) || (m.thumbnail_url as string) || '',
          title: (m.title as string) || '',
          sourceUrl: (m.source_url as string) || '',
          duration: (m.duration as string) || '',
        })
      }
      break
    }
  }

  // Also check nested answer object
  if (typeof data.answer === 'object' && data.answer !== null) {
    const a = data.answer as Record<string, unknown>
    if (!result.images.length) extractMedia(a, result)
  }
}

function parseTextContent(textStr: string, result: SearchResult): void {
  let parsed: unknown
  try {
    parsed = JSON.parse(textStr)
  } catch {
    result.answer = textStr
    return
  }

  if (Array.isArray(parsed)) {
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
  if (Array.isArray(data.structured_answer) && data.structured_answer.length > 0) {
    const first = data.structured_answer[0] as Record<string, unknown> | undefined
    if (first && typeof first.text === 'string' && first.text) {
      result.answer = first.text
      return
    }
  }

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
    if (Array.isArray(answerMap.structured_answer) && answerMap.structured_answer.length > 0) {
      const first = answerMap.structured_answer[0] as Record<string, unknown> | undefined
      if (first && typeof first.text === 'string' && first.text) {
        result.answer = first.text
      }
    }
    return
  }

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
