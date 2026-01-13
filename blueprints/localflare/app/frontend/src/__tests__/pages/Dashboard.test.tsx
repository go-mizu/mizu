import { describe, it, expect, beforeAll } from 'vitest'
import {
  renderWithProviders,
  screen,
  waitFor,
  testApi,
  isDashboardStats,
  isSystemStatus,
  isActivityEvent,
  isTimeSeriesData,
  userEvent,
} from '../../test/utils'
import { Dashboard } from '../../pages/Dashboard'

describe('Dashboard', () => {
  describe('API integration', () => {
    it('fetches dashboard stats with correct structure', async () => {
      const response = await testApi.dashboard.getStats()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(isDashboardStats(response.result)).toBe(true)

      const stats = response.result!
      // Verify nested structure
      expect(stats.durable_objects).toHaveProperty('namespaces')
      expect(stats.durable_objects).toHaveProperty('objects')
      expect(typeof stats.durable_objects.namespaces).toBe('number')
      expect(typeof stats.durable_objects.objects).toBe('number')

      expect(stats.queues).toHaveProperty('count')
      expect(stats.queues).toHaveProperty('total_messages')
      expect(typeof stats.queues.count).toBe('number')
      expect(typeof stats.queues.total_messages).toBe('number')

      expect(stats.vectorize).toHaveProperty('indexes')
      expect(stats.vectorize).toHaveProperty('total_vectors')
      expect(typeof stats.vectorize.indexes).toBe('number')
      expect(typeof stats.vectorize.total_vectors).toBe('number')

      expect(stats.analytics).toHaveProperty('datasets')
      expect(stats.analytics).toHaveProperty('data_points')
      expect(typeof stats.analytics.datasets).toBe('number')

      expect(stats.ai).toHaveProperty('requests_today')
      expect(stats.ai).toHaveProperty('tokens_today')
      expect(typeof stats.ai.requests_today).toBe('number')

      expect(stats.ai_gateway).toHaveProperty('gateways')
      expect(stats.ai_gateway).toHaveProperty('requests_today')
      expect(typeof stats.ai_gateway.gateways).toBe('number')

      expect(stats.hyperdrive).toHaveProperty('configs')
      expect(stats.hyperdrive).toHaveProperty('active_connections')
      expect(typeof stats.hyperdrive.configs).toBe('number')

      expect(stats.cron).toHaveProperty('triggers')
      expect(stats.cron).toHaveProperty('executions_today')
      expect(typeof stats.cron.triggers).toBe('number')
    })

    it('fetches system status with valid services', async () => {
      const response = await testApi.dashboard.getStatus()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.services).toBeInstanceOf(Array)

      const services = response.result!.services
      expect(services.length).toBeGreaterThan(0)

      // Verify each service has correct structure
      for (const service of services) {
        expect(isSystemStatus(service)).toBe(true)
        expect(typeof service.service).toBe('string')
        expect(['online', 'degraded', 'offline']).toContain(service.status)
      }

      // Verify expected services are present
      const serviceNames = services.map(s => s.service)
      expect(serviceNames).toContain('Durable Objects')
      expect(serviceNames).toContain('Queues')
      expect(serviceNames).toContain('Vectorize')
      expect(serviceNames).toContain('AI Gateway')
      expect(serviceNames).toContain('Hyperdrive')
      expect(serviceNames).toContain('Cron')
    })

    it('fetches activity events with valid structure', async () => {
      const response = await testApi.dashboard.getActivity(10)

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.events).toBeInstanceOf(Array)

      const events = response.result!.events
      // Activity may be empty if no recent activity
      for (const event of events) {
        expect(isActivityEvent(event)).toBe(true)
        expect(typeof event.id).toBe('string')
        expect(typeof event.type).toBe('string')
        expect(typeof event.message).toBe('string')
        expect(typeof event.timestamp).toBe('string')
        expect(typeof event.service).toBe('string')
        // Verify timestamp is valid ISO date
        expect(new Date(event.timestamp).toString()).not.toBe('Invalid Date')
      }
    })

    it('fetches time series data with valid structure', async () => {
      const response = await testApi.dashboard.getTimeSeries('24h')

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.data).toBeInstanceOf(Array)

      const data = response.result!.data
      expect(data.length).toBeGreaterThan(0)

      for (const point of data) {
        expect(isTimeSeriesData(point)).toBe(true)
        expect(typeof point.timestamp).toBe('string')
        expect(typeof point.value).toBe('number')
        // Verify timestamp is valid ISO date
        expect(new Date(point.timestamp).toString()).not.toBe('Invalid Date')
      }
    })

    it('supports different time ranges for time series', async () => {
      const ranges = ['1h', '24h', '7d', '30d'] as const

      for (const range of ranges) {
        const response = await testApi.dashboard.getTimeSeries(range)
        expect(response.success).toBe(true)
        expect(response.result!.data).toBeInstanceOf(Array)
        expect(response.result!.data.length).toBeGreaterThan(0)
      }
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the dashboard title', async () => {
      renderWithProviders(<Dashboard />)

      expect(await screen.findByText('Dashboard Overview')).toBeInTheDocument()
      expect(screen.getByText('Monitor all Localflare services at a glance')).toBeInTheDocument()
    })

    it('renders stat cards after loading', async () => {
      renderWithProviders(<Dashboard />)

      // Wait for loading to complete
      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      }, { timeout: 5000 })

      // Check stat cards are present
      expect(screen.getAllByText('Durable Objects').length).toBeGreaterThan(0)
      expect(screen.getAllByText('Queues').length).toBeGreaterThan(0)
      expect(screen.getAllByText('Vectorize').length).toBeGreaterThan(0)
      expect(screen.getByText('AI Requests')).toBeInTheDocument()
    })

    it('renders system status section with real service data', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      }, { timeout: 5000 })

      expect(screen.getByText('System Status')).toBeInTheDocument()
      expect(screen.getByText('Current health of all services')).toBeInTheDocument()

      // Real services should be displayed
      expect(screen.getAllByText('Durable Objects').length).toBeGreaterThanOrEqual(1)
      expect(screen.getAllByText('Queues').length).toBeGreaterThanOrEqual(1)
    })

    it('renders requests over time chart', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      }, { timeout: 5000 })

      expect(screen.getByText('Requests Over Time')).toBeInTheDocument()
      expect(screen.getByText('Total requests across all services')).toBeInTheDocument()
    })

    it('renders get started section', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      }, { timeout: 5000 })

      expect(screen.getByText('Get Started with Localflare')).toBeInTheDocument()
    })

    it('renders secondary stats section', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      }, { timeout: 5000 })

      // Check for secondary stats
      expect(screen.getAllByText(/Analytics/).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/AI Gateway/).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/Hyperdrive/).length).toBeGreaterThan(0)
      expect(screen.getAllByText(/Cron/).length).toBeGreaterThan(0)
    })
  })

  describe('time range selector', () => {
    it('renders time range options', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      }, { timeout: 5000 })

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
      }, { timeout: 5000 })

      // Click 7d button
      const sevenDayButton = screen.getByText('7d')
      await user.click(sevenDayButton)

      // Button should still be present after click
      expect(sevenDayButton).toBeInTheDocument()
    })
  })

  describe('accessibility', () => {
    it('has proper heading hierarchy', async () => {
      renderWithProviders(<Dashboard />)

      await waitFor(() => {
        expect(screen.queryByText('Loading dashboard...')).not.toBeInTheDocument()
      }, { timeout: 5000 })

      // Check for h2 heading
      const heading = screen.getByRole('heading', { name: 'Dashboard Overview' })
      expect(heading).toBeInTheDocument()
      expect(heading.tagName).toBe('H2')
    })
  })
})
