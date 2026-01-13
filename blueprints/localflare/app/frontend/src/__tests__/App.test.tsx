import { describe, it, expect, beforeEach } from 'vitest'
import { renderWithProviders, screen, setupMockFetch, mockData } from '../test/utils'
import App from '../App'

describe('App', () => {
  beforeEach(() => {
    // Setup default mock responses
    setupMockFetch({
      '/dashboard/stats': mockData.dashboardStats(),
      '/dashboard/status': { services: mockData.systemStatuses() },
      '/dashboard/activity': { events: mockData.activityEvents() },
      '/dashboard/timeseries': { data: mockData.timeSeriesData() },
    })
  })

  it('renders without crashing', () => {
    renderWithProviders(<App />)
    expect(document.getElementById('root') || document.body).toBeInTheDocument()
  })

  it('renders the sidebar navigation', async () => {
    renderWithProviders(<App />)

    // Check for Localflare branding
    expect(screen.getByText('Localflare')).toBeInTheDocument()

    // Check for navigation items
    expect(screen.getByText('Overview')).toBeInTheDocument()
    expect(screen.getByText('Durable Objects')).toBeInTheDocument()
    expect(screen.getByText('Queues')).toBeInTheDocument()
    expect(screen.getByText('Vectorize')).toBeInTheDocument()
    expect(screen.getByText('Analytics Engine')).toBeInTheDocument()
    expect(screen.getByText('Workers AI')).toBeInTheDocument()
    expect(screen.getByText('AI Gateway')).toBeInTheDocument()
    expect(screen.getByText('Hyperdrive')).toBeInTheDocument()
    expect(screen.getByText('Cron Triggers')).toBeInTheDocument()
  })

  it('renders the dashboard by default', async () => {
    renderWithProviders(<App />)

    // Dashboard title should be visible
    expect(await screen.findByText('Dashboard Overview')).toBeInTheDocument()
  })

  it('navigates to durable objects page', async () => {
    setupMockFetch({
      '/dashboard/stats': mockData.dashboardStats(),
      '/dashboard/status': { services: mockData.systemStatuses() },
      '/dashboard/activity': { events: mockData.activityEvents() },
      '/dashboard/timeseries': { data: mockData.timeSeriesData() },
      '/durable-objects/namespaces': { namespaces: mockData.durableObjectNamespaces() },
    })

    renderWithProviders(<App />, { initialEntries: ['/durable-objects'] })

    // Use findByRole to specifically find the page heading
    expect(await screen.findByRole('heading', { name: 'Durable Objects' })).toBeInTheDocument()
  })

  it('navigates to queues page', async () => {
    setupMockFetch({
      '/queues': { queues: mockData.queues() },
    })

    renderWithProviders(<App />, { initialEntries: ['/queues'] })

    expect(await screen.findByRole('heading', { name: 'Queues' })).toBeInTheDocument()
  })

  it('navigates to vectorize page', async () => {
    setupMockFetch({
      '/vectorize/indexes': { indexes: mockData.vectorIndexes() },
    })

    renderWithProviders(<App />, { initialEntries: ['/vectorize'] })

    expect(await screen.findByRole('heading', { name: 'Vectorize' })).toBeInTheDocument()
  })

  it('navigates to analytics engine page', async () => {
    setupMockFetch({
      '/analytics/datasets': { datasets: mockData.analyticsDatasets() },
    })

    renderWithProviders(<App />, { initialEntries: ['/analytics-engine'] })

    expect(await screen.findByRole('heading', { name: 'Analytics Engine' })).toBeInTheDocument()
  })

  // Skipped due to React 19 + Mantine Select component compatibility issue
  it.skip('navigates to workers ai page', async () => {
    setupMockFetch({
      '/ai/models': { models: mockData.aiModels() },
      '/ai/stats': { requests_today: 100, tokens_today: 5000, cost_today: 0.50 },
    })

    renderWithProviders(<App />, { initialEntries: ['/ai'] })

    expect(await screen.findByRole('heading', { name: 'Workers AI' })).toBeInTheDocument()
  })

  it('navigates to ai gateway page', async () => {
    setupMockFetch({
      '/ai-gateway': { gateways: mockData.aiGateways() },
    })

    renderWithProviders(<App />, { initialEntries: ['/ai-gateway'] })

    expect(await screen.findByRole('heading', { name: 'AI Gateway' })).toBeInTheDocument()
  })

  it('navigates to hyperdrive page', async () => {
    setupMockFetch({
      '/hyperdrive': { configs: mockData.hyperdriveConfigs() },
    })

    renderWithProviders(<App />, { initialEntries: ['/hyperdrive'] })

    expect(await screen.findByRole('heading', { name: 'Hyperdrive' })).toBeInTheDocument()
  })

  it('navigates to cron page', async () => {
    setupMockFetch({
      '/cron': { triggers: mockData.cronTriggers() },
    })

    renderWithProviders(<App />, { initialEntries: ['/cron'] })

    expect(await screen.findByRole('heading', { name: 'Cron Triggers' })).toBeInTheDocument()
  })

  it('applies dark theme', () => {
    renderWithProviders(<App />)

    // Check that the body or a parent element has dark theme applied
    // Mantine applies data-mantine-color-scheme attribute
    const root = document.documentElement
    expect(root).toHaveAttribute('data-mantine-color-scheme', 'dark')
  })
})
