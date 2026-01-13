import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderWithProviders, screen, waitFor, setupMockFetch, mockData, userEvent } from '../../test/utils'
import { Cron } from '../../pages/Cron'

describe('Cron', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with triggers', () => {
    beforeEach(() => {
      setupMockFetch({
        '/cron': { triggers: mockData.cronTriggers() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<Cron />)

      expect(await screen.findByText('Cron Triggers')).toBeInTheDocument()
    })

    it('renders trigger list', async () => {
      renderWithProviders(<Cron />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('worker-1')).toBeInTheDocument()
      expect(screen.getByText('worker-2')).toBeInTheDocument()
      expect(screen.getByText('worker-3')).toBeInTheDocument()
    })

    it('has create trigger button', async () => {
      renderWithProviders(<Cron />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByRole('button', { name: /Create Trigger/i })).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    beforeEach(() => {
      setupMockFetch({
        '/cron': { triggers: [] },
      })
    })

    it('shows empty state when no triggers', async () => {
      renderWithProviders(<Cron />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('No triggers yet')).toBeInTheDocument()
    })
  })

  describe('create trigger form', () => {
    beforeEach(() => {
      setupMockFetch({
        '/cron': { triggers: mockData.cronTriggers() },
      })
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<Cron />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      const createButton = screen.getByRole('button', { name: /Create Trigger/i })
      await user.click(createButton)

      expect(await screen.findByText('Create Cron Trigger')).toBeInTheDocument()
    })
  })

  describe('error handling', () => {
    it('handles API error gracefully', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('API Error'))

      renderWithProviders(<Cron />)

      await waitFor(() => {
        expect(screen.getByText('Cron Triggers')).toBeInTheDocument()
      })
    })
  })
})
