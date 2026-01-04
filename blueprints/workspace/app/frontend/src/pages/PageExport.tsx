import { useState, useCallback } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Download,
  FileText,
  FileCode,
  FileType,
  X,
  Check,
  Loader2,
  ChevronDown,
  Settings,
  Image,
  Link,
  FolderTree,
  MessageSquare,
  File,
} from 'lucide-react'
import { api } from '../api/client'

type ExportFormat = 'pdf' | 'markdown' | 'html'

interface ExportOptions {
  includeSubpages: boolean
  includeImages: boolean
  includeFiles: boolean
  createFolders: boolean
  includeComments: boolean
  pageSize: 'a4' | 'a3' | 'letter' | 'legal' | 'tabloid' | 'auto'
  orientation: 'portrait' | 'landscape'
  scale: number
}

interface PageExportProps {
  pageId: string
  pageTitle: string
  isOpen: boolean
  onClose: () => void
}

const formatOptions: Array<{
  id: ExportFormat
  label: string
  description: string
  icon: React.ReactNode
}> = [
  {
    id: 'pdf',
    label: 'PDF',
    description: 'Portable document format',
    icon: <FileText size={20} />,
  },
  {
    id: 'markdown',
    label: 'Markdown',
    description: 'MD files with CSV for databases',
    icon: <FileCode size={20} />,
  },
  {
    id: 'html',
    label: 'HTML',
    description: 'Web page with styles',
    icon: <FileType size={20} />,
  },
]

const pageSizeOptions = [
  { value: 'a4', label: 'A4' },
  { value: 'a3', label: 'A3' },
  { value: 'letter', label: 'Letter' },
  { value: 'legal', label: 'Legal' },
  { value: 'tabloid', label: 'Tabloid' },
  { value: 'auto', label: 'Auto' },
]

export function PageExport({ pageId, pageTitle, isOpen, onClose }: PageExportProps) {
  const [selectedFormat, setSelectedFormat] = useState<ExportFormat>('pdf')
  const [isExporting, setIsExporting] = useState(false)
  const [exportProgress, setExportProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [options, setOptions] = useState<ExportOptions>({
    includeSubpages: false,
    includeImages: true,
    includeFiles: true,
    createFolders: true,
    includeComments: false,
    pageSize: 'a4',
    orientation: 'portrait',
    scale: 100,
  })

  // Handle export
  const handleExport = useCallback(async () => {
    setIsExporting(true)
    setExportProgress(0)
    setError(null)
    setSuccess(false)

    try {
      // Simulate progress
      const progressInterval = setInterval(() => {
        setExportProgress((prev) => Math.min(prev + 10, 90))
      }, 200)

      // Request export from API
      const response = await api.post<{
        id: string
        download_url: string
        filename: string
        size: number
        format: string
        page_count: number
      }>(`/pages/${pageId}/export`, {
        format: selectedFormat,
        include_subpages: options.includeSubpages,
        include_images: options.includeImages,
        include_files: options.includeFiles,
        create_folders: options.createFolders,
        include_comments: options.includeComments,
        page_size: options.pageSize,
        orientation: options.orientation,
        scale: options.scale,
      })

      clearInterval(progressInterval)
      setExportProgress(100)

      // Trigger download
      if (response.download_url) {
        const link = document.createElement('a')
        link.href = response.download_url
        link.download = response.filename || `${pageTitle}.${selectedFormat}`
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
      }

      setSuccess(true)
      setTimeout(() => {
        onClose()
      }, 1500)
    } catch (err) {
      console.error('Export failed:', err)
      setError('Failed to export page. Please try again.')
    } finally {
      setIsExporting(false)
    }
  }, [pageId, pageTitle, selectedFormat, options, onClose])

  if (!isOpen) return null

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        style={{
          position: 'fixed',
          inset: 0,
          background: 'rgba(0, 0, 0, 0.4)',
          zIndex: 1000,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
        onClick={onClose}
      >
        <motion.div
          initial={{ scale: 0.95, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          exit={{ scale: 0.95, opacity: 0 }}
          transition={{ type: 'spring', damping: 25, stiffness: 300 }}
          onClick={(e) => e.stopPropagation()}
          className="export-modal"
          style={{
            width: '520px',
            maxHeight: '90vh',
            borderRadius: '12px',
            boxShadow: '0 20px 60px rgba(0, 0, 0, 0.2)',
            overflow: 'hidden',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          {/* Header */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '20px 24px',
              borderBottom: '1px solid var(--border-color)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <Download size={20} style={{ color: 'var(--accent-color)' }} />
              <span style={{ fontSize: '16px', fontWeight: 600, color: 'var(--text-primary)' }}>
                Export Page
              </span>
            </div>
            <button
              onClick={onClose}
              disabled={isExporting}
              style={{
                padding: '4px',
                background: 'none',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer',
                color: 'var(--text-tertiary)',
              }}
            >
              <X size={20} />
            </button>
          </div>

          {/* Content */}
          <div style={{ flex: 1, overflowY: 'auto', padding: '24px' }}>
            {/* Success state */}
            {success ? (
              <motion.div
                initial={{ scale: 0.9, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  padding: '48px',
                }}
              >
                <motion.div
                  initial={{ scale: 0 }}
                  animate={{ scale: 1 }}
                  transition={{ type: 'spring', damping: 10 }}
                  style={{
                    width: '64px',
                    height: '64px',
                    borderRadius: '50%',
                    background: 'var(--success-bg, #dcfce7)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    marginBottom: '16px',
                  }}
                >
                  <Check size={32} style={{ color: 'var(--success-color, #22c55e)' }} />
                </motion.div>
                <span
                  style={{
                    fontSize: '16px',
                    fontWeight: 500,
                    color: 'var(--text-primary)',
                  }}
                >
                  Export complete!
                </span>
              </motion.div>
            ) : (
              <>
                {/* Page info */}
                <div
                  style={{
                    padding: '12px 16px',
                    background: 'var(--bg-secondary)',
                    borderRadius: '8px',
                    marginBottom: '24px',
                  }}
                >
                  <p
                    style={{
                      fontSize: '14px',
                      color: 'var(--text-secondary)',
                      margin: 0,
                    }}
                  >
                    Exporting:
                  </p>
                  <p
                    style={{
                      fontSize: '15px',
                      fontWeight: 500,
                      color: 'var(--text-primary)',
                      margin: '4px 0 0',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {pageTitle}
                  </p>
                </div>

                {/* Format selection */}
                <div style={{ marginBottom: '24px' }}>
                  <label
                    style={{
                      display: 'block',
                      fontSize: '13px',
                      fontWeight: 500,
                      color: 'var(--text-secondary)',
                      marginBottom: '12px',
                    }}
                  >
                    Export Format
                  </label>
                  <div
                    style={{
                      display: 'grid',
                      gridTemplateColumns: 'repeat(3, 1fr)',
                      gap: '8px',
                    }}
                  >
                    {formatOptions.map((format) => (
                      <button
                        key={format.id}
                        onClick={() => setSelectedFormat(format.id)}
                        disabled={isExporting}
                        style={{
                          display: 'flex',
                          flexDirection: 'column',
                          alignItems: 'center',
                          gap: '8px',
                          padding: '16px 12px',
                          background: selectedFormat === format.id
                            ? 'var(--accent-bg)'
                            : 'var(--bg-secondary)',
                          border: selectedFormat === format.id
                            ? '2px solid var(--accent-color)'
                            : '2px solid transparent',
                          borderRadius: '8px',
                          cursor: 'pointer',
                          transition: 'all 0.15s',
                        }}
                      >
                        <div
                          style={{
                            color: selectedFormat === format.id
                              ? 'var(--accent-color)'
                              : 'var(--text-secondary)',
                          }}
                        >
                          {format.icon}
                        </div>
                        <span
                          style={{
                            fontSize: '13px',
                            fontWeight: 500,
                            color: selectedFormat === format.id
                              ? 'var(--accent-color)'
                              : 'var(--text-primary)',
                          }}
                        >
                          {format.label}
                        </span>
                        <span
                          style={{
                            fontSize: '11px',
                            color: 'var(--text-tertiary)',
                            textAlign: 'center',
                          }}
                        >
                          {format.description}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>

                {/* Advanced options toggle */}
                <button
                  onClick={() => setShowAdvanced(!showAdvanced)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px',
                    padding: '8px 0',
                    background: 'none',
                    border: 'none',
                    fontSize: '13px',
                    fontWeight: 500,
                    color: 'var(--text-secondary)',
                    cursor: 'pointer',
                    marginBottom: '16px',
                  }}
                >
                  <Settings size={14} />
                  Export Options
                  <ChevronDown
                    size={14}
                    style={{
                      transform: showAdvanced ? 'rotate(180deg)' : 'none',
                      transition: 'transform 0.15s',
                    }}
                  />
                </button>

                {/* Advanced options */}
                <AnimatePresence>
                  {showAdvanced && (
                    <motion.div
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: 'auto', opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      style={{ overflow: 'hidden', marginBottom: '24px' }}
                    >
                      <div
                        style={{
                          padding: '16px',
                          background: 'var(--bg-secondary)',
                          borderRadius: '8px',
                          display: 'flex',
                          flexDirection: 'column',
                          gap: '12px',
                        }}
                      >
                        {/* Include subpages */}
                        <label
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: '12px',
                            cursor: 'pointer',
                          }}
                        >
                          <input
                            type="checkbox"
                            checked={options.includeSubpages}
                            onChange={(e) =>
                              setOptions({ ...options, includeSubpages: e.target.checked })
                            }
                            disabled={isExporting}
                            style={{ width: '16px', height: '16px' }}
                          />
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <FolderTree size={14} style={{ color: 'var(--text-tertiary)' }} />
                            <div>
                              <span
                                style={{
                                  fontSize: '14px',
                                  fontWeight: 500,
                                  color: 'var(--text-primary)',
                                }}
                              >
                                Include subpages
                              </span>
                              <p
                                style={{
                                  fontSize: '12px',
                                  color: 'var(--text-tertiary)',
                                  margin: '2px 0 0',
                                }}
                              >
                                Export all nested pages as well
                              </p>
                            </div>
                          </div>
                        </label>

                        {/* Create folders for subpages */}
                        {options.includeSubpages && (
                          <label
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: '12px',
                              cursor: 'pointer',
                              paddingLeft: '28px',
                            }}
                          >
                            <input
                              type="checkbox"
                              checked={options.createFolders}
                              onChange={(e) =>
                                setOptions({ ...options, createFolders: e.target.checked })
                              }
                              disabled={isExporting}
                              style={{ width: '16px', height: '16px' }}
                            />
                            <span
                              style={{
                                fontSize: '14px',
                                color: 'var(--text-primary)',
                              }}
                            >
                              Create folders for subpages
                            </span>
                          </label>
                        )}

                        {/* Include images */}
                        <label
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: '12px',
                            cursor: 'pointer',
                          }}
                        >
                          <input
                            type="checkbox"
                            checked={options.includeImages}
                            onChange={(e) =>
                              setOptions({ ...options, includeImages: e.target.checked })
                            }
                            disabled={isExporting}
                            style={{ width: '16px', height: '16px' }}
                          />
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <Image size={14} style={{ color: 'var(--text-tertiary)' }} />
                            <span
                              style={{
                                fontSize: '14px',
                                color: 'var(--text-primary)',
                              }}
                            >
                              Include images
                            </span>
                          </div>
                        </label>

                        {/* Include files */}
                        <label
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: '12px',
                            cursor: 'pointer',
                          }}
                        >
                          <input
                            type="checkbox"
                            checked={options.includeFiles}
                            onChange={(e) =>
                              setOptions({ ...options, includeFiles: e.target.checked })
                            }
                            disabled={isExporting}
                            style={{ width: '16px', height: '16px' }}
                          />
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <File size={14} style={{ color: 'var(--text-tertiary)' }} />
                            <span
                              style={{
                                fontSize: '14px',
                                color: 'var(--text-primary)',
                              }}
                            >
                              Include files and attachments
                            </span>
                          </div>
                        </label>

                        {/* Include comments */}
                        <label
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: '12px',
                            cursor: 'pointer',
                          }}
                        >
                          <input
                            type="checkbox"
                            checked={options.includeComments}
                            onChange={(e) =>
                              setOptions({ ...options, includeComments: e.target.checked })
                            }
                            disabled={isExporting}
                            style={{ width: '16px', height: '16px' }}
                          />
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <MessageSquare size={14} style={{ color: 'var(--text-tertiary)' }} />
                            <span
                              style={{
                                fontSize: '14px',
                                color: 'var(--text-primary)',
                              }}
                            >
                              Include comments
                            </span>
                          </div>
                        </label>

                        {/* PDF-specific options */}
                        {selectedFormat === 'pdf' && (
                          <>
                            <div
                              style={{
                                height: '1px',
                                background: 'var(--border-color)',
                                margin: '4px 0',
                              }}
                            />
                            <div style={{ display: 'flex', gap: '12px' }}>
                              <div style={{ flex: 1 }}>
                                <label
                                  style={{
                                    display: 'block',
                                    fontSize: '12px',
                                    color: 'var(--text-tertiary)',
                                    marginBottom: '6px',
                                  }}
                                >
                                  Page Size
                                </label>
                                <select
                                  value={options.pageSize}
                                  onChange={(e) =>
                                    setOptions({
                                      ...options,
                                      pageSize: e.target.value as ExportOptions['pageSize'],
                                    })
                                  }
                                  disabled={isExporting}
                                  style={{
                                    width: '100%',
                                    padding: '8px 12px',
                                    background: 'var(--bg-primary)',
                                    border: '1px solid var(--border-color)',
                                    borderRadius: '6px',
                                    fontSize: '13px',
                                    color: 'var(--text-primary)',
                                  }}
                                >
                                  {pageSizeOptions.map((size) => (
                                    <option key={size.value} value={size.value}>
                                      {size.label}
                                    </option>
                                  ))}
                                </select>
                              </div>
                              <div style={{ flex: 1 }}>
                                <label
                                  style={{
                                    display: 'block',
                                    fontSize: '12px',
                                    color: 'var(--text-tertiary)',
                                    marginBottom: '6px',
                                  }}
                                >
                                  Orientation
                                </label>
                                <select
                                  value={options.orientation}
                                  onChange={(e) =>
                                    setOptions({
                                      ...options,
                                      orientation: e.target.value as ExportOptions['orientation'],
                                    })
                                  }
                                  disabled={isExporting}
                                  style={{
                                    width: '100%',
                                    padding: '8px 12px',
                                    background: 'var(--bg-primary)',
                                    border: '1px solid var(--border-color)',
                                    borderRadius: '6px',
                                    fontSize: '13px',
                                    color: 'var(--text-primary)',
                                  }}
                                >
                                  <option value="portrait">Portrait</option>
                                  <option value="landscape">Landscape</option>
                                </select>
                              </div>
                            </div>
                            <div>
                              <label
                                style={{
                                  display: 'block',
                                  fontSize: '12px',
                                  color: 'var(--text-tertiary)',
                                  marginBottom: '6px',
                                }}
                              >
                                Scale: {options.scale}%
                              </label>
                              <input
                                type="range"
                                min="50"
                                max="200"
                                step="10"
                                value={options.scale}
                                onChange={(e) =>
                                  setOptions({
                                    ...options,
                                    scale: parseInt(e.target.value),
                                  })
                                }
                                disabled={isExporting}
                                style={{
                                  width: '100%',
                                }}
                              />
                            </div>
                          </>
                        )}
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>

                {/* Error message */}
                {error && (
                  <div
                    style={{
                      padding: '12px 16px',
                      background: 'var(--danger-bg)',
                      borderRadius: '8px',
                      marginBottom: '16px',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '8px',
                    }}
                  >
                    <span style={{ fontSize: '13px', color: 'var(--danger-color)' }}>
                      {error}
                    </span>
                  </div>
                )}

                {/* Progress bar */}
                {isExporting && (
                  <div
                    style={{
                      marginBottom: '16px',
                    }}
                  >
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        marginBottom: '8px',
                      }}
                    >
                      <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
                        Exporting...
                      </span>
                      <span style={{ fontSize: '13px', color: 'var(--text-tertiary)' }}>
                        {exportProgress}%
                      </span>
                    </div>
                    <div
                      style={{
                        height: '4px',
                        background: 'var(--bg-secondary)',
                        borderRadius: '2px',
                        overflow: 'hidden',
                      }}
                    >
                      <motion.div
                        initial={{ width: 0 }}
                        animate={{ width: `${exportProgress}%` }}
                        style={{
                          height: '100%',
                          background: 'var(--accent-color)',
                          borderRadius: '2px',
                        }}
                      />
                    </div>
                  </div>
                )}
              </>
            )}
          </div>

          {/* Footer */}
          {!success && (
            <div
              style={{
                display: 'flex',
                justifyContent: 'flex-end',
                gap: '12px',
                padding: '16px 24px',
                borderTop: '1px solid var(--border-color)',
              }}
            >
              <button
                onClick={onClose}
                disabled={isExporting}
                style={{
                  padding: '10px 20px',
                  background: 'var(--bg-secondary)',
                  border: 'none',
                  borderRadius: '6px',
                  fontSize: '14px',
                  fontWeight: 500,
                  color: 'var(--text-primary)',
                  cursor: 'pointer',
                }}
              >
                Cancel
              </button>
              <button
                onClick={handleExport}
                disabled={isExporting}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  padding: '10px 20px',
                  background: 'var(--accent-color)',
                  border: 'none',
                  borderRadius: '6px',
                  fontSize: '14px',
                  fontWeight: 500,
                  color: 'white',
                  cursor: isExporting ? 'wait' : 'pointer',
                  opacity: isExporting ? 0.7 : 1,
                }}
              >
                {isExporting ? (
                  <>
                    <Loader2 size={16} style={{ animation: 'spin 1s linear infinite' }} />
                    Exporting...
                  </>
                ) : (
                  <>
                    <Download size={16} />
                    Export
                  </>
                )}
              </button>
            </div>
          )}

          {/* CSS for animations */}
          <style>{`
            @keyframes spin {
              from { transform: rotate(0deg); }
              to { transform: rotate(360deg); }
            }
          `}</style>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
