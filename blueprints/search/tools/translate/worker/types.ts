export interface Env {
  ASSETS?: { fetch: typeof fetch }
  TRANSLATE_CACHE: KVNamespace
  ENVIRONMENT: string
}

export type HonoEnv = {
  Bindings: Env
}

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
