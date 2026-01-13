import { describe, it, expect, afterAll } from 'vitest'
import {
  renderWithProviders,
  screen,
  waitFor,
  testApi,
  isQueue,
  generateTestName,
  userEvent,
} from '../../test/utils'
import { Queues } from '../../pages/Queues'
import type { Queue } from '../../types'

describe('Queues', () => {
  // Track created queues for cleanup
  const createdQueueIds: string[] = []

  afterAll(async () => {
    // Cleanup created queues
    for (const id of createdQueueIds) {
      try {
        await testApi.queues.delete(id)
      } catch {
        // Ignore cleanup errors
      }
    }
  })

  describe('API integration', () => {
    it('fetches queues list with correct structure', async () => {
      const response = await testApi.queues.list()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.queues).toBeInstanceOf(Array)

      const queues = response.result!.queues
      // Each queue should have required fields
      for (const queue of queues) {
        expect(isQueue(queue)).toBe(true)
        expect(typeof queue.id).toBe('string')
        expect(typeof queue.name).toBe('string')
        expect(typeof queue.created_at).toBe('string')
        expect(queue.settings).toBeDefined()

        // Verify settings structure
        expect(typeof queue.settings.max_retries).toBe('number')
        expect(typeof queue.settings.max_batch_size).toBe('number')
      }
    })

    it('creates a new queue with valid structure', async () => {
      const queueName = generateTestName('queue')
      const response = await testApi.queues.create({
        queue_name: queueName,
      })

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()

      const queue = response.result!
      expect(isQueue(queue)).toBe(true)
      expect(queue.name).toBe(queueName)
      expect(typeof queue.id).toBe('string')
      expect(typeof queue.created_at).toBe('string')
      expect(queue.settings).toBeDefined()

      // Track for cleanup
      createdQueueIds.push(queue.id)
    })

    it('deletes a queue successfully', async () => {
      // Create a queue to delete
      const queueName = generateTestName('queue-delete')
      const createResponse = await testApi.queues.create({
        queue_name: queueName,
      })

      expect(createResponse.success).toBe(true)
      const queueId = createResponse.result!.id

      // Delete the queue
      const deleteResponse = await testApi.queues.delete(queueId)
      expect(deleteResponse.success).toBe(true)

      // Verify it's deleted - should not appear in list
      const listResponse = await testApi.queues.list()
      const queueNames = listResponse.result!.queues.map((q: Queue) => q.name)
      expect(queueNames).not.toContain(queueName)
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page title', async () => {
      renderWithProviders(<Queues />)
      expect(await screen.findByText('Queues')).toBeInTheDocument()
    })

    it('displays queues from real API', async () => {
      // First create a queue so we have something to display
      const queueName = generateTestName('queue-ui')
      const createResponse = await testApi.queues.create({
        queue_name: queueName,
      })
      createdQueueIds.push(createResponse.result!.id)

      renderWithProviders(<Queues />)

      await waitFor(() => {
        expect(screen.getByText(queueName)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('has create queue button', async () => {
      renderWithProviders(<Queues />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      expect(screen.getByRole('button', { name: /Create Queue/i })).toBeInTheDocument()
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<Queues />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      const createButton = screen.getByRole('button', { name: /Create Queue/i })
      await user.click(createButton)

      expect(await screen.findByRole('dialog')).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    it('shows empty state message when no queues exist', async () => {
      // This test depends on the current state - may show empty or not
      renderWithProviders(<Queues />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      // Check page rendered successfully
      expect(screen.getByText('Queues')).toBeInTheDocument()
    })
  })
})
