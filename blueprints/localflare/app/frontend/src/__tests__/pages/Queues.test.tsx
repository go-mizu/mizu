import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderWithProviders, screen, waitFor, setupMockFetch, mockData, userEvent } from '../../test/utils'
import { Queues } from '../../pages/Queues'

describe('Queues', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with queues', () => {
    beforeEach(() => {
      setupMockFetch({
        '/queues': { queues: mockData.queues() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<Queues />)

      expect(await screen.findByText('Queues')).toBeInTheDocument()
    })

    it('renders queue list', async () => {
      renderWithProviders(<Queues />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('queue-1')).toBeInTheDocument()
      expect(screen.getByText('queue-2')).toBeInTheDocument()
      expect(screen.getByText('queue-3')).toBeInTheDocument()
    })

    it('has create queue button', async () => {
      renderWithProviders(<Queues />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByRole('button', { name: /Create Queue/i })).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    beforeEach(() => {
      setupMockFetch({
        '/queues': { queues: [] },
      })
    })

    it('shows empty state when no queues', async () => {
      renderWithProviders(<Queues />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('No queues yet')).toBeInTheDocument()
    })
  })

  describe('create queue form', () => {
    beforeEach(() => {
      setupMockFetch({
        '/queues': { queues: mockData.queues() },
      })
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<Queues />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      const createButton = screen.getByRole('button', { name: /Create Queue/i })
      await user.click(createButton)

      expect(await screen.findByRole('dialog')).toBeInTheDocument()
    })
  })

  describe('error handling', () => {
    it('handles API error gracefully', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('API Error'))

      renderWithProviders(<Queues />)

      await waitFor(() => {
        expect(screen.getByText('Queues')).toBeInTheDocument()
      })
    })
  })
})
