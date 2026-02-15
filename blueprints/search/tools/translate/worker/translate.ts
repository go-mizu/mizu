/**
 * Translation engine — text-level KV cache + batch translation API.
 * Used by both the page route (KV lookup) and Queue consumer (actual translation).
 *
 * KV schema: key = `t:{tl}:{hex16}`, value = `{sl}\t{translation}`, TTL = 30 days
 */

const TRANSLATE_APIS = [
  { name: 'google', base: 'https://translate.googleapis.com/translate_a/single', maxChars: 4000 },
  { name: 'mymemory', base: 'https://api.mymemory.translated.net/get', maxChars: 450 },
] as const

const BATCH_SEP = '\n\u2016\u2016\u2016\n'
const KV_TTL = 2592000 // 30 days

/* ── Hashing ── */

async function textHash(text: string): Promise<string> {
  const data = new TextEncoder().encode(text)
  const hash = await crypto.subtle.digest('SHA-256', data)
  const bytes = new Uint8Array(hash)
  let hex = ''
  for (let i = 0; i < 8; i++) hex += bytes[i].toString(16).padStart(2, '0')
  return hex
}

/* ── KV Lookup ── */

/**
 * Batch KV lookup for pre-translated texts.
 * Returns cached translations (Map), uncached texts (array), and detected source language.
 */
export async function lookupTranslations(
  kv: KVNamespace,
  texts: string[],
  tl: string,
): Promise<{ cached: Map<string, string>; uncached: string[]; detectedSl: string }> {
  if (texts.length === 0) return { cached: new Map(), uncached: [], detectedSl: '' }

  const hashes = await Promise.all(texts.map(t => textHash(t)))
  const keys = hashes.map(h => `t:${tl}:${h}`)
  const results = await Promise.all(keys.map(k => kv.get(k, 'text')))

  const cached = new Map<string, string>()
  const uncached: string[] = []
  let detectedSl = ''

  for (let i = 0; i < texts.length; i++) {
    const val = results[i]
    if (val) {
      // Value format: "sl\ttranslation"
      const tabIdx = val.indexOf('\t')
      if (tabIdx > 0 && tabIdx <= 5) {
        if (!detectedSl) detectedSl = val.slice(0, tabIdx)
        cached.set(texts[i], val.slice(tabIdx + 1))
      } else {
        cached.set(texts[i], val)
      }
    } else {
      uncached.push(texts[i])
    }
  }

  return { cached, uncached, detectedSl }
}

/* ── KV Write ── */

/**
 * Write translations to KV (text-level cache).
 * Each entry: key = `t:{tl}:{hash}`, value = `{sl}\t{translation}`, TTL = 30 days.
 */
export async function writeTranslations(
  kv: KVNamespace,
  translations: Map<string, string>,
  tl: string,
  sl: string,
): Promise<void> {
  const writes: Promise<void>[] = []
  for (const [orig, translated] of translations) {
    writes.push(
      textHash(orig).then(hex =>
        kv.put(`t:${tl}:${hex}`, `${sl}\t${translated}`, { expirationTtl: KV_TTL })
      )
    )
  }
  await Promise.all(writes)
}

/* ── Batch Translation (Google → MyMemory) ── */

/**
 * Translate all texts via Google Translate with MyMemory fallback.
 * Groups texts into batches by char limit, joins with separator, translates as batched API calls.
 */
export async function batchTranslate(
  texts: string[],
  sl: string,
  tl: string,
): Promise<{ translations: Map<string, string>; detectedSl: string }> {
  const translations = new Map<string, string>()
  let detectedSl = ''

  if (texts.length === 0) return { translations, detectedSl }

  const result = await translateWithGoogle(texts, sl, tl, translations)
  if (result.sl) detectedSl = result.sl

  if (result.failed.length > 0) {
    console.log(`[translate] FALLBACK mymemory remaining=${result.failed.length}`)
    const mmResult = await translateWithMyMemory(result.failed, sl, tl, translations)
    if (mmResult.sl && !detectedSl) detectedSl = mmResult.sl
  }

  console.log(`[translate] DONE translated=${translations.size}/${texts.length} sl=${detectedSl}`)
  return { translations, detectedSl }
}

async function translateWithGoogle(
  texts: string[],
  sl: string,
  tl: string,
  translations: Map<string, string>,
): Promise<{ failed: string[]; sl: string }> {
  const api = TRANSLATE_APIS[0]
  const batches = makeBatches(texts, api.maxChars)
  const failed: string[] = []
  let detectedSl = ''

  console.log(`[translate] google ${texts.length} texts → ${batches.length} batches`)

  for (let i = 0; i < batches.length; i++) {
    const batch = batches[i]
    const joined = batch.join(BATCH_SEP)

    try {
      const params = new URLSearchParams()
      params.set('client', 'gtx')
      params.set('sl', sl)
      params.set('tl', tl)
      params.set('dj', '1')
      params.append('dt', 't')

      let resp: Response
      if (joined.length <= 2000) {
        params.set('q', joined)
        resp = await fetch(`${api.base}?${params.toString()}`, {
          headers: { 'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' },
        })
      } else {
        const body = new URLSearchParams()
        body.set('q', joined)
        resp = await fetch(`${api.base}?${params.toString()}`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
          },
          body: body.toString(),
        })
      }

      if (!resp.ok) {
        console.log(`[translate] google FAIL batch=${i} status=${resp.status} texts=${batch.length}`)
        failed.push(...batch)
        continue
      }

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const data: any = await resp.json()
      if (!data.sentences) { failed.push(...batch); continue }
      if (data.src && !detectedSl) detectedSl = data.src

      const translatedJoined = data.sentences
        .filter((s: { trans?: string }) => s.trans != null)
        .map((s: { trans: string }) => s.trans)
        .join('')

      const parts = translatedJoined.split(/\s*\u2016\u2016\u2016\s*/)
      const matched = Math.min(parts.length, batch.length)

      for (let j = 0; j < matched; j++) {
        if (parts[j] && parts[j].trim()) translations.set(batch[j], parts[j])
        else failed.push(batch[j])
      }
      for (let j = matched; j < batch.length; j++) failed.push(batch[j])

      console.log(`[translate] google OK batch=${i} texts=${batch.length} matched=${matched}`)
    } catch (e) {
      console.log(`[translate] google ERROR batch=${i} err=${e instanceof Error ? e.message : e}`)
      failed.push(...batch)
    }
  }

  return { failed, sl: detectedSl }
}

async function translateWithMyMemory(
  texts: string[],
  sl: string,
  tl: string,
  translations: Map<string, string>,
): Promise<{ sl: string }> {
  const api = TRANSLATE_APIS[1]
  const langSl = sl === 'auto' ? 'en' : sl
  const batches = makeBatches(texts, api.maxChars)
  let detectedSl = ''

  console.log(`[translate] mymemory ${texts.length} texts → ${batches.length} batches`)

  for (let i = 0; i < batches.length; i++) {
    const batch = batches[i]
    const joined = batch.join(BATCH_SEP)

    try {
      const params = new URLSearchParams()
      params.set('q', joined)
      params.set('langpair', `${langSl}|${tl}`)

      const resp = await fetch(`${api.base}?${params.toString()}`, {
        headers: { 'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' },
      })

      if (!resp.ok) {
        console.log(`[translate] mymemory FAIL batch=${i} status=${resp.status}`)
        continue
      }

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const data: any = await resp.json()
      if (!data.responseData?.translatedText) continue

      if (data.responseData.detectedLanguage && !detectedSl) {
        detectedSl = data.responseData.detectedLanguage
      }

      const translatedJoined = data.responseData.translatedText as string
      const parts = translatedJoined.split(/\s*\u2016\u2016\u2016\s*/)
      const matched = Math.min(parts.length, batch.length)

      for (let j = 0; j < matched; j++) {
        if (parts[j] && parts[j].trim()) translations.set(batch[j], parts[j])
      }

      console.log(`[translate] mymemory OK batch=${i} texts=${batch.length} matched=${matched}`)
    } catch (e) {
      console.log(`[translate] mymemory ERROR batch=${i} err=${e instanceof Error ? e.message : e}`)
    }
  }

  return { sl: detectedSl }
}

function makeBatches(texts: string[], maxChars: number): string[][] {
  const batches: string[][] = []
  let current: string[] = []
  let len = 0

  for (const text of texts) {
    const addLen = text.length + BATCH_SEP.length
    if (len + addLen > maxChars && current.length > 0) {
      batches.push(current)
      current = []
      len = 0
    }
    current.push(text)
    len += addLen
  }
  if (current.length > 0) batches.push(current)
  return batches
}
