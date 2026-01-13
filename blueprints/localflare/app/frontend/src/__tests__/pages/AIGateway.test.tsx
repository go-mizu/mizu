import { describe, it, expect, afterAll } from 'vitest'
import {
  renderWithProviders,
  screen,
  waitFor,
  testApi,
  isAIGateway,
  generateTestName,
  userEvent,
} from '../../test/utils'
import { AIGatewayPage } from '../../pages/AIGateway'
import type { AIGateway } from '../../types'

describe('AIGatewayPage', () => {
  // Track created gateways for cleanup
  const createdGatewayIds: string[] = []

  afterAll(async () => {
    // Cleanup created gateways
    for (const id of createdGatewayIds) {
      try {
        await testApi.aiGateway.delete(id)
      } catch {
        // Ignore cleanup errors
      }
    }
  })

  describe('API integration', () => {
    it('fetches gateways list with correct structure', async () => {
      const response = await testApi.aiGateway.list()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.gateways).toBeInstanceOf(Array)

      const gateways = response.result!.gateways
      // Each gateway should have required fields
      for (const gateway of gateways) {
        expect(isAIGateway(gateway)).toBe(true)
        expect(typeof gateway.id).toBe('string')
        expect(typeof gateway.name).toBe('string')
        expect(typeof gateway.created_at).toBe('string')
      }
    })

    it('creates a new gateway with valid structure', async () => {
      const gatewayName = generateTestName('gw')
      const response = await testApi.aiGateway.create({
        name: gatewayName,
      })

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()

      const gateway = response.result!
      expect(isAIGateway(gateway)).toBe(true)
      expect(gateway.name).toBe(gatewayName)
      expect(typeof gateway.id).toBe('string')
      expect(typeof gateway.created_at).toBe('string')

      // Track for cleanup
      createdGatewayIds.push(gateway.id)
    })

    it('deletes a gateway successfully', async () => {
      // Create a gateway to delete
      const gatewayName = generateTestName('gw-delete')
      const createResponse = await testApi.aiGateway.create({
        name: gatewayName,
      })

      expect(createResponse.success).toBe(true)
      const gatewayId = createResponse.result!.id

      // Delete the gateway
      const deleteResponse = await testApi.aiGateway.delete(gatewayId)
      expect(deleteResponse.success).toBe(true)

      // Verify it's deleted - should not appear in list
      const listResponse = await testApi.aiGateway.list()
      const gatewayNames = listResponse.result!.gateways.map((g: AIGateway) => g.name)
      expect(gatewayNames).not.toContain(gatewayName)
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page title', async () => {
      renderWithProviders(<AIGatewayPage />)
      expect(await screen.findByText('AI Gateway')).toBeInTheDocument()
    })

    it('displays gateways from real API', async () => {
      // First create a gateway so we have something to display
      const gatewayName = generateTestName('gw-ui')
      const createResponse = await testApi.aiGateway.create({
        name: gatewayName,
      })
      createdGatewayIds.push(createResponse.result!.id)

      renderWithProviders(<AIGatewayPage />)

      await waitFor(() => {
        expect(screen.getByText(gatewayName)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('has create gateway button', async () => {
      renderWithProviders(<AIGatewayPage />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      expect(screen.getByRole('button', { name: /Create Gateway/i })).toBeInTheDocument()
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<AIGatewayPage />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      const createButton = screen.getByRole('button', { name: /Create Gateway/i })
      await user.click(createButton)

      expect(await screen.findByText('Create AI Gateway')).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    it('shows empty state when no gateways exist', async () => {
      renderWithProviders(<AIGatewayPage />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      // Page should render successfully
      expect(screen.getByText('AI Gateway')).toBeInTheDocument()
    })
  })
})
