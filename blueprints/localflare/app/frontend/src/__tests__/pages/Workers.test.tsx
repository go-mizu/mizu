import { describe, it, expect, afterAll } from 'vitest'
import {
  renderWithProviders,
  screen,
  waitFor,
  testApi,
  isWorker,
  generateTestName,
} from '../../test/utils'
import { Workers } from '../../pages/Workers'
import type { Worker } from '../../types'

describe('Workers Page', () => {
  // Track created workers for cleanup
  const createdWorkerIds: string[] = []

  afterAll(async () => {
    // Cleanup created workers
    for (const id of createdWorkerIds) {
      try {
        await testApi.workers.delete(id)
      } catch {
        // Ignore cleanup errors
      }
    }
  })

  describe('API integration', () => {
    it('fetches workers list with correct structure', async () => {
      const response = await testApi.workers.list()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.workers).toBeInstanceOf(Array)

      const workers = response.result!.workers
      // Each worker should have required fields
      for (const worker of workers) {
        expect(isWorker(worker)).toBe(true)
        expect(typeof worker.id).toBe('string')
        expect(typeof worker.name).toBe('string')
        expect(typeof worker.created_at).toBe('string')
        expect(typeof worker.enabled).toBe('boolean')

        // Verify optional fields have correct types if present
        if (worker.routes) {
          expect(Array.isArray(worker.routes) || worker.routes === null).toBe(true)
        }
        if (worker.bindings) {
          // Bindings is an object (map), not an array
          expect(typeof worker.bindings).toBe('object')
        }
      }
    })

    it('creates a new worker with valid structure', async () => {
      const workerName = generateTestName('worker')
      const response = await testApi.workers.create({
        name: workerName,
        script: 'export default { fetch() { return new Response("Hello") } }',
      })

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()

      const worker = response.result!
      expect(isWorker(worker)).toBe(true)
      expect(worker.name).toBe(workerName)
      expect(typeof worker.id).toBe('string')
      expect(typeof worker.created_at).toBe('string')

      // Track for cleanup
      createdWorkerIds.push(worker.id)
    })

    it('deletes a worker successfully', async () => {
      // Create a worker to delete
      const workerName = generateTestName('worker-delete')
      const createResponse = await testApi.workers.create({
        name: workerName,
      })

      expect(createResponse.success).toBe(true)
      const workerId = createResponse.result!.id

      // Delete the worker
      const deleteResponse = await testApi.workers.delete(workerId)
      expect(deleteResponse.success).toBe(true)

      // Verify it's deleted - should not appear in list
      const listResponse = await testApi.workers.list()
      const workerNames = listResponse.result!.workers.map((w: Worker) => w.name)
      expect(workerNames).not.toContain(workerName)
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page title', async () => {
      renderWithProviders(<Workers />)
      expect(await screen.findByText('Workers')).toBeInTheDocument()
    })

    it('displays workers from real API', async () => {
      // First create a worker so we have something to display
      const workerName = generateTestName('worker-ui')
      const createResponse = await testApi.workers.create({
        name: workerName,
      })
      createdWorkerIds.push(createResponse.result!.id)

      renderWithProviders(<Workers />)

      await waitFor(() => {
        expect(screen.getByText(workerName)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('shows create button', async () => {
      renderWithProviders(<Workers />)

      await waitFor(() => {
        expect(screen.getByText(/Create Worker/i)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('shows worker status from real data', async () => {
      // Create a worker first
      const workerName = generateTestName('worker-status')
      const createResponse = await testApi.workers.create({
        name: workerName,
      })
      createdWorkerIds.push(createResponse.result!.id)

      renderWithProviders(<Workers />)

      await waitFor(() => {
        // Workers have status (active, inactive, error)
        expect(screen.getAllByText(/active|inactive|error/i).length).toBeGreaterThanOrEqual(1)
      }, { timeout: 5000 })
    })
  })

  describe('accessibility', () => {
    it('has proper heading hierarchy', async () => {
      renderWithProviders(<Workers />)

      await waitFor(() => {
        const heading = screen.getByRole('heading', { name: 'Workers' })
        expect(heading).toBeInTheDocument()
      }, { timeout: 5000 })
    })
  })
})
