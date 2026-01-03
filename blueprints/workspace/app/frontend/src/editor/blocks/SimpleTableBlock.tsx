import { createReactBlockSpec } from '@blocknote/react'
import { useState, useCallback, useRef, useEffect } from 'react'
import { Plus, Trash2, GripVertical } from 'lucide-react'

interface TableCell {
  content: string
}

interface TableRow {
  cells: TableCell[]
}

interface TableData {
  rows: TableRow[]
  hasHeader: boolean
}

export const SimpleTableBlock = createReactBlockSpec(
  {
    type: 'simpleTable',
    propSchema: {
      tableData: {
        default: JSON.stringify({
          rows: [
            { cells: [{ content: '' }, { content: '' }, { content: '' }] },
            { cells: [{ content: '' }, { content: '' }, { content: '' }] },
            { cells: [{ content: '' }, { content: '' }, { content: '' }] },
          ],
          hasHeader: true,
        }),
      },
    },
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [tableData, setTableData] = useState<TableData>(() => {
        try {
          return JSON.parse(block.props.tableData as string)
        } catch {
          return {
            rows: [
              { cells: [{ content: '' }, { content: '' }, { content: '' }] },
              { cells: [{ content: '' }, { content: '' }, { content: '' }] },
            ],
            hasHeader: true,
          }
        }
      })

      const [selectedCell, setSelectedCell] = useState<{ row: number; col: number } | null>(null)
      const [hoveredRow, setHoveredRow] = useState<number | null>(null)
      const [hoveredCol, setHoveredCol] = useState<number | null>(null)
      const tableRef = useRef<HTMLTableElement>(null)

      // Sync to block props
      const updateBlockData = useCallback((newData: TableData) => {
        setTableData(newData)
        editor.updateBlock(block, {
          props: { ...block.props, tableData: JSON.stringify(newData) },
        })
      }, [block, editor])

      // Update cell content
      const updateCell = useCallback((rowIndex: number, colIndex: number, content: string) => {
        const newData = { ...tableData }
        newData.rows = [...newData.rows]
        newData.rows[rowIndex] = { ...newData.rows[rowIndex], cells: [...newData.rows[rowIndex].cells] }
        newData.rows[rowIndex].cells[colIndex] = { content }
        updateBlockData(newData)
      }, [tableData, updateBlockData])

      // Add row
      const addRow = useCallback((afterIndex: number) => {
        const colCount = tableData.rows[0]?.cells.length || 3
        const newRow: TableRow = {
          cells: Array(colCount).fill(null).map(() => ({ content: '' })),
        }
        const newRows = [...tableData.rows]
        newRows.splice(afterIndex + 1, 0, newRow)
        updateBlockData({ ...tableData, rows: newRows })
      }, [tableData, updateBlockData])

      // Delete row
      const deleteRow = useCallback((rowIndex: number) => {
        if (tableData.rows.length <= 1) return
        const newRows = tableData.rows.filter((_, i) => i !== rowIndex)
        updateBlockData({ ...tableData, rows: newRows })
      }, [tableData, updateBlockData])

      // Add column
      const addColumn = useCallback((afterIndex: number) => {
        const newRows = tableData.rows.map(row => ({
          cells: [
            ...row.cells.slice(0, afterIndex + 1),
            { content: '' },
            ...row.cells.slice(afterIndex + 1),
          ],
        }))
        updateBlockData({ ...tableData, rows: newRows })
      }, [tableData, updateBlockData])

      // Delete column
      const deleteColumn = useCallback((colIndex: number) => {
        if (tableData.rows[0]?.cells.length <= 1) return
        const newRows = tableData.rows.map(row => ({
          cells: row.cells.filter((_, i) => i !== colIndex),
        }))
        updateBlockData({ ...tableData, rows: newRows })
      }, [tableData, updateBlockData])

      // Toggle header
      const toggleHeader = useCallback(() => {
        updateBlockData({ ...tableData, hasHeader: !tableData.hasHeader })
      }, [tableData, updateBlockData])

      // Handle keyboard navigation
      const handleKeyDown = useCallback((e: React.KeyboardEvent, rowIndex: number, colIndex: number) => {
        if (e.key === 'Tab') {
          e.preventDefault()
          const nextCol = e.shiftKey ? colIndex - 1 : colIndex + 1
          const colCount = tableData.rows[0]?.cells.length || 0

          if (nextCol >= 0 && nextCol < colCount) {
            setSelectedCell({ row: rowIndex, col: nextCol })
          } else if (!e.shiftKey && rowIndex < tableData.rows.length - 1) {
            setSelectedCell({ row: rowIndex + 1, col: 0 })
          } else if (e.shiftKey && rowIndex > 0) {
            setSelectedCell({ row: rowIndex - 1, col: colCount - 1 })
          }
        } else if (e.key === 'Enter' && !e.shiftKey) {
          e.preventDefault()
          if (rowIndex < tableData.rows.length - 1) {
            setSelectedCell({ row: rowIndex + 1, col: colIndex })
          }
        } else if (e.key === 'ArrowUp' && rowIndex > 0) {
          e.preventDefault()
          setSelectedCell({ row: rowIndex - 1, col: colIndex })
        } else if (e.key === 'ArrowDown' && rowIndex < tableData.rows.length - 1) {
          e.preventDefault()
          setSelectedCell({ row: rowIndex + 1, col: colIndex })
        }
      }, [tableData])

      // Auto-focus selected cell
      useEffect(() => {
        if (selectedCell && tableRef.current) {
          const input = tableRef.current.querySelector(
            `input[data-row="${selectedCell.row}"][data-col="${selectedCell.col}"]`
          ) as HTMLInputElement
          input?.focus()
        }
      }, [selectedCell])

      const colCount = tableData.rows[0]?.cells.length || 3

      return (
        <div
          className="simple-table-block"
          style={{
            margin: '8px 0',
            borderRadius: '4px',
            overflow: 'hidden',
            border: '1px solid var(--border-color, rgba(55, 53, 47, 0.16))',
          }}
        >
          {/* Table toolbar */}
          <div
            className="table-toolbar"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              padding: '4px 8px',
              borderBottom: '1px solid var(--border-color, rgba(55, 53, 47, 0.16))',
              background: 'var(--bg-secondary, #f7f6f3)',
              fontSize: '12px',
            }}
          >
            <label
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
                cursor: 'pointer',
                color: 'var(--text-secondary, #787774)',
              }}
            >
              <input
                type="checkbox"
                checked={tableData.hasHeader}
                onChange={toggleHeader}
                style={{ margin: 0 }}
              />
              <span>Header row</span>
            </label>
          </div>

          {/* Column controls */}
          <div
            style={{
              display: 'flex',
              paddingLeft: '28px',
            }}
          >
            {tableData.rows[0]?.cells.map((_, colIndex) => (
              <div
                key={colIndex}
                style={{
                  flex: 1,
                  minWidth: '100px',
                  display: 'flex',
                  justifyContent: 'center',
                  padding: '2px',
                  opacity: hoveredCol === colIndex ? 1 : 0,
                  transition: 'opacity 0.15s',
                }}
                onMouseEnter={() => setHoveredCol(colIndex)}
                onMouseLeave={() => setHoveredCol(null)}
              >
                <button
                  onClick={() => deleteColumn(colIndex)}
                  title="Delete column"
                  style={{
                    padding: '2px 4px',
                    background: 'none',
                    border: 'none',
                    color: 'var(--danger-color, #e03e3e)',
                    cursor: 'pointer',
                    borderRadius: '2px',
                  }}
                >
                  <Trash2 size={12} />
                </button>
              </div>
            ))}
          </div>

          {/* Table */}
          <table
            ref={tableRef}
            style={{
              width: '100%',
              borderCollapse: 'collapse',
              tableLayout: 'fixed',
            }}
          >
            <tbody>
              {tableData.rows.map((row, rowIndex) => (
                <tr
                  key={rowIndex}
                  onMouseEnter={() => setHoveredRow(rowIndex)}
                  onMouseLeave={() => setHoveredRow(null)}
                  style={{
                    background: tableData.hasHeader && rowIndex === 0
                      ? 'var(--bg-secondary, #f7f6f3)'
                      : 'transparent',
                  }}
                >
                  {/* Row controls */}
                  <td
                    style={{
                      width: '28px',
                      padding: '4px',
                      verticalAlign: 'middle',
                    }}
                  >
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '2px',
                        opacity: hoveredRow === rowIndex ? 1 : 0,
                        transition: 'opacity 0.15s',
                      }}
                    >
                      <button
                        onClick={() => deleteRow(rowIndex)}
                        title="Delete row"
                        style={{
                          padding: '2px',
                          background: 'none',
                          border: 'none',
                          color: 'var(--danger-color, #e03e3e)',
                          cursor: 'pointer',
                          borderRadius: '2px',
                        }}
                      >
                        <Trash2 size={12} />
                      </button>
                    </div>
                  </td>

                  {/* Cells */}
                  {row.cells.map((cell, colIndex) => (
                    <td
                      key={colIndex}
                      onMouseEnter={() => setHoveredCol(colIndex)}
                      onMouseLeave={() => setHoveredCol(null)}
                      style={{
                        border: '1px solid var(--border-color, rgba(55, 53, 47, 0.16))',
                        padding: 0,
                        minWidth: '100px',
                      }}
                    >
                      <input
                        type="text"
                        data-row={rowIndex}
                        data-col={colIndex}
                        value={cell.content}
                        onChange={(e) => updateCell(rowIndex, colIndex, e.target.value)}
                        onKeyDown={(e) => handleKeyDown(e, rowIndex, colIndex)}
                        onFocus={() => setSelectedCell({ row: rowIndex, col: colIndex })}
                        style={{
                          width: '100%',
                          padding: '8px 10px',
                          border: 'none',
                          outline: 'none',
                          background: 'transparent',
                          fontSize: '14px',
                          fontWeight: tableData.hasHeader && rowIndex === 0 ? 600 : 400,
                          color: 'var(--text-primary, #37352f)',
                        }}
                        placeholder={tableData.hasHeader && rowIndex === 0 ? 'Header' : ''}
                      />
                    </td>
                  ))}

                  {/* Add column button (only on first row) */}
                  {rowIndex === 0 && (
                    <td
                      style={{
                        width: '28px',
                        padding: '4px',
                        verticalAlign: 'middle',
                      }}
                    >
                      <button
                        onClick={() => addColumn(colCount - 1)}
                        title="Add column"
                        style={{
                          padding: '4px',
                          background: 'none',
                          border: 'none',
                          color: 'var(--text-tertiary, #9b9a97)',
                          cursor: 'pointer',
                          borderRadius: '4px',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                        }}
                      >
                        <Plus size={14} />
                      </button>
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>

          {/* Add row button */}
          <button
            onClick={() => addRow(tableData.rows.length - 1)}
            style={{
              width: '100%',
              padding: '6px',
              background: 'none',
              border: 'none',
              borderTop: '1px solid var(--border-color, rgba(55, 53, 47, 0.16))',
              color: 'var(--text-tertiary, #9b9a97)',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '4px',
              fontSize: '12px',
              transition: 'background 0.15s, color 0.15s',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = 'var(--bg-hover, rgba(55, 53, 47, 0.08))'
              e.currentTarget.style.color = 'var(--text-secondary, #787774)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = 'none'
              e.currentTarget.style.color = 'var(--text-tertiary, #9b9a97)'
            }}
          >
            <Plus size={14} />
            <span>New row</span>
          </button>
        </div>
      )
    },
    // Parse HTML to recreate block when pasting or drag-dropping
    parse: (element: HTMLElement) => {
      if (element.classList.contains('simple-table-block') || element.hasAttribute('data-table-data')) {
        return {
          tableData: element.getAttribute('data-table-data') || JSON.stringify({
            rows: [
              { cells: [{ content: '' }, { content: '' }, { content: '' }] },
              { cells: [{ content: '' }, { content: '' }, { content: '' }] },
            ],
            hasHeader: true,
          }),
        }
      }
      return undefined
    },
    // Convert to external HTML for clipboard/export
    toExternalHTML: ({ block }) => {
      const tableDataStr = (block.props.tableData as string) || '{"rows":[],"hasHeader":true}'
      let tableData: { rows: Array<{ cells: Array<{ content: string }> }>; hasHeader: boolean }
      try {
        tableData = JSON.parse(tableDataStr)
      } catch {
        tableData = { rows: [], hasHeader: true }
      }

      return (
        <table
          className="simple-table-block"
          data-table-data={tableDataStr}
          style={{
            width: '100%',
            borderCollapse: 'collapse',
            border: '1px solid rgba(55, 53, 47, 0.16)',
          }}
        >
          <tbody>
            {tableData.rows.map((row, rowIndex) => (
              <tr key={rowIndex}>
                {row.cells.map((cell, colIndex) => {
                  const Tag = tableData.hasHeader && rowIndex === 0 ? 'th' : 'td'
                  return (
                    <Tag
                      key={colIndex}
                      style={{
                        padding: '8px 12px',
                        border: '1px solid rgba(55, 53, 47, 0.16)',
                        fontWeight: tableData.hasHeader && rowIndex === 0 ? 600 : 400,
                        background: tableData.hasHeader && rowIndex === 0 ? '#f7f6f3' : 'transparent',
                      }}
                    >
                      {cell.content}
                    </Tag>
                  )
                })}
              </tr>
            ))}
          </tbody>
        </table>
      )
    },
  }
)
