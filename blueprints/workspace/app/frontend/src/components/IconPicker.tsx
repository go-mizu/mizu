import { useState, useRef, useEffect, useCallback, lazy, Suspense } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X, Smile, Trash2, Upload, Link2 } from 'lucide-react'

// Lazy load emoji picker for better performance
const Picker = lazy(() => import('@emoji-mart/react').then(mod => ({ default: mod.default })))

interface IconPickerProps {
  currentIcon: string
  onSelect: (icon: string) => void
  size?: 'sm' | 'md' | 'lg'
  allowUpload?: boolean
  allowUrl?: boolean
}

const COMMON_EMOJIS = [
  '📄', '📝', '📋', '📌', '📁', '🗂️', '📊', '📈', '📉', '🎯',
  '💡', '⭐', '✅', '❌', '⚠️', '🔥', '💪', '🚀', '🎉', '💼',
  '📅', '🗓️', '⏰', '🔔', '📧', '💬', '🔍', '⚙️', '🔧', '🛠️',
  '🏠', '🏢', '🌍', '💰', '📦', '🎨', '📸', '🎵', '🎮', '📚',
  '✏️', '🖊️', '📎', '📐', '📏', '🗃️', '🗄️', '🔒', '🔑', '💎',
]

const ICON_SIZES = {
  sm: 24,
  md: 32,
  lg: 48,
}

export function IconPicker({
  currentIcon,
  onSelect,
  size = 'md',
  allowUpload = false,
  allowUrl = false,
}: IconPickerProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [activeTab, setActiveTab] = useState<'emoji' | 'upload' | 'url'>('emoji')
  const [urlInput, setUrlInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const popupRef = useRef<HTMLDivElement>(null)

  // Close on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Close on escape
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setIsOpen(false)
      }
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [])

  const handleSelect = useCallback((emoji: string) => {
    onSelect(emoji)
    setIsOpen(false)
  }, [onSelect])

  const handleEmojiSelect = useCallback((data: { native: string }) => {
    handleSelect(data.native)
  }, [handleSelect])

  const handleRemove = useCallback(() => {
    onSelect('')
    setIsOpen(false)
  }, [onSelect])

  const handleUrlSubmit = useCallback((e: React.FormEvent) => {
    e.preventDefault()
    if (urlInput.trim()) {
      // For URL-based icons, we store the URL with a prefix
      handleSelect(`url:${urlInput.trim()}`)
    }
  }, [urlInput, handleSelect])

  const handleFileUpload = useCallback(async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setIsLoading(true)
    try {
      // In a real app, upload to server
      const reader = new FileReader()
      reader.onload = () => {
        const dataUrl = reader.result as string
        handleSelect(`data:${dataUrl}`)
      }
      reader.readAsDataURL(file)
    } catch (err) {
      console.error('Upload failed:', err)
    } finally {
      setIsLoading(false)
    }
  }, [handleSelect])

  // Render the icon based on type
  const renderIcon = (icon: string) => {
    if (!icon) return null

    if (icon.startsWith('url:')) {
      const url = icon.slice(4)
      return (
        <img
          src={url}
          alt="Icon"
          style={{
            width: ICON_SIZES[size],
            height: ICON_SIZES[size],
            objectFit: 'cover',
            borderRadius: 4,
          }}
        />
      )
    }

    if (icon.startsWith('data:')) {
      const dataUrl = icon.slice(5)
      return (
        <img
          src={dataUrl}
          alt="Icon"
          style={{
            width: ICON_SIZES[size],
            height: ICON_SIZES[size],
            objectFit: 'cover',
            borderRadius: 4,
          }}
        />
      )
    }

    return (
      <span style={{ fontSize: ICON_SIZES[size] * 0.8, lineHeight: 1 }}>
        {icon}
      </span>
    )
  }

  const iconSize = ICON_SIZES[size]

  return (
    <div className="icon-picker" ref={containerRef} style={{ position: 'relative', display: 'inline-block' }}>
      <button
        className="icon-picker-trigger"
        onClick={() => setIsOpen(!isOpen)}
        style={{
          width: iconSize + 8,
          height: iconSize + 8,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'none',
          border: 'none',
          borderRadius: 'var(--radius-md)',
          cursor: 'pointer',
          transition: 'background 0.15s ease',
        }}
        onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-hover)'}
        onMouseLeave={(e) => e.currentTarget.style.background = 'none'}
        title={currentIcon ? 'Change icon' : 'Add icon'}
      >
        {currentIcon ? (
          renderIcon(currentIcon)
        ) : (
          <span style={{ color: 'var(--text-placeholder)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Smile size={iconSize * 0.6} />
          </span>
        )}
      </button>

      <AnimatePresence>
        {isOpen && (
          <motion.div
            ref={popupRef}
            className="icon-picker-popup"
            initial={{ opacity: 0, scale: 0.95, y: -8 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: -8 }}
            transition={{ duration: 0.15, ease: 'easeOut' }}
            style={{
              position: 'absolute',
              top: '100%',
              left: 0,
              zIndex: 1000,
              background: 'var(--bg-primary)',
              border: '1px solid var(--border-color)',
              borderRadius: 'var(--radius-lg)',
              boxShadow: 'var(--shadow-lg)',
              padding: 0,
              minWidth: 352,
              overflow: 'hidden',
            }}
          >
            {/* Header with tabs */}
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 4,
              padding: '8px 12px',
              borderBottom: '1px solid var(--border-color)',
            }}>
              <button
                onClick={() => setActiveTab('emoji')}
                style={{
                  flex: 1,
                  padding: '6px 12px',
                  background: activeTab === 'emoji' ? 'var(--bg-hover)' : 'transparent',
                  border: 'none',
                  borderRadius: 'var(--radius-sm)',
                  cursor: 'pointer',
                  fontSize: 13,
                  fontWeight: 500,
                  color: activeTab === 'emoji' ? 'var(--text-primary)' : 'var(--text-secondary)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: 6,
                }}
              >
                <Smile size={14} />
                Emoji
              </button>
              {allowUpload && (
                <button
                  onClick={() => setActiveTab('upload')}
                  style={{
                    flex: 1,
                    padding: '6px 12px',
                    background: activeTab === 'upload' ? 'var(--bg-hover)' : 'transparent',
                    border: 'none',
                    borderRadius: 'var(--radius-sm)',
                    cursor: 'pointer',
                    fontSize: 13,
                    fontWeight: 500,
                    color: activeTab === 'upload' ? 'var(--text-primary)' : 'var(--text-secondary)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: 6,
                  }}
                >
                  <Upload size={14} />
                  Upload
                </button>
              )}
              {allowUrl && (
                <button
                  onClick={() => setActiveTab('url')}
                  style={{
                    flex: 1,
                    padding: '6px 12px',
                    background: activeTab === 'url' ? 'var(--bg-hover)' : 'transparent',
                    border: 'none',
                    borderRadius: 'var(--radius-sm)',
                    cursor: 'pointer',
                    fontSize: 13,
                    fontWeight: 500,
                    color: activeTab === 'url' ? 'var(--text-primary)' : 'var(--text-secondary)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: 6,
                  }}
                >
                  <Link2 size={14} />
                  URL
                </button>
              )}
            </div>

            {/* Content based on tab */}
            {activeTab === 'emoji' && (
              <div style={{ padding: 8 }}>
                {/* Quick access emojis */}
                <div style={{ marginBottom: 8 }}>
                  <div style={{
                    fontSize: 11,
                    fontWeight: 500,
                    color: 'var(--text-secondary)',
                    padding: '4px 4px 8px',
                    textTransform: 'uppercase',
                    letterSpacing: 0.5,
                  }}>
                    Popular
                  </div>
                  <div style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(10, 1fr)',
                    gap: 2,
                  }}>
                    {COMMON_EMOJIS.map((emoji, i) => (
                      <button
                        key={i}
                        onClick={() => handleSelect(emoji)}
                        style={{
                          width: 32,
                          height: 32,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          background: currentIcon === emoji ? 'var(--accent-bg)' : 'transparent',
                          border: 'none',
                          borderRadius: 'var(--radius-sm)',
                          cursor: 'pointer',
                          fontSize: 20,
                          transition: 'background 0.1s ease',
                        }}
                        onMouseEnter={(e) => {
                          if (currentIcon !== emoji) {
                            e.currentTarget.style.background = 'var(--bg-hover)'
                          }
                        }}
                        onMouseLeave={(e) => {
                          if (currentIcon !== emoji) {
                            e.currentTarget.style.background = 'transparent'
                          }
                        }}
                      >
                        {emoji}
                      </button>
                    ))}
                  </div>
                </div>

                {/* Full emoji picker */}
                <div style={{ borderTop: '1px solid var(--border-color)', paddingTop: 8 }}>
                  <Suspense fallback={
                    <div style={{ height: 300, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-tertiary)' }}>
                      Loading emojis...
                    </div>
                  }>
                    <Picker
                      onEmojiSelect={handleEmojiSelect}
                      theme="auto"
                      set="native"
                      perLine={10}
                      emojiSize={24}
                      emojiButtonSize={32}
                      maxFrequentRows={0}
                      previewPosition="none"
                      skinTonePosition="none"
                      dynamicWidth={false}
                      navPosition="bottom"
                    />
                  </Suspense>
                </div>
              </div>
            )}

            {activeTab === 'upload' && (
              <div style={{ padding: 24, textAlign: 'center' }}>
                <label
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    gap: 12,
                    padding: 32,
                    border: '2px dashed var(--border-color)',
                    borderRadius: 'var(--radius-md)',
                    cursor: 'pointer',
                    transition: 'border-color 0.15s ease',
                  }}
                  onMouseEnter={(e) => e.currentTarget.style.borderColor = 'var(--accent-color)'}
                  onMouseLeave={(e) => e.currentTarget.style.borderColor = 'var(--border-color)'}
                >
                  <Upload size={32} style={{ color: 'var(--text-tertiary)' }} />
                  <div style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                    {isLoading ? 'Uploading...' : 'Click to upload an image'}
                  </div>
                  <div style={{ color: 'var(--text-tertiary)', fontSize: 12 }}>
                    PNG, JPG, GIF up to 2MB
                  </div>
                  <input
                    type="file"
                    accept="image/*"
                    onChange={handleFileUpload}
                    style={{ display: 'none' }}
                    disabled={isLoading}
                  />
                </label>
              </div>
            )}

            {activeTab === 'url' && (
              <div style={{ padding: 16 }}>
                <form onSubmit={handleUrlSubmit}>
                  <div style={{ marginBottom: 12 }}>
                    <label style={{ display: 'block', fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4 }}>
                      Image URL
                    </label>
                    <input
                      type="url"
                      value={urlInput}
                      onChange={(e) => setUrlInput(e.target.value)}
                      placeholder="https://example.com/icon.png"
                      style={{
                        width: '100%',
                        padding: '8px 12px',
                        border: '1px solid var(--border-color)',
                        borderRadius: 'var(--radius-sm)',
                        fontSize: 14,
                        outline: 'none',
                        background: 'var(--bg-primary)',
                        color: 'var(--text-primary)',
                      }}
                      autoFocus
                    />
                  </div>
                  <button
                    type="submit"
                    disabled={!urlInput.trim()}
                    style={{
                      width: '100%',
                      padding: '8px 16px',
                      background: 'var(--accent-color)',
                      color: 'white',
                      border: 'none',
                      borderRadius: 'var(--radius-sm)',
                      cursor: urlInput.trim() ? 'pointer' : 'not-allowed',
                      fontSize: 14,
                      fontWeight: 500,
                      opacity: urlInput.trim() ? 1 : 0.5,
                    }}
                  >
                    Add icon
                  </button>
                </form>
              </div>
            )}

            {/* Remove button */}
            {currentIcon && (
              <div style={{
                borderTop: '1px solid var(--border-color)',
                padding: 8,
              }}>
                <button
                  onClick={handleRemove}
                  style={{
                    width: '100%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: 8,
                    padding: '8px 12px',
                    background: 'transparent',
                    border: 'none',
                    borderRadius: 'var(--radius-sm)',
                    cursor: 'pointer',
                    fontSize: 13,
                    color: 'var(--danger-color)',
                    transition: 'background 0.1s ease',
                  }}
                  onMouseEnter={(e) => e.currentTarget.style.background = 'var(--danger-bg)'}
                  onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
                >
                  <Trash2 size={14} />
                  Remove icon
                </button>
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
