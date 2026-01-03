import { createReactBlockSpec } from '@blocknote/react'
import { defaultProps } from '@blocknote/core'
import { useState, useCallback } from 'react'
import { ChevronRight } from 'lucide-react'

// Toggle block with collapsible content
// Note: BlockNote handles children nesting naturally - blocks can be nested
// under this toggle by indenting them (Tab key) when the cursor is on a child block
export const ToggleBlock = createReactBlockSpec(
  {
    type: 'toggle',
    propSchema: {
      ...defaultProps,
    },
    content: 'inline',
  },
  {
    render: (props) => {
      const [isExpanded, setIsExpanded] = useState(true)

      const handleToggle = useCallback((e: React.MouseEvent) => {
        e.preventDefault()
        e.stopPropagation()
        setIsExpanded(!isExpanded)
      }, [isExpanded])

      return (
        <div
          className="toggle-block"
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: '4px',
            padding: '3px 0',
          }}
        >
          <button
            onClick={handleToggle}
            className="toggle-icon-btn"
            title={isExpanded ? 'Collapse' : 'Expand'}
            aria-expanded={isExpanded}
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
              color: 'var(--bn-colors-sideMenu-icon, rgba(55, 53, 47, 0.65))',
              flexShrink: 0,
              marginTop: '2px',
              transition: 'transform 0.15s ease, background 0.1s ease',
              transform: isExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = 'var(--bn-colors-hovered-background, rgba(55, 53, 47, 0.08))'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = 'none'
            }}
          >
            <ChevronRight size={14} strokeWidth={2.5} />
          </button>

          <div
            ref={props.contentRef}
            className="toggle-content"
            style={{
              flex: 1,
              minWidth: 0,
              lineHeight: 1.5,
            }}
          />
        </div>
      )
    },
  }
)
