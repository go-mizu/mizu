import { render, type RenderOptions } from '@testing-library/react'
import { MantineProvider, createTheme } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { BrowserRouter, MemoryRouter } from 'react-router-dom'
import { type ReactElement, type ReactNode } from 'react'
import { vi } from 'vitest'
import type {
  DashboardStats,
  DurableObjectNamespace,
  Queue,
  VectorIndex,
  AnalyticsDataset,
  AIGateway,
  HyperdriveConfig,
  CronTrigger,
  SystemStatus,
  ActivityEvent,
  TimeSeriesData,
} from '../types'

// Test theme matching main.tsx
const testTheme = createTheme({
  primaryColor: 'orange',
  defaultRadius: 'md',
})

interface WrapperProps {
  children: ReactNode
  initialEntries?: string[]
}

// Default wrapper with all providers
function AllProviders({ children, initialEntries }: WrapperProps) {
  const Router = initialEntries ? MemoryRouter : BrowserRouter
  const routerProps = initialEntries ? { initialEntries } : {}

  return (
    <MantineProvider theme={testTheme} defaultColorScheme="dark">
      <Notifications position="top-right" />
      <Router {...routerProps}>{children}</Router>
    </MantineProvider>
  )
}

// Custom render function
interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  initialEntries?: string[]
}

export function renderWithProviders(
  ui: ReactElement,
  options?: CustomRenderOptions
) {
  const { initialEntries, ...renderOptions } = options || {}

  return render(ui, {
    wrapper: ({ children }) => (
      <AllProviders initialEntries={initialEntries}>{children}</AllProviders>
    ),
    ...renderOptions,
  })
}

// Mock API response helper
export function mockApiResponse<T>(data: T) {
  return {
    success: true,
    result: data,
  }
}

// Mock data factories
export const mockData = {
  dashboardStats: (): DashboardStats => ({
    durable_objects: { namespaces: 3, objects: 156 },
    queues: { count: 5, total_messages: 1234 },
    vectorize: { indexes: 3, total_vectors: 50234 },
    analytics: { datasets: 4, data_points: 1200000 },
    ai: { requests_today: 1245, tokens_today: 2100000 },
    ai_gateway: { gateways: 2, requests_today: 4520 },
    hyperdrive: { configs: 3, active_connections: 12 },
    cron: { triggers: 5, executions_today: 288 },
  }),

  systemStatuses: (): SystemStatus[] => [
    { service: 'Durable Objects', status: 'online' },
    { service: 'Queues', status: 'online' },
    { service: 'Vectorize', status: 'online' },
    { service: 'Analytics Engine', status: 'online' },
    { service: 'Workers AI', status: 'online' },
    { service: 'AI Gateway', status: 'online' },
    { service: 'Hyperdrive', status: 'online' },
    { service: 'Cron', status: 'online' },
  ],

  activityEvents: (): ActivityEvent[] => [
    {
      id: '1',
      type: 'queue',
      message: 'Queue message processed',
      timestamp: new Date().toISOString(),
      service: 'Queues',
    },
    {
      id: '2',
      type: 'do',
      message: 'Durable Object created',
      timestamp: new Date(Date.now() - 60000).toISOString(),
      service: 'Durable Objects',
    },
    {
      id: '3',
      type: 'vector',
      message: 'Vector index updated',
      timestamp: new Date(Date.now() - 120000).toISOString(),
      service: 'Vectorize',
    },
  ],

  timeSeriesData: (points = 24): TimeSeriesData[] => {
    const now = Date.now()
    return Array.from({ length: points }, (_, i) => ({
      timestamp: new Date(now - (points - i) * 3600000).toISOString(),
      value: Math.floor(Math.random() * 1000) + 500,
    }))
  },

  durableObjectNamespace: (overrides?: Partial<DurableObjectNamespace>): DurableObjectNamespace => ({
    id: 'ns-1',
    name: 'test-namespace',
    class_name: 'TestClass',
    script_name: 'test-worker',
    created_at: new Date().toISOString(),
    object_count: 10,
    ...overrides,
  }),

  durableObjectNamespaces: (count = 3): DurableObjectNamespace[] =>
    Array.from({ length: count }, (_, i) =>
      mockData.durableObjectNamespace({
        id: `ns-${i + 1}`,
        name: `namespace-${i + 1}`,
        class_name: `Class${i + 1}`,
        object_count: (i + 1) * 10,
      })
    ),

  queue: (overrides?: Partial<Queue>): Queue => ({
    id: 'q-1',
    name: 'test-queue',
    created_at: new Date().toISOString(),
    message_count: 100,
    ready_count: 80,
    delayed_count: 15,
    failed_count: 5,
    settings: {
      max_retries: 3,
      batch_size: 10,
      max_batch_timeout: 5000,
      message_retention_seconds: 86400,
      delivery_delay: 0,
    },
    consumers: [],
    ...overrides,
  }),

  queues: (count = 3): Queue[] =>
    Array.from({ length: count }, (_, i) =>
      mockData.queue({
        id: `q-${i + 1}`,
        name: `queue-${i + 1}`,
        message_count: (i + 1) * 100,
      })
    ),

  vectorIndex: (overrides?: Partial<VectorIndex>): VectorIndex => ({
    id: 'idx-1',
    name: 'test-index',
    dimensions: 768,
    metric: 'cosine',
    description: 'Test vector index',
    created_at: new Date().toISOString(),
    vector_count: 10000,
    namespace_count: 3,
    ...overrides,
  }),

  vectorIndexes: (count = 3): VectorIndex[] =>
    Array.from({ length: count }, (_, i) =>
      mockData.vectorIndex({
        id: `idx-${i + 1}`,
        name: `index-${i + 1}`,
        dimensions: [384, 768, 1536][i] || 768,
        vector_count: (i + 1) * 5000,
      })
    ),

  analyticsDataset: (overrides?: Partial<AnalyticsDataset>): AnalyticsDataset => ({
    id: 'ds-1',
    name: 'test-dataset',
    created_at: new Date().toISOString(),
    data_points: 50000,
    estimated_size_bytes: 5000000,
    last_write: new Date().toISOString(),
    ...overrides,
  }),

  analyticsDatasets: (count = 3): AnalyticsDataset[] =>
    Array.from({ length: count }, (_, i) =>
      mockData.analyticsDataset({
        id: `ds-${i + 1}`,
        name: `dataset-${i + 1}`,
        data_points: (i + 1) * 25000,
      })
    ),

  aiGateway: (overrides?: Partial<AIGateway>): AIGateway => ({
    id: 'gw-1',
    name: 'test-gateway',
    created_at: new Date().toISOString(),
    settings: {
      cache_enabled: true,
      cache_ttl: 3600,
      rate_limit_enabled: true,
      rate_limit: 1000,
      rate_limit_period: '1m',
      logging_enabled: true,
      retry_enabled: true,
      retry_count: 3,
    },
    stats: {
      total_requests: 5000,
      cached_requests: 1500,
      error_count: 50,
      total_tokens: 500000,
      total_cost: 25.50,
    },
    ...overrides,
  }),

  aiGateways: (count = 2): AIGateway[] =>
    Array.from({ length: count }, (_, i) =>
      mockData.aiGateway({
        id: `gw-${i + 1}`,
        name: `gateway-${i + 1}`,
      })
    ),

  hyperdriveConfig: (overrides?: Partial<HyperdriveConfig>): HyperdriveConfig => ({
    id: 'hd-1',
    name: 'test-hyperdrive',
    created_at: new Date().toISOString(),
    origin: {
      scheme: 'postgres',
      host: 'db.example.com',
      port: 5432,
      database: 'testdb',
      user: 'testuser',
    },
    caching: {
      enabled: true,
      max_age: 300,
      stale_while_revalidate: 60,
    },
    status: 'connected',
    ...overrides,
  }),

  hyperdriveConfigs: (count = 3): HyperdriveConfig[] =>
    Array.from({ length: count }, (_, i) =>
      mockData.hyperdriveConfig({
        id: `hd-${i + 1}`,
        name: `hyperdrive-${i + 1}`,
        status: ['connected', 'idle', 'disconnected'][i] as HyperdriveConfig['status'],
      })
    ),

  cronTrigger: (overrides?: Partial<CronTrigger>): CronTrigger => ({
    id: 'cron-1',
    cron: '*/5 * * * *',
    script_name: 'test-worker',
    enabled: true,
    created_at: new Date().toISOString(),
    last_run: new Date(Date.now() - 300000).toISOString(),
    next_run: new Date(Date.now() + 300000).toISOString(),
    ...overrides,
  }),

  cronTriggers: (count = 3): CronTrigger[] =>
    Array.from({ length: count }, (_, i) =>
      mockData.cronTrigger({
        id: `cron-${i + 1}`,
        cron: ['*/5 * * * *', '0 * * * *', '0 0 * * *'][i] || '*/5 * * * *',
        script_name: `worker-${i + 1}`,
        enabled: i !== 2,
      })
    ),

  // AI Models
  aiModels: () => [
    { id: 'llama-2-7b', name: '@cf/meta/llama-2-7b-chat-int8', task: 'text-generation' },
    { id: 'mistral-7b', name: '@cf/mistral/mistral-7b-instruct-v0.1', task: 'text-generation' },
    { id: 'bge-small', name: '@cf/baai/bge-small-en-v1.5', task: 'text-embeddings' },
  ],
}

// Setup mock API responses
export function setupMockFetch(responses: Record<string, unknown>) {
  const mockFn = vi.mocked(global.fetch)
  mockFn.mockReset()

  return mockFn.mockImplementation(async (input: RequestInfo | URL) => {
    const url = typeof input === 'string' ? input : input.toString()
    const path = url.replace('/api', '')

    // Find matching response
    let matchedResponse: unknown = undefined
    for (const [key, value] of Object.entries(responses)) {
      if (path === key || path.startsWith(key + '?')) {
        matchedResponse = value
        break
      }
    }

    if (matchedResponse === undefined) {
      return {
        ok: false,
        status: 404,
        json: async () => ({ error: 'Not found', path }),
        text: async () => JSON.stringify({ error: 'Not found', path }),
        headers: new Headers(),
      } as Response
    }

    return {
      ok: true,
      status: 200,
      json: async () => mockApiResponse(matchedResponse),
      text: async () => JSON.stringify(mockApiResponse(matchedResponse)),
      headers: new Headers(),
    } as Response
  })
}

// Wait for loading to complete
export async function waitForLoadingToComplete() {
  const { waitFor } = await import('@testing-library/react')
  await waitFor(
    () => {
      const loadingElements = document.querySelectorAll('[data-loading="true"]')
      if (loadingElements.length > 0) {
        throw new Error('Still loading')
      }
    },
    { timeout: 3000 }
  )
}

// Re-export everything from testing-library
export * from '@testing-library/react'
export { default as userEvent } from '@testing-library/user-event'
