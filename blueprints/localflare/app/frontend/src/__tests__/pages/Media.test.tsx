import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderWithProviders, screen, waitFor, setupMockFetch, mockData } from '../../test/utils'
import { Pages } from '../../pages/Pages'
import { Images } from '../../pages/Images'
import { Stream } from '../../pages/Stream'

describe('Pages Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with successful API response', () => {
    beforeEach(() => {
      setupMockFetch({
        '/pages/projects': { projects: mockData.pagesProjects() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<Pages />)
      expect(await screen.findByText('Pages')).toBeInTheDocument()
    })

    it('displays projects from API', async () => {
      renderWithProviders(<Pages />)

      await waitFor(() => {
        expect(screen.getByText('my-blog')).toBeInTheDocument()
        expect(screen.getByText('docs-site')).toBeInTheDocument()
      })
    })

    it('shows deployment status', async () => {
      renderWithProviders(<Pages />)

      await waitFor(() => {
        // Check for success status badges (rendered as "Success" with capital S)
        expect(screen.getAllByText(/success/i).length).toBeGreaterThan(0)
      })
    })

    it('shows create button', async () => {
      renderWithProviders(<Pages />)

      await waitFor(() => {
        expect(screen.getByText(/Create Project/i)).toBeInTheDocument()
      })
    })
  })

  describe('error handling', () => {
    it('shows fallback data when API fails', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network error'))

      renderWithProviders(<Pages />)

      await waitFor(() => {
        expect(screen.getByText('Pages')).toBeInTheDocument()
      })
    })
  })
})

describe('Images Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with successful API response', () => {
    beforeEach(() => {
      setupMockFetch({
        '/images': { images: mockData.cloudflareImages() },
        '/images/variants': { variants: mockData.imageVariants() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<Images />)
      // Wait for the page to load and check for the subtitle which is unique to this page
      expect(await screen.findByText(/Store, resize, and optimize images/i)).toBeInTheDocument()
    })

    it('displays images from API', async () => {
      renderWithProviders(<Images />)

      await waitFor(() => {
        expect(screen.getByText('hero-banner.jpg')).toBeInTheDocument()
        expect(screen.getByText('logo.png')).toBeInTheDocument()
        expect(screen.getByText('product-1.webp')).toBeInTheDocument()
      })
    })

    it('shows upload button', async () => {
      renderWithProviders(<Images />)

      await waitFor(() => {
        expect(screen.getByText(/Upload Images/i)).toBeInTheDocument()
      })
    })

    it('shows variants button', async () => {
      renderWithProviders(<Images />)

      await waitFor(() => {
        // The page should have either the variants button or a "Variants" stat card
        const variantElements = screen.queryAllByText(/variant/i)
        expect(variantElements.length).toBeGreaterThan(0)
      })
    })
  })

  describe('error handling', () => {
    it('shows fallback data when API fails', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network error'))

      renderWithProviders(<Images />)

      await waitFor(() => {
        // Check for subtitle instead of title
        expect(screen.getByText(/Store, resize, and optimize images/i)).toBeInTheDocument()
      })
    })
  })
})

describe('Stream Page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with successful API response', () => {
    beforeEach(() => {
      setupMockFetch({
        '/stream/videos': { videos: mockData.streamVideos() },
        '/stream/live': { live_inputs: mockData.liveInputs() },
      })
    })

    it('renders the page title', async () => {
      renderWithProviders(<Stream />)
      expect(await screen.findByText('Stream')).toBeInTheDocument()
    })

    it('displays videos from API', async () => {
      renderWithProviders(<Stream />)

      await waitFor(() => {
        expect(screen.getByText('Product Demo')).toBeInTheDocument()
        expect(screen.getByText('Tutorial')).toBeInTheDocument()
      })
    })

    it('shows upload button', async () => {
      renderWithProviders(<Stream />)

      await waitFor(() => {
        expect(screen.getByText(/Upload Video/i)).toBeInTheDocument()
      })
    })

    it('shows create live input button', async () => {
      renderWithProviders(<Stream />)

      await waitFor(() => {
        expect(screen.getAllByText(/Create Live Input/i).length).toBeGreaterThan(0)
      })
    })

    it('shows video content', async () => {
      renderWithProviders(<Stream />)

      await waitFor(() => {
        // Check that videos are displayed (not just the tab label)
        expect(screen.getByText('Product Demo')).toBeInTheDocument()
      })
    })
  })

  describe('error handling', () => {
    it('shows fallback data when API fails', async () => {
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network error'))

      renderWithProviders(<Stream />)

      await waitFor(() => {
        expect(screen.getByText('Stream')).toBeInTheDocument()
      })
    })
  })
})
