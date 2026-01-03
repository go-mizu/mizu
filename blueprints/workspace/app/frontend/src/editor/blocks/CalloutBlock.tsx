import { createReactBlockSpec } from '@blocknote/react'
import { useState, useRef, useEffect } from 'react'

const CALLOUT_COLORS = {
  default: { bg: '#f7f6f3', border: '#e3e2de' },
  gray: { bg: '#f1f1ef', border: '#e3e2de' },
  brown: { bg: '#f4eeee', border: '#e8dcdc' },
  orange: { bg: '#fbecdd', border: '#f5ddc4' },
  yellow: { bg: '#fbf3db', border: '#f5e9c1' },
  green: { bg: '#edf3ec', border: '#d4e4d1' },
  blue: { bg: '#e7f3f8', border: '#cce4ed' },
  purple: { bg: '#f6f3f9', border: '#e4dbed' },
  pink: { bg: '#faf1f5', border: '#f0dce4' },
  red: { bg: '#fdebec', border: '#f5d5d7' },
}

const COMMON_ICONS = ['💡', '📝', '⚠️', '❗', '✅', '❌', '🔥', '💭', '📌', '🎯', '💬', '📢']

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
      const [showColorPicker, setShowColorPicker] = useState(false)
      const iconPickerRef = useRef<HTMLDivElement>(null)
      const colorPickerRef = useRef<HTMLDivElement>(null)

      const colors = CALLOUT_COLORS[block.props.backgroundColor as keyof typeof CALLOUT_COLORS] || CALLOUT_COLORS.default

      // Close pickers on click outside
      useEffect(() => {
        const handleClickOutside = (e: MouseEvent) => {
          if (iconPickerRef.current && !iconPickerRef.current.contains(e.target as Node)) {
            setShowIconPicker(false)
          }
          if (colorPickerRef.current && !colorPickerRef.current.contains(e.target as Node)) {
            setShowColorPicker(false)
          }
        }
        document.addEventListener('mousedown', handleClickOutside)
        return () => document.removeEventListener('mousedown', handleClickOutside)
      }, [])

      const updateIcon = (icon: string) => {
        editor.updateBlock(block, { props: { ...block.props, icon } })
        setShowIconPicker(false)
      }

      const updateColor = (color: string) => {
        editor.updateBlock(block, { props: { ...block.props, backgroundColor: color } })
        setShowColorPicker(false)
      }

      return (
        <div
          className="callout-block"
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            padding: '16px',
            borderRadius: '4px',
            backgroundColor: colors.bg,
            border: `1px solid ${colors.border}`,
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
                  background: 'white',
                  border: '1px solid #e3e2de',
                  borderRadius: '8px',
                  boxShadow: '0 4px 16px rgba(0,0,0,0.1)',
                  padding: '8px',
                  display: 'grid',
                  gridTemplateColumns: 'repeat(6, 1fr)',
                  gap: '4px',
                  width: '200px',
                }}
              >
                {COMMON_ICONS.map((icon) => (
                  <button
                    key={icon}
                    onClick={() => updateIcon(icon)}
                    style={{
                      fontSize: '1.2em',
                      background: 'none',
                      border: 'none',
                      cursor: 'pointer',
                      padding: '6px',
                      borderRadius: '4px',
                    }}
                    className="icon-option"
                  >
                    {icon}
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="callout-content" style={{ flex: 1 }}>
            <div ref={contentRef} />
          </div>

          <div className="callout-color-wrapper" style={{ position: 'relative' }}>
            <button
              onClick={() => setShowColorPicker(!showColorPicker)}
              style={{
                width: '20px',
                height: '20px',
                borderRadius: '4px',
                backgroundColor: colors.bg,
                border: `2px solid ${colors.border}`,
                cursor: 'pointer',
                marginLeft: '8px',
              }}
              title="Change color"
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
                  background: 'white',
                  border: '1px solid #e3e2de',
                  borderRadius: '8px',
                  boxShadow: '0 4px 16px rgba(0,0,0,0.1)',
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
                      backgroundColor: c.bg,
                      border: block.props.backgroundColor === name
                        ? '2px solid #2383e2'
                        : `1px solid ${c.border}`,
                      cursor: 'pointer',
                    }}
                    title={name}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      )
    },
  }
)
