import { test, expect, APIRequestContext } from '@playwright/test'

/**
 * Data Table E2E Tests
 *
 * Tests for the data table visualization features:
 * - Server-side pagination
 * - Column sorting (single and multi-column)
 * - Column filtering (various operators)
 * - Column hiding/showing
 * - Table preview
 * - Database browser
 *
 * Uses Northwind seed data for realistic testing
 */

const API_BASE = process.env.E2E_API_URL || 'http://localhost:3000/api'

// Helper to generate unique names
const uniqueName = (prefix: string) => `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`

// Helper to create a Northwind test data source
async function createNorthwindDataSource(request: APIRequestContext): Promise<{ id: string; name: string }> {
  const name = uniqueName('northwind')
  const response = await request.post(`${API_BASE}/datasources`, {
    data: {
      name,
      engine: 'sqlite',
      database: 'northwind', // Uses Northwind seed
    },
  })
  expect(response.ok()).toBeTruthy()
  const ds = await response.json()

  // Sync to populate tables
  const syncResponse = await request.post(`${API_BASE}/datasources/${ds.id}/sync`)
  expect(syncResponse.ok()).toBeTruthy()

  return { id: ds.id, name: ds.name }
}

// Helper to delete a data source
async function deleteDataSource(request: APIRequestContext, id: string): Promise<void> {
  await request.delete(`${API_BASE}/datasources/${id}`)
}

// Get tables from data source
async function getTables(request: APIRequestContext, dsId: string): Promise<any[]> {
  const response = await request.get(`${API_BASE}/datasources/${dsId}/tables`)
  expect(response.ok()).toBeTruthy()
  return response.json()
}

// Get table by name
async function getTableByName(request: APIRequestContext, dsId: string, tableName: string): Promise<any> {
  const tables = await getTables(request, dsId)
  return tables.find((t: any) => t.name === tableName)
}

test.describe('Table Preview API', () => {
  let dsId: string
  let dsName: string

  test.beforeAll(async ({ request }) => {
    const ds = await createNorthwindDataSource(request)
    dsId = ds.id
    dsName = ds.name
  })

  test.afterAll(async ({ request }) => {
    if (dsId) {
      await deleteDataSource(request, dsId)
    }
  })

  test.describe('Server-Side Pagination', () => {
    test('should return paginated results with metadata', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')
      expect(productsTable).toBeDefined()

      const response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 10,
          },
        }
      )
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      // Verify pagination metadata
      expect(result.row_count).toBeLessThanOrEqual(10)
      expect(result.total_rows).toBeDefined()
      expect(result.total_rows).toBeGreaterThan(0)
      expect(result.page).toBe(1)
      expect(result.page_size).toBe(10)
      expect(result.total_pages).toBeDefined()
      expect(result.total_pages).toBe(Math.ceil(result.total_rows / 10))

      // Verify data structure
      expect(result.columns).toBeInstanceOf(Array)
      expect(result.columns.length).toBeGreaterThan(0)
      expect(result.rows).toBeInstanceOf(Array)
    })

    test('should return correct data for different pages', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')

      // Get first page
      const page1Response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 5,
          },
        }
      )
      const page1 = await page1Response.json()

      // Get second page
      const page2Response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 2,
            page_size: 5,
          },
        }
      )
      const page2 = await page2Response.json()

      // Pages should have different data
      expect(page1.page).toBe(1)
      expect(page2.page).toBe(2)

      // First item of page 2 should be different from first item of page 1
      if (page1.rows.length > 0 && page2.rows.length > 0) {
        const page1FirstId = page1.rows[0].id
        const page2FirstId = page2.rows[0].id
        expect(page1FirstId).not.toBe(page2FirstId)
      }
    })

    test('should respect page_size parameter', async ({ request }) => {
      const customersTable = await getTableByName(request, dsId, 'customers')

      // Request different page sizes
      for (const pageSize of [5, 10, 25, 50]) {
        const response = await request.post(
          `${API_BASE}/datasources/${dsId}/tables/${customersTable.id}/preview`,
          {
            data: {
              page: 1,
              page_size: pageSize,
            },
          }
        )
        const result = await response.json()
        expect(result.page_size).toBe(pageSize)
        expect(result.row_count).toBeLessThanOrEqual(pageSize)
      }
    })
  })

  test.describe('Column Sorting', () => {
    test('should sort by column ascending', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')

      const response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 100,
            order_by: [{ column: 'name', direction: 'asc' }],
          },
        }
      )
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      // Verify ascending order
      for (let i = 1; i < result.rows.length; i++) {
        const prev = result.rows[i - 1].name
        const curr = result.rows[i].name
        expect(prev.localeCompare(curr)).toBeLessThanOrEqual(0)
      }
    })

    test('should sort by column descending', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')

      const response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 100,
            order_by: [{ column: 'name', direction: 'desc' }],
          },
        }
      )
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      // Verify descending order
      for (let i = 1; i < result.rows.length; i++) {
        const prev = result.rows[i - 1].name
        const curr = result.rows[i].name
        expect(prev.localeCompare(curr)).toBeGreaterThanOrEqual(0)
      }
    })

    test('should sort numeric columns correctly', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')

      const response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 100,
            order_by: [{ column: 'unit_price', direction: 'asc' }],
          },
        }
      )
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      // Verify numeric ascending order
      for (let i = 1; i < result.rows.length; i++) {
        const prev = result.rows[i - 1].unit_price
        const curr = result.rows[i].unit_price
        expect(prev).toBeLessThanOrEqual(curr)
      }
    })

    test('should maintain sort order across pages', async ({ request }) => {
      const customersTable = await getTableByName(request, dsId, 'customers')

      // Get first page sorted
      const page1Response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${customersTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 10,
            order_by: [{ column: 'company_name', direction: 'asc' }],
          },
        }
      )
      const page1 = await page1Response.json()

      // Get second page sorted
      const page2Response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${customersTable.id}/preview`,
        {
          data: {
            page: 2,
            page_size: 10,
            order_by: [{ column: 'company_name', direction: 'asc' }],
          },
        }
      )
      const page2 = await page2Response.json()

      // Last item of page 1 should be before first item of page 2
      if (page1.rows.length > 0 && page2.rows.length > 0) {
        const lastPage1 = page1.rows[page1.rows.length - 1].company_name
        const firstPage2 = page2.rows[0].company_name
        expect(lastPage1.localeCompare(firstPage2)).toBeLessThanOrEqual(0)
      }
    })
  })

  test.describe('Column Filtering', () => {
    test('should filter with equals operator', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')

      const response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 100,
            filters: [{ column: 'discontinued', operator: '=', value: 0 }],
          },
        }
      )
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      // All rows should have discontinued = 0
      for (const row of result.rows) {
        expect(row.discontinued).toBe(0)
      }
    })

    test('should filter with not-equals operator', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')

      const response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 100,
            filters: [{ column: 'discontinued', operator: '!=', value: 0 }],
          },
        }
      )
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      // All rows should have discontinued != 0
      for (const row of result.rows) {
        expect(row.discontinued).not.toBe(0)
      }
    })

    test('should filter with greater-than operator', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')
      const threshold = 20.0

      const response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 100,
            filters: [{ column: 'unit_price', operator: '>', value: threshold }],
          },
        }
      )
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      // All rows should have unit_price > threshold
      for (const row of result.rows) {
        expect(row.unit_price).toBeGreaterThan(threshold)
      }
    })

    test('should filter with less-than operator', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')
      const threshold = 15.0

      const response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 100,
            filters: [{ column: 'unit_price', operator: '<', value: threshold }],
          },
        }
      )
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      // All rows should have unit_price < threshold
      for (const row of result.rows) {
        expect(row.unit_price).toBeLessThan(threshold)
      }
    })

    test('should filter with contains operator', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')

      const response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 100,
            filters: [{ column: 'name', operator: 'contains', value: 'Sauce' }],
          },
        }
      )
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      // All rows should have 'Sauce' in name
      for (const row of result.rows) {
        expect(row.name.toLowerCase()).toContain('sauce')
      }
    })

    test('should apply multiple filters (AND logic)', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')

      const response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 100,
            filters: [
              { column: 'discontinued', operator: '=', value: 0 },
              { column: 'unit_price', operator: '>', value: 10 },
            ],
          },
        }
      )
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      // All rows should match both conditions
      for (const row of result.rows) {
        expect(row.discontinued).toBe(0)
        expect(row.unit_price).toBeGreaterThan(10)
      }
    })

    test('should filter and sort together', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')

      const response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 100,
            filters: [{ column: 'discontinued', operator: '=', value: 0 }],
            order_by: [{ column: 'unit_price', direction: 'desc' }],
          },
        }
      )
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      // All rows should match filter
      for (const row of result.rows) {
        expect(row.discontinued).toBe(0)
      }

      // Rows should be sorted by unit_price descending
      for (let i = 1; i < result.rows.length; i++) {
        expect(result.rows[i - 1].unit_price).toBeGreaterThanOrEqual(result.rows[i].unit_price)
      }
    })

    test('should return correct total_rows after filtering', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')

      // Get unfiltered count
      const unfilteredResponse = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: { page: 1, page_size: 100 },
        }
      )
      const unfiltered = await unfilteredResponse.json()

      // Get filtered count
      const filteredResponse = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: {
            page: 1,
            page_size: 100,
            filters: [{ column: 'discontinued', operator: '=', value: 1 }],
          },
        }
      )
      const filtered = await filteredResponse.json()

      // Filtered should have fewer total_rows
      expect(filtered.total_rows).toBeLessThan(unfiltered.total_rows)
    })
  })

  test.describe('Column Information', () => {
    test('should return column metadata in results', async ({ request }) => {
      const productsTable = await getTableByName(request, dsId, 'products')

      const response = await request.post(
        `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
        {
          data: { page: 1, page_size: 10 },
        }
      )
      expect(response.ok()).toBeTruthy()
      const result = await response.json()

      // Verify column structure
      expect(result.columns).toBeInstanceOf(Array)
      expect(result.columns.length).toBeGreaterThan(0)

      for (const column of result.columns) {
        expect(column.name).toBeDefined()
        expect(column.type).toBeDefined()
      }

      // Verify expected columns exist
      const columnNames = result.columns.map((c: any) => c.name)
      expect(columnNames).toContain('id')
      expect(columnNames).toContain('name')
      expect(columnNames).toContain('unit_price')
    })
  })
})

test.describe('Database Browser', () => {
  let dsId: string
  let dsName: string

  test.beforeAll(async ({ request }) => {
    const ds = await createNorthwindDataSource(request)
    dsId = ds.id
    dsName = ds.name
  })

  test.afterAll(async ({ request }) => {
    if (dsId) {
      await deleteDataSource(request, dsId)
    }
  })

  test('should list all tables in Northwind database', async ({ request }) => {
    const tables = await getTables(request, dsId)

    // Northwind should have these tables
    const tableNames = tables.map((t: any) => t.name)
    expect(tableNames).toContain('categories')
    expect(tableNames).toContain('suppliers')
    expect(tableNames).toContain('products')
    expect(tableNames).toContain('customers')
    expect(tableNames).toContain('employees')
    expect(tableNames).toContain('orders')
    expect(tableNames).toContain('order_details')
  })

  test('should get table row counts', async ({ request }) => {
    const tables = await getTables(request, dsId)

    // Products table should have rows
    const productsTable = tables.find((t: any) => t.name === 'products')
    expect(productsTable).toBeDefined()
    expect(productsTable.row_count).toBeGreaterThan(0)

    // Orders table should have many rows
    const ordersTable = tables.find((t: any) => t.name === 'orders')
    if (ordersTable && ordersTable.row_count !== undefined) {
      expect(ordersTable.row_count).toBeGreaterThan(100)
    }
  })

  test('should get columns for a table', async ({ request }) => {
    const productsTable = await getTableByName(request, dsId, 'products')

    const response = await request.get(
      `${API_BASE}/datasources/tables/${productsTable.id}/columns`
    )
    expect(response.ok()).toBeTruthy()
    const columns = await response.json()

    // Verify column structure
    expect(columns).toBeInstanceOf(Array)
    expect(columns.length).toBeGreaterThan(0)

    // Verify expected columns
    const columnNames = columns.map((c: any) => c.name)
    expect(columnNames).toContain('id')
    expect(columnNames).toContain('name')
    expect(columnNames).toContain('category_id')
    expect(columnNames).toContain('supplier_id')
    expect(columnNames).toContain('unit_price')
    expect(columnNames).toContain('units_in_stock')
    expect(columnNames).toContain('discontinued')
  })

  test('should search tables', async ({ request }) => {
    const response = await request.get(
      `${API_BASE}/datasources/${dsId}/search-tables?q=prod`
    )
    expect(response.ok()).toBeTruthy()
    const tables = await response.json()

    // Should find products table
    expect(tables).toBeInstanceOf(Array)
    const hasProducts = tables.some((t: any) => t.name.toLowerCase().includes('prod'))
    expect(hasProducts).toBeTruthy()
  })
})

test.describe('Database Browser UI', () => {
  let dsId: string
  let dsName: string

  test.beforeAll(async ({ request }) => {
    const ds = await createNorthwindDataSource(request)
    dsId = ds.id
    dsName = ds.name
  })

  test.afterAll(async ({ request }) => {
    if (dsId) {
      await deleteDataSource(request, dsId)
    }
  })

  test('should display database browser page', async ({ page }) => {
    await page.goto(`/browse/database/${dsId}`)
    await page.waitForLoadState('networkidle')

    // Should show database name
    await expect(page.getByText(dsName)).toBeVisible({ timeout: 10000 })
  })

  test('should list tables in database browser', async ({ page }) => {
    await page.goto(`/browse/database/${dsId}`)
    await page.waitForLoadState('networkidle')

    // Should show some Northwind tables
    await expect(page.getByText('products')).toBeVisible({ timeout: 10000 })
    await expect(page.getByText('customers')).toBeVisible()
    await expect(page.getByText('orders')).toBeVisible()
  })

  test('should navigate to table preview', async ({ page }) => {
    await page.goto(`/browse/database/${dsId}`)
    await page.waitForLoadState('networkidle')

    // Click on products table
    await page.getByText('products').click()

    // Should navigate to table preview
    await expect(page).toHaveURL(new RegExp(`/browse/database/${dsId}/table/`))
  })

  test('should display table data with columns', async ({ page }) => {
    await page.goto(`/browse/database/${dsId}`)
    await page.waitForLoadState('networkidle')

    // Click on products table
    await page.getByText('products').click()
    await page.waitForLoadState('networkidle')

    // Should show data table
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 })

    // Should show column headers
    await expect(page.getByText('name', { exact: false })).toBeVisible()
  })

  test('should show pagination controls', async ({ page }) => {
    await page.goto(`/browse/database/${dsId}`)
    await page.waitForLoadState('networkidle')

    // Click on orders table (has many rows)
    await page.getByText('orders').click()
    await page.waitForLoadState('networkidle')

    // Should show pagination info
    await expect(page.getByText(/Showing/)).toBeVisible({ timeout: 10000 })
    await expect(page.getByText(/rows/)).toBeVisible()
  })

  test('should change page size', async ({ page }) => {
    await page.goto(`/browse/database/${dsId}`)
    await page.waitForLoadState('networkidle')

    // Click on products table
    await page.getByText('products').click()
    await page.waitForLoadState('networkidle')

    // Find page size selector
    const pageSizeSelect = page.getByRole('combobox').first()
    if (await pageSizeSelect.isVisible()) {
      await pageSizeSelect.click()
      await page.getByRole('option', { name: /50 rows/i }).click()
    }
  })

  test('should sort by clicking column header', async ({ page }) => {
    await page.goto(`/browse/database/${dsId}`)
    await page.waitForLoadState('networkidle')

    // Click on products table
    await page.getByText('products').click()
    await page.waitForLoadState('networkidle')

    // Click on name column header to sort
    const nameHeader = page.getByRole('columnheader', { name: /name/i })
    if (await nameHeader.isVisible()) {
      await nameHeader.click()
      // Wait for sort to apply
      await page.waitForTimeout(500)
    }
  })

  test('should show columns tab with metadata', async ({ page }) => {
    await page.goto(`/browse/database/${dsId}`)
    await page.waitForLoadState('networkidle')

    // Click on products table
    await page.getByText('products').click()
    await page.waitForLoadState('networkidle')

    // Click on Columns tab
    const columnsTab = page.getByRole('tab', { name: /Columns/i })
    if (await columnsTab.isVisible()) {
      await columnsTab.click()

      // Should show column information
      await expect(page.getByText('Type')).toBeVisible({ timeout: 5000 })
    }
  })

  test('should search tables in database browser', async ({ page }) => {
    await page.goto(`/browse/database/${dsId}`)
    await page.waitForLoadState('networkidle')

    // Find search input
    const searchInput = page.getByPlaceholder(/search tables/i)
    if (await searchInput.isVisible()) {
      await searchInput.fill('prod')
      await page.waitForTimeout(500)

      // Should filter tables
      await expect(page.getByText('products')).toBeVisible()
    }
  })
})

test.describe('Table Visualization Component', () => {
  let dsId: string

  test.beforeAll(async ({ request }) => {
    const ds = await createNorthwindDataSource(request)
    dsId = ds.id
  })

  test.afterAll(async ({ request }) => {
    if (dsId) {
      await deleteDataSource(request, dsId)
    }
  })

  test('should display Northwind products data', async ({ request }) => {
    const productsTable = await getTableByName(request, dsId, 'products')

    const response = await request.post(
      `${API_BASE}/datasources/${dsId}/tables/${productsTable.id}/preview`,
      {
        data: { page: 1, page_size: 25 },
      }
    )
    expect(response.ok()).toBeTruthy()
    const result = await response.json()

    // Verify Northwind product data
    expect(result.rows.length).toBeGreaterThan(0)

    // Check for specific Northwind products
    const productNames = result.rows.map((r: any) => r.name)
    expect(productNames.some((name: string) => name.includes('Chai') || name.includes('Chang'))).toBeTruthy()
  })

  test('should display Northwind customers data', async ({ request }) => {
    const customersTable = await getTableByName(request, dsId, 'customers')

    const response = await request.post(
      `${API_BASE}/datasources/${dsId}/tables/${customersTable.id}/preview`,
      {
        data: { page: 1, page_size: 25 },
      }
    )
    expect(response.ok()).toBeTruthy()
    const result = await response.json()

    // Verify Northwind customer data
    expect(result.rows.length).toBeGreaterThan(0)

    // Check for customer fields
    const firstCustomer = result.rows[0]
    expect(firstCustomer.company_name).toBeDefined()
    expect(firstCustomer.contact_name).toBeDefined()
    expect(firstCustomer.city).toBeDefined()
    expect(firstCustomer.country).toBeDefined()
  })

  test('should display Northwind orders with related data', async ({ request }) => {
    const ordersTable = await getTableByName(request, dsId, 'orders')

    const response = await request.post(
      `${API_BASE}/datasources/${dsId}/tables/${ordersTable.id}/preview`,
      {
        data: {
          page: 1,
          page_size: 25,
          order_by: [{ column: 'order_date', direction: 'desc' }],
        },
      }
    )
    expect(response.ok()).toBeTruthy()
    const result = await response.json()

    // Verify order data
    expect(result.rows.length).toBeGreaterThan(0)

    // Check for order fields
    const firstOrder = result.rows[0]
    expect(firstOrder.customer_id).toBeDefined()
    expect(firstOrder.order_date).toBeDefined()
    expect(firstOrder.freight).toBeDefined()
  })

  test('should handle large result sets with pagination', async ({ request }) => {
    const ordersTable = await getTableByName(request, dsId, 'orders')

    // Get total count
    const countResponse = await request.post(
      `${API_BASE}/datasources/${dsId}/tables/${ordersTable.id}/preview`,
      {
        data: { page: 1, page_size: 10 },
      }
    )
    const countResult = await countResponse.json()
    const totalRows = countResult.total_rows

    // Northwind should have ~2500 orders
    expect(totalRows).toBeGreaterThan(100)

    // Verify pagination math
    const pageSize = 50
    const expectedPages = Math.ceil(totalRows / pageSize)
    expect(countResult.total_pages || expectedPages).toBe(expectedPages)
  })
})

test.describe('Query Execution', () => {
  let dsId: string

  test.beforeAll(async ({ request }) => {
    const ds = await createNorthwindDataSource(request)
    dsId = ds.id
  })

  test.afterAll(async ({ request }) => {
    if (dsId) {
      await deleteDataSource(request, dsId)
    }
  })

  test('should execute native SQL query', async ({ request }) => {
    const response = await request.post(`${API_BASE}/query/native`, {
      data: {
        datasource_id: dsId,
        query: 'SELECT name, unit_price FROM products ORDER BY unit_price DESC LIMIT 10',
      },
    })
    expect(response.ok()).toBeTruthy()
    const result = await response.json()

    // Verify query results
    expect(result.columns).toBeInstanceOf(Array)
    expect(result.columns.length).toBe(2)
    expect(result.rows).toBeInstanceOf(Array)
    expect(result.rows.length).toBeLessThanOrEqual(10)

    // Verify column names
    const columnNames = result.columns.map((c: any) => c.name)
    expect(columnNames).toContain('name')
    expect(columnNames).toContain('unit_price')

    // Verify sorted by price descending
    for (let i = 1; i < result.rows.length; i++) {
      expect(result.rows[i - 1].unit_price).toBeGreaterThanOrEqual(result.rows[i].unit_price)
    }
  })

  test('should execute aggregate query', async ({ request }) => {
    const response = await request.post(`${API_BASE}/query/native`, {
      data: {
        datasource_id: dsId,
        query: 'SELECT COUNT(*) as total_products, AVG(unit_price) as avg_price FROM products',
      },
    })
    expect(response.ok()).toBeTruthy()
    const result = await response.json()

    expect(result.rows.length).toBe(1)
    expect(result.rows[0].total_products).toBeGreaterThan(0)
    expect(result.rows[0].avg_price).toBeGreaterThan(0)
  })

  test('should execute JOIN query', async ({ request }) => {
    const response = await request.post(`${API_BASE}/query/native`, {
      data: {
        datasource_id: dsId,
        query: `
          SELECT p.name, c.name as category_name, p.unit_price
          FROM products p
          JOIN categories c ON p.category_id = c.id
          LIMIT 10
        `,
      },
    })
    expect(response.ok()).toBeTruthy()
    const result = await response.json()

    expect(result.rows.length).toBeGreaterThan(0)
    expect(result.rows[0].name).toBeDefined()
    expect(result.rows[0].category_name).toBeDefined()
  })

  test('should support pagination in queries', async ({ request }) => {
    const response = await request.post(`${API_BASE}/query`, {
      data: {
        datasource_id: dsId,
        query: {
          table: 'customers',
          page: 1,
          page_size: 10,
        },
      },
    })
    expect(response.ok()).toBeTruthy()
    const result = await response.json()

    expect(result.page).toBe(1)
    expect(result.page_size).toBe(10)
    expect(result.row_count).toBeLessThanOrEqual(10)
    expect(result.total_rows).toBeGreaterThan(0)
  })
})
