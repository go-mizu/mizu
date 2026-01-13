import { describe, it, expect, vi } from 'vitest'
import { renderWithProviders, screen, userEvent } from '../../test/utils'
import { DataTable, type Column } from '../../components/common/DataTable'
import { IconTrash, IconEdit } from '@tabler/icons-react'

interface TestRow {
  id: string
  name: string
  count: number
  status: 'active' | 'inactive'
}

const testData: TestRow[] = [
  { id: '1', name: 'Alpha', count: 100, status: 'active' },
  { id: '2', name: 'Beta', count: 200, status: 'inactive' },
  { id: '3', name: 'Gamma', count: 50, status: 'active' },
  { id: '4', name: 'Delta', count: 150, status: 'inactive' },
  { id: '5', name: 'Epsilon', count: 75, status: 'active' },
]

const columns: Column<TestRow>[] = [
  { key: 'name', label: 'Name', sortable: true },
  { key: 'count', label: 'Count', sortable: true },
  { key: 'status', label: 'Status' },
]

describe('DataTable', () => {
  describe('rendering', () => {
    it('renders table with data', () => {
      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
        />
      )

      expect(screen.getByText('Name')).toBeInTheDocument()
      expect(screen.getByText('Count')).toBeInTheDocument()
      expect(screen.getByText('Status')).toBeInTheDocument()

      expect(screen.getByText('Alpha')).toBeInTheDocument()
      expect(screen.getByText('Beta')).toBeInTheDocument()
      expect(screen.getByText('100')).toBeInTheDocument()
    })

    it('renders custom cell content with render function', () => {
      const columnsWithRender: Column<TestRow>[] = [
        ...columns.slice(0, 2),
        {
          key: 'status',
          label: 'Status',
          render: (row) => <span data-testid={`status-${row.id}`}>{row.status.toUpperCase()}</span>,
        },
      ]

      renderWithProviders(
        <DataTable
          data={testData}
          columns={columnsWithRender}
          getRowKey={(row) => row.id}
        />
      )

      expect(screen.getByTestId('status-1')).toHaveTextContent('ACTIVE')
      expect(screen.getByTestId('status-2')).toHaveTextContent('INACTIVE')
    })

    it('hides columns marked as hidden', () => {
      const columnsWithHidden: Column<TestRow>[] = [
        columns[0],
        { ...columns[1], hidden: true },
        columns[2],
      ]

      renderWithProviders(
        <DataTable
          data={testData}
          columns={columnsWithHidden}
          getRowKey={(row) => row.id}
        />
      )

      expect(screen.getByText('Name')).toBeInTheDocument()
      expect(screen.queryByText('Count')).not.toBeInTheDocument()
      expect(screen.getByText('Status')).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    it('shows empty state when data is empty', () => {
      renderWithProviders(
        <DataTable
          data={[]}
          columns={columns}
          getRowKey={(row) => row.id}
          emptyState={{
            title: 'No items found',
            description: 'Create your first item to get started',
          }}
        />
      )

      expect(screen.getByText('No items found')).toBeInTheDocument()
      expect(screen.getByText('Create your first item to get started')).toBeInTheDocument()
    })

    it('shows empty state with action button', async () => {
      const onAction = vi.fn()
      const user = userEvent.setup()

      renderWithProviders(
        <DataTable
          data={[]}
          columns={columns}
          getRowKey={(row) => row.id}
          emptyState={{
            title: 'No items',
            action: {
              label: 'Create Item',
              onClick: onAction,
            },
          }}
        />
      )

      const button = screen.getByText('Create Item')
      await user.click(button)

      expect(onAction).toHaveBeenCalled()
    })
  })

  describe('loading state', () => {
    it('shows loading state when loading', () => {
      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          loading
        />
      )

      expect(screen.queryByText('Alpha')).not.toBeInTheDocument()
    })
  })

  describe('search functionality', () => {
    it('renders search input by default', () => {
      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
        />
      )

      expect(screen.getByPlaceholderText('Search...')).toBeInTheDocument()
    })

    it('filters data based on search', async () => {
      const user = userEvent.setup()

      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
        />
      )

      const searchInput = screen.getByPlaceholderText('Search...')
      await user.type(searchInput, 'Alpha')

      expect(screen.getByText('Alpha')).toBeInTheDocument()
      expect(screen.queryByText('Beta')).not.toBeInTheDocument()
      expect(screen.queryByText('Gamma')).not.toBeInTheDocument()
    })

    it('shows no results message when search has no matches', async () => {
      const user = userEvent.setup()

      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
        />
      )

      const searchInput = screen.getByPlaceholderText('Search...')
      await user.type(searchInput, 'xyz')

      expect(screen.getByText(/No results found for "xyz"/)).toBeInTheDocument()
    })

    it('can disable search', () => {
      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          searchable={false}
        />
      )

      expect(screen.queryByPlaceholderText('Search...')).not.toBeInTheDocument()
    })

    it('uses custom search placeholder', () => {
      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          searchPlaceholder="Find items..."
        />
      )

      expect(screen.getByPlaceholderText('Find items...')).toBeInTheDocument()
    })
  })

  describe('sorting', () => {
    it('sorts by column when clicking sortable header', async () => {
      const user = userEvent.setup()

      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
        />
      )

      const nameHeader = screen.getByText('Name')
      await user.click(nameHeader)

      // After sorting, check the first cell
      const rows = screen.getAllByRole('row')
      // First row is header, second is first data row
      expect(rows[1]).toHaveTextContent('Alpha')
    })

    it('toggles sort direction on second click', async () => {
      const user = userEvent.setup()

      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
        />
      )

      const nameHeader = screen.getByText('Name')

      // First click - ascending
      await user.click(nameHeader)
      // Second click - descending
      await user.click(nameHeader)

      const rows = screen.getAllByRole('row')
      // First data row should now be last alphabetically
      expect(rows[1]).toHaveTextContent('Gamma')
    })
  })

  describe('row actions', () => {
    it('renders action menu for each row', () => {
      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          actions={[
            { label: 'Edit', icon: <IconEdit size={14} />, onClick: () => {} },
            { label: 'Delete', icon: <IconTrash size={14} />, onClick: () => {}, color: 'red' },
          ]}
        />
      )

      // Action menu triggers should be present for each row
      const menuTriggers = screen.getAllByRole('button')
      expect(menuTriggers.length).toBeGreaterThanOrEqual(testData.length)
    })

    it('calls action onClick when clicked', async () => {
      const onEdit = vi.fn()
      const user = userEvent.setup()

      renderWithProviders(
        <DataTable
          data={testData.slice(0, 1)} // Just one row for simplicity
          columns={columns}
          getRowKey={(row) => row.id}
          actions={[{ label: 'Edit', onClick: onEdit }]}
        />
      )

      // Open menu
      const menuTrigger = screen.getByRole('button')
      await user.click(menuTrigger)

      // Click action
      const editItem = await screen.findByText('Edit')
      await user.click(editItem)

      expect(onEdit).toHaveBeenCalledWith(testData[0])
    })
  })

  describe('row click', () => {
    it('calls onRowClick when row is clicked', async () => {
      const onRowClick = vi.fn()
      const user = userEvent.setup()

      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          onRowClick={onRowClick}
        />
      )

      const firstRow = screen.getByText('Alpha').closest('tr')
      if (firstRow) {
        await user.click(firstRow)
        expect(onRowClick).toHaveBeenCalledWith(testData[0])
      }
    })
  })

  describe('bulk actions', () => {
    it('renders checkboxes when bulkActions provided', () => {
      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          bulkActions={[{ label: 'Delete All', onClick: () => {} }]}
        />
      )

      const checkboxes = screen.getAllByRole('checkbox')
      // Header checkbox + one per row
      expect(checkboxes.length).toBe(testData.length + 1)
    })

    it('selects individual rows', async () => {
      const user = userEvent.setup()

      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          bulkActions={[{ label: 'Delete', onClick: () => {} }]}
        />
      )

      const checkboxes = screen.getAllByRole('checkbox')
      // Click first data row checkbox (index 1)
      await user.click(checkboxes[1])

      expect(screen.getByText('1 item selected')).toBeInTheDocument()
    })

    it('selects all rows with header checkbox', async () => {
      const user = userEvent.setup()

      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          bulkActions={[{ label: 'Delete', onClick: () => {} }]}
        />
      )

      const headerCheckbox = screen.getAllByRole('checkbox')[0]
      await user.click(headerCheckbox)

      expect(screen.getByText(`${testData.length} items selected`)).toBeInTheDocument()
    })

    it('calls bulk action with selected rows', async () => {
      const onDelete = vi.fn()
      const user = userEvent.setup()

      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          bulkActions={[{ label: 'Delete', onClick: onDelete }]}
        />
      )

      // Select first two rows
      const checkboxes = screen.getAllByRole('checkbox')
      await user.click(checkboxes[1])
      await user.click(checkboxes[2])

      // Click bulk action
      const deleteButton = screen.getByText('Delete')
      await user.click(deleteButton)

      expect(onDelete).toHaveBeenCalledWith([testData[0], testData[1]])
    })

    it('clears selection with clear button', async () => {
      const user = userEvent.setup()

      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          bulkActions={[{ label: 'Delete', onClick: () => {} }]}
        />
      )

      // Select a row
      const checkboxes = screen.getAllByRole('checkbox')
      await user.click(checkboxes[1])

      expect(screen.getByText('1 item selected')).toBeInTheDocument()

      // Clear selection
      const clearButton = screen.getByText('Clear')
      await user.click(clearButton)

      expect(screen.queryByText('1 item selected')).not.toBeInTheDocument()
    })
  })

  describe('pagination', () => {
    it('renders pagination when config provided', () => {
      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          pagination={{
            page: 1,
            pageSize: 2,
            total: testData.length,
            onPageChange: () => {},
          }}
        />
      )

      // Should show pagination controls
      expect(screen.getByText(/Showing 1-2 of 5/)).toBeInTheDocument()
    })

    it('calls onPageChange when page changes', async () => {
      const onPageChange = vi.fn()
      const user = userEvent.setup()

      renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          pagination={{
            page: 1,
            pageSize: 2,
            total: testData.length,
            onPageChange,
          }}
        />
      )

      // Click next page
      const nextButton = screen.getByRole('button', { name: /2/ })
      await user.click(nextButton)

      expect(onPageChange).toHaveBeenCalledWith(2)
    })
  })

  describe('toolbar actions', () => {
    it('renders refresh button when onRefresh provided', () => {
      const { container } = renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          onRefresh={() => {}}
        />
      )

      // Find refresh icon by class
      const refreshIcon = container.querySelector('.tabler-icon-refresh')
      expect(refreshIcon).toBeInTheDocument()
    })

    it('calls onRefresh when refresh clicked', async () => {
      const onRefresh = vi.fn()
      const user = userEvent.setup()

      const { container } = renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          onRefresh={onRefresh}
        />
      )

      const refreshButton = container.querySelector('.tabler-icon-refresh')?.closest('button')
      if (refreshButton) await user.click(refreshButton)

      expect(onRefresh).toHaveBeenCalled()
    })

    it('renders export button when exportable', () => {
      const { container } = renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          exportable
          onExport={() => {}}
        />
      )

      // Find download icon by class
      const downloadIcon = container.querySelector('.tabler-icon-download')
      expect(downloadIcon).toBeInTheDocument()
    })

    it('calls onExport when export clicked', async () => {
      const onExport = vi.fn()
      const user = userEvent.setup()

      const { container } = renderWithProviders(
        <DataTable
          data={testData}
          columns={columns}
          getRowKey={(row) => row.id}
          exportable
          onExport={onExport}
        />
      )

      const exportButton = container.querySelector('.tabler-icon-download')?.closest('button')
      if (exportButton) await user.click(exportButton)

      expect(onExport).toHaveBeenCalled()
    })
  })
})
