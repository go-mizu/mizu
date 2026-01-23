import { test, expect } from '@playwright/test'

const API_BASE = 'http://localhost:8080/api'

// Helper to get or create the Northwind datasource
async function getNorthwindDataSource(request: any): Promise<{ id: string }> {
  const response = await request.get(`${API_BASE}/datasources`)
  const datasources = await response.json()
  const northwind = datasources.find((ds: any) => ds.name === 'Northwind')
  if (northwind) return { id: northwind.id }

  // Seed if not exists
  await request.post(`${API_BASE}/seed`)
  const newResponse = await request.get(`${API_BASE}/datasources`)
  const newDatasources = await newResponse.json()
  return { id: newDatasources.find((ds: any) => ds.name === 'Northwind')?.id || '' }
}

test.describe('Query Builder API Tests', () => {
  test.describe('Basic Query Execution', () => {
    test('should execute simple SELECT query', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            limit: 10,
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows).toBeDefined()
      expect(result.rows.length).toBeLessThanOrEqual(10)
      expect(result.columns).toBeDefined()
      expect(result.columns.length).toBeGreaterThan(0)
    })

    test('should execute SELECT with specific columns', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            columns: ['name', 'unit_price'],
            limit: 5,
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.columns).toHaveLength(2)
      expect(result.columns.map((c: any) => c.name)).toEqual(['name', 'unit_price'])
    })

    test('should execute SELECT with pagination', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
          },
          page: 1,
          page_size: 5,
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.page).toBe(1)
      expect(result.page_size).toBe(5)
      expect(result.rows.length).toBeLessThanOrEqual(5)
      expect(result.total_rows).toBeGreaterThan(0)
    })
  })

  test.describe('Filter Operations', () => {
    test('should filter with equals operator', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            filters: [
              { column: 'discontinued', operator: '=', value: 0 },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.every((r: any) => r.discontinued === 0)).toBeTruthy()
    })

    test('should filter with greater than operator', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            columns: ['name', 'unit_price'],
            filters: [
              { column: 'unit_price', operator: '>', value: 50 },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.every((r: any) => r.unit_price > 50)).toBeTruthy()
    })

    test('should filter with BETWEEN operator', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            columns: ['name', 'unit_price'],
            filters: [
              { column: 'unit_price', operator: 'BETWEEN', value: [10, 30] },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.every((r: any) => r.unit_price >= 10 && r.unit_price <= 30)).toBeTruthy()
    })

    test('should filter with contains operator', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            columns: ['name', 'unit_price'],
            filters: [
              { column: 'name', operator: 'contains', value: 'Sauce' },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.every((r: any) => r.name.toLowerCase().includes('sauce'))).toBeTruthy()
    })

    test('should filter with starts-with operator', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            columns: ['name'],
            filters: [
              { column: 'name', operator: 'starts-with', value: 'Chef' },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.every((r: any) => r.name.startsWith('Chef'))).toBeTruthy()
    })

    test('should filter with IS NULL operator', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'customers',
            columns: ['company_name', 'region'],
            filters: [
              { column: 'region', operator: 'IS NULL' },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.every((r: any) => r.region === null)).toBeTruthy()
    })

    test('should filter with IN operator', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'customers',
            columns: ['company_name', 'country'],
            filters: [
              { column: 'country', operator: 'IN', value: ['USA', 'UK', 'Germany'] },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.every((r: any) => ['USA', 'UK', 'Germany'].includes(r.country))).toBeTruthy()
    })

    test('should combine multiple filters with AND', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            columns: ['name', 'unit_price', 'discontinued'],
            filters: [
              { column: 'unit_price', operator: '>', value: 20 },
              { column: 'discontinued', operator: '=', value: 0 },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.every((r: any) => r.unit_price > 20 && r.discontinued === 0)).toBeTruthy()
    })
  })

  test.describe('Aggregation Operations', () => {
    test('should execute COUNT aggregation', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            aggregations: [
              { function: 'count', alias: 'total_products' },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows).toHaveLength(1)
      expect(result.rows[0].total_products).toBeGreaterThan(0)
    })

    test('should execute SUM aggregation', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            aggregations: [
              { function: 'sum', column: 'units_in_stock', alias: 'total_stock' },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows).toHaveLength(1)
      expect(result.rows[0].total_stock).toBeGreaterThan(0)
    })

    test('should execute AVG aggregation', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            aggregations: [
              { function: 'avg', column: 'unit_price', alias: 'avg_price' },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows).toHaveLength(1)
      expect(result.rows[0].avg_price).toBeGreaterThan(0)
    })

    test('should execute DISTINCT count aggregation', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            aggregations: [
              { function: 'distinct', column: 'category_id', alias: 'unique_categories' },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows).toHaveLength(1)
      expect(result.rows[0].unique_categories).toBeGreaterThan(0)
    })

    test('should execute MIN/MAX aggregations', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            aggregations: [
              { function: 'min', column: 'unit_price', alias: 'min_price' },
              { function: 'max', column: 'unit_price', alias: 'max_price' },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows).toHaveLength(1)
      expect(result.rows[0].min_price).toBeLessThan(result.rows[0].max_price)
    })

    test('should execute aggregation with GROUP BY', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            aggregations: [
              { function: 'count', alias: 'product_count' },
              { function: 'avg', column: 'unit_price', alias: 'avg_price' },
            ],
            group_by: ['category_id'],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.length).toBeGreaterThan(1)
      expect(result.rows.every((r: any) => r.product_count > 0)).toBeTruthy()
      expect(result.rows.every((r: any) => r.category_id !== undefined)).toBeTruthy()
    })

    test('should execute aggregation with HAVING', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            aggregations: [
              { function: 'count', alias: 'product_count' },
            ],
            group_by: ['category_id'],
            having: [
              { function: 'count', operator: '>', value: 5 },
            ],
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.every((r: any) => r.product_count > 5)).toBeTruthy()
    })
  })

  test.describe('JOIN Operations', () => {
    test('should execute LEFT JOIN query', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            columns: ['name', 'unit_price'],
            joins: [
              {
                type: 'left',
                target_table: 'categories',
                conditions: [
                  { source_column: 'category_id', target_column: 'id' },
                ],
              },
            ],
            limit: 10,
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.length).toBeGreaterThan(0)
    })

    test('should execute INNER JOIN query', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'order_details',
            columns: ['quantity', 'unit_price'],
            joins: [
              {
                type: 'inner',
                target_table: 'products',
                conditions: [
                  { source_column: 'product_id', target_column: 'id' },
                ],
              },
            ],
            limit: 10,
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.length).toBeGreaterThan(0)
    })
  })

  test.describe('Sort and Limit Operations', () => {
    test('should sort results ascending', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            columns: ['name', 'unit_price'],
            order_by: [
              { column: 'unit_price', direction: 'asc' },
            ],
            limit: 10,
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      const prices = result.rows.map((r: any) => r.unit_price)
      for (let i = 1; i < prices.length; i++) {
        expect(prices[i]).toBeGreaterThanOrEqual(prices[i - 1])
      }
    })

    test('should sort results descending', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            columns: ['name', 'unit_price'],
            order_by: [
              { column: 'unit_price', direction: 'desc' },
            ],
            limit: 10,
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      const prices = result.rows.map((r: any) => r.unit_price)
      for (let i = 1; i < prices.length; i++) {
        expect(prices[i]).toBeLessThanOrEqual(prices[i - 1])
      }
    })

    test('should sort with multiple columns', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            columns: ['category_id', 'name', 'unit_price'],
            order_by: [
              { column: 'category_id', direction: 'asc' },
              { column: 'unit_price', direction: 'desc' },
            ],
            limit: 20,
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.length).toBeGreaterThan(0)
    })

    test('should respect limit', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            limit: 5,
          },
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.length).toBe(5)
    })
  })

  test.describe('Native SQL Queries', () => {
    test('should execute native SQL query', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query/native`, {
        data: {
          datasource_id: ds.id,
          query: 'SELECT name, unit_price FROM products WHERE unit_price > ? ORDER BY unit_price DESC LIMIT 5',
          params: [50],
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows).toHaveLength(5)
      expect(result.rows.every((r: any) => r.unit_price > 50)).toBeTruthy()
    })

    test('should execute native SQL with aggregations', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query/native`, {
        data: {
          datasource_id: ds.id,
          query: `
            SELECT
              c.name as category,
              COUNT(*) as product_count,
              AVG(p.unit_price) as avg_price
            FROM products p
            JOIN categories c ON p.category_id = c.id
            GROUP BY c.id
            ORDER BY product_count DESC
          `,
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.rows.length).toBeGreaterThan(0)
      expect(result.rows.every((r: any) => r.category && r.product_count > 0)).toBeTruthy()
    })
  })

  test.describe('Complex Queries', () => {
    test('should execute query with all features combined', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            columns: ['name', 'unit_price', 'units_in_stock'],
            filters: [
              { column: 'discontinued', operator: '=', value: 0 },
              { column: 'unit_price', operator: '>', value: 10 },
            ],
            order_by: [
              { column: 'unit_price', direction: 'desc' },
            ],
            limit: 15,
          },
          page: 1,
          page_size: 5,
        },
      })

      expect(response.ok()).toBeTruthy()
      const result = await response.json()
      expect(result.page).toBe(1)
      expect(result.rows.length).toBe(5)
      expect(result.total_rows).toBeGreaterThan(5)
      expect(result.rows.every((r: any) => r.unit_price > 10)).toBeTruthy()
    })
  })

  test.describe('Error Handling', () => {
    test('should return error for invalid table name', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'invalid; DROP TABLE users;',
          },
        },
      })

      expect(response.status()).toBe(500)
      const result = await response.json()
      expect(result.error).toBeDefined()
    })

    test('should return error for invalid column name', async ({ request }) => {
      const ds = await getNorthwindDataSource(request)

      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: ds.id,
          query: {
            table: 'products',
            columns: ['valid_column', 'invalid; --'],
          },
        },
      })

      expect(response.status()).toBe(500)
      const result = await response.json()
      expect(result.error).toBeDefined()
    })

    test('should return error for non-existent datasource', async ({ request }) => {
      const response = await request.post(`${API_BASE}/query`, {
        data: {
          datasource_id: 'non-existent-id',
          query: {
            table: 'products',
          },
        },
      })

      expect(response.status()).toBe(404)
    })
  })
})

test.describe('Query Builder UI Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    // Wait for app to load
    await page.waitForSelector('[data-testid="mode-toggle"]', { timeout: 10000 })
  })

  test('should display mode toggle', async ({ page }) => {
    const toggle = page.locator('[data-testid="mode-toggle"]')
    await expect(toggle).toBeVisible()
  })

  test('should switch between Simple and Native query modes', async ({ page }) => {
    // Start in Simple mode
    const toggle = page.locator('[data-testid="mode-toggle"]')
    await expect(toggle).toBeVisible()

    // Click Native query tab
    await page.click('text=Native query')

    // SQL editor should be visible
    const sqlEditor = page.locator('[data-testid="sql-editor"]')
    await expect(sqlEditor).toBeVisible()

    // Click Simple tab to go back
    await page.click('text=Simple')

    // SQL editor should not be visible
    await expect(sqlEditor).not.toBeVisible()
  })

  test('should have Visualize button', async ({ page }) => {
    const button = page.locator('[data-testid="btn-run-query"]')
    await expect(button).toBeVisible()
    await expect(button).toHaveText('Visualize')
  })
})
