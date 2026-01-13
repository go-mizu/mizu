import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderWithProviders, screen, waitFor, setupMockFetch, mockData, userEvent } from '../../test/utils'
import { Hyperdrive } from '../../pages/Hyperdrive'

describe('Hyperdrive', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with configs', () => {
    beforeEach(() => {
      setupMockFetch({
        '/hyperdrive': { configs: mockData.hyperdriveConfigs() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<Hyperdrive />)

      expect(await screen.findByText('Hyperdrive')).toBeInTheDocument()
    })

    it('renders config list', async () => {
      renderWithProviders(<Hyperdrive />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('hyperdrive-1')).toBeInTheDocument()
      expect(screen.getByText('hyperdrive-2')).toBeInTheDocument()
      expect(screen.getByText('hyperdrive-3')).toBeInTheDocument()
    })

    it('has create config button', async () => {
      renderWithProviders(<Hyperdrive />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByRole('button', { name: /Create Config/i })).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    beforeEach(() => {
      setupMockFetch({
        '/hyperdrive': { configs: [] },
      })
    })

    it('shows empty state when no configs', async () => {
      renderWithProviders(<Hyperdrive />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('No configs yet')).toBeInTheDocument()
    })
  })

  describe('create config form', () => {
    beforeEach(() => {
      setupMockFetch({
        '/hyperdrive': { configs: mockData.hyperdriveConfigs() },
      })
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<Hyperdrive />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      const createButton = screen.getByRole('button', { name: /Create Config/i })
      await user.click(createButton)

      expect(await screen.findByText('Create Hyperdrive Config')).toBeInTheDocument()
    })
  })

  describe('error handling', () => {
    it('handles API error gracefully', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('API Error'))

      renderWithProviders(<Hyperdrive />)

      await waitFor(() => {
        expect(screen.getByText('Hyperdrive')).toBeInTheDocument()
      })
    })
  })
})
