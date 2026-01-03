import { useCallback, useMemo, useState, useRef, useEffect } from 'react'
import DataEditor, {
  GridColumn,
  GridCell,
  GridCellKind,
  EditableGridCell,
  Item,
  Rectangle,
  CompactSelection,
  GridSelection,
  CellClickedEventArgs,
  Theme,
} from '@glideapps/glide-data-grid'
import '@glideapps/glide-data-grid/dist/index.css'
import { DatabaseRow, Property, PropertyType, PropertyOption, Database, api } from '../../api/client'
import { Plus, Trash2 } from 'lucide-react'

interface DataGridProps {
  rows: DatabaseRow[]
  properties: Property[]
  database?: Database
  hiddenProperties?: string[]
  onAddRow: (initialProperties?: Record<string, unknown>) => Promise<DatabaseRow | null>
  onUpdateRow: (rowId: string, updates: Record<string, unknown>) => void
  onDeleteRow: (rowId: string) => void
  onAddProperty?: (property: Omit<Property, 'id'>) => void
  onUpdateProperty?: (propertyId: string, updates: Partial<Property>) => void
  onDeleteProperty?: (propertyId: string) => void
  onHiddenPropertiesChange?: (hiddenProperties: string[]) => void
}

// Map property types to Glide Data Grid cell kinds
function getCellKind(type: PropertyType): GridCellKind {
  switch (type) {
    case 'text':
    case 'email':
    case 'phone':
      return GridCellKind.Text
    case 'number':
      return GridCellKind.Number
    case 'checkbox':
      return GridCellKind.Boolean
    case 'url':
      return GridCellKind.Uri
    case 'select':
    case 'status':
      return GridCellKind.Text // Will use custom rendering
    case 'multi_select':
      return GridCellKind.Bubble
    case 'date':
    case 'created_time':
    case 'last_edited_time':
      return GridCellKind.Text // Date as text for now
    case 'person':
    case 'created_by':
    case 'last_edited_by':
      return GridCellKind.Text
    case 'files':
      return GridCellKind.Text
    case 'relation':
      return GridCellKind.Bubble
    case 'rollup':
    case 'formula':
      return GridCellKind.Text
    default:
      return GridCellKind.Text
  }
}

// Format date for display
function formatDate(value: unknown): string {
  if (!value) return ''
  try {
    const dateStr = typeof value === 'object' && value !== null && 'start' in value
      ? (value as { start: string }).start
      : String(value)
    const date = new Date(dateStr)
    if (isNaN(date.getTime())) return String(value)
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  } catch {
    return String(value)
  }
}

// Format select option for display with color tag
function getSelectDisplayValue(value: unknown, options: PropertyOption[]): string {
  if (!value) return ''
  const option = options.find(o => o.id === value)
  return option?.name || String(value)
}

// Format multi-select values for bubble display
function getMultiSelectValues(value: unknown, options: PropertyOption[]): string[] {
  if (!value || !Array.isArray(value)) return []
  return value
    .map(v => options.find(o => o.id === v)?.name || String(v))
    .filter(Boolean)
}

// Custom theme matching our app design
const customTheme: Partial<Theme> = {
  accentColor: 'var(--accent-color)',
  accentLight: 'var(--accent-bg)',
  textDark: 'var(--text-primary)',
  textMedium: 'var(--text-secondary)',
  textLight: 'var(--text-tertiary)',
  bgCell: 'var(--bg-primary)',
  bgCellMedium: 'var(--bg-secondary)',
  bgHeader: 'var(--bg-secondary)',
  bgHeaderHasFocus: 'var(--bg-tertiary)',
  bgHeaderHovered: 'var(--bg-tertiary)',
  borderColor: 'var(--border-color)',
  fontFamily: 'var(--font-sans, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif)',
  baseFontStyle: '14px',
  headerFontStyle: '600 13px',
  editorFontSize: '14px',
  lineHeight: 1.5,
}

export function DataGrid({
  rows,
  properties,
  hiddenProperties = [],
  onAddRow,
  onUpdateRow,
  onDeleteRow,
}: DataGridProps) {
  const [selection, setSelection] = useState<GridSelection>({
    columns: CompactSelection.empty(),
    rows: CompactSelection.empty(),
  })
  const gridRef = useRef<any>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const [gridHeight, setGridHeight] = useState(400)

  // Calculate grid height based on container
  useEffect(() => {
    const updateHeight = () => {
      if (containerRef.current) {
        const rect = containerRef.current.getBoundingClientRect()
        const availableHeight = window.innerHeight - rect.top - 100 // Leave room for footer
        setGridHeight(Math.max(300, availableHeight))
      }
    }
    updateHeight()
    window.addEventListener('resize', updateHeight)
    return () => window.removeEventListener('resize', updateHeight)
  }, [])

  // Get visible properties
  const visibleProperties = useMemo(() => {
    return properties.filter(p => !hiddenProperties.includes(p.id))
  }, [properties, hiddenProperties])

  // Create columns from properties
  const columns: GridColumn[] = useMemo(() => {
    return visibleProperties.map(prop => ({
      id: prop.id,
      title: prop.name,
      width: 180,
      grow: prop.type === 'text' ? 1 : 0,
      themeOverride: {
        // Override for different column types if needed
      },
    }))
  }, [visibleProperties])

  // Get cell content for a given location
  const getCellContent = useCallback((cell: Item): GridCell => {
    const [col, row] = cell
    if (row >= rows.length || col >= visibleProperties.length) {
      return { kind: GridCellKind.Text, data: '', displayData: '', allowOverlay: false }
    }

    const rowData = rows[row]
    const property = visibleProperties[col]
    const value = rowData.properties[property.id]
    const isReadOnly = ['created_time', 'created_by', 'last_edited_time', 'last_edited_by', 'rollup', 'formula'].includes(property.type)

    switch (property.type) {
      case 'text':
      case 'email':
      case 'phone':
        return {
          kind: GridCellKind.Text,
          data: value ? String(value) : '',
          displayData: value ? String(value) : '',
          allowOverlay: true,
          readonly: false,
        }

      case 'number':
        return {
          kind: GridCellKind.Number,
          data: typeof value === 'number' ? value : (value ? parseFloat(String(value)) : undefined),
          displayData: value !== undefined && value !== null ? String(value) : '',
          allowOverlay: true,
          readonly: false,
        }

      case 'checkbox':
        return {
          kind: GridCellKind.Boolean,
          data: Boolean(value),
          allowOverlay: false,
          readonly: false,
        }

      case 'url':
        return {
          kind: GridCellKind.Uri,
          data: value ? String(value) : '',
          displayData: value ? String(value) : '',
          allowOverlay: true,
          readonly: false,
          hoverEffect: true,
        }

      case 'select':
      case 'status':
        return {
          kind: GridCellKind.Text,
          data: value ? String(value) : '',
          displayData: getSelectDisplayValue(value, property.options || []),
          allowOverlay: true,
          readonly: false,
        }

      case 'multi_select':
        return {
          kind: GridCellKind.Bubble,
          data: getMultiSelectValues(value, property.options || []),
          allowOverlay: true,
        }

      case 'date':
        return {
          kind: GridCellKind.Text,
          data: value ? String(value) : '',
          displayData: formatDate(value),
          allowOverlay: true,
          readonly: false,
        }

      case 'created_time':
      case 'last_edited_time':
        return {
          kind: GridCellKind.Text,
          data: value ? String(value) : '',
          displayData: formatDate(value),
          allowOverlay: false,
          readonly: true,
        }

      case 'person':
        const personValue = Array.isArray(value)
          ? (value as string[]).join(', ')
          : value ? String(value) : ''
        return {
          kind: GridCellKind.Text,
          data: personValue,
          displayData: personValue,
          allowOverlay: true,
          readonly: false,
        }

      case 'created_by':
      case 'last_edited_by':
        return {
          kind: GridCellKind.Text,
          data: value ? String(value) : '',
          displayData: value ? String(value) : '',
          allowOverlay: false,
          readonly: true,
        }

      case 'relation':
        const relations = Array.isArray(value)
          ? (value as Array<{ id: string; title: string }>).map(r => r.title)
          : []
        return {
          kind: GridCellKind.Bubble,
          data: relations,
          allowOverlay: true,
        }

      case 'files':
        const files = Array.isArray(value)
          ? (value as Array<{ name: string }>).map(f => f.name).join(', ')
          : ''
        return {
          kind: GridCellKind.Text,
          data: files,
          displayData: files,
          allowOverlay: true,
          readonly: false,
        }

      case 'rollup':
      case 'formula':
        return {
          kind: GridCellKind.Text,
          data: value !== undefined ? String(value) : '',
          displayData: value !== undefined ? String(value) : '-',
          allowOverlay: false,
          readonly: true,
        }

      default:
        return {
          kind: GridCellKind.Text,
          data: value ? String(value) : '',
          displayData: value ? String(value) : '',
          allowOverlay: true,
          readonly: isReadOnly,
        }
    }
  }, [rows, visibleProperties])

  // Handle cell edits
  const onCellEdited = useCallback((cell: Item, newValue: EditableGridCell) => {
    const [col, row] = cell
    if (row >= rows.length || col >= visibleProperties.length) return

    const rowData = rows[row]
    const property = visibleProperties[col]

    let valueToSave: unknown

    switch (newValue.kind) {
      case GridCellKind.Text:
        valueToSave = newValue.data
        break
      case GridCellKind.Number:
        valueToSave = newValue.data
        break
      case GridCellKind.Boolean:
        valueToSave = newValue.data
        break
      case GridCellKind.Uri:
        valueToSave = newValue.data
        break
      default:
        // For bubble cells and other types
        valueToSave = (newValue as any).data
        // For multi-select, convert display names back to IDs
        if (property.type === 'multi_select') {
          const names = (newValue as any).data as string[]
          valueToSave = names?.map(name => {
            const option = property.options?.find(o => o.name === name)
            return option?.id || name
          }) || []
        }
    }

    // Call the update handler to persist to backend
    onUpdateRow(rowData.id, { [property.id]: valueToSave })
  }, [rows, visibleProperties, onUpdateRow])

  // Handle row deletion
  const handleDeleteSelected = useCallback(() => {
    const selectedRowIndices = selection.rows.toArray()
    if (selectedRowIndices.length === 0) return

    if (!confirm(`Delete ${selectedRowIndices.length} row(s)?`)) return

    selectedRowIndices.forEach(rowIndex => {
      if (rowIndex < rows.length) {
        onDeleteRow(rows[rowIndex].id)
      }
    })

    // Clear selection after delete
    setSelection({
      columns: CompactSelection.empty(),
      rows: CompactSelection.empty(),
    })
  }, [selection, rows, onDeleteRow])

  // Handle add row
  const handleAddRow = useCallback(async () => {
    await onAddRow()
  }, [onAddRow])

  // Handle cell click for special types (select dropdowns, date pickers, etc.)
  const onCellClicked = useCallback((cell: Item, event: CellClickedEventArgs) => {
    const [col] = cell
    const property = visibleProperties[col]

    // For select/status, we could open a custom dropdown
    // For now, using text editing
    if (property?.type === 'select' || property?.type === 'status') {
      // Could implement custom dropdown here
    }
  }, [visibleProperties])

  const selectedRowCount = selection.rows.length

  return (
    <div className="data-grid-container" ref={containerRef}>
      {/* Toolbar */}
      <div className="data-grid-toolbar" style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        padding: '8px 0',
        marginBottom: 8,
      }}>
        {selectedRowCount > 0 && (
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            padding: '6px 12px',
            background: 'var(--accent-bg)',
            borderRadius: 'var(--radius-md)',
          }}>
            <span style={{ fontSize: 13, fontWeight: 500 }}>{selectedRowCount} selected</span>
            <button
              onClick={handleDeleteSelected}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 4,
                padding: '4px 12px',
                background: 'var(--error-color)',
                color: 'white',
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                fontSize: 12,
                cursor: 'pointer',
              }}
            >
              <Trash2 size={12} />
              Delete
            </button>
          </div>
        )}
        <div style={{ flex: 1 }} />
        <span style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>
          {rows.length} {rows.length === 1 ? 'row' : 'rows'}
        </span>
      </div>

      {/* Grid */}
      <div style={{
        border: '1px solid var(--border-color)',
        borderRadius: 'var(--radius-md)',
        overflow: 'hidden',
      }}>
        <DataEditor
          ref={gridRef}
          columns={columns}
          rows={rows.length}
          getCellContent={getCellContent}
          onCellEdited={onCellEdited}
          onCellClicked={onCellClicked}
          gridSelection={selection}
          onGridSelectionChange={setSelection}
          theme={customTheme}
          width="100%"
          height={gridHeight}
          rowMarkers="both"
          smoothScrollX
          smoothScrollY
          getCellsForSelection
          rowSelect="multi"
          columnSelect="none"
          rangeSelect="multi-rect"
          keybindings={{
            downFill: true,
            rightFill: true,
            clear: true,
            copy: true,
            paste: true,
            search: true,
            selectAll: true,
            selectColumn: true,
            selectRow: true,
          }}
        />
      </div>

      {/* Add row button */}
      <button
        onClick={handleAddRow}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '8px 12px',
          marginTop: 8,
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          color: 'var(--text-tertiary)',
          fontSize: 13,
          width: '100%',
          textAlign: 'left',
        }}
      >
        <Plus size={14} />
        <span>New</span>
      </button>
    </div>
  )
}

export default DataGrid
