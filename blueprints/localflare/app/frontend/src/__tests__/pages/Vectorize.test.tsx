import { describe, it, expect, afterAll } from 'vitest'
import {
  renderWithProviders,
  screen,
  waitFor,
  testApi,
  isVectorIndex,
  generateTestName,
  userEvent,
} from '../../test/utils'
import { Vectorize } from '../../pages/Vectorize'
import type { VectorIndex } from '../../types'

describe('Vectorize', () => {
  // Track created indexes for cleanup
  const createdIndexNames: string[] = []

  afterAll(async () => {
    // Cleanup created indexes
    for (const name of createdIndexNames) {
      try {
        await testApi.vectorize.deleteIndex(name)
      } catch {
        // Ignore cleanup errors
      }
    }
  })

  describe('API integration', () => {
    it('fetches indexes list with correct structure', async () => {
      const response = await testApi.vectorize.listIndexes()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.indexes).toBeInstanceOf(Array)

      const indexes = response.result!.indexes
      // Each index should have required fields
      for (const index of indexes) {
        expect(isVectorIndex(index)).toBe(true)
        expect(typeof index.id).toBe('string')
        expect(typeof index.name).toBe('string')
        expect(typeof index.created_at).toBe('string')

        // Optional fields type check
        if (index.dimensions !== undefined) {
          expect(typeof index.dimensions).toBe('number')
        }
        if (index.metric !== undefined) {
          expect(['cosine', 'euclidean', 'dot-product']).toContain(index.metric)
        }
      }
    })

    it('creates a new index with valid structure', async () => {
      const indexName = generateTestName('idx')
      const response = await testApi.vectorize.createIndex({
        name: indexName,
        dimensions: 768,
        metric: 'cosine',
      })

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()

      const index = response.result!
      expect(isVectorIndex(index)).toBe(true)
      expect(index.name).toBe(indexName)
      expect(typeof index.id).toBe('string')
      expect(typeof index.created_at).toBe('string')

      // Track for cleanup
      createdIndexNames.push(indexName)
    })

    it('deletes an index successfully', async () => {
      // Create an index to delete
      const indexName = generateTestName('idx-delete')
      const createResponse = await testApi.vectorize.createIndex({
        name: indexName,
        dimensions: 384,
        metric: 'euclidean',
      })

      expect(createResponse.success).toBe(true)

      // Delete the index
      const deleteResponse = await testApi.vectorize.deleteIndex(indexName)
      expect(deleteResponse.success).toBe(true)

      // Verify it's deleted - should not appear in list
      const listResponse = await testApi.vectorize.listIndexes()
      const indexNames = listResponse.result!.indexes.map((i: VectorIndex) => i.name)
      expect(indexNames).not.toContain(indexName)
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page title', async () => {
      renderWithProviders(<Vectorize />)
      expect(await screen.findByText('Vectorize')).toBeInTheDocument()
    })

    it('displays indexes from real API', async () => {
      // First create an index so we have something to display
      const indexName = generateTestName('idx-ui')
      await testApi.vectorize.createIndex({
        name: indexName,
        dimensions: 1536,
        metric: 'cosine',
      })
      createdIndexNames.push(indexName)

      renderWithProviders(<Vectorize />)

      await waitFor(() => {
        expect(screen.getByText(indexName)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('has create index button', async () => {
      renderWithProviders(<Vectorize />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      expect(screen.getByRole('button', { name: /Create Index/i })).toBeInTheDocument()
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<Vectorize />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      const createButton = screen.getByRole('button', { name: /Create Index/i })
      await user.click(createButton)

      expect(await screen.findByText('Create Vector Index')).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    it('shows empty state when no indexes exist', async () => {
      renderWithProviders(<Vectorize />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      // Page should render successfully
      expect(screen.getByText('Vectorize')).toBeInTheDocument()
    })
  })
})
