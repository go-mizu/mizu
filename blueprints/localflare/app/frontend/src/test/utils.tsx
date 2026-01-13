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
    <MantineProvider theme={testTheme} defaultColorScheme="light">
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
      max_batch_size: 10,
      max_batch_timeout: 5000,
      message_ttl: 86400,
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

  // KV Namespaces
  kvNamespaces: () => [
    { id: 'kv-1', title: 'CONFIG', created_at: new Date().toISOString() },
    { id: 'kv-2', title: 'SESSIONS', created_at: new Date().toISOString() },
    { id: 'kv-3', title: 'CACHE', created_at: new Date().toISOString() },
  ],

  kvKeys: () => [
    { name: 'app:settings', expiration: 0, metadata: {} },
    { name: 'feature:flags', expiration: 0, metadata: {} },
    { name: 'session:user123', expiration: 3600, metadata: { user: 'user123' } },
  ],

  // R2 Buckets
  r2Buckets: () => [
    { id: 'bucket-1', name: 'assets', location: 'auto', created_at: new Date().toISOString() },
    { id: 'bucket-2', name: 'uploads', location: 'auto', created_at: new Date().toISOString() },
    { id: 'bucket-3', name: 'backups', location: 'auto', created_at: new Date().toISOString() },
  ],

  r2Objects: () => [
    { key: 'images/logo.png', size: 10240, last_modified: new Date().toISOString() },
    { key: 'images/hero.jpg', size: 102400, last_modified: new Date().toISOString() },
    { key: 'assets/style.css', size: 5120, last_modified: new Date().toISOString() },
  ],

  // D1 Databases
  d1Databases: () => [
    { id: 'd1-1', name: 'main', version: 'v1', num_tables: 5, file_size: 1024000, created_at: new Date().toISOString() },
    { id: 'd1-2', name: 'analytics', version: 'v1', num_tables: 3, file_size: 512000, created_at: new Date().toISOString() },
  ],

  d1Tables: () => [
    { name: 'users', row_count: 150 },
    { name: 'posts', row_count: 500 },
    { name: 'comments', row_count: 1200 },
  ],

  // Workers
  workers: () => [
    {
      id: 'w-1',
      name: 'api-router',
      script: 'export default { async fetch() {} }',
      routes: ['api.example.com/*'],
      bindings: [
        { name: 'KV', type: 'kv_namespace', namespace_id: 'kv-1' },
        { name: 'D1', type: 'd1', database_id: 'd1-1' },
      ],
      status: 'active' as const,
      created_at: new Date().toISOString(),
      modified_at: new Date().toISOString(),
    },
    {
      id: 'w-2',
      name: 'image-optimizer',
      script: 'export default { async fetch() {} }',
      routes: ['images.example.com/*'],
      bindings: [
        { name: 'R2', type: 'r2_bucket', bucket_name: 'assets' },
      ],
      status: 'active' as const,
      created_at: new Date().toISOString(),
      modified_at: new Date().toISOString(),
    },
  ],

  // Pages Projects
  pagesProjects: () => [
    {
      name: 'my-blog',
      subdomain: 'my-blog',
      created_at: new Date(Date.now() - 172800000).toISOString(),
      production_branch: 'main',
      latest_deployment: {
        id: 'deploy-1',
        url: 'https://my-blog.pages.dev',
        environment: 'production',
        deployment_trigger: { type: 'push', metadata: { branch: 'main', commit_hash: 'abc123' } },
        created_at: new Date(Date.now() - 3600000).toISOString(),
        status: 'success',
      },
      domains: ['blog.example.com'],
    },
    {
      name: 'docs-site',
      subdomain: 'docs-site',
      created_at: new Date(Date.now() - 604800000).toISOString(),
      production_branch: 'main',
      latest_deployment: {
        id: 'deploy-2',
        url: 'https://docs-site.pages.dev',
        environment: 'production',
        deployment_trigger: { type: 'push', metadata: { branch: 'main', commit_hash: 'def456' } },
        created_at: new Date(Date.now() - 86400000).toISOString(),
        status: 'success',
      },
      domains: ['docs.example.com'],
    },
  ],

  // Images
  cloudflareImages: () => [
    { id: 'img-1', filename: 'hero-banner.jpg', uploaded: new Date().toISOString(), variants: ['public', 'thumbnail'], meta: { width: 1920, height: 1080 } },
    { id: 'img-2', filename: 'logo.png', uploaded: new Date().toISOString(), variants: ['public', 'thumbnail'], meta: { width: 512, height: 512 } },
    { id: 'img-3', filename: 'product-1.webp', uploaded: new Date().toISOString(), variants: ['public', 'thumbnail', 'product'], meta: { width: 800, height: 800 } },
  ],

  imageVariants: () => [
    { id: 'public', name: 'public', options: { fit: 'scale-down', width: 1920, height: 1080 }, never_require_signed_urls: true },
    { id: 'thumbnail', name: 'thumbnail', options: { fit: 'cover', width: 150, height: 150 }, never_require_signed_urls: true },
  ],

  // Stream
  streamVideos: () => [
    { uid: 'vid-1', name: 'Product Demo', created: new Date().toISOString(), duration: 245, size: 45 * 1024 * 1024, status: { state: 'ready' }, playback: { hls: 'https://example.com/vid-1.m3u8' } },
    { uid: 'vid-2', name: 'Tutorial', created: new Date().toISOString(), duration: 1234, size: 234 * 1024 * 1024, status: { state: 'ready' }, playback: { hls: 'https://example.com/vid-2.m3u8' } },
  ],

  liveInputs: () => [
    { uid: 'live-1', name: 'Main Studio', created: new Date().toISOString(), status: 'connected', rtmps: { url: 'rtmps://live.cloudflare.com:443/live', streamKey: 'xxx' } },
    { uid: 'live-2', name: 'Backup', created: new Date().toISOString(), status: 'disconnected', rtmps: { url: 'rtmps://live.cloudflare.com:443/live', streamKey: 'yyy' } },
  ],

  // Observability
  logEntries: () => [
    { id: 'log-1', timestamp: new Date().toISOString(), level: 'info', message: 'Request processed', worker: 'api-router' },
    { id: 'log-2', timestamp: new Date().toISOString(), level: 'warn', message: 'Rate limit warning', worker: 'api-router' },
    { id: 'log-3', timestamp: new Date().toISOString(), level: 'error', message: 'Connection failed', worker: 'db-worker' },
  ],

  traces: () => [
    { id: 'trace-1', name: 'POST /api/users', start_time: new Date().toISOString(), duration_ms: 245, status: 'success', spans: [] },
    { id: 'trace-2', name: 'GET /api/products', start_time: new Date().toISOString(), duration_ms: 89, status: 'success', spans: [] },
  ],

  // Settings
  apiTokens: () => [
    { id: 'tok-1', name: 'CI/CD Token', permissions: ['workers:read', 'workers:write'], created_at: new Date().toISOString(), last_used: new Date().toISOString() },
    { id: 'tok-2', name: 'Monitoring', permissions: ['analytics:read'], created_at: new Date().toISOString(), last_used: new Date().toISOString() },
  ],

  accountMembers: () => [
    { id: 'mem-1', email: 'admin@example.com', name: 'Admin User', role: 'admin', status: 'active', joined: new Date().toISOString() },
    { id: 'mem-2', email: 'dev@example.com', name: 'Developer', role: 'developer', status: 'active', joined: new Date().toISOString() },
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
