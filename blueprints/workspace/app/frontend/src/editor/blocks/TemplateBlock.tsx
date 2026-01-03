import { createReactBlockSpec } from '@blocknote/react'
import { useState, useCallback } from 'react'
import { Copy, Plus, Settings, Trash2, ChevronRight, Layout } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'

interface TemplateContent {
  type: string
  content?: unknown[]
  props?: Record<string, unknown>
}

export const TemplateBlock = createReactBlockSpec(
  {
    type: 'templateButton',
    propSchema: {
      buttonText: {
        default: 'New item from template',
      },
      templateContent: {
        default: '[]', // JSON string of template blocks
      },
      templateName: {
        default: '',
      },
      buttonStyle: {
        default: 'default', // 'default' | 'primary' | 'outline'
      },
    },
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [isExpanded, setIsExpanded] = useState(false)
      const [isEditing, setIsEditing] = useState(false)
      const [isHovered, setIsHovered] = useState(false)
      const [duplicateCount, setDuplicateCount] = useState(0)
      const [showSettings, setShowSettings] = useState(false)

      const buttonText = (block.props.buttonText as string) || 'New item from template'
      const templateContentStr = (block.props.templateContent as string) || '[]'
      const templateName = (block.props.templateName as string) || ''
      const buttonStyle = (block.props.buttonStyle as string) || 'default'

      // Parse template content
      let templateContent: TemplateContent[] = []
      try {
        templateContent = JSON.parse(templateContentStr)
      } catch {
        templateContent = []
      }

      // Handle template duplication
      const handleDuplicate = useCallback(() => {
        if (templateContent.length === 0) {
          // If no template content, just show a message
          console.log('No template content to duplicate')
          return
        }

        // TODO: Insert duplicated template blocks after this block
        // This would use editor.insertBlocks() with the template content

        setDuplicateCount((prev) => prev + 1)

        // Visual feedback
        const feedback = document.createElement('div')
        feedback.textContent = 'Content created!'
        feedback.style.cssText = `
          position: fixed;
          bottom: 24px;
          right: 24px;
          padding: 12px 20px;
          background: var(--accent-color);
          color: white;
          border-radius: 8px;
          font-size: 14px;
          font-weight: 500;
          z-index: 9999;
          animation: fadeInUp 0.3s ease-out;
        `
        document.body.appendChild(feedback)
        setTimeout(() => feedback.remove(), 2000)
      }, [templateContent])

      // Toggle template preview
      const togglePreview = useCallback(() => {
        setIsExpanded(!isExpanded)
      }, [isExpanded])

      // Update button text
      const handleButtonTextChange = useCallback((newText: string) => {
        editor.updateBlock(block, {
          props: { ...block.props, buttonText: newText },
        })
      }, [block, editor])

      // Update button style
      const handleStyleChange = useCallback((style: string) => {
        editor.updateBlock(block, {
          props: { ...block.props, buttonStyle: style },
        })
        setShowSettings(false)
      }, [block, editor])

      // Get button style classes
      const getButtonStyles = () => {
        switch (buttonStyle) {
          case 'primary':
            return {
              background: 'var(--accent-color)',
              color: 'white',
              border: 'none',
            }
          case 'outline':
            return {
              background: 'transparent',
              color: 'var(--accent-color)',
              border: '1px solid var(--accent-color)',
            }
          default:
            return {
              background: 'var(--bg-secondary)',
              color: 'var(--text-primary)',
              border: '1px solid var(--border-color)',
            }
        }
      }

      return (
        <div
          className="template-block"
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => {
            setIsHovered(false)
            setShowSettings(false)
          }}
          style={{
            margin: '12px 0',
            position: 'relative',
          }}
        >
          {/* Main template button */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
            }}
          >
            <button
              onClick={handleDuplicate}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                padding: '10px 16px',
                borderRadius: '6px',
                fontSize: '14px',
                fontWeight: 500,
                cursor: 'pointer',
                transition: 'all 0.15s ease',
                ...getButtonStyles(),
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateY(-1px)'
                e.currentTarget.style.boxShadow = '0 2px 8px rgba(0,0,0,0.1)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)'
                e.currentTarget.style.boxShadow = 'none'
              }}
            >
              <Plus size={16} />
              {buttonText}
            </button>

            {/* Preview toggle */}
            <AnimatePresence>
              {isHovered && (
                <motion.button
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.8 }}
                  onClick={togglePreview}
                  title={isExpanded ? 'Hide template' : 'Preview template'}
                  style={{
                    padding: '8px',
                    borderRadius: '4px',
                    background: 'none',
                    border: 'none',
                    color: 'var(--text-tertiary)',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    transition: 'color 0.15s',
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.color = 'var(--text-primary)'
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.color = 'var(--text-tertiary)'
                  }}
                >
                  <ChevronRight
                    size={16}
                    style={{
                      transform: isExpanded ? 'rotate(90deg)' : 'rotate(0deg)',
                      transition: 'transform 0.15s',
                    }}
                  />
                </motion.button>
              )}
            </AnimatePresence>

            {/* Settings button */}
            <AnimatePresence>
              {isHovered && (
                <motion.div
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.8 }}
                  style={{ position: 'relative' }}
                >
                  <button
                    onClick={() => setShowSettings(!showSettings)}
                    title="Template settings"
                    style={{
                      padding: '8px',
                      borderRadius: '4px',
                      background: showSettings ? 'var(--bg-hover)' : 'none',
                      border: 'none',
                      color: 'var(--text-tertiary)',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      transition: 'color 0.15s, background 0.15s',
                    }}
                    onMouseEnter={(e) => {
                      if (!showSettings) {
                        e.currentTarget.style.color = 'var(--text-primary)'
                      }
                    }}
                    onMouseLeave={(e) => {
                      if (!showSettings) {
                        e.currentTarget.style.color = 'var(--text-tertiary)'
                      }
                    }}
                  >
                    <Settings size={16} />
                  </button>

                  {/* Settings dropdown */}
                  <AnimatePresence>
                    {showSettings && (
                      <motion.div
                        initial={{ opacity: 0, y: -8, scale: 0.95 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: -8, scale: 0.95 }}
                        transition={{ duration: 0.15 }}
                        style={{
                          position: 'absolute',
                          top: '100%',
                          right: 0,
                          marginTop: '4px',
                          background: 'var(--bg-primary)',
                          borderRadius: '8px',
                          boxShadow: '0 4px 16px rgba(0,0,0,0.15), 0 0 0 1px rgba(0,0,0,0.05)',
                          padding: '8px',
                          minWidth: '180px',
                          zIndex: 100,
                        }}
                      >
                        <div
                          style={{
                            fontSize: '11px',
                            fontWeight: 500,
                            color: 'var(--text-tertiary)',
                            padding: '4px 8px',
                            textTransform: 'uppercase',
                            letterSpacing: '0.5px',
                          }}
                        >
                          Button Style
                        </div>
                        {[
                          { id: 'default', label: 'Default' },
                          { id: 'primary', label: 'Primary' },
                          { id: 'outline', label: 'Outline' },
                        ].map((style) => (
                          <button
                            key={style.id}
                            onClick={() => handleStyleChange(style.id)}
                            style={{
                              width: '100%',
                              padding: '8px',
                              borderRadius: '4px',
                              background: buttonStyle === style.id ? 'var(--accent-bg)' : 'none',
                              border: 'none',
                              color: buttonStyle === style.id ? 'var(--accent-color)' : 'var(--text-primary)',
                              fontSize: '13px',
                              textAlign: 'left',
                              cursor: 'pointer',
                              display: 'flex',
                              alignItems: 'center',
                              gap: '8px',
                              transition: 'background 0.1s',
                            }}
                            onMouseEnter={(e) => {
                              if (buttonStyle !== style.id) {
                                e.currentTarget.style.background = 'var(--bg-hover)'
                              }
                            }}
                            onMouseLeave={(e) => {
                              if (buttonStyle !== style.id) {
                                e.currentTarget.style.background = 'none'
                              }
                            }}
                          >
                            <Layout size={14} />
                            {style.label}
                          </button>
                        ))}

                        <div
                          style={{
                            height: 1,
                            background: 'var(--border-color)',
                            margin: '8px 0',
                          }}
                        />

                        <div
                          style={{
                            fontSize: '11px',
                            fontWeight: 500,
                            color: 'var(--text-tertiary)',
                            padding: '4px 8px',
                            textTransform: 'uppercase',
                            letterSpacing: '0.5px',
                          }}
                        >
                          Button Text
                        </div>
                        <input
                          type="text"
                          value={buttonText}
                          onChange={(e) => handleButtonTextChange(e.target.value)}
                          style={{
                            width: '100%',
                            padding: '8px',
                            border: '1px solid var(--border-color)',
                            borderRadius: '4px',
                            fontSize: '13px',
                            background: 'var(--bg-primary)',
                            color: 'var(--text-primary)',
                            margin: '0 0 4px',
                          }}
                          placeholder="Button text..."
                        />
                      </motion.div>
                    )}
                  </AnimatePresence>
                </motion.div>
              )}
            </AnimatePresence>

            {/* Usage count badge */}
            {duplicateCount > 0 && (
              <span
                style={{
                  fontSize: '11px',
                  color: 'var(--text-tertiary)',
                  marginLeft: '4px',
                }}
              >
                Used {duplicateCount} time{duplicateCount !== 1 ? 's' : ''}
              </span>
            )}
          </div>

          {/* Template preview/editor */}
          <AnimatePresence>
            {isExpanded && (
              <motion.div
                initial={{ height: 0, opacity: 0 }}
                animate={{ height: 'auto', opacity: 1 }}
                exit={{ height: 0, opacity: 0 }}
                transition={{ duration: 0.2 }}
                style={{ overflow: 'hidden' }}
              >
                <div
                  style={{
                    marginTop: '12px',
                    padding: '16px',
                    background: 'var(--bg-secondary)',
                    borderRadius: '8px',
                    border: '1px dashed var(--border-color)',
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      marginBottom: '12px',
                    }}
                  >
                    <span
                      style={{
                        fontSize: '12px',
                        fontWeight: 500,
                        color: 'var(--text-tertiary)',
                        textTransform: 'uppercase',
                        letterSpacing: '0.5px',
                      }}
                    >
                      Template Preview
                    </span>
                    <button
                      onClick={() => setIsEditing(!isEditing)}
                      style={{
                        padding: '4px 8px',
                        borderRadius: '4px',
                        background: 'none',
                        border: '1px solid var(--border-color)',
                        fontSize: '12px',
                        color: 'var(--text-secondary)',
                        cursor: 'pointer',
                      }}
                    >
                      {isEditing ? 'Done' : 'Edit Template'}
                    </button>
                  </div>

                  {templateContent.length > 0 ? (
                    <div className="template-preview-content">
                      {templateContent.map((item, index) => (
                        <div
                          key={index}
                          style={{
                            padding: '8px',
                            marginBottom: '4px',
                            background: 'var(--bg-primary)',
                            borderRadius: '4px',
                            fontSize: '13px',
                            color: 'var(--text-secondary)',
                          }}
                        >
                          {/* Simplified block preview */}
                          <span style={{ color: 'var(--text-tertiary)' }}>
                            {item.type || 'Block'}
                          </span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div
                      style={{
                        padding: '24px',
                        textAlign: 'center',
                        color: 'var(--text-tertiary)',
                        fontSize: '13px',
                      }}
                    >
                      <Copy size={24} style={{ marginBottom: '8px', opacity: 0.5 }} />
                      <p>No template content defined.</p>
                      <p style={{ marginTop: '4px', fontSize: '12px' }}>
                        Add content below to create a reusable template.
                      </p>
                    </div>
                  )}
                </div>
              </motion.div>
            )}
          </AnimatePresence>

          {/* Template name badge */}
          {templateName && (
            <div
              style={{
                marginTop: '4px',
                fontSize: '11px',
                color: 'var(--text-tertiary)',
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
              }}
            >
              <Layout size={12} />
              Template: {templateName}
            </div>
          )}

          {/* CSS for animations */}
          <style>{`
            @keyframes fadeInUp {
              from {
                opacity: 0;
                transform: translateY(10px);
              }
              to {
                opacity: 1;
                transform: translateY(0);
              }
            }
          `}</style>
        </div>
      )
    },
    // Parse HTML to recreate block when pasting or drag-dropping
    parse: (element: HTMLElement) => {
      if (element.classList.contains('template-block') || element.hasAttribute('data-template-button')) {
        return {
          buttonText: element.getAttribute('data-button-text') || 'New item from template',
          templateContent: element.getAttribute('data-template-content') || '[]',
          templateName: element.getAttribute('data-template-name') || '',
          buttonStyle: element.getAttribute('data-button-style') || 'default',
        }
      }
      return undefined
    },
    // Convert to external HTML for clipboard/export
    toExternalHTML: ({ block }) => {
      const { buttonText, templateContent, templateName, buttonStyle } = block.props as Record<string, string>
      return (
        <div
          className="template-block"
          data-template-button="true"
          data-button-text={buttonText}
          data-template-content={templateContent}
          data-template-name={templateName}
          data-button-style={buttonStyle}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '8px',
            padding: '10px 16px',
            borderRadius: '6px',
            background: '#f7f6f3',
            border: '1px solid #e3e2de',
          }}
        >
          <span>➕</span>
          {buttonText || 'New item from template'}
        </div>
      )
    },
  }
)
