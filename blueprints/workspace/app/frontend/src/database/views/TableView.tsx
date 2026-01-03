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
  GridMouseEventArgs,
} from '@glideapps/glide-data-grid'
import { allCells, type DropdownCellType, type TagsCellType, type DatePickerType } from '@glideapps/glide-data-grid-cells'
import '@glideapps/glide-data-grid/dist/index.css'
import '@glideapps/glide-data-grid-cells/dist/index.css'
import { DatabaseRow, Property, PropertyType, Database, FileAttachment } from '../../api/client'
import { filesCellRenderer, createFilesCell } from '../cells/FilesCell'
import { SidePeek } from '../SidePeek'
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
  Edit2,
  Expand,
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

  // Add row icon (plus sign for trailing row)
  addRow: (p) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${p.fgColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <line x1="12" y1="5" x2="12" y2="19"/>
    <line x1="5" y1="12" x2="19" y2="12"/>
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
  textHeader: '#787774', // Header text color
  fgIconHeader: '#787774', // Header icon foreground color (this controls icon colors!)
  bgIconHeader: '#ffffff', // Header icon background color
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

export function TableView({
  rows,
  properties,
  database,
  hiddenProperties = [],
  onAddRow,
  onUpdateRow,
  onDeleteRow,
  onAddProperty,
  onUpdateProperty,
  onDeleteProperty,
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
  const [detailRowIndex, setDetailRowIndex] = useState<number>(-1)
  const [searchQuery, setSearchQuery] = useState('')
  const [hoveredRow, setHoveredRow] = useState<number | null>(null)
  const [openButtonBounds, setOpenButtonBounds] = useState<{ x: number; y: number; width: number; height: number } | null>(null)
  const [isOpenButtonHovered, setIsOpenButtonHovered] = useState(false)
  const hideTimeoutRef = useRef<number | null>(null)
  const [showColumnVisibility, setShowColumnVisibility] = useState(false)
  const [showAddProperty, setShowAddProperty] = useState<{ x: number; y: number } | null>(null)
  const [newPropertyName, setNewPropertyName] = useState('')
  const [propertyTypeSearch, setPropertyTypeSearch] = useState('')
  const columnVisibilityRef = useRef<HTMLDivElement>(null)
  const addPropertyRef = useRef<HTMLDivElement>(null)
  const propertyNameInputRef = useRef<HTMLInputElement>(null)


  // Header menu state for column editing
  const [headerMenu, setHeaderMenu] = useState<{
    property: Property
    bounds: { x: number; y: number }
  } | null>(null)
  const [editingColumnName, setEditingColumnName] = useState<string | null>(null)
  const [columnNameInput, setColumnNameInput] = useState('')
  const headerMenuRef = useRef<HTMLDivElement>(null)

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
        setShowAddProperty(null)
        setNewPropertyName('')
        setPropertyTypeSearch('')
      }
      if (headerMenuRef.current && !headerMenuRef.current.contains(e.target as Node)) {
        setHeaderMenu(null)
        setEditingColumnName(null)
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

  // Create columns with icons and menu
  const columns: GridColumn[] = useMemo(() => {
    return visibleProperties.map((prop, index) => ({
      id: prop.id,
      title: prop.name,
      icon: getHeaderIconName(prop.type),
      width: index === 0 ? 280 : 150,
      grow: index === 0 ? 1 : 0,
      hasMenu: true,
      themeOverride: index === 0 ? {
        baseFontStyle: '500 14px',
        textDark: '#37352f',
      } : undefined,
    }))
  }, [visibleProperties])

  // Handle header menu click
  const onHeaderMenuClick = useCallback((col: number, bounds: { x: number; y: number; width: number; height: number }) => {
    if (col < visibleProperties.length) {
      setHeaderMenu({
        property: visibleProperties[col],
        bounds: { x: bounds.x, y: bounds.y + bounds.height },
      })
    }
  }, [visibleProperties])

  // Handle column rename
  const handleColumnRename = useCallback((propertyId: string, newName: string) => {
    if (newName.trim()) {
      onUpdateProperty(propertyId, { name: newName.trim() })
    }
    setEditingColumnName(null)
    setColumnNameInput('')
    setHeaderMenu(null)
  }, [onUpdateProperty])


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
        // Use DropdownCell from glide-data-grid-cells for single select with colors
        const selectedOption = property.options?.find(o => o.id === value)
        const dropdownOptions = property.options?.map(o => ({
          value: o.id,
          label: o.name,
          color: selectColorMap[o.color || 'gray'] || selectColorMap.gray,
        })) || []
        return {
          kind: GridCellKind.Custom,
          allowOverlay: true,
          copyData: selectedOption?.name || '',
          data: {
            kind: 'dropdown-cell',
            value: value as string || null,
            allowedValues: dropdownOptions.map(o => ({ value: o.value, label: o.label })),
          },
        } as DropdownCellType

      case 'multi_select':
        // Use TagsCell from glide-data-grid-cells for multi-select with colors
        const selectedTagIds = Array.isArray(value) ? value : []
        const possibleTags = property.options?.map(o => ({
          tag: o.name,
          color: selectColorMap[o.color || 'gray'] || selectColorMap.gray,
        })) || []
        const selectedTags = selectedTagIds
          .map(id => property.options?.find(o => o.id === id)?.name)
          .filter(Boolean) as string[]
        return {
          kind: GridCellKind.Custom,
          allowOverlay: true,
          copyData: selectedTags.join(', '),
          data: {
            kind: 'tags-cell',
            possibleTags,
            tags: selectedTags,
          },
        } as TagsCellType

      case 'date':
        // Use DatePickerCell from glide-data-grid-cells for date editing
        const dateValue = value ? new Date(String(value)) : undefined
        return {
          kind: GridCellKind.Custom,
          allowOverlay: true,
          copyData: formatDate(value),
          data: {
            kind: 'date-picker-cell',
            date: dateValue,
            displayDate: formatDate(value),
            format: 'date', // 'date' | 'datetime' | 'time'
          },
        } as DatePickerType

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
        // Use custom FilesCell for file attachments with upload support
        const filesData: FileAttachment[] = Array.isArray(value)
          ? (value as FileAttachment[])
          : []
        return createFilesCell(filesData)

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
      case GridCellKind.Custom:
        // Handle custom cells from glide-data-grid-cells
        const customData = (newValue as any).data
        if (customData?.kind === 'dropdown-cell') {
          // DropdownCell: value is the selected option value (id)
          valueToSave = customData.value || ''
        } else if (customData?.kind === 'tags-cell') {
          // TagsCell: tags is array of tag names, need to convert to ids
          const tagNames = customData.tags as string[]
          valueToSave = tagNames?.map(name => {
            const option = property.options?.find(o => o.name === name)
            return option?.id || name
          }) || []
        } else if (customData?.kind === 'date-picker-cell') {
          // DatePickerCell: date is a Date object
          valueToSave = customData.date ? customData.date.toISOString() : null
        } else if (customData?.kind === 'files-cell') {
          // FilesCell: files is array of FileAttachment objects
          valueToSave = customData.files || []
        } else {
          valueToSave = customData
        }
        break
      default:
        valueToSave = (newValue as any).data
    }

    onUpdateRow(rowData.id, { [property.id]: valueToSave })
  }, [filteredRows, visibleProperties, onUpdateRow])

  // Handle row marker click (open side peek)
  const onRowMarkerClick = useCallback((row: number) => {
    if (row < filteredRows.length) {
      setDetailRow(filteredRows[row])
      setDetailRowIndex(row)
    }
  }, [filteredRows])

  // Handle side peek navigation
  const handleNavigate = useCallback((direction: 'prev' | 'next') => {
    const newIndex = direction === 'prev' ? detailRowIndex - 1 : detailRowIndex + 1
    if (newIndex >= 0 && newIndex < filteredRows.length) {
      setDetailRow(filteredRows[newIndex])
      setDetailRowIndex(newIndex)
    }
  }, [detailRowIndex, filteredRows])

  // Handle row update from side peek
  const handleRowUpdate = useCallback((updatedRow: DatabaseRow) => {
    // Find and update the row in the filtered list
    const index = filteredRows.findIndex(r => r.id === updatedRow.id)
    if (index >= 0) {
      onUpdateRow(updatedRow.id, updatedRow.properties)
    }
    setDetailRow(updatedRow)
  }, [filteredRows, onUpdateRow])

  // Handle cell click - custom cells handle their own overlays now
  const onCellClicked = useCallback((_cell: Item, _event: CellClickedEventArgs) => {
    // Custom cells from glide-data-grid-cells handle their own dropdowns/overlays
  }, [])

  // Handle item hover to show open button (Notion-style)
  const onItemHovered = useCallback((args: GridMouseEventArgs) => {
    // Clear any pending hide timeout when hovering
    if (hideTimeoutRef.current) {
      clearTimeout(hideTimeoutRef.current)
      hideTimeoutRef.current = null
    }

    if (args.kind === 'cell' && args.location[1] < filteredRows.length) {
      const row = args.location[1]
      const col = args.location[0]
      // Show open button only when hovering on the first column (title)
      if (col === 0 && args.bounds) {
        setHoveredRow(row)
        setOpenButtonBounds({
          x: args.bounds.x,
          y: args.bounds.y,
          width: args.bounds.width,
          height: args.bounds.height,
        })
      } else if (hoveredRow !== row) {
        // Different row, hide immediately
        setHoveredRow(null)
        setOpenButtonBounds(null)
      }
      // Keep button visible while hovering anywhere on the same row
    } else if (args.kind === 'out-of-bounds' || args.kind === 'header') {
      // Delay hiding to allow moving to the button
      hideTimeoutRef.current = window.setTimeout(() => {
        if (!isOpenButtonHovered) {
          setHoveredRow(null)
          setOpenButtonBounds(null)
        }
      }, 100)
    }
  }, [filteredRows.length, hoveredRow, isOpenButtonHovered])

  // Handle open button click
  const handleOpenButtonClick = useCallback(() => {
    if (hoveredRow !== null && hoveredRow < filteredRows.length) {
      setDetailRow(filteredRows[hoveredRow])
      setDetailRowIndex(hoveredRow)
    }
  }, [hoveredRow, filteredRows])


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

  // Handle row appended from grid (returns position)
  const handleRowAppended = useCallback(async () => {
    await onAddRow()
    return 'bottom' as const
  }, [onAddRow])

  // Toggle property visibility
  const togglePropertyVisibility = useCallback((propertyId: string) => {
    const newHidden = hiddenProperties.includes(propertyId)
      ? hiddenProperties.filter(id => id !== propertyId)
      : [...hiddenProperties, propertyId]
    onHiddenPropertiesChange?.(newHidden)
  }, [hiddenProperties, onHiddenPropertiesChange])

  // Handle add property with type-specific default names
  const handleAddProperty = useCallback((type: PropertyType) => {
    // Generate default name based on type
    const typeNames: Record<PropertyType, string> = {
      text: 'Text',
      number: 'Number',
      select: 'Select',
      multi_select: 'Tags',
      status: 'Status',
      date: 'Date',
      person: 'Person',
      checkbox: 'Checkbox',
      url: 'URL',
      email: 'Email',
      phone: 'Phone',
      files: 'Files',
      relation: 'Relation',
      rollup: 'Rollup',
      formula: 'Formula',
      created_time: 'Created time',
      created_by: 'Created by',
      last_edited_time: 'Last edited time',
      last_edited_by: 'Last edited by',
    }

    // Use custom name if provided, otherwise generate default
    const existingNames = properties.map(p => p.name)
    let name = newPropertyName.trim() || typeNames[type] || 'Property'

    // Ensure name is unique
    if (existingNames.includes(name)) {
      const baseName = name
      let counter = 1
      while (existingNames.includes(name)) {
        counter++
        name = `${baseName} ${counter}`
      }
    }

    // Backend expects only: name, type, config (with options inside for select types)
    // Do NOT send 'options' at root level - it breaks BindJSON
    const isSelectType = type === 'select' || type === 'multi_select' || type === 'status'
    const propertyData: Omit<Property, 'id'> = {
      name,
      type,
    }
    // Only add config for select types with options structure
    if (isSelectType) {
      propertyData.config = { options: [] }
    }
    onAddProperty(propertyData)
    setShowAddProperty(null)
    setNewPropertyName('')
    setPropertyTypeSearch('')
  }, [onAddProperty, properties, newPropertyName])

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
              type="button"
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
            type="button"
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
                  type="button"
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
            type="button"
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
            type="button"
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
          onCellClicked={onCellClicked}
          onRowAppended={handleRowAppended}
          trailingRowOptions={{
            hint: 'New',
            sticky: true,
            tint: true,
            addIcon: 'addRow', // Custom icon defined in headerIcons
          }}
          onHeaderMenuClick={onHeaderMenuClick}
          onItemHovered={onItemHovered}
          gridSelection={selection}
          onGridSelectionChange={setSelection}
          theme={notionTheme}
          headerIcons={headerIcons}
          customRenderers={[...allCells, filesCellRenderer]}
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
              onClick={(e) => {
                e.preventDefault()
                e.stopPropagation()
                const rect = e.currentTarget.getBoundingClientRect()
                setShowAddProperty({ x: rect.left - 280, y: rect.bottom + 4 })
                setNewPropertyName('')
                setPropertyTypeSearch('')
                setTimeout(() => propertyNameInputRef.current?.focus(), 50)
              }}
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

      {/* Add property popup - Notion style positioned dropdown */}
      {showAddProperty && (
        <>
          {/* Invisible backdrop to close on click outside */}
          <div
            onClick={() => {
              setShowAddProperty(null)
              setNewPropertyName('')
              setPropertyTypeSearch('')
            }}
            style={{
              position: 'fixed',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              zIndex: 999,
            }}
          />
          <div
            ref={addPropertyRef}
            style={{
              position: 'fixed',
              top: showAddProperty.y,
              left: Math.max(10, Math.min(showAddProperty.x, window.innerWidth - 320)),
              background: '#ffffff',
              border: '1px solid rgba(55,53,47,0.09)',
              borderRadius: 6,
              boxShadow: 'rgba(15, 15, 15, 0.05) 0px 0px 0px 1px, rgba(15, 15, 15, 0.1) 0px 3px 6px, rgba(15, 15, 15, 0.2) 0px 9px 24px',
              width: 300,
              maxHeight: 480,
              overflowY: 'auto',
              zIndex: 1000,
            }}
          >
            {/* Property name input */}
            <div style={{ padding: '10px 10px 8px' }}>
              <input
                ref={propertyNameInputRef}
                type="text"
                value={newPropertyName}
                onChange={(e) => setNewPropertyName(e.target.value)}
                placeholder="Property name"
                style={{
                  width: '100%',
                  padding: '6px 8px',
                  border: '1px solid rgba(55,53,47,0.16)',
                  borderRadius: 4,
                  fontSize: 14,
                  outline: 'none',
                  background: '#fff',
                }}
                onFocus={(e) => e.currentTarget.style.borderColor = '#2383e2'}
                onBlur={(e) => e.currentTarget.style.borderColor = 'rgba(55,53,47,0.16)'}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') {
                    setShowAddProperty(null)
                    setNewPropertyName('')
                    setPropertyTypeSearch('')
                  }
                }}
              />
            </div>

            {/* Search filter */}
            <div style={{ padding: '0 10px 8px' }}>
              <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '5px 8px',
                background: 'rgba(55,53,47,0.04)',
                borderRadius: 4,
              }}>
                <Search size={14} style={{ color: '#9a9a97' }} />
                <input
                  type="text"
                  value={propertyTypeSearch}
                  onChange={(e) => setPropertyTypeSearch(e.target.value)}
                  placeholder="Search for a property type..."
                  style={{
                    flex: 1,
                    border: 'none',
                    background: 'none',
                    outline: 'none',
                    fontSize: 13,
                    color: '#37352f',
                  }}
                />
              </div>
            </div>

            {/* Divider */}
            <div style={{ height: 1, background: 'rgba(55,53,47,0.09)', margin: '0 0 6px' }} />

            {/* Suggested section - only show if no search query */}
            {!propertyTypeSearch && (
              <>
                <div style={{
                  padding: '6px 12px 4px',
                  fontSize: 11,
                  fontWeight: 500,
                  color: 'rgba(55,53,47,0.5)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.5px',
                }}>
                  Suggested
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, padding: '0 6px 8px' }}>
                  {propertyTypes.slice(0, 4).map(({ type, label, icon }) => (
                    <button
                      type="button"
                      key={`suggested-${type}`}
                      onClick={() => handleAddProperty(type)}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                        padding: '6px 8px',
                        background: 'none',
                        border: 'none',
                        cursor: 'pointer',
                        textAlign: 'left',
                        fontSize: 13,
                        color: '#37352f',
                        borderRadius: 4,
                        transition: 'background 0.1s',
                      }}
                      onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(55,53,47,0.04)'}
                      onMouseLeave={(e) => e.currentTarget.style.background = 'none'}
                    >
                      <span style={{ color: 'rgba(55,53,47,0.5)', display: 'flex' }}>{icon}</span>
                      <span>{label}</span>
                    </button>
                  ))}
                </div>
                <div style={{ height: 1, background: 'rgba(55,53,47,0.09)', margin: '0 0 6px' }} />
              </>
            )}

            {/* Type section */}
            <div style={{
              padding: '6px 12px 4px',
              fontSize: 11,
              fontWeight: 500,
              color: 'rgba(55,53,47,0.5)',
              textTransform: 'uppercase',
              letterSpacing: '0.5px',
            }}>
              Type
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2, padding: '0 6px 10px' }}>
              {propertyTypes
                .filter(({ label }) =>
                  !propertyTypeSearch || label.toLowerCase().includes(propertyTypeSearch.toLowerCase())
                )
                .map(({ type, label, icon }) => (
                  <button
                    type="button"
                    key={type}
                    onClick={() => handleAddProperty(type)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      padding: '6px 8px',
                      background: 'none',
                      border: 'none',
                      cursor: 'pointer',
                      textAlign: 'left',
                      fontSize: 13,
                      color: '#37352f',
                      borderRadius: 4,
                      transition: 'background 0.1s',
                    }}
                    onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(55,53,47,0.04)'}
                    onMouseLeave={(e) => e.currentTarget.style.background = 'none'}
                  >
                    <span style={{ color: 'rgba(55,53,47,0.5)', display: 'flex' }}>{icon}</span>
                    <span>{label}</span>
                  </button>
                ))}
            </div>
          </div>
        </>
      )}


      {/* Header menu for column editing */}
      {headerMenu && (
        <>
          {/* Backdrop */}
          <div
            onClick={() => {
              setHeaderMenu(null)
              setEditingColumnName(null)
            }}
            style={{
              position: 'fixed',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              zIndex: 1000,
            }}
          />
          {/* Menu */}
          <div
            ref={headerMenuRef}
            style={{
              position: 'fixed',
              top: headerMenu.bounds.y + 4,
              left: headerMenu.bounds.x,
              minWidth: 220,
              background: '#ffffff',
              border: '1px solid rgba(55,53,47,0.09)',
              borderRadius: 6,
              boxShadow: 'rgba(15, 15, 15, 0.05) 0px 0px 0px 1px, rgba(15, 15, 15, 0.1) 0px 3px 6px, rgba(15, 15, 15, 0.2) 0px 9px 24px',
              zIndex: 1001,
              padding: '6px 0',
            }}
          >
            {/* Column name editing */}
            {editingColumnName === headerMenu.property.id ? (
              <div style={{ padding: '8px 12px' }}>
                <input
                  type="text"
                  value={columnNameInput}
                  onChange={(e) => setColumnNameInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      handleColumnRename(headerMenu.property.id, columnNameInput)
                    }
                    if (e.key === 'Escape') {
                      setEditingColumnName(null)
                      setColumnNameInput('')
                    }
                  }}
                  onBlur={() => handleColumnRename(headerMenu.property.id, columnNameInput)}
                  autoFocus
                  style={{
                    width: '100%',
                    padding: '6px 10px',
                    border: '1px solid #2383e2',
                    borderRadius: 4,
                    fontSize: 14,
                    outline: 'none',
                  }}
                />
              </div>
            ) : (
              <>
                {/* Property name display */}
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '6px 12px',
                  fontSize: 12,
                  fontWeight: 500,
                  color: 'rgba(55,53,47,0.65)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.5px',
                }}>
                  {getPropertyIcon(headerMenu.property.type)}
                  <span>{headerMenu.property.type.replace('_', ' ')}</span>
                </div>
                <div style={{ height: 1, background: 'rgba(55,53,47,0.09)', margin: '4px 0' }} />

                {/* Rename option */}
                <button
                  type="button"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    setEditingColumnName(headerMenu.property.id)
                    setColumnNameInput(headerMenu.property.name)
                  }}
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
                  <Edit2 size={14} style={{ opacity: 0.65 }} />
                  <span>Rename</span>
                </button>

                {/* Hide option */}
                <button
                  type="button"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    togglePropertyVisibility(headerMenu.property.id)
                    setHeaderMenu(null)
                  }}
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
                  <EyeOff size={14} style={{ opacity: 0.65 }} />
                  <span>Hide in view</span>
                </button>

                <div style={{ height: 1, background: 'rgba(55,53,47,0.09)', margin: '4px 0' }} />

                {/* Delete option */}
                <button
                  type="button"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    if (confirm(`Delete "${headerMenu.property.name}" property? This cannot be undone.`)) {
                      onDeleteProperty(headerMenu.property.id)
                      setHeaderMenu(null)
                    }
                  }}
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
                    color: '#eb5757',
                    transition: 'background 0.1s',
                  }}
                  onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(235,87,87,0.04)'}
                  onMouseLeave={(e) => e.currentTarget.style.background = 'none'}
                >
                  <Trash2 size={14} />
                  <span>Delete property</span>
                </button>
              </>
            )}
          </div>
        </>
      )}

      {/* Notion-style OPEN button overlay */}
      {hoveredRow !== null && openButtonBounds && (
        <button
          type="button"
          onClick={handleOpenButtonClick}
          onMouseEnter={() => {
            setIsOpenButtonHovered(true)
            if (hideTimeoutRef.current) {
              clearTimeout(hideTimeoutRef.current)
              hideTimeoutRef.current = null
            }
          }}
          onMouseLeave={() => {
            setIsOpenButtonHovered(false)
            // Hide after leaving button
            hideTimeoutRef.current = window.setTimeout(() => {
              setHoveredRow(null)
              setOpenButtonBounds(null)
            }, 100)
          }}
          style={{
            position: 'fixed',
            top: openButtonBounds.y + (openButtonBounds.height - 22) / 2,
            left: openButtonBounds.x + openButtonBounds.width - 54,
            height: 22,
            padding: '0 6px',
            display: 'flex',
            alignItems: 'center',
            gap: 4,
            background: isOpenButtonHovered ? 'rgba(55, 53, 47, 0.16)' : 'rgba(55, 53, 47, 0.08)',
            border: 'none',
            borderRadius: 4,
            cursor: 'pointer',
            fontSize: 11,
            fontWeight: 500,
            color: isOpenButtonHovered ? 'rgba(55, 53, 47, 0.9)' : 'rgba(55, 53, 47, 0.6)',
            textTransform: 'uppercase',
            letterSpacing: '0.5px',
            zIndex: 10,
            transition: 'all 0.15s ease',
            boxShadow: '0 0 0 1px rgba(55, 53, 47, 0.05)',
          }}
        >
          <Expand size={12} strokeWidth={2} />
          <span>Open</span>
        </button>
      )}

      {/* Side Peek Panel */}
      {detailRow && database && (
        <SidePeek
          row={detailRow}
          database={database}
          rows={filteredRows}
          currentIndex={detailRowIndex}
          isOpen={!!detailRow}
          onClose={() => {
            setDetailRow(null)
            setDetailRowIndex(-1)
          }}
          onNavigate={handleNavigate}
          onUpdate={handleRowUpdate}
          onDelete={(rowId) => {
            onDeleteRow(rowId)
            setDetailRow(null)
            setDetailRowIndex(-1)
          }}
          onAddProperty={onAddProperty}
        />
      )}
    </div>
  )
}

export default TableView
