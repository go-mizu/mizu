import { createReactBlockSpec } from '@blocknote/react'
import { useState, useCallback, useRef, useEffect } from 'react'
import { ChevronRight } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'

export const ToggleBlock = createReactBlockSpec(
  {
    type: 'toggle',
    propSchema: {
      collapsed: {
        default: true,
      },
    },
    content: 'inline',
  },
  {
    render: ({ block, editor, contentRef }) => {
      const [isCollapsed, setIsCollapsed] = useState(block.props.collapsed !== false)
      const childrenRef = useRef<HTMLDivElement>(null)

      // Sync local state with block props
      useEffect(() => {
        if (block.props.collapsed !== isCollapsed) {
          setIsCollapsed(block.props.collapsed !== false)
        }
      }, [block.props.collapsed])

      const toggleCollapse = useCallback(() => {
        const newState = !isCollapsed
        setIsCollapsed(newState)
        editor.updateBlock(block, { props: { ...block.props, collapsed: newState } })
      }, [block, editor, isCollapsed])

      const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && e.shiftKey) {
          e.preventDefault()
          e.stopPropagation()
          toggleCollapse()
        }
        // Also toggle on arrow keys when focus is on header
        if (e.key === 'ArrowRight' && isCollapsed) {
          e.preventDefault()
          toggleCollapse()
        }
        if (e.key === 'ArrowLeft' && !isCollapsed) {
          e.preventDefault()
          toggleCollapse()
        }
      }, [toggleCollapse, isCollapsed])

      const handleIconClick = useCallback((e: React.MouseEvent) => {
        e.preventDefault()
        e.stopPropagation()
        toggleCollapse()
      }, [toggleCollapse])

      return (
        <div
          className="toggle-block"
          data-collapsed={isCollapsed}
          style={{
            padding: '3px 0',
            margin: '1px 0',
          }}
        >
          <div
            className="toggle-header"
            onKeyDown={handleKeyDown}
            style={{
              display: 'flex',
              alignItems: 'flex-start',
              gap: '4px',
            }}
          >
            <button
              className="toggle-icon-btn"
              onClick={handleIconClick}
              title={isCollapsed ? 'Expand' : 'Collapse'}
              aria-expanded={!isCollapsed}
              aria-label={isCollapsed ? 'Expand toggle' : 'Collapse toggle'}
              tabIndex={0}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                width: '24px',
                height: '24px',
                padding: 0,
                background: 'none',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
                color: 'var(--text-secondary, rgba(55, 53, 47, 0.65))',
                flexShrink: 0,
                marginTop: '2px',
                transition: 'background 0.1s ease, color 0.1s ease',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background = 'var(--bg-hover, rgba(55, 53, 47, 0.08))'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = 'none'
              }}
            >
              <motion.div
                animate={{ rotate: isCollapsed ? 0 : 90 }}
                transition={{ duration: 0.15, ease: 'easeOut' }}
                style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}
              >
                <ChevronRight size={14} strokeWidth={2.5} />
              </motion.div>
            </button>

            <div
              className="toggle-content"
              ref={contentRef}
              style={{
                flex: 1,
                minWidth: 0,
                lineHeight: 1.5,
              }}
            />
          </div>

          <AnimatePresence initial={false}>
            {!isCollapsed && (
              <motion.div
                ref={childrenRef}
                className="toggle-children"
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: 'auto', opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                transition={{
                  duration: 0.2,
                  ease: [0.4, 0, 0.2, 1],
                  opacity: { duration: 0.15 }
                }}
                style={{
                  overflow: 'hidden',
                  marginLeft: '28px',
                  paddingTop: '4px',
                  borderLeft: '1px solid var(--border-color, rgba(55, 53, 47, 0.09))',
                  paddingLeft: '12px',
                }}
              >
                <div
                  className="toggle-children-content"
                  contentEditable={true}
                  suppressContentEditableWarning
                  data-placeholder="Empty toggle. Click or drag blocks inside."
                  style={{
                    minHeight: '32px',
                    outline: 'none',
                    color: 'var(--text-primary, #37352f)',
                  }}
                />
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      )
    },
  }
)
