import { ENDPOINTS, CHROME_HEADERS, API_VERSION, MODE_PAYLOAD, MODEL_PREFERENCE } from './config'
import type { SessionStore } from './storage'
import type { SSEPayload, SessionState, SearchResult, Citation, WebResult, MediaItem, ThinkingStep } from './types'

function uuid(): string {
  return crypto.randomUUID()
}

function extractDomain(url: string): string {
  try { return new URL(url).hostname.replace(/^www\./, '') } catch { return url }
}

function favicon(url: string): string {
  return `https://www.google.com/s2/favicons?domain=${extractDomain(url)}&sz=32`
}

// --- Session Pool: in-memory + KV with 429 rotation ---

const MAX_RETRIES = 3

/** In-memory session pool — survives across requests within a single worker instance. */
let memPool: SessionState[] = []
const rateLimited = new Set<string>() // csrfTokens that got 429'd

/** Get a non-rate-limited session from pool, or create fresh. */
export async function getPooledSession(store: SessionStore): Promise<SessionState> {
  // 1. Check in-memory pool
  for (const s of memPool) {
    if (!rateLimited.has(s.csrfToken)) return s
  }

  // 2. Check stored pool
  const pool = await store.getPool()
  for (const s of pool) {
    if (s.csrfToken && !rateLimited.has(s.csrfToken)) {
      memPool.push(s)
      return s
    }
  }

  // 3. Try legacy single-session cache
  const legacy = await store.getLegacy()
  if (legacy?.csrfToken && !rateLimited.has(legacy.csrfToken)) {
    memPool.push(legacy)
    return legacy
  }

  // 4. Create fresh session
  return createFreshSession(store)
}

/** Create a brand new anonymous session, add to pool. */
export async function createFreshSession(store: SessionStore): Promise<SessionState> {
  const session = await _rawInitSession()
  memPool.push(session)

  // Persist to store (best effort)
  await store.addToPool(session)
  await store.saveLegacy(session)

  return session
}

/** Mark a session as rate-limited (429). */
function markSessionRateLimited(session: SessionState): void {
  rateLimited.add(session.csrfToken)
}

/** Raw session initialization — always fetches fresh from Perplexity. */
async function _rawInitSession(): Promise<SessionState> {
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

  return {
    csrfToken,
    cookies,
    createdAt: new Date().toISOString(),
  }
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

/** Initialize a session: uses pool (memory → storage → fresh). */
export async function initSession(store: SessionStore): Promise<SessionState> {
  return getPooledSession(store)
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
  store: SessionStore,
  query: string,
  mode: string = 'auto',
  model: string = '',
  followUpUUID: string | null = null,
  sessionOverride?: SessionState,
): Promise<SearchResult> {
  const payload = buildPayload(query, mode, model, followUpUUID)
  let session = sessionOverride || await initSession(store)
  const start = Date.now()

  let resp: Response | null = null
  for (let attempt = 0; attempt < MAX_RETRIES; attempt++) {
    resp = await fetch(ENDPOINTS.sseAsk, {
      method: 'POST',
      headers: buildHeaders(session),
      body: JSON.stringify(payload),
    })
    if (resp.status === 429) {
      markSessionRateLimited(session)
      session = await createFreshSession(store)
      continue
    }
    break
  }

  if (!resp || !resp.ok) {
    const text = resp ? await resp.text() : 'No response'
    throw new Error(`SSE request failed: HTTP ${resp?.status || 0} — ${text.slice(0, 300)}`)
  }

  const body = await resp.text()
  const { lastChunk, thinkingSteps } = parseSSEStream(body)

  if (!lastChunk) throw new Error('Empty SSE response')

  const result = extractSearchResult(lastChunk, query, mode, model)
  result.durationMs = Date.now() - start
  result.createdAt = new Date().toISOString()
  result.thinkingSteps = thinkingSteps

  return result
}

/** Split text into word-group chunks for simulated streaming. */
function splitIntoWordChunks(text: string, wordsPerChunk: number = 3): string[] {
  const chunks: string[] = []
  let i = 0
  while (i < text.length) {
    let wordCount = 0
    let j = i
    while (j < text.length && wordCount < wordsPerChunk) {
      // Skip whitespace
      while (j < text.length && text[j] === ' ') j++
      // Skip word
      if (j < text.length) {
        while (j < text.length && text[j] !== ' ' && text[j] !== '\n') j++
        wordCount++
      }
    }
    // Also consume trailing newlines to keep markdown blocks together
    while (j < text.length && text[j] === '\n') j++
    if (j > i) {
      chunks.push(text.slice(i, j))
      i = j
    } else {
      break
    }
  }
  return chunks
}

const sleep = (ms: number) => new Promise<void>(r => setTimeout(r, ms))

/** Stream search: returns a ReadableStream of SSE events for client consumption. */
export function streamSearch(
  store: SessionStore,
  query: string,
  mode: string = 'auto',
  model: string = '',
  followUpUUID: string | null = null,
  sessionOverride?: SessionState,
): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()

  function sendEvent(controller: ReadableStreamDefaultController, event: string, data: unknown): void {
    const json = JSON.stringify(data)
    controller.enqueue(encoder.encode(`event: ${event}\ndata: ${json}\n\n`))
  }

  return new ReadableStream({
    async start(controller) {
      const t0 = Date.now()

      try {
        // Build payload first (CPU-only, instant) — session init is the slow part
        const payload = buildPayload(query, mode, model, followUpUUID)

        // Emit progress BEFORE session init so the user sees immediate feedback
        sendEvent(controller, 'progress', { status: 'searching', message: 'Searching the web...' })

        let session = sessionOverride || await initSession(store)
        const tSession = Date.now()

        // Fetch with 429 retry + session rotation
        let resp: Response | null = null
        let tFetch = 0
        for (let attempt = 0; attempt < MAX_RETRIES; attempt++) {
          resp = await fetch(ENDPOINTS.sseAsk, {
            method: 'POST',
            headers: buildHeaders(session),
            body: JSON.stringify(payload),
          })
          tFetch = Date.now()

          if (resp.status === 429) {
            markSessionRateLimited(session)
            sendEvent(controller, 'progress', { status: 'rotating', message: `Rate limited, creating fresh session (attempt ${attempt + 2})...` })
            session = await createFreshSession(store)
            continue
          }
          break
        }

        if (!resp || !resp.ok) {
          const text = resp ? await resp.text() : 'No response'
          const status = resp?.status || 0
          sendEvent(controller, 'error', { message: `HTTP ${status}: ${text.slice(0, 300)}` })
          controller.close()
          return
        }

        if (!resp.body) {
          sendEvent(controller, 'error', { message: 'No response body from upstream' })
          controller.close()
          return
        }

        // Read Perplexity response body incrementally
        const reader = resp.body.getReader()
        const decoder = new TextDecoder()
        let sseBuffer = ''
        let lastData: Record<string, unknown> | null = null
        let sentSources = false
        const thinkingSteps: ThinkingStep[] = []
        const seenStepTypes = new Set<string>()
        let tFirstByte = 0
        let tFirstAnswer = 0

        // Collect answer text for simulated word-by-word streaming
        let fullAnswer = ''
        let pendingAnswer = ''

        while (true) {
          const { done, value } = await reader.read()
          if (done) break
          if (!tFirstByte) tFirstByte = Date.now()

          sseBuffer += decoder.decode(value, { stream: true })

          // Perplexity SSE chunks are delimited by \r\n\r\n
          const parts = sseBuffer.split('\r\n\r\n')
          sseBuffer = parts.pop() || ''

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

            // Extract and emit thinking steps
            const newSteps = extractThinkingSteps(data, seenStepTypes, Date.now() - t0)
            // Bulk delivery: redistribute timestamps to show logical progression
            // (scraping endpoint delivers all steps in the final chunk at once)
            if (newSteps.length > 1) {
              const elapsed = Date.now() - t0
              for (let si = 0; si < newSteps.length; si++) {
                // Spread steps across first ~30% of total elapsed time
                newSteps[si].timestamp = Math.round(elapsed * 0.3 * (si + 1) / (newSteps.length + 1))
              }
            }
            for (let si = 0; si < newSteps.length; si++) {
              if (si > 0) await sleep(newSteps.length > 1 ? 200 : 30)
              thinkingSteps.push(newSteps[si])
              sendEvent(controller, 'thinking', { step: newSteps[si] })
            }

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
                  thumbnail: w.thumbnail,
                }))
                sendEvent(controller, 'sources', { citations, webResults })
                sentSources = true
              }
            }

            // Detect new answer text
            const answer = extractAnswerText(data)
            if (answer && answer.length > fullAnswer.length) {
              const delta = answer.slice(fullAnswer.length)
              fullAnswer = answer
              pendingAnswer += delta
            }
          }

          // Stream any pending answer text as word-by-word chunks
          if (pendingAnswer) {
            if (!tFirstAnswer) {
              tFirstAnswer = Date.now()
              sendEvent(controller, 'progress', { status: 'answering', message: 'Generating answer...' })
            }
            const wordChunks = splitIntoWordChunks(pendingAnswer, 3)
            let streamed = ''
            for (const wc of wordChunks) {
              streamed += wc
              const currentFull = fullAnswer.slice(0, fullAnswer.length - pendingAnswer.length + streamed.length)
              sendEvent(controller, 'chunk', { delta: wc, full: currentFull })
              await sleep(12)
            }
            pendingAnswer = ''
          }
        }

        // Final result
        if (lastData) {
          const result = extractSearchResult(lastData, query, mode, model)
          result.durationMs = Date.now() - t0
          result.createdAt = new Date().toISOString()
          result.thinkingSteps = thinkingSteps

          // If sources weren't emitted during streaming, emit from final result
          if (!sentSources && result.citations.length > 0) {
            sendEvent(controller, 'sources', { citations: result.citations, webResults: result.webResults })
          }

          // Emit media
          if (result.images.length > 0 || result.videos.length > 0) {
            sendEvent(controller, 'media', { images: result.images, videos: result.videos })
          }

          // Emit related
          if (result.relatedQueries.length > 0) {
            sendEvent(controller, 'related', { queries: result.relatedQueries })
          }

          // Emit done with TTFB metrics
          const timing = {
            sessionMs: tSession - t0,
            fetchMs: tFetch - tSession,
            firstByteMs: tFirstByte ? tFirstByte - t0 : 0,
            firstAnswerMs: tFirstAnswer ? tFirstAnswer - t0 : 0,
            totalMs: Date.now() - t0,
          }
          sendEvent(controller, 'done', { result, timing })
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

/** Extract thinking steps from a data chunk. Tracks seen step types to avoid duplicates. */
function extractThinkingSteps(
  data: Record<string, unknown>,
  seenStepTypes: Set<string>,
  elapsedMs: number,
): ThinkingStep[] {
  const steps: ThinkingStep[] = []

  const textField = data.text
  if (typeof textField !== 'string') return steps

  let parsed: unknown
  try {
    parsed = JSON.parse(textField)
  } catch {
    return steps
  }

  if (!Array.isArray(parsed)) return steps

  for (const step of parsed) {
    if (typeof step !== 'object' || step === null) continue
    const s = step as Record<string, unknown>
    const stepType = (s.step_type as string) || ''
    if (!stepType) continue

    // Build a unique key: stepType + content hash to detect new steps
    const content = extractStepContent(s)
    const key = `${stepType}:${content.slice(0, 100)}`
    if (seenStepTypes.has(key)) continue
    seenStepTypes.add(key)

    // Skip FINAL — that's the answer, not a thinking step
    if (stepType === 'FINAL') continue

    if (content) {
      steps.push({
        stepType,
        content,
        status: (s.status as string) || undefined,
        timestamp: elapsedMs,
      })
    }
  }

  return steps
}

/** Extract readable content from a step object. */
function extractStepContent(step: Record<string, unknown>): string {
  const stepType = (step.step_type as string) || ''
  const content = step.content
  if (typeof content === 'string') return content
  if (typeof content === 'object' && content !== null) {
    const c = content as Record<string, unknown>

    // Human-readable formatting for known step types
    if (stepType === 'INITIAL_QUERY' || stepType === 'INITIAL') {
      if (typeof c.query === 'string') return c.query
      if (typeof c.text === 'string') return c.text
    }
    if (stepType === 'SEARCH_WEB' && Array.isArray(c.queries)) {
      const queries = (c.queries as Record<string, unknown>[])
        .map(q => (q.query as string) || '')
        .filter(Boolean)
      return queries.length ? `Searching: ${queries.join(', ')}` : 'Searching the web...'
    }
    if (stepType === 'SEARCH_RESULTS' && Array.isArray(c.web_results)) {
      const results = c.web_results as Record<string, unknown>[]
      const titles = results.slice(0, 5).map(r => (r.name as string) || '').filter(Boolean)
      const suffix = results.length > 5 ? ` and ${results.length - 5} more` : ''
      return titles.length ? `Found ${results.length} sources: ${titles.join(', ')}${suffix}` : `Found ${results.length} sources`
    }
    if (stepType === 'READING' || stepType === 'READ_RESULTS' || stepType === 'ANALYZE') {
      return typeof c.text === 'string' ? c.text : 'Reading and analyzing sources...'
    }
    if (stepType === 'THINKING' || stepType === 'REASONING') {
      return typeof c.text === 'string' ? c.text : 'Reasoning through the information...'
    }
    if (stepType === 'REWRITE_QUERY') {
      if (typeof c.query === 'string') return `Refined query: ${c.query}`
      return 'Refining the search query...'
    }

    // Try common fields
    for (const key of ['text', 'answer', 'query', 'message', 'description', 'thought']) {
      if (typeof c[key] === 'string' && (c[key] as string).length > 0) return c[key] as string
    }
    // Stringify as last resort
    try { return JSON.stringify(content) } catch { return '' }
  }
  // Try other common fields directly on the step
  for (const key of ['text', 'query', 'message', 'thought']) {
    if (typeof step[key] === 'string' && (step[key] as string).length > 0) return step[key] as string
  }
  return ''
}

/** Unwrap a value that may be a nested JSON string containing {answer:"...", structured_answer:[...]}. */
function unwrapAnswer(val: unknown): string {
  if (typeof val !== 'string' || !val) return ''
  // Try to parse as JSON
  let obj: Record<string, unknown>
  try {
    const parsed = JSON.parse(val)
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) return val
    obj = parsed as Record<string, unknown>
  } catch {
    return val // plain text
  }
  // Extract from parsed object
  if (Array.isArray(obj.structured_answer) && obj.structured_answer.length > 0) {
    const first = obj.structured_answer[0] as Record<string, unknown>
    if (typeof first?.text === 'string' && first.text) return first.text
  }
  if (typeof obj.answer === 'string' && obj.answer) return unwrapAnswer(obj.answer) // recurse
  return val
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
                if (typeof a === 'string') return unwrapAnswer(a)
              }
            }
          }
        }
        return ''
      }
      if (typeof parsed === 'object' && parsed !== null) {
        const obj = parsed as Record<string, unknown>
        if (typeof obj.answer === 'string') return unwrapAnswer(obj.answer)
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
  if (typeof data.answer === 'string') return unwrapAnswer(data.answer)
  if (typeof data.answer === 'object' && data.answer !== null) {
    const a = data.answer as Record<string, unknown>
    if (typeof a.answer === 'string') return unwrapAnswer(a.answer)
  }
  return ''
}

/** Extract web results from any level of a data chunk (including deeply nested step arrays). */
function extractWebResultsFromData(data: Record<string, unknown>): WebResult[] {
  // Top-level web_results
  if (Array.isArray(data.web_results)) {
    return parseWebResults(data.web_results as unknown[])
  }
  // Nested answer object
  if (typeof data.answer === 'object' && data.answer !== null) {
    const a = data.answer as Record<string, unknown>
    if (Array.isArray(a.web_results)) {
      return parseWebResults(a.web_results as unknown[])
    }
  }
  // Parse text field — may be a step array or JSON object
  if (typeof data.text === 'string') {
    try {
      const parsed = JSON.parse(data.text)
      // Direct object with web_results
      if (!Array.isArray(parsed) && typeof parsed === 'object' && parsed !== null) {
        if (Array.isArray(parsed.web_results)) return parseWebResults(parsed.web_results)
      }
      // Step array: dig into FINAL step's content.answer
      if (Array.isArray(parsed)) {
        for (const step of parsed) {
          if (typeof step !== 'object' || step === null) continue
          const s = step as Record<string, unknown>
          // Check SEARCH_RESULTS step content
          if (s.step_type === 'SEARCH_RESULTS' && typeof s.content === 'object' && s.content !== null) {
            const c = s.content as Record<string, unknown>
            if (Array.isArray(c.web_results)) return parseWebResults(c.web_results as unknown[])
          }
          // Check FINAL step's nested answer JSON
          if (s.step_type === 'FINAL' && typeof s.content === 'object' && s.content !== null) {
            const c = s.content as Record<string, unknown>
            if (typeof c.answer === 'string') {
              try {
                const inner = JSON.parse(c.answer)
                if (typeof inner === 'object' && inner !== null && Array.isArray(inner.web_results)) {
                  return parseWebResults(inner.web_results)
                }
              } catch { /* not JSON */ }
            }
          }
        }
      }
    } catch { /* not JSON */ }
  }
  return []
}

/** Parse SSE stream text, return the last data chunk and all thinking steps. */
function parseSSEStream(text: string): { lastChunk: Record<string, unknown> | null; thinkingSteps: ThinkingStep[] } {
  const chunks = text.split('\r\n\r\n')
  let last: Record<string, unknown> | null = null
  const thinkingSteps: ThinkingStep[] = []
  const seenStepTypes = new Set<string>()

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
      const data = JSON.parse(dataStr) as Record<string, unknown>
      last = data
      const newSteps = extractThinkingSteps(data, seenStepTypes, 0)
      thinkingSteps.push(...newSteps)
    } catch { /* skip malformed */ }
  }

  return { lastChunk: last, thinkingSteps }
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
    thinkingSteps: [],
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
      thumbnail: w.thumbnail,
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
      thumbnail: (m.thumbnail as string) || (m.thumbnail_url as string) || (m.image_url as string) || undefined,
    }))
}
