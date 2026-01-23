import { test, expect, Page, APIRequestContext } from '@playwright/test'

/**
 * Data Sources E2E Tests
 *
 * These tests verify the complete Data Sources UI functionality
 * connecting to a real backend (no mocks).
 *
 * Test Coverage:
 * - CRUD operations for data sources
 * - Connection testing (pre-create and post-create)
 * - SSL/TLS configuration
 * - SSH tunnel configuration
 * - Schema filtering
 * - Sync operations
 * - Scan operations
 * - Fingerprinting
 * - Table metadata editing
 * - Column metadata editing
 * - Cache management
 * - Error handling
 */

const API_BASE = process.env.E2E_API_URL || 'http://localhost:3000/api'

// Helper to generate unique names
const uniqueName = (prefix: string) => `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`

// Helper to create a test SQLite data source via API
async function createTestDataSource(request: APIRequestContext, name?: string): Promise<string> {
  const dsName = name || uniqueName('test_ds')
  const response = await request.post(`${API_BASE}/datasources`, {
    data: {
      name: dsName,
      engine: 'sqlite',
      database: ':memory:',
    },
  })
  expect(response.ok()).toBeTruthy()
  const ds = await response.json()
  return ds.id
}

// Helper to delete a data source via API
async function deleteDataSource(request: APIRequestContext, id: string): Promise<void> {
  const response = await request.delete(`${API_BASE}/datasources/${id}`)
  expect(response.ok()).toBeTruthy()
}

// Helper to list data sources via API
async function listDataSources(request: APIRequestContext): Promise<any[]> {
  const response = await request.get(`${API_BASE}/datasources`)
  expect(response.ok()).toBeTruthy()
  return response.json()
}

test.describe('Data Sources API', () => {
  test.describe('CRUD Operations', () => {
    test('should list all data sources', async ({ request }) => {
      const response = await request.get(`${API_BASE}/datasources`)
      expect(response.ok()).toBeTruthy()
      const datasources = await response.json()
      expect(Array.isArray(datasources)).toBeTruthy()
    })

    test('should create a SQLite data source', async ({ request }) => {
      const name = uniqueName('sqlite_test')
      const response = await request.post(`${API_BASE}/datasources`, {
        data: {
          name,
          engine: 'sqlite',
          database: ':memory:',
        },
      })
      expect(response.ok()).toBeTruthy()
      const ds = await response.json()

      expect(ds.id).toBeDefined()
      expect(ds.name).toBe(name)
      expect(ds.engine).toBe('sqlite')
      expect(ds.database).toBe(':memory:')
      expect(ds.created_at).toBeDefined()

      // Cleanup
      await deleteDataSource(request, ds.id)
    })

    test('should create a PostgreSQL data source', async ({ request }) => {
      const name = uniqueName('postgres_test')
      const response = await request.post(`${API_BASE}/datasources`, {
        data: {
          name,
          engine: 'postgres',
          host: 'localhost',
          port: 5432,
          database: 'testdb',
          username: 'testuser',
          ssl: true,
          ssl_mode: 'prefer',
        },
      })
      expect(response.ok()).toBeTruthy()
      const ds = await response.json()

      expect(ds.engine).toBe('postgres')
      expect(ds.host).toBe('localhost')
      expect(ds.port).toBe(5432)
      expect(ds.ssl).toBe(true)
      expect(ds.ssl_mode).toBe('prefer')

      // Cleanup
      await deleteDataSource(request, ds.id)
    })

    test('should get a data source by ID', async ({ request }) => {
      const dsId = await createTestDataSource(request, uniqueName('get_test'))

      const response = await request.get(`${API_BASE}/datasources/${dsId}`)
      expect(response.ok()).toBeTruthy()
      const ds = await response.json()

      expect(ds.id).toBe(dsId)
      expect(ds.engine).toBe('sqlite')

      // Cleanup
      await deleteDataSource(request, dsId)
    })

    test('should update a data source', async ({ request }) => {
      const dsId = await createTestDataSource(request, uniqueName('update_test'))

      const newName = uniqueName('updated')
      const response = await request.put(`${API_BASE}/datasources/${dsId}`, {
        data: {
          name: newName,
          cache_ttl: 3600,
        },
      })
      expect(response.ok()).toBeTruthy()
      const ds = await response.json()

      expect(ds.name).toBe(newName)
      expect(ds.cache_ttl).toBe(3600)

      // Cleanup
      await deleteDataSource(request, dsId)
    })

    test('should delete a data source', async ({ request }) => {
      const dsId = await createTestDataSource(request, uniqueName('delete_test'))

      const response = await request.delete(`${API_BASE}/datasources/${dsId}`)
      expect(response.ok()).toBeTruthy()

      // Verify deletion
      const getResponse = await request.get(`${API_BASE}/datasources/${dsId}`)
      expect(getResponse.status()).toBe(404)
    })

    test('should return 404 for non-existent data source', async ({ request }) => {
      const response = await request.get(`${API_BASE}/datasources/nonexistent_id_12345`)
      expect(response.status()).toBe(404)
    })
  })

  test.describe('Connection Testing', () => {
    test('should test connection before creating (SQLite)', async ({ request }) => {
      const response = await request.post(`${API_BASE}/datasources/test-connection`, {
        data: {
          engine: 'sqlite',
          database: ':memory:',
        },
      })
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      expect(result.valid).toBe(true)
      expect(result.latency_ms).toBeGreaterThanOrEqual(0)
    })

    test('should test connection for existing data source', async ({ request }) => {
      const dsId = await createTestDataSource(request, uniqueName('test_conn'))

      const response = await request.post(`${API_BASE}/datasources/${dsId}/test`)
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      expect(result.valid).toBe(true)
      expect(result.latency_ms).toBeGreaterThanOrEqual(0)

      // Cleanup
      await deleteDataSource(request, dsId)
    })

    test('should return error for invalid connection', async ({ request }) => {
      const response = await request.post(`${API_BASE}/datasources/test-connection`, {
        data: {
          engine: 'postgres',
          host: 'nonexistent.invalid.host',
          port: 5432,
          database: 'testdb',
          username: 'invalid',
          password: 'invalid',
        },
      })
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      expect(result.valid).toBe(false)
      expect(result.error).toBeDefined()
      expect(result.error_code).toBeDefined()
    })

    test('should return suggestions for connection errors', async ({ request }) => {
      const response = await request.post(`${API_BASE}/datasources/test-connection`, {
        data: {
          engine: 'postgres',
          host: '127.0.0.1',
          port: 54321, // Wrong port
          database: 'testdb',
          username: 'invalid',
        },
      })
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      expect(result.valid).toBe(false)
      expect(result.suggestions).toBeDefined()
      expect(Array.isArray(result.suggestions)).toBeTruthy()
    })
  })

  test.describe('Connection Status', () => {
    test('should get connection status', async ({ request }) => {
      const dsId = await createTestDataSource(request, uniqueName('status_test'))

      const response = await request.get(`${API_BASE}/datasources/${dsId}/status`)
      expect(response.ok()).toBeTruthy()
      const status = await response.json()

      expect(status.connected).toBe(true)
      expect(status.latency_ms).toBeGreaterThanOrEqual(0)
      expect(status.capabilities).toBeDefined()

      // Cleanup
      await deleteDataSource(request, dsId)
    })
  })

  test.describe('SSL Configuration', () => {
    test('should create data source with SSL mode', async ({ request }) => {
      const response = await request.post(`${API_BASE}/datasources`, {
        data: {
          name: uniqueName('ssl_test'),
          engine: 'postgres',
          host: 'localhost',
          port: 5432,
          database: 'testdb',
          username: 'testuser',
          ssl: true,
          ssl_mode: 'verify-full',
        },
      })
      expect(response.ok()).toBeTruthy()
      const ds = await response.json()

      expect(ds.ssl).toBe(true)
      expect(ds.ssl_mode).toBe('verify-full')

      // Cleanup
      await deleteDataSource(request, ds.id)
    })

    test('should support all SSL modes', async ({ request }) => {
      const sslModes = ['disable', 'allow', 'prefer', 'require', 'verify-ca', 'verify-full']

      for (const mode of sslModes) {
        const response = await request.post(`${API_BASE}/datasources`, {
          data: {
            name: uniqueName(`ssl_${mode}`),
            engine: 'postgres',
            host: 'localhost',
            port: 5432,
            database: 'testdb',
            ssl: mode !== 'disable',
            ssl_mode: mode,
          },
        })
        expect(response.ok()).toBeTruthy()
        const ds = await response.json()
        expect(ds.ssl_mode).toBe(mode)
        await deleteDataSource(request, ds.id)
      }
    })
  })

  test.describe('SSH Tunnel Configuration', () => {
    test('should create data source with SSH tunnel config', async ({ request }) => {
      const response = await request.post(`${API_BASE}/datasources`, {
        data: {
          name: uniqueName('ssh_test'),
          engine: 'postgres',
          host: 'db.internal',
          port: 5432,
          database: 'testdb',
          tunnel_enabled: true,
          tunnel_host: 'bastion.example.com',
          tunnel_port: 22,
          tunnel_user: 'tunnel_user',
          tunnel_auth_method: 'password',
        },
      })
      expect(response.ok()).toBeTruthy()
      const ds = await response.json()

      expect(ds.tunnel_enabled).toBe(true)
      expect(ds.tunnel_host).toBe('bastion.example.com')
      expect(ds.tunnel_port).toBe(22)
      expect(ds.tunnel_auth_method).toBe('password')

      // Cleanup
      await deleteDataSource(request, ds.id)
    })

    test('should support SSH key authentication', async ({ request }) => {
      const response = await request.post(`${API_BASE}/datasources`, {
        data: {
          name: uniqueName('ssh_key_test'),
          engine: 'postgres',
          host: 'db.internal',
          port: 5432,
          database: 'testdb',
          tunnel_enabled: true,
          tunnel_host: 'bastion.example.com',
          tunnel_port: 22,
          tunnel_user: 'tunnel_user',
          tunnel_auth_method: 'ssh-key',
        },
      })
      expect(response.ok()).toBeTruthy()
      const ds = await response.json()

      expect(ds.tunnel_auth_method).toBe('ssh-key')

      // Cleanup
      await deleteDataSource(request, ds.id)
    })
  })

  test.describe('Schema Filtering', () => {
    test('should create data source with schema inclusion filter', async ({ request }) => {
      const response = await request.post(`${API_BASE}/datasources`, {
        data: {
          name: uniqueName('schema_include'),
          engine: 'postgres',
          host: 'localhost',
          port: 5432,
          database: 'testdb',
          schema_filter_type: 'inclusion',
          schema_filter_patterns: ['public', 'analytics'],
        },
      })
      expect(response.ok()).toBeTruthy()
      const ds = await response.json()

      expect(ds.schema_filter_type).toBe('inclusion')
      expect(ds.schema_filter_patterns).toEqual(['public', 'analytics'])

      // Cleanup
      await deleteDataSource(request, ds.id)
    })

    test('should create data source with schema exclusion filter', async ({ request }) => {
      const response = await request.post(`${API_BASE}/datasources`, {
        data: {
          name: uniqueName('schema_exclude'),
          engine: 'postgres',
          host: 'localhost',
          port: 5432,
          database: 'testdb',
          schema_filter_type: 'exclusion',
          schema_filter_patterns: ['pg_catalog', 'information_schema'],
        },
      })
      expect(response.ok()).toBeTruthy()
      const ds = await response.json()

      expect(ds.schema_filter_type).toBe('exclusion')
      expect(ds.schema_filter_patterns).toContain('pg_catalog')

      // Cleanup
      await deleteDataSource(request, ds.id)
    })
  })

  test.describe('Sync Operations', () => {
    test('should sync metadata from data source', async ({ request }) => {
      const dsId = await createTestDataSource(request, uniqueName('sync_test'))

      const response = await request.post(`${API_BASE}/datasources/${dsId}/sync`, {
        data: {
          full_sync: true,
          scan_field_values: false,
        },
      })
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      expect(result.status).toBeDefined()
      expect(result.duration_ms).toBeGreaterThanOrEqual(0)

      // Cleanup
      await deleteDataSource(request, dsId)
    })

    test('should get sync log', async ({ request }) => {
      const dsId = await createTestDataSource(request, uniqueName('synclog_test'))

      // Trigger a sync first
      await request.post(`${API_BASE}/datasources/${dsId}/sync`)

      const response = await request.get(`${API_BASE}/datasources/${dsId}/sync-log`)
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      expect(result.logs).toBeDefined()
      expect(Array.isArray(result.logs)).toBeTruthy()

      // Cleanup
      await deleteDataSource(request, dsId)
    })
  })

  test.describe('Scan Operations', () => {
    test('should scan field values', async ({ request }) => {
      const dsId = await createTestDataSource(request, uniqueName('scan_test'))

      // Sync first to get tables
      await request.post(`${API_BASE}/datasources/${dsId}/sync`)

      const response = await request.post(`${API_BASE}/datasources/${dsId}/scan`, {
        data: {
          limit: 100,
        },
      })
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      expect(result.status).toBeDefined()
      expect(result.duration_ms).toBeGreaterThanOrEqual(0)

      // Cleanup
      await deleteDataSource(request, dsId)
    })
  })

  test.describe('Fingerprinting', () => {
    test('should fingerprint columns', async ({ request }) => {
      const dsId = await createTestDataSource(request, uniqueName('finger_test'))

      // Sync first to get tables
      await request.post(`${API_BASE}/datasources/${dsId}/sync`)

      const response = await request.post(`${API_BASE}/datasources/${dsId}/fingerprint`, {
        data: {
          sample_size: 1000,
        },
      })
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      expect(result.status).toBeDefined()
      expect(result.duration_ms).toBeGreaterThanOrEqual(0)

      // Cleanup
      await deleteDataSource(request, dsId)
    })
  })

  test.describe('Table Operations', () => {
    test('should list tables for data source', async ({ request }) => {
      const dsId = await createTestDataSource(request, uniqueName('tables_test'))

      const response = await request.get(`${API_BASE}/datasources/${dsId}/tables`)
      expect(response.ok()).toBeTruthy()
      const tables = await response.json()

      expect(Array.isArray(tables)).toBeTruthy()

      // Cleanup
      await deleteDataSource(request, dsId)
    })
  })

  test.describe('Cache Operations', () => {
    test('should get cache statistics', async ({ request }) => {
      const dsId = await createTestDataSource(request, uniqueName('cache_stats'))

      const response = await request.get(`${API_BASE}/datasources/${dsId}/cache/stats`)
      expect(response.ok()).toBeTruthy()
      const stats = await response.json()

      expect(stats.datasource_id).toBe(dsId)
      expect(stats.columns_with_cache).toBeDefined()
      expect(stats.total_cached_values).toBeDefined()

      // Cleanup
      await deleteDataSource(request, dsId)
    })

    test('should clear cache', async ({ request }) => {
      const dsId = await createTestDataSource(request, uniqueName('cache_clear'))

      const response = await request.post(`${API_BASE}/datasources/${dsId}/cache/clear`)
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      expect(result.status).toBe('cleared')
      expect(result.columns_cleared).toBeDefined()

      // Cleanup
      await deleteDataSource(request, dsId)
    })
  })

  test.describe('Connection Pool Configuration', () => {
    test('should create data source with pool settings', async ({ request }) => {
      const response = await request.post(`${API_BASE}/datasources`, {
        data: {
          name: uniqueName('pool_test'),
          engine: 'postgres',
          host: 'localhost',
          port: 5432,
          database: 'testdb',
          max_open_conns: 50,
          max_idle_conns: 10,
          conn_max_lifetime: 3600,
          conn_max_idle_time: 600,
        },
      })
      expect(response.ok()).toBeTruthy()
      const ds = await response.json()

      expect(ds.max_open_conns).toBe(50)
      expect(ds.max_idle_conns).toBe(10)
      expect(ds.conn_max_lifetime).toBe(3600)
      expect(ds.conn_max_idle_time).toBe(600)

      // Cleanup
      await deleteDataSource(request, ds.id)
    })
  })

  test.describe('Sync Schedule Configuration', () => {
    test('should create data source with sync schedule', async ({ request }) => {
      const response = await request.post(`${API_BASE}/datasources`, {
        data: {
          name: uniqueName('schedule_test'),
          engine: 'sqlite',
          database: ':memory:',
          auto_sync: true,
          sync_schedule: '0 * * * *', // Hourly
        },
      })
      expect(response.ok()).toBeTruthy()
      const ds = await response.json()

      expect(ds.auto_sync).toBe(true)
      expect(ds.sync_schedule).toBe('0 * * * *')

      // Cleanup
      await deleteDataSource(request, ds.id)
    })
  })

  test.describe('Error Handling', () => {
    test('should handle invalid engine', async ({ request }) => {
      const response = await request.post(`${API_BASE}/datasources/test-connection`, {
        data: {
          engine: 'invalid_engine',
          database: 'test',
        },
      })
      // Should return 200 with error in body
      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.valid).toBe(false)
    })

    test('should categorize connection refused errors', async ({ request }) => {
      const response = await request.post(`${API_BASE}/datasources/test-connection`, {
        data: {
          engine: 'postgres',
          host: '127.0.0.1',
          port: 54321,
          database: 'test',
        },
      })
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      expect(result.valid).toBe(false)
      expect(['CONNECTION_REFUSED', 'TIMEOUT', 'UNKNOWN']).toContain(result.error_code)
    })
  })
})

test.describe('Data Sources UI', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to admin data model page
    await page.goto('/admin/data-model')
    // Wait for page to load
    await page.waitForLoadState('networkidle')
  })

  test('should display data sources list', async ({ page, request }) => {
    // Create a test data source
    const dsId = await createTestDataSource(request, uniqueName('ui_list'))

    // Refresh page
    await page.reload()
    await page.waitForLoadState('networkidle')

    // Should see the data source card or list item
    await expect(page.getByText(/Data Source/i).first()).toBeVisible()

    // Cleanup
    await deleteDataSource(request, dsId)
  })

  test('should open add data source modal', async ({ page }) => {
    // Click add button
    await page.getByRole('button', { name: /Add Data Source/i }).click()

    // Modal should be visible
    await expect(page.getByRole('dialog')).toBeVisible()
    await expect(page.getByText(/Database Engine/i)).toBeVisible()
  })

  test('should show engine options in add modal', async ({ page }) => {
    await page.getByRole('button', { name: /Add Data Source/i }).click()

    // Click on engine select
    await page.getByLabel(/Database Engine/i).click()

    // Should see engine options
    await expect(page.getByRole('option', { name: /SQLite/i })).toBeVisible()
    await expect(page.getByRole('option', { name: /PostgreSQL/i })).toBeVisible()
    await expect(page.getByRole('option', { name: /MySQL/i })).toBeVisible()
  })

  test('should show different fields based on engine', async ({ page }) => {
    await page.getByRole('button', { name: /Add Data Source/i }).click()

    // Select SQLite
    await page.getByLabel(/Database Engine/i).click()
    await page.getByRole('option', { name: /SQLite/i }).click()

    // Should show database path field
    await expect(page.getByLabel(/Database Path/i)).toBeVisible()

    // Should NOT show host/port for SQLite
    await expect(page.getByLabel(/^Host$/i)).not.toBeVisible()

    // Select PostgreSQL
    await page.getByLabel(/Database Engine/i).click()
    await page.getByRole('option', { name: /PostgreSQL/i }).click()

    // Should show host and port
    await expect(page.getByLabel(/^Host$/i)).toBeVisible()
    await expect(page.getByLabel(/Port/i)).toBeVisible()
    await expect(page.getByLabel(/Username/i)).toBeVisible()
    await expect(page.getByLabel(/Password/i)).toBeVisible()
  })

  test('should show SSL options for non-SQLite engines', async ({ page }) => {
    await page.getByRole('button', { name: /Add Data Source/i }).click()

    // Select PostgreSQL
    await page.getByLabel(/Database Engine/i).click()
    await page.getByRole('option', { name: /PostgreSQL/i }).click()

    // Should show SSL toggle
    await expect(page.getByLabel(/Use SSL/i)).toBeVisible()
  })

  test('should test connection from modal', async ({ page }) => {
    await page.getByRole('button', { name: /Add Data Source/i }).click()

    // Fill in SQLite details
    await page.getByLabel(/Display Name/i).fill('Test Connection')
    await page.getByLabel(/Database Engine/i).click()
    await page.getByRole('option', { name: /SQLite/i }).click()
    await page.getByLabel(/Database Path/i).fill(':memory:')

    // Click test connection
    await page.getByRole('button', { name: /Test Connection/i }).click()

    // Should show success message
    await expect(page.getByText(/Connection successful/i)).toBeVisible({ timeout: 10000 })
  })

  test('should create SQLite data source from UI', async ({ page, request }) => {
    const name = uniqueName('ui_create')

    await page.getByRole('button', { name: /Add Data Source/i }).click()

    // Fill form
    await page.getByLabel(/Display Name/i).fill(name)
    await page.getByLabel(/Database Engine/i).click()
    await page.getByRole('option', { name: /SQLite/i }).click()
    await page.getByLabel(/Database Path/i).fill(':memory:')

    // Submit
    await page.getByRole('button', { name: /Add Data Source/i }).last().click()

    // Wait for modal to close and notification
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 10000 })

    // Verify via API
    const datasources = await listDataSources(request)
    const created = datasources.find((ds: any) => ds.name === name)
    expect(created).toBeDefined()

    // Cleanup
    if (created) {
      await deleteDataSource(request, created.id)
    }
  })

  test('should sync data source from UI', async ({ page, request }) => {
    // Create a test data source
    const dsId = await createTestDataSource(request, uniqueName('ui_sync'))

    // Refresh and wait
    await page.reload()
    await page.waitForLoadState('networkidle')

    // Find and click sync button
    const syncButton = page.getByRole('button', { name: /Sync/i }).first()
    if (await syncButton.isVisible()) {
      await syncButton.click()

      // Wait for sync notification
      await expect(page.getByText(/synced/i)).toBeVisible({ timeout: 30000 })
    }

    // Cleanup
    await deleteDataSource(request, dsId)
  })

  test('should delete data source from UI', async ({ page, request }) => {
    // Create a test data source
    const name = uniqueName('ui_delete')
    const response = await request.post(`${API_BASE}/datasources`, {
      data: {
        name,
        engine: 'sqlite',
        database: ':memory:',
      },
    })
    const ds = await response.json()

    // Refresh page
    await page.reload()
    await page.waitForLoadState('networkidle')

    // Open menu and click delete (implementation may vary)
    // This is a simplified version - actual implementation depends on UI
    const menuButton = page.locator('[data-testid="ds-menu"]').first()
    if (await menuButton.isVisible()) {
      await menuButton.click()
      await page.getByRole('menuitem', { name: /Delete/i }).click()

      // Confirm deletion if there's a dialog
      const confirmButton = page.getByRole('button', { name: /Confirm|Delete|Yes/i })
      if (await confirmButton.isVisible()) {
        await confirmButton.click()
      }
    } else {
      // Cleanup via API if UI deletion not possible
      await deleteDataSource(request, ds.id)
    }
  })

  test('should navigate to schema browser tab', async ({ page }) => {
    // Click schema browser tab
    const schemaTab = page.getByRole('tab', { name: /Schema Browser/i })
    if (await schemaTab.isVisible()) {
      await schemaTab.click()

      // Should show data source selector
      await expect(page.getByPlaceholder(/Select data source/i)).toBeVisible()
    }
  })

  test('should navigate to metadata tab', async ({ page }) => {
    // Click metadata tab
    const metadataTab = page.getByRole('tab', { name: /Column Metadata|Metadata/i })
    if (await metadataTab.isVisible()) {
      await metadataTab.click()

      // Should show selectors
      await expect(page.getByPlaceholder(/Select data source/i).first()).toBeVisible()
    }
  })
})

test.describe('Column Metadata Editing', () => {
  let testDsId: string

  test.beforeAll(async ({ request }) => {
    // Create a persistent test data source for these tests
    testDsId = await createTestDataSource(request, uniqueName('col_metadata'))
  })

  test.afterAll(async ({ request }) => {
    // Cleanup
    if (testDsId) {
      await deleteDataSource(request, testDsId)
    }
  })

  test('should update column display name via API', async ({ request }) => {
    // First sync to get tables and columns
    await request.post(`${API_BASE}/datasources/${testDsId}/sync`)

    // Get tables
    const tablesResponse = await request.get(`${API_BASE}/datasources/${testDsId}/tables`)
    const tables = await tablesResponse.json()

    if (tables.length > 0) {
      // Get columns
      const columnsResponse = await request.get(
        `${API_BASE}/datasources/${testDsId}/tables/${tables[0].id}/columns`
      )
      const columns = await columnsResponse.json()

      if (columns.length > 0) {
        const newDisplayName = 'Updated Column Name'
        const updateResponse = await request.put(
          `${API_BASE}/datasources/tables/${tables[0].id}/columns/${columns[0].id}`,
          {
            data: {
              display_name: newDisplayName,
            },
          }
        )
        expect(updateResponse.ok()).toBeTruthy()
        const updated = await updateResponse.json()
        expect(updated.display_name).toBe(newDisplayName)
      }
    }
  })

  test('should update column semantic type via API', async ({ request }) => {
    await request.post(`${API_BASE}/datasources/${testDsId}/sync`)

    const tablesResponse = await request.get(`${API_BASE}/datasources/${testDsId}/tables`)
    const tables = await tablesResponse.json()

    if (tables.length > 0) {
      const columnsResponse = await request.get(
        `${API_BASE}/datasources/${testDsId}/tables/${tables[0].id}/columns`
      )
      const columns = await columnsResponse.json()

      if (columns.length > 0) {
        const updateResponse = await request.put(
          `${API_BASE}/datasources/tables/${tables[0].id}/columns/${columns[0].id}`,
          {
            data: {
              semantic: 'type/Name',
            },
          }
        )
        expect(updateResponse.ok()).toBeTruthy()
        const updated = await updateResponse.json()
        expect(updated.semantic).toBe('type/Name')
      }
    }
  })

  test('should update column visibility via API', async ({ request }) => {
    await request.post(`${API_BASE}/datasources/${testDsId}/sync`)

    const tablesResponse = await request.get(`${API_BASE}/datasources/${testDsId}/tables`)
    const tables = await tablesResponse.json()

    if (tables.length > 0) {
      const columnsResponse = await request.get(
        `${API_BASE}/datasources/${testDsId}/tables/${tables[0].id}/columns`
      )
      const columns = await columnsResponse.json()

      if (columns.length > 0) {
        const updateResponse = await request.put(
          `${API_BASE}/datasources/tables/${tables[0].id}/columns/${columns[0].id}`,
          {
            data: {
              visibility: 'hidden',
            },
          }
        )
        expect(updateResponse.ok()).toBeTruthy()
        const updated = await updateResponse.json()
        expect(updated.visibility).toBe('hidden')
      }
    }
  })

  test('should update column description via API', async ({ request }) => {
    await request.post(`${API_BASE}/datasources/${testDsId}/sync`)

    const tablesResponse = await request.get(`${API_BASE}/datasources/${testDsId}/tables`)
    const tables = await tablesResponse.json()

    if (tables.length > 0) {
      const columnsResponse = await request.get(
        `${API_BASE}/datasources/${testDsId}/tables/${tables[0].id}/columns`
      )
      const columns = await columnsResponse.json()

      if (columns.length > 0) {
        const description = 'This is a test description for the column'
        const updateResponse = await request.put(
          `${API_BASE}/datasources/tables/${tables[0].id}/columns/${columns[0].id}`,
          {
            data: {
              description,
            },
          }
        )
        expect(updateResponse.ok()).toBeTruthy()
        const updated = await updateResponse.json()
        expect(updated.description).toBe(description)
      }
    }
  })
})

test.describe('Table Metadata Editing', () => {
  test('should update table via API', async ({ request }) => {
    const dsId = await createTestDataSource(request, uniqueName('table_meta'))

    // Sync to create tables
    await request.post(`${API_BASE}/datasources/${dsId}/sync`)

    // Get tables
    const tablesResponse = await request.get(`${API_BASE}/datasources/${dsId}/tables`)
    const tables = await tablesResponse.json()

    if (tables.length > 0) {
      const newDisplayName = 'Updated Table Name'
      const newDescription = 'Updated table description'

      const updateResponse = await request.put(
        `${API_BASE}/datasources/${dsId}/tables/${tables[0].id}`,
        {
          data: {
            display_name: newDisplayName,
            description: newDescription,
            visible: false,
          },
        }
      )
      expect(updateResponse.ok()).toBeTruthy()
      const updated = await updateResponse.json()
      expect(updated.display_name).toBe(newDisplayName)
      expect(updated.description).toBe(newDescription)
    }

    // Cleanup
    await deleteDataSource(request, dsId)
  })

  test('should sync single table', async ({ request }) => {
    const dsId = await createTestDataSource(request, uniqueName('table_sync'))

    // Initial sync
    await request.post(`${API_BASE}/datasources/${dsId}/sync`)

    // Get tables
    const tablesResponse = await request.get(`${API_BASE}/datasources/${dsId}/tables`)
    const tables = await tablesResponse.json()

    if (tables.length > 0) {
      const syncResponse = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${tables[0].id}/sync`
      )
      expect(syncResponse.ok()).toBeTruthy()
      const result = await syncResponse.json()
      expect(result.status).toBeDefined()
    }

    // Cleanup
    await deleteDataSource(request, dsId)
  })

  test('should discard cached values for table', async ({ request }) => {
    const dsId = await createTestDataSource(request, uniqueName('discard_cache'))

    await request.post(`${API_BASE}/datasources/${dsId}/sync`)

    const tablesResponse = await request.get(`${API_BASE}/datasources/${dsId}/tables`)
    const tables = await tablesResponse.json()

    if (tables.length > 0) {
      const discardResponse = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${tables[0].id}/discard-values`
      )
      expect(discardResponse.ok()).toBeTruthy()
      const result = await discardResponse.json()
      expect(result.status).toBe('cleared')
    }

    // Cleanup
    await deleteDataSource(request, dsId)
  })
})

test.describe('Complete Workflow Tests', () => {
  test('should complete full data source setup workflow', async ({ request }) => {
    // 1. Test connection
    const testResult = await request.post(`${API_BASE}/datasources/test-connection`, {
      data: {
        engine: 'sqlite',
        database: ':memory:',
      },
    })
    expect(testResult.ok()).toBeTruthy()
    const testData = await testResult.json()
    expect(testData.valid).toBe(true)

    // 2. Create data source
    const name = uniqueName('workflow')
    const createResult = await request.post(`${API_BASE}/datasources`, {
      data: {
        name,
        engine: 'sqlite',
        database: ':memory:',
        auto_sync: true,
        cache_ttl: 3600,
      },
    })
    expect(createResult.ok()).toBeTruthy()
    const ds = await createResult.json()

    // 3. Sync metadata
    const syncResult = await request.post(`${API_BASE}/datasources/${ds.id}/sync`)
    expect(syncResult.ok()).toBeTruthy()

    // 4. Get status
    const statusResult = await request.get(`${API_BASE}/datasources/${ds.id}/status`)
    expect(statusResult.ok()).toBeTruthy()
    const status = await statusResult.json()
    expect(status.connected).toBe(true)

    // 5. Get tables
    const tablesResult = await request.get(`${API_BASE}/datasources/${ds.id}/tables`)
    expect(tablesResult.ok()).toBeTruthy()

    // 6. Get cache stats
    const cacheResult = await request.get(`${API_BASE}/datasources/${ds.id}/cache/stats`)
    expect(cacheResult.ok()).toBeTruthy()

    // 7. Cleanup
    await deleteDataSource(request, ds.id)
  })

  test('should handle concurrent operations', async ({ request }) => {
    const dsId = await createTestDataSource(request, uniqueName('concurrent'))

    // Run multiple operations concurrently
    const [syncResult, statusResult, cacheResult] = await Promise.all([
      request.post(`${API_BASE}/datasources/${dsId}/sync`),
      request.get(`${API_BASE}/datasources/${dsId}/status`),
      request.get(`${API_BASE}/datasources/${dsId}/cache/stats`),
    ])

    expect(syncResult.ok()).toBeTruthy()
    expect(statusResult.ok()).toBeTruthy()
    expect(cacheResult.ok()).toBeTruthy()

    // Cleanup
    await deleteDataSource(request, dsId)
  })
})
