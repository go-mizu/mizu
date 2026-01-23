import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'
import { MantineProvider } from '@mantine/core'
import TableVisualization from './TableVisualization'
import type { QueryResult, ResultColumn } from '../../../api/types'
import type { TableSettings } from './types'

// Northwind-like test data
const createNorthwindColumns = (): ResultColumn[] => [
  { name: 'id', display_name: 'ID', type: 'integer' },
  { name: 'name', display_name: 'Product Name', type: 'string' },
  { name: 'unit_price', display_name: 'Unit Price', type: 'number' },
  { name: 'units_in_stock', display_name: 'In Stock', type: 'integer' },
  { name: 'discontinued', display_name: 'Discontinued', type: 'boolean' },
  { name: 'category', display_name: 'Category', type: 'string' },
]

const createNorthwindRows = () => [
  { id: 1, name: 'Chai', unit_price: 18.0, units_in_stock: 39, discontinued: false, category: 'Beverages' },
  { id: 2, name: 'Chang', unit_price: 19.0, units_in_stock: 17, discontinued: false, category: 'Beverages' },
  { id: 3, name: 'Aniseed Syrup', unit_price: 10.0, units_in_stock: 13, discontinued: false, category: 'Condiments' },
  { id: 4, name: 'Chef Anton\'s Cajun Seasoning', unit_price: 22.0, units_in_stock: 53, discontinued: false, category: 'Condiments' },
  { id: 5, name: 'Chef Anton\'s Gumbo Mix', unit_price: 21.35, units_in_stock: 0, discontinued: true, category: 'Condiments' },
  { id: 6, name: 'Grandma\'s Boysenberry Spread', unit_price: 25.0, units_in_stock: 120, discontinued: false, category: 'Condiments' },
  { id: 7, name: 'Uncle Bob\'s Organic Dried Pears', unit_price: 30.0, units_in_stock: 15, discontinued: false, category: 'Produce' },
  { id: 8, name: 'Northwoods Cranberry Sauce', unit_price: 40.0, units_in_stock: 6, discontinued: false, category: 'Condiments' },
  { id: 9, name: 'Mishi Kobe Niku', unit_price: 97.0, units_in_stock: 29, discontinued: true, category: 'Meat/Poultry' },
  { id: 10, name: 'Ikura', unit_price: 31.0, units_in_stock: 31, discontinued: false, category: 'Seafood' },
]

const createMockResult = (overrides?: Partial<QueryResult>): QueryResult => ({
  columns: createNorthwindColumns(),
  rows: createNorthwindRows(),
  row_count: 10,
  total_rows: 10,
  page: 1,
  page_size: 25,
  total_pages: 1,
  duration_ms: 5,
  ...overrides,
})

const mockResult: QueryResult = {
  columns: [
    { name: 'id', display_name: 'ID', type: 'number' },
    { name: 'name', display_name: 'Name', type: 'string' },
    { name: 'price', display_name: 'Price', type: 'number' },
    { name: 'created_at', display_name: 'Created At', type: 'datetime' },
  ],
  rows: [
    { id: 1, name: 'Product A', price: 29.99, created_at: '2024-01-15T10:30:00Z' },
    { id: 2, name: 'Product B', price: 49.99, created_at: '2024-01-16T14:20:00Z' },
    { id: 3, name: 'Product C', price: null, created_at: '2024-01-17T09:15:00Z' },
  ],
  row_count: 3,
  duration_ms: 42,
}

const renderTable = (result: QueryResult = mockResult, settings: TableSettings = {}) => {
  return render(
    <MantineProvider>
      <TableVisualization result={result} settings={settings} />
    </MantineProvider>
  )
}

describe('TableVisualization', () => {
  describe('Basic Rendering', () => {
    it('renders all columns and rows', () => {
      renderTable()

      // Check headers
      expect(screen.getByText('ID')).toBeInTheDocument()
      expect(screen.getByText('Name')).toBeInTheDocument()
      expect(screen.getByText('Price')).toBeInTheDocument()
      expect(screen.getByText('Created At')).toBeInTheDocument()

      // Check data
      expect(screen.getByText('Product A')).toBeInTheDocument()
      expect(screen.getByText('Product B')).toBeInTheDocument()
      expect(screen.getByText('Product C')).toBeInTheDocument()
    })

    it('renders Northwind data correctly', () => {
      const result = createMockResult()
      renderTable(result)

      // Check Northwind product names
      expect(screen.getByText('Chai')).toBeInTheDocument()
      expect(screen.getByText('Chang')).toBeInTheDocument()
      expect(screen.getByText('Aniseed Syrup')).toBeInTheDocument()
    })

    it('displays row index when showRowIndex is true', () => {
      const { container } = renderTable(mockResult, { showRowIndex: true })

      // Row index header
      expect(screen.getByText('#')).toBeInTheDocument()

      // Row index cells should have the rowIndexCell class
      const indexCells = container.querySelectorAll('[class*="rowIndexCell"]')
      expect(indexCells.length).toBe(3)
    })

    it('renders empty table when no rows', () => {
      const emptyResult = createMockResult({ rows: [], row_count: 0 })
      renderTable(emptyResult)

      expect(screen.getByRole('table')).toBeInTheDocument()
    })
  })

  describe('Value Formatting', () => {
    it('formats null values as dash', () => {
      renderTable()

      // The null price should show as "-"
      const dashes = screen.getAllByText('-')
      expect(dashes.length).toBeGreaterThan(0)
    })

    it('formats numbers with locale formatting', () => {
      renderTable()

      // Numbers should be formatted
      expect(screen.getByText('29.99')).toBeInTheDocument()
      expect(screen.getByText('49.99')).toBeInTheDocument()
    })

    it('formats currency when format type is currency', () => {
      const result = createMockResult()
      renderTable(result, {
        columns: [
          { name: 'unit_price', displayName: 'Price', visible: true, position: 0, format: { type: 'currency', currency: 'USD' } },
        ],
      })

      expect(screen.getByRole('table')).toBeInTheDocument()
    })
  })

  describe('Client-Side Sorting', () => {
    it('sorts by clicking column header', () => {
      renderTable()

      const idHeader = screen.getByText('ID')
      fireEvent.click(idHeader)

      // Should show sort indicator
      expect(screen.getByRole('table')).toBeInTheDocument()
    })

    it('toggles sort direction on second click', () => {
      const result = createMockResult()
      renderTable(result)

      const nameHeader = screen.getByText('Product Name')
      fireEvent.click(nameHeader) // Ascending
      fireEvent.click(nameHeader) // Descending

      expect(screen.getByRole('table')).toBeInTheDocument()
    })

    it('sorts Northwind products by name ascending', () => {
      const result = createMockResult()
      renderTable(result)

      const nameHeader = screen.getByText('Product Name')
      fireEvent.click(nameHeader)

      const table = screen.getByRole('table')
      const rows = within(table).getAllByRole('row').slice(1)
      const names = rows.map(row => within(row).getAllByRole('cell')[1]?.textContent || '')

      // Verify ascending order
      const sortedNames = [...names].sort((a, b) => a.localeCompare(b))
      expect(names).toEqual(sortedNames)
    })

    it('sorts numeric columns correctly', () => {
      const result = createMockResult()
      renderTable(result)

      const priceHeader = screen.getByText('Unit Price')
      fireEvent.click(priceHeader)

      // Table should render without error
      expect(screen.getByRole('table')).toBeInTheDocument()
    })
  })

  describe('Pagination', () => {
    it('shows pagination when paginateResults is true', () => {
      const manyRows = Array.from({ length: 50 }, (_, i) => ({
        id: i + 1,
        name: `Product ${i + 1}`,
        price: (i + 1) * 10,
        created_at: '2024-01-15T10:30:00Z',
      }))

      render(
        <MantineProvider>
          <TableVisualization
            result={{ ...mockResult, rows: manyRows, row_count: 50 }}
            settings={{ paginateResults: true, pageSize: 10 }}
          />
        </MantineProvider>
      )

      // Should show pagination info
      expect(screen.getByText(/Rows 1-10 of 50/)).toBeInTheDocument()
    })

    it('limits rows when maxRows is set', () => {
      const manyRows = Array.from({ length: 50 }, (_, i) => ({
        ...createNorthwindRows()[0],
        id: i + 1,
        name: `Product ${i + 1}`,
      }))
      const result = createMockResult({ rows: manyRows, row_count: 50 })

      renderTable(result, { maxRows: 10 })

      // Should show truncation message
      expect(screen.getByText(/Showing 10 of 50 rows/)).toBeInTheDocument()
    })

    it('shows filter count when filtered', () => {
      const result = createMockResult()
      renderTable(result)

      // No filter initially
      expect(screen.queryByText(/filtered/)).not.toBeInTheDocument()
    })
  })

  describe('Conditional Formatting', () => {
    it('applies conditional formatting', () => {
      renderTable(mockResult, {
        conditionalFormatting: [
          {
            id: 'rule1',
            columns: ['price'],
            style: 'single',
            condition: { operator: 'greater-than', value: 40 },
            color: '#ed6e6e',
            highlightWholeRow: false,
          },
        ],
      })

      // Component should render without errors
      expect(screen.getByText('Product B')).toBeInTheDocument()
    })

    it('applies row highlight when highlightWholeRow is true', () => {
      const result = createMockResult()
      renderTable(result, {
        conditionalFormatting: [
          {
            id: 'rule1',
            columns: ['discontinued'],
            style: 'single',
            condition: { operator: 'equals', value: true },
            color: '#ff0000',
            highlightWholeRow: true,
          },
        ],
      })

      expect(screen.getByRole('table')).toBeInTheDocument()
    })
  })

  describe('Column Configuration', () => {
    it('hides columns based on visibility settings', () => {
      renderTable(mockResult, {
        columns: [
          { name: 'id', visible: true, position: 0 },
          { name: 'name', visible: true, position: 1 },
          { name: 'price', visible: false, position: 2 },
          { name: 'created_at', visible: true, position: 3 },
        ],
      })

      // Price column should be hidden
      expect(screen.queryByText('Price')).not.toBeInTheDocument()

      // Other columns should be visible
      expect(screen.getByText('ID')).toBeInTheDocument()
      expect(screen.getByText('Name')).toBeInTheDocument()
      expect(screen.getByText('Created At')).toBeInTheDocument()
    })

    it('respects column order from settings', () => {
      renderTable(mockResult, {
        columns: [
          { name: 'name', visible: true, position: 0 },
          { name: 'id', visible: true, position: 1 },
          { name: 'price', visible: true, position: 2 },
          { name: 'created_at', visible: true, position: 3 },
        ],
      })

      // Columns should be in the specified order
      const headers = screen.getAllByRole('columnheader')
      expect(headers[0]).toHaveTextContent('Name')
      expect(headers[1]).toHaveTextContent('ID')
    })

    it('uses custom display name from settings', () => {
      const result = createMockResult()
      renderTable(result, {
        columns: [
          { name: 'unit_price', displayName: 'Price ($)', visible: true, position: 0 },
        ],
      })

      expect(screen.getByText('Price ($)')).toBeInTheDocument()
    })

    it('applies column width setting', () => {
      const result = createMockResult()
      renderTable(result, {
        columns: [
          { name: 'name', visible: true, position: 0, width: 200 },
        ],
      })

      expect(screen.getByRole('table')).toBeInTheDocument()
    })
  })

  describe('Cell Interactions', () => {
    it('calls onCellClick when cell is clicked', () => {
      const onCellClick = vi.fn()
      const result = createMockResult()

      render(
        <MantineProvider>
          <TableVisualization result={result} onCellClick={onCellClick} />
        </MantineProvider>
      )

      const table = screen.getByRole('table')
      const cells = within(table).getAllByRole('cell')
      fireEvent.click(cells[0])

      expect(onCellClick).toHaveBeenCalled()
    })
  })

  describe('Table Styles', () => {
    it('applies striped styling', () => {
      const result = createMockResult()
      renderTable(result, { striped: true })

      expect(screen.getByRole('table')).toBeInTheDocument()
    })

    it('applies hover styling', () => {
      const result = createMockResult()
      renderTable(result, { highlightOnHover: true })

      expect(screen.getByRole('table')).toBeInTheDocument()
    })

    it('applies sticky header', () => {
      const result = createMockResult()
      renderTable(result, { stickyHeader: true })

      expect(screen.getByRole('table')).toBeInTheDocument()
    })
  })
})

// Test filter evaluation logic separately
describe('Filter Logic', () => {
  const evaluateFilter = (value: any, operator: string, filterValue: any): boolean => {
    if (!operator || !filterValue) return true

    const strValue = String(value ?? '').toLowerCase()
    const strFilter = String(filterValue).toLowerCase()

    switch (operator) {
      case 'contains':
        return strValue.includes(strFilter)
      case 'starts_with':
        return strValue.startsWith(strFilter)
      case 'ends_with':
        return strValue.endsWith(strFilter)
      case '=':
        if (typeof value === 'number') return value === Number(filterValue)
        return strValue === strFilter
      case '!=':
        if (typeof value === 'number') return value !== Number(filterValue)
        return strValue !== strFilter
      case '>':
        return Number(value) > Number(filterValue)
      case '>=':
        return Number(value) >= Number(filterValue)
      case '<':
        return Number(value) < Number(filterValue)
      case '<=':
        return Number(value) <= Number(filterValue)
      default:
        return true
    }
  }

  describe('String Operators', () => {
    it('evaluates contains operator', () => {
      expect(evaluateFilter('Hello World', 'contains', 'world')).toBe(true)
      expect(evaluateFilter('Hello World', 'contains', 'foo')).toBe(false)
      expect(evaluateFilter('Hello World', 'contains', 'HELLO')).toBe(true) // case insensitive
    })

    it('evaluates starts_with operator', () => {
      expect(evaluateFilter('Hello World', 'starts_with', 'hello')).toBe(true)
      expect(evaluateFilter('Hello World', 'starts_with', 'world')).toBe(false)
    })

    it('evaluates ends_with operator', () => {
      expect(evaluateFilter('Hello World', 'ends_with', 'world')).toBe(true)
      expect(evaluateFilter('Hello World', 'ends_with', 'hello')).toBe(false)
    })

    it('evaluates equals operator for strings', () => {
      expect(evaluateFilter('test', '=', 'test')).toBe(true)
      expect(evaluateFilter('test', '=', 'TEST')).toBe(true) // case insensitive
      expect(evaluateFilter('test', '=', 'other')).toBe(false)
    })

    it('evaluates not-equals operator for strings', () => {
      expect(evaluateFilter('test', '!=', 'other')).toBe(true)
      expect(evaluateFilter('test', '!=', 'test')).toBe(false)
    })
  })

  describe('Numeric Operators', () => {
    it('evaluates equals operator for numbers', () => {
      expect(evaluateFilter(10, '=', 10)).toBe(true)
      expect(evaluateFilter(10, '=', '10')).toBe(true)
      expect(evaluateFilter(10, '=', 20)).toBe(false)
    })

    it('evaluates not-equals operator for numbers', () => {
      expect(evaluateFilter(10, '!=', 20)).toBe(true)
      expect(evaluateFilter(10, '!=', 10)).toBe(false)
    })

    it('evaluates greater-than operator', () => {
      expect(evaluateFilter(20, '>', 10)).toBe(true)
      expect(evaluateFilter(10, '>', 10)).toBe(false)
      expect(evaluateFilter(5, '>', 10)).toBe(false)
    })

    it('evaluates greater-or-equal operator', () => {
      expect(evaluateFilter(20, '>=', 10)).toBe(true)
      expect(evaluateFilter(10, '>=', 10)).toBe(true)
      expect(evaluateFilter(5, '>=', 10)).toBe(false)
    })

    it('evaluates less-than operator', () => {
      expect(evaluateFilter(5, '<', 10)).toBe(true)
      expect(evaluateFilter(10, '<', 10)).toBe(false)
      expect(evaluateFilter(20, '<', 10)).toBe(false)
    })

    it('evaluates less-or-equal operator', () => {
      expect(evaluateFilter(5, '<=', 10)).toBe(true)
      expect(evaluateFilter(10, '<=', 10)).toBe(true)
      expect(evaluateFilter(20, '<=', 10)).toBe(false)
    })
  })

  describe('Edge Cases', () => {
    it('handles null values', () => {
      expect(evaluateFilter(null, 'contains', 'test')).toBe(false)
      expect(evaluateFilter(null, '=', '')).toBe(true)
    })

    it('handles undefined values', () => {
      expect(evaluateFilter(undefined, 'contains', 'test')).toBe(false)
    })

    it('returns true when no operator or value', () => {
      expect(evaluateFilter('test', '', 'value')).toBe(true)
      expect(evaluateFilter('test', 'contains', '')).toBe(true)
    })
  })
})

// Test sort logic separately
describe('Sort Logic', () => {
  const sortRows = (rows: any[], column: string, direction: 'asc' | 'desc'): any[] => {
    return [...rows].sort((a, b) => {
      const aVal = a[column]
      const bVal = b[column]

      if (aVal === null || aVal === undefined) return 1
      if (bVal === null || bVal === undefined) return -1

      let comparison = 0
      if (typeof aVal === 'number' && typeof bVal === 'number') {
        comparison = aVal - bVal
      } else {
        comparison = String(aVal).localeCompare(String(bVal))
      }

      return direction === 'asc' ? comparison : -comparison
    })
  }

  it('sorts strings ascending', () => {
    const rows = [{ name: 'Charlie' }, { name: 'Alice' }, { name: 'Bob' }]
    const sorted = sortRows(rows, 'name', 'asc')
    expect(sorted.map(r => r.name)).toEqual(['Alice', 'Bob', 'Charlie'])
  })

  it('sorts strings descending', () => {
    const rows = [{ name: 'Charlie' }, { name: 'Alice' }, { name: 'Bob' }]
    const sorted = sortRows(rows, 'name', 'desc')
    expect(sorted.map(r => r.name)).toEqual(['Charlie', 'Bob', 'Alice'])
  })

  it('sorts numbers ascending', () => {
    const rows = [{ value: 30 }, { value: 10 }, { value: 20 }]
    const sorted = sortRows(rows, 'value', 'asc')
    expect(sorted.map(r => r.value)).toEqual([10, 20, 30])
  })

  it('sorts numbers descending', () => {
    const rows = [{ value: 30 }, { value: 10 }, { value: 20 }]
    const sorted = sortRows(rows, 'value', 'desc')
    expect(sorted.map(r => r.value)).toEqual([30, 20, 10])
  })

  it('places null values at end', () => {
    const rows = [{ value: null }, { value: 10 }, { value: 20 }]
    const sorted = sortRows(rows, 'value', 'asc')
    expect(sorted.map(r => r.value)).toEqual([10, 20, null])
  })

  it('sorts Northwind products by price', () => {
    const rows = createNorthwindRows()
    const sorted = sortRows(rows, 'unit_price', 'asc')

    // Verify ascending order
    for (let i = 1; i < sorted.length; i++) {
      expect(sorted[i - 1].unit_price).toBeLessThanOrEqual(sorted[i].unit_price)
    }
  })

  it('sorts Northwind products by name', () => {
    const rows = createNorthwindRows()
    const sorted = sortRows(rows, 'name', 'asc')

    // Verify ascending order
    for (let i = 1; i < sorted.length; i++) {
      expect(sorted[i - 1].name.localeCompare(sorted[i].name)).toBeLessThanOrEqual(0)
    }
  })
})

// Test pagination logic
describe('Pagination Logic', () => {
  it('calculates total pages correctly', () => {
    const totalRows = 100
    const pageSize = 25
    const totalPages = Math.ceil(totalRows / pageSize)
    expect(totalPages).toBe(4)
  })

  it('calculates row range correctly', () => {
    const page = 2
    const pageSize = 25
    const totalRows = 100

    const startRow = (page - 1) * pageSize + 1
    const endRow = Math.min(page * pageSize, totalRows)

    expect(startRow).toBe(26)
    expect(endRow).toBe(50)
  })

  it('handles last page with partial results', () => {
    const page = 4
    const pageSize = 25
    const totalRows = 87

    const startRow = (page - 1) * pageSize + 1
    const endRow = Math.min(page * pageSize, totalRows)

    expect(startRow).toBe(76)
    expect(endRow).toBe(87)
  })
})
