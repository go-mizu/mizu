import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface BookmarkItem {
  id: string
  type: 'question' | 'dashboard' | 'collection'
  name: string
  position: number
  addedAt: string
}

export interface RecentItem {
  id: string
  type: 'question' | 'dashboard' | 'collection' | 'table'
  name: string
  viewedAt: string
}

interface BookmarkStore {
  // Bookmarks
  bookmarks: BookmarkItem[]
  addBookmark: (item: Omit<BookmarkItem, 'position' | 'addedAt'>) => void
  removeBookmark: (id: string) => void
  reorderBookmarks: (fromIndex: number, toIndex: number) => void
  isBookmarked: (id: string) => boolean

  // Recent items
  recentItems: RecentItem[]
  addRecentItem: (item: Omit<RecentItem, 'viewedAt'>) => void
  clearRecent: () => void
  getRecentByType: (type: RecentItem['type']) => RecentItem[]

  // Pinned items (for home page)
  pinnedItems: { id: string; type: 'question' | 'dashboard' }[]
  pinItem: (id: string, type: 'question' | 'dashboard') => void
  unpinItem: (id: string) => void
  isPinned: (id: string) => boolean
}

const MAX_RECENT_ITEMS = 20

export const useBookmarkStore = create<BookmarkStore>()(
  persist(
    (set, get) => ({
      // Bookmarks
      bookmarks: [],

      addBookmark: (item) => {
        const existing = get().bookmarks.find(b => b.id === item.id)
        if (existing) return

        set((state) => ({
          bookmarks: [
            ...state.bookmarks,
            {
              ...item,
              position: state.bookmarks.length,
              addedAt: new Date().toISOString(),
            },
          ],
        }))
      },

      removeBookmark: (id) => {
        set((state) => ({
          bookmarks: state.bookmarks
            .filter(b => b.id !== id)
            .map((b, i) => ({ ...b, position: i })),
        }))
      },

      reorderBookmarks: (fromIndex, toIndex) => {
        set((state) => {
          const bookmarks = [...state.bookmarks]
          const [removed] = bookmarks.splice(fromIndex, 1)
          bookmarks.splice(toIndex, 0, removed)
          return {
            bookmarks: bookmarks.map((b, i) => ({ ...b, position: i })),
          }
        })
      },

      isBookmarked: (id) => {
        return get().bookmarks.some(b => b.id === id)
      },

      // Recent items
      recentItems: [],

      addRecentItem: (item) => {
        set((state) => {
          // Remove if already exists
          const filtered = state.recentItems.filter(r => r.id !== item.id)
          // Add to front
          const updated = [
            { ...item, viewedAt: new Date().toISOString() },
            ...filtered,
          ].slice(0, MAX_RECENT_ITEMS)
          return { recentItems: updated }
        })
      },

      clearRecent: () => {
        set({ recentItems: [] })
      },

      getRecentByType: (type) => {
        return get().recentItems.filter(r => r.type === type)
      },

      // Pinned items
      pinnedItems: [],

      pinItem: (id, type) => {
        const existing = get().pinnedItems.find(p => p.id === id)
        if (existing) return

        set((state) => ({
          pinnedItems: [...state.pinnedItems, { id, type }],
        }))
      },

      unpinItem: (id) => {
        set((state) => ({
          pinnedItems: state.pinnedItems.filter(p => p.id !== id),
        }))
      },

      isPinned: (id) => {
        return get().pinnedItems.some(p => p.id === id)
      },
    }),
    {
      name: 'bi-bookmarks',
      partialize: (state) => ({
        bookmarks: state.bookmarks,
        recentItems: state.recentItems,
        pinnedItems: state.pinnedItems,
      }),
    }
  )
)

// Helper hook for bookmark actions
export function useBookmarkActions(id: string, type: BookmarkItem['type'], name: string) {
  const { addBookmark, removeBookmark, isBookmarked } = useBookmarkStore()
  const bookmarked = isBookmarked(id)

  const toggle = () => {
    if (bookmarked) {
      removeBookmark(id)
    } else {
      addBookmark({ id, type, name })
    }
  }

  return { bookmarked, toggle }
}

// Helper hook for pin actions
export function usePinActions(id: string, type: 'question' | 'dashboard') {
  const { pinItem, unpinItem, isPinned } = useBookmarkStore()
  const pinned = isPinned(id)

  const toggle = () => {
    if (pinned) {
      unpinItem(id)
    } else {
      pinItem(id, type)
    }
  }

  return { pinned, toggle }
}
