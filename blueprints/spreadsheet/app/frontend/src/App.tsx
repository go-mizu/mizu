import { useCallback, useState } from 'react'
import DataEditor, {
  GridCell,
  GridCellKind,
  GridColumn,
  Item,
  EditableGridCell,
} from '@glideapps/glide-data-grid'

// Generate column headers A, B, C, ..., Z, AA, AB, etc.
function getColumnLabel(index: number): string {
  let result = ''
  index++
  while (index > 0) {
    index--
    result = String.fromCharCode(65 + (index % 26)) + result
    index = Math.floor(index / 26)
  }
  return result
}

// Number of rows and columns
const NUM_ROWS = 1000
const NUM_COLS = 26

// Generate columns
const columns: GridColumn[] = Array.from({ length: NUM_COLS }, (_, i) => ({
  id: getColumnLabel(i),
  title: getColumnLabel(i),
  width: 100,
}))

// Cell data storage
type CellData = {
  value: string
  formula?: string
}

function App() {
  const [data, setData] = useState<Map<string, CellData>>(new Map())

  // Get cell key
  const getCellKey = (col: number, row: number) => `${col}:${row}`

  // Get cell content
  const getContent = useCallback(
    (cell: Item): GridCell => {
      const [col, row] = cell
      const key = getCellKey(col, row)
      const cellData = data.get(key)

      return {
        kind: GridCellKind.Text,
        allowOverlay: true,
        displayData: cellData?.value ?? '',
        data: cellData?.value ?? '',
      }
    },
    [data]
  )

  // Handle cell edit
  const onCellEdited = useCallback(
    (cell: Item, newValue: EditableGridCell) => {
      if (newValue.kind !== GridCellKind.Text) return

      const [col, row] = cell
      const key = getCellKey(col, row)

      setData((prev) => {
        const newData = new Map(prev)
        if (newValue.data === '') {
          newData.delete(key)
        } else {
          newData.set(key, {
            value: newValue.data,
            formula: newValue.data.startsWith('=') ? newValue.data : undefined,
          })
        }
        return newData
      })

      // TODO: Send to backend
      // const value = newValue.data
      // const isFormula = value.startsWith('=')
      // api.setCellValue(sheetId, row, col, isFormula ? undefined : value, isFormula ? value : undefined)
    },
    []
  )

  return (
    <div className="app">
      <header className="header">
        <div className="header-left">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="logo">
            <rect x="3" y="3" width="18" height="18" rx="2" />
            <path d="M3 9h18M9 3v18" />
          </svg>
          <span className="title">Untitled Spreadsheet</span>
        </div>
        <div className="header-right">
          <button className="share-btn">Share</button>
        </div>
      </header>

      <div className="toolbar">
        <div className="toolbar-group">
          <button title="Undo">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M3 10h10a5 5 0 0 1 5 5v2M3 10l5-5M3 10l5 5" />
            </svg>
          </button>
          <button title="Redo">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M21 10H11a5 5 0 0 0-5 5v2M21 10l-5-5M21 10l-5 5" />
            </svg>
          </button>
        </div>
        <div className="toolbar-divider" />
        <div className="toolbar-group">
          <select className="font-select" defaultValue="Arial">
            <option value="Arial">Arial</option>
            <option value="Helvetica">Helvetica</option>
            <option value="Times New Roman">Times New Roman</option>
            <option value="Courier New">Courier New</option>
          </select>
          <select className="size-select" defaultValue="10">
            <option value="8">8</option>
            <option value="9">9</option>
            <option value="10">10</option>
            <option value="11">11</option>
            <option value="12">12</option>
            <option value="14">14</option>
            <option value="18">18</option>
            <option value="24">24</option>
          </select>
        </div>
        <div className="toolbar-divider" />
        <div className="toolbar-group">
          <button title="Bold" className="format-btn">
            <strong>B</strong>
          </button>
          <button title="Italic" className="format-btn">
            <em>I</em>
          </button>
          <button title="Underline" className="format-btn">
            <u>U</u>
          </button>
        </div>
      </div>

      <div className="formula-bar">
        <div className="cell-ref">A1</div>
        <div className="formula-input-container">
          <span className="fx">fx</span>
          <input type="text" className="formula-input" placeholder="Enter formula or value" />
        </div>
      </div>

      <div className="grid-container">
        <DataEditor
          getCellContent={getContent}
          columns={columns}
          rows={NUM_ROWS}
          onCellEdited={onCellEdited}
          rowMarkers="number"
          smoothScrollX
          smoothScrollY
          theme={{
            bgCell: '#ffffff',
            bgHeader: '#f8f9fa',
            bgHeaderHovered: '#e8eaed',
            textHeader: '#5f6368',
            borderColor: '#e2e2e2',
            accentColor: '#1a73e8',
            accentLight: '#e8f0fe',
          }}
        />
      </div>

      <footer className="sheet-tabs">
        <button className="add-sheet">+</button>
        <div className="sheet-tab active">Sheet1</div>
        <div className="sheet-tab">Sheet2</div>
        <div className="sheet-tab">Sheet3</div>
      </footer>
    </div>
  )
}

export default App
