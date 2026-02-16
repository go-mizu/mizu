import { Hono } from 'hono'
import { cors } from 'hono/cors'
import type { HonoEnv, SessionState, Account } from '../types'
import { initSession, search, streamSearch, createFreshSession } from '../perplexity'
import { streamSearchAPI } from '../api-stream'
import { ThreadManager } from '../threads'
import { AccountManager } from '../accounts'
import { backgroundRegister } from '../register'
import {
  getAccountStore, getThreadStore, getSessionStore, getOGStore,
  type AccountStore, type SessionStore,
} from '../storage'
import { DEFAULT_MODE, ENDPOINTS, CHROME_HEADERS, MAGIC_LINK_REGEX, SIGNIN_SUBJECT } from '../config'

/** Modes that require an authenticated account (pro queries). */
const PRO_MODES = new Set(['pro', 'reasoning', 'deep'])

/**
 * Get session for search:
 * - auto mode → anonymous session pool
 * - pro/reasoning/deep → try account session, fallback to anonymous
 */
async function getSessionForMode(
  sessionStore: SessionStore,
  accountStore: AccountStore,
  mode: string,
  ctx: ExecutionContext,
  secret: string,
): Promise<{ session: SessionState; accountId: string | null }> {
  if (!PRO_MODES.has(mode)) {
    const session = await initSession(sessionStore)
    return { session, accountId: null }
  }

  const am = new AccountManager(accountStore, secret)
  const account = await am.nextAccount()

  if (account) {
    return { session: account.session, accountId: account.id }
  }

  // No accounts — trigger background registration and fall back to anonymous
  ctx.waitUntil(backgroundRegister(accountStore, secret))
  const session = await initSession(sessionStore)
  return { session, accountId: null }
}

/** Record account usage after a successful pro query. */
async function recordAccountUsage(accountStore: AccountStore, accountId: string | null, ctx: ExecutionContext, secret: string): Promise<void> {
  if (!accountId) return
  const am = new AccountManager(accountStore, secret)
  await am.recordUsage(accountId)

  const needsReg = await am.needsRegistration()
  if (needsReg) {
    ctx.waitUntil(backgroundRegister(accountStore, secret))
  }
}

const app = new Hono<HonoEnv>()
app.use('*', cors())

// GET /api/warm — pre-warm Perplexity session
app.get('/warm', async (c) => {
  if (c.env.PERPLEXITY_API_KEY) {
    return c.json({ ok: true, backend: 'api', ms: 0 })
  }
  const t0 = Date.now()
  const sessionStore = getSessionStore(c.env)
  const session = await initSession(sessionStore)
  const accountStore = getAccountStore(c.env, c.req.query('storage'))
  const secret = c.env.ACCOUNT_SECRET || 'default-dev-secret'
  const am = new AccountManager(accountStore, secret)

  const needsReg = await am.needsRegistration()
  if (needsReg) {
    c.executionCtx.waitUntil(backgroundRegister(accountStore, secret))
  }

  const { active, total } = await am.listAccounts()
  return c.json({
    ok: true,
    backend: 'scraper',
    storage: accountStore.name,
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
  const sessionStore = getSessionStore(c.env)
  const threadStore = getThreadStore(c.env)

  let stream: ReadableStream<Uint8Array>

  if (apiKey) {
    let history: Array<{ role: string; content: string }> | undefined
    if (threadId) {
      const tm = new ThreadManager(threadStore)
      const thread = await tm.getThread(threadId)
      if (thread) {
        history = thread.messages.map(m => ({ role: m.role, content: m.content }))
      }
    }
    stream = streamSearchAPI(apiKey, query, mode, history)
  } else {
    const accountStore = getAccountStore(c.env, c.req.query('storage'))
    const secret = c.env.ACCOUNT_SECRET || 'default-dev-secret'
    const { session, accountId } = await getSessionForMode(sessionStore, accountStore, mode, c.executionCtx, secret)

    let followUpUUID: string | null = null
    if (threadId) {
      const tm = new ThreadManager(threadStore)
      const thread = await tm.getThread(threadId)
      if (thread) {
        followUpUUID = tm.getLastBackendUUID(thread)
      }
    }

    const rawStream = streamSearch(sessionStore, query, mode, '', followUpUUID, session)
    if (accountId && PRO_MODES.has(mode)) {
      c.executionCtx.waitUntil(recordAccountUsage(accountStore, accountId, c.executionCtx, secret))
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
  const sessionStore = getSessionStore(c.env)
  const threadStore = getThreadStore(c.env)
  const tm = new ThreadManager(threadStore)
  const secret = c.env.ACCOUNT_SECRET || 'default-dev-secret'

  try {
    let sessionOverride: SessionState | undefined
    let accountId: string | null = null

    if (!apiKey && PRO_MODES.has(mode)) {
      const accountStore = getAccountStore(c.env, c.req.query('storage'))
      const res = await getSessionForMode(sessionStore, accountStore, mode, c.executionCtx, secret)
      sessionOverride = res.session
      accountId = res.accountId
    }

    if (body.threadId) {
      const thread = await tm.getThread(body.threadId)
      if (!thread) return c.json({ error: 'thread not found' }, 404)
      const followUpUUID = tm.getLastBackendUUID(thread)
      const result = await search(sessionStore, body.query, mode, '', followUpUUID, sessionOverride)
      const updated = await tm.addFollowUp(body.threadId, body.query, result)
      if (accountId) {
        const accountStore = getAccountStore(c.env, c.req.query('storage'))
        c.executionCtx.waitUntil(recordAccountUsage(accountStore, accountId, c.executionCtx, secret))
      }
      return c.json({ result, thread: updated })
    }

    const result = await search(sessionStore, body.query, mode, '', null, sessionOverride)
    const thread = await tm.createThread(body.query, mode, result.model, result)
    if (accountId) {
      const accountStore = getAccountStore(c.env, c.req.query('storage'))
      c.executionCtx.waitUntil(recordAccountUsage(accountStore, accountId, c.executionCtx, secret))
    }
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

  const tm = new ThreadManager(getThreadStore(c.env))

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
  const tm = new ThreadManager(getThreadStore(c.env))
  const thread = await tm.getThread(c.req.param('id'))
  if (!thread) return c.json({ error: 'not found' }, 404)
  return c.json(thread)
})

// POST /api/thread/:id/follow-up
app.post('/thread/:id/follow-up', async (c) => {
  const id = c.req.param('id')
  const body = await c.req.json<{ query: string; mode?: string }>()
  if (!body.query?.trim()) return c.json({ error: 'query is required' }, 400)

  const sessionStore = getSessionStore(c.env)
  const threadStore = getThreadStore(c.env)
  const tm = new ThreadManager(threadStore)
  const thread = await tm.getThread(id)
  if (!thread) return c.json({ error: 'thread not found' }, 404)

  const mode = body.mode || thread.mode || DEFAULT_MODE
  const secret = c.env.ACCOUNT_SECRET || 'default-dev-secret'

  try {
    let sessionOverride: SessionState | undefined
    let accountId: string | null = null

    if (!c.env.PERPLEXITY_API_KEY && PRO_MODES.has(mode)) {
      const accountStore = getAccountStore(c.env, c.req.query('storage'))
      const res = await getSessionForMode(sessionStore, accountStore, mode, c.executionCtx, secret)
      sessionOverride = res.session
      accountId = res.accountId
    }

    const followUpUUID = tm.getLastBackendUUID(thread)
    const result = await search(sessionStore, body.query, mode, '', followUpUUID, sessionOverride)
    const updated = await tm.addFollowUp(id, body.query, result)
    if (accountId) {
      const accountStore = getAccountStore(c.env, c.req.query('storage'))
      c.executionCtx.waitUntil(recordAccountUsage(accountStore, accountId, c.executionCtx, secret))
    }
    return c.json({ result, thread: updated })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.json({ error: msg }, 500)
  }
})

// DELETE /api/thread/:id
app.delete('/thread/:id', async (c) => {
  const tm = new ThreadManager(getThreadStore(c.env))
  const ok = await tm.deleteThread(c.req.param('id'))
  if (!ok) return c.json({ error: 'not found' }, 404)
  return c.json({ ok: true })
})

// GET /api/threads
app.get('/threads', async (c) => {
  const tm = new ThreadManager(getThreadStore(c.env))
  const threads = await tm.listThreads()
  return c.json({ threads })
})

// GET /api/og?url=...
app.get('/og', async (c) => {
  const url = c.req.query('url')
  if (!url) return c.json({ error: 'url required' }, 400)

  const ogStore = getOGStore(c.env)
  const cached = await ogStore.get(url)
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

    await ogStore.set(url, og, 86400)
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
// TEMPORARY DEBUG/TEST ENDPOINTS
// All support ?storage=memory|kv|d1 query parameter
// ============================================================

// GET /api/storage/test — test read/write on selected backend
app.get('/storage/test', async (c) => {
  const override = c.req.query('storage')
  const accountStore = getAccountStore(c.env, override)
  const t0 = Date.now()
  try {
    const active = await accountStore.countActive()
    const accounts = await accountStore.listAccounts()
    return c.json({
      ok: true,
      backend: accountStore.name,
      active,
      total: accounts.length,
      ms: Date.now() - t0,
    })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.json({ ok: false, backend: accountStore.name, error: msg, ms: Date.now() - t0 }, 500)
  }
})

// GET /api/accounts?storage=memory — list all accounts
app.get('/accounts', async (c) => {
  const accountStore = getAccountStore(c.env, c.req.query('storage'))
  const secret = c.env.ACCOUNT_SECRET || 'default-dev-secret'
  const am = new AccountManager(accountStore, secret)
  const { accounts, active, total } = await am.listAccounts()
  return c.json({ backend: accountStore.name, accounts, active, total })
})

// GET /api/accounts/logs?storage=memory — view registration logs
app.get('/accounts/logs', async (c) => {
  const accountStore = getAccountStore(c.env, c.req.query('storage'))
  const secret = c.env.ACCOUNT_SECRET || 'default-dev-secret'
  const am = new AccountManager(accountStore, secret)
  const logs = await am.getLogs()
  return c.json({ backend: accountStore.name, logs, count: logs.length })
})

// POST /api/accounts/register?storage=memory — full synchronous registration
app.post('/accounts/register', async (c) => {
  const accountStore = getAccountStore(c.env, c.req.query('storage'))
  const secret = c.env.ACCOUNT_SECRET || 'default-dev-secret'
  const am = new AccountManager(accountStore, secret)
  const t0 = Date.now()
  const steps: Array<{ step: string; ms: number; detail?: string }> = []

  try {
    // Step 1: Lock
    const locked = await am.tryLock()
    steps.push({ step: 'lock', ms: Date.now() - t0, detail: locked ? 'acquired' : 'held' })
    if (!locked) {
      return c.json({ error: 'Registration already in progress (lock held)', backend: accountStore.name, steps }, 409)
    }

    // Step 2: Init session (CSRF + cookies)
    let cookies = ''
    const cookieMap = new Map<string, string>()

    const sessionResp = await fetch(ENDPOINTS.session, { headers: { ...CHROME_HEADERS }, redirect: 'manual' })
    for (const sc of sessionResp.headers.getAll?.('set-cookie') ?? []) {
      const nv = sc.split(';')[0]; const eq = nv.indexOf('=')
      if (eq > 0) cookieMap.set(nv.slice(0, eq).trim(), nv.slice(eq + 1).trim())
    }
    cookies = [...cookieMap].map(([k, v]) => `${k}=${v}`).join('; ')

    const csrfResp = await fetch(ENDPOINTS.csrf, { headers: { ...CHROME_HEADERS, Cookie: cookies }, redirect: 'manual' })
    for (const sc of csrfResp.headers.getAll?.('set-cookie') ?? []) {
      const nv = sc.split(';')[0]; const eq = nv.indexOf('=')
      if (eq > 0) cookieMap.set(nv.slice(0, eq).trim(), nv.slice(eq + 1).trim())
    }
    cookies = [...cookieMap].map(([k, v]) => `${k}=${v}`).join('; ')

    const csrfBody = await csrfResp.text()
    let csrfToken = ''
    try { const j = JSON.parse(csrfBody); if (j.csrfToken) csrfToken = j.csrfToken } catch {}
    if (!csrfToken) {
      const m = cookies.match(/next-auth\.csrf-token=([^;]+)/)
      if (m) {
        const v = m[1]; const p = v.split('%')
        if (p.length > 1) csrfToken = p[0]
        else { try { csrfToken = decodeURIComponent(v).split('|')[0] } catch { csrfToken = v } }
      }
    }

    steps.push({ step: 'session', ms: Date.now() - t0, detail: `csrf=${csrfToken.length}c, cookies=${cookies.length}c` })
    if (!csrfToken) { await am.unlock(); return c.json({ error: 'CSRF failed', backend: accountStore.name, steps }, 500) }

    // Step 3: Create temp email
    const { createTempEmail } = await import('../email')
    const { client: emailClient, provider } = await createTempEmail()
    const email = emailClient.email()
    steps.push({ step: 'email', ms: Date.now() - t0, detail: `${provider}: ${email}` })

    // Step 4: Request magic link
    const formData = `email=${encodeURIComponent(email)}&csrfToken=${encodeURIComponent(csrfToken)}&callbackUrl=${encodeURIComponent('https://www.perplexity.ai/')}&json=true`
    const signinResp = await fetch(ENDPOINTS.signin, {
      method: 'POST',
      headers: { ...CHROME_HEADERS, 'Content-Type': 'application/x-www-form-urlencoded', Cookie: cookies, Accept: '*/*', 'Sec-Fetch-Dest': 'empty', 'Sec-Fetch-Mode': 'cors', 'Sec-Fetch-Site': 'same-origin', Origin: 'https://www.perplexity.ai', Referer: 'https://www.perplexity.ai/' },
      body: formData,
    })
    const signinBody = await signinResp.text()
    steps.push({ step: 'signin', ms: Date.now() - t0, detail: `HTTP ${signinResp.status}: ${signinBody.slice(0, 100)}` })
    if (!signinResp.ok) { await am.unlock(); return c.json({ error: `Signin HTTP ${signinResp.status}`, backend: accountStore.name, steps }, 500) }

    // Step 5: Wait for magic link email
    const emailBody = await emailClient.waitForMessage(SIGNIN_SUBJECT, 25000)
    steps.push({ step: 'email_received', ms: Date.now() - t0, detail: `${emailBody.length}chars` })

    const linkMatch = emailBody.match(MAGIC_LINK_REGEX)
    if (!linkMatch?.[1]) { await am.unlock(); return c.json({ error: 'Magic link not found', backend: accountStore.name, steps, preview: emailBody.slice(0, 300) }, 500) }
    const magicLink = linkMatch[1]
    steps.push({ step: 'magic_link', ms: Date.now() - t0, detail: `${magicLink.slice(0, 60)}...` })

    // Step 6: Complete auth (follow redirects)
    const authCookieMap = new Map<string, string>()
    for (const part of cookies.split('; ')) { const eq = part.indexOf('='); if (eq > 0) authCookieMap.set(part.slice(0, eq), part.slice(eq + 1)) }

    const authResp = await fetch(magicLink, { headers: { ...CHROME_HEADERS, Cookie: cookies }, redirect: 'manual' })
    for (const sc of authResp.headers.getAll?.('set-cookie') ?? []) {
      const nv = sc.split(';')[0]; const eq = nv.indexOf('=')
      if (eq > 0) authCookieMap.set(nv.slice(0, eq).trim(), nv.slice(eq + 1).trim())
    }

    let location = authResp.headers.get('location')
    let redirectCount = 0
    for (let i = 0; i < 5 && location; i++) {
      const url = location.startsWith('http') ? location : `https://www.perplexity.ai${location}`
      const authCookies = [...authCookieMap].map(([k, v]) => `${k}=${v}`).join('; ')
      const rr = await fetch(url, { headers: { ...CHROME_HEADERS, Cookie: authCookies }, redirect: 'manual' })
      for (const sc of rr.headers.getAll?.('set-cookie') ?? []) {
        const nv = sc.split(';')[0]; const eq = nv.indexOf('=')
        if (eq > 0) authCookieMap.set(nv.slice(0, eq).trim(), nv.slice(eq + 1).trim())
      }
      location = rr.headers.get('location')
      redirectCount++
    }

    const finalCookies = [...authCookieMap].map(([k, v]) => `${k}=${v}`).join('; ')
    let finalCsrf = ''
    const cm = finalCookies.match(/next-auth\.csrf-token=([^;]+)/)
    if (cm) {
      const v = cm[1]; const p = v.split('%')
      if (p.length > 1) finalCsrf = p[0]
      else { try { finalCsrf = decodeURIComponent(v).split('|')[0] } catch { finalCsrf = v } }
    }

    steps.push({ step: 'auth', ms: Date.now() - t0, detail: `redirects=${redirectCount}, cookies=${finalCookies.length}c, csrf=${finalCsrf ? 'yes' : 'no'}` })

    // Step 7: Save account
    const authSession: SessionState = { csrfToken: finalCsrf || csrfToken, cookies: finalCookies, createdAt: new Date().toISOString() }
    const accountId = await am.addAccount(email, provider, emailClient.password(), authSession, 5)
    steps.push({ step: 'saved', ms: Date.now() - t0, detail: `id=${accountId}` })

    await am.log({
      timestamp: new Date().toISOString(),
      event: 'account_saved',
      message: `Account registered: ${email} (id: ${accountId})`,
      provider, email, accountId,
      durationMs: Date.now() - t0,
    })

    // Step 8: Verify persistence
    const verify = await am.getAccount(accountId)
    const { active, total } = await am.listAccounts()
    steps.push({ step: 'verify', ms: Date.now() - t0, detail: `persisted=${!!verify}, active=${active}, total=${total}` })

    await am.unlock()
    return c.json({ ok: true, backend: accountStore.name, accountId, email, provider, active, total, steps, durationMs: Date.now() - t0 })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    steps.push({ step: 'error', ms: Date.now() - t0, detail: msg })
    await am.log({ timestamp: new Date().toISOString(), event: 'error', message: `Registration failed: ${msg}`, error: e instanceof Error ? e.stack || msg : msg, durationMs: Date.now() - t0 })
    await am.unlock()
    return c.json({ error: msg, backend: accountStore.name, steps }, 500)
  }
})

// GET /api/accounts/:id?storage=memory — view account details
app.get('/accounts/:id', async (c) => {
  const accountStore = getAccountStore(c.env, c.req.query('storage'))
  const secret = c.env.ACCOUNT_SECRET || 'default-dev-secret'
  const am = new AccountManager(accountStore, secret)
  const account = await am.getAccount(c.req.param('id'))
  if (!account) return c.json({ error: 'not found', backend: accountStore.name }, 404)
  return c.json({
    ...account,
    backend: accountStore.name,
    session: {
      ...account.session,
      cookies: account.session.cookies.slice(0, 80) + '...',
      csrfToken: account.session.csrfToken.slice(0, 20) + '...',
    },
  })
})

// DELETE /api/accounts/:id?storage=memory — delete one account
app.delete('/accounts/:id', async (c) => {
  const accountStore = getAccountStore(c.env, c.req.query('storage'))
  const secret = c.env.ACCOUNT_SECRET || 'default-dev-secret'
  const am = new AccountManager(accountStore, secret)
  const ok = await am.deleteAccount(c.req.param('id'))
  if (!ok) return c.json({ error: 'not found', backend: accountStore.name }, 404)
  return c.json({ ok: true, backend: accountStore.name })
})

// DELETE /api/accounts?storage=memory — delete ALL accounts
app.delete('/accounts', async (c) => {
  const accountStore = getAccountStore(c.env, c.req.query('storage'))
  const secret = c.env.ACCOUNT_SECRET || 'default-dev-secret'
  const am = new AccountManager(accountStore, secret)
  const deleted = await am.deleteAll()
  return c.json({ ok: true, deleted, backend: accountStore.name })
})

// POST /api/accounts/test-email — test email provider only
app.post('/accounts/test-email', async (c) => {
  const accountStore = getAccountStore(c.env, c.req.query('storage'))
  const secret = c.env.ACCOUNT_SECRET || 'default-dev-secret'
  const am = new AccountManager(accountStore, secret)
  const t0 = Date.now()
  try {
    const { createTempEmail } = await import('../email')
    const { client, provider } = await createTempEmail()
    await am.log({ timestamp: new Date().toISOString(), event: 'email_created', message: `[TEST] ${provider}: ${client.email()}`, provider, email: client.email(), durationMs: Date.now() - t0 })
    return c.json({ ok: true, backend: accountStore.name, provider, email: client.email(), durationMs: Date.now() - t0 })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    await am.log({ timestamp: new Date().toISOString(), event: 'error', message: `[TEST] Email failed: ${msg}`, error: msg, durationMs: Date.now() - t0 })
    return c.json({ ok: false, backend: accountStore.name, error: msg, durationMs: Date.now() - t0 }, 500)
  }
})

// ============================================================

app.onError((err, c) => {
  console.error('[API Error]', err.message)
  return c.json({ error: err.message }, 500)
})

export default app
