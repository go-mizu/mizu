import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { AIMode, AISession, AIResponse, Canvas, ModelInfo } from '../types/ai'

interface AIState {
  // Current mode
  mode: AIMode

  // Model selection
  selectedModelId: string | null
  models: ModelInfo[]

  // Current session
  currentSessionId: string | null
  sessions: AISession[]

  // Current query state
  isLoading: boolean
  isStreaming: boolean
  streamingContent: string
  streamingThinking: string[]
  currentResponse: AIResponse | null
  error: string | null

  // AI availability
  aiAvailable: boolean
  availableModes: AIMode[]

  // Canvas
  currentCanvas: Canvas | null

  // Actions
  setMode: (mode: AIMode) => void
  setSelectedModelId: (id: string | null) => void
  setModels: (models: ModelInfo[]) => void
  setCurrentSessionId: (id: string | null) => void
  setSessions: (sessions: AISession[]) => void
  addSession: (session: AISession) => void
  removeSession: (id: string) => void
  setLoading: (loading: boolean) => void
  setStreaming: (streaming: boolean) => void
  appendStreamContent: (content: string) => void
  addThinkingStep: (step: string) => void
  resetStream: () => void
  setCurrentResponse: (response: AIResponse | null) => void
  setError: (error: string | null) => void
  setAIAvailable: (available: boolean) => void
  setAvailableModes: (modes: AIMode[]) => void
  setCurrentCanvas: (canvas: Canvas | null) => void
}

export const useAIStore = create<AIState>()(
  persist(
    (set) => ({
      // Initial state
      mode: 'quick',
      selectedModelId: null,
      models: [],
      currentSessionId: null,
      sessions: [],
      isLoading: false,
      isStreaming: false,
      streamingContent: '',
      streamingThinking: [],
      currentResponse: null,
      error: null,
      aiAvailable: false,
      availableModes: [],
      currentCanvas: null,

      // Actions
      setMode: (mode) => set({ mode }),
      setSelectedModelId: (selectedModelId) => set({ selectedModelId }),
      setModels: (models) => set({ models }),
      setCurrentSessionId: (currentSessionId) => set({ currentSessionId }),
      setSessions: (sessions) => set({ sessions }),
      addSession: (session) => set((state) => ({
        sessions: [session, ...state.sessions],
      })),
      removeSession: (id) => set((state) => ({
        sessions: state.sessions.filter((s) => s.id !== id),
        currentSessionId: state.currentSessionId === id ? null : state.currentSessionId,
      })),
      setLoading: (isLoading) => set({ isLoading }),
      setStreaming: (isStreaming) => set({ isStreaming }),
      appendStreamContent: (content) => set((state) => ({
        streamingContent: state.streamingContent + content,
      })),
      addThinkingStep: (step) => set((state) => ({
        streamingThinking: [...state.streamingThinking, step],
      })),
      resetStream: () => set({
        streamingContent: '',
        streamingThinking: [],
        isStreaming: false,
      }),
      setCurrentResponse: (currentResponse) => set({ currentResponse }),
      setError: (error) => set({ error }),
      setAIAvailable: (aiAvailable) => set({ aiAvailable }),
      setAvailableModes: (availableModes) => set({ availableModes }),
      setCurrentCanvas: (currentCanvas) => set({ currentCanvas }),
    }),
    {
      name: 'ai-storage',
      partialize: (state) => ({
        mode: state.mode,
        selectedModelId: state.selectedModelId,
      }),
    }
  )
)
