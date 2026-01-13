import { describe, it, expect, afterAll } from 'vitest'
import {
  renderWithProviders,
  screen,
  waitFor,
  testApi,
  isKVNamespace,
  isR2Bucket,
  isD1Database,
  generateTestName,
} from '../../test/utils'
import { KV } from '../../pages/KV'
import { R2 } from '../../pages/R2'
import { D1 } from '../../pages/D1'
import type { KVNamespace, R2Bucket, D1Database } from '../../types'

describe('KV Page', () => {
  // Track created namespaces for cleanup
  const createdNamespaceIds: string[] = []

  afterAll(async () => {
    for (const id of createdNamespaceIds) {
      try {
        await testApi.kv.deleteNamespace(id)
      } catch {
        // Ignore cleanup errors
      }
    }
  })

  describe('API integration', () => {
    it('fetches namespaces list with correct structure', async () => {
      const response = await testApi.kv.listNamespaces()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.namespaces).toBeInstanceOf(Array)

      const namespaces = response.result!.namespaces
      for (const ns of namespaces) {
        expect(isKVNamespace(ns)).toBe(true)
        expect(typeof ns.id).toBe('string')
        expect(typeof ns.title).toBe('string')
        expect(typeof ns.created_at).toBe('string')
      }
    })

    it('creates a new namespace with valid structure', async () => {
      const title = generateTestName('KV')
      const response = await testApi.kv.createNamespace({ title })

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()

      const ns = response.result!
      expect(isKVNamespace(ns)).toBe(true)
      expect(ns.title).toBe(title)
      expect(typeof ns.id).toBe('string')

      createdNamespaceIds.push(ns.id)
    })

    it('deletes a namespace successfully', async () => {
      const title = generateTestName('KV-delete')
      const createResponse = await testApi.kv.createNamespace({ title })
      expect(createResponse.success).toBe(true)
      const namespaceId = createResponse.result!.id

      const deleteResponse = await testApi.kv.deleteNamespace(namespaceId)
      expect(deleteResponse.success).toBe(true)

      const listResponse = await testApi.kv.listNamespaces()
      const ids = listResponse.result!.namespaces.map((n: KVNamespace) => n.id)
      expect(ids).not.toContain(namespaceId)
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page title', async () => {
      renderWithProviders(<KV />)
      expect(await screen.findByText('Workers KV')).toBeInTheDocument()
    })

    it('displays namespaces from real API', async () => {
      const title = generateTestName('KV-ui')
      const createResponse = await testApi.kv.createNamespace({ title })
      createdNamespaceIds.push(createResponse.result!.id)

      renderWithProviders(<KV />)

      await waitFor(() => {
        expect(screen.getByText(title)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('shows create button', async () => {
      renderWithProviders(<KV />)

      await waitFor(() => {
        expect(screen.getByText(/Create Namespace/i)).toBeInTheDocument()
      }, { timeout: 5000 })
    })
  })
})

describe('R2 Page', () => {
  // Track created buckets for cleanup
  const createdBucketNames: string[] = []

  afterAll(async () => {
    for (const name of createdBucketNames) {
      try {
        await testApi.r2.deleteBucket(name)
      } catch {
        // Ignore cleanup errors
      }
    }
  })

  describe('API integration', () => {
    it('fetches buckets list with correct structure', async () => {
      const response = await testApi.r2.listBuckets()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.buckets).toBeInstanceOf(Array)

      const buckets = response.result!.buckets
      for (const bucket of buckets) {
        expect(isR2Bucket(bucket)).toBe(true)
        expect(typeof bucket.name).toBe('string')
        expect(typeof bucket.created_at).toBe('string')
      }
    })

    it('creates a new bucket with valid structure', async () => {
      const name = generateTestName('r2').toLowerCase().replace(/[^a-z0-9-]/g, '-')
      const response = await testApi.r2.createBucket({ name })

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()

      const bucket = response.result!
      expect(isR2Bucket(bucket)).toBe(true)
      expect(bucket.name).toBe(name)

      createdBucketNames.push(name)
    })

    it('deletes a bucket successfully', async () => {
      const name = generateTestName('r2-delete').toLowerCase().replace(/[^a-z0-9-]/g, '-')
      const createResponse = await testApi.r2.createBucket({ name })
      expect(createResponse.success).toBe(true)

      const deleteResponse = await testApi.r2.deleteBucket(name)
      expect(deleteResponse.success).toBe(true)

      const listResponse = await testApi.r2.listBuckets()
      const names = listResponse.result!.buckets.map((b: R2Bucket) => b.name)
      expect(names).not.toContain(name)
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page title', async () => {
      renderWithProviders(<R2 />)
      expect(await screen.findByText('R2 Object Storage')).toBeInTheDocument()
    })

    it('displays buckets from real API', async () => {
      const name = generateTestName('r2-ui').toLowerCase().replace(/[^a-z0-9-]/g, '-')
      await testApi.r2.createBucket({ name })
      createdBucketNames.push(name)

      renderWithProviders(<R2 />)

      await waitFor(() => {
        expect(screen.getByText(name)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('shows create button', async () => {
      renderWithProviders(<R2 />)

      await waitFor(() => {
        expect(screen.getByText(/Create Bucket/i)).toBeInTheDocument()
      }, { timeout: 5000 })
    })
  })
})

describe('D1 Page', () => {
  // Track created databases for cleanup
  const createdDatabaseIds: string[] = []

  afterAll(async () => {
    for (const id of createdDatabaseIds) {
      try {
        await testApi.d1.deleteDatabase(id)
      } catch {
        // Ignore cleanup errors
      }
    }
  })

  describe('API integration', () => {
    it('fetches databases list with correct structure', async () => {
      const response = await testApi.d1.listDatabases()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.databases).toBeInstanceOf(Array)

      const databases = response.result!.databases
      for (const db of databases) {
        expect(isD1Database(db)).toBe(true)
        expect(typeof db.name).toBe('string')
        expect(typeof db.created_at).toBe('string')
        // uuid can be either uuid or id
        expect(typeof (db.uuid || (db as any).id)).toBe('string')
      }
    })

    it('creates a new database with valid structure', async () => {
      const name = generateTestName('d1')
      const response = await testApi.d1.createDatabase({ name })

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()

      const db = response.result!
      expect(isD1Database(db)).toBe(true)
      expect(db.name).toBe(name)

      const dbId = db.uuid || (db as any).id
      createdDatabaseIds.push(dbId)
    })

    it('deletes a database successfully', async () => {
      const name = generateTestName('d1-delete')
      const createResponse = await testApi.d1.createDatabase({ name })
      expect(createResponse.success).toBe(true)
      const dbId = createResponse.result!.uuid || (createResponse.result as any).id

      const deleteResponse = await testApi.d1.deleteDatabase(dbId)
      expect(deleteResponse.success).toBe(true)

      const listResponse = await testApi.d1.listDatabases()
      const ids = listResponse.result!.databases.map((d: D1Database) => d.uuid || (d as any).id)
      expect(ids).not.toContain(dbId)
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page title', async () => {
      renderWithProviders(<D1 />)
      expect(await screen.findByText('D1 Database')).toBeInTheDocument()
    })

    it('displays databases from real API', async () => {
      const name = generateTestName('d1-ui')
      const createResponse = await testApi.d1.createDatabase({ name })
      const dbId = createResponse.result!.uuid || (createResponse.result as any).id
      createdDatabaseIds.push(dbId)

      renderWithProviders(<D1 />)

      await waitFor(() => {
        expect(screen.getByText(name)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('shows create button', async () => {
      renderWithProviders(<D1 />)

      await waitFor(() => {
        expect(screen.getByText(/Create Database/i)).toBeInTheDocument()
      }, { timeout: 5000 })
    })
  })
})
