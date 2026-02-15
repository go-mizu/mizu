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
