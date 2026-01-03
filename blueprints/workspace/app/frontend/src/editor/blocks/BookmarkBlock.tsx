import { createReactBlockSpec } from '@blocknote/react'
import { useState, useEffect, useCallback, useRef } from 'react'
import { Link2, ExternalLink, RefreshCw, X, Globe, Edit2 } from 'lucide-react'
import { motion } from 'framer-motion'

interface BookmarkMetadata {
  title: string
  description: string
  image?: string
  favicon?: string
  siteName?: string
}

// Fetch URL metadata from backend or fallback to basic parsing
async function fetchUrlMetadata(url: string): Promise<BookmarkMetadata> {
  try {
    // Try to fetch metadata from our API
    const response = await fetch(`/api/v1/unfurl?url=${encodeURIComponent(url)}`)
    if (response.ok) {
      return await response.json()
    }
  } catch (err) {
    console.warn('Failed to fetch URL metadata from API:', err)
  }

  // Fallback: extract basic info from the URL
  try {
    const urlObj = new URL(url)
    const domain = urlObj.hostname.replace('www.', '')
    return {
      title: domain,
      description: url,
      siteName: domain,
      favicon: `https://www.google.com/s2/favicons?domain=${domain}&sz=32`,
    }
  } catch {
    return {
      title: 'Link',
      description: url,
    }
  }
}

export const BookmarkBlock = createReactBlockSpec(
  {
    type: 'bookmark',
    propSchema: {
      url: {
        default: '',
      },
      title: {
        default: '',
      },
      description: {
        default: '',
      },
      image: {
        default: '',
      },
      favicon: {
        default: '',
      },
      siteName: {
        default: '',
      },
    },
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [isEditing, setIsEditing] = useState(!block.props.url)
      const [urlInput, setUrlInput] = useState(block.props.url || '')
      const [isLoading, setIsLoading] = useState(false)
      const [error, setError] = useState<string | null>(null)
      const [imageError, setImageError] = useState(false)
      const [isHovered, setIsHovered] = useState(false)
      const inputRef = useRef<HTMLInputElement>(null)

      const { url, title, description, image, favicon, siteName } = block.props

      // Focus input when editing
      useEffect(() => {
        if (isEditing && inputRef.current) {
          inputRef.current.focus()
          inputRef.current.select()
        }
      }, [isEditing])

      // Fetch metadata when URL changes
      const fetchMetadata = useCallback(async (newUrl: string) => {
        if (!newUrl) return

        setIsLoading(true)
        setError(null)
        setImageError(false)

        try {
          const metadata = await fetchUrlMetadata(newUrl)
          editor.updateBlock(block, {
            props: {
              ...block.props,
              url: newUrl,
              title: metadata.title || '',
              description: metadata.description || '',
              image: metadata.image || '',
              favicon: metadata.favicon || '',
              siteName: metadata.siteName || '',
            },
          })
          setIsEditing(false)
        } catch (err) {
          setError('Failed to fetch link preview')
        } finally {
          setIsLoading(false)
        }
      }, [block, editor])

      const handleSubmit = useCallback((e?: React.FormEvent) => {
        e?.preventDefault()
        if (!urlInput.trim()) return

        // Add https:// if no protocol
        let normalizedUrl = urlInput.trim()
        if (!normalizedUrl.startsWith('http://') && !normalizedUrl.startsWith('https://')) {
          normalizedUrl = 'https://' + normalizedUrl
        }

        setUrlInput(normalizedUrl)
        fetchMetadata(normalizedUrl)
      }, [urlInput, fetchMetadata])

      const handleRefresh = useCallback((e: React.MouseEvent) => {
        e.preventDefault()
        e.stopPropagation()
        if (url) {
          fetchMetadata(url)
        }
      }, [url, fetchMetadata])

      const handleEdit = useCallback((e: React.MouseEvent) => {
        e.preventDefault()
        e.stopPropagation()
        setUrlInput(url)
        setIsEditing(true)
      }, [url])

      const handleRemove = useCallback((e: React.MouseEvent) => {
        e.preventDefault()
        e.stopPropagation()
        editor.removeBlocks([block])
      }, [block, editor])

      const getDomain = (u: string) => {
        try {
          return new URL(u).hostname.replace('www.', '')
        } catch {
          return u
        }
      }

      // Render URL input form
      if (isEditing || !url) {
        return (
          <div
            className="bookmark-block-input"
            style={{
              padding: '12px 16px',
              border: '1px solid var(--border-color, rgba(55, 53, 47, 0.16))',
              borderRadius: '4px',
              background: 'var(--bg-secondary, #f7f6f3)',
              margin: '4px 0',
            }}
          >
            <form onSubmit={handleSubmit} style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
              <Link2 size={16} style={{ color: 'var(--text-tertiary)', flexShrink: 0 }} />
              <input
                ref={inputRef}
                type="text"
                value={urlInput}
                onChange={(e) => setUrlInput(e.target.value)}
                placeholder="Paste a link to create a bookmark..."
                disabled={isLoading}
                style={{
                  flex: 1,
                  padding: '8px 12px',
                  border: '1px solid var(--border-color, #e3e2de)',
                  borderRadius: '4px',
                  fontSize: '14px',
                  outline: 'none',
                  background: 'var(--bg-primary, white)',
                  color: 'var(--text-primary, #37352f)',
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') {
                    if (url) {
                      setIsEditing(false)
                      setUrlInput(url)
                    } else {
                      editor.removeBlocks([block])
                    }
                  }
                }}
              />
              <button
                type="submit"
                disabled={isLoading || !urlInput.trim()}
                style={{
                  padding: '8px 16px',
                  background: 'var(--accent-color, #2383e2)',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: isLoading || !urlInput.trim() ? 'not-allowed' : 'pointer',
                  fontSize: '14px',
                  fontWeight: 500,
                  opacity: isLoading || !urlInput.trim() ? 0.5 : 1,
                  whiteSpace: 'nowrap',
                }}
              >
                {isLoading ? 'Loading...' : 'Embed link'}
              </button>
              {url && (
                <button
                  type="button"
                  onClick={() => {
                    setIsEditing(false)
                    setUrlInput(url)
                  }}
                  style={{
                    padding: '8px 16px',
                    background: 'none',
                    border: '1px solid var(--border-color, #e3e2de)',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    fontSize: '14px',
                    color: 'var(--text-primary)',
                  }}
                >
                  Cancel
                </button>
              )}
            </form>
            {error && (
              <div style={{ color: 'var(--danger-color, #e03e3e)', fontSize: '12px', marginTop: '8px' }}>
                {error}
              </div>
            )}
          </div>
        )
      }

      // Render bookmark card
      return (
        <motion.div
          className="bookmark-block"
          data-bookmark-url={url}
          data-bookmark-title={title}
          data-bookmark-description={description}
          data-bookmark-image={image}
          data-bookmark-favicon={favicon}
          data-bookmark-sitename={siteName}
          initial={{ opacity: 0, y: 4 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.2 }}
          style={{
            position: 'relative',
            display: 'flex',
            border: '1px solid var(--border-color, rgba(55, 53, 47, 0.16))',
            borderRadius: '4px',
            overflow: 'hidden',
            margin: '4px 0',
            cursor: 'pointer',
            transition: 'box-shadow 0.2s ease, border-color 0.2s ease',
            background: 'var(--bg-primary, white)',
            boxShadow: isHovered ? 'var(--shadow-sm)' : 'none',
          }}
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => setIsHovered(false)}
          onClick={() => window.open(url, '_blank', 'noopener,noreferrer')}
        >
          {/* Content section */}
          <div
            style={{
              flex: 1,
              padding: '14px 16px',
              minWidth: 0,
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'space-between',
            }}
          >
            <div>
              <div
                style={{
                  fontSize: '14px',
                  fontWeight: 500,
                  color: 'var(--text-primary, #37352f)',
                  marginBottom: '4px',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {title || getDomain(url)}
              </div>
              {description && (
                <div
                  style={{
                    fontSize: '12px',
                    color: 'var(--text-secondary, #787774)',
                    overflow: 'hidden',
                    display: '-webkit-box',
                    WebkitLineClamp: 2,
                    WebkitBoxOrient: 'vertical',
                    lineHeight: 1.4,
                  }}
                >
                  {description}
                </div>
              )}
            </div>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                marginTop: '10px',
              }}
            >
              {favicon && !imageError ? (
                <img
                  src={favicon}
                  alt=""
                  width={14}
                  height={14}
                  style={{ borderRadius: '2px' }}
                  onError={() => setImageError(true)}
                />
              ) : (
                <Globe size={14} style={{ color: 'var(--text-tertiary, #9b9a97)' }} />
              )}
              <span
                style={{
                  fontSize: '12px',
                  color: 'var(--text-tertiary, #9b9a97)',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {siteName || getDomain(url)}
              </span>
            </div>
          </div>

          {/* Image section */}
          {image && !imageError && (
            <div
              style={{
                width: '200px',
                flexShrink: 0,
                background: 'var(--bg-secondary, #f7f6f3)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <img
                src={image}
                alt=""
                style={{
                  width: '100%',
                  height: '100%',
                  objectFit: 'cover',
                }}
                onError={() => setImageError(true)}
              />
            </div>
          )}

          {/* Actions overlay */}
          {isHovered && (
            <motion.div
              className="bookmark-actions"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ duration: 0.1 }}
              onClick={(e) => e.stopPropagation()}
              style={{
                position: 'absolute',
                top: '8px',
                right: '8px',
                display: 'flex',
                gap: '4px',
                background: 'var(--bg-primary, white)',
                padding: '4px',
                borderRadius: '4px',
                boxShadow: 'var(--shadow-sm)',
              }}
            >
              <button
                onClick={handleRefresh}
                title="Refresh preview"
                style={{
                  padding: '6px',
                  background: 'none',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: 'var(--text-secondary)',
                  transition: 'background 0.1s ease',
                }}
                onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-hover)'}
                onMouseLeave={(e) => e.currentTarget.style.background = 'none'}
              >
                <RefreshCw size={14} />
              </button>
              <button
                onClick={handleEdit}
                title="Edit URL"
                style={{
                  padding: '6px',
                  background: 'none',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: 'var(--text-secondary)',
                  transition: 'background 0.1s ease',
                }}
                onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-hover)'}
                onMouseLeave={(e) => e.currentTarget.style.background = 'none'}
              >
                <Edit2 size={14} />
              </button>
              <button
                onClick={handleRemove}
                title="Remove"
                style={{
                  padding: '6px',
                  background: 'none',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: 'var(--text-secondary)',
                  transition: 'background 0.1s ease',
                }}
                onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-hover)'}
                onMouseLeave={(e) => e.currentTarget.style.background = 'none'}
              >
                <X size={14} />
              </button>
            </motion.div>
          )}
        </motion.div>
      )
    },
    // Parse HTML to recreate block when pasting or drag-dropping
    parse: (element: HTMLElement) => {
      if (element.classList.contains('bookmark-block') || element.hasAttribute('data-bookmark-url')) {
        return {
          url: element.getAttribute('data-bookmark-url') || '',
          title: element.getAttribute('data-bookmark-title') || '',
          description: element.getAttribute('data-bookmark-description') || '',
          image: element.getAttribute('data-bookmark-image') || '',
          favicon: element.getAttribute('data-bookmark-favicon') || '',
          siteName: element.getAttribute('data-bookmark-sitename') || '',
        }
      }
      return undefined
    },
    // Convert to external HTML for clipboard/export
    toExternalHTML: ({ block }) => {
      const { url, title, description, favicon, siteName } = block.props as Record<string, string>
      const domain = (() => {
        try {
          return new URL(url || '').hostname.replace('www.', '')
        } catch {
          return url || ''
        }
      })()

      return (
        <a
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          className="bookmark-block"
          data-bookmark-url={url}
          data-bookmark-title={title}
          data-bookmark-description={description}
          data-bookmark-favicon={favicon}
          data-bookmark-sitename={siteName}
          style={{
            display: 'block',
            padding: '12px 16px',
            border: '1px solid rgba(55, 53, 47, 0.16)',
            borderRadius: '4px',
            textDecoration: 'none',
            color: 'inherit',
          }}
        >
          <div style={{ fontWeight: 500 }}>{title || domain}</div>
          {description && <div style={{ fontSize: '12px', color: '#787774', marginTop: '4px' }}>{description}</div>}
          <div style={{ fontSize: '12px', color: '#9b9a97', marginTop: '8px' }}>{siteName || domain}</div>
        </a>
      )
    },
  }
)
