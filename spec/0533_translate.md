# 0533 — Translate App (Cloudflare Worker + React)

**Status**: Draft
**Date**: 2026-02-14

## Overview

Full-stack translation app deployed as a Cloudflare Worker at `https://translate.go-mizu.workers.dev`. Hono API backend with multi-provider translation (Google Translate primary, MyMemory + LibreTranslate fallbacks). React 19 frontend with shadcn design style.

Reference implementation: [cuberkam/free_translate_api](https://github.com/cuberkam/free_translate_api) — Django wrapper around `googletrans` Python library. We reimplement the same Google Translate scraping approach natively in TypeScript for CF Worker runtime.

## Architecture

```
tools/translate/
  worker/
    index.ts              — Hono app, CORS, SPA fallback via ASSETS binding
    types.ts              — Shared TypeScript interfaces
    routes/
      translate.ts        — POST /api/translate
      languages.ts        — GET /api/languages
      detect.ts           — POST /api/detect
      tts.ts              — GET /api/tts?tl=LANG&q=TEXT (proxy)
    providers/
      base.ts             — BaseProvider interface + fallback chain
      google.ts           — Google Translate (primary)
      mymemory.ts         — MyMemory API (fallback 1)
      libre.ts            — LibreTranslate (fallback 2)
  src/
    main.tsx              — React entry, QueryClient
    App.tsx               — Single-page layout (no router needed)
    index.css             — Tailwind v4 @theme, shadcn neutral palette
    lib/utils.ts          — cn() helper (clsx + tailwind-merge)
    api/client.ts         — Typed fetch wrapper for /api/* endpoints
    components/
      ui/                 — shadcn-styled primitives (button, select, textarea, skeleton, toast)
      translator.tsx      — Main two-column translation panel
      language-select.tsx — Searchable language dropdown with combobox
      rich-panel.tsx      — Definitions/synonyms/examples (collapsible)
      history-panel.tsx   — Translation history sidebar (localStorage)
      audio-button.tsx    — TTS playback button
    stores/
      translate.ts        — Zustand: source/target lang, text, result, loading, history
  package.json
  wrangler.toml
  tsconfig.json
  vite.config.ts
  index.html
```

Build pipeline follows cc-viewer pattern:
- `vite build` → `dist/` (React SPA assets)
- `wrangler deploy` → CF Worker with `[assets]` binding to `./dist`
- `wrangler.toml`: `not_found_handling = "single-page-application"`

## Backend

### API Endpoints

#### POST /api/translate

**Request:**
```json
{
  "text": "hello world",
  "from": "auto",
  "to": "vi"
}
```

**Response:**
```json
{
  "translation": "xin chào thế giới",
  "detectedLanguage": "en",
  "confidence": 0.98,
  "pronunciation": {
    "sourcePhonetic": "həˈlō wərld",
    "targetTranslit": "xin chào thế giới"
  },
  "alternatives": ["chào thế giới", "xin chào thế giới"],
  "definitions": [
    {
      "partOfSpeech": "noun",
      "entries": [
        {
          "definition": "used as a greeting",
          "example": "hello there, Katie!"
        }
      ]
    }
  ],
  "synonyms": [
    {
      "partOfSpeech": "noun",
      "entries": [["greeting", "welcome", "salutation"]]
    }
  ],
  "examples": [
    "She said <b>hello</b> to everyone in the room."
  ],
  "provider": "google"
}
```

Rich fields (`definitions`, `synonyms`, `examples`, `pronunciation`) are only populated for single-word queries from the Google provider. Multi-word text returns `null` for these fields.

When Google fails (429/503), falls back to MyMemory then LibreTranslate. Fallback responses have `provider: "mymemory"` or `provider: "libre"` and `null` for all rich fields.

#### GET /api/languages

**Response:**
```json
{
  "languages": [
    { "code": "af", "name": "Afrikaans" },
    { "code": "ar", "name": "Arabic" },
    { "code": "auto", "name": "Auto-detect" },
    ...
  ]
}
```

Returns 100+ languages supported by Google Translate, sorted alphabetically with `auto` at the top.

#### POST /api/detect

**Request:**
```json
{ "text": "bonjour le monde" }
```

**Response:**
```json
{
  "language": "fr",
  "confidence": 0.99
}
```

#### GET /api/tts?tl=LANG&q=TEXT

Proxies Google's TTS endpoint to avoid CORS issues:
```
https://translate.google.com/translate_tts?ie=UTF-8&client=tw-ob&tl=LANG&q=TEXT&total=1&idx=0&textlen=LEN
```

Returns audio/mpeg stream. Capped at 200 characters per request (Google TTS limit).

### Provider Implementation

#### BaseProvider Interface

```typescript
interface TranslateResult {
  translation: string
  detectedLanguage: string
  confidence: number
  pronunciation: {
    sourcePhonetic: string | null
    targetTranslit: string | null
  } | null
  alternatives: string[] | null
  definitions: Definition[] | null
  synonyms: SynonymGroup[] | null
  examples: string[] | null
  provider: string
}

interface BaseProvider {
  name: string
  translate(text: string, from: string, to: string): Promise<TranslateResult>
  supportedLanguages(): Language[]
}
```

#### Google Provider (Primary)

Endpoint: `https://translate.googleapis.com/translate_a/single`

**Request construction:**
- `client=gtx` (Chrome extension client — most permissive, no auth needed)
- `dj=1` (clean JSON output with named keys)
- `dt` params: `t` (translation), `bd` (dictionary), `at` (alternatives), `ex` (examples), `md` (definitions), `ss` (synonyms), `rw` (related), `rm` (transliteration), `ld` (language detection)
- GET for text ≤2000 chars, POST with `application/x-www-form-urlencoded` body for longer text

**Response parsing** (with `dj=1`):
- `sentences[].trans` → translated text (concatenated)
- `sentences[last].src_translit` → source phonetic/IPA
- `sentences[last].translit` → target transliteration
- `dict[]` → back-translations per part of speech
- `definitions[]` → monolingual definitions with examples
- `synsets[]` → synonyms grouped by part of speech and usage register
- `examples.example[]` → usage examples with `<b>` highlighting
- `alternative_translations[].alternative[]` → alternative translations
- `src` → detected source language
- `confidence` → detection confidence
- `ld_result` → detailed language detection data

Rich fields (`definitions`, `synsets`, `examples`) only populated for single-word queries.

**Error handling:**
- HTTP 429 (rate limit) or 503 → trigger fallback to MyMemory
- HTTP 400 → invalid language code, return error to client
- Network error → trigger fallback

#### MyMemory Provider (Fallback 1)

Endpoint: `https://api.mymemory.translated.net/get`

**Request:** `?q=TEXT&langpair=SL|TL`

**Response parsing:**
- `responseData.translatedText` → translation
- `responseData.match` → confidence score (0-1)
- `responseData.detectedLanguage` → detected source language (when `SL=auto`)

Limits: 5000 words/day without API key. Basic translation only (no definitions/synonyms).

#### LibreTranslate Provider (Fallback 2)

Endpoint: `https://libretranslate.com/translate`

**Request:** POST JSON `{ q, source, target }`

**Response parsing:**
- `translatedText` → translation
- `detectedLanguage.language` → detected source
- `detectedLanguage.confidence` → confidence

Basic translation only. Some public instances may be unreliable.

### Fallback Chain Logic

```typescript
async function translateWithFallback(text: string, from: string, to: string): Promise<TranslateResult> {
  const providers = [googleProvider, myMemoryProvider, libreProvider]
  for (const provider of providers) {
    try {
      return await provider.translate(text, from, to)
    } catch (err) {
      if (isLastProvider(provider)) throw err
      // Continue to next provider
    }
  }
}
```

## Frontend

### Design System — shadcn Style

Based on shadcn/ui's neutral theme tokens adapted for Tailwind CSS 4 `@theme`:

```css
@theme {
  --color-background: hsl(0 0% 100%);
  --color-foreground: hsl(240 10% 3.9%);
  --color-card: hsl(0 0% 100%);
  --color-card-foreground: hsl(240 10% 3.9%);
  --color-primary: hsl(240 5.9% 10%);
  --color-primary-foreground: hsl(0 0% 98%);
  --color-secondary: hsl(240 4.8% 95.9%);
  --color-secondary-foreground: hsl(240 5.9% 10%);
  --color-muted: hsl(240 4.8% 95.9%);
  --color-muted-foreground: hsl(240 3.8% 46.1%);
  --color-accent: hsl(240 4.8% 95.9%);
  --color-accent-foreground: hsl(240 5.9% 10%);
  --color-destructive: hsl(0 84.2% 60.2%);
  --color-border: hsl(240 5.9% 90%);
  --color-input: hsl(240 5.9% 90%);
  --color-ring: hsl(240 5.9% 10%);
  --radius-sm: 0.375rem;
  --radius-md: 0.5rem;
  --radius-lg: 0.75rem;
}
```

Dark mode via `.dark` class on `<html>`:
```css
.dark {
  --color-background: hsl(240 10% 3.9%);
  --color-foreground: hsl(0 0% 98%);
  --color-card: hsl(240 10% 3.9%);
  --color-border: hsl(240 3.7% 15.9%);
  /* ... */
}
```

Font: Inter (system fallback). Component primitives use CVA (class-variance-authority) + clsx + tailwind-merge via `cn()` utility. Icons from lucide-react.

### Page Layout

Single-page app, no routing needed. Vertically stacked:

```
┌──────────────────────────────────────────────────┐
│  🌐 Translate                        [🌙 Theme]  │  ← Header (sticky)
├──────────────────────────────────────────────────┤
│                                                    │
│  ┌─────────────────┐  ⇆  ┌─────────────────┐    │
│  │ English     ▾   │     │ Vietnamese   ▾   │    │
│  ├─────────────────┤     ├─────────────────┤    │
│  │                 │     │                 │    │
│  │  Type or paste  │     │  Translation    │    │
│  │  text here...   │     │  appears here   │    │
│  │                 │     │                 │    │
│  ├─────────────────┤     ├─────────────────┤    │
│  │ 🔊  0/5000      │     │ 🔊  📋          │    │
│  └─────────────────┘     └─────────────────┘    │
│                                                    │
│  ┌──────────────────────────────────────────┐    │  ← Rich panel (single words)
│  │ Definitions                           ▾  │    │
│  │ noun: a greeting or salutation           │    │
│  │ "she said hello to everyone"             │    │
│  ├──────────────────────────────────────────┤    │
│  │ Synonyms                              ▾  │    │
│  │ greeting, welcome, salutation            │    │
│  ├──────────────────────────────────────────┤    │
│  │ Examples                              ▾  │    │
│  │ • She said hello to everyone in the room │    │
│  └──────────────────────────────────────────┘    │
│                                                    │
│  ┌──────────────────────────────────────────┐    │  ← History
│  │ Recent Translations                      │    │
│  │ hello → xin chào                    [×]  │    │
│  │ goodbye → tạm biệt                 [×]  │    │
│  │ thank you → cảm ơn                 [×]  │    │
│  └──────────────────────────────────────────┘    │
│                                                    │
└──────────────────────────────────────────────────┘
```

### Component Breakdown

#### `<Translator />` — Main Panel
- Two-column grid on desktop (`grid-cols-2`), stacked on mobile
- Source panel: `<LanguageSelect />` + `<textarea>` + char count + `<AudioButton />`
- Target panel: `<LanguageSelect />` + readonly `<textarea>` + copy button + `<AudioButton />`
- Swap button centered between columns (rotates on click)
- Auto-translate: debounce 500ms after typing stops via `useDebouncedCallback`
- Loading state: skeleton pulse on target textarea

#### `<LanguageSelect />` — Searchable Dropdown
- Combobox pattern: text input + scrollable dropdown list
- Fuzzy search by language name or code
- "Auto-detect" option at top (source only)
- Recently used languages pinned at top (stored in localStorage)
- Shows detected language as badge when `from=auto`

#### `<RichPanel />` — Definitions/Synonyms/Examples
- Only rendered when translation result includes rich data (single-word queries)
- Three collapsible sections: Definitions, Synonyms, Examples
- Definitions: grouped by part of speech, each with definition text + optional example
- Synonyms: grouped by part of speech, comma-separated list
- Examples: bulleted list with `<b>` tags rendered as bold

#### `<HistoryPanel />` — Recent Translations
- Stored in localStorage (last 50 entries)
- Each entry: source text (truncated) → translated text (truncated), language pair
- Click to restore translation (fills source/target)
- Clear individual items or clear all
- Collapsible section below main translator

#### `<AudioButton />` — TTS Playback
- Calls `/api/tts?tl=LANG&q=TEXT` and plays returned audio
- Uses `Audio` API
- Loading spinner while fetching
- Disabled when text is empty or >200 chars

### State Management (Zustand)

```typescript
interface TranslateStore {
  // Input
  sourceText: string
  sourceLang: string       // ISO code or 'auto'
  targetLang: string       // ISO code
  setSourceText: (text: string) => void
  setSourceLang: (lang: string) => void
  setTargetLang: (lang: string) => void
  swapLanguages: () => void

  // Result
  result: TranslateResult | null
  loading: boolean
  error: string | null

  // History
  history: HistoryEntry[]
  addToHistory: (entry: HistoryEntry) => void
  removeFromHistory: (id: string) => void
  clearHistory: () => void
  restoreFromHistory: (entry: HistoryEntry) => void
}
```

History persisted to localStorage via Zustand `persist` middleware.

### UX Details

- **Auto-detect**: When `sourceLang = 'auto'`, the detected language appears as a small badge next to the language selector after translation completes
- **Debounced auto-translate**: 500ms debounce after user stops typing. No explicit "Translate" button needed (but could add one for accessibility)
- **Swap languages**: Animated swap button (ArrowLeftRight icon). Swaps source/target languages AND text. Disabled when source is 'auto'
- **Copy to clipboard**: Click copy icon on target panel → copies translation → shows toast "Copied to clipboard"
- **Character limit**: 5000 chars max in source textarea. Counter shows `123/5000`. Disable input after limit
- **Responsive**: On screens <768px, columns stack vertically (source on top, target below)
- **Keyboard shortcuts**: Ctrl+Enter to translate, Ctrl+Shift+S to swap
- **Empty state**: Target panel shows muted "Translation will appear here" placeholder
- **Error state**: Red toast with error message, target panel shows last successful translation (not cleared)

## Deployment

- **URL**: `https://translate.go-mizu.workers.dev`
- **wrangler.toml**: Standard CF Worker config with `[assets]` binding
- **No KV needed**: All state is client-side (localStorage). No server-side caching initially
- **No auth**: Public app, no API key required
- **Build**: `npm run deploy` → `vite build && wrangler deploy`

## Dependencies

```json
{
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

## Languages

Full list from Google Translate (100+ languages). Stored as static constant in `worker/providers/google.ts`. Key languages:

| Code | Name | Code | Name |
|------|------|------|------|
| auto | Auto-detect | ko | Korean |
| en | English | ja | Japanese |
| es | Spanish | zh-CN | Chinese (Simplified) |
| fr | French | zh-TW | Chinese (Traditional) |
| de | German | ar | Arabic |
| it | Italian | hi | Hindi |
| pt | Portuguese | vi | Vietnamese |
| ru | Russian | th | Thai |
| tr | Turkish | id | Indonesian |

## Non-Goals

- No user accounts or authentication
- No server-side translation caching (may add KV caching later)
- No document/file upload translation
- No batch/bulk translation API
- No translation quality scoring or comparison between providers
