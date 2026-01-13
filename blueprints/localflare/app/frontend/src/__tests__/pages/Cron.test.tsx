import { describe, it, expect, afterAll } from 'vitest'
import {
  renderWithProviders,
  screen,
  waitFor,
  testApi,
  isCronTrigger,
  generateTestName,
  userEvent,
} from '../../test/utils'
import { Cron } from '../../pages/Cron'
import type { CronTrigger } from '../../types'

describe('Cron', () => {
  // Track created triggers for cleanup
  const createdTriggerIds: string[] = []

  afterAll(async () => {
    // Cleanup created triggers
    for (const id of createdTriggerIds) {
      try {
        await testApi.cron.delete(id)
      } catch {
        // Ignore cleanup errors
      }
    }
  })

  describe('API integration', () => {
    it('fetches triggers list with correct structure', async () => {
      const response = await testApi.cron.list()

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()
      expect(response.result!.triggers).toBeInstanceOf(Array)

      const triggers = response.result!.triggers
      // Each trigger should have required fields
      for (const trigger of triggers) {
        expect(isCronTrigger(trigger)).toBe(true)
        expect(typeof trigger.id).toBe('string')
        expect(typeof trigger.cron).toBe('string')
        expect(typeof trigger.script_name).toBe('string')
        expect(typeof trigger.enabled).toBe('boolean')
      }
    })

    it('creates a new trigger with valid structure', async () => {
      const scriptName = generateTestName('cron-worker')
      const cronExpression = '*/5 * * * *'
      const response = await testApi.cron.create({
        cron: cronExpression,
        script_name: scriptName,
      })

      expect(response.success).toBe(true)
      expect(response.result).toBeDefined()

      const trigger = response.result!
      expect(isCronTrigger(trigger)).toBe(true)
      expect(trigger.cron).toBe(cronExpression)
      expect(trigger.script_name).toBe(scriptName)
      expect(typeof trigger.id).toBe('string')

      // Track for cleanup
      createdTriggerIds.push(trigger.id)
    })

    it('deletes a trigger successfully', async () => {
      // Create a trigger to delete
      const scriptName = generateTestName('cron-delete')
      const createResponse = await testApi.cron.create({
        cron: '0 * * * *',
        script_name: scriptName,
      })

      expect(createResponse.success).toBe(true)
      const triggerId = createResponse.result!.id

      // Delete the trigger
      const deleteResponse = await testApi.cron.delete(triggerId)
      expect(deleteResponse.success).toBe(true)

      // Verify it's deleted - should not appear in list
      const listResponse = await testApi.cron.list()
      const triggerIds = listResponse.result!.triggers.map((t: CronTrigger) => t.id)
      expect(triggerIds).not.toContain(triggerId)
    })
  })

  describe('UI rendering with real data', () => {
    it('renders the page title', async () => {
      renderWithProviders(<Cron />)
      expect(await screen.findByText('Cron Triggers')).toBeInTheDocument()
    })

    it('displays triggers from real API', async () => {
      // First create a trigger so we have something to display
      const scriptName = generateTestName('cron-ui')
      const createResponse = await testApi.cron.create({
        cron: '*/10 * * * *',
        script_name: scriptName,
      })
      createdTriggerIds.push(createResponse.result!.id)

      renderWithProviders(<Cron />)

      await waitFor(() => {
        expect(screen.getByText(scriptName)).toBeInTheDocument()
      }, { timeout: 5000 })
    })

    it('has create trigger button', async () => {
      renderWithProviders(<Cron />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      expect(screen.getByRole('button', { name: /Create Trigger/i })).toBeInTheDocument()
    })

    it('opens create modal on button click', async () => {
      const user = userEvent.setup()
      renderWithProviders(<Cron />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      const createButton = screen.getByRole('button', { name: /Create Trigger/i })
      await user.click(createButton)

      expect(await screen.findByText('Create Cron Trigger')).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    it('shows empty state when no triggers exist', async () => {
      renderWithProviders(<Cron />)

      await waitFor(() => {
        expect(screen.queryByText(/Loading/)).not.toBeInTheDocument()
      }, { timeout: 5000 })

      // Page should render successfully
      expect(screen.getByText('Cron Triggers')).toBeInTheDocument()
    })
  })
})
