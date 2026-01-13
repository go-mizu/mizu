import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderWithProviders, screen, waitFor, setupMockFetch, mockData, userEvent } from '../../test/utils'
import { DurableObjects } from '../../pages/DurableObjects'

describe('DurableObjects', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with namespaces', () => {
    beforeEach(() => {
      setupMockFetch({
        '/durable-objects/namespaces': { namespaces: mockData.durableObjectNamespaces() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<DurableObjects />)

      expect(await screen.findByText('Durable Objects')).toBeInTheDocument()
    })

    it('renders namespace list', async () => {
      renderWithProviders(<DurableObjects />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('namespace-1')).toBeInTheDocument()
      expect(screen.getByText('namespace-2')).toBeInTheDocument()
      expect(screen.getByText('namespace-3')).toBeInTheDocument()
    })

    it('shows namespace class names', async () => {
      renderWithProviders(<DurableObjects />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('Class1')).toBeInTheDocument()
      expect(screen.getByText('Class2')).toBeInTheDocument()
      expect(screen.getByText('Class3')).toBeInTheDocument()
    })

    it('has create namespace button', async () => {
      renderWithProviders(<DurableObjects />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByRole('button', { name: /Create Namespace/i })).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    beforeEach(() => {
      setupMockFetch({
        '/durable-objects/namespaces': { namespaces: [] },
      })
    })

    it('shows empty state when no namespaces', async () => {
      renderWithProviders(<DurableObjects />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      expect(screen.getByText('No namespaces yet')).toBeInTheDocument()
    })
  })

  describe('create namespace form', () => {
    beforeEach(() => {
      setupMockFetch({
        '/durable-objects/namespaces': { namespaces: mockData.durableObjectNamespaces() },
      })
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<DurableObjects />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      })

      const createButton = screen.getByRole('button', { name: /Create Namespace/i })
      await user.click(createButton)

      // Modal should open - check for modal title
      expect(await screen.findByText('Create Durable Object Namespace')).toBeInTheDocument()
    })
  })

  describe('error handling', () => {
    it('handles API error gracefully', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('API Error'))

      renderWithProviders(<DurableObjects />)

      await waitFor(() => {
        expect(screen.getByText('Durable Objects')).toBeInTheDocument()
      })
    })
  })
})
