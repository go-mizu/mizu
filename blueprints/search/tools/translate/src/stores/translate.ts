import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { TranslateResult } from '@/api/client'

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
        const { sourceLang, targetLang, result } = get()
        if (sourceLang === 'auto') return
        set({ sourceLang: targetLang, targetLang: sourceLang, sourceText: result?.translation || '', result: null })
      },
      setResult: (result) => set({ result }),
      setLoading: (loading) => set({ loading }),
      setError: (error) => set({ error }),
      addToHistory: (entry) => set((state) => ({ history: [entry, ...state.history].slice(0, 50) })),
      removeFromHistory: (id) => set((state) => ({ history: state.history.filter((h) => h.id !== id) })),
      clearHistory: () => set({ history: [] }),
      restoreFromHistory: (entry) => set({ sourceText: entry.sourceText, sourceLang: entry.sourceLang, targetLang: entry.targetLang, result: null }),
    }),
    {
      name: 'translate-storage',
      partialize: (state) => ({ history: state.history, sourceLang: state.sourceLang, targetLang: state.targetLang }),
    }
  )
)
