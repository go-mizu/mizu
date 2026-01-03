import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useDatabaseStore } from '../../src/stores/databaseStore'
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

describe('useDatabaseStore', () => {
  beforeEach(() => {
    useDatabaseStore.setState({
      databases: {},
      rows: {},
      views: {},
      activeViewId: null,
      activeFilters: [],
      activeSorts: [],
      loading: {},
    })
    vi.clearAllMocks()
  })

  describe('fetchDatabase', () => {
    it('should fetch and cache a database', async () => {
      const mockDatabase = {
        id: 'db-1',
        workspace_id: 'ws-1',
        name: 'Tasks',
        properties: [],
        created_at: '',
        updated_at: '',
      }

      vi.mocked(api.get).mockResolvedValueOnce(mockDatabase)

      const result = await useDatabaseStore.getState().fetchDatabase('db-1')

      expect(api.get).toHaveBeenCalledWith('/databases/db-1')
      expect(result).toEqual(mockDatabase)
      expect(useDatabaseStore.getState().databases['db-1']).toEqual(mockDatabase)
    })

    it('should return cached database without API call', async () => {
      const cachedDb = {
        id: 'cached-db',
        workspace_id: 'ws-1',
        name: 'Cached',
        properties: [],
        created_at: '',
        updated_at: '',
      }

      useDatabaseStore.setState({ databases: { 'cached-db': cachedDb } })

      const result = await useDatabaseStore.getState().fetchDatabase('cached-db')

      expect(api.get).not.toHaveBeenCalled()
      expect(result).toEqual(cachedDb)
    })

    it('should handle fetch errors', async () => {
      vi.mocked(api.get).mockRejectedValueOnce(new Error('Not found'))

      const result = await useDatabaseStore.getState().fetchDatabase('nonexistent')

      expect(result).toBeNull()
    })
  })

  describe('fetchRows', () => {
    it('should fetch rows for a database', async () => {
      const mockRows = [
        { id: 'row-1', database_id: 'db-1', properties: { title: 'Row 1' }, created_at: '', updated_at: '' },
        { id: 'row-2', database_id: 'db-1', properties: { title: 'Row 2' }, created_at: '', updated_at: '' },
      ]

      vi.mocked(api.get).mockResolvedValueOnce({ rows: mockRows })

      const result = await useDatabaseStore.getState().fetchRows('db-1')

      expect(api.get).toHaveBeenCalledWith('/databases/db-1/rows')
      expect(result).toEqual(mockRows)
      expect(useDatabaseStore.getState().rows['db-1']).toEqual(mockRows)
    })

    it('should include filters and sorts in request', async () => {
      vi.mocked(api.get).mockResolvedValueOnce({ rows: [] })

      const filters = [{ property: 'status', operator: 'equals', value: 'done' }]
      const sorts = [{ property: 'created_at', direction: 'desc' as const }]

      await useDatabaseStore.getState().fetchRows('db-1', filters, sorts)

      expect(api.get).toHaveBeenCalledWith(
        expect.stringContaining('/databases/db-1/rows?')
      )
    })

    it('should return empty array on error', async () => {
      vi.mocked(api.get).mockRejectedValueOnce(new Error('Network error'))

      const result = await useDatabaseStore.getState().fetchRows('db-1')

      expect(result).toEqual([])
    })
  })

  describe('row operations', () => {
    describe('createRow', () => {
      it('should create a new row', async () => {
        const newRow = {
          id: 'new-row',
          database_id: 'db-1',
          properties: { title: 'New row' },
          created_at: '',
          updated_at: '',
        }

        vi.mocked(api.post).mockResolvedValueOnce(newRow)

        const result = await useDatabaseStore.getState().createRow('db-1')

        expect(api.post).toHaveBeenCalledWith('/databases/db-1/rows', {
          properties: { title: 'New row' },
        })
        expect(result).toEqual(newRow)
        expect(useDatabaseStore.getState().rows['db-1']).toContainEqual(newRow)
      })

      it('should create row with custom properties', async () => {
        const newRow = {
          id: 'new-row',
          database_id: 'db-1',
          properties: { title: 'Custom', status: 'todo' },
          created_at: '',
          updated_at: '',
        }

        vi.mocked(api.post).mockResolvedValueOnce(newRow)

        await useDatabaseStore.getState().createRow('db-1', { status: 'todo' })

        expect(api.post).toHaveBeenCalledWith('/databases/db-1/rows', {
          properties: { title: 'New row', status: 'todo' },
        })
      })
    })

    describe('updateRow', () => {
      it('should optimistically update row properties', async () => {
        const existingRow = {
          id: 'row-1',
          database_id: 'db-1',
          properties: { title: 'Original', status: 'todo' },
          created_at: '',
          updated_at: '',
        }

        useDatabaseStore.setState({ rows: { 'db-1': [existingRow] } })
        vi.mocked(api.patch).mockResolvedValueOnce({})

        await useDatabaseStore.getState().updateRow('row-1', { status: 'done' })

        const updatedRows = useDatabaseStore.getState().rows['db-1']
        expect(updatedRows[0].properties.status).toBe('done')
        expect(api.patch).toHaveBeenCalledWith('/rows/row-1', { properties: { status: 'done' } })
      })

      it('should revert on error', async () => {
        const existingRow = {
          id: 'row-1',
          database_id: 'db-1',
          properties: { title: 'Original', status: 'todo' },
          created_at: '',
          updated_at: '',
        }

        useDatabaseStore.setState({ rows: { 'db-1': [existingRow] } })
        vi.mocked(api.patch).mockRejectedValueOnce(new Error('Failed'))

        await expect(
          useDatabaseStore.getState().updateRow('row-1', { status: 'done' })
        ).rejects.toThrow('Failed')

        expect(useDatabaseStore.getState().rows['db-1'][0].properties.status).toBe('todo')
      })
    })

    describe('deleteRow', () => {
      it('should optimistically delete row', async () => {
        const rows = [
          { id: 'row-1', database_id: 'db-1', properties: {}, created_at: '', updated_at: '' },
          { id: 'row-2', database_id: 'db-1', properties: {}, created_at: '', updated_at: '' },
        ]

        useDatabaseStore.setState({ rows: { 'db-1': rows } })
        vi.mocked(api.delete).mockResolvedValueOnce({})

        await useDatabaseStore.getState().deleteRow('row-1', 'db-1')

        expect(useDatabaseStore.getState().rows['db-1']).toHaveLength(1)
        expect(useDatabaseStore.getState().rows['db-1'][0].id).toBe('row-2')
      })
    })

    describe('duplicateRow', () => {
      it('should duplicate a row with copied properties', async () => {
        const originalRow = {
          id: 'row-1',
          database_id: 'db-1',
          properties: { title: 'Original', status: 'done' },
          created_at: '',
          updated_at: '',
        }

        const duplicatedRow = {
          id: 'row-2',
          database_id: 'db-1',
          properties: { title: 'Original (copy)', status: 'done' },
          created_at: '',
          updated_at: '',
        }

        useDatabaseStore.setState({ rows: { 'db-1': [originalRow] } })
        vi.mocked(api.post).mockResolvedValueOnce(duplicatedRow)

        const result = await useDatabaseStore.getState().duplicateRow('row-1', 'db-1')

        expect(result).toEqual(duplicatedRow)
        expect(useDatabaseStore.getState().rows['db-1']).toHaveLength(2)
      })

      it('should return null if original row not found', async () => {
        useDatabaseStore.setState({ rows: { 'db-1': [] } })

        const result = await useDatabaseStore.getState().duplicateRow('nonexistent', 'db-1')

        expect(result).toBeNull()
        expect(api.post).not.toHaveBeenCalled()
      })
    })
  })

  describe('property operations', () => {
    describe('addProperty', () => {
      it('should add a new property', async () => {
        const updatedDb = {
          id: 'db-1',
          workspace_id: 'ws-1',
          name: 'Tasks',
          properties: [{ id: 'prop-1', name: 'Status', type: 'select', options: [] }],
          created_at: '',
          updated_at: '',
        }

        vi.mocked(api.post).mockResolvedValueOnce(updatedDb)

        await useDatabaseStore.getState().addProperty('db-1', { name: 'Status', type: 'select' })

        expect(api.post).toHaveBeenCalledWith('/databases/db-1/properties', {
          name: 'Status',
          type: 'select',
        })
        expect(useDatabaseStore.getState().databases['db-1']).toEqual(updatedDb)
      })
    })

    describe('updateProperty', () => {
      it('should optimistically update property', async () => {
        const database = {
          id: 'db-1',
          workspace_id: 'ws-1',
          name: 'Tasks',
          properties: [{ id: 'prop-1', name: 'Old Name', type: 'text' as const }],
          created_at: '',
          updated_at: '',
        }

        useDatabaseStore.setState({ databases: { 'db-1': database } })
        vi.mocked(api.put).mockResolvedValueOnce({})

        await useDatabaseStore.getState().updateProperty('db-1', 'prop-1', { name: 'New Name' })

        const updatedDb = useDatabaseStore.getState().databases['db-1']
        expect(updatedDb.properties[0].name).toBe('New Name')
      })
    })

    describe('deleteProperty', () => {
      it('should optimistically delete property', async () => {
        const database = {
          id: 'db-1',
          workspace_id: 'ws-1',
          name: 'Tasks',
          properties: [
            { id: 'prop-1', name: 'Title', type: 'text' as const },
            { id: 'prop-2', name: 'Status', type: 'select' as const },
          ],
          created_at: '',
          updated_at: '',
        }

        useDatabaseStore.setState({ databases: { 'db-1': database } })
        vi.mocked(api.delete).mockResolvedValueOnce({})

        await useDatabaseStore.getState().deleteProperty('db-1', 'prop-2')

        expect(useDatabaseStore.getState().databases['db-1'].properties).toHaveLength(1)
        expect(useDatabaseStore.getState().databases['db-1'].properties[0].id).toBe('prop-1')
      })
    })
  })

  describe('view operations', () => {
    it('should set active view', () => {
      useDatabaseStore.getState().setActiveView('view-1')

      expect(useDatabaseStore.getState().activeViewId).toBe('view-1')
    })

    it('should set filters', () => {
      const filters = [{ property: 'status', operator: 'equals', value: 'done' }]

      useDatabaseStore.getState().setFilters(filters)

      expect(useDatabaseStore.getState().activeFilters).toEqual(filters)
    })

    it('should set sorts', () => {
      const sorts = [{ property: 'created_at', direction: 'desc' as const }]

      useDatabaseStore.getState().setSorts(sorts)

      expect(useDatabaseStore.getState().activeSorts).toEqual(sorts)
    })

    it('should fetch views', async () => {
      const mockViews = [
        { id: 'view-1', database_id: 'db-1', name: 'All Tasks', type: 'table' as const, config: {} },
        { id: 'view-2', database_id: 'db-1', name: 'Board', type: 'board' as const, config: {} },
      ]

      vi.mocked(api.get).mockResolvedValueOnce({ views: mockViews })

      const result = await useDatabaseStore.getState().fetchViews('db-1')

      expect(result).toEqual(mockViews)
      expect(useDatabaseStore.getState().views['db-1']).toEqual(mockViews)
    })

    it('should create a view', async () => {
      const newView = {
        id: 'view-new',
        database_id: 'db-1',
        name: 'New View',
        type: 'list' as const,
        config: {},
      }

      vi.mocked(api.post).mockResolvedValueOnce(newView)

      const result = await useDatabaseStore.getState().createView('db-1', {
        name: 'New View',
        type: 'list',
      })

      expect(result).toEqual(newView)
      expect(useDatabaseStore.getState().views['db-1']).toContainEqual(newView)
    })

    it('should delete a view', async () => {
      useDatabaseStore.setState({
        views: {
          'db-1': [
            { id: 'view-1', database_id: 'db-1', name: 'View 1', type: 'table' as const, config: {} },
            { id: 'view-2', database_id: 'db-1', name: 'View 2', type: 'board' as const, config: {} },
          ],
        },
      })
      vi.mocked(api.delete).mockResolvedValueOnce({})

      await useDatabaseStore.getState().deleteView('view-1')

      expect(useDatabaseStore.getState().views['db-1']).toHaveLength(1)
      expect(useDatabaseStore.getState().views['db-1'][0].id).toBe('view-2')
    })
  })

  describe('cache management', () => {
    it('should invalidate database cache', () => {
      useDatabaseStore.setState({
        databases: { 'db-1': { id: 'db-1', name: 'Tasks' } as never },
        rows: { 'db-1': [] },
        views: { 'db-1': [] },
      })

      useDatabaseStore.getState().invalidateDatabase('db-1')

      expect(useDatabaseStore.getState().databases['db-1']).toBeUndefined()
      expect(useDatabaseStore.getState().rows['db-1']).toBeUndefined()
      expect(useDatabaseStore.getState().views['db-1']).toBeUndefined()
    })
  })
})
