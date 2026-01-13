import { describe, it, expect, afterAll } from 'vitest'
import {
  renderWithProviders,
  screen,
  waitFor,
  testApi,
  isPagesProject,
  isCloudflareImage,
  isStreamVideo,
  isLiveInput,
  generateTestName,
} from '../../test/utils'
import { Pages } from '../../pages/Pages'
import { Images } from '../../pages/Images'
import { Stream } from '../../pages/Stream'
import type { PagesProject, LiveInput } from '../../types'

describe('Pages Page', () => {
  // Track created projects for cleanup
  const createdProjectNames: string[] = []

  afterAll(async () => {
    for (const name of createdProjectNames) {
      try {
        await testApi.pages.deleteProject(name)
      } catch {
        // Ignore cleanup errors
      }
    }
  })

  describe('API integration', () => {
    it('fetches projects list with correct structure', async () => {
      const response = await testApi.pages.listProjects()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.projects).toBeInstanceOf(Array)

      const projects = response.result!.projects
      for (const project of projects) {
        expect(isPagesProject(project)).toBe(true)
        expect(typeof project.name).toBe('string')
        expect(typeof project.subdomain).toBe('string')
        expect(typeof project.created_at).toBe('string')
      }
    })

    it('creates a new project with valid structure', async () => {
      const name = generateTestName('pages').toLowerCase().replace(/[^a-z0-9-]/g, '-')
      const response = await testApi.pages.createProject({ name })

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()

      const project = response.result!
      expect(isPagesProject(project)).toBe(true)
      expect(project.name).toBe(name)

      createdProjectNames.push(name)
    })

    it('deletes a project successfully', async () => {
      const name = generateTestName('pages-delete').toLowerCase().replace(/[^a-z0-9-]/g, '-')
      const createResponse = await testApi.pages.createProject({ name })
      expect(createResponse.success).toBe(true)

      const deleteResponse = await testApi.pages.deleteProject(name)
      expect(deleteResponse.success).toBe(true)

      const listResponse = await testApi.pages.listProjects()
      const names = listResponse.result!.projects.map((p: PagesProject) => p.name)
      expect(names).not.toContain(name)
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page title', async () => {
      renderWithProviders(<Pages />)
      expect(await screen.findByText('Pages')).toBeInTheDocument()
    })

    it('displays projects from real API', async () => {
      const name = generateTestName('pages-ui').toLowerCase().replace(/[^a-z0-9-]/g, '-')
      await testApi.pages.createProject({ name })
      createdProjectNames.push(name)

      renderWithProviders(<Pages />)

      await waitFor(() => {
        expect(screen.getByText(name)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('shows create button', async () => {
      renderWithProviders(<Pages />)

      await waitFor(() => {
        expect(screen.getByText(/Create Project/i)).toBeInTheDocument()
      }, { timeout: 5000 })
    })
  })
})

describe('Images Page', () => {
  describe('API integration', () => {
    it('fetches images list with correct structure', async () => {
      const response = await testApi.images.list()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.images).toBeInstanceOf(Array)

      const images = response.result!.images
      for (const image of images) {
        expect(isCloudflareImage(image)).toBe(true)
        expect(typeof image.id).toBe('string')
        expect(typeof image.filename).toBe('string')
        expect(typeof image.uploaded).toBe('string')
      }
    })

    it('fetches variants list with correct structure', async () => {
      const response = await testApi.images.listVariants()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.variants).toBeInstanceOf(Array)

      const variants = response.result!.variants
      for (const variant of variants) {
        expect(typeof variant.id).toBe('string')
        expect(typeof variant.name).toBe('string')
        expect(variant.options).toBeDefined()
      }
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page subtitle', async () => {
      renderWithProviders(<Images />)
      expect(await screen.findByText(/Store, resize, and optimize images/i)).toBeInTheDocument()
    })

    it('shows upload button', async () => {
      renderWithProviders(<Images />)

      await waitFor(() => {
        expect(screen.getByText(/Upload Images/i)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('shows variant-related content', async () => {
      renderWithProviders(<Images />)

      await waitFor(() => {
        const variantElements = screen.queryAllByText(/variant/i)
        expect(variantElements.length).toBeGreaterThan(0)
      }, { timeout: 5000 })
    })
  })
})

describe('Stream Page', () => {
  // Track created live inputs for cleanup
  const createdLiveInputIds: string[] = []

  afterAll(async () => {
    // Note: No delete API in testApi for live inputs, cleanup may need manual handling
  })

  describe('API integration', () => {
    it('fetches videos list with correct structure', async () => {
      const response = await testApi.stream.listVideos()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.videos).toBeInstanceOf(Array)

      const videos = response.result!.videos
      for (const video of videos) {
        expect(isStreamVideo(video)).toBe(true)
        expect(typeof video.uid).toBe('string')
        expect(typeof video.name).toBe('string')
        expect(typeof video.created).toBe('string')
        expect(typeof video.duration).toBe('number')
        expect(typeof video.size).toBe('number')
        expect(video.status).toBeDefined()
        expect(typeof video.status.state).toBe('string')
      }
    })

    it('fetches live inputs list with correct structure', async () => {
      const response = await testApi.stream.listLiveInputs()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.live_inputs).toBeInstanceOf(Array)

      const liveInputs = response.result!.live_inputs
      for (const input of liveInputs) {
        expect(isLiveInput(input)).toBe(true)
        expect(typeof input.uid).toBe('string')
        expect(typeof input.name).toBe('string')
        expect(['connected', 'disconnected']).toContain(input.status)
        expect(input.rtmps).toBeDefined()
        expect(typeof input.rtmps.url).toBe('string')
        expect(typeof input.rtmps.streamKey).toBe('string')
      }
    })

    it('creates a new live input with valid structure', async () => {
      const name = generateTestName('live')
      const response = await testApi.stream.createLiveInput({ name })

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()

      const input = response.result!
      expect(isLiveInput(input)).toBe(true)
      expect(input.name).toBe(name)
      expect(typeof input.uid).toBe('string')

      createdLiveInputIds.push(input.uid)
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page title', async () => {
      renderWithProviders(<Stream />)
      expect(await screen.findByText('Stream')).toBeInTheDocument()
    })

    it('shows upload button', async () => {
      renderWithProviders(<Stream />)

      await waitFor(() => {
        expect(screen.getByText(/Upload Video/i)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('shows create live input button', async () => {
      renderWithProviders(<Stream />)

      await waitFor(() => {
        expect(screen.getAllByText(/Create Live Input/i).length).toBeGreaterThan(0)
      }, { timeout: 5000 })
    })

    it('displays live inputs from real API', async () => {
      const name = generateTestName('live-ui')
      const createResponse = await testApi.stream.createLiveInput({ name })
      createdLiveInputIds.push(createResponse.result!.uid)

      renderWithProviders(<Stream />)

      await waitFor(() => {
        expect(screen.getByText(name)).toBeInTheDocument()
      }, { timeout: 5000 })
    })
  })
})
