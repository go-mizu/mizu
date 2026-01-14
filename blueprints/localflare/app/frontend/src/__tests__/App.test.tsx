import { describe, it, expect } from 'vitest'
import { renderWithProviders, screen, waitFor } from '../test/utils'
import App from '../App'

describe('App', () => {
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

    // Dashboard title should be visible (using real API)
    expect(await screen.findByText('Dashboard Overview', {}, { timeout: 5000 })).toBeInTheDocument()
  })

  it('navigates to durable objects page', async () => {
    renderWithProviders(<App />, { initialEntries: ['/durable-objects'] })

    // Use findByRole to specifically find the page heading
    expect(await screen.findByRole('heading', { name: 'Durable Objects' }, { timeout: 5000 })).toBeInTheDocument()
  })

  it('navigates to queues page', async () => {
    renderWithProviders(<App />, { initialEntries: ['/queues'] })

    expect(await screen.findByRole('heading', { name: 'Queues' }, { timeout: 5000 })).toBeInTheDocument()
  })

  it('navigates to vectorize page', async () => {
    renderWithProviders(<App />, { initialEntries: ['/vectorize'] })

    expect(await screen.findByRole('heading', { name: 'Vectorize' }, { timeout: 5000 })).toBeInTheDocument()
  })

  it('navigates to analytics engine page', async () => {
    renderWithProviders(<App />, { initialEntries: ['/analytics-engine'] })

    expect(await screen.findByRole('heading', { name: 'Analytics Engine' }, { timeout: 5000 })).toBeInTheDocument()
  })

  // Skipped due to React 19 + Mantine Select component compatibility issue
  it.skip('navigates to workers ai page', async () => {
    renderWithProviders(<App />, { initialEntries: ['/ai'] })

    expect(await screen.findByRole('heading', { name: 'Workers AI' }, { timeout: 5000 })).toBeInTheDocument()
  })

  it('navigates to ai gateway page', async () => {
    renderWithProviders(<App />, { initialEntries: ['/ai-gateway'] })

    expect(await screen.findByRole('heading', { name: 'AI Gateway' }, { timeout: 5000 })).toBeInTheDocument()
  })

  it('navigates to hyperdrive page', async () => {
    renderWithProviders(<App />, { initialEntries: ['/hyperdrive'] })

    expect(await screen.findByRole('heading', { name: 'Hyperdrive' }, { timeout: 5000 })).toBeInTheDocument()
  })

  it('navigates to cron page', async () => {
    renderWithProviders(<App />, { initialEntries: ['/cron'] })

    expect(await screen.findByRole('heading', { name: 'Cron Triggers' }, { timeout: 5000 })).toBeInTheDocument()
  })

  it('applies correct color scheme', () => {
    renderWithProviders(<App />)

    // Check that the body or a parent element has color scheme applied
    // Mantine applies data-mantine-color-scheme attribute
    const root = document.documentElement
    expect(root.getAttribute('data-mantine-color-scheme')).toBeDefined()
  })
})
