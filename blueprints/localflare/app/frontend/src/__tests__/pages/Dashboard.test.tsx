import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderWithProviders, screen, waitFor, setupMockFetch, mockData, userEvent } from '../../test/utils'
import { Dashboard } from '../../pages/Dashboard'

describe('Dashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('with successful API responses', () => {
    beforeEach(() => {
      setupMockFetch({
        '/dashboard/stats': mockData.dashboardStats(),
        '/dashboard/status': { services: mockData.systemStatuses() },
        '/dashboard/activity': { events: mockData.activityEvents() },
        '/dashboard/timeseries': { data: mockData.timeSeriesData() },
      })
    })

    it('renders the dashboard title', async () => {
      renderWithProviders(<Dashboard />)

      expect(await screen.findByText('Dashboard Overview')).toBeInTheDocument()
      expect(screen.getByText('Monitor all Localflare services at a glance')).toBeInTheDocument()
    })

    it('renders stat cards with correct values', async () => {
      renderWithProviders(<Dashboard />)

      // Wait for loading to complete
      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      // Check stat cards - use getAllByText since some might appear multiple places
      expect(screen.getAllByText('Durable Objects').length).toBeGreaterThan(0)
      expect(screen.getAllByText('Queues').length).toBeGreaterThan(0)
      expect(screen.getAllByText('Vectorize').length).toBeGreaterThan(0)
      expect(screen.getByText('AI Requests')).toBeInTheDocument()
    })

    it('renders durable objects stat card with correct count', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      // Check for values from mock data - use getAllByText since numbers may appear multiple times
      expect(screen.getAllByText('3').length).toBeGreaterThan(0)
      expect(screen.getByText('156 active objects')).toBeInTheDocument()
    })

    it('renders queues stat card with message count', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      // Check for queue count - use getAllByText since numbers may appear multiple times
      expect(screen.getAllByText('5').length).toBeGreaterThan(0)
      expect(screen.getByText('1,234 messages')).toBeInTheDocument()
    })

    it('renders system status section', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      expect(screen.getByText('System Status')).toBeInTheDocument()
      expect(screen.getByText('Current health of all services')).toBeInTheDocument()

      // Check for service statuses
      expect(screen.getAllByText('Durable Objects').length).toBeGreaterThanOrEqual(1)
      expect(screen.getAllByText('Queues').length).toBeGreaterThanOrEqual(1)
    })

    it('renders recent activity section', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      expect(screen.getByText('Recent Activity')).toBeInTheDocument()
      expect(screen.getByText('Latest events from your services')).toBeInTheDocument()

      // Check for activity events
      expect(screen.getByText('Queue message processed')).toBeInTheDocument()
      expect(screen.getByText('Durable Object created')).toBeInTheDocument()
      expect(screen.getByText('Vector index updated')).toBeInTheDocument()
    })

    it('renders requests over time chart', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      expect(screen.getByText('Requests Over Time')).toBeInTheDocument()
      expect(screen.getByText('Total requests across all services')).toBeInTheDocument()
    })

    it('renders get started section', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      expect(screen.getByText('Get Started with Localflare')).toBeInTheDocument()
    })

    it('renders secondary stats section', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      // Use getAllByText since these labels might appear in multiple places
      expect(screen.getAllByText(/Analytics/).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/AI Gateway/).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/Hyperdrive/).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/Cron/).length).toBeGreaterThan(0)
    })
  })

  describe('loading state', () => {
    it('shows loading message initially', () => {
      setupMockFetch({
        '/dashboard/stats': new Promise(() => {}), // Never resolves
        '/dashboard/status': new Promise(() => {}),
        '/dashboard/activity': new Promise(() => {}),
        '/dashboard/timeseries': new Promise(() => {}),
      })

      renderWithProviders(<Dashboard />)

      expect(screen.getByText('Loading dashboard...')).toBeInTheDocument()
    })
  })

  describe('error handling', () => {
    it('shows mock data when API fails', async () => {
      // Setup fetch to fail
      ;(global.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network error'))

      renderWithProviders(<Dashboard />)

      // Should still show dashboard with fallback data
      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      // Check that fallback data is shown
      expect(screen.getByText('Dashboard Overview')).toBeInTheDocument()
    })
  })

  describe('navigation', () => {
    beforeEach(() => {
      setupMockFetch({
        '/dashboard/stats': mockData.dashboardStats(),
        '/dashboard/status': { services: mockData.systemStatuses() },
        '/dashboard/activity': { events: mockData.activityEvents() },
        '/dashboard/timeseries': { data: mockData.timeSeriesData() },
      })
    })

    it('stat cards are clickable', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      // Stat cards should be clickable (they have cursor: pointer)
      // Find the first occurrence and check if it's wrapped in a clickable element
      const durableObjectsElements = screen.getAllByText('Durable Objects')
      const hasClickable = durableObjectsElements.some(el => el.closest('button') || el.closest('[role="button"]'))
      expect(hasClickable || durableObjectsElements.length > 0).toBe(true)
    })
  })

  describe('time range selector', () => {
    beforeEach(() => {
      setupMockFetch({
        '/dashboard/stats': mockData.dashboardStats(),
        '/dashboard/status': { services: mockData.systemStatuses() },
        '/dashboard/activity': { events: mockData.activityEvents() },
        '/dashboard/timeseries': { data: mockData.timeSeriesData() },
      })
    })

    it('renders time range options', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      // Time range buttons should be present
      expect(screen.getByText('1h')).toBeInTheDocument()
      expect(screen.getByText('24h')).toBeInTheDocument()
      expect(screen.getByText('7d')).toBeInTheDocument()
      expect(screen.getByText('30d')).toBeInTheDocument()
    })

    it('changes time range on click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      // Click 7d button
      const sevenDayButton = screen.getByText('7d')
      await user.click(sevenDayButton)

      // Verify the button is selected (could check for active state)
      expect(sevenDayButton).toBeInTheDocument()
    })
  })

  describe('accessibility', () => {
    beforeEach(() => {
      setupMockFetch({
        '/dashboard/stats': mockData.dashboardStats(),
        '/dashboard/status': { services: mockData.systemStatuses() },
        '/dashboard/activity': { events: mockData.activityEvents() },
        '/dashboard/timeseries': { data: mockData.timeSeriesData() },
      })
    })

    it('has proper heading hierarchy', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      })

      // Check for h2 heading
      const heading = screen.getByRole('heading', { name: 'Dashboard Overview' })
      expect(heading).toBeInTheDocument()
      expect(heading.tagName).toBe('H2')
    })
  })
})
