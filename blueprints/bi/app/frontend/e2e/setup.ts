import { test as base, expect } from '@playwright/test'

/**
 * E2E Test Setup and Utilities
 *
 * This file provides common utilities and fixtures for e2e tests.
 */

const API_BASE = process.env.E2E_API_URL || 'http://localhost:3000/api'

// Extended test interface with custom fixtures
interface TestFixtures {
  apiUrl: string
  createDataSource: (name: string, engine?: string) => Promise<string>
  deleteDataSource: (id: string) => Promise<void>
}

export const test = base.extend<TestFixtures>({
  apiUrl: API_BASE,

  createDataSource: async ({ request }, use) => {
    const createdIds: string[] = []

    const createDataSource = async (name: string, engine = 'sqlite') => {
      const response = await request.post(`${API_BASE}/datasources`, {
        data: {
          name,
          engine,
          database: engine === 'sqlite' ? ':memory:' : 'testdb',
          host: engine !== 'sqlite' ? 'localhost' : undefined,
          port: engine === 'postgres' ? 5432 : engine === 'mysql' ? 3306 : undefined,
        },
      })
      expect(response.ok()).toBeTruthy()
      const ds = await response.json()
      createdIds.push(ds.id)
      return ds.id
    }

    await use(createDataSource)

    // Cleanup: delete all created data sources
    for (const id of createdIds) {
      try {
        await request.delete(`${API_BASE}/datasources/${id}`)
      } catch {
        // Ignore cleanup errors
      }
    }
  },

  deleteDataSource: async ({ request }, use) => {
    const deleteDataSource = async (id: string) => {
      const response = await request.delete(`${API_BASE}/datasources/${id}`)
      expect(response.ok()).toBeTruthy()
    }

    await use(deleteDataSource)
  },
})

export { expect }

/**
 * Common semantic type options for testing
 */
export const SEMANTIC_TYPES = {
  keys: ['type/PK', 'type/FK'],
  numbers: ['type/Price', 'type/Currency', 'type/Score', 'type/Percentage', 'type/Quantity'],
  text: ['type/Name', 'type/Title', 'type/Description', 'type/Category', 'type/URL', 'type/Email', 'type/Phone'],
  dates: ['type/CreationDate', 'type/UpdateDate', 'type/JoinDate', 'type/Birthdate'],
  geo: ['type/Latitude', 'type/Longitude', 'type/City', 'type/State', 'type/Country', 'type/ZipCode', 'type/Address'],
}

/**
 * SSL modes for testing
 */
export const SSL_MODES = ['disable', 'allow', 'prefer', 'require', 'verify-ca', 'verify-full']

/**
 * Engine types for testing
 */
export const ENGINE_TYPES = ['sqlite', 'postgres', 'mysql']

/**
 * Visibility options for testing
 */
export const VISIBILITY_OPTIONS = ['everywhere', 'detail_only', 'hidden']

/**
 * Filter widget types for testing
 */
export const FILTER_WIDGET_TYPES = ['search', 'dropdown', 'input', 'none']

/**
 * Generate a unique name for test resources
 */
export function uniqueName(prefix: string): string {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`
}

/**
 * Wait for a condition to be true
 */
export async function waitFor(
  condition: () => Promise<boolean>,
  timeout = 10000,
  interval = 100
): Promise<void> {
  const start = Date.now()
  while (Date.now() - start < timeout) {
    if (await condition()) {
      return
    }
    await new Promise((resolve) => setTimeout(resolve, interval))
  }
  throw new Error('Timeout waiting for condition')
}

/**
 * Retry a function with exponential backoff
 */
export async function retry<T>(
  fn: () => Promise<T>,
  maxRetries = 3,
  baseDelayMs = 100
): Promise<T> {
  let lastError: Error | undefined
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn()
    } catch (error) {
      lastError = error as Error
      await new Promise((resolve) => setTimeout(resolve, baseDelayMs * Math.pow(2, i)))
    }
  }
  throw lastError
}
