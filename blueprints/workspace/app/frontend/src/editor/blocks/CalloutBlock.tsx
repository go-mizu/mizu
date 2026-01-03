import { createReactBlockSpec } from '@blocknote/react'
import { useState, useRef, useEffect } from 'react'
import data from '@emoji-mart/data'
import Picker from '@emoji-mart/react'

const CALLOUT_COLORS: Record<string, { bg: string; border: string; darkBg: string; darkBorder: string }> = {
  default: { bg: '#f7f6f3', border: '#e3e2de', darkBg: '#2f2f2f', darkBorder: '#404040' },
  gray: { bg: '#f1f1ef', border: '#e3e2de', darkBg: '#373737', darkBorder: '#484848' },
  brown: { bg: '#f4eeee', border: '#e8dcdc', darkBg: '#3d3632', darkBorder: '#4d433c' },
  orange: { bg: '#fbecdd', border: '#f5ddc4', darkBg: '#3d3225', darkBorder: '#5c4a35' },
  yellow: { bg: '#fbf3db', border: '#f5e9c1', darkBg: '#3d3825', darkBorder: '#5c5235' },
  green: { bg: '#edf3ec', border: '#d4e4d1', darkBg: '#2b3329', darkBorder: '#3d4d39' },
  blue: { bg: '#e7f3f8', border: '#cce4ed', darkBg: '#252f38', darkBorder: '#35485c' },
  purple: { bg: '#f6f3f9', border: '#e4dbed', darkBg: '#332b3d', darkBorder: '#4d3d5c' },
  pink: { bg: '#faf1f5', border: '#f0dce4', darkBg: '#3d2b35', darkBorder: '#5c3d4d' },
  red: { bg: '#fdebec', border: '#f5d5d7', darkBg: '#3d2b2b', darkBorder: '#5c3d3d' },
}

const QUICK_ICONS = ['💡', '📝', '⚠️', '❗', '✅', '❌', '🔥', '💭', '📌', '🎯', '💬', '📢', '🚀', '💪', '🎉', '❓']

export const CalloutBlock = createReactBlockSpec(
  {
    type: 'callout',
    propSchema: {
      icon: {
        default: '💡',
      },
      backgroundColor: {
        default: 'default',
        values: Object.keys(CALLOUT_COLORS),
      },
    },
    content: 'inline',
  },
  {
    render: ({ block, editor, contentRef }) => {
      const [showIconPicker, setShowIconPicker] = useState(false)
      const [showFullEmojiPicker, setShowFullEmojiPicker] = useState(false)
      const [showColorPicker, setShowColorPicker] = useState(false)
      const iconPickerRef = useRef<HTMLDivElement>(null)
      const colorPickerRef = useRef<HTMLDivElement>(null)
      const emojiPickerRef = useRef<HTMLDivElement>(null)

      const colorKey = (block.props.backgroundColor as string) || 'default'
      const colors = CALLOUT_COLORS[colorKey] || CALLOUT_COLORS.default

      // Detect dark mode
      const isDark = typeof document !== 'undefined' &&
        document.documentElement.getAttribute('data-theme') === 'dark'

      const bgColor = isDark ? colors.darkBg : colors.bg
      const borderColor = isDark ? colors.darkBorder : colors.border

      // Close pickers on click outside
      useEffect(() => {
        const handleClickOutside = (e: MouseEvent) => {
          if (iconPickerRef.current && !iconPickerRef.current.contains(e.target as Node)) {
            setShowIconPicker(false)
          }
          if (colorPickerRef.current && !colorPickerRef.current.contains(e.target as Node)) {
            setShowColorPicker(false)
          }
          if (emojiPickerRef.current && !emojiPickerRef.current.contains(e.target as Node)) {
            setShowFullEmojiPicker(false)
          }
        }
        document.addEventListener('mousedown', handleClickOutside)
        return () => document.removeEventListener('mousedown', handleClickOutside)
      }, [])

      const updateIcon = (icon: string) => {
        editor.updateBlock(block, { props: { ...block.props, icon } })
        setShowIconPicker(false)
        setShowFullEmojiPicker(false)
      }

      const updateColor = (color: string) => {
        editor.updateBlock(block, { props: { ...block.props, backgroundColor: color } })
        setShowColorPicker(false)
      }

      const handleEmojiSelect = (emoji: { native: string }) => {
        updateIcon(emoji.native)
      }

      return (
        <div
          className="callout-block"
          data-callout-icon={block.props.icon}
          data-callout-color={block.props.backgroundColor}
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            padding: '16px',
            borderRadius: '4px',
            backgroundColor: bgColor,
            border: `1px solid ${borderColor}`,
            margin: '4px 0',
          }}
        >
          <div className="callout-icon-wrapper" style={{ position: 'relative' }}>
            <button
              className="callout-icon"
              onClick={() => setShowIconPicker(!showIconPicker)}
              style={{
                fontSize: '1.5em',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                padding: '0 8px 0 0',
                lineHeight: 1,
              }}
              title="Change icon"
            >
              {block.props.icon}
            </button>

            {showIconPicker && (
              <div
                ref={iconPickerRef}
                className="callout-icon-picker"
                style={{
                  position: 'absolute',
                  top: '100%',
                  left: 0,
                  zIndex: 100,
                  background: 'var(--bg-primary, white)',
                  border: '1px solid var(--border-color, #e3e2de)',
                  borderRadius: '8px',
                  boxShadow: '0 4px 16px rgba(0,0,0,0.15)',
                  padding: '8px',
                  width: '220px',
                }}
              >
                {/* Quick icons grid */}
                <div style={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(8, 1fr)',
                  gap: '2px',
                  marginBottom: '8px',
                }}>
                  {QUICK_ICONS.map((icon) => (
                    <button
                      key={icon}
                      onClick={() => updateIcon(icon)}
                      style={{
                        fontSize: '1.1em',
                        background: 'none',
                        border: 'none',
                        cursor: 'pointer',
                        padding: '6px',
                        borderRadius: '4px',
                        transition: 'background 0.1s',
                      }}
                      onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-hover, rgba(0,0,0,0.05))'}
                      onMouseLeave={(e) => e.currentTarget.style.background = 'none'}
                    >
                      {icon}
                    </button>
                  ))}
                </div>

                {/* Show full picker button */}
                <button
                  onClick={() => {
                    setShowIconPicker(false)
                    setShowFullEmojiPicker(true)
                  }}
                  style={{
                    width: '100%',
                    padding: '6px 12px',
                    background: 'var(--bg-secondary, #f7f6f3)',
                    border: '1px solid var(--border-color, #e3e2de)',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    fontSize: '12px',
                    color: 'var(--text-secondary, #787774)',
                  }}
                >
                  Browse all emoji...
                </button>
              </div>
            )}

            {/* Full emoji picker */}
            {showFullEmojiPicker && (
              <div
                ref={emojiPickerRef}
                style={{
                  position: 'absolute',
                  top: '100%',
                  left: 0,
                  zIndex: 200,
                }}
              >
                <Picker
                  data={data}
                  onEmojiSelect={handleEmojiSelect}
                  theme={isDark ? 'dark' : 'light'}
                  previewPosition="none"
                  skinTonePosition="search"
                />
              </div>
            )}
          </div>

          <div className="callout-content" style={{ flex: 1, minWidth: 0 }}>
            <div ref={contentRef} style={{ outline: 'none' }} />
          </div>

          <div className="callout-color-wrapper" style={{ position: 'relative' }}>
            <button
              onClick={() => setShowColorPicker(!showColorPicker)}
              style={{
                width: '20px',
                height: '20px',
                borderRadius: '4px',
                backgroundColor: bgColor,
                border: `2px solid ${borderColor}`,
                cursor: 'pointer',
                marginLeft: '8px',
                transition: 'transform 0.1s',
              }}
              title="Change color"
              onMouseEnter={(e) => e.currentTarget.style.transform = 'scale(1.1)'}
              onMouseLeave={(e) => e.currentTarget.style.transform = 'scale(1)'}
            />

            {showColorPicker && (
              <div
                ref={colorPickerRef}
                className="callout-color-picker"
                style={{
                  position: 'absolute',
                  top: '100%',
                  right: 0,
                  zIndex: 100,
                  background: 'var(--bg-primary, white)',
                  border: '1px solid var(--border-color, #e3e2de)',
                  borderRadius: '8px',
                  boxShadow: '0 4px 16px rgba(0,0,0,0.15)',
                  padding: '8px',
                  display: 'grid',
                  gridTemplateColumns: 'repeat(5, 1fr)',
                  gap: '4px',
                }}
              >
                {Object.entries(CALLOUT_COLORS).map(([name, c]) => (
                  <button
                    key={name}
                    onClick={() => updateColor(name)}
                    style={{
                      width: '24px',
                      height: '24px',
                      borderRadius: '4px',
                      backgroundColor: isDark ? c.darkBg : c.bg,
                      border: block.props.backgroundColor === name
                        ? '2px solid var(--accent-color, #2383e2)'
                        : `1px solid ${isDark ? c.darkBorder : c.border}`,
                      cursor: 'pointer',
                      transition: 'transform 0.1s',
                    }}
                    title={name.charAt(0).toUpperCase() + name.slice(1)}
                    onMouseEnter={(e) => e.currentTarget.style.transform = 'scale(1.15)'}
                    onMouseLeave={(e) => e.currentTarget.style.transform = 'scale(1)'}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      )
    },
    // Parse HTML to recreate block when pasting or drag-dropping
    parse: (element: HTMLElement) => {
      if (element.classList.contains('callout-block') || element.hasAttribute('data-callout-icon')) {
        return {
          icon: element.getAttribute('data-callout-icon') || '💡',
          backgroundColor: element.getAttribute('data-callout-color') || 'default',
        }
      }
      return undefined
    },
    // Convert to external HTML for clipboard/export
    toExternalHTML: ({ block, contentRef }) => {
      const colorKey = (block.props.backgroundColor as string) || 'default'
      const colors = CALLOUT_COLORS[colorKey] || CALLOUT_COLORS.default
      return (
        <div
          className="callout-block"
          data-callout-icon={block.props.icon}
          data-callout-color={block.props.backgroundColor}
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            padding: '16px',
            borderRadius: '4px',
            backgroundColor: colors.bg,
            border: `1px solid ${colors.border}`,
          }}
        >
          <span style={{ fontSize: '1.5em', marginRight: '8px' }}>{block.props.icon}</span>
          <div ref={contentRef} style={{ flex: 1 }} />
        </div>
      )
    },
  }
)
