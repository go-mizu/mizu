import { describe, it, expect, afterAll } from 'vitest'
import {
  renderWithProviders,
  screen,
  waitFor,
  testApi,
  isDurableObjectNamespace,
  generateTestName,
  userEvent,
} from '../../test/utils'
import { DurableObjects } from '../../pages/DurableObjects'
import type { DurableObjectNamespace } from '../../types'

describe('DurableObjects', () => {
  // Track created namespaces for cleanup
  const createdNamespaceIds: string[] = []

  afterAll(async () => {
    // Cleanup created namespaces
    for (const id of createdNamespaceIds) {
      try {
        await testApi.durableObjects.deleteNamespace(id)
      } catch {
        // Ignore cleanup errors
      }
    }
  })

  describe('API integration', () => {
    it('fetches namespaces list with correct structure', async () => {
      const response = await testApi.durableObjects.listNamespaces()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.namespaces).toBeInstanceOf(Array)

      const namespaces = response.result!.namespaces
      // Each namespace should have required fields
      for (const ns of namespaces) {
        expect(isDurableObjectNamespace(ns)).toBe(true)
        expect(typeof ns.id).toBe('string')
        expect(typeof ns.name).toBe('string')
        expect(typeof ns.class_name).toBe('string')
        expect(typeof ns.created_at).toBe('string')
      }
    })

    it('creates a new namespace with valid structure', async () => {
      const namespaceName = generateTestName('ns')
      const className = 'TestClass'
      const response = await testApi.durableObjects.createNamespace({
        name: namespaceName,
        class_name: className,
      })

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()

      const ns = response.result!
      expect(isDurableObjectNamespace(ns)).toBe(true)
      expect(ns.name).toBe(namespaceName)
      expect(ns.class_name).toBe(className)
      expect(typeof ns.id).toBe('string')
      expect(typeof ns.created_at).toBe('string')

      // Track for cleanup
      createdNamespaceIds.push(ns.id)
    })

    it('deletes a namespace successfully', async () => {
      // Create a namespace to delete
      const namespaceName = generateTestName('ns-delete')
      const createResponse = await testApi.durableObjects.createNamespace({
        name: namespaceName,
        class_name: 'DeleteClass',
      })

      expect(createResponse.success).toBe(true)
      const namespaceId = createResponse.result!.id

      // Delete the namespace
      const deleteResponse = await testApi.durableObjects.deleteNamespace(namespaceId)
      expect(deleteResponse.success).toBe(true)

      // Verify it's deleted - should not appear in list
      const listResponse = await testApi.durableObjects.listNamespaces()
      const namespaceNames = listResponse.result!.namespaces.map((n: DurableObjectNamespace) => n.name)
      expect(namespaceNames).not.toContain(namespaceName)
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page title', async () => {
      renderWithProviders(<DurableObjects />)
      expect(await screen.findByText('Durable Objects')).toBeInTheDocument()
    })

    it('displays namespaces from real API', async () => {
      // First create a namespace so we have something to display
      const namespaceName = generateTestName('ns-ui')
      const createResponse = await testApi.durableObjects.createNamespace({
        name: namespaceName,
        class_name: 'UIClass',
      })
      createdNamespaceIds.push(createResponse.result!.id)

      renderWithProviders(<DurableObjects />)

      await waitFor(() => {
        expect(screen.getByText(namespaceName)).toBeInTheDocument()
      }, { timeout: 10000 })
    }, 15000)

    it('has create namespace button', async () => {
      renderWithProviders(<DurableObjects />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      expect(screen.getByRole('button', { name: /Create Namespace/i })).toBeInTheDocument()
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<DurableObjects />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      const createButton = screen.getByRole('button', { name: /Create Namespace/i })
      await user.click(createButton)

      expect(await screen.findByText('Create Durable Object Namespace')).toBeInTheDocument()
    })
  })
})
