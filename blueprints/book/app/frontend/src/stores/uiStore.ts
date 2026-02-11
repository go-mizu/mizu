import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface UIState {
  sidebarOpen: boolean
  shelfView: 'grid' | 'list' | 'table'
  sortBy: string
  theme: 'light' | 'dark'

  toggleSidebar: () => void
  setShelfView: (v: 'grid' | 'list' | 'table') => void
  setSortBy: (s: string) => void
  setTheme: (t: 'light' | 'dark') => void
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      sidebarOpen: true,
      shelfView: 'grid',
      sortBy: 'date_added',
      theme: 'light',

      toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
      setShelfView: (shelfView) => set({ shelfView }),
      setSortBy: (sortBy) => set({ sortBy }),
      setTheme: (theme) => set({ theme }),
    }),
    { name: 'book-ui' }
  )
)
