import type { TranslateProvider } from './base'
import type { TranslateResult, DetectResult, Language } from '../types'
import { GOOGLE_LANGUAGES } from './google'

const TRANSLATE_URL = 'https://libretranslate.com/translate'
const DETECT_URL = 'https://libretranslate.com/detect'

class LibreTranslateProvider implements TranslateProvider {
  name = 'libre'

  async translate(text: string, from: string, to: string): Promise<TranslateResult> {
    const resp = await fetch(TRANSLATE_URL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        q: text,
        source: from === 'auto' ? 'auto' : from,
        target: to,
      }),
    })
    if (!resp.ok) throw new Error(`LibreTranslate HTTP ${resp.status}`)

    const data: any = await resp.json()

    return {
      translation: data.translatedText || '',
      detectedLanguage: data.detectedLanguage?.language || from,
      confidence: data.detectedLanguage?.confidence ?? 1.0,
      pronunciation: null,
      alternatives: null,
      definitions: null,
      synonyms: null,
      examples: null,
      provider: this.name,
    }
  }

  async detect(text: string): Promise<DetectResult> {
    const resp = await fetch(DETECT_URL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ q: text }),
    })
    if (!resp.ok) throw new Error(`LibreTranslate detect HTTP ${resp.status}`)

    const data: any = await resp.json()
    // LibreTranslate returns an array of detections
    const top = Array.isArray(data) ? data[0] : data
    return {
      language: top?.language || 'en',
      confidence: top?.confidence ?? 0,
    }
  }

  languages(): Language[] {
    return GOOGLE_LANGUAGES
  }
}

export const libreProvider = new LibreTranslateProvider()
