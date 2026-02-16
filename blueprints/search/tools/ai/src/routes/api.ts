import { Hono } from 'hono'
import { cors } from 'hono/cors'
import type { HonoEnv, SessionState } from '../types'
import { initSession, search, streamSearch, createFreshSession } from '../perplexity'
import { streamSearchAPI } from '../api-stream'
import { ThreadManager } from '../threads'
import { AccountManager } from '../accounts'
import { backgroundRegister } from '../register'
import { Cache } from '../cache'
import { DEFAULT_MODE } from '../config'

/** Modes that require an authenticated account (pro queries). */
const PRO_MODES = new Set(['pro', 'reasoning', 'deep'])

/**
 * Get session for search:
 * - auto mode → anonymous session pool
 * - pro/reasoning/deep → try account session, fallback to anonymous
 *
 * Also triggers background registration if no active accounts exist.
 */
async function getSessionForMode(
  kv: KVNamespace,
  mode: string,
  ctx: ExecutionContext,
): Promise<{ session: SessionState; accountId: string | null }> {
  if (!PRO_MODES.has(mode)) {
    // Auto mode — use anonymous session
    const session = await initSession(kv)
    return { session, accountId: null }
  }

  // Pro mode — try to get an authenticated account
  const am = new AccountManager(kv)
  const account = await am.nextAccount()

  if (account) {
    return { session: account.session, accountId: account.id }
  }

  // No accounts available — trigger background registration and fall back to anonymous
  ctx.waitUntil(backgroundRegister(kv))
  const session = await initSession(kv)
  return { session, accountId: null }
}

/**
 * Record account usage after a successful pro query.
 * Marks account exhausted if proQueries reaches 0.
 * Triggers background registration if we're running low.
 */
async function recordAccountUsage(kv: KVNamespace, accountId: string | null, ctx: ExecutionContext): Promise<void> {
  if (!accountId) return
  const am = new AccountManager(kv)
  await am.recordUsage(accountId)

  // Check if we need to register more accounts (fire-and-forget)
  const needsReg = await am.needsRegistration()
  if (needsReg) {
    ctx.waitUntil(backgroundRegister(kv))
  }
}

const app = new Hono<HonoEnv>()
app.use('*', cors())

// GET /api/warm — pre-warm Perplexity session (call on page load to reduce TTFB)
app.get('/warm', async (c) => {
  if (c.env.PERPLEXITY_API_KEY) {
    return c.json({ ok: true, backend: 'api', ms: 0 })
  }
  const t0 = Date.now()
  const session = await initSession(c.env.KV)

  // Also trigger background registration if needed
  const am = new AccountManager(c.env.KV)
  const needsReg = await am.needsRegistration()
  if (needsReg) {
    c.executionCtx.waitUntil(backgroundRegister(c.env.KV))
  }

  const { active, total } = await am.listAccounts()
  return c.json({
    ok: true,
    backend: 'scraper',
    cached: !!session.csrfToken,
    accounts: { active, total },
    ms: Date.now() - t0,
  })
})

// GET /api/stream?q=query&mode=auto&threadId=xxx — SSE streaming search
app.get('/stream', async (c) => {
  const query = c.req.query('q')?.trim()
  if (!query) return c.json({ error: 'q is required' }, 400)

  const mode = c.req.query('mode') || DEFAULT_MODE
  const threadId = c.req.query('threadId') || ''
  const apiKey = c.env.PERPLEXITY_API_KEY

  let stream: ReadableStream<Uint8Array>

  if (apiKey) {
    // Official API — real token-by-token streaming, ~1.3s TTFB
    let history: Array<{ role: string; content: string }> | undefined
    if (threadId) {
      const tm = new ThreadManager(c.env.KV)
      const thread = await tm.getThread(threadId)
      if (thread) {
        history = thread.messages.map(m => ({ role: m.role, content: m.content }))
      }
    }
    stream = streamSearchAPI(apiKey, query, mode, history)
  } else {
    // Scraping — use account for pro modes, anonymous for auto
    const { session, accountId } = await getSessionForMode(c.env.KV, mode, c.executionCtx)

    let followUpUUID: string | null = null
    if (threadId) {
      const tm = new ThreadManager(c.env.KV)
      const thread = await tm.getThread(threadId)
      if (thread) {
        followUpUUID = new ThreadManager(c.env.KV).getLastBackendUUID(thread)
      }
    }

    // Wrap stream to record account usage on completion
    const rawStream = streamSearch(c.env.KV, query, mode, '', followUpUUID, session)
    if (accountId && PRO_MODES.has(mode)) {
      // Record usage after streaming completes
      c.executionCtx.waitUntil(
        recordAccountUsage(c.env.KV, accountId, c.executionCtx)
      )
    }
    stream = rawStream
  }

  return new Response(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
      'Access-Control-Allow-Origin': '*',
    },
  })
})

// POST /api/search — execute search, return JSON + save thread
app.post('/search', async (c) => {
  const body = await c.req.json<{ query: string; mode?: string; threadId?: string }>()
  if (!body.query?.trim()) return c.json({ error: 'query is required' }, 400)

  const mode = body.mode || DEFAULT_MODE
  const apiKey = c.env.PERPLEXITY_API_KEY
  const tm = new ThreadManager(c.env.KV)

  try {
    let sessionOverride: SessionState | undefined
    let accountId: string | null = null

    // For scraping pro modes, get account session
    if (!apiKey && PRO_MODES.has(mode)) {
      const res = await getSessionForMode(c.env.KV, mode, c.executionCtx)
      sessionOverride = res.session
      accountId = res.accountId
    }

    if (body.threadId) {
      const thread = await tm.getThread(body.threadId)
      if (!thread) return c.json({ error: 'thread not found' }, 404)
      const followUpUUID = tm.getLastBackendUUID(thread)
      const result = await search(c.env.KV, body.query, mode, '', followUpUUID, sessionOverride)
      const updated = await tm.addFollowUp(body.threadId, body.query, result)
      if (accountId) c.executionCtx.waitUntil(recordAccountUsage(c.env.KV, accountId, c.executionCtx))
      return c.json({ result, thread: updated })
    }

    const result = await search(c.env.KV, body.query, mode, '', null, sessionOverride)
    const thread = await tm.createThread(body.query, mode, result.model, result)
    if (accountId) c.executionCtx.waitUntil(recordAccountUsage(c.env.KV, accountId, c.executionCtx))
    return c.json({ result, thread })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.json({ error: msg }, 500)
  }
})

// POST /api/thread/save — save streaming result as thread
app.post('/thread/save', async (c) => {
  const body = await c.req.json<{
    query: string
    mode: string
    threadId?: string
    result: {
      answer: string
      citations: unknown[]
      webResults: unknown[]
      relatedQueries: string[]
      images: unknown[]
      videos: unknown[]
      thinkingSteps: unknown[]
      backendUUID: string
      model: string
      durationMs: number
    }
  }>()

  const tm = new ThreadManager(c.env.KV)

  try {
    const searchResult = {
      query: body.query,
      answer: body.result.answer,
      citations: body.result.citations as any[],
      webResults: body.result.webResults as any[],
      relatedQueries: body.result.relatedQueries || [],
      images: body.result.images as any[] || [],
      videos: body.result.videos as any[] || [],
      thinkingSteps: body.result.thinkingSteps as any[] || [],
      backendUUID: body.result.backendUUID || '',
      mode: body.mode,
      model: body.result.model || '',
      durationMs: body.result.durationMs || 0,
      createdAt: new Date().toISOString(),
    }

    if (body.threadId) {
      const updated = await tm.addFollowUp(body.threadId, body.query, searchResult)
      return c.json({ thread: updated })
    }

    const thread = await tm.createThread(body.query, body.mode, searchResult.model, searchResult)
    return c.json({ thread })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.json({ error: msg }, 500)
  }
})

// GET /api/thread/:id
app.get('/thread/:id', async (c) => {
  const tm = new ThreadManager(c.env.KV)
  const thread = await tm.getThread(c.req.param('id'))
  if (!thread) return c.json({ error: 'not found' }, 404)
  return c.json(thread)
})

// POST /api/thread/:id/follow-up
app.post('/thread/:id/follow-up', async (c) => {
  const id = c.req.param('id')
  const body = await c.req.json<{ query: string; mode?: string }>()
  if (!body.query?.trim()) return c.json({ error: 'query is required' }, 400)

  const tm = new ThreadManager(c.env.KV)
  const thread = await tm.getThread(id)
  if (!thread) return c.json({ error: 'thread not found' }, 404)

  const mode = body.mode || thread.mode || DEFAULT_MODE

  try {
    let sessionOverride: SessionState | undefined
    let accountId: string | null = null

    if (!c.env.PERPLEXITY_API_KEY && PRO_MODES.has(mode)) {
      const res = await getSessionForMode(c.env.KV, mode, c.executionCtx)
      sessionOverride = res.session
      accountId = res.accountId
    }

    const followUpUUID = tm.getLastBackendUUID(thread)
    const result = await search(c.env.KV, body.query, mode, '', followUpUUID, sessionOverride)
    const updated = await tm.addFollowUp(id, body.query, result)
    if (accountId) c.executionCtx.waitUntil(recordAccountUsage(c.env.KV, accountId, c.executionCtx))
    return c.json({ result, thread: updated })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.json({ error: msg }, 500)
  }
})

// DELETE /api/thread/:id
app.delete('/thread/:id', async (c) => {
  const tm = new ThreadManager(c.env.KV)
  const ok = await tm.deleteThread(c.req.param('id'))
  if (!ok) return c.json({ error: 'not found' }, 404)
  return c.json({ ok: true })
})

// GET /api/threads
app.get('/threads', async (c) => {
  const tm = new ThreadManager(c.env.KV)
  const threads = await tm.listThreads()
  return c.json({ threads })
})

// GET /api/og?url=... — extract Open Graph metadata from a URL (for link previews)
app.get('/og', async (c) => {
  const url = c.req.query('url')
  if (!url) return c.json({ error: 'url required' }, 400)

  const cache = new Cache(c.env.KV)
  const cacheKey = `og:${url}`
  const cached = await cache.get<{ title: string; description: string; image: string; siteName: string }>(cacheKey)
  if (cached) return c.json(cached)

  try {
    const resp = await fetch(url, {
      headers: {
        'User-Agent': 'Mozilla/5.0 (compatible; AI-Search/1.0; +https://ai-search.go-mizu.workers.dev)',
        'Accept': 'text/html',
      },
      redirect: 'follow',
      signal: AbortSignal.timeout(5000),
    })
    if (!resp.ok) return c.json({ error: `HTTP ${resp.status}` }, 502)

    const reader = resp.body?.getReader()
    if (!reader) return c.json({ error: 'no body' }, 502)
    let html = ''
    const decoder = new TextDecoder()
    while (html.length < 50000) {
      const { done, value } = await reader.read()
      if (done) break
      html += decoder.decode(value, { stream: true })
      if (html.includes('</head>')) break
    }
    reader.cancel()

    const og = extractOGMeta(html)
    if (og.image && !og.image.startsWith('http')) {
      try { og.image = new URL(og.image, url).href } catch { /* leave as-is */ }
    }

    await cache.set(cacheKey, og, 86400)
    return c.json(og)
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.json({ error: msg }, 502)
  }
})

function extractOGMeta(html: string): { title: string; description: string; image: string; siteName: string } {
  const getMeta = (property: string): string => {
    const re1 = new RegExp(`<meta[^>]*(?:property|name)=["']${property}["'][^>]*content=["']([^"']*)["']`, 'i')
    const re2 = new RegExp(`<meta[^>]*content=["']([^"']*)["'][^>]*(?:property|name)=["']${property}["']`, 'i')
    return (html.match(re1)?.[1] || html.match(re2)?.[1] || '').trim()
  }
  const getTitle = (): string => {
    const m = html.match(/<title[^>]*>([^<]+)<\/title>/i)
    return m?.[1]?.trim() || ''
  }
  const getFirstImage = (): string => {
    const m = html.match(/<img[^>]*src=["']([^"']+(?:\.jpg|\.jpeg|\.png|\.webp)[^"']*)["']/i)
    return m?.[1] || ''
  }
  return {
    title: getMeta('og:title') || getMeta('twitter:title') || getTitle(),
    description: getMeta('og:description') || getMeta('twitter:description') || getMeta('description') || '',
    image: getMeta('og:image') || getMeta('twitter:image') || getMeta('twitter:image:src') || getFirstImage(),
    siteName: getMeta('og:site_name') || '',
  }
}

// ============================================================
// TEMPORARY DEBUG ENDPOINTS — remove after testing
// ============================================================

// GET /api/kv-test — test raw KV read/write
app.get('/kv-test', async (c) => {
  const kv = c.env.KV
  const key = 'test:debug:' + Date.now()
  const value = { hello: 'world', ts: new Date().toISOString() }
  try {
    await kv.put(key, JSON.stringify(value))
    const read = await kv.get(key, 'text')
    await kv.delete(key)
    return c.json({ ok: true, written: value, readBack: read ? JSON.parse(read) : null })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.json({ ok: false, error: msg }, 500)
  }
})

// GET /api/kv-list — list all account-related KV keys
app.get('/kv-list', async (c) => {
  const kv = c.env.KV
  const results: Record<string, unknown> = {}
  // Check known keys
  for (const key of ['accounts:index', 'accounts:robin', 'accounts:log', 'accounts:lock']) {
    const val = await kv.get(key, 'text')
    results[key] = val ? JSON.parse(val) : null
  }
  // List account:* keys
  const list = await kv.list({ prefix: 'account:' })
  results['account_keys'] = list.keys.map(k => k.name)
  return c.json(results)
})

// GET /api/accounts — list all accounts + summary
app.get('/accounts', async (c) => {
  const am = new AccountManager(c.env.KV)
  const { accounts, active, total } = await am.listAccounts()
  return c.json({ accounts, active, total })
})

// GET /api/accounts/logs — view full registration logs
app.get('/accounts/logs', async (c) => {
  const am = new AccountManager(c.env.KV)
  const logs = await am.getLogs()
  return c.json({ logs, count: logs.length })
})

// POST /api/accounts/register — run registration synchronously for debugging
app.post('/accounts/register', async (c) => {
  const am = new AccountManager(c.env.KV)
  const t0 = Date.now()
  const steps: Array<{ step: string; ms: number; detail?: string }> = []

  try {
    // Step 1: Check lock
    const locked = await am.tryLock()
    steps.push({ step: 'lock', ms: Date.now() - t0, detail: locked ? 'acquired' : 'held' })
    if (!locked) {
      return c.json({ error: 'Registration already in progress (lock held)', steps }, 409)
    }

    // Step 2: Init anonymous session
    const { ENDPOINTS, CHROME_HEADERS, MAGIC_LINK_REGEX, SIGNIN_SUBJECT } = await import('../config')

    let cookies = ''
    const sessionResp = await fetch(ENDPOINTS.session, {
      headers: { ...CHROME_HEADERS },
      redirect: 'manual',
    })
    // Extract cookies manually (same as register.ts)
    const setCookies1 = sessionResp.headers.getAll?.('set-cookie') ?? []
    const cookieMap = new Map<string, string>()
    for (const sc of setCookies1) {
      const nameVal = sc.split(';')[0]
      const eq = nameVal.indexOf('=')
      if (eq > 0) cookieMap.set(nameVal.slice(0, eq).trim(), nameVal.slice(eq + 1).trim())
    }
    cookies = Array.from(cookieMap.entries()).map(([k, v]) => `${k}=${v}`).join('; ')

    const csrfResp = await fetch(ENDPOINTS.csrf, {
      headers: { ...CHROME_HEADERS, Cookie: cookies },
      redirect: 'manual',
    })
    const setCookies2 = csrfResp.headers.getAll?.('set-cookie') ?? []
    for (const sc of setCookies2) {
      const nameVal = sc.split(';')[0]
      const eq = nameVal.indexOf('=')
      if (eq > 0) cookieMap.set(nameVal.slice(0, eq).trim(), nameVal.slice(eq + 1).trim())
    }
    cookies = Array.from(cookieMap.entries()).map(([k, v]) => `${k}=${v}`).join('; ')

    const csrfBody = await csrfResp.text()
    let csrfToken = ''
    try {
      const json = JSON.parse(csrfBody)
      if (json.csrfToken) csrfToken = json.csrfToken
    } catch { /* not JSON */ }
    if (!csrfToken) {
      const match = cookies.match(/next-auth\.csrf-token=([^;]+)/)
      if (match) {
        const val = match[1]
        const parts = val.split('%')
        if (parts.length > 1) csrfToken = parts[0]
        else {
          try {
            const decoded = decodeURIComponent(val)
            const pipeParts = decoded.split('|')
            csrfToken = pipeParts.length > 1 ? pipeParts[0] : decoded
          } catch { csrfToken = val }
        }
      }
    }

    steps.push({
      step: 'session',
      ms: Date.now() - t0,
      detail: `session=${sessionResp.status}, csrf=${csrfResp.status}, csrfToken=${csrfToken.length}chars, cookies=${cookies.length}chars`,
    })

    if (!csrfToken) {
      await am.unlock()
      return c.json({ error: 'CSRF extraction failed', steps, csrfBody: csrfBody.slice(0, 200), cookies: cookies.slice(0, 200) }, 500)
    }

    // Step 3: Create temp email
    const { createTempEmail } = await import('../email')
    const { client: emailClient, provider } = await createTempEmail()
    const email = emailClient.email()
    steps.push({ step: 'email', ms: Date.now() - t0, detail: `${provider}: ${email}` })

    // Step 4: Request magic link
    const formData = `email=${encodeURIComponent(email)}&csrfToken=${encodeURIComponent(csrfToken)}&callbackUrl=${encodeURIComponent('https://www.perplexity.ai/')}&json=true`
    const signinResp = await fetch(ENDPOINTS.signin, {
      method: 'POST',
      headers: {
        ...CHROME_HEADERS,
        'Content-Type': 'application/x-www-form-urlencoded',
        'Cookie': cookies,
        'Accept': '*/*',
        'Sec-Fetch-Dest': 'empty',
        'Sec-Fetch-Mode': 'cors',
        'Sec-Fetch-Site': 'same-origin',
        'Origin': 'https://www.perplexity.ai',
        'Referer': 'https://www.perplexity.ai/',
      },
      body: formData,
    })
    const signinBody = await signinResp.text()
    steps.push({ step: 'signin', ms: Date.now() - t0, detail: `HTTP ${signinResp.status}: ${signinBody.slice(0, 200)}` })

    if (!signinResp.ok) {
      await am.unlock()
      return c.json({ error: `Signin failed: HTTP ${signinResp.status}`, steps }, 500)
    }

    // Step 5: Wait for magic link email (25s timeout)
    const emailBody = await emailClient.waitForMessage(SIGNIN_SUBJECT, 25000)
    steps.push({ step: 'email_received', ms: Date.now() - t0, detail: `body=${emailBody.length}chars` })

    const linkMatch = emailBody.match(MAGIC_LINK_REGEX)
    if (!linkMatch?.[1]) {
      await am.unlock()
      return c.json({ error: 'Magic link not found in email', steps, emailPreview: emailBody.slice(0, 500) }, 500)
    }
    const magicLink = linkMatch[1]
    steps.push({ step: 'magic_link', ms: Date.now() - t0, detail: `${magicLink.slice(0, 80)}...` })

    // Step 6: Complete auth
    let authCookies = cookies
    const authResp = await fetch(magicLink, {
      headers: { ...CHROME_HEADERS, Cookie: authCookies },
      redirect: 'manual',
    })
    const authSetCookies = authResp.headers.getAll?.('set-cookie') ?? []
    const authCookieMap = new Map<string, string>()
    for (const part of authCookies.split('; ')) {
      const eq = part.indexOf('=')
      if (eq > 0) authCookieMap.set(part.slice(0, eq), part.slice(eq + 1))
    }
    for (const sc of authSetCookies) {
      const nameVal = sc.split(';')[0]
      const eq = nameVal.indexOf('=')
      if (eq > 0) authCookieMap.set(nameVal.slice(0, eq).trim(), nameVal.slice(eq + 1).trim())
    }
    authCookies = Array.from(authCookieMap.entries()).map(([k, v]) => `${k}=${v}`).join('; ')

    let location = authResp.headers.get('location')
    let redirectCount = 0
    for (let i = 0; i < 5 && location; i++) {
      const url = location.startsWith('http') ? location : `https://www.perplexity.ai${location}`
      const redirectResp = await fetch(url, {
        headers: { ...CHROME_HEADERS, Cookie: authCookies },
        redirect: 'manual',
      })
      const rdSetCookies = redirectResp.headers.getAll?.('set-cookie') ?? []
      for (const sc of rdSetCookies) {
        const nameVal = sc.split(';')[0]
        const eq = nameVal.indexOf('=')
        if (eq > 0) authCookieMap.set(nameVal.slice(0, eq).trim(), nameVal.slice(eq + 1).trim())
      }
      authCookies = Array.from(authCookieMap.entries()).map(([k, v]) => `${k}=${v}`).join('; ')
      location = redirectResp.headers.get('location')
      redirectCount++
    }

    // Extract final CSRF
    let finalCsrf = ''
    const csrfMatch2 = authCookies.match(/next-auth\.csrf-token=([^;]+)/)
    if (csrfMatch2) {
      const val = csrfMatch2[1]
      const parts = val.split('%')
      if (parts.length > 1) finalCsrf = parts[0]
      else {
        try { finalCsrf = decodeURIComponent(val).split('|')[0] } catch { finalCsrf = val }
      }
    }

    steps.push({
      step: 'auth',
      ms: Date.now() - t0,
      detail: `redirects=${redirectCount}, cookies=${authCookies.length}chars, csrf=${finalCsrf ? 'yes' : 'no'}`,
    })

    // Step 7: Save account
    const authSession: import('../types').SessionState = {
      csrfToken: finalCsrf || csrfToken,
      cookies: authCookies,
      createdAt: new Date().toISOString(),
    }

    const accountId = await am.addAccount(email, authSession, 5)
    steps.push({ step: 'saved', ms: Date.now() - t0, detail: `id=${accountId}, email=${email}` })

    await am.log({
      timestamp: new Date().toISOString(),
      event: 'account_saved',
      message: `Account registered: ${email} (id: ${accountId}, proQueries: 5)`,
      provider,
      email,
      accountId,
      durationMs: Date.now() - t0,
    })

    await am.unlock()
    return c.json({ ok: true, accountId, email, provider, steps, durationMs: Date.now() - t0 })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    const stack = e instanceof Error ? e.stack : undefined
    steps.push({ step: 'error', ms: Date.now() - t0, detail: msg })
    await am.log({
      timestamp: new Date().toISOString(),
      event: 'error',
      message: `Registration failed: ${msg}`,
      error: stack || msg,
      durationMs: Date.now() - t0,
    })
    await am.unlock()
    return c.json({ error: msg, steps, stack }, 500)
  }
})

// DELETE /api/accounts/:id — delete a specific account
app.delete('/accounts/:id', async (c) => {
  const am = new AccountManager(c.env.KV)
  const ok = await am.deleteAccount(c.req.param('id'))
  if (!ok) return c.json({ error: 'account not found' }, 404)
  return c.json({ ok: true })
})

// DELETE /api/accounts — delete ALL accounts (reset)
app.delete('/accounts', async (c) => {
  const am = new AccountManager(c.env.KV)
  const { accounts } = await am.listAccounts()
  for (const a of accounts) {
    await am.deleteAccount(a.id)
  }
  // Also clear robin pointer and logs
  const cache = new Cache(c.env.KV)
  await cache.delete('accounts:robin')
  return c.json({ ok: true, deleted: accounts.length })
})

// GET /api/accounts/:id — view full account details (including session)
app.get('/accounts/:id', async (c) => {
  const cache = new Cache(c.env.KV)
  const account = await cache.get<import('../types').Account>(`account:${c.req.param('id')}`)
  if (!account) return c.json({ error: 'account not found' }, 404)
  // Mask cookies for safety — show first 50 chars only
  const masked = {
    ...account,
    session: {
      ...account.session,
      cookies: account.session.cookies.slice(0, 80) + '...',
      csrfToken: account.session.csrfToken.slice(0, 20) + '...',
    },
  }
  return c.json(masked)
})

// POST /api/accounts/test-email — test email provider only (no registration)
app.post('/accounts/test-email', async (c) => {
  const { createTempEmail } = await import('../email')
  const am = new AccountManager(c.env.KV)
  const t0 = Date.now()
  try {
    const { client, provider } = await createTempEmail()
    const result = {
      ok: true,
      provider,
      email: client.email(),
      durationMs: Date.now() - t0,
    }
    await am.log({
      timestamp: new Date().toISOString(),
      event: 'email_created',
      message: `[TEST] Email created via ${provider}: ${client.email()}`,
      provider,
      email: client.email(),
      durationMs: Date.now() - t0,
    })
    return c.json(result)
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    await am.log({
      timestamp: new Date().toISOString(),
      event: 'error',
      message: `[TEST] Email creation failed: ${msg}`,
      error: msg,
      durationMs: Date.now() - t0,
    })
    return c.json({ ok: false, error: msg, durationMs: Date.now() - t0 }, 500)
  }
})

// ============================================================

app.onError((err, c) => {
  console.error('[API Error]', err.message)
  return c.json({ error: err.message }, 500)
})

export default app
