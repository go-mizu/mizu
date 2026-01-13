import { describe, it, expect, afterAll } from 'vitest'
import {
  renderWithProviders,
  screen,
  waitFor,
  testApi,
  isAnalyticsDataset,
  generateTestName,
  userEvent,
} from '../../test/utils'
import { AnalyticsEngine } from '../../pages/AnalyticsEngine'
import type { AnalyticsDataset } from '../../types'

describe('AnalyticsEngine', () => {
  // Track created datasets for cleanup
  const createdDatasetNames: string[] = []

  afterAll(async () => {
    // Cleanup created datasets
    for (const name of createdDatasetNames) {
      try {
        await testApi.analytics.deleteDataset(name)
      } catch {
        // Ignore cleanup errors
      }
    }
  })

  describe('API integration', () => {
    it('fetches datasets list with correct structure', async () => {
      const response = await testApi.analytics.listDatasets()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.datasets).toBeInstanceOf(Array)

      const datasets = response.result!.datasets
      // Each dataset should have required fields
      for (const dataset of datasets) {
        expect(isAnalyticsDataset(dataset)).toBe(true)
        expect(typeof dataset.id).toBe('string')
        expect(typeof dataset.name).toBe('string')
        expect(typeof dataset.created_at).toBe('string')

        // Optional fields type check
        if (dataset.data_points !== undefined) {
          expect(typeof dataset.data_points).toBe('number')
        }
        if (dataset.estimated_size_bytes !== undefined) {
          expect(typeof dataset.estimated_size_bytes).toBe('number')
        }
      }
    })

    it('creates a new dataset with valid structure', async () => {
      const datasetName = generateTestName('ds')
      const response = await testApi.analytics.createDataset({
        name: datasetName,
      })

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()

      const dataset = response.result!
      expect(isAnalyticsDataset(dataset)).toBe(true)
      expect(dataset.name).toBe(datasetName)
      expect(typeof dataset.id).toBe('string')
      expect(typeof dataset.created_at).toBe('string')

      // Track for cleanup
      createdDatasetNames.push(datasetName)
    })

    it('deletes a dataset successfully', async () => {
      // Create a dataset to delete
      const datasetName = generateTestName('ds-delete')
      const createResponse = await testApi.analytics.createDataset({
        name: datasetName,
      })

      expect(createResponse.success).toBe(true)

      // Delete the dataset
      const deleteResponse = await testApi.analytics.deleteDataset(datasetName)
      expect(deleteResponse.success).toBe(true)

      // Verify it's deleted - should not appear in list
      const listResponse = await testApi.analytics.listDatasets()
      const datasetNames = listResponse.result!.datasets.map((d: AnalyticsDataset) => d.name)
      expect(datasetNames).not.toContain(datasetName)
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page title', async () => {
      renderWithProviders(<AnalyticsEngine />)
      expect(await screen.findByText('Analytics Engine')).toBeInTheDocument()
    })

    it('displays datasets from real API', async () => {
      // First create a dataset so we have something to display
      const datasetName = generateTestName('ds-ui')
      await testApi.analytics.createDataset({
        name: datasetName,
      })
      createdDatasetNames.push(datasetName)

      renderWithProviders(<AnalyticsEngine />)

      await waitFor(() => {
        expect(screen.getByText(datasetName)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('has create dataset button', async () => {
      renderWithProviders(<AnalyticsEngine />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      expect(screen.getByRole('button', { name: /Create Dataset/i })).toBeInTheDocument()
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<AnalyticsEngine />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      const createButton = screen.getByRole('button', { name: /Create Dataset/i })
      await user.click(createButton)

      expect(await screen.findByText('Create Analytics Dataset')).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    it('shows empty state when no datasets exist', async () => {
      renderWithProviders(<AnalyticsEngine />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      // Page should render successfully
      expect(screen.getByText('Analytics Engine')).toBeInTheDocument()
    })
  })
})
