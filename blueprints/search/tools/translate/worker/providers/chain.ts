import type { TranslateResult, DetectResult, Language } from '../types'
import { googleProvider } from './google'
import { mymemoryProvider } from './mymemory'
import { libreProvider } from './libre'
import type { TranslateProvider } from './base'

const providers: TranslateProvider[] = [googleProvider, mymemoryProvider, libreProvider]

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
      console.error(`[${provider.name}] translate failed:`, lastError.message)
    }
  }
  throw lastError ?? new Error('All translation providers failed')
}

export async function detectWithFallback(text: string): Promise<DetectResult> {
  let lastError: Error | null = null
  for (const provider of providers) {
    try {
      return await provider.detect(text)
    } catch (err) {
      lastError = err instanceof Error ? err : new Error(String(err))
      console.error(`[${provider.name}] detect failed:`, lastError.message)
    }
  }
  throw lastError ?? new Error('All detection providers failed')
}

export function allLanguages(): Language[] {
  return googleProvider.languages()
}
