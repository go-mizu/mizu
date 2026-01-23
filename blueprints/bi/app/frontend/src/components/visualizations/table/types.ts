// Table visualization types - Metabase feature parity

export interface TableSettings {
  // Card-level settings
  title?: string
  description?: string
  hideIfNoResults?: boolean
  paginateResults?: boolean
  showRowIndex?: boolean

  // Column configuration
  columns?: TableColumnConfig[]

  // Conditional formatting rules
  conditionalFormatting?: ConditionalFormattingRule[]

  // Pagination
  pageSize?: number  // Default: 10, options: 10, 25, 50, 100

  // Display
  maxHeight?: number
  maxRows?: number
  stickyHeader?: boolean
  striped?: boolean
  highlightOnHover?: boolean
}

export interface TableColumnConfig {
  name: string                    // Column identifier
  displayName?: string            // Custom display name
  visible: boolean                // Show/hide column
  position: number                // Column order
  width?: number                  // Fixed width in pixels
  alignment?: 'left' | 'center' | 'right' | 'auto'
  wrap?: boolean                  // Text wrapping

  // Formatting
  format?: ColumnFormat

  // Click behavior
  clickBehavior?: ColumnClickBehavior
}

export interface ColumnFormat {
  type: 'auto' | 'text' | 'number' | 'currency' | 'percent' | 'date' | 'link' | 'email' | 'image'

  // Number formatting
  decimals?: number
  prefix?: string
  suffix?: string
  useGrouping?: boolean           // Thousands separator
  negativeInRed?: boolean
  showMiniBar?: boolean           // Mini bar chart in cell

  // Currency
  currency?: string               // USD, EUR, etc.
  currencyStyle?: 'symbol' | 'code' | 'name'

  // Date formatting
  dateStyle?: 'short' | 'medium' | 'long' | 'full'
  timeStyle?: 'none' | 'short' | 'medium' | 'long'

  // Link formatting
  linkText?: string               // Static text or template
  linkUrl?: string                // URL template with {{column}} placeholders
  openInNewTab?: boolean
}

export interface ColumnClickBehavior {
  type: 'none' | 'link' | 'filter' | 'detail' | 'custom'
  url?: string                    // For link type
  targetDashboard?: string        // For custom navigation
  parameterMapping?: Record<string, string>  // Map columns to parameters
}

export interface ConditionalFormattingRule {
  id: string
  columns: string[]               // Which columns to affect
  style: 'single' | 'range'       // Single color or gradient

  // For single color style
  condition?: FormatCondition
  color: string                   // Hex color

  // For range style (gradient)
  colorRange?: ColorRange

  highlightWholeRow?: boolean     // Highlight row instead of cell
}

export interface FormatCondition {
  operator: 'equals' | 'not-equals' | 'greater-than' | 'less-than'
          | 'greater-or-equal' | 'less-or-equal' | 'between'
          | 'is-null' | 'is-not-null' | 'contains' | 'starts-with' | 'ends-with'
  value?: any
  valueEnd?: any                  // For 'between' operator
}

export interface ColorRange {
  type: 'diverging' | 'sequential'
  min?: number                    // Auto-detect if not specified
  max?: number
  midpoint?: number               // For diverging
  colors: string[]                // 2 or 3 colors
}

// Column statistics for mini bars and color ranges
export interface ColumnStats {
  min: number
  max: number
  distinctCount: number
}

// Pagination state
export interface PaginationState {
  currentPage: number
  pageSize: number
  totalRows: number
}

// Sort state (single column for backwards compatibility)
export interface SortState {
  column: string | null
  direction: 'asc' | 'desc'
}

// Multi-column sort state
export interface MultiSortState {
  columns: Array<{
    column: string
    direction: 'asc' | 'desc'
  }>
}
