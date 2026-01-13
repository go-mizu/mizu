import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderWithProviders, screen, waitFor, setupMockFetch, mockData, userEvent } from '../../test/utils'
import { Vectorize } from '../../pages/Vectorize'

describe('Vectorize', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with indexes', () => {
    beforeEach(() => {
      setupMockFetch({
        '/vectorize/indexes': { indexes: mockData.vectorIndexes() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<Vectorize />)

      expect(await screen.findByText('Vectorize')).toBeInTheDocument()
    })

    it('renders index list', async () => {
      renderWithProviders(<Vectorize />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('index-1')).toBeInTheDocument()
      expect(screen.getByText('index-2')).toBeInTheDocument()
      expect(screen.getByText('index-3')).toBeInTheDocument()
    })

    it('has create index button', async () => {
      renderWithProviders(<Vectorize />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByRole('button', { name: /Create Index/i })).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    beforeEach(() => {
      setupMockFetch({
        '/vectorize/indexes': { indexes: [] },
      })
    })

    it('shows empty state when no indexes', async () => {
      renderWithProviders(<Vectorize />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('No indexes yet')).toBeInTheDocument()
    })
  })

  describe('create index form', () => {
    beforeEach(() => {
      setupMockFetch({
        '/vectorize/indexes': { indexes: mockData.vectorIndexes() },
      })
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<Vectorize />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      const createButton = screen.getByRole('button', { name: /Create Index/i })
      await user.click(createButton)

      expect(await screen.findByText('Create Vector Index')).toBeInTheDocument()
    })
  })

  describe('error handling', () => {
    it('handles API error gracefully', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('API Error'))

      renderWithProviders(<Vectorize />)

      await waitFor(() => {
        expect(screen.getByText('Vectorize')).toBeInTheDocument()
      })
    })
  })
})
