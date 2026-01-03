import { create } from 'zustand'
import { api, Page, Block } from '../api/client'

interface PageState {
  // Data
  pages: Record<string, Page>
  blocks: Record<string, Block[]>
  favorites: Set<string>

  // Loading states
  loading: Record<string, boolean>
  errors: Record<string, string | null>

  // Actions
  fetchPage: (pageId: string) => Promise<Page | null>
  fetchPages: (workspaceId: string) => Promise<Page[]>
  fetchBlocks: (pageId: string) => Promise<Block[]>
  updatePage: (pageId: string, data: Partial<Page>) => Promise<void>
  deletePage: (pageId: string) => Promise<void>
  createPage: (workspaceId: string, parentId?: string) => Promise<Page | null>
  duplicatePage: (pageId: string) => Promise<Page | null>
  movePage: (pageId: string, newParentId: string) => Promise<void>

  // Favorites
  toggleFavorite: (pageId: string) => Promise<void>
  isFavorite: (pageId: string) => boolean

  // Blocks
  saveBlocks: (pageId: string, blocks: Block[]) => Promise<void>

  // Cache management
  invalidatePage: (pageId: string) => void
  clearCache: () => void
}

export const usePageStore = create<PageState>((set, get) => ({
  pages: {},
  blocks: {},
  favorites: new Set(),
  loading: {},
  errors: {},

  fetchPage: async (pageId) => {
    const { pages, loading } = get()
    if (pages[pageId] && !loading[pageId]) {
      return pages[pageId]
    }

    set({ loading: { ...get().loading, [pageId]: true } })

    try {
      const page = await api.get<Page>(`/pages/${pageId}`)
      set({
        pages: { ...get().pages, [pageId]: page },
        loading: { ...get().loading, [pageId]: false },
        errors: { ...get().errors, [pageId]: null },
      })
      return page
    } catch (err) {
      set({
        loading: { ...get().loading, [pageId]: false },
        errors: { ...get().errors, [pageId]: (err as Error).message },
      })
      return null
    }
  },

  fetchPages: async (workspaceId) => {
    set({ loading: { ...get().loading, workspace: true } })

    try {
      const result = await api.get<{ pages: Page[] }>(`/workspaces/${workspaceId}/pages`)
      const pagesMap = { ...get().pages }
      result.pages.forEach(page => {
        pagesMap[page.id] = page
      })
      set({
        pages: pagesMap,
        loading: { ...get().loading, workspace: false },
      })
      return result.pages
    } catch (err) {
      set({ loading: { ...get().loading, workspace: false } })
      return []
    }
  },

  fetchBlocks: async (pageId) => {
    const { blocks } = get()
    if (blocks[pageId]) {
      return blocks[pageId]
    }

    try {
      const result = await api.get<{ blocks: Block[] }>(`/pages/${pageId}/blocks`)
      set({ blocks: { ...get().blocks, [pageId]: result.blocks } })
      return result.blocks
    } catch (err) {
      return []
    }
  },

  updatePage: async (pageId, data) => {
    const { pages } = get()
    const currentPage = pages[pageId]

    // Optimistic update
    if (currentPage) {
      set({ pages: { ...pages, [pageId]: { ...currentPage, ...data } } })
    }

    try {
      await api.patch(`/pages/${pageId}`, data)
    } catch (err) {
      // Revert on error
      if (currentPage) {
        set({ pages: { ...get().pages, [pageId]: currentPage } })
      }
      throw err
    }
  },

  deletePage: async (pageId) => {
    const { pages } = get()
    const currentPage = pages[pageId]

    // Optimistic delete
    const newPages = { ...pages }
    delete newPages[pageId]
    set({ pages: newPages })

    try {
      await api.delete(`/pages/${pageId}`)
    } catch (err) {
      // Revert on error
      if (currentPage) {
        set({ pages: { ...get().pages, [pageId]: currentPage } })
      }
      throw err
    }
  },

  createPage: async (workspaceId, parentId) => {
    try {
      const page = await api.post<Page>('/pages', {
        workspace_id: workspaceId,
        parent_id: parentId,
        parent_type: parentId ? 'page' : 'workspace',
        title: 'Untitled',
      })
      set({ pages: { ...get().pages, [page.id]: page } })
      return page
    } catch (err) {
      return null
    }
  },

  duplicatePage: async (pageId) => {
    try {
      const page = await api.post<Page>(`/pages/${pageId}/duplicate`)
      set({ pages: { ...get().pages, [page.id]: page } })
      return page
    } catch (err) {
      return null
    }
  },

  movePage: async (pageId, newParentId) => {
    const { pages } = get()
    const currentPage = pages[pageId]

    if (!currentPage) return

    // Optimistic update
    set({
      pages: {
        ...pages,
        [pageId]: { ...currentPage, parent_id: newParentId },
      },
    })

    try {
      await api.post(`/pages/${pageId}/move`, { parent_id: newParentId })
    } catch (err) {
      // Revert on error
      set({ pages: { ...get().pages, [pageId]: currentPage } })
      throw err
    }
  },

  toggleFavorite: async (pageId) => {
    const { favorites } = get()
    const isFav = favorites.has(pageId)

    // Optimistic update
    const newFavorites = new Set(favorites)
    if (isFav) {
      newFavorites.delete(pageId)
    } else {
      newFavorites.add(pageId)
    }
    set({ favorites: newFavorites })

    try {
      await api.post(`/favorites/${pageId}`, { action: isFav ? 'remove' : 'add' })
    } catch (err) {
      // Revert on error
      set({ favorites })
      throw err
    }
  },

  isFavorite: (pageId) => {
    return get().favorites.has(pageId)
  },

  saveBlocks: async (pageId, blocks) => {
    set({ blocks: { ...get().blocks, [pageId]: blocks } })

    try {
      await api.put(`/pages/${pageId}/blocks`, { blocks })
    } catch (err) {
      console.error('Failed to save blocks:', err)
      throw err
    }
  },

  invalidatePage: (pageId) => {
    const { pages, blocks } = get()
    const newPages = { ...pages }
    const newBlocks = { ...blocks }
    delete newPages[pageId]
    delete newBlocks[pageId]
    set({ pages: newPages, blocks: newBlocks })
  },

  clearCache: () => {
    set({ pages: {}, blocks: {}, loading: {}, errors: {} })
  },
}))
