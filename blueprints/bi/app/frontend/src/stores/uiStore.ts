import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface UIState {
  // Sidebar
  sidebarCollapsed: boolean
  sidebarWidth: number

  // Theme
  theme: 'light' | 'dark' | 'system'
  resolvedTheme: 'light' | 'dark'

  // Command palette
  commandPaletteOpen: boolean

  // Modals
  createQuestionModalOpen: boolean
  createDashboardModalOpen: boolean
  createCollectionModalOpen: boolean

  // Actions
  toggleSidebar: () => void
  setSidebarCollapsed: (collapsed: boolean) => void
  setSidebarWidth: (width: number) => void
  setTheme: (theme: 'light' | 'dark' | 'system') => void
  openCommandPalette: () => void
  closeCommandPalette: () => void
  toggleCommandPalette: () => void
  openCreateQuestionModal: () => void
  closeCreateQuestionModal: () => void
  openCreateDashboardModal: () => void
  closeCreateDashboardModal: () => void
  openCreateCollectionModal: () => void
  closeCreateCollectionModal: () => void
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      // Initial state
      sidebarCollapsed: false,
      sidebarWidth: 260,
      theme: 'system',
      resolvedTheme: 'light',
      commandPaletteOpen: false,
      createQuestionModalOpen: false,
      createDashboardModalOpen: false,
      createCollectionModalOpen: false,

      // Actions
      toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
      setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
      setSidebarWidth: (width) => set({ sidebarWidth: width }),
      setTheme: (theme) => {
        const resolvedTheme = theme === 'system'
          ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
          : theme
        set({ theme, resolvedTheme })
      },
      openCommandPalette: () => set({ commandPaletteOpen: true }),
      closeCommandPalette: () => set({ commandPaletteOpen: false }),
      toggleCommandPalette: () => set((state) => ({ commandPaletteOpen: !state.commandPaletteOpen })),
      openCreateQuestionModal: () => set({ createQuestionModalOpen: true }),
      closeCreateQuestionModal: () => set({ createQuestionModalOpen: false }),
      openCreateDashboardModal: () => set({ createDashboardModalOpen: true }),
      closeCreateDashboardModal: () => set({ createDashboardModalOpen: false }),
      openCreateCollectionModal: () => set({ createCollectionModalOpen: true }),
      closeCreateCollectionModal: () => set({ createCollectionModalOpen: false }),
    }),
    {
      name: 'bi-ui-storage',
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
        sidebarWidth: state.sidebarWidth,
        theme: state.theme,
      }),
    }
  )
)
