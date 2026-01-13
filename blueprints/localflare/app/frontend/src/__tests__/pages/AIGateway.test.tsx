import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderWithProviders, screen, waitFor, setupMockFetch, mockData, userEvent } from '../../test/utils'
import { AIGatewayPage } from '../../pages/AIGateway'

describe('AIGatewayPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with gateways', () => {
    beforeEach(() => {
      setupMockFetch({
        '/ai-gateway': { gateways: mockData.aiGateways() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<AIGatewayPage />)

      expect(await screen.findByText('AI Gateway')).toBeInTheDocument()
    })

    it('renders gateway list', async () => {
      renderWithProviders(<AIGatewayPage />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('gateway-1')).toBeInTheDocument()
      expect(screen.getByText('gateway-2')).toBeInTheDocument()
    })

    it('has create gateway button', async () => {
      renderWithProviders(<AIGatewayPage />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByRole('button', { name: /Create Gateway/i })).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    beforeEach(() => {
      setupMockFetch({
        '/ai-gateway': { gateways: [] },
      })
    })

    it('shows empty state when no gateways', async () => {
      renderWithProviders(<AIGatewayPage />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('No gateways yet')).toBeInTheDocument()
    })
  })

  describe('create gateway form', () => {
    beforeEach(() => {
      setupMockFetch({
        '/ai-gateway': { gateways: mockData.aiGateways() },
      })
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<AIGatewayPage />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      const createButton = screen.getByRole('button', { name: /Create Gateway/i })
      await user.click(createButton)

      expect(await screen.findByText('Create AI Gateway')).toBeInTheDocument()
    })
  })

  describe('error handling', () => {
    it('handles API error gracefully', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('API Error'))

      renderWithProviders(<AIGatewayPage />)

      await waitFor(() => {
        expect(screen.getByText('AI Gateway')).toBeInTheDocument()
      })
    })
  })
})
