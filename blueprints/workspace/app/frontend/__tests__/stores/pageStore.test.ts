import { describe, it, expect, vi, beforeEach } from 'vitest'
import { usePageStore } from '../../src/stores/pageStore'
import { api } from '../../src/api/client'

vi.mock('../../src/api/client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}))

describe('usePageStore', () => {
  beforeEach(() => {
    // Reset the store state before each test
    usePageStore.setState({
      pages: {},
      blocks: {},
      favorites: new Set(),
      loading: {},
      errors: {},
    })
    vi.clearAllMocks()
  })

  describe('fetchPage', () => {
    it('should fetch and cache a page', async () => {
      const mockPage = {
        id: 'page-123',
        workspace_id: 'ws-1',
        parent_type: 'workspace',
        parent_id: 'ws-1',
        title: 'Test Page',
        created_at: '2024-01-01',
        updated_at: '2024-01-01',
      }

      vi.mocked(api.get).mockResolvedValueOnce(mockPage)

      const result = await usePageStore.getState().fetchPage('page-123')

      expect(api.get).toHaveBeenCalledWith('/pages/page-123')
      expect(result).toEqual(mockPage)
      expect(usePageStore.getState().pages['page-123']).toEqual(mockPage)
    })

    it('should return cached page without API call', async () => {
      const cachedPage = {
        id: 'cached-page',
        workspace_id: 'ws-1',
        parent_type: 'workspace',
        parent_id: 'ws-1',
        title: 'Cached Page',
        created_at: '2024-01-01',
        updated_at: '2024-01-01',
      }

      usePageStore.setState({ pages: { 'cached-page': cachedPage } })

      const result = await usePageStore.getState().fetchPage('cached-page')

      expect(api.get).not.toHaveBeenCalled()
      expect(result).toEqual(cachedPage)
    })

    it('should handle fetch errors', async () => {
      vi.mocked(api.get).mockRejectedValueOnce(new Error('Not found'))

      const result = await usePageStore.getState().fetchPage('nonexistent')

      expect(result).toBeNull()
      expect(usePageStore.getState().errors['nonexistent']).toBe('Not found')
    })
  })

  describe('fetchPages', () => {
    it('should fetch all pages for a workspace', async () => {
      const mockPages = [
        { id: 'page-1', title: 'Page 1', workspace_id: 'ws-1', parent_type: 'workspace', parent_id: 'ws-1', created_at: '', updated_at: '' },
        { id: 'page-2', title: 'Page 2', workspace_id: 'ws-1', parent_type: 'workspace', parent_id: 'ws-1', created_at: '', updated_at: '' },
      ]

      vi.mocked(api.get).mockResolvedValueOnce({ pages: mockPages })

      const result = await usePageStore.getState().fetchPages('ws-1')

      expect(api.get).toHaveBeenCalledWith('/workspaces/ws-1/pages')
      expect(result).toHaveLength(2)
      expect(usePageStore.getState().pages['page-1']).toEqual(mockPages[0])
      expect(usePageStore.getState().pages['page-2']).toEqual(mockPages[1])
    })

    it('should return empty array on error', async () => {
      vi.mocked(api.get).mockRejectedValueOnce(new Error('Network error'))

      const result = await usePageStore.getState().fetchPages('ws-1')

      expect(result).toEqual([])
    })
  })

  describe('updatePage', () => {
    it('should optimistically update page', async () => {
      const originalPage = {
        id: 'page-1',
        workspace_id: 'ws-1',
        parent_type: 'workspace',
        parent_id: 'ws-1',
        title: 'Original Title',
        created_at: '',
        updated_at: '',
      }

      usePageStore.setState({ pages: { 'page-1': originalPage } })
      vi.mocked(api.patch).mockResolvedValueOnce({})

      await usePageStore.getState().updatePage('page-1', { title: 'New Title' })

      expect(usePageStore.getState().pages['page-1'].title).toBe('New Title')
      expect(api.patch).toHaveBeenCalledWith('/pages/page-1', { title: 'New Title' })
    })

    it('should revert on API error', async () => {
      const originalPage = {
        id: 'page-1',
        workspace_id: 'ws-1',
        parent_type: 'workspace',
        parent_id: 'ws-1',
        title: 'Original Title',
        created_at: '',
        updated_at: '',
      }

      usePageStore.setState({ pages: { 'page-1': originalPage } })
      vi.mocked(api.patch).mockRejectedValueOnce(new Error('Server error'))

      await expect(
        usePageStore.getState().updatePage('page-1', { title: 'New Title' })
      ).rejects.toThrow('Server error')

      expect(usePageStore.getState().pages['page-1'].title).toBe('Original Title')
    })
  })

  describe('deletePage', () => {
    it('should optimistically delete page', async () => {
      const page = {
        id: 'page-to-delete',
        workspace_id: 'ws-1',
        parent_type: 'workspace',
        parent_id: 'ws-1',
        title: 'Delete Me',
        created_at: '',
        updated_at: '',
      }

      usePageStore.setState({ pages: { 'page-to-delete': page } })
      vi.mocked(api.delete).mockResolvedValueOnce({})

      await usePageStore.getState().deletePage('page-to-delete')

      expect(usePageStore.getState().pages['page-to-delete']).toBeUndefined()
      expect(api.delete).toHaveBeenCalledWith('/pages/page-to-delete')
    })

    it('should revert on API error', async () => {
      const page = {
        id: 'page-1',
        workspace_id: 'ws-1',
        parent_type: 'workspace',
        parent_id: 'ws-1',
        title: 'Keep Me',
        created_at: '',
        updated_at: '',
      }

      usePageStore.setState({ pages: { 'page-1': page } })
      vi.mocked(api.delete).mockRejectedValueOnce(new Error('Cannot delete'))

      await expect(usePageStore.getState().deletePage('page-1')).rejects.toThrow('Cannot delete')

      expect(usePageStore.getState().pages['page-1']).toEqual(page)
    })
  })

  describe('createPage', () => {
    it('should create a new page in workspace', async () => {
      const newPage = {
        id: 'new-page',
        workspace_id: 'ws-1',
        parent_type: 'workspace',
        parent_id: 'ws-1',
        title: 'Untitled',
        created_at: '',
        updated_at: '',
      }

      vi.mocked(api.post).mockResolvedValueOnce(newPage)

      const result = await usePageStore.getState().createPage('ws-1')

      expect(api.post).toHaveBeenCalledWith('/pages', {
        workspace_id: 'ws-1',
        parent_id: undefined,
        parent_type: 'workspace',
        title: 'Untitled',
      })
      expect(result).toEqual(newPage)
      expect(usePageStore.getState().pages['new-page']).toEqual(newPage)
    })

    it('should create a page with parent', async () => {
      const newPage = {
        id: 'child-page',
        workspace_id: 'ws-1',
        parent_type: 'page',
        parent_id: 'parent-page',
        title: 'Untitled',
        created_at: '',
        updated_at: '',
      }

      vi.mocked(api.post).mockResolvedValueOnce(newPage)

      await usePageStore.getState().createPage('ws-1', 'parent-page')

      expect(api.post).toHaveBeenCalledWith('/pages', {
        workspace_id: 'ws-1',
        parent_id: 'parent-page',
        parent_type: 'page',
        title: 'Untitled',
      })
    })

    it('should return null on error', async () => {
      vi.mocked(api.post).mockRejectedValueOnce(new Error('Failed'))

      const result = await usePageStore.getState().createPage('ws-1')

      expect(result).toBeNull()
    })
  })

  describe('favorites', () => {
    it('should toggle favorite on', async () => {
      vi.mocked(api.post).mockResolvedValueOnce({})

      await usePageStore.getState().toggleFavorite('page-1')

      expect(usePageStore.getState().favorites.has('page-1')).toBe(true)
      expect(api.post).toHaveBeenCalledWith('/favorites/page-1', { action: 'add' })
    })

    it('should toggle favorite off', async () => {
      usePageStore.setState({ favorites: new Set(['page-1']) })
      vi.mocked(api.post).mockResolvedValueOnce({})

      await usePageStore.getState().toggleFavorite('page-1')

      expect(usePageStore.getState().favorites.has('page-1')).toBe(false)
      expect(api.post).toHaveBeenCalledWith('/favorites/page-1', { action: 'remove' })
    })

    it('should check if page is favorite', () => {
      usePageStore.setState({ favorites: new Set(['fav-page']) })

      expect(usePageStore.getState().isFavorite('fav-page')).toBe(true)
      expect(usePageStore.getState().isFavorite('not-fav')).toBe(false)
    })

    it('should revert on toggle error', async () => {
      usePageStore.setState({ favorites: new Set() })
      vi.mocked(api.post).mockRejectedValueOnce(new Error('Failed'))

      await expect(usePageStore.getState().toggleFavorite('page-1')).rejects.toThrow('Failed')

      expect(usePageStore.getState().favorites.has('page-1')).toBe(false)
    })
  })

  describe('blocks', () => {
    it('should fetch blocks for a page', async () => {
      const mockBlocks = [
        { id: 'block-1', page_id: 'page-1', type: 'paragraph', content: {}, position: 0, created_at: '', updated_at: '' },
        { id: 'block-2', page_id: 'page-1', type: 'heading', content: {}, position: 1, created_at: '', updated_at: '' },
      ]

      vi.mocked(api.get).mockResolvedValueOnce({ blocks: mockBlocks })

      const result = await usePageStore.getState().fetchBlocks('page-1')

      expect(api.get).toHaveBeenCalledWith('/pages/page-1/blocks')
      expect(result).toEqual(mockBlocks)
      expect(usePageStore.getState().blocks['page-1']).toEqual(mockBlocks)
    })

    it('should return cached blocks', async () => {
      const cachedBlocks = [{ id: 'cached-block', page_id: 'page-1', type: 'paragraph', content: {}, position: 0, created_at: '', updated_at: '' }]
      usePageStore.setState({ blocks: { 'page-1': cachedBlocks } })

      const result = await usePageStore.getState().fetchBlocks('page-1')

      expect(api.get).not.toHaveBeenCalled()
      expect(result).toEqual(cachedBlocks)
    })

    it('should save blocks', async () => {
      const blocks = [{ id: 'new-block', page_id: 'page-1', type: 'paragraph', content: {}, position: 0, created_at: '', updated_at: '' }]
      vi.mocked(api.put).mockResolvedValueOnce({})

      await usePageStore.getState().saveBlocks('page-1', blocks)

      expect(usePageStore.getState().blocks['page-1']).toEqual(blocks)
      expect(api.put).toHaveBeenCalledWith('/pages/page-1/blocks', { blocks })
    })
  })

  describe('cache management', () => {
    it('should invalidate a page cache', () => {
      const page = { id: 'page-1', workspace_id: 'ws-1', parent_type: 'workspace', parent_id: 'ws-1', title: 'Test', created_at: '', updated_at: '' }
      const blocks = [{ id: 'block-1', page_id: 'page-1', type: 'paragraph', content: {}, position: 0, created_at: '', updated_at: '' }]

      usePageStore.setState({
        pages: { 'page-1': page },
        blocks: { 'page-1': blocks },
      })

      usePageStore.getState().invalidatePage('page-1')

      expect(usePageStore.getState().pages['page-1']).toBeUndefined()
      expect(usePageStore.getState().blocks['page-1']).toBeUndefined()
    })

    it('should clear all cache', () => {
      usePageStore.setState({
        pages: { 'page-1': { id: 'page-1', title: 'Test' } as never },
        blocks: { 'page-1': [] },
        loading: { 'page-1': true },
        errors: { 'page-1': 'Some error' },
      })

      usePageStore.getState().clearCache()

      expect(usePageStore.getState().pages).toEqual({})
      expect(usePageStore.getState().blocks).toEqual({})
      expect(usePageStore.getState().loading).toEqual({})
      expect(usePageStore.getState().errors).toEqual({})
    })
  })
})
