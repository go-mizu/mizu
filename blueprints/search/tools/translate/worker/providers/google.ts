import type { TranslateProvider } from './base'
import type { TranslateResult, DetectResult, Language, Definition, SynonymGroup, Pronunciation } from '../types'

const BASE_URL = 'https://translate.googleapis.com/translate_a/single'

// All dt params for rich data (single-word queries)
const RICH_DT_PARAMS = ['t', 'bd', 'at', 'ex', 'md', 'ss', 'rw', 'rm', 'ld']
// Minimal dt params for multi-word queries
const BASIC_DT_PARAMS = ['t', 'rm']

function isSingleWord(text: string): boolean {
  return text.trim().split(/\s+/).length === 1
}

function buildParams(text: string, from: string, to: string): URLSearchParams {
  const dtParams = isSingleWord(text) ? RICH_DT_PARAMS : BASIC_DT_PARAMS
  const params = new URLSearchParams()
  params.set('client', 'gtx')
  params.set('sl', from)
  params.set('tl', to)
  params.set('dj', '1')
  for (const dt of dtParams) {
    params.append('dt', dt)
  }
  return params
}

/* eslint-disable @typescript-eslint/no-explicit-any */
function parseTranslation(data: any): string {
  if (!data.sentences) return ''
  return data.sentences
    .filter((s: any) => s.trans != null)
    .map((s: any) => s.trans)
    .join('')
}

function parsePronunciation(data: any): Pronunciation | null {
  if (!data.sentences || data.sentences.length === 0) return null
  const last = data.sentences[data.sentences.length - 1]
  const sourcePhonetic = last.src_translit || null
  const targetTranslit = last.translit || null
  if (!sourcePhonetic && !targetTranslit) return null
  return { sourcePhonetic, targetTranslit }
}

function parseAlternatives(data: any): string[] | null {
  if (!data.alternative_translations) return null
  const alts: string[] = []
  for (const group of data.alternative_translations) {
    if (!group.alternative) continue
    for (const alt of group.alternative) {
      if (alt.word_postproc && alt.word_postproc !== parseTranslation(data)) {
        alts.push(alt.word_postproc)
      }
    }
  }
  return alts.length > 0 ? alts : null
}

function parseDefinitions(data: any): Definition[] | null {
  if (!data.definitions) return null
  const defs: Definition[] = []
  for (const group of data.definitions) {
    const entries = (group.entry || []).map((e: any) => ({
      definition: e.gloss || '',
      example: e.example || null,
    }))
    if (entries.length > 0) {
      defs.push({ partOfSpeech: group.pos || '', entries })
    }
  }
  return defs.length > 0 ? defs : null
}

function parseSynonyms(data: any): SynonymGroup[] | null {
  if (!data.synsets) return null
  const groups: SynonymGroup[] = []
  for (const synset of data.synsets) {
    const entries: string[][] = (synset.entry || []).map((e: any) =>
      (e.synonym || []) as string[]
    )
    if (entries.length > 0) {
      groups.push({ partOfSpeech: synset.pos || '', entries })
    }
  }
  return groups.length > 0 ? groups : null
}

function parseExamples(data: any): string[] | null {
  if (!data.examples?.example) return null
  const examples: string[] = data.examples.example
    .map((e: any) => (e.text || '') as string)
    .filter((t: string) => t.length > 0)
  return examples.length > 0 ? examples : null
}

function parseConfidence(data: any): number {
  if (typeof data.confidence === 'number') return data.confidence
  if (data.ld_result?.srclangs_confidences?.[0] != null) {
    return data.ld_result.srclangs_confidences[0]
  }
  return 1.0
}

function parseDetectedLanguage(data: any): string {
  return data.src || 'auto'
}
/* eslint-enable @typescript-eslint/no-explicit-any */

async function callGoogle(text: string, from: string, to: string): Promise<any> {
  const params = buildParams(text, from, to)

  if (text.length <= 2000) {
    params.set('q', text)
    const url = `${BASE_URL}?${params.toString()}`
    const resp = await fetch(url, {
      headers: { 'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' },
    })
    if (!resp.ok) throw new Error(`Google Translate HTTP ${resp.status}`)
    return resp.json()
  }

  // POST for longer text
  const url = `${BASE_URL}?${params.toString()}`
  const body = new URLSearchParams()
  body.set('q', text)
  const resp = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
      'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
    },
    body: body.toString(),
  })
  if (!resp.ok) throw new Error(`Google Translate HTTP ${resp.status}`)
  return resp.json()
}

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
  { code: 'si', name: 'Sinhala' },
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

class GoogleProvider implements TranslateProvider {
  name = 'google'

  async translate(text: string, from: string, to: string): Promise<TranslateResult> {
    const data = await callGoogle(text, from, to)

    return {
      translation: parseTranslation(data),
      detectedLanguage: parseDetectedLanguage(data),
      confidence: parseConfidence(data),
      pronunciation: parsePronunciation(data),
      alternatives: parseAlternatives(data),
      definitions: parseDefinitions(data),
      synonyms: parseSynonyms(data),
      examples: parseExamples(data),
      provider: this.name,
    }
  }

  async detect(text: string): Promise<DetectResult> {
    const data = await callGoogle(text, 'auto', 'en')
    return {
      language: parseDetectedLanguage(data),
      confidence: parseConfidence(data),
    }
  }

  languages(): Language[] {
    return GOOGLE_LANGUAGES
  }
}

export const googleProvider = new GoogleProvider()
