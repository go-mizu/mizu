# Translate App Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a full-stack translation app (Hono API + React frontend) deployed as a Cloudflare Worker at `https://translate.go-mizu.workers.dev`.

**Architecture:** Multi-provider translation backend (Google Translate primary, MyMemory + LibreTranslate fallbacks) with adapter pattern. React 19 SPA frontend with shadcn design style, Tailwind CSS 4, Zustand state management. Follows cc-viewer project scaffold exactly.

**Tech Stack:** Hono, React 19, Vite, Tailwind CSS 4, Zustand, React Query, CVA, Lucide React, Cloudflare Workers

**Spec:** `spec/0533_translate.md`
**Reference project:** `tools/cc-viewer/` (same scaffold pattern)

---

### Task 1: Project Scaffold

**Files:**
- Create: `tools/translate/package.json`
- Create: `tools/translate/wrangler.toml`
- Create: `tools/translate/tsconfig.json`
- Create: `tools/translate/vite.config.ts`
- Create: `tools/translate/index.html`

**Step 1: Create package.json**

```json
{
  "name": "translate",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "wrangler dev",
    "build": "vite build",
    "deploy": "vite build && wrangler deploy",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "hono": "^4.11.9",
    "react": "^19.1.0",
    "react-dom": "^19.1.0",
    "@tanstack/react-query": "^5.75.0",
    "clsx": "^2.1.1",
    "tailwind-merge": "^3.3.0",
    "class-variance-authority": "^0.7.1",
    "lucide-react": "^0.487.0",
    "zustand": "^5.0.0"
  },
  "devDependencies": {
    "@cloudflare/workers-types": "^4.20260210.0",
    "@tailwindcss/vite": "^4.1.4",
    "@types/react": "^19.1.2",
    "@types/react-dom": "^19.1.2",
    "@vitejs/plugin-react": "^4.4.1",
    "tailwindcss": "^4.1.4",
    "typescript": "^5.9.3",
    "vite": "^6.3.0",
    "wrangler": "^4.63.0"
  }
}
```

**Step 2: Create wrangler.toml**

```toml
name = "translate"
main = "worker/index.ts"
compatibility_date = "2026-02-09"
compatibility_flags = ["nodejs_compat"]

[assets]
directory = "./dist"
binding = "ASSETS"
not_found_handling = "single-page-application"

[vars]
ENVIRONMENT = "production"
```

**Step 3: Create tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ESNext",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "lib": ["ESNext", "DOM", "DOM.Iterable"],
    "types": ["@cloudflare/workers-types", "@types/node"],
    "jsx": "react-jsx",
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "esModuleInterop": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": ["src/**/*.ts", "src/**/*.tsx", "worker/**/*.ts"],
  "exclude": ["node_modules", "dist"]
}
```

**Step 4: Create vite.config.ts**

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
```

**Step 5: Create index.html**

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="description" content="Free multi-provider translation with definitions, synonyms, and pronunciation" />
    <title>Translate</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

**Step 6: Install dependencies**

Run: `cd tools/translate && npm install`
Expected: `node_modules/` created, no errors

**Step 7: Commit**

```bash
git add tools/translate/package.json tools/translate/wrangler.toml tools/translate/tsconfig.json tools/translate/vite.config.ts tools/translate/index.html tools/translate/package-lock.json
git commit -m "feat(translate): scaffold project with Hono + React + Tailwind"
```

---

### Task 2: Worker Types + Hono Entry Point

**Files:**
- Create: `tools/translate/worker/types.ts`
- Create: `tools/translate/worker/index.ts`

**Step 1: Create worker/types.ts**

```typescript
export interface Env {
  ASSETS?: { fetch: typeof fetch }
  ENVIRONMENT: string
}

export type HonoEnv = {
  Bindings: Env
}

// --- Provider types ---

export interface Language {
  code: string
  name: string
}

export interface Definition {
  partOfSpeech: string
  entries: Array<{
    definition: string
    example: string | null
  }>
}

export interface SynonymGroup {
  partOfSpeech: string
  entries: string[][]
}

export interface Pronunciation {
  sourcePhonetic: string | null
  targetTranslit: string | null
}

export interface TranslateResult {
  translation: string
  detectedLanguage: string
  confidence: number
  pronunciation: Pronunciation | null
  alternatives: string[] | null
  definitions: Definition[] | null
  synonyms: SynonymGroup[] | null
  examples: string[] | null
  provider: string
}

export interface TranslateRequest {
  text: string
  from: string
  to: string
}

export interface DetectRequest {
  text: string
}

export interface DetectResult {
  language: string
  confidence: number
}
```

**Step 2: Create worker/index.ts**

```typescript
import { Hono } from 'hono'
import { cors } from 'hono/cors'
import type { Env, HonoEnv } from './types'
import translateRoute from './routes/translate'
import languagesRoute from './routes/languages'
import detectRoute from './routes/detect'
import ttsRoute from './routes/tts'

const app = new Hono<HonoEnv>()

app.use('/api/*', cors({
  origin: '*',
  allowMethods: ['GET', 'POST', 'OPTIONS'],
  allowHeaders: ['Content-Type'],
}))

app.get('/api/health', (c) => c.json({ status: 'ok' }))
app.route('/api', translateRoute)
app.route('/api', languagesRoute)
app.route('/api', detectRoute)
app.route('/api', ttsRoute)

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url)

    if (url.pathname.startsWith('/api/')) {
      return app.fetch(request, env, ctx)
    }

    if (env.ASSETS) {
      return env.ASSETS.fetch(request)
    }

    return new Response(JSON.stringify({
      error: 'No static assets available',
      detail: 'Run "npm run build" to generate the SPA, or use /api/ endpoints directly.',
      api: {
        health: '/api/health',
        translate: 'POST /api/translate',
        languages: 'GET /api/languages',
        detect: 'POST /api/detect',
        tts: 'GET /api/tts?tl=LANG&q=TEXT',
      },
    }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  },
}
```

**Step 3: Create placeholder route files** (so TypeScript doesn't error)

Create four empty route stubs that export a Hono app:

`worker/routes/translate.ts`:
```typescript
import { Hono } from 'hono'
import type { HonoEnv } from '../types'
const app = new Hono<HonoEnv>()
export default app
```

`worker/routes/languages.ts` — same pattern.
`worker/routes/detect.ts` — same pattern.
`worker/routes/tts.ts` — same pattern.

**Step 4: Verify typecheck**

Run: `cd tools/translate && npx tsc --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add tools/translate/worker/
git commit -m "feat(translate): add worker types and Hono entry point"
```

---

### Task 3: Google Translate Provider

**Files:**
- Create: `tools/translate/worker/providers/base.ts`
- Create: `tools/translate/worker/providers/google.ts`

**Step 1: Create providers/base.ts**

```typescript
import type { TranslateResult, DetectResult, Language } from '../types'

export interface TranslateProvider {
  name: string
  translate(text: string, from: string, to: string): Promise<TranslateResult>
  detect(text: string): Promise<DetectResult>
  languages(): Language[]
}
```

**Step 2: Create providers/google.ts**

This is the primary provider. It calls `translate.googleapis.com/translate_a/single` with `client=gtx` and `dj=1`.

Key implementation details:
- GET for text ≤2000 chars, POST for longer text
- `dt` params: `t,bd,at,ex,md,ss,rw,rm,ld` for rich data
- Response has `sentences[]` (translation), `dict[]` (back-translations), `definitions[]`, `synsets[]`, `examples.example[]`
- Rich fields only populated for single-word queries
- `sentences[last].src_translit` = source phonetic, `sentences[last].translit` = target transliteration

```typescript
import type { TranslateProvider } from './base'
import type { TranslateResult, DetectResult, Language } from '../types'

const DT_RICH = ['t', 'bd', 'at', 'ex', 'md', 'ss', 'rw', 'rm', 'ld']
const DT_BASIC = ['t', 'rm']
const BASE_URL = 'https://translate.googleapis.com/translate_a/single'

export const GOOGLE_LANGUAGES: Language[] = [
  { code: 'auto', name: 'Auto-detect' },
  { code: 'af', name: 'Afrikaans' },
  { code: 'sq', name: 'Albanian' },
  { code: 'am', name: 'Amharic' },
  { code: 'ar', name: 'Arabic' },
  { code: 'hy', name: 'Armenian' },
  { code: 'az', name: 'Azerbaijani' },
  { code: 'eu', name: 'Basque' },
  { code: 'be', name: 'Belarusian' },
  { code: 'bn', name: 'Bengali' },
  { code: 'bs', name: 'Bosnian' },
  { code: 'bg', name: 'Bulgarian' },
  { code: 'ca', name: 'Catalan' },
  { code: 'ceb', name: 'Cebuano' },
  { code: 'zh-CN', name: 'Chinese (Simplified)' },
  { code: 'zh-TW', name: 'Chinese (Traditional)' },
  { code: 'co', name: 'Corsican' },
  { code: 'hr', name: 'Croatian' },
  { code: 'cs', name: 'Czech' },
  { code: 'da', name: 'Danish' },
  { code: 'nl', name: 'Dutch' },
  { code: 'en', name: 'English' },
  { code: 'eo', name: 'Esperanto' },
  { code: 'et', name: 'Estonian' },
  { code: 'fi', name: 'Finnish' },
  { code: 'fr', name: 'French' },
  { code: 'fy', name: 'Frisian' },
  { code: 'gl', name: 'Galician' },
  { code: 'ka', name: 'Georgian' },
  { code: 'de', name: 'German' },
  { code: 'el', name: 'Greek' },
  { code: 'gu', name: 'Gujarati' },
  { code: 'ht', name: 'Haitian Creole' },
  { code: 'ha', name: 'Hausa' },
  { code: 'haw', name: 'Hawaiian' },
  { code: 'he', name: 'Hebrew' },
  { code: 'hi', name: 'Hindi' },
  { code: 'hmn', name: 'Hmong' },
  { code: 'hu', name: 'Hungarian' },
  { code: 'is', name: 'Icelandic' },
  { code: 'ig', name: 'Igbo' },
  { code: 'id', name: 'Indonesian' },
  { code: 'ga', name: 'Irish' },
  { code: 'it', name: 'Italian' },
  { code: 'ja', name: 'Japanese' },
  { code: 'jv', name: 'Javanese' },
  { code: 'kn', name: 'Kannada' },
  { code: 'kk', name: 'Kazakh' },
  { code: 'km', name: 'Khmer' },
  { code: 'rw', name: 'Kinyarwanda' },
  { code: 'ko', name: 'Korean' },
  { code: 'ku', name: 'Kurdish' },
  { code: 'ky', name: 'Kyrgyz' },
  { code: 'lo', name: 'Lao' },
  { code: 'la', name: 'Latin' },
  { code: 'lv', name: 'Latvian' },
  { code: 'lt', name: 'Lithuanian' },
  { code: 'lb', name: 'Luxembourgish' },
  { code: 'mk', name: 'Macedonian' },
  { code: 'mg', name: 'Malagasy' },
  { code: 'ms', name: 'Malay' },
  { code: 'ml', name: 'Malayalam' },
  { code: 'mt', name: 'Maltese' },
  { code: 'mi', name: 'Maori' },
  { code: 'mr', name: 'Marathi' },
  { code: 'mn', name: 'Mongolian' },
  { code: 'my', name: 'Myanmar (Burmese)' },
  { code: 'ne', name: 'Nepali' },
  { code: 'no', name: 'Norwegian' },
  { code: 'ny', name: 'Nyanja (Chichewa)' },
  { code: 'or', name: 'Odia (Oriya)' },
  { code: 'ps', name: 'Pashto' },
  { code: 'fa', name: 'Persian' },
  { code: 'pl', name: 'Polish' },
  { code: 'pt', name: 'Portuguese' },
  { code: 'pa', name: 'Punjabi' },
  { code: 'ro', name: 'Romanian' },
  { code: 'ru', name: 'Russian' },
  { code: 'sm', name: 'Samoan' },
  { code: 'gd', name: 'Scots Gaelic' },
  { code: 'sr', name: 'Serbian' },
  { code: 'st', name: 'Sesotho' },
  { code: 'sn', name: 'Shona' },
  { code: 'sd', name: 'Sindhi' },
  { code: 'si', name: 'Sinhala (Sinhalese)' },
  { code: 'sk', name: 'Slovak' },
  { code: 'sl', name: 'Slovenian' },
  { code: 'so', name: 'Somali' },
  { code: 'es', name: 'Spanish' },
  { code: 'su', name: 'Sundanese' },
  { code: 'sw', name: 'Swahili' },
  { code: 'sv', name: 'Swedish' },
  { code: 'tl', name: 'Tagalog (Filipino)' },
  { code: 'tg', name: 'Tajik' },
  { code: 'ta', name: 'Tamil' },
  { code: 'tt', name: 'Tatar' },
  { code: 'te', name: 'Telugu' },
  { code: 'th', name: 'Thai' },
  { code: 'tr', name: 'Turkish' },
  { code: 'tk', name: 'Turkmen' },
  { code: 'uk', name: 'Ukrainian' },
  { code: 'ur', name: 'Urdu' },
  { code: 'ug', name: 'Uyghur' },
  { code: 'uz', name: 'Uzbek' },
  { code: 'vi', name: 'Vietnamese' },
  { code: 'cy', name: 'Welsh' },
  { code: 'xh', name: 'Xhosa' },
  { code: 'yi', name: 'Yiddish' },
  { code: 'yo', name: 'Yoruba' },
  { code: 'zu', name: 'Zulu' },
]

function buildURL(text: string, from: string, to: string, rich: boolean): string {
  const dt = rich ? DT_RICH : DT_BASIC
  const params = new URLSearchParams({
    client: 'gtx',
    sl: from === 'auto' ? 'auto' : from,
    tl: to,
    dj: '1',
  })
  for (const d of dt) params.append('dt', d)
  if (text.length <= 2000) {
    params.set('q', text)
  }
  return `${BASE_URL}?${params}`
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function parseResponse(data: any): TranslateResult {
  // Concatenate sentence translations
  const sentences = data.sentences || []
  const translation = sentences
    .filter((s: any) => s.trans !== undefined)
    .map((s: any) => s.trans)
    .join('')

  // Pronunciation from last sentence
  const lastSentence = sentences[sentences.length - 1] || {}
  const sourcePhonetic = lastSentence.src_translit || null
  const targetTranslit = lastSentence.translit || null

  // Alternative translations
  const altTranslations = data.alternative_translations || []
  const alternatives: string[] = []
  for (const at of altTranslations) {
    for (const alt of at.alternative || []) {
      if (alt.word_postproc && alt.word_postproc !== translation) {
        alternatives.push(alt.word_postproc)
      }
    }
  }

  // Definitions (monolingual)
  const defs = (data.definitions || []).map((d: any) => ({
    partOfSpeech: d.pos,
    entries: (d.entry || []).map((e: any) => ({
      definition: e.gloss,
      example: e.example || null,
    })),
  }))

  // Synonyms
  const syns = (data.synsets || []).map((s: any) => ({
    partOfSpeech: s.pos,
    entries: (s.entry || []).map((e: any) => e.synonym || []),
  }))

  // Examples
  const exampleList = (data.examples?.example || []).map((e: any) => e.text)

  return {
    translation,
    detectedLanguage: data.src || 'unknown',
    confidence: data.confidence ?? data.ld_result?.srclangs_confidences?.[0] ?? 0,
    pronunciation: (sourcePhonetic || targetTranslit)
      ? { sourcePhonetic, targetTranslit }
      : null,
    alternatives: alternatives.length > 0 ? alternatives : null,
    definitions: defs.length > 0 ? defs : null,
    synonyms: syns.length > 0 ? syns : null,
    examples: exampleList.length > 0 ? exampleList : null,
    provider: 'google',
  }
}

export const googleProvider: TranslateProvider = {
  name: 'google',

  async translate(text: string, from: string, to: string): Promise<TranslateResult> {
    const isSingleWord = text.trim().split(/\s+/).length === 1
    const url = buildURL(text, from, to, isSingleWord)

    const init: RequestInit = {}
    if (text.length > 2000) {
      init.method = 'POST'
      init.headers = { 'Content-Type': 'application/x-www-form-urlencoded' }
      init.body = new URLSearchParams({ q: text })
    }

    const res = await fetch(url, init)
    if (!res.ok) {
      throw new Error(`Google Translate error: ${res.status}`)
    }
    const data = await res.json()
    return parseResponse(data)
  },

  async detect(text: string): Promise<DetectResult> {
    const params = new URLSearchParams({
      client: 'gtx', sl: 'auto', tl: 'en', dj: '1', q: text.slice(0, 200),
    })
    params.append('dt', 't')
    params.append('dt', 'ld')
    const res = await fetch(`${BASE_URL}?${params}`)
    if (!res.ok) throw new Error(`Google detect error: ${res.status}`)
    const data: any = await res.json()
    return {
      language: data.src || 'unknown',
      confidence: data.ld_result?.srclangs_confidences?.[0] ?? data.confidence ?? 0,
    }
  },

  languages(): Language[] {
    return GOOGLE_LANGUAGES
  },
}
```

**Step 3: Verify typecheck**

Run: `cd tools/translate && npx tsc --noEmit`
Expected: No errors

**Step 4: Commit**

```bash
git add tools/translate/worker/providers/
git commit -m "feat(translate): add Google Translate provider with rich data parsing"
```

---

### Task 4: MyMemory + LibreTranslate Providers + Fallback Chain

**Files:**
- Create: `tools/translate/worker/providers/mymemory.ts`
- Create: `tools/translate/worker/providers/libre.ts`
- Create: `tools/translate/worker/providers/chain.ts`

**Step 1: Create providers/mymemory.ts**

```typescript
import type { TranslateProvider } from './base'
import type { TranslateResult, DetectResult, Language } from '../types'
import { GOOGLE_LANGUAGES } from './google'

const BASE_URL = 'https://api.mymemory.translated.net/get'

export const myMemoryProvider: TranslateProvider = {
  name: 'mymemory',

  async translate(text: string, from: string, to: string): Promise<TranslateResult> {
    const sl = from === 'auto' ? 'en' : from  // MyMemory doesn't support auto
    const params = new URLSearchParams({
      q: text.slice(0, 500),  // MyMemory limit
      langpair: `${sl}|${to}`,
    })
    const res = await fetch(`${BASE_URL}?${params}`)
    if (!res.ok) throw new Error(`MyMemory error: ${res.status}`)
    const data: any = await res.json()
    if (data.responseStatus !== 200) {
      throw new Error(`MyMemory: ${data.responseDetails || 'unknown error'}`)
    }
    return {
      translation: data.responseData.translatedText,
      detectedLanguage: data.responseData.detectedLanguage || sl,
      confidence: data.responseData.match ?? 0,
      pronunciation: null,
      alternatives: null,
      definitions: null,
      synonyms: null,
      examples: null,
      provider: 'mymemory',
    }
  },

  async detect(text: string): Promise<DetectResult> {
    // MyMemory doesn't have a dedicated detect endpoint; translate to en and check
    const result = await this.translate(text.slice(0, 100), 'auto', 'en')
    return { language: result.detectedLanguage, confidence: result.confidence }
  },

  languages(): Language[] {
    return GOOGLE_LANGUAGES.filter(l => l.code !== 'auto')
  },
}
```

**Step 2: Create providers/libre.ts**

```typescript
import type { TranslateProvider } from './base'
import type { TranslateResult, DetectResult, Language } from '../types'

const BASE_URL = 'https://libretranslate.com'

export const libreProvider: TranslateProvider = {
  name: 'libre',

  async translate(text: string, from: string, to: string): Promise<TranslateResult> {
    const res = await fetch(`${BASE_URL}/translate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        q: text.slice(0, 5000),
        source: from === 'auto' ? 'auto' : from,
        target: to,
      }),
    })
    if (!res.ok) throw new Error(`LibreTranslate error: ${res.status}`)
    const data: any = await res.json()
    return {
      translation: data.translatedText,
      detectedLanguage: data.detectedLanguage?.language || from,
      confidence: data.detectedLanguage?.confidence ?? 0,
      pronunciation: null,
      alternatives: null,
      definitions: null,
      synonyms: null,
      examples: null,
      provider: 'libre',
    }
  },

  async detect(text: string): Promise<DetectResult> {
    const res = await fetch(`${BASE_URL}/detect`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ q: text.slice(0, 200) }),
    })
    if (!res.ok) throw new Error(`LibreTranslate detect error: ${res.status}`)
    const data: any = await res.json()
    const best = data[0] || { language: 'unknown', confidence: 0 }
    return { language: best.language, confidence: best.confidence }
  },

  languages(): Language[] {
    // LibreTranslate supports fewer languages; return the standard list
    // and let failures fall through to error handling
    return []
  },
}
```

**Step 3: Create providers/chain.ts**

```typescript
import type { TranslateProvider } from './base'
import type { TranslateResult, DetectResult, Language } from '../types'
import { googleProvider } from './google'
import { myMemoryProvider } from './mymemory'
import { libreProvider } from './libre'

const providers: TranslateProvider[] = [googleProvider, myMemoryProvider, libreProvider]

export async function translateWithFallback(
  text: string,
  from: string,
  to: string
): Promise<TranslateResult> {
  let lastError: Error | null = null
  for (const provider of providers) {
    try {
      return await provider.translate(text, from, to)
    } catch (err) {
      lastError = err instanceof Error ? err : new Error(String(err))
    }
  }
  throw lastError || new Error('All translation providers failed')
}

export async function detectWithFallback(text: string): Promise<DetectResult> {
  let lastError: Error | null = null
  for (const provider of providers) {
    try {
      return await provider.detect(text)
    } catch (err) {
      lastError = err instanceof Error ? err : new Error(String(err))
    }
  }
  throw lastError || new Error('All detection providers failed')
}

export function allLanguages(): Language[] {
  return googleProvider.languages()
}
```

**Step 4: Verify typecheck**

Run: `cd tools/translate && npx tsc --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add tools/translate/worker/providers/
git commit -m "feat(translate): add MyMemory + LibreTranslate providers and fallback chain"
```

---

### Task 5: API Routes

**Files:**
- Modify: `tools/translate/worker/routes/translate.ts`
- Modify: `tools/translate/worker/routes/languages.ts`
- Modify: `tools/translate/worker/routes/detect.ts`
- Modify: `tools/translate/worker/routes/tts.ts`

**Step 1: Implement translate route**

`worker/routes/translate.ts`:
```typescript
import { Hono } from 'hono'
import type { HonoEnv, TranslateRequest } from '../types'
import { translateWithFallback } from '../providers/chain'

const app = new Hono<HonoEnv>()

app.post('/translate', async (c) => {
  const body = await c.req.json<TranslateRequest>()
  if (!body.text || !body.to) {
    return c.json({ error: 'Missing required fields: text, to' }, 400)
  }
  if (body.text.length > 5000) {
    return c.json({ error: 'Text exceeds 5000 character limit' }, 400)
  }
  try {
    const result = await translateWithFallback(body.text, body.from || 'auto', body.to)
    return c.json(result)
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Translation failed'
    return c.json({ error: message }, 502)
  }
})

export default app
```

**Step 2: Implement languages route**

`worker/routes/languages.ts`:
```typescript
import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { allLanguages } from '../providers/chain'

const app = new Hono<HonoEnv>()

app.get('/languages', (c) => {
  return c.json({ languages: allLanguages() })
})

export default app
```

**Step 3: Implement detect route**

`worker/routes/detect.ts`:
```typescript
import { Hono } from 'hono'
import type { HonoEnv, DetectRequest } from '../types'
import { detectWithFallback } from '../providers/chain'

const app = new Hono<HonoEnv>()

app.post('/detect', async (c) => {
  const body = await c.req.json<DetectRequest>()
  if (!body.text) {
    return c.json({ error: 'Missing required field: text' }, 400)
  }
  try {
    const result = await detectWithFallback(body.text)
    return c.json(result)
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Detection failed'
    return c.json({ error: message }, 502)
  }
})

export default app
```

**Step 4: Implement TTS proxy route**

`worker/routes/tts.ts`:
```typescript
import { Hono } from 'hono'
import type { HonoEnv } from '../types'

const app = new Hono<HonoEnv>()

app.get('/tts', async (c) => {
  const tl = c.req.query('tl')
  const q = c.req.query('q')
  if (!tl || !q) {
    return c.json({ error: 'Missing required params: tl, q' }, 400)
  }
  if (q.length > 200) {
    return c.json({ error: 'Text exceeds 200 character TTS limit' }, 400)
  }
  const ttsUrl = `https://translate.google.com/translate_tts?ie=UTF-8&client=tw-ob&tl=${encodeURIComponent(tl)}&q=${encodeURIComponent(q)}&total=1&idx=0&textlen=${q.length}`
  const res = await fetch(ttsUrl)
  if (!res.ok) {
    return c.json({ error: `TTS fetch failed: ${res.status}` }, 502)
  }
  return new Response(res.body, {
    headers: {
      'Content-Type': 'audio/mpeg',
      'Cache-Control': 'public, max-age=86400',
    },
  })
})

export default app
```

**Step 5: Verify typecheck**

Run: `cd tools/translate && npx tsc --noEmit`
Expected: No errors

**Step 6: Test with wrangler dev**

Run: `cd tools/translate && npx wrangler dev --port 8790`
Then test: `curl -X POST http://localhost:8790/api/translate -H 'Content-Type: application/json' -d '{"text":"hello","from":"auto","to":"vi"}'`
Expected: JSON response with `translation`, `detectedLanguage`, `provider: "google"`

**Step 7: Commit**

```bash
git add tools/translate/worker/routes/
git commit -m "feat(translate): implement translate, languages, detect, and TTS API routes"
```

---

### Task 6: React SPA Skeleton + Tailwind Setup

**Files:**
- Create: `tools/translate/src/main.tsx`
- Create: `tools/translate/src/App.tsx`
- Create: `tools/translate/src/index.css`
- Create: `tools/translate/src/lib/utils.ts`
- Create: `tools/translate/src/vite-env.d.ts`

**Step 1: Create src/vite-env.d.ts**

```typescript
/// <reference types="vite/client" />
```

**Step 2: Create src/lib/utils.ts**

```typescript
import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
```

**Step 3: Create src/index.css** (shadcn neutral palette)

```css
@import "tailwindcss";

@custom-variant dark (&:is(.dark *));

@theme {
  --color-background: #ffffff;
  --color-foreground: #0a0a0a;
  --color-card: #ffffff;
  --color-card-foreground: #0a0a0a;
  --color-popover: #ffffff;
  --color-popover-foreground: #0a0a0a;
  --color-primary: #171717;
  --color-primary-foreground: #fafafa;
  --color-secondary: #f5f5f5;
  --color-secondary-foreground: #171717;
  --color-muted: #f5f5f5;
  --color-muted-foreground: #737373;
  --color-accent: #f5f5f5;
  --color-accent-foreground: #171717;
  --color-destructive: #ef4444;
  --color-destructive-foreground: #fafafa;
  --color-border: #e5e5e5;
  --color-input: #e5e5e5;
  --color-ring: #171717;
  --radius-sm: 0.25rem;
  --radius-md: 0.375rem;
  --radius-lg: 0.5rem;
  --radius-xl: 0.75rem;
}

.dark {
  --color-background: #0a0a0a;
  --color-foreground: #fafafa;
  --color-card: #171717;
  --color-card-foreground: #fafafa;
  --color-popover: #171717;
  --color-popover-foreground: #fafafa;
  --color-primary: #fafafa;
  --color-primary-foreground: #171717;
  --color-secondary: #262626;
  --color-secondary-foreground: #fafafa;
  --color-muted: #262626;
  --color-muted-foreground: #a3a3a3;
  --color-accent: #262626;
  --color-accent-foreground: #fafafa;
  --color-destructive: #dc2626;
  --color-destructive-foreground: #fafafa;
  --color-border: #262626;
  --color-input: #262626;
  --color-ring: #d4d4d4;
}

html {
  font-family: "Inter", ui-sans-serif, system-ui, -apple-system, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

body {
  background-color: var(--color-background);
  color: var(--color-foreground);
}

::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--color-muted-foreground);
}
```

**Step 4: Create src/main.tsx**

```typescript
import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import App from "./App"
import "./index.css"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>
)
```

**Step 5: Create src/App.tsx** (minimal shell)

```typescript
export default function App() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="border-b border-border px-6 py-4">
        <h1 className="text-xl font-semibold">Translate</h1>
      </header>
      <main className="mx-auto max-w-5xl px-4 py-8">
        <p className="text-muted-foreground">Translation app coming soon...</p>
      </main>
    </div>
  )
}
```

**Step 6: Verify build**

Run: `cd tools/translate && npx vite build`
Expected: Build succeeds, `dist/` directory created

**Step 7: Commit**

```bash
git add tools/translate/src/
git commit -m "feat(translate): add React SPA skeleton with shadcn Tailwind theme"
```

---

### Task 7: UI Primitives (Button, Card, Textarea, Skeleton)

**Files:**
- Create: `tools/translate/src/components/ui/button.tsx`
- Create: `tools/translate/src/components/ui/card.tsx`
- Create: `tools/translate/src/components/ui/textarea.tsx`
- Create: `tools/translate/src/components/ui/skeleton.tsx`

**Step 1: Create button.tsx** (copy pattern from cc-viewer)

```typescript
import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground shadow-sm hover:bg-primary/90",
        destructive: "bg-destructive text-destructive-foreground shadow-sm hover:bg-destructive/90",
        outline: "border border-border bg-background shadow-sm hover:bg-secondary hover:text-secondary-foreground",
        secondary: "bg-secondary text-secondary-foreground shadow-sm hover:bg-secondary/80",
        ghost: "hover:bg-secondary hover:text-secondary-foreground",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 rounded-md px-3 text-xs",
        lg: "h-10 rounded-md px-8",
        icon: "h-9 w-9",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, ...props }, ref) => {
    return (
      <button
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button, buttonVariants }
```

**Step 2: Create card.tsx** (same as cc-viewer)

```typescript
import * as React from "react"
import { cn } from "@/lib/utils"

const Card = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("rounded-lg border border-border bg-card text-card-foreground shadow-sm", className)}
      {...props}
    />
  )
)
Card.displayName = "Card"

const CardHeader = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("flex flex-col space-y-1.5 p-6", className)} {...props} />
  )
)
CardHeader.displayName = "CardHeader"

const CardContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("p-6 pt-0", className)} {...props} />
  )
)
CardContent.displayName = "CardContent"

export { Card, CardHeader, CardContent }
```

**Step 3: Create textarea.tsx**

```typescript
import * as React from "react"
import { cn } from "@/lib/utils"

const Textarea = React.forwardRef<
  HTMLTextAreaElement,
  React.TextareaHTMLAttributes<HTMLTextAreaElement>
>(({ className, ...props }, ref) => {
  return (
    <textarea
      className={cn(
        "flex min-h-[120px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 resize-none",
        className
      )}
      ref={ref}
      {...props}
    />
  )
})
Textarea.displayName = "Textarea"

export { Textarea }
```

**Step 4: Create skeleton.tsx**

```typescript
import { cn } from "@/lib/utils"

function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("animate-pulse rounded-md bg-muted", className)} {...props} />
  )
}

export { Skeleton }
```

**Step 5: Verify build**

Run: `cd tools/translate && npx vite build`
Expected: Build succeeds

**Step 6: Commit**

```bash
git add tools/translate/src/components/ui/
git commit -m "feat(translate): add shadcn UI primitives (button, card, textarea, skeleton)"
```

---

### Task 8: API Client + Zustand Store

**Files:**
- Create: `tools/translate/src/api/client.ts`
- Create: `tools/translate/src/stores/translate.ts`

**Step 1: Create src/api/client.ts**

```typescript
import type { TranslateResult, Language, DetectResult } from '../../worker/types'

const API_BASE = '/api'

export async function translateText(
  text: string,
  from: string,
  to: string
): Promise<TranslateResult> {
  const res = await fetch(`${API_BASE}/translate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text, from, to }),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Translation failed' }))
    throw new Error(err.error || `HTTP ${res.status}`)
  }
  return res.json()
}

export async function fetchLanguages(): Promise<Language[]> {
  const res = await fetch(`${API_BASE}/languages`)
  if (!res.ok) throw new Error('Failed to fetch languages')
  const data = await res.json()
  return data.languages
}

export async function detectLanguage(text: string): Promise<DetectResult> {
  const res = await fetch(`${API_BASE}/detect`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  })
  if (!res.ok) throw new Error('Detection failed')
  return res.json()
}

export function ttsUrl(lang: string, text: string): string {
  return `${API_BASE}/tts?tl=${encodeURIComponent(lang)}&q=${encodeURIComponent(text.slice(0, 200))}`
}
```

**Step 2: Create src/stores/translate.ts**

```typescript
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { TranslateResult } from '../../worker/types'

export interface HistoryEntry {
  id: string
  sourceText: string
  translation: string
  sourceLang: string
  targetLang: string
  timestamp: number
}

interface TranslateState {
  sourceText: string
  sourceLang: string
  targetLang: string
  result: TranslateResult | null
  loading: boolean
  error: string | null
  history: HistoryEntry[]

  setSourceText: (text: string) => void
  setSourceLang: (lang: string) => void
  setTargetLang: (lang: string) => void
  swapLanguages: () => void
  setResult: (result: TranslateResult | null) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
  addToHistory: (entry: HistoryEntry) => void
  removeFromHistory: (id: string) => void
  clearHistory: () => void
  restoreFromHistory: (entry: HistoryEntry) => void
}

export const useTranslateStore = create<TranslateState>()(
  persist(
    (set, get) => ({
      sourceText: '',
      sourceLang: 'auto',
      targetLang: 'vi',
      result: null,
      loading: false,
      error: null,
      history: [],

      setSourceText: (text) => set({ sourceText: text }),
      setSourceLang: (lang) => set({ sourceLang: lang }),
      setTargetLang: (lang) => set({ targetLang: lang }),
      swapLanguages: () => {
        const { sourceLang, targetLang, sourceText, result } = get()
        if (sourceLang === 'auto') return
        set({
          sourceLang: targetLang,
          targetLang: sourceLang,
          sourceText: result?.translation || '',
          result: null,
        })
      },
      setResult: (result) => set({ result }),
      setLoading: (loading) => set({ loading }),
      setError: (error) => set({ error }),
      addToHistory: (entry) =>
        set((state) => ({
          history: [entry, ...state.history].slice(0, 50),
        })),
      removeFromHistory: (id) =>
        set((state) => ({
          history: state.history.filter((h) => h.id !== id),
        })),
      clearHistory: () => set({ history: [] }),
      restoreFromHistory: (entry) =>
        set({
          sourceText: entry.sourceText,
          sourceLang: entry.sourceLang,
          targetLang: entry.targetLang,
          result: null,
        }),
    }),
    {
      name: 'translate-storage',
      partialize: (state) => ({
        history: state.history,
        sourceLang: state.sourceLang,
        targetLang: state.targetLang,
      }),
    }
  )
)
```

**Step 3: Verify build**

Run: `cd tools/translate && npx vite build`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add tools/translate/src/api/ tools/translate/src/stores/
git commit -m "feat(translate): add API client and Zustand store with history persistence"
```

---

### Task 9: Language Select Component

**Files:**
- Create: `tools/translate/src/components/language-select.tsx`

**Step 1: Create the searchable language dropdown**

```typescript
import { useState, useRef, useEffect } from 'react'
import { ChevronDown, Search } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Language } from '../../worker/types'

interface LanguageSelectProps {
  languages: Language[]
  value: string
  onChange: (code: string) => void
  detectedLang?: string | null
  showAutoDetect?: boolean
}

export function LanguageSelect({
  languages,
  value,
  onChange,
  detectedLang,
  showAutoDetect = false,
}: LanguageSelectProps) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setSearch('')
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const filtered = languages.filter(
    (l) =>
      (showAutoDetect || l.code !== 'auto') &&
      (l.name.toLowerCase().includes(search.toLowerCase()) ||
        l.code.toLowerCase().includes(search.toLowerCase()))
  )

  const selected = languages.find((l) => l.code === value)
  const displayName =
    value === 'auto' && detectedLang
      ? `Auto-detect (${detectedLang})`
      : selected?.name || value

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className={cn(
          'flex h-9 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm',
          'hover:bg-secondary focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring'
        )}
      >
        <span className="truncate">{displayName}</span>
        <ChevronDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
      </button>

      {open && (
        <div className="absolute z-50 mt-1 w-full rounded-md border border-border bg-popover shadow-lg">
          <div className="flex items-center border-b border-border px-3">
            <Search className="mr-2 h-4 w-4 shrink-0 opacity-50" />
            <input
              type="text"
              placeholder="Search languages..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="flex h-9 w-full bg-transparent py-2 text-sm outline-none placeholder:text-muted-foreground"
              autoFocus
            />
          </div>
          <div className="max-h-64 overflow-y-auto p-1">
            {filtered.map((lang) => (
              <button
                key={lang.code}
                type="button"
                onClick={() => {
                  onChange(lang.code)
                  setOpen(false)
                  setSearch('')
                }}
                className={cn(
                  'flex w-full items-center rounded-sm px-2 py-1.5 text-sm',
                  lang.code === value
                    ? 'bg-secondary text-secondary-foreground'
                    : 'hover:bg-secondary/50'
                )}
              >
                {lang.name}
                <span className="ml-auto text-xs text-muted-foreground">{lang.code}</span>
              </button>
            ))}
            {filtered.length === 0 && (
              <p className="px-2 py-4 text-center text-sm text-muted-foreground">
                No languages found
              </p>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
```

**Step 2: Verify build**

Run: `cd tools/translate && npx vite build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add tools/translate/src/components/language-select.tsx
git commit -m "feat(translate): add searchable language select dropdown"
```

---

### Task 10: Audio Button + Copy Button Components

**Files:**
- Create: `tools/translate/src/components/audio-button.tsx`

**Step 1: Create audio-button.tsx**

```typescript
import { useState, useRef } from 'react'
import { Volume2, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ttsUrl } from '@/api/client'

interface AudioButtonProps {
  lang: string
  text: string
}

export function AudioButton({ lang, text }: AudioButtonProps) {
  const [playing, setPlaying] = useState(false)
  const audioRef = useRef<HTMLAudioElement | null>(null)

  const disabled = !text || text.length > 200

  async function handlePlay() {
    if (disabled) return
    if (audioRef.current) {
      audioRef.current.pause()
      audioRef.current = null
    }
    setPlaying(true)
    try {
      const audio = new Audio(ttsUrl(lang, text))
      audioRef.current = audio
      audio.onended = () => setPlaying(false)
      audio.onerror = () => setPlaying(false)
      await audio.play()
    } catch {
      setPlaying(false)
    }
  }

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={handlePlay}
      disabled={disabled}
      title={disabled ? 'Text too long for TTS (max 200 chars)' : 'Listen'}
    >
      {playing ? (
        <Loader2 className="h-4 w-4 animate-spin" />
      ) : (
        <Volume2 className="h-4 w-4" />
      )}
    </Button>
  )
}
```

**Step 2: Verify build**

Run: `cd tools/translate && npx vite build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add tools/translate/src/components/audio-button.tsx
git commit -m "feat(translate): add TTS audio playback button"
```

---

### Task 11: Rich Data Panel (Definitions, Synonyms, Examples)

**Files:**
- Create: `tools/translate/src/components/rich-panel.tsx`

**Step 1: Create rich-panel.tsx**

```typescript
import { useState } from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Definition, SynonymGroup } from '../../worker/types'

interface RichPanelProps {
  definitions: Definition[] | null
  synonyms: SynonymGroup[] | null
  examples: string[] | null
}

function CollapsibleSection({
  title,
  children,
  defaultOpen = true,
}: {
  title: string
  children: React.ReactNode
  defaultOpen?: boolean
}) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <div className="border-b border-border last:border-b-0">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between px-4 py-3 text-sm font-medium hover:bg-secondary/50"
      >
        {title}
        <ChevronDown
          className={cn('h-4 w-4 transition-transform', open && 'rotate-180')}
        />
      </button>
      {open && <div className="px-4 pb-4">{children}</div>}
    </div>
  )
}

export function RichPanel({ definitions, synonyms, examples }: RichPanelProps) {
  if (!definitions && !synonyms && !examples) return null

  return (
    <div className="mt-4 rounded-lg border border-border bg-card">
      {definitions && definitions.length > 0 && (
        <CollapsibleSection title="Definitions">
          <div className="space-y-3">
            {definitions.map((def, i) => (
              <div key={i}>
                <span className="text-xs font-medium italic text-muted-foreground">
                  {def.partOfSpeech}
                </span>
                <ul className="mt-1 space-y-1.5">
                  {def.entries.map((entry, j) => (
                    <li key={j} className="text-sm">
                      <span>{entry.definition}</span>
                      {entry.example && (
                        <p className="mt-0.5 text-xs italic text-muted-foreground">
                          &ldquo;{entry.example}&rdquo;
                        </p>
                      )}
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </CollapsibleSection>
      )}

      {synonyms && synonyms.length > 0 && (
        <CollapsibleSection title="Synonyms">
          <div className="space-y-2">
            {synonyms.map((group, i) => (
              <div key={i}>
                <span className="text-xs font-medium italic text-muted-foreground">
                  {group.partOfSpeech}
                </span>
                <div className="mt-1 flex flex-wrap gap-1.5">
                  {group.entries.flat().map((syn, j) => (
                    <span
                      key={j}
                      className="rounded-md bg-secondary px-2 py-0.5 text-xs text-secondary-foreground"
                    >
                      {syn}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </CollapsibleSection>
      )}

      {examples && examples.length > 0 && (
        <CollapsibleSection title="Examples">
          <ul className="space-y-1.5">
            {examples.map((ex, i) => (
              <li
                key={i}
                className="text-sm text-muted-foreground"
                dangerouslySetInnerHTML={{ __html: `&bull; ${ex}` }}
              />
            ))}
          </ul>
        </CollapsibleSection>
      )}
    </div>
  )
}
```

**Step 2: Verify build**

Run: `cd tools/translate && npx vite build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add tools/translate/src/components/rich-panel.tsx
git commit -m "feat(translate): add collapsible definitions/synonyms/examples panel"
```

---

### Task 12: History Panel

**Files:**
- Create: `tools/translate/src/components/history-panel.tsx`

**Step 1: Create history-panel.tsx**

```typescript
import { X, Trash2, Clock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useTranslateStore, type HistoryEntry } from '@/stores/translate'

export function HistoryPanel() {
  const { history, removeFromHistory, clearHistory, restoreFromHistory } =
    useTranslateStore()

  if (history.length === 0) return null

  return (
    <div className="mt-6">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
          <Clock className="h-4 w-4" />
          Recent Translations
        </div>
        <Button variant="ghost" size="sm" onClick={clearHistory}>
          <Trash2 className="h-3.5 w-3.5 mr-1" />
          Clear
        </Button>
      </div>
      <div className="space-y-1.5">
        {history.map((entry: HistoryEntry) => (
          <div
            key={entry.id}
            className="group flex items-center gap-3 rounded-md border border-border px-3 py-2 text-sm hover:bg-secondary/50 cursor-pointer"
            onClick={() => restoreFromHistory(entry)}
          >
            <div className="flex-1 min-w-0">
              <span className="truncate block">{entry.sourceText}</span>
              <span className="truncate block text-muted-foreground">
                → {entry.translation}
              </span>
            </div>
            <span className="text-xs text-muted-foreground shrink-0">
              {entry.sourceLang} → {entry.targetLang}
            </span>
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                removeFromHistory(entry.id)
              }}
              className="opacity-0 group-hover:opacity-100 transition-opacity"
            >
              <X className="h-3.5 w-3.5 text-muted-foreground hover:text-foreground" />
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
```

**Step 2: Verify build**

Run: `cd tools/translate && npx vite build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add tools/translate/src/components/history-panel.tsx
git commit -m "feat(translate): add translation history panel with localStorage persistence"
```

---

### Task 13: Main Translator Component

**Files:**
- Create: `tools/translate/src/components/translator.tsx`

This is the main two-column translation panel that ties everything together.

**Step 1: Create translator.tsx**

```typescript
import { useEffect, useRef, useCallback } from 'react'
import { ArrowLeftRight, Copy, Check, Loader2 } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import { LanguageSelect } from '@/components/language-select'
import { AudioButton } from '@/components/audio-button'
import { RichPanel } from '@/components/rich-panel'
import { useTranslateStore } from '@/stores/translate'
import { translateText, fetchLanguages } from '@/api/client'
import { useState } from 'react'

export function Translator() {
  const {
    sourceText, sourceLang, targetLang, result, loading, error,
    setSourceText, setSourceLang, setTargetLang, swapLanguages,
    setResult, setLoading, setError, addToHistory,
  } = useTranslateStore()

  const [copied, setCopied] = useState(false)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const { data: languages = [] } = useQuery({
    queryKey: ['languages'],
    queryFn: fetchLanguages,
  })

  const doTranslate = useCallback(async (text: string) => {
    if (!text.trim() || !targetLang) return
    setLoading(true)
    setError(null)
    try {
      const res = await translateText(text, sourceLang, targetLang)
      setResult(res)
      addToHistory({
        id: Date.now().toString(),
        sourceText: text.slice(0, 100),
        translation: res.translation.slice(0, 100),
        sourceLang: res.detectedLanguage || sourceLang,
        targetLang,
        timestamp: Date.now(),
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Translation failed')
    } finally {
      setLoading(false)
    }
  }, [sourceLang, targetLang, setResult, setLoading, setError, addToHistory])

  // Debounced auto-translate
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current)
    if (!sourceText.trim()) {
      setResult(null)
      return
    }
    debounceRef.current = setTimeout(() => doTranslate(sourceText), 500)
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [sourceText, sourceLang, targetLang, doTranslate, setResult])

  async function handleCopy() {
    if (!result?.translation) return
    await navigator.clipboard.writeText(result.translation)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const detectedLang = result?.detectedLanguage
    ? languages.find(l => l.code === result.detectedLanguage)?.name || result.detectedLanguage
    : null

  return (
    <div>
      <div className="grid grid-cols-1 md:grid-cols-[1fr_auto_1fr] gap-3 items-start">
        {/* Source panel */}
        <div className="space-y-2">
          <LanguageSelect
            languages={languages}
            value={sourceLang}
            onChange={setSourceLang}
            detectedLang={sourceLang === 'auto' ? detectedLang : null}
            showAutoDetect
          />
          <Textarea
            placeholder="Type or paste text here..."
            value={sourceText}
            onChange={(e) => {
              if (e.target.value.length <= 5000) setSourceText(e.target.value)
            }}
            className="min-h-[200px] text-base"
          />
          <div className="flex items-center justify-between">
            <AudioButton
              lang={result?.detectedLanguage || sourceLang}
              text={sourceText}
            />
            <span className="text-xs text-muted-foreground">
              {sourceText.length}/5000
            </span>
          </div>
        </div>

        {/* Swap button */}
        <div className="flex items-center justify-center pt-10">
          <Button
            variant="ghost"
            size="icon"
            onClick={swapLanguages}
            disabled={sourceLang === 'auto'}
            title="Swap languages"
          >
            <ArrowLeftRight className="h-4 w-4" />
          </Button>
        </div>

        {/* Target panel */}
        <div className="space-y-2">
          <LanguageSelect
            languages={languages}
            value={targetLang}
            onChange={setTargetLang}
          />
          {loading ? (
            <Skeleton className="min-h-[200px] w-full rounded-md" />
          ) : (
            <Textarea
              placeholder="Translation will appear here..."
              value={result?.translation || ''}
              readOnly
              className="min-h-[200px] text-base bg-secondary/30"
            />
          )}
          <div className="flex items-center justify-between">
            <AudioButton lang={targetLang} text={result?.translation || ''} />
            <Button
              variant="ghost"
              size="icon"
              onClick={handleCopy}
              disabled={!result?.translation}
              title="Copy translation"
            >
              {copied ? (
                <Check className="h-4 w-4 text-green-600" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>
      </div>

      {error && (
        <div className="mt-3 rounded-md bg-destructive/10 px-4 py-2 text-sm text-destructive">
          {error}
        </div>
      )}

      {/* Rich data panel */}
      <RichPanel
        definitions={result?.definitions || null}
        synonyms={result?.synonyms || null}
        examples={result?.examples || null}
      />
    </div>
  )
}
```

**Step 2: Verify build**

Run: `cd tools/translate && npx vite build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add tools/translate/src/components/translator.tsx
git commit -m "feat(translate): add main translator component with debounced auto-translate"
```

---

### Task 14: Wire Up App.tsx + Theme Toggle

**Files:**
- Modify: `tools/translate/src/App.tsx`

**Step 1: Update App.tsx with full layout**

```typescript
import { useState, useEffect } from 'react'
import { Languages, Moon, Sun } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Translator } from '@/components/translator'
import { HistoryPanel } from '@/components/history-panel'

function useTheme() {
  const [dark, setDark] = useState(() => {
    if (typeof window === 'undefined') return false
    return localStorage.getItem('theme') === 'dark' ||
      (!localStorage.getItem('theme') && window.matchMedia('(prefers-color-scheme: dark)').matches)
  })

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
    localStorage.setItem('theme', dark ? 'dark' : 'light')
  }, [dark])

  return { dark, toggle: () => setDark(!dark) }
}

export default function App() {
  const { dark, toggle } = useTheme()

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="sticky top-0 z-40 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <Languages className="h-5 w-5" />
            <h1 className="text-lg font-semibold">Translate</h1>
          </div>
          <Button variant="ghost" size="icon" onClick={toggle}>
            {dark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </Button>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-8">
        <Translator />
        <HistoryPanel />
      </main>
    </div>
  )
}
```

**Step 2: Verify full build**

Run: `cd tools/translate && npx vite build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add tools/translate/src/App.tsx
git commit -m "feat(translate): wire up App with translator, history, and theme toggle"
```

---

### Task 15: Create favicon + Final Polish

**Files:**
- Create: `tools/translate/public/favicon.svg`

**Step 1: Create favicon.svg**

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m5 8 6 6"/><path d="m4 14 6-6 2-3"/><path d="M2 5h12"/><path d="M7 2h1"/><path d="m22 22-5-10-5 10"/><path d="M14 18h6"/></svg>
```

**Step 2: Verify full build one more time**

Run: `cd tools/translate && npx vite build`
Expected: Build succeeds, `dist/` has index.html + assets

**Step 3: Commit**

```bash
git add tools/translate/public/
git commit -m "feat(translate): add favicon"
```

---

### Task 16: Deploy to Cloudflare Workers

**Step 1: Build and deploy**

Run: `cd tools/translate && npm run deploy`
Expected: `vite build` succeeds, then `wrangler deploy` publishes to `https://translate.go-mizu.workers.dev`

**Step 2: Verify live**

Test: `curl -X POST https://translate.go-mizu.workers.dev/api/translate -H 'Content-Type: application/json' -d '{"text":"hello","from":"auto","to":"vi"}'`
Expected: JSON response with translation

Test: Open `https://translate.go-mizu.workers.dev` in browser
Expected: Full React SPA loads with translator UI

**Step 3: Final commit**

```bash
git add -A tools/translate/
git commit -m "feat(translate): deploy to translate.go-mizu.workers.dev"
```

---

## Summary

| Task | Description | Est. Files |
|------|-------------|------------|
| 1 | Project scaffold (package.json, wrangler, tsconfig, vite, html) | 5 |
| 2 | Worker types + Hono entry point + route stubs | 6 |
| 3 | Google Translate provider (primary, rich data) | 2 |
| 4 | MyMemory + LibreTranslate providers + fallback chain | 3 |
| 5 | API routes (translate, languages, detect, TTS proxy) | 4 |
| 6 | React SPA skeleton + Tailwind CSS setup | 5 |
| 7 | UI primitives (button, card, textarea, skeleton) | 4 |
| 8 | API client + Zustand store | 2 |
| 9 | Language select (searchable dropdown) | 1 |
| 10 | Audio button (TTS playback) | 1 |
| 11 | Rich panel (definitions, synonyms, examples) | 1 |
| 12 | History panel | 1 |
| 13 | Main translator component | 1 |
| 14 | App.tsx + theme toggle | 1 |
| 15 | Favicon + final polish | 1 |
| 16 | Deploy to Cloudflare Workers | 0 |
