import { useMemo, useState, useCallback } from 'react'
import { Box, Table, Text, Group, ActionIcon, Badge, Select, Tooltip } from '@mantine/core'
import { IconChevronUp, IconChevronDown, IconChevronLeft, IconChevronRight } from '@tabler/icons-react'
import type { QueryResult, ResultColumn } from '../../../api/types'
import type {
  TableSettings,
  TableColumnConfig,
  FormatCondition,
  ColumnStats,
  SortState,
  PaginationState
} from './types'
import MiniBar from './MiniBar'
import classes from './TableVisualization.module.css'

interface TableVisualizationProps {
  result: QueryResult
  settings?: TableSettings
  height?: number
  onCellClick?: (row: Record<string, any>, column: ResultColumn) => void
  onSettingsChange?: (settings: TableSettings) => void
}

export default function TableVisualization({
  result,
  settings = {},
  height,
  onCellClick,
}: TableVisualizationProps) {
  const { columns, rows } = result
  const {
    showRowIndex = false,
    paginateResults = false,
    pageSize: defaultPageSize = 25,
    conditionalFormatting = [],
    maxRows = 100,
    maxHeight = 500,
    striped = true,
    highlightOnHover = true,
    stickyHeader = true,
  } = settings

  // Sort state
  const [sortState, setSortState] = useState<SortState>({ column: null, direction: 'asc' })

  // Pagination state
  const [pagination, setPagination] = useState<PaginationState>({
    currentPage: 1,
    pageSize: defaultPageSize,
    totalRows: rows.length,
  })

  // Get column config or defaults
  const getColumnConfig = useCallback((col: ResultColumn): TableColumnConfig => {
    const configured = settings.columns?.find(c => c.name === col.name)
    return configured || {
      name: col.name,
      displayName: col.display_name || col.name,
      visible: true,
      position: columns.indexOf(col),
    }
  }, [settings.columns, columns])

  // Visible columns in order
  const visibleColumns = useMemo(() => {
    const configs = columns.map(col => ({
      column: col,
      config: getColumnConfig(col),
    }))

    return configs
      .filter(({ config }) => config.visible !== false)
      .sort((a, b) => (a.config.position ?? 0) - (b.config.position ?? 0))
  }, [columns, getColumnConfig])

  // Calculate column stats for mini bars and color ranges
  const columnStats = useMemo(() => {
    const stats: Record<string, ColumnStats> = {}

    columns.forEach(col => {
      if (col.type === 'number' || col.type === 'integer' || col.type === 'float') {
        const values = rows
          .map(row => row[col.name])
          .filter(v => v !== null && v !== undefined && typeof v === 'number')

        if (values.length > 0) {
          stats[col.name] = {
            min: Math.min(...values),
            max: Math.max(...values),
            distinctCount: new Set(values).size,
          }
        }
      }
    })

    return stats
  }, [columns, rows])

  // Sort data
  const sortedRows = useMemo(() => {
    if (!sortState.column) return rows

    return [...rows].sort((a, b) => {
      const aVal = a[sortState.column!]
      const bVal = b[sortState.column!]

      if (aVal === null || aVal === undefined) return 1
      if (bVal === null || bVal === undefined) return -1

      let comparison = 0
      if (typeof aVal === 'number' && typeof bVal === 'number') {
        comparison = aVal - bVal
      } else {
        comparison = String(aVal).localeCompare(String(bVal))
      }

      return sortState.direction === 'asc' ? comparison : -comparison
    })
  }, [rows, sortState])

  // Paginate data
  const displayRows = useMemo(() => {
    if (paginateResults) {
      const start = (pagination.currentPage - 1) * pagination.pageSize
      const end = start + pagination.pageSize
      return sortedRows.slice(start, end)
    }
    return sortedRows.slice(0, maxRows)
  }, [sortedRows, paginateResults, pagination, maxRows])

  // Handle sort click
  const handleSort = (columnName: string) => {
    setSortState(prev => ({
      column: columnName,
      direction: prev.column === columnName && prev.direction === 'asc' ? 'desc' : 'asc',
    }))
  }

  // Format cell value
  const formatValue = (value: any, column: ResultColumn, config: TableColumnConfig): React.ReactNode => {
    if (value === null || value === undefined) return <span className={classes.nullValue}>-</span>

    const format = config.format

    // Number formatting
    if ((column.type === 'number' || column.type === 'integer' || column.type === 'float') && typeof value === 'number') {
      // Mini bar
      if (format?.showMiniBar && columnStats[column.name]) {
        return (
          <MiniBar
            value={value}
            min={columnStats[column.name].min}
            max={columnStats[column.name].max}
          />
        )
      }

      const options: Intl.NumberFormatOptions = {
        minimumFractionDigits: format?.decimals ?? 0,
        maximumFractionDigits: format?.decimals ?? 2,
        useGrouping: format?.useGrouping ?? true,
      }

      if (format?.type === 'currency') {
        options.style = 'currency'
        options.currency = format.currency || 'USD'
        options.currencyDisplay = format.currencyStyle || 'symbol'
      } else if (format?.type === 'percent') {
        options.style = 'percent'
      }

      let formatted = new Intl.NumberFormat(undefined, options).format(value)

      if (format?.prefix) formatted = format.prefix + formatted
      if (format?.suffix) formatted = formatted + format.suffix

      const isNegative = value < 0
      return (
        <span className={isNegative && format?.negativeInRed ? classes.negativeValue : undefined}>
          {formatted}
        </span>
      )
    }

    // Date formatting
    if ((column.type === 'datetime' || column.type === 'date' || column.type === 'timestamp') && value) {
      try {
        const date = new Date(value)
        const options: Intl.DateTimeFormatOptions = {}

        const dateStyle = format?.dateStyle || 'medium'
        switch (dateStyle) {
          case 'short':
            options.dateStyle = 'short'
            break
          case 'medium':
            options.dateStyle = 'medium'
            break
          case 'long':
            options.dateStyle = 'long'
            break
          case 'full':
            options.dateStyle = 'full'
            break
        }

        const timeStyle = format?.timeStyle || (column.type === 'datetime' || column.type === 'timestamp' ? 'short' : 'none')
        if (timeStyle && timeStyle !== 'none') {
          switch (timeStyle) {
            case 'short':
              options.timeStyle = 'short'
              break
            case 'medium':
              options.timeStyle = 'medium'
              break
            case 'long':
              options.timeStyle = 'long'
              break
          }
        }

        return new Intl.DateTimeFormat(undefined, options).format(date)
      } catch {
        return String(value)
      }
    }

    // Link formatting
    if (format?.type === 'link' || format?.type === 'email') {
      const url = format.type === 'email' ? `mailto:${value}` : String(value)
      return (
        <a
          href={url}
          target={format.openInNewTab ? '_blank' : undefined}
          rel={format.openInNewTab ? 'noopener noreferrer' : undefined}
          className={classes.link}
        >
          {format.linkText || String(value)}
        </a>
      )
    }

    // Image formatting
    if (format?.type === 'image') {
      return (
        <img
          src={String(value)}
          alt=""
          className={classes.cellImage}
        />
      )
    }

    return String(value)
  }

  // Evaluate conditional formatting for a cell
  const getCellStyle = (value: any, columnName: string, row: Record<string, any>): React.CSSProperties => {
    const style: React.CSSProperties = {}

    for (const rule of conditionalFormatting) {
      if (!rule.columns.includes(columnName) && !rule.highlightWholeRow) continue

      if (rule.style === 'single' && rule.condition) {
        const targetValue = rule.columns[0] ? row[rule.columns[0]] : value
        if (evaluateCondition(targetValue, rule.condition)) {
          style.backgroundColor = hexToRgba(rule.color, 0.3)
        }
      } else if (rule.style === 'range' && rule.colorRange && columnStats[columnName]) {
        const numValue = typeof value === 'number' ? value : null
        if (numValue !== null) {
          const color = getColorFromRange(numValue, rule.colorRange, columnStats[columnName])
          style.backgroundColor = hexToRgba(color, 0.3)
        }
      }
    }

    return style
  }

  // Check if row should be highlighted
  const getRowHighlight = (row: Record<string, any>): string | undefined => {
    for (const rule of conditionalFormatting) {
      if (!rule.highlightWholeRow) continue

      if (rule.style === 'single' && rule.condition) {
        const targetValue = rule.columns[0] ? row[rule.columns[0]] : null
        if (targetValue !== null && evaluateCondition(targetValue, rule.condition)) {
          return hexToRgba(rule.color, 0.15)
        }
      }
    }
    return undefined
  }

  // Get column alignment
  const getAlignment = (column: ResultColumn, config: TableColumnConfig): 'left' | 'center' | 'right' => {
    if (config.alignment && config.alignment !== 'auto') return config.alignment

    // Auto alignment based on type
    if (column.type === 'number' || column.type === 'integer' || column.type === 'float') {
      return 'right'
    }
    return 'left'
  }

  // Pagination controls
  const totalPages = Math.ceil(rows.length / pagination.pageSize)
  const startRow = (pagination.currentPage - 1) * pagination.pageSize + 1
  const endRow = Math.min(pagination.currentPage * pagination.pageSize, rows.length)

  return (
    <Box className={classes.container} style={{ maxHeight: height || maxHeight }}>
      <Box className={classes.tableWrapper}>
        <Table
          striped={striped}
          highlightOnHover={highlightOnHover}
          withTableBorder
          className={classes.table}
          stickyHeader={stickyHeader}
        >
          <Table.Thead className={classes.thead}>
            <Table.Tr>
              {showRowIndex && (
                <Table.Th className={classes.rowIndexHeader}>#</Table.Th>
              )}
              {visibleColumns.map(({ column, config }) => (
                <Table.Th
                  key={column.name}
                  className={classes.th}
                  style={{
                    textAlign: getAlignment(column, config),
                    width: config.width,
                    minWidth: config.width,
                  }}
                  onClick={() => handleSort(column.name)}
                >
                  <Group gap={4} justify={getAlignment(column, config) === 'right' ? 'flex-end' : 'flex-start'} wrap="nowrap">
                    <span className={classes.headerText}>
                      {config.displayName || column.display_name || column.name}
                    </span>
                    {sortState.column === column.name && (
                      sortState.direction === 'asc'
                        ? <IconChevronUp size={14} />
                        : <IconChevronDown size={14} />
                    )}
                  </Group>
                </Table.Th>
              ))}
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {displayRows.map((row, rowIndex) => {
              const rowHighlight = getRowHighlight(row)
              const actualRowIndex = paginateResults
                ? (pagination.currentPage - 1) * pagination.pageSize + rowIndex + 1
                : rowIndex + 1

              return (
                <Table.Tr
                  key={rowIndex}
                  style={rowHighlight ? { backgroundColor: rowHighlight } : undefined}
                >
                  {showRowIndex && (
                    <Table.Td className={classes.rowIndexCell}>
                      <Badge
                        variant="light"
                        color="brand"
                        size="sm"
                        className={classes.rowIndexBadge}
                      >
                        {actualRowIndex}
                      </Badge>
                    </Table.Td>
                  )}
                  {visibleColumns.map(({ column, config }) => {
                    const value = row[column.name]
                    const cellStyle = getCellStyle(value, column.name, row)
                    const alignment = getAlignment(column, config)

                    return (
                      <Table.Td
                        key={column.name}
                        className={classes.td}
                        style={{
                          textAlign: alignment,
                          ...cellStyle,
                        }}
                        onClick={() => onCellClick?.(row, column)}
                      >
                        <div
                          className={classes.cellContent}
                          style={{
                            justifyContent: alignment === 'right' ? 'flex-end' : alignment === 'center' ? 'center' : 'flex-start',
                          }}
                        >
                          {formatValue(value, column, config)}
                        </div>
                      </Table.Td>
                    )
                  })}
                </Table.Tr>
              )
            })}
          </Table.Tbody>
        </Table>
      </Box>

      {/* Pagination / Row count */}
      {paginateResults ? (
        <Group justify="flex-end" gap="sm" className={classes.pagination}>
          <Text size="sm" c="dimmed">
            Rows {startRow}-{endRow} of {rows.length}
          </Text>
          <Group gap={4}>
            <Tooltip label="Previous page">
              <ActionIcon
                variant="subtle"
                size="sm"
                disabled={pagination.currentPage === 1}
                onClick={() => setPagination(p => ({ ...p, currentPage: p.currentPage - 1 }))}
              >
                <IconChevronLeft size={16} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label="Next page">
              <ActionIcon
                variant="subtle"
                size="sm"
                disabled={pagination.currentPage >= totalPages}
                onClick={() => setPagination(p => ({ ...p, currentPage: p.currentPage + 1 }))}
              >
                <IconChevronRight size={16} />
              </ActionIcon>
            </Tooltip>
          </Group>
          <Select
            size="xs"
            w={70}
            value={String(pagination.pageSize)}
            data={['10', '25', '50', '100']}
            onChange={(value) => setPagination(p => ({
              ...p,
              pageSize: Number(value) || 25,
              currentPage: 1,
            }))}
          />
        </Group>
      ) : rows.length > maxRows ? (
        <Text size="sm" c="dimmed" ta="center" py="sm">
          Showing {maxRows} of {rows.length} rows
        </Text>
      ) : null}
    </Box>
  )
}

// Helper: Evaluate condition
function evaluateCondition(value: any, condition: FormatCondition): boolean {
  const { operator, value: condValue, valueEnd } = condition

  switch (operator) {
    case 'equals':
      return value === condValue
    case 'not-equals':
      return value !== condValue
    case 'greater-than':
      return Number(value) > Number(condValue)
    case 'less-than':
      return Number(value) < Number(condValue)
    case 'greater-or-equal':
      return Number(value) >= Number(condValue)
    case 'less-or-equal':
      return Number(value) <= Number(condValue)
    case 'between':
      return Number(value) >= Number(condValue) && Number(value) <= Number(valueEnd)
    case 'is-null':
      return value === null || value === undefined
    case 'is-not-null':
      return value !== null && value !== undefined
    case 'contains':
      return String(value).toLowerCase().includes(String(condValue).toLowerCase())
    case 'starts-with':
      return String(value).toLowerCase().startsWith(String(condValue).toLowerCase())
    case 'ends-with':
      return String(value).toLowerCase().endsWith(String(condValue).toLowerCase())
    default:
      return false
  }
}

// Helper: Get color from range
function getColorFromRange(
  value: number,
  colorRange: { type: string; min?: number; max?: number; midpoint?: number; colors: string[] },
  stats: ColumnStats
): string {
  const min = colorRange.min ?? stats.min
  const max = colorRange.max ?? stats.max
  const { midpoint, colors, type } = colorRange

  if (max === min) return colors[0]

  let position: number
  if (type === 'diverging' && midpoint !== undefined) {
    if (value < midpoint) {
      position = 0.5 * ((value - min) / (midpoint - min))
    } else {
      position = 0.5 + 0.5 * ((value - midpoint) / (max - midpoint))
    }
  } else {
    position = (value - min) / (max - min)
  }

  position = Math.max(0, Math.min(1, position))

  // Interpolate between colors
  if (colors.length === 2) {
    return interpolateColor(colors[0], colors[1], position)
  } else if (colors.length === 3) {
    if (position < 0.5) {
      return interpolateColor(colors[0], colors[1], position * 2)
    } else {
      return interpolateColor(colors[1], colors[2], (position - 0.5) * 2)
    }
  }

  return colors[0]
}

// Helper: Interpolate between two colors
function interpolateColor(color1: string, color2: string, factor: number): string {
  const r1 = parseInt(color1.slice(1, 3), 16)
  const g1 = parseInt(color1.slice(3, 5), 16)
  const b1 = parseInt(color1.slice(5, 7), 16)

  const r2 = parseInt(color2.slice(1, 3), 16)
  const g2 = parseInt(color2.slice(3, 5), 16)
  const b2 = parseInt(color2.slice(5, 7), 16)

  const r = Math.round(r1 + (r2 - r1) * factor)
  const g = Math.round(g1 + (g2 - g1) * factor)
  const b = Math.round(b1 + (b2 - b1) * factor)

  return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`
}

// Helper: Convert hex to rgba
function hexToRgba(hex: string, alpha: number): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}
