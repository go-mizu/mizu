import { useState, useRef, useMemo, useCallback, useEffect } from 'react'
import { ChevronLeft, ChevronRight, ZoomIn, ZoomOut, Calendar } from 'lucide-react'
import { format, addDays, addWeeks, addMonths, startOfWeek, startOfMonth, endOfMonth, differenceInDays, parseISO, isValid, isSameDay } from 'date-fns'
import { DatabaseRow, Property } from '../../api/client'

interface TimelineViewProps {
  rows: DatabaseRow[]
  properties: Property[]
  groupBy: string | null
  onAddRow: (initialProperties?: Record<string, unknown>) => Promise<DatabaseRow | null>
  onUpdateRow: (rowId: string, updates: Record<string, unknown>) => void
  onDeleteRow: (rowId: string) => void
  onAddProperty: (property: Omit<Property, 'id'>) => void
  onUpdateProperty: (propertyId: string, updates: Partial<Property>) => void
  onDeleteProperty: (propertyId: string) => void
}

type ZoomLevel = 'day' | 'week' | 'month' | 'quarter'

const ZOOM_CONFIGS = {
  day: { label: 'Day', columnWidth: 50, dateFormat: 'd', headerFormat: 'MMM yyyy' },
  week: { label: 'Week', columnWidth: 100, dateFormat: "'W'w", headerFormat: 'MMM yyyy' },
  month: { label: 'Month', columnWidth: 150, dateFormat: 'MMM', headerFormat: 'yyyy' },
  quarter: { label: 'Quarter', columnWidth: 200, dateFormat: "'Q'Q", headerFormat: 'yyyy' },
}

const COLORS = [
  '#2383e2', '#0f7b6c', '#d9730d', '#9065b0',
  '#c14c8a', '#e03e3e', '#dfab01', '#64473a',
]

export function TimelineView({
  rows,
  properties,
  groupBy,
  onAddRow,
  onUpdateRow,
  onDeleteRow,
}: TimelineViewProps) {
  const [zoom, setZoom] = useState<ZoomLevel>('week')
  const [startDate, setStartDate] = useState(() => startOfMonth(new Date()))
  const [draggedRow, setDraggedRow] = useState<string | null>(null)
  const [dragStartX, setDragStartX] = useState(0)
  const [dragType, setDragType] = useState<'move' | 'resize-start' | 'resize-end'>('move')
  const containerRef = useRef<HTMLDivElement>(null)
  const timelineRef = useRef<HTMLDivElement>(null)
  const [selectedRow, setSelectedRow] = useState<string | null>(null)

  const config = ZOOM_CONFIGS[zoom]

  // Find date properties automatically
  const dateProperty = properties.find(p => p.type === 'date')?.id || ''
  const endDateProperty = properties.find(p => p.type === 'date' && p.id !== dateProperty)?.id

  // Get group property
  const groupProp = groupBy
    ? properties.find(p => p.id === groupBy)
    : null

  // Calculate timeline columns based on zoom level
  const columns = useMemo(() => {
    const cols: { date: Date; label: string }[] = []
    let current = startDate
    const numColumns = zoom === 'day' ? 60 : zoom === 'week' ? 16 : zoom === 'month' ? 12 : 8

    for (let i = 0; i < numColumns; i++) {
      cols.push({
        date: current,
        label: format(current, config.dateFormat),
      })
      if (zoom === 'day') current = addDays(current, 1)
      else if (zoom === 'week') current = addWeeks(current, 1)
      else if (zoom === 'month') current = addMonths(current, 1)
      else current = addMonths(current, 3)
    }
    return cols
  }, [startDate, zoom, config.dateFormat])

  // Group rows
  const groupedRows = useMemo(() => {
    if (!groupProp) {
      return [{ id: 'all', name: 'All items', color: undefined as string | undefined, rows }]
    }

    const groups: Record<string, DatabaseRow[]> = {}
    const ungrouped: DatabaseRow[] = []

    rows.forEach(row => {
      const value = row.properties[groupProp.id] as string
      if (value) {
        if (!groups[value]) groups[value] = []
        groups[value].push(row)
      } else {
        ungrouped.push(row)
      }
    })

    const result = Object.entries(groups).map(([id, rows]) => {
      const option = groupProp.options?.find(o => o.id === id)
      return { id, name: option?.name || id, color: option?.color, rows }
    })

    if (ungrouped.length > 0) {
      result.push({ id: 'ungrouped', name: 'No group', color: undefined, rows: ungrouped })
    }

    return result
  }, [rows, groupProp])

  // Calculate bar position
  const getBarPosition = useCallback((row: DatabaseRow) => {
    const startValue = row.properties[dateProperty] as string
    if (!startValue) return null

    let start: Date
    try {
      start = parseISO(startValue)
      if (!isValid(start)) return null
    } catch {
      return null
    }

    let end = start
    if (endDateProperty) {
      const endValue = row.properties[endDateProperty] as string
      if (endValue) {
        try {
          const parsed = parseISO(endValue)
          if (isValid(parsed)) end = parsed
        } catch {}
      }
    }

    const startDiff = differenceInDays(start, columns[0].date)
    const duration = Math.max(1, differenceInDays(end, start) + 1)

    let left: number
    let width: number

    if (zoom === 'day') {
      left = startDiff * config.columnWidth
      width = duration * config.columnWidth
    } else if (zoom === 'week') {
      left = (startDiff / 7) * config.columnWidth
      width = Math.max((duration / 7) * config.columnWidth, 20)
    } else if (zoom === 'month') {
      left = (startDiff / 30) * config.columnWidth
      width = Math.max((duration / 30) * config.columnWidth, 20)
    } else {
      left = (startDiff / 90) * config.columnWidth
      width = Math.max((duration / 90) * config.columnWidth, 20)
    }

    return { left, width }
  }, [dateProperty, endDateProperty, columns, zoom, config.columnWidth])

  // Navigate timeline
  const navigate = (direction: 'prev' | 'next') => {
    const delta = direction === 'next' ? 1 : -1
    if (zoom === 'day') setStartDate(addWeeks(startDate, delta))
    else if (zoom === 'week') setStartDate(addMonths(startDate, delta))
    else if (zoom === 'month') setStartDate(addMonths(startDate, delta * 3))
    else setStartDate(addMonths(startDate, delta * 12))
  }

  // Go to today
  const goToToday = () => {
    if (zoom === 'day') setStartDate(startOfWeek(new Date()))
    else if (zoom === 'week') setStartDate(startOfMonth(new Date()))
    else if (zoom === 'month') setStartDate(startOfMonth(new Date()))
    else setStartDate(startOfMonth(new Date()))
  }

  // Zoom controls
  const zoomIn = () => {
    if (zoom === 'quarter') setZoom('month')
    else if (zoom === 'month') setZoom('week')
    else if (zoom === 'week') setZoom('day')
  }

  const zoomOut = () => {
    if (zoom === 'day') setZoom('week')
    else if (zoom === 'week') setZoom('month')
    else if (zoom === 'month') setZoom('quarter')
  }

  // Handle bar drag
  const handleMouseDown = (e: React.MouseEvent, rowId: string, type: 'move' | 'resize-start' | 'resize-end') => {
    e.stopPropagation()
    setDraggedRow(rowId)
    setDragStartX(e.clientX)
    setDragType(type)
  }

  const handleMouseUp = () => {
    setDraggedRow(null)
  }

  useEffect(() => {
    document.addEventListener('mouseup', handleMouseUp)
    return () => document.removeEventListener('mouseup', handleMouseUp)
  }, [])

  // Calculate today line position
  const todayPosition = useMemo(() => {
    const diff = differenceInDays(new Date(), columns[0].date)
    if (zoom === 'day') return diff * config.columnWidth
    if (zoom === 'week') return (diff / 7) * config.columnWidth
    if (zoom === 'month') return (diff / 30) * config.columnWidth
    return (diff / 90) * config.columnWidth
  }, [columns, zoom, config.columnWidth])

  const totalWidth = columns.length * config.columnWidth

  return (
    <div className="timeline-view" ref={containerRef}>
      {/* Toolbar */}
      <div className="timeline-toolbar">
        <div className="timeline-nav">
          <button className="icon-btn" onClick={() => navigate('prev')} title="Previous">
            <ChevronLeft size={16} />
          </button>
          <button className="btn-secondary" onClick={goToToday}>
            Today
          </button>
          <button className="icon-btn" onClick={() => navigate('next')} title="Next">
            <ChevronRight size={16} />
          </button>
          <span className="timeline-period">
            {format(startDate, config.headerFormat)}
          </span>
        </div>
        <div className="timeline-zoom">
          <button className="icon-btn" onClick={zoomIn} disabled={zoom === 'day'} title="Zoom in">
            <ZoomIn size={16} />
          </button>
          <span className="zoom-label">{config.label}</span>
          <button className="icon-btn" onClick={zoomOut} disabled={zoom === 'quarter'} title="Zoom out">
            <ZoomOut size={16} />
          </button>
        </div>
      </div>

      {/* Timeline */}
      <div className="timeline-container" ref={timelineRef}>
        {/* Header */}
        <div className="timeline-header" style={{ width: totalWidth }}>
          <div className="timeline-sidebar-header">
            <span>Name</span>
          </div>
          <div className="timeline-columns">
            {columns.map((col, i) => (
              <div
                key={i}
                className={`timeline-column-header ${isSameDay(col.date, new Date()) ? 'today' : ''}`}
                style={{ width: config.columnWidth }}
              >
                {col.label}
              </div>
            ))}
          </div>
        </div>

        {/* Body */}
        <div className="timeline-body" style={{ width: totalWidth }}>
          {groupedRows.map((group, groupIndex) => (
            <div key={group.id} className="timeline-group">
              {groupProp && (
                <div className="timeline-group-header">
                  <span
                    className="group-indicator"
                    style={{ backgroundColor: group.color || COLORS[groupIndex % COLORS.length] }}
                  />
                  <span>{group.name}</span>
                  <span className="group-count">{group.rows.length}</span>
                </div>
              )}

              {group.rows.map((row, rowIndex) => {
                const pos = getBarPosition(row)
                const title = row.properties.title as string || 'Untitled'

                return (
                  <div key={row.id} className={`timeline-row ${selectedRow === row.id ? 'selected' : ''}`}>
                    <div className="timeline-sidebar-cell" onClick={() => setSelectedRow(row.id)}>
                      <span className="row-title">{title}</span>
                    </div>
                    <div className="timeline-grid">
                      {columns.map((col, i) => (
                        <div
                          key={i}
                          className={`timeline-cell ${isSameDay(col.date, new Date()) ? 'today' : ''}`}
                          style={{ width: config.columnWidth }}
                        />
                      ))}

                      {/* Bar */}
                      {pos && (
                        <div
                          className={`timeline-bar ${draggedRow === row.id ? 'dragging' : ''}`}
                          style={{
                            left: pos.left,
                            width: pos.width,
                            backgroundColor: COLORS[rowIndex % COLORS.length],
                          }}
                          onMouseDown={(e) => handleMouseDown(e, row.id, 'move')}
                          onClick={() => setSelectedRow(row.id)}
                        >
                          {/* Resize handles */}
                          <div
                            className="resize-handle start"
                            onMouseDown={(e) => handleMouseDown(e, row.id, 'resize-start')}
                          />
                          <span className="bar-label">{title}</span>
                          <div
                            className="resize-handle end"
                            onMouseDown={(e) => handleMouseDown(e, row.id, 'resize-end')}
                          />
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          ))}

          {/* Today line */}
          {todayPosition >= 0 && todayPosition <= totalWidth && (
            <div
              className="today-line"
              style={{ left: 200 + todayPosition }}
            />
          )}
        </div>
      </div>

      {/* Empty state */}
      {rows.length === 0 && (
        <div className="empty-state">
          <Calendar size={48} />
          <h3>No items to display</h3>
          <p>Add items with a date property to see them on the timeline.</p>
        </div>
      )}

      {/* Add row button */}
      <button className="add-row-btn" onClick={() => onAddRow()}>
        <svg width="12" height="12" viewBox="0 0 12 12">
          <path d="M6 2v8M2 6h8" stroke="currentColor" strokeWidth="1.5" />
        </svg>
        <span>New</span>
      </button>
    </div>
  )
}
