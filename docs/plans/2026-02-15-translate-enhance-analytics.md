# Translate Enhancement: Analytics, Chunking, Queue

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the MyMemory 500-char silent truncation, add full Cloudflare Analytics Engine telemetry, and enhance queue monitoring.

**Architecture:** Single Analytics Engine dataset with structured blob/double fields for all event types. MyMemory provider gains sentence-boundary chunking. Queue consumer writes analytics events for every operation. All routes instrument request timing and outcomes.

**Tech Stack:** Cloudflare Workers, Analytics Engine, Queues, KV, Hono, TypeScript

---

### Task 1: Add Analytics Engine binding and types

**Files:**
- Modify: `tools/translate/wrangler.toml`
- Modify: `tools/translate/worker/types.ts`

**Step 1: Add Analytics Engine dataset to wrangler.toml**

Add after the `[vars]` section in `wrangler.toml`:

```toml
[[analytics_engine_datasets]]
binding = "ANALYTICS"
dataset = "translate_events"
```

**Step 2: Add ANALYTICS to Env interface**

In `worker/types.ts`, add `ANALYTICS` to the `Env` interface:

```typescript
export interface Env {
  ASSETS?: { fetch: typeof fetch }
  BROWSER: Fetcher
  TRANSLATE_CACHE: KVNamespace
  TRANSLATE_QUEUE: Queue<TranslateMessage>
  ANALYTICS: AnalyticsEngineDataset
  ENVIRONMENT: string
}
```

**Step 3: Verify TypeScript compiles**

Run: `cd tools/translate && npx tsc --noEmit`
Expected: No errors (AnalyticsEngineDataset is a built-in CF Workers type)

**Step 4: Commit**

```bash
git add tools/translate/wrangler.toml tools/translate/worker/types.ts
git commit -m "feat(translate): add Analytics Engine binding and type"
```

---

### Task 2: Create analytics helper module

**Files:**
- Create: `tools/translate/worker/analytics.ts`

**Step 1: Create the analytics module**

Create `worker/analytics.ts`:

```typescript
/**
 * Analytics Engine helper — writes structured events.
 *
 * Single dataset: translate_events
 * Schema:
 *   blob1  = event type (translate|page|tts|detect|queue|error)
 *   blob2  = source lang or URL
 *   blob3  = target lang
 *   blob4  = provider (google|mymemory|libre)
 *   blob5  = extra context (error msg, render mode)
 *   blob6  = cache status (HIT|MISS)
 *   double1 = latency ms
 *   double2 = char count or text count
 *   double3 = success (1) or failure (0)
 *   double4 = cache hits
 *   double5 = total items
 */

import type { Env } from './types'

export interface AnalyticsEvent {
  event: string
  sl?: string
  tl?: string
  provider?: string
  extra?: string
  cache?: string
  latencyMs?: number
  chars?: number
  success?: boolean
  cacheHits?: number
  total?: number
}

export function track(env: Env, data: AnalyticsEvent): void {
  try {
    env.ANALYTICS?.writeDataPoint({
      blobs: [
        data.event,
        data.sl ?? '',
        data.tl ?? '',
        data.provider ?? '',
        data.extra ?? '',
        data.cache ?? '',
      ],
      doubles: [
        data.latencyMs ?? 0,
        data.chars ?? 0,
        data.success === false ? 0 : 1,
        data.cacheHits ?? 0,
        data.total ?? 0,
      ],
    })
  } catch {
    // Analytics should never break the request
  }
}
```

**Step 2: Verify TypeScript compiles**

Run: `cd tools/translate && npx tsc --noEmit`
Expected: No errors

**Step 3: Commit**

```bash
git add tools/translate/worker/analytics.ts
git commit -m "feat(translate): add analytics helper module"
```

---

### Task 3: MyMemory sentence-boundary chunking

**Files:**
- Modify: `tools/translate/worker/providers/mymemory.ts`

**Step 1: Replace truncation with chunking**

Replace the entire `mymemory.ts` with:

```typescript
import type { TranslateProvider } from './base'
import type { TranslateResult, DetectResult, Language } from '../types'
import { GOOGLE_LANGUAGES } from './google'

const BASE_URL = 'https://api.mymemory.translated.net/get'
const MAX_CHARS = 500

/**
 * Split text into chunks of <=MAX_CHARS at sentence boundaries.
 * Falls back to word boundaries if a single sentence exceeds MAX_CHARS.
 */
function chunkText(text: string): string[] {
  if (text.length <= MAX_CHARS) return [text]

  // Split on sentence-ending punctuation followed by whitespace
  const sentences = text.split(/(?<=[.!?\n])\s+/)
  const chunks: string[] = []
  let current = ''

  for (const sentence of sentences) {
    if (sentence.length > MAX_CHARS) {
      // Flush current buffer
      if (current) { chunks.push(current.trim()); current = '' }
      // Split long sentence at word boundaries
      const words = sentence.split(/\s+/)
      let wordChunk = ''
      for (const word of words) {
        if (wordChunk.length + word.length + 1 > MAX_CHARS && wordChunk) {
          chunks.push(wordChunk.trim())
          wordChunk = ''
        }
        wordChunk += (wordChunk ? ' ' : '') + word
      }
      if (wordChunk) chunks.push(wordChunk.trim())
    } else if (current.length + sentence.length + 1 > MAX_CHARS) {
      chunks.push(current.trim())
      current = sentence
    } else {
      current += (current ? ' ' : '') + sentence
    }
  }
  if (current.trim()) chunks.push(current.trim())
  return chunks
}

class MyMemoryProvider implements TranslateProvider {
  name = 'mymemory'

  async translate(text: string, from: string, to: string): Promise<TranslateResult> {
    const sourceLang = from === 'auto' ? 'en' : from
    const langpair = `${sourceLang}|${to}`

    const chunks = chunkText(text)
    const translations: string[] = []
    let detectedLanguage = sourceLang
    let match = 1.0

    for (const chunk of chunks) {
      const params = new URLSearchParams({ q: chunk, langpair })
      const resp = await fetch(`${BASE_URL}?${params.toString()}`)
      if (!resp.ok) throw new Error(`MyMemory HTTP ${resp.status}`)

      const data: any = await resp.json()
      const translated = data.responseData?.translatedText || ''
      translations.push(translated)

      if (data.responseData?.detectedLanguage && typeof data.responseData.detectedLanguage === 'string') {
        detectedLanguage = data.responseData.detectedLanguage
      }
      if (typeof data.responseData?.match === 'number') {
        match = Math.min(match, data.responseData.match)
      }
    }

    return {
      translation: translations.join(' '),
      detectedLanguage,
      confidence: match,
      pronunciation: null,
      alternatives: null,
      definitions: null,
      synonyms: null,
      examples: null,
      provider: this.name,
    }
  }

  async detect(text: string): Promise<DetectResult> {
    const result = await this.translate(text, 'auto', 'en')
    return {
      language: result.detectedLanguage,
      confidence: result.confidence,
    }
  }

  languages(): Language[] {
    return GOOGLE_LANGUAGES
  }
}

export const mymemoryProvider = new MyMemoryProvider()
```

**Step 2: Verify TypeScript compiles**

Run: `cd tools/translate && npx tsc --noEmit`
Expected: No errors

**Step 3: Commit**

```bash
git add tools/translate/worker/providers/mymemory.ts
git commit -m "fix(translate): replace MyMemory 500-char truncation with sentence chunking"
```

---

### Task 4: Instrument /api/translate route

**Files:**
- Modify: `tools/translate/worker/routes/translate.ts`

**Step 1: Add analytics tracking**

Replace `routes/translate.ts` with:

```typescript
import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { translateWithFallback } from '../providers/chain'
import { track } from '../analytics'

const route = new Hono<HonoEnv>()

route.post('/translate', async (c) => {
  const body = await c.req.json<{ text?: string; from?: string; to?: string }>()

  if (!body.text || typeof body.text !== 'string' || body.text.trim().length === 0) {
    return c.json({ error: 'Missing or empty "text" field' }, 400)
  }

  if (!body.to || typeof body.to !== 'string') {
    return c.json({ error: 'Missing "to" field (target language code)' }, 400)
  }

  if (body.text.length > 5000) {
    return c.json({ error: 'Text exceeds maximum length of 5000 characters' }, 400)
  }

  const from = body.from || 'auto'
  const t0 = Date.now()

  try {
    const result = await translateWithFallback(body.text, from, body.to)
    track(c.env, {
      event: 'translate',
      sl: result.detectedLanguage || from,
      tl: body.to,
      provider: result.provider,
      latencyMs: Date.now() - t0,
      chars: body.text.length,
      success: true,
    })
    return c.json(result)
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Translation failed'
    track(c.env, {
      event: 'translate',
      sl: from,
      tl: body.to,
      extra: message,
      latencyMs: Date.now() - t0,
      chars: body.text.length,
      success: false,
    })
    return c.json({ error: message }, 502)
  }
})

export default route
```

**Step 2: Verify TypeScript compiles**

Run: `cd tools/translate && npx tsc --noEmit`

**Step 3: Commit**

```bash
git add tools/translate/worker/routes/translate.ts
git commit -m "feat(translate): add analytics to /api/translate route"
```

---

### Task 5: Instrument /api/detect route

**Files:**
- Modify: `tools/translate/worker/routes/detect.ts`

**Step 1: Add analytics tracking**

Replace `routes/detect.ts` with:

```typescript
import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { detectWithFallback } from '../providers/chain'
import { track } from '../analytics'

const route = new Hono<HonoEnv>()

route.post('/detect', async (c) => {
  const body = await c.req.json<{ text?: string }>()

  if (!body.text || typeof body.text !== 'string' || body.text.trim().length === 0) {
    return c.json({ error: 'Missing or empty "text" field' }, 400)
  }

  const t0 = Date.now()

  try {
    const result = await detectWithFallback(body.text)
    track(c.env, {
      event: 'detect',
      sl: result.language,
      latencyMs: Date.now() - t0,
      chars: body.text.length,
      success: true,
    })
    return c.json(result)
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Detection failed'
    track(c.env, {
      event: 'detect',
      extra: message,
      latencyMs: Date.now() - t0,
      chars: body.text.length,
      success: false,
    })
    return c.json({ error: message }, 502)
  }
})

export default route
```

**Step 2: Commit**

```bash
git add tools/translate/worker/routes/detect.ts
git commit -m "feat(translate): add analytics to /api/detect route"
```

---

### Task 6: Instrument /api/tts route

**Files:**
- Modify: `tools/translate/worker/routes/tts.ts`

**Step 1: Add analytics tracking**

Replace `routes/tts.ts` with:

```typescript
import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { track } from '../analytics'

const TTS_BASE = 'https://translate.google.com/translate_tts'

const route = new Hono<HonoEnv>()

route.get('/tts', async (c) => {
  const tl = c.req.query('tl')
  const q = c.req.query('q')

  if (!tl || typeof tl !== 'string') {
    return c.json({ error: 'Missing "tl" query parameter (target language)' }, 400)
  }

  if (!q || typeof q !== 'string' || q.trim().length === 0) {
    return c.json({ error: 'Missing or empty "q" query parameter (text)' }, 400)
  }

  if (q.length > 200) {
    return c.json({ error: 'Text exceeds maximum length of 200 characters for TTS' }, 400)
  }

  const params = new URLSearchParams({
    ie: 'UTF-8',
    client: 'tw-ob',
    tl,
    q,
    total: '1',
    idx: '0',
    textlen: String(q.length),
  })

  const t0 = Date.now()

  try {
    const resp = await fetch(`${TTS_BASE}?${params.toString()}`, {
      headers: {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
      },
    })

    if (!resp.ok) {
      track(c.env, {
        event: 'tts',
        tl,
        extra: `upstream ${resp.status}`,
        latencyMs: Date.now() - t0,
        chars: q.length,
        success: false,
      })
      return c.json({ error: `TTS upstream returned ${resp.status}` }, 502)
    }

    const audio = await resp.arrayBuffer()
    track(c.env, {
      event: 'tts',
      tl,
      latencyMs: Date.now() - t0,
      chars: q.length,
      success: true,
    })
    return new Response(audio, {
      headers: {
        'Content-Type': 'audio/mpeg',
        'Cache-Control': 'public, max-age=86400',
      },
    })
  } catch (err) {
    const message = err instanceof Error ? err.message : 'TTS request failed'
    track(c.env, {
      event: 'tts',
      tl,
      extra: message,
      latencyMs: Date.now() - t0,
      chars: q.length,
      success: false,
    })
    return c.json({ error: message }, 502)
  }
})

export default route
```

**Step 2: Commit**

```bash
git add tools/translate/worker/routes/tts.ts
git commit -m "feat(translate): add analytics to /api/tts route"
```

---

### Task 7: Instrument /page routes

**Files:**
- Modify: `tools/translate/worker/routes/page.ts`

**Step 1: Add analytics tracking to the main page translation route**

Add import at top of `routes/page.ts`:

```typescript
import { track } from '../analytics'
```

Then add `track()` calls at these points in the `GET /page/:tl` handler:

1. **After page cache HIT** (line ~60, before return):
```typescript
track(c.env, {
  event: 'page',
  sl: targetUrl,
  tl,
  cache: 'HIT',
  latencyMs: Date.now() - t0,
  chars: cached.length,
  success: true,
})
```

2. **After successful page translation** (line ~187, before return):
```typescript
track(c.env, {
  event: 'page',
  sl: targetUrl,
  tl,
  extra: usedBrowser ? 'browser' : 'fetch',
  cache: 'MISS',
  latencyMs: Date.now() - t0,
  chars: translatedHtml.length,
  success: true,
  cacheHits: cached.size,
  total: texts.length,
})
```

3. **After browser render failure** (line ~142, before error return):
```typescript
track(c.env, {
  event: 'page',
  sl: targetUrl,
  tl,
  extra: message,
  cache: 'MISS',
  latencyMs: Date.now() - t0,
  success: false,
})
```

**Step 2: Verify TypeScript compiles**

Run: `cd tools/translate && npx tsc --noEmit`

**Step 3: Commit**

```bash
git add tools/translate/worker/routes/page.ts
git commit -m "feat(translate): add analytics to page translation routes"
```

---

### Task 8: Instrument queue consumer + enhance DLQ

**Files:**
- Modify: `tools/translate/worker/queue.ts`

**Step 1: Add analytics tracking and enhanced DLQ**

Replace `queue.ts` with:

```typescript
/**
 * Queue consumer — translates text batches and writes results to KV.
 *
 * Message format: { texts: string[], tl: string }
 * On success: each translation written to KV as `t:{tl}:{hash}` → `{sl}\t{translation}`.
 * On failure: retries up to 3 times with exponential backoff.
 * Analytics: tracks every queue operation (received, translated, retried, dead-lettered).
 */

import type { Env, TranslateMessage } from './types'
import { batchTranslate, writeTranslations } from './translate'
import { track } from './analytics'

export async function handleQueue(
  batch: MessageBatch<TranslateMessage>,
  env: Env,
): Promise<void> {
  console.log(`[queue] BATCH received=${batch.messages.length} queue=${batch.queue}`)

  for (const msg of batch.messages) {
    const { texts, tl } = msg.body
    const attempt = msg.attempts
    const t0 = Date.now()

    console.log(`[queue] MSG id=${msg.id} texts=${texts.length} tl=${tl} attempt=${attempt}`)

    try {
      const { translations, detectedSl } = await batchTranslate(texts, 'auto', tl)
      const sl = detectedSl || 'en'

      console.log(`[queue] TRANSLATED ${translations.size}/${texts.length} sl=${sl}`)

      await writeTranslations(env.TRANSLATE_CACHE, translations, tl, sl)

      console.log(`[queue] KV_WRITTEN ${translations.size} entries`)

      track(env, {
        event: 'queue',
        sl,
        tl,
        latencyMs: Date.now() - t0,
        chars: texts.reduce((sum, t) => sum + t.length, 0),
        success: true,
        cacheHits: translations.size,
        total: texts.length,
      })

      msg.ack()
    } catch (e) {
      const err = e instanceof Error ? e.message : String(e)
      console.log(`[queue] ERROR id=${msg.id} err=${err} attempt=${attempt}`)

      if (attempt < 3) {
        track(env, {
          event: 'queue',
          tl,
          extra: `retry:${attempt + 1} ${err}`,
          latencyMs: Date.now() - t0,
          chars: texts.reduce((sum, t) => sum + t.length, 0),
          success: false,
          total: texts.length,
        })
        msg.retry({ delaySeconds: (attempt + 1) * 10 })
      } else {
        console.log(`[queue] DEAD_LETTER id=${msg.id} texts=${texts.length} tl=${tl} err=${err}`)
        track(env, {
          event: 'error',
          tl,
          provider: 'queue_dlq',
          extra: err,
          latencyMs: Date.now() - t0,
          chars: texts.reduce((sum, t) => sum + t.length, 0),
          success: false,
          total: texts.length,
        })
        msg.ack() // give up after 3 attempts
      }
    }
  }
}
```

**Step 2: Verify TypeScript compiles**

Run: `cd tools/translate && npx tsc --noEmit`

**Step 3: Commit**

```bash
git add tools/translate/worker/queue.ts
git commit -m "feat(translate): add analytics to queue consumer, enhance DLQ tracking"
```

---

### Task 9: Build and verify

**Step 1: Run TypeScript check**

Run: `cd tools/translate && npx tsc --noEmit`
Expected: No errors

**Step 2: Run Vite build**

Run: `cd tools/translate && npx vite build`
Expected: Build succeeds

**Step 3: Final commit (if any fixes needed)**

```bash
git add -A tools/translate/
git commit -m "chore(translate): fix any build issues"
```
