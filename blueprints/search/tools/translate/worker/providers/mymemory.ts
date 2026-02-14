import type { TranslateProvider } from './base'
import type { TranslateResult, DetectResult, Language } from '../types'
import { GOOGLE_LANGUAGES } from './google'

const BASE_URL = 'https://api.mymemory.translated.net/get'
const MAX_CHARS = 500

class MyMemoryProvider implements TranslateProvider {
  name = 'mymemory'

  async translate(text: string, from: string, to: string): Promise<TranslateResult> {
    const truncated = text.slice(0, MAX_CHARS)
    // MyMemory doesn't support auto-detect, default to 'en'
    const sourceLang = from === 'auto' ? 'en' : from
    const langpair = `${sourceLang}|${to}`

    const params = new URLSearchParams({ q: truncated, langpair })
    const resp = await fetch(`${BASE_URL}?${params.toString()}`)
    if (!resp.ok) throw new Error(`MyMemory HTTP ${resp.status}`)

    const data: any = await resp.json()
    const translation = data.responseData?.translatedText || ''
    const detectedLanguage = data.responseData?.detectedLanguage || sourceLang
    const match = data.responseData?.match ?? 1.0

    return {
      translation,
      detectedLanguage: typeof detectedLanguage === 'string' ? detectedLanguage : sourceLang,
      confidence: typeof match === 'number' ? match : 1.0,
      pronunciation: null,
      alternatives: null,
      definitions: null,
      synonyms: null,
      examples: null,
      provider: this.name,
    }
  }

  async detect(text: string): Promise<DetectResult> {
    // MyMemory doesn't have a dedicated detect endpoint.
    // Translate with auto→en and check the detected language.
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
