import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderWithProviders, screen, waitFor, setupMockFetch, mockData, userEvent } from '../../test/utils'
import { AnalyticsEngine } from '../../pages/AnalyticsEngine'

describe('AnalyticsEngine', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with datasets', () => {
    beforeEach(() => {
      setupMockFetch({
        '/analytics/datasets': { datasets: mockData.analyticsDatasets() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<AnalyticsEngine />)

      expect(await screen.findByText('Analytics Engine')).toBeInTheDocument()
    })

    it('renders dataset list', async () => {
      renderWithProviders(<AnalyticsEngine />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('dataset-1')).toBeInTheDocument()
      expect(screen.getByText('dataset-2')).toBeInTheDocument()
      expect(screen.getByText('dataset-3')).toBeInTheDocument()
    })

    it('has create dataset button', async () => {
      renderWithProviders(<AnalyticsEngine />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByRole('button', { name: /Create Dataset/i })).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    beforeEach(() => {
      setupMockFetch({
        '/analytics/datasets': { datasets: [] },
      })
    })

    it('shows empty state when no datasets', async () => {
      renderWithProviders(<AnalyticsEngine />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('No datasets yet')).toBeInTheDocument()
    })
  })

  describe('create dataset form', () => {
    beforeEach(() => {
      setupMockFetch({
        '/analytics/datasets': { datasets: mockData.analyticsDatasets() },
      })
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<AnalyticsEngine />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      const createButton = screen.getByRole('button', { name: /Create Dataset/i })
      await user.click(createButton)

      expect(await screen.findByText('Create Analytics Dataset')).toBeInTheDocument()
    })
  })

  describe('error handling', () => {
    it('handles API error gracefully', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('API Error'))

      renderWithProviders(<AnalyticsEngine />)

      await waitFor(() => {
        expect(screen.getByText('Analytics Engine')).toBeInTheDocument()
      })
    })
  })
})
