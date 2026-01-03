import { useState, useCallback, useMemo, useRef } from 'react'
import {
  format,
  startOfMonth,
  endOfMonth,
  startOfWeek,
  endOfWeek,
  addDays,
  addMonths,
  subMonths,
  isSameMonth,
  isSameDay,
  isToday,
  parseISO,
  differenceInDays,
} from 'date-fns'
import { X, Plus, Calendar, ChevronLeft, ChevronRight, GripHorizontal } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { DatabaseRow, Property } from '../../api/client'

interface CalendarViewProps {
  rows: DatabaseRow[]
  properties: Property[]
  groupBy: string | null
  onAddRow: () => void
  onUpdateRow: (rowId: string, updates: Record<string, unknown>) => void
  onDeleteRow: (rowId: string) => void
  onAddProperty: (property: Omit<Property, 'id'>) => void
  onUpdateProperty: (propertyId: string, updates: Partial<Property>) => void
  onDeleteProperty: (propertyId: string) => void
}

type ViewMode = 'month' | 'week'

interface EventFormData {
  title: string
  date: Date
}

interface DragState {
  eventId: string
  originalDate: Date
  currentDate: Date | null
}

export function CalendarView({
  rows,
  properties,
  onAddRow,
  onUpdateRow,
  onDeleteRow,
}: CalendarViewProps) {
  const [currentDate, setCurrentDate] = useState(new Date())
  const [viewMode, setViewMode] = useState<ViewMode>('month')
  const [selectedDate, setSelectedDate] = useState<Date | null>(null)
  const [showEventForm, setShowEventForm] = useState(false)
  const [eventFormData, setEventFormData] = useState<EventFormData | null>(null)
  const [selectedEvent, setSelectedEvent] = useState<DatabaseRow | null>(null)
  const [isCreating, setIsCreating] = useState(false)
  const [dragState, setDragState] = useState<DragState | null>(null)
  const [hoveredDate, setHoveredDate] = useState<Date | null>(null)
  const titleInputRef = useRef<HTMLInputElement>(null)
  const calendarRef = useRef<HTMLDivElement>(null)

  // Find date property
  const dateProperty = useMemo(() => {
    return properties.find((p) => p.type === 'date')
  }, [properties])

  // Get title property
  const titleProperty = useMemo(() => {
    return properties.find((p) => p.type === 'text') || properties[0]
  }, [properties])

  // Map rows to calendar events
  const events = useMemo(() => {
    if (!dateProperty) return new Map<string, DatabaseRow[]>()

    const eventMap = new Map<string, DatabaseRow[]>()

    rows.forEach((row) => {
      const dateValue = row.properties[dateProperty.id]
      if (dateValue && typeof dateValue === 'string') {
        try {
          const date = parseISO(dateValue)
          const key = format(date, 'yyyy-MM-dd')
          const existing = eventMap.get(key) || []
          eventMap.set(key, [...existing, row])
        } catch {
          // Invalid date, skip
        }
      }
    })

    return eventMap
  }, [rows, dateProperty])

  // Generate calendar days
  const calendarDays = useMemo(() => {
    const monthStart = startOfMonth(currentDate)
    const monthEnd = endOfMonth(monthStart)
    const startDate = startOfWeek(monthStart)
    const endDate = endOfWeek(monthEnd)

    const days: Date[] = []
    let day = startDate

    while (day <= endDate) {
      days.push(day)
      day = addDays(day, 1)
    }

    return days
  }, [currentDate])

  // Navigation handlers
  const goToPreviousMonth = useCallback(() => {
    setCurrentDate((prev) => subMonths(prev, 1))
  }, [])

  const goToNextMonth = useCallback(() => {
    setCurrentDate((prev) => addMonths(prev, 1))
  }, [])

  const goToToday = useCallback(() => {
    setCurrentDate(new Date())
  }, [])

  // Handle date click - open event creation form
  const handleDateClick = useCallback((date: Date) => {
    if (dragState) return // Don't open form while dragging
    setSelectedDate(date)
    setEventFormData({ title: '', date })
    setSelectedEvent(null)
    setShowEventForm(true)
    setTimeout(() => titleInputRef.current?.focus(), 100)
  }, [dragState])

  // Handle event click - open event detail
  const handleEventClick = useCallback((event: DatabaseRow, e: React.MouseEvent) => {
    e.stopPropagation()
    if (dragState) return // Don't open form while dragging
    setSelectedEvent(event)
    const dateValue = dateProperty ? event.properties[dateProperty.id] : null
    setSelectedDate(dateValue ? parseISO(dateValue as string) : new Date())
    setEventFormData({
      title: titleProperty ? (event.properties[titleProperty.id] as string) || '' : '',
      date: dateValue ? parseISO(dateValue as string) : new Date(),
    })
    setShowEventForm(true)
    setTimeout(() => titleInputRef.current?.focus(), 100)
  }, [dateProperty, titleProperty, dragState])

  // Drag handlers for events
  const handleDragStart = useCallback((event: DatabaseRow, e: React.DragEvent) => {
    e.stopPropagation()
    if (!dateProperty) return

    const dateValue = event.properties[dateProperty.id]
    if (!dateValue) return

    const originalDate = parseISO(dateValue as string)
    setDragState({
      eventId: event.id,
      originalDate,
      currentDate: null,
    })

    // Set drag image
    const target = e.target as HTMLElement
    if (target) {
      e.dataTransfer.setDragImage(target, 10, 10)
    }
    e.dataTransfer.effectAllowed = 'move'
  }, [dateProperty])

  const handleDragOver = useCallback((date: Date, e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    e.dataTransfer.dropEffect = 'move'

    if (dragState && !isSameDay(date, dragState.currentDate || dragState.originalDate)) {
      setDragState({ ...dragState, currentDate: date })
      setHoveredDate(date)
    }
  }, [dragState])

  const handleDragLeave = useCallback(() => {
    setHoveredDate(null)
  }, [])

  const handleDrop = useCallback((targetDate: Date, e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()

    if (!dragState || !dateProperty) {
      setDragState(null)
      setHoveredDate(null)
      return
    }

    // Only update if the date actually changed
    if (!isSameDay(targetDate, dragState.originalDate)) {
      const newDateValue = format(targetDate, "yyyy-MM-dd'T'HH:mm:ss")
      onUpdateRow(dragState.eventId, { [dateProperty.id]: newDateValue })
    }

    setDragState(null)
    setHoveredDate(null)
  }, [dragState, dateProperty, onUpdateRow])

  const handleDragEnd = useCallback(() => {
    setDragState(null)
    setHoveredDate(null)
  }, [])

  // Handle create/update event
  const handleSaveEvent = useCallback(async () => {
    if (!eventFormData || !dateProperty || !titleProperty || isCreating) return

    setIsCreating(true)
    try {
      const updates: Record<string, unknown> = {
        [dateProperty.id]: format(eventFormData.date, "yyyy-MM-dd'T'HH:mm:ss"),
        [titleProperty.id]: eventFormData.title || 'Untitled',
      }

      if (selectedEvent) {
        onUpdateRow(selectedEvent.id, updates)
      } else {
        onAddRow()
      }
      setShowEventForm(false)
      setEventFormData(null)
      setSelectedEvent(null)
    } catch (err) {
      console.error('Failed to save event:', err)
    } finally {
      setIsCreating(false)
    }
  }, [eventFormData, dateProperty, titleProperty, selectedEvent, isCreating, onAddRow, onUpdateRow])

  // Handle delete event
  const handleDeleteEvent = useCallback(() => {
    if (!selectedEvent) return
    onDeleteRow(selectedEvent.id)
    setShowEventForm(false)
    setSelectedEvent(null)
    setEventFormData(null)
  }, [selectedEvent, onDeleteRow])

  // Handle form key events
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSaveEvent()
    }
    if (e.key === 'Escape') {
      setShowEventForm(false)
    }
  }, [handleSaveEvent])

  // Get events for a specific date
  const getEventsForDate = useCallback((date: Date) => {
    const key = format(date, 'yyyy-MM-dd')
    return events.get(key) || []
  }, [events])

  // Calculate if event is being dragged to this date
  const isDropTarget = useCallback((date: Date) => {
    return dragState && hoveredDate && isSameDay(date, hoveredDate)
  }, [dragState, hoveredDate])

  if (!dateProperty) {
    return (
      <div
        className="calendar-empty-state"
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '48px',
          color: 'var(--text-secondary)',
          fontSize: '14px',
        }}
      >
        <Calendar size={20} style={{ marginRight: '8px' }} />
        <p>Add a Date property to use Calendar view</p>
      </div>
    )
  }

  return (
    <div className="calendar-view" ref={calendarRef}>
      {/* Calendar header */}
      <div
        className="calendar-header"
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 16px',
          borderBottom: '1px solid var(--border-color)',
        }}
      >
        <div className="calendar-nav" style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
          <button
            onClick={goToPreviousMonth}
            title="Previous month"
            style={{
              width: '28px',
              height: '28px',
              borderRadius: '4px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'none',
              border: 'none',
              color: 'var(--text-secondary)',
              cursor: 'pointer',
              transition: 'background 0.15s',
            }}
            onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)' }}
            onMouseLeave={(e) => { e.currentTarget.style.background = 'none' }}
          >
            <ChevronLeft size={18} />
          </button>
          <button
            onClick={goToToday}
            style={{
              padding: '4px 12px',
              borderRadius: '4px',
              background: 'none',
              border: '1px solid var(--border-color)',
              color: 'var(--text-primary)',
              fontSize: '13px',
              fontWeight: 500,
              cursor: 'pointer',
              transition: 'background 0.15s',
            }}
            onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)' }}
            onMouseLeave={(e) => { e.currentTarget.style.background = 'none' }}
          >
            Today
          </button>
          <button
            onClick={goToNextMonth}
            title="Next month"
            style={{
              width: '28px',
              height: '28px',
              borderRadius: '4px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'none',
              border: 'none',
              color: 'var(--text-secondary)',
              cursor: 'pointer',
              transition: 'background 0.15s',
            }}
            onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)' }}
            onMouseLeave={(e) => { e.currentTarget.style.background = 'none' }}
          >
            <ChevronRight size={18} />
          </button>
        </div>

        <h2
          style={{
            fontSize: '16px',
            fontWeight: 600,
            color: 'var(--text-primary)',
            margin: 0,
          }}
        >
          {format(currentDate, 'MMMM yyyy')}
        </h2>

        <div
          className="view-toggle"
          style={{
            display: 'flex',
            background: 'var(--bg-secondary)',
            borderRadius: '6px',
            padding: '2px',
          }}
        >
          <button
            onClick={() => setViewMode('month')}
            style={{
              padding: '4px 12px',
              borderRadius: '4px',
              background: viewMode === 'month' ? 'var(--bg-primary)' : 'none',
              border: 'none',
              color: viewMode === 'month' ? 'var(--text-primary)' : 'var(--text-secondary)',
              fontSize: '13px',
              fontWeight: 500,
              cursor: 'pointer',
              boxShadow: viewMode === 'month' ? '0 1px 2px rgba(0,0,0,0.05)' : 'none',
            }}
          >
            Month
          </button>
          <button
            onClick={() => setViewMode('week')}
            style={{
              padding: '4px 12px',
              borderRadius: '4px',
              background: viewMode === 'week' ? 'var(--bg-primary)' : 'none',
              border: 'none',
              color: viewMode === 'week' ? 'var(--text-primary)' : 'var(--text-secondary)',
              fontSize: '13px',
              fontWeight: 500,
              cursor: 'pointer',
              boxShadow: viewMode === 'week' ? '0 1px 2px rgba(0,0,0,0.05)' : 'none',
            }}
          >
            Week
          </button>
        </div>
      </div>

      {/* Weekday headers */}
      <div
        className="calendar-weekdays"
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(7, 1fr)',
          borderBottom: '1px solid var(--border-color)',
        }}
      >
        {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map((day) => (
          <div
            key={day}
            style={{
              padding: '8px',
              textAlign: 'center',
              fontSize: '12px',
              fontWeight: 500,
              color: 'var(--text-tertiary)',
              textTransform: 'uppercase',
            }}
          >
            {day}
          </div>
        ))}
      </div>

      {/* Calendar grid */}
      <div
        className="calendar-grid"
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(7, 1fr)',
          flex: 1,
        }}
      >
        {calendarDays.map((day) => {
          const dayEvents = getEventsForDate(day)
          const isCurrentMonth = isSameMonth(day, currentDate)
          const isSelected = selectedDate ? isSameDay(day, selectedDate) : false
          const isDraggingOver = isDropTarget(day)

          return (
            <div
              key={day.toISOString()}
              className="calendar-day"
              onClick={() => handleDateClick(day)}
              onDragOver={(e) => handleDragOver(day, e)}
              onDragLeave={handleDragLeave}
              onDrop={(e) => handleDrop(day, e)}
              style={{
                minHeight: '100px',
                padding: '4px',
                borderRight: '1px solid var(--border-color)',
                borderBottom: '1px solid var(--border-color)',
                background: isDraggingOver
                  ? 'var(--accent-bg)'
                  : isSelected
                    ? 'rgba(35, 131, 226, 0.04)'
                    : !isCurrentMonth
                      ? 'var(--bg-secondary)'
                      : 'var(--bg-primary)',
                cursor: 'pointer',
                transition: 'background 0.15s',
                position: 'relative',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  marginBottom: '4px',
                }}
              >
                <span
                  style={{
                    width: '24px',
                    height: '24px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    borderRadius: '50%',
                    fontSize: '13px',
                    fontWeight: isToday(day) ? 600 : 400,
                    color: isToday(day)
                      ? 'white'
                      : !isCurrentMonth
                        ? 'var(--text-tertiary)'
                        : 'var(--text-primary)',
                    background: isToday(day) ? 'var(--accent-color)' : 'none',
                  }}
                >
                  {format(day, 'd')}
                </span>
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    handleDateClick(day)
                  }}
                  title="Add event"
                  style={{
                    width: '20px',
                    height: '20px',
                    borderRadius: '4px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: 'none',
                    border: 'none',
                    color: 'var(--text-tertiary)',
                    cursor: 'pointer',
                    opacity: 0,
                    transition: 'opacity 0.15s, background 0.15s',
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.opacity = '1'
                    e.currentTarget.style.background = 'var(--bg-hover)'
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.opacity = '0'
                    e.currentTarget.style.background = 'none'
                  }}
                >
                  <Plus size={12} />
                </button>
              </div>

              <div className="day-events" style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                {dayEvents.slice(0, 3).map((event) => (
                  <div
                    key={event.id}
                    draggable
                    onDragStart={(e) => handleDragStart(event, e)}
                    onDragEnd={handleDragEnd}
                    onClick={(e) => handleEventClick(event, e)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '4px',
                      padding: '2px 6px',
                      borderRadius: '4px',
                      background: 'var(--accent-color)',
                      color: 'white',
                      fontSize: '12px',
                      fontWeight: 500,
                      cursor: 'grab',
                      overflow: 'hidden',
                      opacity: dragState?.eventId === event.id ? 0.5 : 1,
                      transition: 'opacity 0.15s, transform 0.1s',
                    }}
                    onMouseEnter={(e) => {
                      if (!dragState) e.currentTarget.style.transform = 'scale(1.02)'
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.transform = 'scale(1)'
                    }}
                  >
                    <GripHorizontal size={10} style={{ opacity: 0.7, flexShrink: 0 }} />
                    <span
                      style={{
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {titleProperty
                        ? (event.properties[titleProperty.id] as string) || 'Untitled'
                        : 'Untitled'}
                    </span>
                  </div>
                ))}
                {dayEvents.length > 3 && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                    }}
                    style={{
                      padding: '2px 6px',
                      borderRadius: '4px',
                      background: 'none',
                      border: 'none',
                      color: 'var(--text-tertiary)',
                      fontSize: '11px',
                      cursor: 'pointer',
                      textAlign: 'left',
                    }}
                  >
                    +{dayEvents.length - 3} more
                  </button>
                )}
              </div>

              {/* Drop indicator */}
              {isDraggingOver && (
                <motion.div
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  style={{
                    position: 'absolute',
                    inset: '4px',
                    border: '2px dashed var(--accent-color)',
                    borderRadius: '4px',
                    pointerEvents: 'none',
                  }}
                />
              )}
            </div>
          )
        })}
      </div>

      {/* Event form modal */}
      <AnimatePresence>
        {showEventForm && eventFormData && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="modal-overlay"
            onClick={() => setShowEventForm(false)}
            style={{
              position: 'fixed',
              inset: 0,
              background: 'rgba(0, 0, 0, 0.4)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              zIndex: 100,
            }}
          >
            <motion.div
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              onClick={(e) => e.stopPropagation()}
              style={{
                background: 'var(--bg-primary)',
                borderRadius: '8px',
                width: '400px',
                maxWidth: '90vw',
                boxShadow: '0 4px 24px rgba(0, 0, 0, 0.2)',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '16px 20px',
                  borderBottom: '1px solid var(--border-color)',
                }}
              >
                <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>
                  {selectedEvent ? 'Edit event' : 'New event'}
                </h3>
                <button
                  onClick={() => setShowEventForm(false)}
                  style={{
                    width: '28px',
                    height: '28px',
                    borderRadius: '4px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: 'none',
                    border: 'none',
                    color: 'var(--text-tertiary)',
                    cursor: 'pointer',
                  }}
                >
                  <X size={18} />
                </button>
              </div>

              <div style={{ padding: '20px' }}>
                <div style={{ marginBottom: '16px' }}>
                  <input
                    ref={titleInputRef}
                    type="text"
                    placeholder="Event title"
                    value={eventFormData.title}
                    onChange={(e) =>
                      setEventFormData({ ...eventFormData, title: e.target.value })
                    }
                    onKeyDown={handleKeyDown}
                    style={{
                      width: '100%',
                      padding: '10px 12px',
                      border: '1px solid var(--border-color)',
                      borderRadius: '6px',
                      fontSize: '14px',
                      background: 'var(--bg-primary)',
                      color: 'var(--text-primary)',
                    }}
                  />
                </div>

                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <Calendar size={16} style={{ color: 'var(--text-tertiary)' }} />
                  <input
                    type="date"
                    value={format(eventFormData.date, 'yyyy-MM-dd')}
                    onChange={(e) => {
                      const newDate = new Date(e.target.value)
                      if (!isNaN(newDate.getTime())) {
                        setEventFormData({ ...eventFormData, date: newDate })
                      }
                    }}
                    style={{
                      flex: 1,
                      padding: '10px 12px',
                      border: '1px solid var(--border-color)',
                      borderRadius: '6px',
                      fontSize: '14px',
                      background: 'var(--bg-primary)',
                      color: 'var(--text-primary)',
                    }}
                  />
                </div>
              </div>

              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: selectedEvent ? 'space-between' : 'flex-end',
                  padding: '16px 20px',
                  borderTop: '1px solid var(--border-color)',
                  gap: '8px',
                }}
              >
                {selectedEvent && (
                  <button
                    onClick={handleDeleteEvent}
                    style={{
                      padding: '8px 16px',
                      borderRadius: '6px',
                      background: 'none',
                      border: '1px solid var(--danger-color)',
                      color: 'var(--danger-color)',
                      fontSize: '14px',
                      fontWeight: 500,
                      cursor: 'pointer',
                    }}
                  >
                    Delete
                  </button>
                )}
                <div style={{ display: 'flex', gap: '8px' }}>
                  <button
                    onClick={() => setShowEventForm(false)}
                    style={{
                      padding: '8px 16px',
                      borderRadius: '6px',
                      background: 'none',
                      border: '1px solid var(--border-color)',
                      color: 'var(--text-primary)',
                      fontSize: '14px',
                      fontWeight: 500,
                      cursor: 'pointer',
                    }}
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleSaveEvent}
                    disabled={isCreating}
                    style={{
                      padding: '8px 16px',
                      borderRadius: '6px',
                      background: 'var(--accent-color)',
                      border: 'none',
                      color: 'white',
                      fontSize: '14px',
                      fontWeight: 500,
                      cursor: isCreating ? 'not-allowed' : 'pointer',
                      opacity: isCreating ? 0.7 : 1,
                    }}
                  >
                    {isCreating ? 'Saving...' : selectedEvent ? 'Save' : 'Create'}
                  </button>
                </div>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
