import type { TranslateResult, DetectResult, Language } from '../types'

export interface TranslateProvider {
  name: string
  translate(text: string, from: string, to: string): Promise<TranslateResult>
  detect(text: string): Promise<DetectResult>
  languages(): Language[]
}
