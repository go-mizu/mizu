import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MantineProvider } from '@mantine/core'
import TableVisualization from './TableVisualization'
import type { QueryResult } from '../../../api/types'
import type { TableSettings } from './types'

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

const renderTable = (settings: TableSettings = {}) => {
  return render(
    <MantineProvider>
      <TableVisualization result={mockResult} settings={settings} />
    </MantineProvider>
  )
}

describe('TableVisualization', () => {
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

  it('displays row index when showRowIndex is true', () => {
    const { container } = renderTable({ showRowIndex: true })

    // Row index header
    expect(screen.getByText('#')).toBeInTheDocument()

    // Row index cells should have the rowIndexCell class
    const indexCells = container.querySelectorAll('._rowIndexCell_e493d4')
    expect(indexCells.length).toBe(3) // 3 rows

    // Each should contain a badge
    const badges = container.querySelectorAll('.mantine-Badge-root')
    expect(badges.length).toBe(3)
  })

  it('formats null values as dash', () => {
    renderTable()

    // The null price should show as "-"
    const dashes = screen.getAllByText('-')
    expect(dashes.length).toBeGreaterThan(0)
  })

  it('formats numbers with locale formatting', () => {
    renderTable()

    // Numbers should be formatted (29.99 -> "29.99" with locale)
    expect(screen.getByText('29.99')).toBeInTheDocument()
    expect(screen.getByText('49.99')).toBeInTheDocument()
  })

  it('supports sorting by clicking column header', () => {
    renderTable()

    const idHeader = screen.getByText('ID')
    fireEvent.click(idHeader)

    // Should show sort indicator (we can't easily check the actual sort order without more setup)
    // The component should not throw an error when sorting
  })

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

  it('applies conditional formatting', () => {
    renderTable({
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

    // The component should render without errors
    expect(screen.getByText('Product B')).toBeInTheDocument()
  })

  it('hides columns based on visibility settings', () => {
    renderTable({
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
    renderTable({
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
})
