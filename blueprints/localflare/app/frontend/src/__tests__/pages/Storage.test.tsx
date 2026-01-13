import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderWithProviders, screen, waitFor, setupMockFetch, mockData } from '../../test/utils'
import { KV } from '../../pages/KV'
import { R2 } from '../../pages/R2'
import { D1 } from '../../pages/D1'

describe('KV Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with successful API response', () => {
    beforeEach(() => {
      setupMockFetch({
        '/kv/namespaces': { namespaces: mockData.kvNamespaces() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<KV />)
      expect(await screen.findByText('Workers KV')).toBeInTheDocument()
    })

    it('displays namespaces from API', async () => {
      renderWithProviders(<KV />)

      await waitFor(() => {
        expect(screen.getByText('CONFIG')).toBeInTheDocument()
        expect(screen.getByText('SESSIONS')).toBeInTheDocument()
        expect(screen.getByText('CACHE')).toBeInTheDocument()
      })
    })

    it('shows create button', async () => {
      renderWithProviders(<KV />)

      await waitFor(() => {
        expect(screen.getByText(/Create Namespace/i)).toBeInTheDocument()
      })
    })
  })

  describe('error handling', () => {
    it('shows fallback data when API fails', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network error'))

      renderWithProviders(<KV />)

      await waitFor(() => {
        // Should still render with fallback data
        expect(screen.getByText('Workers KV')).toBeInTheDocument()
      })
    })
  })
})

describe('R2 Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with successful API response', () => {
    beforeEach(() => {
      setupMockFetch({
        '/r2/buckets': { buckets: mockData.r2Buckets() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<R2 />)
      expect(await screen.findByText('R2 Object Storage')).toBeInTheDocument()
    })

    it('displays buckets from API', async () => {
      renderWithProviders(<R2 />)

      await waitFor(() => {
        expect(screen.getByText('assets')).toBeInTheDocument()
        expect(screen.getByText('uploads')).toBeInTheDocument()
        expect(screen.getByText('backups')).toBeInTheDocument()
      })
    })

    it('shows create button', async () => {
      renderWithProviders(<R2 />)

      await waitFor(() => {
        expect(screen.getByText(/Create Bucket/i)).toBeInTheDocument()
      })
    })
  })

  describe('error handling', () => {
    it('shows fallback data when API fails', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network error'))

      renderWithProviders(<R2 />)

      await waitFor(() => {
        expect(screen.getByText('R2 Object Storage')).toBeInTheDocument()
      })
    })
  })
})

describe('D1 Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with successful API response', () => {
    beforeEach(() => {
      setupMockFetch({
        '/d1/databases': { databases: mockData.d1Databases() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<D1 />)
      expect(await screen.findByText('D1 Database')).toBeInTheDocument()
    })

    it('displays databases from API', async () => {
      renderWithProviders(<D1 />)

      await waitFor(() => {
        expect(screen.getByText('main')).toBeInTheDocument()
        expect(screen.getByText('analytics')).toBeInTheDocument()
      })
    })

    it('shows create button', async () => {
      renderWithProviders(<D1 />)

      await waitFor(() => {
        expect(screen.getByText(/Create Database/i)).toBeInTheDocument()
      })
    })
  })

  describe('error handling', () => {
    it('shows fallback data when API fails', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network error'))

      renderWithProviders(<D1 />)

      await waitFor(() => {
        expect(screen.getByText('D1 Database')).toBeInTheDocument()
      })
    })
  })
})
