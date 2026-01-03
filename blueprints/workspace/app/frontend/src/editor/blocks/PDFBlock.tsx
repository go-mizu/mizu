import { createReactBlockSpec } from '@blocknote/react'
import { useState, useRef, useCallback, useEffect } from 'react'
import {
  FileText,
  Upload,
  ZoomIn,
  ZoomOut,
  ChevronLeft,
  ChevronRight,
  Download,
  Maximize2,
  Minimize2,
  Link2,
  Loader2,
  AlertCircle,
  RotateCw,
} from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'

// PDF zoom levels
const ZOOM_LEVELS = [0.5, 0.75, 1, 1.25, 1.5, 2, 3]

export const PDFBlock = createReactBlockSpec(
  {
    type: 'pdf',
    propSchema: {
      url: {
        default: '',
      },
      name: {
        default: '',
      },
    },
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [currentPage, setCurrentPage] = useState(1)
      const [totalPages, setTotalPages] = useState(0)
      const [scale, setScale] = useState(1)
      const [isFullscreen, setIsFullscreen] = useState(false)
      const [showUpload, setShowUpload] = useState(!block.props.url)
      const [isLoading, setIsLoading] = useState(true)
      const [error, setError] = useState<string | null>(null)
      const [rotation, setRotation] = useState(0)
      const [isHovered, setIsHovered] = useState(false)

      const containerRef = useRef<HTMLDivElement>(null)
      const fileInputRef = useRef<HTMLInputElement>(null)
      const iframeRef = useRef<HTMLIFrameElement>(null)

      // Handle file selection
      const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0]
        if (!file) return

        if (file.type !== 'application/pdf') {
          setError('Please select a PDF file')
          return
        }

        const url = URL.createObjectURL(file)
        editor.updateBlock(block, {
          props: {
            ...block.props,
            url,
            name: file.name,
          },
        })
        setShowUpload(false)
        setError(null)
        setIsLoading(true)
      }

      // Handle URL input
      const handleUrlInput = () => {
        const url = prompt('Enter PDF URL (direct link to PDF file):')
        if (url) {
          editor.updateBlock(block, {
            props: {
              ...block.props,
              url,
              name: url.split('/').pop()?.split('?')[0] || 'Document.pdf',
            },
          })
          setShowUpload(false)
          setError(null)
          setIsLoading(true)
        }
      }

      // Handle download
      const handleDownload = useCallback(() => {
        if (block.props.url) {
          const a = document.createElement('a')
          a.href = block.props.url
          a.download = block.props.name || 'document.pdf'
          a.target = '_blank'
          a.click()
        }
      }, [block.props.url, block.props.name])

      // Toggle fullscreen
      const toggleFullscreen = useCallback(async () => {
        if (!containerRef.current) return

        try {
          if (!isFullscreen) {
            await containerRef.current.requestFullscreen()
          } else {
            await document.exitFullscreen()
          }
        } catch (err) {
          console.error('Fullscreen error:', err)
        }
      }, [isFullscreen])

      // Handle fullscreen change events
      useEffect(() => {
        const handleFullscreenChange = () => {
          setIsFullscreen(!!document.fullscreenElement)
        }

        document.addEventListener('fullscreenchange', handleFullscreenChange)
        return () => {
          document.removeEventListener('fullscreenchange', handleFullscreenChange)
        }
      }, [])

      // Zoom functions
      const zoomIn = useCallback(() => {
        const currentIndex = ZOOM_LEVELS.findIndex(z => z >= scale)
        if (currentIndex < ZOOM_LEVELS.length - 1) {
          setScale(ZOOM_LEVELS[currentIndex + 1])
        }
      }, [scale])

      const zoomOut = useCallback(() => {
        const currentIndex = ZOOM_LEVELS.findIndex(z => z >= scale)
        if (currentIndex > 0) {
          setScale(ZOOM_LEVELS[currentIndex - 1])
        }
      }, [scale])

      // Page navigation
      const goToPage = useCallback((page: number) => {
        const validPage = Math.max(1, Math.min(totalPages || 1, page))
        setCurrentPage(validPage)
      }, [totalPages])

      // Rotate PDF
      const rotate = useCallback(() => {
        setRotation((r) => (r + 90) % 360)
      }, [])

      // Handle iframe load
      const handleIframeLoad = useCallback(() => {
        setIsLoading(false)
        setError(null)
        // For basic iframe embedding, we can't reliably get page count
        // Set a reasonable default
        if (totalPages === 0) {
          setTotalPages(1)
        }
      }, [totalPages])

      // Handle iframe error
      const handleIframeError = useCallback(() => {
        setIsLoading(false)
        setError('Failed to load PDF. The file may be corrupted or the URL is invalid.')
      }, [])

      // Build PDF viewer URL with parameters
      const getPdfViewerUrl = useCallback(() => {
        if (!block.props.url) return ''

        // Use browser's built-in PDF viewer with hash parameters
        const params = new URLSearchParams()
        params.set('page', currentPage.toString())
        params.set('zoom', Math.round(scale * 100).toString())

        // For local files (blob URLs), we can't add query params
        // Use hash fragment instead
        const baseUrl = block.props.url
        return `${baseUrl}#page=${currentPage}&zoom=${Math.round(scale * 100)}`
      }, [block.props.url, currentPage, scale])

      // Upload mode UI
      if (showUpload || !block.props.url) {
        return (
          <div
            className="pdf-block upload-mode"
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              padding: '48px 32px',
              background: 'var(--bg-secondary)',
              borderRadius: '8px',
              border: '1px dashed var(--border-color)',
              margin: '8px 0',
            }}
          >
            <input
              ref={fileInputRef}
              type="file"
              accept=".pdf,application/pdf"
              onChange={handleFileSelect}
              style={{ display: 'none' }}
            />
            <FileText size={40} style={{ color: 'var(--text-tertiary)', marginBottom: '16px' }} />
            <p style={{ color: 'var(--text-primary)', marginBottom: '8px', fontSize: '15px', fontWeight: 500 }}>
              Add a PDF document
            </p>
            <p style={{ color: 'var(--text-tertiary)', marginBottom: '20px', fontSize: '13px' }}>
              Upload a file or embed from URL
            </p>
            <div style={{ display: 'flex', gap: '12px' }}>
              <button
                onClick={() => fileInputRef.current?.click()}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  padding: '10px 20px',
                  background: 'var(--accent-color)',
                  color: 'white',
                  border: 'none',
                  borderRadius: '6px',
                  fontSize: '14px',
                  fontWeight: 500,
                  cursor: 'pointer',
                  transition: 'opacity 0.15s',
                }}
                onMouseEnter={(e) => { e.currentTarget.style.opacity = '0.9' }}
                onMouseLeave={(e) => { e.currentTarget.style.opacity = '1' }}
              >
                <Upload size={16} />
                Upload PDF
              </button>
              <button
                onClick={handleUrlInput}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  padding: '10px 20px',
                  background: 'var(--bg-primary)',
                  color: 'var(--text-primary)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '6px',
                  fontSize: '14px',
                  fontWeight: 500,
                  cursor: 'pointer',
                  transition: 'background 0.15s',
                }}
                onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)' }}
                onMouseLeave={(e) => { e.currentTarget.style.background = 'var(--bg-primary)' }}
              >
                <Link2 size={16} />
                Embed URL
              </button>
            </div>
            {error && (
              <div style={{ marginTop: '16px', color: 'var(--danger-color)', fontSize: '13px', display: 'flex', alignItems: 'center', gap: '6px' }}>
                <AlertCircle size={14} />
                {error}
              </div>
            )}
          </div>
        )
      }

      return (
        <div
          ref={containerRef}
          className={`pdf-block ${isFullscreen ? 'fullscreen' : ''}`}
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => setIsHovered(false)}
          style={{
            margin: '8px 0',
            borderRadius: '8px',
            overflow: 'hidden',
            border: '1px solid var(--border-color)',
            background: 'var(--bg-secondary)',
            position: 'relative',
          }}
        >
          {/* Toolbar */}
          <div
            className="pdf-toolbar"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '8px 12px',
              background: 'var(--bg-primary)',
              borderBottom: '1px solid var(--border-color)',
            }}
          >
            {/* Left section: File info */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: 0, flex: 1 }}>
              <FileText size={16} style={{ color: 'var(--text-tertiary)', flexShrink: 0 }} />
              <span
                style={{
                  fontSize: '13px',
                  color: 'var(--text-primary)',
                  fontWeight: 500,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {block.props.name || 'Document.pdf'}
              </span>
            </div>

            {/* Center section: Page navigation */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
              <button
                onClick={() => goToPage(currentPage - 1)}
                disabled={currentPage <= 1}
                title="Previous page"
                style={{
                  width: '28px',
                  height: '28px',
                  borderRadius: '4px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'none',
                  color: currentPage <= 1 ? 'var(--text-tertiary)' : 'var(--text-secondary)',
                  border: 'none',
                  cursor: currentPage <= 1 ? 'not-allowed' : 'pointer',
                  opacity: currentPage <= 1 ? 0.5 : 1,
                  transition: 'background 0.15s',
                }}
                onMouseEnter={(e) => {
                  if (currentPage > 1) e.currentTarget.style.background = 'var(--bg-hover)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'none'
                }}
              >
                <ChevronLeft size={16} />
              </button>
              <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                <input
                  type="number"
                  min={1}
                  max={totalPages || 1}
                  value={currentPage}
                  onChange={(e) => goToPage(parseInt(e.target.value) || 1)}
                  style={{
                    width: '40px',
                    padding: '4px 8px',
                    border: '1px solid var(--border-color)',
                    borderRadius: '4px',
                    fontSize: '12px',
                    textAlign: 'center',
                    background: 'var(--bg-primary)',
                    color: 'var(--text-primary)',
                  }}
                />
                <span style={{ fontSize: '12px', color: 'var(--text-tertiary)' }}>
                  / {totalPages || '?'}
                </span>
              </div>
              <button
                onClick={() => goToPage(currentPage + 1)}
                disabled={totalPages > 0 && currentPage >= totalPages}
                title="Next page"
                style={{
                  width: '28px',
                  height: '28px',
                  borderRadius: '4px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'none',
                  color: (totalPages > 0 && currentPage >= totalPages) ? 'var(--text-tertiary)' : 'var(--text-secondary)',
                  border: 'none',
                  cursor: (totalPages > 0 && currentPage >= totalPages) ? 'not-allowed' : 'pointer',
                  opacity: (totalPages > 0 && currentPage >= totalPages) ? 0.5 : 1,
                  transition: 'background 0.15s',
                }}
                onMouseEnter={(e) => {
                  if (!(totalPages > 0 && currentPage >= totalPages)) e.currentTarget.style.background = 'var(--bg-hover)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'none'
                }}
              >
                <ChevronRight size={16} />
              </button>
            </div>

            {/* Right section: Zoom and actions */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
              {/* Zoom controls */}
              <button
                onClick={zoomOut}
                disabled={scale <= ZOOM_LEVELS[0]}
                title="Zoom out"
                style={{
                  width: '28px',
                  height: '28px',
                  borderRadius: '4px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'none',
                  color: scale <= ZOOM_LEVELS[0] ? 'var(--text-tertiary)' : 'var(--text-secondary)',
                  border: 'none',
                  cursor: scale <= ZOOM_LEVELS[0] ? 'not-allowed' : 'pointer',
                  transition: 'background 0.15s',
                }}
                onMouseEnter={(e) => {
                  if (scale > ZOOM_LEVELS[0]) e.currentTarget.style.background = 'var(--bg-hover)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'none'
                }}
              >
                <ZoomOut size={16} />
              </button>
              <span
                style={{
                  fontSize: '12px',
                  color: 'var(--text-secondary)',
                  minWidth: '48px',
                  textAlign: 'center',
                  fontWeight: 500,
                }}
              >
                {Math.round(scale * 100)}%
              </span>
              <button
                onClick={zoomIn}
                disabled={scale >= ZOOM_LEVELS[ZOOM_LEVELS.length - 1]}
                title="Zoom in"
                style={{
                  width: '28px',
                  height: '28px',
                  borderRadius: '4px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'none',
                  color: scale >= ZOOM_LEVELS[ZOOM_LEVELS.length - 1] ? 'var(--text-tertiary)' : 'var(--text-secondary)',
                  border: 'none',
                  cursor: scale >= ZOOM_LEVELS[ZOOM_LEVELS.length - 1] ? 'not-allowed' : 'pointer',
                  transition: 'background 0.15s',
                }}
                onMouseEnter={(e) => {
                  if (scale < ZOOM_LEVELS[ZOOM_LEVELS.length - 1]) e.currentTarget.style.background = 'var(--bg-hover)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'none'
                }}
              >
                <ZoomIn size={16} />
              </button>

              <div style={{ width: 1, height: 20, background: 'var(--border-color)', margin: '0 4px' }} />

              {/* Rotate */}
              <button
                onClick={rotate}
                title="Rotate"
                style={{
                  width: '28px',
                  height: '28px',
                  borderRadius: '4px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'none',
                  color: 'var(--text-secondary)',
                  border: 'none',
                  cursor: 'pointer',
                  transition: 'background 0.15s, color 0.15s',
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
                <RotateCw size={16} />
              </button>

              {/* Download */}
              <button
                onClick={handleDownload}
                title="Download PDF"
                style={{
                  width: '28px',
                  height: '28px',
                  borderRadius: '4px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'none',
                  color: 'var(--text-secondary)',
                  border: 'none',
                  cursor: 'pointer',
                  transition: 'background 0.15s, color 0.15s',
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
                <Download size={16} />
              </button>

              {/* Fullscreen */}
              <button
                onClick={toggleFullscreen}
                title={isFullscreen ? 'Exit fullscreen' : 'Fullscreen'}
                style={{
                  width: '28px',
                  height: '28px',
                  borderRadius: '4px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'none',
                  color: 'var(--text-secondary)',
                  border: 'none',
                  cursor: 'pointer',
                  transition: 'background 0.15s, color 0.15s',
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
                {isFullscreen ? <Minimize2 size={16} /> : <Maximize2 size={16} />}
              </button>
            </div>
          </div>

          {/* PDF Viewer */}
          <div
            className="pdf-viewer-container"
            style={{
              position: 'relative',
              height: isFullscreen ? 'calc(100vh - 56px)' : '600px',
              background: '#525659',
              overflow: 'auto',
            }}
          >
            {/* Loading overlay */}
            <AnimatePresence>
              {isLoading && (
                <motion.div
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    right: 0,
                    bottom: 0,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: 'rgba(82, 86, 89, 0.9)',
                    zIndex: 10,
                  }}
                >
                  <Loader2
                    size={32}
                    style={{
                      color: 'white',
                      animation: 'spin 1s linear infinite',
                    }}
                  />
                  <span style={{ color: 'white', marginTop: '12px', fontSize: '14px' }}>
                    Loading PDF...
                  </span>
                </motion.div>
              )}
            </AnimatePresence>

            {/* Error overlay */}
            {error && (
              <div
                style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  right: 0,
                  bottom: 0,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'rgba(82, 86, 89, 0.95)',
                  zIndex: 10,
                }}
              >
                <AlertCircle size={40} style={{ color: '#ff6b6b', marginBottom: '16px' }} />
                <p style={{ color: 'white', fontSize: '15px', marginBottom: '8px' }}>
                  Unable to load PDF
                </p>
                <p style={{ color: 'rgba(255,255,255,0.7)', fontSize: '13px', textAlign: 'center', maxWidth: '300px' }}>
                  {error}
                </p>
                <button
                  onClick={() => {
                    setShowUpload(true)
                    setError(null)
                  }}
                  style={{
                    marginTop: '20px',
                    padding: '8px 16px',
                    background: 'white',
                    color: '#37352f',
                    border: 'none',
                    borderRadius: '6px',
                    fontSize: '13px',
                    fontWeight: 500,
                    cursor: 'pointer',
                  }}
                >
                  Try another file
                </button>
              </div>
            )}

            {/* PDF iframe */}
            <div
              style={{
                width: '100%',
                height: '100%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                transform: `rotate(${rotation}deg)`,
                transition: 'transform 0.3s ease',
              }}
            >
              <iframe
                ref={iframeRef}
                src={getPdfViewerUrl()}
                title={block.props.name || 'PDF Viewer'}
                onLoad={handleIframeLoad}
                onError={handleIframeError}
                style={{
                  width: rotation % 180 === 0 ? '100%' : 'calc(100vh - 56px)',
                  height: rotation % 180 === 0 ? '100%' : '100%',
                  border: 'none',
                  background: 'white',
                }}
              />
            </div>
          </div>

          {/* CSS for spinner animation */}
          <style>{`
            @keyframes spin {
              from { transform: rotate(0deg); }
              to { transform: rotate(360deg); }
            }
            .pdf-block.fullscreen {
              position: fixed;
              top: 0;
              left: 0;
              right: 0;
              bottom: 0;
              z-index: 9999;
              margin: 0;
              border-radius: 0;
            }
            .pdf-block input[type="number"]::-webkit-inner-spin-button,
            .pdf-block input[type="number"]::-webkit-outer-spin-button {
              -webkit-appearance: none;
              margin: 0;
            }
            .pdf-block input[type="number"] {
              -moz-appearance: textfield;
            }
          `}</style>
        </div>
      )
    },
    // Parse HTML to recreate block when pasting or drag-dropping
    parse: (element: HTMLElement) => {
      if (element.classList.contains('pdf-block') || element.hasAttribute('data-pdf-url')) {
        return {
          url: element.getAttribute('data-pdf-url') || '',
          name: element.getAttribute('data-pdf-name') || '',
        }
      }
      return undefined
    },
    // Convert to external HTML for clipboard/export
    toExternalHTML: ({ block }) => {
      const { url, name } = block.props as Record<string, string>
      return (
        <div
          className="pdf-block"
          data-pdf-url={url}
          data-pdf-name={name}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
            padding: '12px 16px',
            border: '1px solid rgba(55, 53, 47, 0.16)',
            borderRadius: '4px',
          }}
        >
          <span style={{ fontSize: '24px' }}>📄</span>
          <div>
            <div style={{ fontWeight: 500 }}>{name || 'PDF Document'}</div>
            <div style={{ fontSize: '12px', color: '#787774' }}>PDF file</div>
          </div>
        </div>
      )
    },
  }
)
