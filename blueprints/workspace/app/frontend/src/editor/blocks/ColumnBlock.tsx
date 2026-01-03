import { createReactBlockSpec, useBlockNoteEditor } from '@blocknote/react'
import { useState, useCallback, useRef, useEffect } from 'react'
import { Plus, Trash2, GripVertical, LayoutGrid } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'

// Column layout presets
const COLUMN_LAYOUTS = [
  { id: '2-equal', label: '2 columns', widths: [50, 50] },
  { id: '3-equal', label: '3 columns', widths: [33.33, 33.33, 33.34] },
  { id: '2-left', label: 'Left sidebar', widths: [30, 70] },
  { id: '2-right', label: 'Right sidebar', widths: [70, 30] },
  { id: '3-center', label: 'Center focus', widths: [25, 50, 25] },
  { id: '4-equal', label: '4 columns', widths: [25, 25, 25, 25] },
]

const MIN_COLUMN_WIDTH = 15 // Minimum column width percentage
const MAX_COLUMNS = 5

// Helper to generate unique IDs
const generateId = () => `col-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`

export const ColumnListBlock = createReactBlockSpec(
  {
    type: 'columnList',
    propSchema: {
      columnWidths: {
        default: '50,50',
      },
      columnIds: {
        default: '',
      },
    },
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [isDragging, setIsDragging] = useState(false)
      const [dragIndex, setDragIndex] = useState<number | null>(null)
      const [isHovered, setIsHovered] = useState(false)
      const [hoveredDivider, setHoveredDivider] = useState<number | null>(null)
      const [showLayoutPicker, setShowLayoutPicker] = useState(false)
      const containerRef = useRef<HTMLDivElement>(null)
      const columnRefs = useRef<(HTMLDivElement | null)[]>([])

      // Parse column widths
      const widthsStr = (block.props.columnWidths as string) || '50,50'
      const widths = widthsStr.split(',').map(Number).filter(n => !isNaN(n))

      // Initialize column IDs if empty
      const idsStr = (block.props.columnIds as string) || ''
      const columnIds = idsStr ? idsStr.split(',') : widths.map(() => generateId())

      // Ensure we have IDs for all columns
      useEffect(() => {
        if (!idsStr && widths.length > 0) {
          const ids = widths.map(() => generateId())
          editor.updateBlock(block, {
            props: { ...block.props, columnIds: ids.join(',') },
          })
        }
      }, [])

      // Handle divider drag for resizing columns
      const handleDividerDrag = useCallback(
        (e: React.MouseEvent, index: number) => {
          e.preventDefault()
          e.stopPropagation()

          const container = containerRef.current
          if (!container) return

          const containerRect = container.getBoundingClientRect()
          const startX = e.clientX
          const startWidths = [...widths]

          setIsDragging(true)
          setDragIndex(index)

          const handleMouseMove = (moveEvent: MouseEvent) => {
            const deltaX = moveEvent.clientX - startX
            const deltaPercent = (deltaX / containerRect.width) * 100

            const newWidths = [...startWidths]
            const newLeftWidth = startWidths[index] + deltaPercent
            const newRightWidth = startWidths[index + 1] - deltaPercent

            // Enforce minimum width
            if (newLeftWidth >= MIN_COLUMN_WIDTH && newRightWidth >= MIN_COLUMN_WIDTH) {
              newWidths[index] = Math.round(newLeftWidth * 100) / 100
              newWidths[index + 1] = Math.round(newRightWidth * 100) / 100

              editor.updateBlock(block, {
                props: { ...block.props, columnWidths: newWidths.join(',') },
              })
            }
          }

          const handleMouseUp = () => {
            setIsDragging(false)
            setDragIndex(null)
            document.removeEventListener('mousemove', handleMouseMove)
            document.removeEventListener('mouseup', handleMouseUp)
            document.body.style.cursor = ''
            document.body.style.userSelect = ''
          }

          document.body.style.cursor = 'col-resize'
          document.body.style.userSelect = 'none'
          document.addEventListener('mousemove', handleMouseMove)
          document.addEventListener('mouseup', handleMouseUp)
        },
        [block, editor, widths]
      )

      // Add a new column
      const addColumn = useCallback(() => {
        if (widths.length >= MAX_COLUMNS) return

        const newWidths = [...widths]
        // Calculate space to take from each existing column
        const newColumnWidth = 100 / (newWidths.length + 1)
        const scaleFactor = (100 - newColumnWidth) / 100
        const adjustedWidths = newWidths.map(w => Math.round(w * scaleFactor * 100) / 100)
        adjustedWidths.push(newColumnWidth)

        const newIds = [...columnIds, generateId()]

        editor.updateBlock(block, {
          props: {
            ...block.props,
            columnWidths: adjustedWidths.join(','),
            columnIds: newIds.join(','),
          },
        })
      }, [block, editor, widths, columnIds])

      // Remove a column
      const removeColumn = useCallback((index: number) => {
        if (widths.length <= 2) return // Minimum 2 columns

        const removedWidth = widths[index]
        const newWidths = widths.filter((_, i) => i !== index)
        // Distribute removed column's width proportionally
        const totalRemaining = newWidths.reduce((a, b) => a + b, 0)
        const adjustedWidths = newWidths.map(w =>
          Math.round((w / totalRemaining) * 100 * 100) / 100
        )

        const newIds = columnIds.filter((_, i) => i !== index)

        editor.updateBlock(block, {
          props: {
            ...block.props,
            columnWidths: adjustedWidths.join(','),
            columnIds: newIds.join(','),
          },
        })
      }, [block, editor, widths, columnIds])

      // Reset columns to equal widths
      const resetToEqual = useCallback(() => {
        const equalWidth = Math.round((100 / widths.length) * 100) / 100
        const newWidths = widths.map(() => equalWidth)
        editor.updateBlock(block, {
          props: { ...block.props, columnWidths: newWidths.join(',') },
        })
      }, [block, editor, widths])

      // Apply a preset layout
      const applyLayout = useCallback((layout: typeof COLUMN_LAYOUTS[0]) => {
        const newIds = layout.widths.map((_, i) => columnIds[i] || generateId())
        editor.updateBlock(block, {
          props: {
            ...block.props,
            columnWidths: layout.widths.join(','),
            columnIds: newIds.join(','),
          },
        })
        setShowLayoutPicker(false)
      }, [block, editor, columnIds])

      return (
        <div
          ref={containerRef}
          className={`column-list-block ${isDragging ? 'dragging' : ''}`}
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => setIsHovered(false)}
          style={{
            position: 'relative',
            margin: '8px 0',
            borderRadius: '4px',
            minHeight: '80px',
          }}
        >
          {/* Floating toolbar */}
          <AnimatePresence>
            {isHovered && (
              <motion.div
                initial={{ opacity: 0, y: -4 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -4 }}
                transition={{ duration: 0.1 }}
                className="column-toolbar"
                style={{
                  position: 'absolute',
                  top: '-36px',
                  left: '50%',
                  transform: 'translateX(-50%)',
                  display: 'flex',
                  gap: '4px',
                  padding: '6px 8px',
                  background: 'var(--bg-primary, white)',
                  borderRadius: '8px',
                  boxShadow: '0 2px 8px rgba(0,0,0,0.15), 0 0 0 1px rgba(0,0,0,0.05)',
                  zIndex: 20,
                }}
              >
                <button
                  onClick={() => setShowLayoutPicker(!showLayoutPicker)}
                  title="Change layout"
                  style={{
                    padding: '6px 10px',
                    background: 'none',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '6px',
                    fontSize: '12px',
                    fontWeight: 500,
                    color: 'var(--text-secondary)',
                    transition: 'all 0.15s ease',
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.background = 'var(--bg-hover)'
                    e.currentTarget.style.color = 'var(--text-primary)'
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = 'none'
                    e.currentTarget.style.color = 'var(--text-secondary)'
                  }}
                >
                  <LayoutGrid size={14} />
                  Layout
                </button>
                <div style={{ width: 1, background: 'var(--border-color)', margin: '0 4px' }} />
                <button
                  onClick={addColumn}
                  disabled={widths.length >= MAX_COLUMNS}
                  title="Add column"
                  style={{
                    padding: '6px 10px',
                    background: 'none',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: widths.length >= MAX_COLUMNS ? 'not-allowed' : 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '6px',
                    fontSize: '12px',
                    fontWeight: 500,
                    color: widths.length >= MAX_COLUMNS ? 'var(--text-tertiary)' : 'var(--text-secondary)',
                    opacity: widths.length >= MAX_COLUMNS ? 0.5 : 1,
                    transition: 'all 0.15s ease',
                  }}
                  onMouseEnter={(e) => {
                    if (widths.length < MAX_COLUMNS) {
                      e.currentTarget.style.background = 'var(--bg-hover)'
                      e.currentTarget.style.color = 'var(--text-primary)'
                    }
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = 'none'
                    e.currentTarget.style.color = widths.length >= MAX_COLUMNS ? 'var(--text-tertiary)' : 'var(--text-secondary)'
                  }}
                >
                  <Plus size={14} />
                  Add
                </button>
                <button
                  onClick={resetToEqual}
                  title="Reset to equal widths"
                  style={{
                    padding: '6px 10px',
                    background: 'none',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    fontSize: '12px',
                    fontWeight: 500,
                    color: 'var(--text-secondary)',
                    transition: 'all 0.15s ease',
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.background = 'var(--bg-hover)'
                    e.currentTarget.style.color = 'var(--text-primary)'
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = 'none'
                    e.currentTarget.style.color = 'var(--text-secondary)'
                  }}
                >
                  Reset
                </button>
              </motion.div>
            )}
          </AnimatePresence>

          {/* Layout picker dropdown */}
          <AnimatePresence>
            {showLayoutPicker && (
              <motion.div
                initial={{ opacity: 0, y: -8, scale: 0.95 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: -8, scale: 0.95 }}
                transition={{ duration: 0.15 }}
                style={{
                  position: 'absolute',
                  top: '-160px',
                  left: '50%',
                  transform: 'translateX(-50%)',
                  background: 'var(--bg-primary, white)',
                  borderRadius: '8px',
                  boxShadow: '0 4px 16px rgba(0,0,0,0.15), 0 0 0 1px rgba(0,0,0,0.05)',
                  padding: '8px',
                  zIndex: 30,
                  display: 'grid',
                  gridTemplateColumns: 'repeat(3, 1fr)',
                  gap: '6px',
                  minWidth: '240px',
                }}
                onMouseLeave={() => setShowLayoutPicker(false)}
              >
                {COLUMN_LAYOUTS.map((layout) => (
                  <button
                    key={layout.id}
                    onClick={() => applyLayout(layout)}
                    style={{
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      gap: '6px',
                      padding: '8px',
                      background: 'none',
                      border: '1px solid var(--border-color)',
                      borderRadius: '6px',
                      cursor: 'pointer',
                      transition: 'all 0.15s ease',
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.background = 'var(--bg-hover)'
                      e.currentTarget.style.borderColor = 'var(--accent-color)'
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.background = 'none'
                      e.currentTarget.style.borderColor = 'var(--border-color)'
                    }}
                  >
                    <div style={{ display: 'flex', gap: '2px', width: '100%', height: '20px' }}>
                      {layout.widths.map((width, i) => (
                        <div
                          key={i}
                          style={{
                            width: `${width}%`,
                            height: '100%',
                            background: 'var(--accent-color)',
                            borderRadius: '2px',
                            opacity: 0.5,
                          }}
                        />
                      ))}
                    </div>
                    <span style={{ fontSize: '10px', color: 'var(--text-secondary)' }}>
                      {layout.label}
                    </span>
                  </button>
                ))}
              </motion.div>
            )}
          </AnimatePresence>

          {/* Columns container */}
          <div
            className="column-list-content"
            style={{
              display: 'flex',
              gap: 0,
              width: '100%',
              position: 'relative',
              border: isDragging ? '1px dashed var(--accent-color)' : '1px solid transparent',
              borderRadius: '4px',
              transition: 'border-color 0.15s ease',
            }}
          >
            {widths.map((width, index) => (
              <div
                key={columnIds[index] || index}
                ref={(el) => { columnRefs.current[index] = el }}
                className="column-wrapper"
                style={{
                  width: `${width}%`,
                  position: 'relative',
                  minHeight: '60px',
                  display: 'flex',
                  flexDirection: 'column',
                }}
              >
                {/* Column content area */}
                <div
                  className="column-block-content"
                  data-column-id={columnIds[index]}
                  style={{
                    flex: 1,
                    padding: '12px 16px',
                    background: isHovered
                      ? 'rgba(55, 53, 47, 0.02)'
                      : 'transparent',
                    borderRadius: '4px',
                    transition: 'background 0.15s ease',
                    minHeight: '60px',
                    position: 'relative',
                  }}
                >
                  {/* Placeholder when empty */}
                  <div
                    className="column-placeholder"
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      height: '100%',
                      minHeight: '40px',
                      color: 'var(--text-tertiary)',
                      fontSize: '13px',
                      fontStyle: 'italic',
                      userSelect: 'none',
                    }}
                  >
                    Column {index + 1}
                  </div>
                </div>

                {/* Column remove button */}
                <AnimatePresence>
                  {isHovered && widths.length > 2 && (
                    <motion.button
                      initial={{ opacity: 0, scale: 0.8 }}
                      animate={{ opacity: 1, scale: 1 }}
                      exit={{ opacity: 0, scale: 0.8 }}
                      transition={{ duration: 0.1 }}
                      onClick={(e) => {
                        e.preventDefault()
                        e.stopPropagation()
                        removeColumn(index)
                      }}
                      title="Remove column"
                      style={{
                        position: 'absolute',
                        top: '4px',
                        right: '4px',
                        padding: '4px',
                        background: 'var(--bg-primary, white)',
                        border: '1px solid var(--border-color)',
                        borderRadius: '4px',
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        color: 'var(--text-tertiary)',
                        zIndex: 5,
                        boxShadow: '0 1px 4px rgba(0,0,0,0.1)',
                        transition: 'all 0.15s ease',
                      }}
                      onMouseEnter={(e) => {
                        e.currentTarget.style.color = 'var(--danger-color)'
                        e.currentTarget.style.borderColor = 'var(--danger-color)'
                        e.currentTarget.style.background = 'var(--danger-bg, #fff5f5)'
                      }}
                      onMouseLeave={(e) => {
                        e.currentTarget.style.color = 'var(--text-tertiary)'
                        e.currentTarget.style.borderColor = 'var(--border-color)'
                        e.currentTarget.style.background = 'var(--bg-primary, white)'
                      }}
                    >
                      <Trash2 size={12} />
                    </motion.button>
                  )}
                </AnimatePresence>

                {/* Resize divider between columns */}
                {index < widths.length - 1 && (
                  <div
                    className={`column-divider ${dragIndex === index ? 'active' : ''} ${hoveredDivider === index ? 'hovered' : ''}`}
                    style={{
                      position: 'absolute',
                      right: 0,
                      top: 0,
                      bottom: 0,
                      width: '16px',
                      marginRight: '-8px',
                      cursor: 'col-resize',
                      zIndex: 10,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                    onMouseDown={(e) => handleDividerDrag(e, index)}
                    onMouseEnter={() => setHoveredDivider(index)}
                    onMouseLeave={() => setHoveredDivider(null)}
                  >
                    {/* Visual divider line */}
                    <div
                      style={{
                        width: '3px',
                        height: '100%',
                        background: (hoveredDivider === index || dragIndex === index)
                          ? 'var(--accent-color, #2383e2)'
                          : 'var(--border-color)',
                        borderRadius: '2px',
                        transition: 'all 0.15s ease',
                        opacity: (hoveredDivider === index || dragIndex === index || isHovered) ? 1 : 0,
                      }}
                    />

                    {/* Grip handle indicator */}
                    <AnimatePresence>
                      {(hoveredDivider === index || dragIndex === index) && (
                        <motion.div
                          initial={{ opacity: 0, scale: 0.8 }}
                          animate={{ opacity: 1, scale: 1 }}
                          exit={{ opacity: 0, scale: 0.8 }}
                          transition={{ duration: 0.1 }}
                          style={{
                            position: 'absolute',
                            top: '50%',
                            transform: 'translateY(-50%)',
                            background: 'var(--bg-primary, white)',
                            border: '1px solid var(--accent-color, #2383e2)',
                            borderRadius: '4px',
                            padding: '6px 3px',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            boxShadow: '0 2px 6px rgba(0,0,0,0.1)',
                          }}
                        >
                          <GripVertical size={12} style={{ color: 'var(--accent-color)' }} />
                        </motion.div>
                      )}
                    </AnimatePresence>
                  </div>
                )}
              </div>
            ))}
          </div>

          {/* Width indicator during drag */}
          <AnimatePresence>
            {isDragging && dragIndex !== null && (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                style={{
                  position: 'absolute',
                  bottom: '-24px',
                  left: '50%',
                  transform: 'translateX(-50%)',
                  background: 'var(--text-primary)',
                  color: 'white',
                  padding: '4px 8px',
                  borderRadius: '4px',
                  fontSize: '11px',
                  fontWeight: 500,
                  whiteSpace: 'nowrap',
                }}
              >
                {Math.round(widths[dragIndex])}% / {Math.round(widths[dragIndex + 1])}%
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      )
    },
  }
)

// Column selector component for slash menu
export function ColumnLayoutPicker({
  onSelect,
}: {
  onSelect: (layout: { id: string; widths: number[] }) => void
}) {
  return (
    <div
      className="column-layout-picker"
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(3, 1fr)',
        gap: '8px',
        padding: '12px',
      }}
    >
      {COLUMN_LAYOUTS.map((layout) => (
        <button
          key={layout.id}
          className="layout-option"
          onClick={() => onSelect(layout)}
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '8px',
            padding: '12px',
            background: 'none',
            border: '1px solid var(--border-color)',
            borderRadius: '6px',
            cursor: 'pointer',
            transition: 'all 0.15s ease',
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = 'var(--bg-hover)'
            e.currentTarget.style.borderColor = 'var(--accent-color)'
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = 'none'
            e.currentTarget.style.borderColor = 'var(--border-color)'
          }}
        >
          <div
            className="layout-preview"
            style={{
              display: 'flex',
              gap: '2px',
              width: '100%',
              height: '28px',
            }}
          >
            {layout.widths.map((width, i) => (
              <div
                key={i}
                className="preview-column"
                style={{
                  width: `${width}%`,
                  height: '100%',
                  background: 'var(--accent-color)',
                  borderRadius: '3px',
                  opacity: 0.4,
                }}
              />
            ))}
          </div>
          <span
            className="layout-label"
            style={{
              fontSize: '11px',
              color: 'var(--text-secondary)',
              fontWeight: 500,
            }}
          >
            {layout.label}
          </span>
        </button>
      ))}
    </div>
  )
}

export { ColumnListBlock as ColumnBlock }
