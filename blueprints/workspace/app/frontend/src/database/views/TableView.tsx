import { useState, useCallback, useRef, useEffect, useMemo } from 'react'
import DataEditor, {
  GridColumn,
  GridCell,
  GridCellKind,
  EditableGridCell,
  Item,
  CompactSelection,
  GridSelection,
  Theme,
  DataEditorRef,
  SpriteMap,
  CellClickedEventArgs,
} from '@glideapps/glide-data-grid'
import '@glideapps/glide-data-grid/dist/index.css'
import { DatabaseRow, Property, PropertyType, Database } from '../../api/client'
import { RowDetailModal } from '../RowDetailModal'
import {
  Plus,
  Trash2,
  Eye,
  EyeOff,
  Search,
  Type,
  Hash,
  Calendar,
  CheckSquare,
  Link,
  Mail,
  Phone,
  Users,
  Paperclip,
  List,
  CircleDot,
  Clock,
  User,
  X,
} from 'lucide-react'

// SVG icon generators for header icons (returns SVG string for Glide Data Grid)
const headerIcons: SpriteMap = {
  // Text type icon (Aa)
  headerText: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <polyline points="4 7 4 4 20 4 20 7"/>
    <line x1="9" y1="20" x2="15" y2="20"/>
    <line x1="12" y1="4" x2="12" y2="20"/>
  </svg>`,

  // Number type icon (#)
  headerNumber: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <line x1="4" y1="9" x2="20" y2="9"/>
    <line x1="4" y1="15" x2="20" y2="15"/>
    <line x1="10" y1="3" x2="8" y2="21"/>
    <line x1="16" y1="3" x2="14" y2="21"/>
  </svg>`,

  // Date type icon (calendar)
  headerDate: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
    <line x1="16" y1="2" x2="16" y2="6"/>
    <line x1="8" y1="2" x2="8" y2="6"/>
    <line x1="3" y1="10" x2="21" y2="10"/>
  </svg>`,

  // Checkbox type icon
  headerCheckbox: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <polyline points="9 11 12 14 22 4"/>
    <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/>
  </svg>`,

  // URL type icon (link)
  headerUrl: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
    <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
  </svg>`,

  // Email type icon
  headerEmail: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
    <polyline points="22,6 12,13 2,6"/>
  </svg>`,

  // Phone type icon
  headerPhone: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/>
  </svg>`,

  // Person/users type icon
  headerPerson: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
    <circle cx="9" cy="7" r="4"/>
    <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
    <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
  </svg>`,

  // Files type icon
  headerFiles: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"/>
  </svg>`,

  // Select type icon (dot in circle)
  headerSelect: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <circle cx="12" cy="12" r="10"/>
    <circle cx="12" cy="12" r="3" fill="${p.fgColor}"/>
  </svg>`,

  // Multi-select type icon (list)
  headerMultiSelect: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <line x1="8" y1="6" x2="21" y2="6"/>
    <line x1="8" y1="12" x2="21" y2="12"/>
    <line x1="8" y1="18" x2="21" y2="18"/>
    <line x1="3" y1="6" x2="3.01" y2="6"/>
    <line x1="3" y1="12" x2="3.01" y2="12"/>
    <line x1="3" y1="18" x2="3.01" y2="18"/>
  </svg>`,

  // Status type icon (badge)
  headerStatus: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <circle cx="12" cy="12" r="10"/>
    <path d="M8 12l2 2 4-4"/>
  </svg>`,

  // Created/edited time icon (clock)
  headerTime: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <circle cx="12" cy="12" r="10"/>
    <polyline points="12 6 12 12 16 14"/>
  </svg>`,

  // Created/edited by icon (user)
  headerUser: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
    <circle cx="12" cy="7" r="4"/>
  </svg>`,

  // Relation type icon
  headerRelation: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <polyline points="15 3 21 3 21 9"/>
    <polyline points="9 21 3 21 3 15"/>
    <line x1="21" y1="3" x2="14" y2="10"/>
    <line x1="3" y1="21" x2="10" y2="14"/>
  </svg>`,

  // Formula type icon
  headerFormula: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <line x1="4" y1="4" x2="20" y2="4"/>
    <path d="M9 4v16l3-3 3 3V4"/>
  </svg>`,

  // Rollup type icon
  headerRollup: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
  </svg>`,
}

// Map property type to header icon name
function getHeaderIconName(type: PropertyType): string {
  switch (type) {
    case 'text': return 'headerText'
    case 'number': return 'headerNumber'
    case 'date': return 'headerDate'
    case 'checkbox': return 'headerCheckbox'
    case 'url': return 'headerUrl'
    case 'email': return 'headerEmail'
    case 'phone': return 'headerPhone'
    case 'person': return 'headerPerson'
    case 'files': return 'headerFiles'
    case 'select': return 'headerSelect'
    case 'multi_select': return 'headerMultiSelect'
    case 'status': return 'headerStatus'
    case 'created_time':
    case 'last_edited_time': return 'headerTime'
    case 'created_by':
    case 'last_edited_by': return 'headerUser'
    case 'relation': return 'headerRelation'
    case 'formula': return 'headerFormula'
    case 'rollup': return 'headerRollup'
    default: return 'headerText'
  }
}

interface TableViewProps {
  rows: DatabaseRow[]
  properties: Property[]
  groupBy: string | null
  database?: Database
  hiddenProperties?: string[]
  onAddRow: (initialProperties?: Record<string, unknown>) => Promise<DatabaseRow | null>
  onUpdateRow: (rowId: string, updates: Record<string, unknown>) => void
  onDeleteRow: (rowId: string) => void
  onAddProperty: (property: Omit<Property, 'id'>) => void
  onUpdateProperty: (propertyId: string, updates: Partial<Property>) => void
  onDeleteProperty: (propertyId: string) => void
  onHiddenPropertiesChange?: (hiddenProperties: string[]) => void
  onRowsReorder?: (rowIds: string[]) => void
}

// Notion-inspired theme
const notionTheme: Partial<Theme> = {
  accentColor: '#2383e2',
  accentLight: 'rgba(35, 131, 226, 0.1)',
  textDark: '#37352f',
  textMedium: '#787774',
  textLight: '#b4b4b0',
  bgCell: '#ffffff',
  bgCellMedium: '#fbfbfa',
  bgHeader: '#ffffff',
  bgHeaderHasFocus: '#f7f6f3',
  bgHeaderHovered: '#f7f6f3',
  borderColor: '#e9e9e7',
  fontFamily: 'ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif',
  baseFontStyle: '14px',
  headerFontStyle: '500 14px',
  editorFontSize: '14px',
  lineHeight: 1.5,
  cellHorizontalPadding: 8,
  cellVerticalPadding: 6,
  headerIconSize: 16,
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
      month: '2-digit',
      day: '2-digit',
      year: 'numeric',
    })
  } catch {
    return String(value)
  }
}

// Get icon for property type
function getPropertyIcon(type: PropertyType) {
  const iconProps = { size: 14, strokeWidth: 1.75, style: { opacity: 0.7 } }
  switch (type) {
    case 'text': return <Type {...iconProps} />
    case 'number': return <Hash {...iconProps} />
    case 'date': return <Calendar {...iconProps} />
    case 'checkbox': return <CheckSquare {...iconProps} />
    case 'url': return <Link {...iconProps} />
    case 'email': return <Mail {...iconProps} />
    case 'phone': return <Phone {...iconProps} />
    case 'person': return <Users {...iconProps} />
    case 'files': return <Paperclip {...iconProps} />
    case 'select': return <CircleDot {...iconProps} />
    case 'multi_select': return <List {...iconProps} />
    case 'status': return <CircleDot {...iconProps} />
    case 'created_time':
    case 'last_edited_time': return <Clock {...iconProps} />
    case 'created_by':
    case 'last_edited_by': return <User {...iconProps} />
    default: return <Type {...iconProps} />
  }
}

// Color map for select options
const selectColorMap: Record<string, string> = {
  gray: '#e9e9e7',
  brown: '#eee0da',
  orange: '#fadec9',
  yellow: '#fdecc8',
  green: '#dbeddb',
  blue: '#d3e5ef',
  purple: '#e8deee',
  pink: '#f5e0e9',
  red: '#ffd5d2',
}

// Sub-component for select dropdown options
function SelectDropdownOptions({
  property,
  currentValue,
  onSelect,
}: {
  property: Property
  currentValue: unknown
  onSelect: (optionId: string | null) => void
}) {
  const options = property.options || []

  if (options.length === 0) {
    return (
      <div style={{ padding: '8px 12px', fontSize: 13, color: 'rgba(55,53,47,0.5)' }}>
        No options yet
      </div>
    )
  }

  return (
    <>
      {options.map((option) => {
        const isSelected = property.type === 'multi_select'
          ? Array.isArray(currentValue) && currentValue.includes(option.id)
          : currentValue === option.id
        const bgColor = selectColorMap[option.color || 'gray'] || selectColorMap.gray

        return (
          <button
            key={option.id}
            onClick={() => onSelect(option.id)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              width: '100%',
              padding: '6px 12px',
              background: isSelected ? 'rgba(35, 131, 226, 0.08)' : 'transparent',
              border: 'none',
              cursor: 'pointer',
              textAlign: 'left',
              fontSize: 14,
              transition: 'background 0.1s',
            }}
            onMouseEnter={(e) => !isSelected && (e.currentTarget.style.background = 'rgba(55,53,47,0.04)')}
            onMouseLeave={(e) => !isSelected && (e.currentTarget.style.background = 'transparent')}
          >
            {property.type === 'multi_select' && (
              <div style={{
                width: 16,
                height: 16,
                border: isSelected ? 'none' : '1px solid rgba(55,53,47,0.3)',
                borderRadius: 3,
                background: isSelected ? '#2383e2' : 'transparent',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flexShrink: 0,
              }}>
                {isSelected && (
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="3">
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                )}
              </div>
            )}
            <span style={{
              display: 'inline-block',
              padding: '2px 8px',
              borderRadius: 3,
              background: bgColor,
              color: '#37352f',
              fontSize: 13,
            }}>
              {option.name}
            </span>
            {property.type !== 'multi_select' && isSelected && (
              <svg style={{ marginLeft: 'auto' }} width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#2383e2" strokeWidth="2">
                <polyline points="20 6 9 17 4 12" />
              </svg>
            )}
          </button>
        )
      })}
    </>
  )
}

export function TableView({
  rows,
  properties,
  database,
  hiddenProperties = [],
  onAddRow,
  onUpdateRow,
  onDeleteRow,
  onAddProperty,
  onHiddenPropertiesChange,
}: TableViewProps) {
  const gridRef = useRef<DataEditorRef>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const [gridHeight, setGridHeight] = useState(400)
  const [selection, setSelection] = useState<GridSelection>({
    columns: CompactSelection.empty(),
    rows: CompactSelection.empty(),
  })
  const [detailRow, setDetailRow] = useState<DatabaseRow | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [showColumnVisibility, setShowColumnVisibility] = useState(false)
  const [showAddProperty, setShowAddProperty] = useState(false)
  const columnVisibilityRef = useRef<HTMLDivElement>(null)
  const addPropertyRef = useRef<HTMLDivElement>(null)

  // Select dropdown state
  const [selectDropdown, setSelectDropdown] = useState<{
    cell: Item
    property: Property
    row: DatabaseRow
    bounds: { x: number; y: number; width: number; height: number }
  } | null>(null)

  // Calculate grid height
  useEffect(() => {
    const updateHeight = () => {
      if (containerRef.current) {
        const rect = containerRef.current.getBoundingClientRect()
        const availableHeight = window.innerHeight - rect.top - 80
        setGridHeight(Math.max(300, Math.min(availableHeight, 700)))
      }
    }
    updateHeight()
    window.addEventListener('resize', updateHeight)
    return () => window.removeEventListener('resize', updateHeight)
  }, [])

  // Close menus on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (columnVisibilityRef.current && !columnVisibilityRef.current.contains(e.target as Node)) {
        setShowColumnVisibility(false)
      }
      if (addPropertyRef.current && !addPropertyRef.current.contains(e.target as Node)) {
        setShowAddProperty(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Get visible properties
  const visibleProperties = useMemo(() => {
    return properties.filter(p => !hiddenProperties.includes(p.id))
  }, [properties, hiddenProperties])

  // Filter rows by search query
  const filteredRows = useMemo(() => {
    if (!searchQuery.trim()) return rows
    const query = searchQuery.toLowerCase()
    return rows.filter(row => {
      return properties.some(prop => {
        const value = row.properties[prop.id]
        if (value === null || value === undefined) return false
        return String(value).toLowerCase().includes(query)
      })
    })
  }, [rows, searchQuery, properties])

  // Create columns with icons
  const columns: GridColumn[] = useMemo(() => {
    return visibleProperties.map((prop, index) => ({
      id: prop.id,
      title: prop.name,
      icon: getHeaderIconName(prop.type),
      width: index === 0 ? 280 : 150,
      grow: index === 0 ? 1 : 0,
      hasMenu: false,
      themeOverride: index === 0 ? {
        baseFontStyle: '500 14px',
        textDark: '#37352f',
      } : undefined,
    }))
  }, [visibleProperties])

  // Get cell content
  const getCellContent = useCallback((cell: Item): GridCell => {
    const [col, row] = cell
    if (row >= filteredRows.length || col >= visibleProperties.length) {
      return { kind: GridCellKind.Text, data: '', displayData: '', allowOverlay: false }
    }

    const rowData = filteredRows[row]
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
        const numValue = typeof value === 'number' ? value : (value ? parseFloat(String(value)) : undefined)
        return {
          kind: GridCellKind.Number,
          data: numValue,
          displayData: numValue !== undefined && !isNaN(numValue) ? String(numValue) : '',
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
          displayData: value ? String(value).replace(/^https?:\/\//, '').slice(0, 35) : '',
          allowOverlay: true,
          readonly: false,
          hoverEffect: true,
        }

      case 'select':
      case 'status':
        const selectedOption = property.options?.find(o => o.id === value)
        return {
          kind: GridCellKind.Bubble,
          data: selectedOption ? [selectedOption.name] : [],
          allowOverlay: false, // Use custom dropdown instead
        }

      case 'multi_select':
        const selectedIds = Array.isArray(value) ? value : []
        const selectedNames = selectedIds
          .map(id => property.options?.find(o => o.id === id)?.name)
          .filter(Boolean) as string[]
        return {
          kind: GridCellKind.Bubble,
          data: selectedNames,
          allowOverlay: false, // Use custom dropdown instead
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
          themeOverride: { textDark: '#787774' },
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
          themeOverride: { textDark: '#787774' },
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
          ? (value as Array<{ name: string }>).map(f => f.name)
          : []
        return {
          kind: GridCellKind.Bubble,
          data: files,
          allowOverlay: true,
        }

      case 'rollup':
      case 'formula':
        return {
          kind: GridCellKind.Text,
          data: value !== undefined ? String(value) : '',
          displayData: value !== undefined ? String(value) : '-',
          allowOverlay: false,
          readonly: true,
          themeOverride: { textDark: '#787774' },
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
  }, [filteredRows, visibleProperties])

  // Handle cell edits
  const onCellEdited = useCallback((cell: Item, newValue: EditableGridCell) => {
    const [col, row] = cell
    if (row >= filteredRows.length || col >= visibleProperties.length) return

    const rowData = filteredRows[row]
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
        valueToSave = (newValue as any).data
        if (property.type === 'select' || property.type === 'status') {
          const names = (newValue as any).data as string[]
          const name = names?.[0]
          const option = property.options?.find(o => o.name === name)
          valueToSave = option?.id || ''
        } else if (property.type === 'multi_select') {
          const names = (newValue as any).data as string[]
          valueToSave = names?.map(name => {
            const option = property.options?.find(o => o.name === name)
            return option?.id || name
          }) || []
        }
    }

    onUpdateRow(rowData.id, { [property.id]: valueToSave })
  }, [filteredRows, visibleProperties, onUpdateRow])

  // Handle row double-click
  const onCellActivated = useCallback((cell: Item) => {
    const [, row] = cell
    if (row < filteredRows.length) {
      setDetailRow(filteredRows[row])
    }
  }, [filteredRows])

  // Handle cell click for select dropdowns
  const onCellClicked = useCallback((cell: Item, event: CellClickedEventArgs) => {
    const [col, row] = cell
    if (row >= filteredRows.length || col >= visibleProperties.length) return

    const property = visibleProperties[col]
    const rowData = filteredRows[row]

    // Open dropdown for select/status/multi_select types
    if (property.type === 'select' || property.type === 'status' || property.type === 'multi_select') {
      setSelectDropdown({
        cell,
        property,
        row: rowData,
        bounds: event.bounds,
      })
    }
  }, [filteredRows, visibleProperties])

  // Handle select option change
  const handleSelectOption = useCallback((optionId: string | null) => {
    if (!selectDropdown) return

    const { property, row, cell } = selectDropdown
    const [, rowIndex] = cell

    // Get fresh row data from filteredRows to avoid stale state
    const currentRow = filteredRows[rowIndex] || row

    if (property.type === 'multi_select') {
      // For multi-select, toggle the option
      const currentValues = Array.isArray(currentRow.properties[property.id])
        ? (currentRow.properties[property.id] as string[])
        : []
      const newValues = currentValues.includes(optionId || '')
        ? currentValues.filter(v => v !== optionId)
        : [...currentValues, optionId || ''].filter(Boolean)
      onUpdateRow(currentRow.id, { [property.id]: newValues })

      // Update the dropdown state with fresh row data for next toggle
      setSelectDropdown(prev => prev ? { ...prev, row: { ...currentRow, properties: { ...currentRow.properties, [property.id]: newValues } } } : null)
    } else {
      // For single select/status, set the value
      onUpdateRow(currentRow.id, { [property.id]: optionId || '' })
      setSelectDropdown(null)
    }
  }, [selectDropdown, onUpdateRow, filteredRows])

  // Handle delete selected
  const handleDeleteSelected = useCallback(() => {
    const selectedRowIndices = selection.rows.toArray()
    if (selectedRowIndices.length === 0) return

    if (!confirm(`Delete ${selectedRowIndices.length} row(s)?`)) return

    selectedRowIndices.forEach(rowIndex => {
      if (rowIndex < filteredRows.length) {
        onDeleteRow(filteredRows[rowIndex].id)
      }
    })

    setSelection({
      columns: CompactSelection.empty(),
      rows: CompactSelection.empty(),
    })
  }, [selection, filteredRows, onDeleteRow])

  // Handle add row
  const handleAddRow = useCallback(async () => {
    await onAddRow()
  }, [onAddRow])

  // Toggle property visibility
  const togglePropertyVisibility = useCallback((propertyId: string) => {
    const newHidden = hiddenProperties.includes(propertyId)
      ? hiddenProperties.filter(id => id !== propertyId)
      : [...hiddenProperties, propertyId]
    onHiddenPropertiesChange?.(newHidden)
  }, [hiddenProperties, onHiddenPropertiesChange])

  // Handle add property
  const handleAddProperty = useCallback((type: PropertyType) => {
    onAddProperty({
      name: 'New Property',
      type,
      options: type === 'select' || type === 'multi_select' || type === 'status' ? [] : undefined,
    })
    setShowAddProperty(false)
  }, [onAddProperty])

  const selectedRowCount = selection.rows.length

  const propertyTypes: { type: PropertyType; label: string; icon: React.ReactNode }[] = [
    { type: 'text', label: 'Text', icon: <Type size={14} /> },
    { type: 'number', label: 'Number', icon: <Hash size={14} /> },
    { type: 'select', label: 'Select', icon: <CircleDot size={14} /> },
    { type: 'multi_select', label: 'Multi-select', icon: <List size={14} /> },
    { type: 'status', label: 'Status', icon: <CircleDot size={14} /> },
    { type: 'date', label: 'Date', icon: <Calendar size={14} /> },
    { type: 'person', label: 'Person', icon: <Users size={14} /> },
    { type: 'checkbox', label: 'Checkbox', icon: <CheckSquare size={14} /> },
    { type: 'url', label: 'URL', icon: <Link size={14} /> },
    { type: 'email', label: 'Email', icon: <Mail size={14} /> },
    { type: 'phone', label: 'Phone', icon: <Phone size={14} /> },
    { type: 'files', label: 'Files & media', icon: <Paperclip size={14} /> },
  ]

  return (
    <div className="table-view" ref={containerRef} style={{ fontFamily: notionTheme.fontFamily }}>
      {/* Toolbar - Notion style */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: 4,
        padding: '4px 0',
        marginBottom: 2,
      }}>
        {/* Search */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          padding: '4px 8px',
          background: searchQuery ? '#f7f6f3' : 'transparent',
          borderRadius: 4,
          transition: 'all 0.15s',
        }}>
          <Search size={14} style={{ color: '#9a9a97' }} />
          <input
            type="text"
            placeholder="Search..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            style={{
              width: searchQuery ? 160 : 50,
              border: 'none',
              background: 'none',
              outline: 'none',
              fontSize: 14,
              color: '#37352f',
              transition: 'width 0.2s',
            }}
            onFocus={(e) => e.currentTarget.style.width = '160px'}
            onBlur={(e) => !searchQuery && (e.currentTarget.style.width = '50px')}
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              style={{
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                padding: 2,
                color: '#9a9a97',
                display: 'flex',
                borderRadius: 2,
              }}
            >
              <X size={12} />
            </button>
          )}
        </div>

        <div style={{ flex: 1 }} />

        {/* Properties toggle */}
        <div ref={columnVisibilityRef} style={{ position: 'relative' }}>
          <button
            onClick={() => setShowColumnVisibility(!showColumnVisibility)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 4,
              padding: '4px 8px',
              background: showColumnVisibility ? '#f7f6f3' : 'none',
              border: 'none',
              borderRadius: 4,
              cursor: 'pointer',
              fontSize: 14,
              color: '#9a9a97',
            }}
            onMouseEnter={(e) => e.currentTarget.style.background = '#f7f6f3'}
            onMouseLeave={(e) => !showColumnVisibility && (e.currentTarget.style.background = 'none')}
          >
            <Eye size={14} />
          </button>
          {showColumnVisibility && (
            <div style={{
              position: 'absolute',
              top: '100%',
              right: 0,
              marginTop: 4,
              background: '#ffffff',
              border: '1px solid rgba(55,53,47,0.09)',
              borderRadius: 6,
              boxShadow: 'rgba(15, 15, 15, 0.05) 0px 0px 0px 1px, rgba(15, 15, 15, 0.1) 0px 3px 6px, rgba(15, 15, 15, 0.2) 0px 9px 24px',
              minWidth: 220,
              maxHeight: 320,
              overflowY: 'auto',
              zIndex: 100,
            }}>
              <div style={{
                padding: '8px 12px 6px',
                fontSize: 11,
                fontWeight: 500,
                color: 'rgba(55,53,47,0.65)',
                textTransform: 'uppercase',
                letterSpacing: '0.5px',
              }}>
                Properties
              </div>
              {properties.map((prop) => (
                <button
                  key={prop.id}
                  onClick={() => togglePropertyVisibility(prop.id)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    width: '100%',
                    padding: '6px 12px',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    textAlign: 'left',
                    fontSize: 14,
                    color: hiddenProperties.includes(prop.id) ? 'rgba(55,53,47,0.35)' : '#37352f',
                    transition: 'background 0.1s',
                  }}
                  onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(55,53,47,0.04)'}
                  onMouseLeave={(e) => e.currentTarget.style.background = 'none'}
                >
                  <span style={{ color: 'rgba(55,53,47,0.45)' }}>{getPropertyIcon(prop.type)}</span>
                  <span style={{ flex: 1 }}>{prop.name}</span>
                  {hiddenProperties.includes(prop.id) ? (
                    <EyeOff size={14} style={{ color: 'rgba(55,53,47,0.35)' }} />
                  ) : (
                    <Eye size={14} style={{ color: '#2383e2' }} />
                  )}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Row count */}
        <span style={{ fontSize: 12, color: 'rgba(55,53,47,0.5)', padding: '0 8px' }}>
          {filteredRows.length}
        </span>
      </div>

      {/* Bulk actions */}
      {selectedRowCount > 0 && (
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '6px 10px',
          background: '#e8f4ff',
          borderRadius: 4,
          marginBottom: 6,
        }}>
          <span style={{ fontSize: 13, fontWeight: 500, color: '#37352f' }}>
            {selectedRowCount} selected
          </span>
          <button
            onClick={handleDeleteSelected}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 4,
              padding: '3px 8px',
              background: '#eb5757',
              color: 'white',
              border: 'none',
              borderRadius: 4,
              fontSize: 12,
              cursor: 'pointer',
              fontWeight: 500,
            }}
          >
            <Trash2 size={12} />
            Delete
          </button>
          <button
            onClick={() => setSelection({ columns: CompactSelection.empty(), rows: CompactSelection.empty() })}
            style={{
              padding: '3px 8px',
              background: 'white',
              border: '1px solid rgba(55,53,47,0.16)',
              borderRadius: 4,
              fontSize: 12,
              cursor: 'pointer',
              color: '#37352f',
            }}
          >
            Clear
          </button>
        </div>
      )}

      {/* Grid container */}
      <div style={{
        borderRadius: 0,
        overflow: 'hidden',
        borderTop: '1px solid #e9e9e7',
      }}>
        <DataEditor
          ref={gridRef}
          columns={columns}
          rows={filteredRows.length}
          getCellContent={getCellContent}
          onCellEdited={onCellEdited}
          onCellActivated={onCellActivated}
          onCellClicked={onCellClicked}
          gridSelection={selection}
          onGridSelectionChange={setSelection}
          theme={notionTheme}
          headerIcons={headerIcons}
          width="100%"
          height={gridHeight}
          rowMarkers="clickable-number"
          rowMarkerWidth={32}
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
            search: false,
            selectAll: true,
            selectColumn: false,
            selectRow: true,
          }}
          onPaste
          rowHeight={32}
          headerHeight={32}
          verticalBorder={false}
          rightElement={
            <div
              onClick={() => setShowAddProperty(true)}
              style={{
                width: 42,
                height: 32,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: 'var(--bg-header, #ffffff)',
                borderLeft: '1px solid #e9e9e7',
                cursor: 'pointer',
                color: 'rgba(55,53,47,0.5)',
                transition: 'all 0.15s',
              }}
              title="Add a property"
              onMouseEnter={(e) => {
                e.currentTarget.style.background = '#f7f6f3'
                e.currentTarget.style.color = '#37352f'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = 'var(--bg-header, #ffffff)'
                e.currentTarget.style.color = 'rgba(55,53,47,0.5)'
              }}
            >
              <Plus size={16} strokeWidth={1.5} />
            </div>
          }
          rightElementProps={{
            fill: false,
            sticky: true,
          }}
        />
      </div>

      {/* Add row button - Notion style */}
      <button
        onClick={handleAddRow}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 4,
          padding: '4px 6px',
          marginTop: 0,
          marginLeft: 6,
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          color: 'rgba(55,53,47,0.5)',
          fontSize: 14,
          textAlign: 'left',
          borderRadius: 3,
          transition: 'background 0.1s',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.background = 'rgba(55,53,47,0.04)'
          e.currentTarget.style.color = '#37352f'
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.background = 'none'
          e.currentTarget.style.color = 'rgba(55,53,47,0.5)'
        }}
      >
        <Plus size={12} />
        <span>New</span>
      </button>

      {/* Add property menu */}
      {showAddProperty && (
        <div
          ref={addPropertyRef}
          style={{
            position: 'fixed',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            background: '#ffffff',
            border: '1px solid rgba(55,53,47,0.09)',
            borderRadius: 6,
            boxShadow: 'rgba(15, 15, 15, 0.05) 0px 0px 0px 1px, rgba(15, 15, 15, 0.1) 0px 3px 6px, rgba(15, 15, 15, 0.2) 0px 9px 24px',
            minWidth: 240,
            maxHeight: 400,
            overflowY: 'auto',
            zIndex: 1000,
          }}
        >
          <div style={{
            padding: '10px 12px',
            borderBottom: '1px solid rgba(55,53,47,0.09)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}>
            <span style={{ fontSize: 14, fontWeight: 500, color: '#37352f' }}>
              Property type
            </span>
            <button
              onClick={() => setShowAddProperty(false)}
              style={{
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                padding: 4,
                color: 'rgba(55,53,47,0.45)',
                display: 'flex',
                borderRadius: 3,
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(55,53,47,0.08)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'none'}
            >
              <X size={14} />
            </button>
          </div>
          {propertyTypes.map(({ type, label, icon }) => (
            <button
              key={type}
              onClick={() => handleAddProperty(type)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                width: '100%',
                padding: '8px 12px',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                textAlign: 'left',
                fontSize: 14,
                color: '#37352f',
                transition: 'background 0.1s',
              }}
              onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(55,53,47,0.04)'}
              onMouseLeave={(e) => e.currentTarget.style.background = 'none'}
            >
              <span style={{ color: 'rgba(55,53,47,0.45)', display: 'flex' }}>{icon}</span>
              <span>{label}</span>
            </button>
          ))}
        </div>
      )}

      {/* Backdrop for add property */}
      {showAddProperty && (
        <div
          onClick={() => setShowAddProperty(false)}
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(15,15,15,0.6)',
            zIndex: 999,
          }}
        />
      )}

      {/* Select dropdown overlay */}
      {selectDropdown && (
        <>
          {/* Backdrop */}
          <div
            onClick={() => setSelectDropdown(null)}
            style={{
              position: 'fixed',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              zIndex: 1000,
            }}
          />
          {/* Dropdown menu */}
          <div
            style={{
              position: 'fixed',
              top: selectDropdown.bounds.y + selectDropdown.bounds.height + 4,
              left: selectDropdown.bounds.x,
              minWidth: Math.max(selectDropdown.bounds.width, 200),
              maxWidth: 300,
              background: '#ffffff',
              border: '1px solid rgba(55,53,47,0.09)',
              borderRadius: 6,
              boxShadow: 'rgba(15, 15, 15, 0.05) 0px 0px 0px 1px, rgba(15, 15, 15, 0.1) 0px 3px 6px, rgba(15, 15, 15, 0.2) 0px 9px 24px',
              zIndex: 1001,
              maxHeight: 300,
              overflowY: 'auto',
              padding: '6px 0',
            }}
          >
            {/* Options */}
            <SelectDropdownOptions
              property={selectDropdown.property}
              currentValue={selectDropdown.row.properties[selectDropdown.property.id]}
              onSelect={handleSelectOption}
            />

            {/* Clear option for single select */}
            {selectDropdown.property.type !== 'multi_select' && Boolean(selectDropdown.row.properties[selectDropdown.property.id]) && (
              <>
                <div style={{ height: 1, background: 'rgba(55,53,47,0.09)', margin: '4px 0' }} />
                <button
                  onClick={() => handleSelectOption(null)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    width: '100%',
                    padding: '6px 12px',
                    background: 'transparent',
                    border: 'none',
                    cursor: 'pointer',
                    textAlign: 'left',
                    fontSize: 13,
                    color: 'rgba(55,53,47,0.5)',
                    transition: 'background 0.1s',
                  }}
                  onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(55,53,47,0.04)'}
                  onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
                >
                  <X size={12} />
                  Clear
                </button>
              </>
            )}

            {/* Done button for multi-select */}
            {selectDropdown.property.type === 'multi_select' && (
              <>
                <div style={{ height: 1, background: 'rgba(55,53,47,0.09)', margin: '4px 0' }} />
                <button
                  onClick={() => setSelectDropdown(null)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: 'calc(100% - 16px)',
                    margin: '4px 8px',
                    padding: '6px 12px',
                    background: '#2383e2',
                    border: 'none',
                    borderRadius: 4,
                    cursor: 'pointer',
                    fontSize: 13,
                    fontWeight: 500,
                    color: 'white',
                  }}
                >
                  Done
                </button>
              </>
            )}
          </div>
        </>
      )}

      {/* Row Detail Modal */}
      {detailRow && database && (
        <RowDetailModal
          row={detailRow}
          database={database}
          onClose={() => setDetailRow(null)}
          onUpdate={() => {}}
          onDelete={(rowId) => {
            onDeleteRow(rowId)
            setDetailRow(null)
          }}
        />
      )}
    </div>
  )
}

export default TableView
