import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { SearchResponse, SearchSettings, SearchLens, UserPreference } from '../types'

interface SearchState {
  // Search state
  query: string
  results: SearchResponse | null
  isLoading: boolean
  error: string | null

  // Settings
  settings: SearchSettings
  lenses: SearchLens[]
  preferences: UserPreference[]

  // Recent searches
  recentSearches: string[]

  // Actions
  setQuery: (query: string) => void
  setResults: (results: SearchResponse | null) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
  setSettings: (settings: SearchSettings) => void
  setLenses: (lenses: SearchLens[]) => void
  setPreferences: (preferences: UserPreference[]) => void
  addRecentSearch: (query: string) => void
  removeRecentSearch: (query: string) => void
  clearRecentSearches: () => void
  updateSettings: (updates: Partial<SearchSettings>) => void
}

const defaultSettings: SearchSettings = {
  safe_search: 'moderate',
  results_per_page: 10,
  region: 'us',
  language: 'en',
  theme: 'system',
  open_in_new_tab: false,
  show_thumbnails: true,
  show_instant_answers: true,
  show_knowledge_panel: true,
  save_history: true,
  autocomplete_enabled: true,
}

export const useSearchStore = create<SearchState>()(
  persist(
    (set) => ({
      // Initial state
      query: '',
      results: null,
      isLoading: false,
      error: null,
      settings: defaultSettings,
      lenses: [],
      preferences: [],
      recentSearches: [],

      // Actions
      setQuery: (query) => set({ query }),
      setResults: (results) => set({ results }),
      setLoading: (isLoading) => set({ isLoading }),
      setError: (error) => set({ error }),
      setSettings: (settings) => set({ settings }),
      setLenses: (lenses) => set({ lenses }),
      setPreferences: (preferences) => set({ preferences }),
      addRecentSearch: (query) =>
        set((state) => ({
          recentSearches: [
            query,
            ...state.recentSearches.filter((q) => q !== query),
          ].slice(0, 10),
        })),
      removeRecentSearch: (query) =>
        set((state) => ({
          recentSearches: state.recentSearches.filter((q) => q !== query),
        })),
      clearRecentSearches: () => set({ recentSearches: [] }),
      updateSettings: (updates) =>
        set((state) => ({
          settings: { ...state.settings, ...updates },
        })),
    }),
    {
      name: 'search-storage',
      partialize: (state) => ({
        settings: state.settings,
        recentSearches: state.recentSearches,
      }),
    }
  )
)
