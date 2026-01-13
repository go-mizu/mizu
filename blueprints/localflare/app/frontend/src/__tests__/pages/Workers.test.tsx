import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderWithProviders, screen, waitFor, setupMockFetch, mockData } from '../../test/utils'
import { Workers } from '../../pages/Workers'

describe('Workers Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with successful API response', () => {
    beforeEach(() => {
      setupMockFetch({
        '/workers': { workers: mockData.workers() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<Workers />)
      expect(await screen.findByText('Workers')).toBeInTheDocument()
    })

    it('displays workers from API', async () => {
      renderWithProviders(<Workers />)

      await waitFor(() => {
        expect(screen.getByText('api-router')).toBeInTheDocument()
        expect(screen.getByText('image-optimizer')).toBeInTheDocument()
      })
    })

    it('shows create button', async () => {
      renderWithProviders(<Workers />)

      await waitFor(() => {
        expect(screen.getByText(/Create Worker/i)).toBeInTheDocument()
      })
    })

    it('shows active status for workers', async () => {
      renderWithProviders(<Workers />)

      await waitFor(() => {
        // Both workers have active status in mock data
        expect(screen.getAllByText(/active/i).length).toBeGreaterThanOrEqual(1)
      })
    })
  })

  describe('error handling', () => {
    it('shows fallback data when API fails', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network error'))

      renderWithProviders(<Workers />)

      await waitFor(() => {
        // Should still show the page with fallback data
        expect(screen.getByText('Workers')).toBeInTheDocument()
      })
    })
  })

  describe('accessibility', () => {
    beforeEach(() => {
      setupMockFetch({
        '/workers': { workers: mockData.workers() },
      })
    })

    it('has proper heading hierarchy', async () => {
      renderWithProviders(<Workers />)

      await waitFor(() => {
        const heading = screen.getByRole('heading', { name: 'Workers' })
        expect(heading).toBeInTheDocument()
      })
    })
  })
})
