export interface Language {
  code: string
  name: string
}

export interface Definition {
  partOfSpeech: string
  entries: Array<{ definition: string; example: string | null }>
}

export interface SynonymGroup {
  partOfSpeech: string
  entries: string[][]
}

export interface TranslateResult {
  translation: string
  detectedLanguage: string
  confidence: number
  pronunciation: { sourcePhonetic: string | null; targetTranslit: string | null } | null
  alternatives: string[] | null
  definitions: Definition[] | null
  synonyms: SynonymGroup[] | null
  examples: string[] | null
  provider: string
}

const API_BASE = '/api'

export async function translateText(text: string, from: string, to: string): Promise<TranslateResult> {
  const res = await fetch(`${API_BASE}/translate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text, from, to }),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Translation failed' })) as { error?: string }
    throw new Error(err.error || `HTTP ${res.status}`)
  }
  return res.json()
}

export async function fetchLanguages(): Promise<Language[]> {
  const res = await fetch(`${API_BASE}/languages`)
  if (!res.ok) throw new Error('Failed to fetch languages')
  const data = await res.json() as { languages: Language[] }
  return data.languages
}

export function ttsUrl(lang: string, text: string): string {
  return `${API_BASE}/tts?tl=${encodeURIComponent(lang)}&q=${encodeURIComponent(text.slice(0, 200))}`
}
