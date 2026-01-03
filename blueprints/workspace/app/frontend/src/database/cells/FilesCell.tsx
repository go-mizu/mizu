import * as React from 'react'
import { useState, useRef, useCallback } from 'react'
import {
  CustomCell,
  CustomRenderer,
  GridCellKind,
  getMiddleCenterBias,
  measureTextCached,
} from '@glideapps/glide-data-grid'
import { api, FileAttachment } from '../../api/client'

// Files cell data structure
export interface FilesCellData {
  kind: 'files-cell'
  files: FileAttachment[]
}

export type FilesCellType = CustomCell<FilesCellData>

// File type detection utilities
const IMAGE_EXTENSIONS = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico', 'tiff', 'tif']
const VIDEO_EXTENSIONS = ['mp4', 'webm', 'mov', 'avi', 'mkv', 'wmv', 'flv']
const AUDIO_EXTENSIONS = ['mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a', 'wma']
const DOCUMENT_EXTENSIONS = ['pdf', 'doc', 'docx', 'txt', 'rtf', 'odt']
const SPREADSHEET_EXTENSIONS = ['xls', 'xlsx', 'csv', 'ods']
const PRESENTATION_EXTENSIONS = ['ppt', 'pptx', 'odp']
const ARCHIVE_EXTENSIONS = ['zip', 'rar', '7z', 'tar', 'gz', 'bz2']
const CODE_EXTENSIONS = ['js', 'ts', 'tsx', 'jsx', 'py', 'go', 'rs', 'java', 'c', 'cpp', 'h', 'css', 'html', 'json', 'xml', 'yaml', 'yml', 'md']

function getExtension(filename: string): string {
  const parts = filename.split('.')
  return parts.length > 1 ? parts.pop()?.toLowerCase() || '' : ''
}

function getMimeTypeFromExtension(ext: string): string {
  const extMap: Record<string, string> = {
    // Images
    jpg: 'image/jpeg', jpeg: 'image/jpeg', png: 'image/png',
    gif: 'image/gif', webp: 'image/webp', svg: 'image/svg+xml',
    bmp: 'image/bmp', ico: 'image/x-icon', tiff: 'image/tiff', tif: 'image/tiff',
    // Videos
    mp4: 'video/mp4', webm: 'video/webm', mov: 'video/quicktime',
    avi: 'video/x-msvideo', mkv: 'video/x-matroska',
    // Audio
    mp3: 'audio/mpeg', wav: 'audio/wav', ogg: 'audio/ogg',
    flac: 'audio/flac', aac: 'audio/aac', m4a: 'audio/mp4',
    // Documents
    pdf: 'application/pdf',
    doc: 'application/msword',
    docx: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    xls: 'application/vnd.ms-excel',
    xlsx: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    ppt: 'application/vnd.ms-powerpoint',
    pptx: 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
    txt: 'text/plain', csv: 'text/csv', json: 'application/json',
    // Archives
    zip: 'application/zip', rar: 'application/x-rar-compressed', '7z': 'application/x-7z-compressed',
  }
  return extMap[ext] || 'application/octet-stream'
}

function detectFileType(file: FileAttachment): string {
  // If type is already set and valid, use it
  if (file.type && file.type !== 'application/octet-stream') {
    return file.type
  }
  // Otherwise, detect from filename extension
  const ext = getExtension(file.name)
  return getMimeTypeFromExtension(ext)
}

// Helper to check if a file is an image
function isImageFile(file: FileAttachment): boolean {
  const type = detectFileType(file)
  if (type.startsWith('image/')) return true
  const ext = getExtension(file.name)
  return IMAGE_EXTENSIONS.includes(ext)
}

function isVideoFile(file: FileAttachment): boolean {
  const type = detectFileType(file)
  if (type.startsWith('video/')) return true
  const ext = getExtension(file.name)
  return VIDEO_EXTENSIONS.includes(ext)
}

function isAudioFile(file: FileAttachment): boolean {
  const type = detectFileType(file)
  if (type.startsWith('audio/')) return true
  const ext = getExtension(file.name)
  return AUDIO_EXTENSIONS.includes(ext)
}

function getFileCategory(file: FileAttachment): 'image' | 'video' | 'audio' | 'document' | 'spreadsheet' | 'presentation' | 'archive' | 'code' | 'other' {
  if (isImageFile(file)) return 'image'
  if (isVideoFile(file)) return 'video'
  if (isAudioFile(file)) return 'audio'

  const ext = getExtension(file.name)
  if (DOCUMENT_EXTENSIONS.includes(ext)) return 'document'
  if (SPREADSHEET_EXTENSIONS.includes(ext)) return 'spreadsheet'
  if (PRESENTATION_EXTENSIONS.includes(ext)) return 'presentation'
  if (ARCHIVE_EXTENSIONS.includes(ext)) return 'archive'
  if (CODE_EXTENSIONS.includes(ext)) return 'code'

  return 'other'
}

// Thumbnail size in pixels
const THUMB_SIZE = 20
const THUMB_GAP = 4

// Custom renderer for files cell
export const filesCellRenderer: CustomRenderer<FilesCellType> = {
  kind: GridCellKind.Custom,
  isMatch: (cell: CustomCell): cell is FilesCellType =>
    (cell.data as FilesCellData).kind === 'files-cell',
  draw: (args, cell) => {
    const { ctx, rect, theme, imageLoader, col, row } = args
    const { files } = cell.data

    // Clear background
    ctx.fillStyle = theme.bgCell
    ctx.fillRect(rect.x, rect.y, rect.width, rect.height)

    if (!files || files.length === 0) {
      // Draw placeholder
      ctx.fillStyle = theme.textLight
      ctx.font = theme.baseFontStyle
      const text = 'Add files...'
      const y = rect.y + rect.height / 2 + getMiddleCenterBias(ctx, theme)
      ctx.fillText(text, rect.x + theme.cellHorizontalPadding, y)
      return true
    }

    // Draw file thumbnails/icons
    let x = rect.x + theme.cellHorizontalPadding
    const y = rect.y + (rect.height - THUMB_SIZE) / 2

    for (let i = 0; i < files.length && x < rect.x + rect.width - 30; i++) {
      const file = files[i]
      const category = getFileCategory(file)

      if (category === 'image' && file.url) {
        // Draw image thumbnail
        const img = imageLoader?.loadOrGetImage(file.thumbnailUrl || file.url, col, row)
        if (img) {
          ctx.save()
          ctx.beginPath()
          ctx.roundRect(x, y, THUMB_SIZE, THUMB_SIZE, 3)
          ctx.clip()
          ctx.drawImage(img, x, y, THUMB_SIZE, THUMB_SIZE)
          ctx.restore()

          ctx.strokeStyle = theme.borderColor
          ctx.lineWidth = 1
          ctx.beginPath()
          ctx.roundRect(x, y, THUMB_SIZE, THUMB_SIZE, 3)
          ctx.stroke()
        } else {
          // Loading placeholder with image icon
          ctx.fillStyle = theme.bgCellMedium
          ctx.beginPath()
          ctx.roundRect(x, y, THUMB_SIZE, THUMB_SIZE, 3)
          ctx.fill()
          // Draw small image icon
          ctx.fillStyle = theme.textLight
          ctx.font = '11px sans-serif'
          ctx.fillText('🖼', x + 3, y + THUMB_SIZE - 4)
        }
      } else {
        // Draw file type indicator box with category-based colors
        const categoryColors: Record<string, string> = {
          video: '#e74c3c',
          audio: '#9b59b6',
          document: '#3498db',
          spreadsheet: '#27ae60',
          presentation: '#e67e22',
          archive: '#7f8c8d',
          code: '#2c3e50',
          other: theme.bgCellMedium,
        }
        ctx.fillStyle = categoryColors[category] || theme.bgCellMedium
        ctx.beginPath()
        ctx.roundRect(x, y, THUMB_SIZE, THUMB_SIZE, 3)
        ctx.fill()

        // Draw file extension or icon
        const ext = getExtension(file.name).toUpperCase().slice(0, 3)
        if (ext) {
          ctx.fillStyle = category === 'other' ? theme.textMedium : '#ffffff'
          ctx.font = 'bold 8px sans-serif'
          const extWidth = ctx.measureText(ext).width
          ctx.fillText(ext, x + (THUMB_SIZE - extWidth) / 2, y + THUMB_SIZE / 2 + 3)
        } else {
          // Generic file icon
          ctx.fillStyle = category === 'other' ? theme.textMedium : '#ffffff'
          ctx.font = '11px sans-serif'
          ctx.fillText('📄', x + 3, y + THUMB_SIZE - 4)
        }
      }

      x += THUMB_SIZE + THUMB_GAP
    }

    // Show remaining count if more files
    if (files.length > 0) {
      const visibleCount = Math.floor((rect.width - theme.cellHorizontalPadding * 2 - 20) / (THUMB_SIZE + THUMB_GAP))
      const remainingCount = files.length - visibleCount
      if (remainingCount > 0) {
        ctx.fillStyle = theme.textMedium
        ctx.font = theme.baseFontStyle
        ctx.fillText(`+${remainingCount}`, x, rect.y + rect.height / 2 + getMiddleCenterBias(ctx, theme))
      }
    }

    return true
  },
  // Editor uses the curried function pattern: () => (props) => JSX
  provideEditor: () => (p) => {
    const { value, onChange, onFinishedEditing } = p
    return React.createElement(FilesCellEditor, {
      value,
      onChange,
      onFinishedEditing,
    })
  },
  onPaste: (v, d) => {
    // Support pasting file URLs
    if (v.startsWith('http://') || v.startsWith('https://')) {
      const filename = v.split('/').pop() || 'file'
      return {
        ...d,
        files: [...(d.files || []), {
          id: `paste-${Date.now()}`,
          name: filename,
          url: v,
          type: 'application/octet-stream',
        }],
      }
    }
    return undefined
  },
}

// Editor component props
interface FilesCellEditorProps {
  value: FilesCellType
  onChange: (newValue: FilesCellType) => void
  onFinishedEditing: (newValue?: FilesCellType) => void
}

// Editor component for files cell
function FilesCellEditor({ value, onChange, onFinishedEditing }: FilesCellEditorProps) {
  const [files, setFiles] = useState<FileAttachment[]>(value.data.files || [])
  const [isUploading, setIsUploading] = useState(false)
  const [showUrlInput, setShowUrlInput] = useState(false)
  const [urlInput, setUrlInput] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const updateFiles = useCallback((newFiles: FileAttachment[]) => {
    setFiles(newFiles)
    onChange({
      ...value,
      data: {
        ...value.data,
        files: newFiles,
      },
    })
  }, [value, onChange])

  const handleFileUpload = useCallback(async (fileList: FileList | null) => {
    if (!fileList || fileList.length === 0) return

    setIsUploading(true)
    const newFiles: FileAttachment[] = [...files]

    for (let i = 0; i < fileList.length; i++) {
      const file = fileList[i]
      try {
        const result = await api.upload(file)
        newFiles.push({
          id: result.id,
          name: result.filename,
          url: result.url,
          type: result.type,
          size: file.size,
        })
      } catch (err) {
        console.error('Failed to upload file:', err)
        alert(`Failed to upload ${file.name}`)
      }
    }

    setIsUploading(false)
    updateFiles(newFiles)
  }, [files, updateFiles])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragOver(false)
    handleFileUpload(e.dataTransfer.files)
  }, [handleFileUpload])

  const handleRemoveFile = useCallback((fileId: string) => {
    const newFiles = files.filter(f => f.id !== fileId)
    updateFiles(newFiles)
  }, [files, updateFiles])

  const handleAddUrl = useCallback(() => {
    if (!urlInput.trim()) return

    const urlParts = urlInput.split('/')
    const filename = urlParts[urlParts.length - 1] || 'External file'
    const ext = filename.split('.').pop()?.toLowerCase() || ''

    const typeMap: Record<string, string> = {
      jpg: 'image/jpeg', jpeg: 'image/jpeg', png: 'image/png',
      gif: 'image/gif', webp: 'image/webp', svg: 'image/svg+xml',
      pdf: 'application/pdf', mp4: 'video/mp4', mp3: 'audio/mpeg',
    }

    const newFile: FileAttachment = {
      id: `url-${Date.now()}`,
      name: filename,
      url: urlInput,
      type: typeMap[ext] || 'application/octet-stream',
    }

    updateFiles([...files, newFile])
    setUrlInput('')
    setShowUrlInput(false)
  }, [urlInput, files, updateFiles])

  const formatFileSize = (bytes?: number): string => {
    if (!bytes) return ''
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  const getFileIcon = (file: FileAttachment): string => {
    const category = getFileCategory(file)
    switch (category) {
      case 'image': return '🖼️'
      case 'video': return '🎬'
      case 'audio': return '🎵'
      case 'document': return '📄'
      case 'spreadsheet': return '📊'
      case 'presentation': return '📽️'
      case 'archive': return '🗜️'
      case 'code': return '💻'
      default: return '📎'
    }
  }

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        minWidth: 280,
        maxWidth: 400,
        background: '#ffffff',
        fontFamily: 'ui-sans-serif, -apple-system, BlinkMacSystemFont, sans-serif',
      }}
      onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
      onDragLeave={() => setDragOver(false)}
      onDrop={handleDrop}
    >
      {/* Files list */}
      {files.length > 0 && (
        <div style={{ maxHeight: 200, overflowY: 'auto', padding: 8 }}>
          {files.map((file) => (
            <div
              key={file.id}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '6px 8px',
                borderRadius: 4,
                marginBottom: 4,
                background: 'rgba(55,53,47,0.04)',
              }}
            >
              {isImageFile(file) && file.url ? (
                <img
                  src={file.thumbnailUrl || file.url}
                  alt={file.name}
                  style={{ width: 32, height: 32, objectFit: 'cover', borderRadius: 4 }}
                />
              ) : (
                <span style={{ fontSize: 20 }}>{getFileIcon(file)}</span>
              )}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{
                  fontSize: 13,
                  color: '#37352f',
                  whiteSpace: 'nowrap',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                }}>
                  <a href={file.url} target="_blank" rel="noopener noreferrer" style={{ color: 'inherit', textDecoration: 'none' }}>
                    {file.name}
                  </a>
                </div>
                {file.size && (
                  <div style={{ fontSize: 11, color: '#787774' }}>{formatFileSize(file.size)}</div>
                )}
              </div>
              <button
                type="button"
                onClick={() => handleRemoveFile(file.id)}
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  padding: 4,
                  color: '#787774',
                  fontSize: 16,
                }}
              >
                ×
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Drop zone / Empty state */}
      {(files.length === 0 || dragOver) && (
        <div style={{
          padding: 16,
          textAlign: 'center',
          color: '#787774',
          background: dragOver ? 'rgba(35,131,226,0.1)' : undefined,
          border: dragOver ? '2px dashed #2383e2' : '2px dashed transparent',
          borderRadius: 4,
          margin: 8,
          transition: 'all 0.15s',
        }}>
          <div style={{ fontSize: 24, marginBottom: 4 }}>📁</div>
          <div style={{ fontSize: 13 }}>
            {isUploading ? 'Uploading...' : 'Drop files here or click Upload'}
          </div>
        </div>
      )}

      {/* URL input */}
      {showUrlInput && (
        <div style={{ padding: '8px 12px', borderTop: '1px solid rgba(55,53,47,0.09)' }}>
          <input
            type="url"
            placeholder="Paste file URL..."
            value={urlInput}
            onChange={(e) => setUrlInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleAddUrl()
              if (e.key === 'Escape') setShowUrlInput(false)
              e.stopPropagation()
            }}
            autoFocus
            style={{
              width: '100%',
              padding: '6px 8px',
              border: '1px solid rgba(55,53,47,0.16)',
              borderRadius: 4,
              fontSize: 13,
              outline: 'none',
              boxSizing: 'border-box',
            }}
          />
          <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
            <button
              type="button"
              onClick={handleAddUrl}
              style={{
                flex: 1,
                padding: '6px 12px',
                background: '#2383e2',
                color: 'white',
                border: 'none',
                borderRadius: 4,
                fontSize: 13,
                cursor: 'pointer',
              }}
            >
              Add
            </button>
            <button
              type="button"
              onClick={() => setShowUrlInput(false)}
              style={{
                padding: '6px 12px',
                background: 'rgba(55,53,47,0.08)',
                border: 'none',
                borderRadius: 4,
                fontSize: 13,
                cursor: 'pointer',
              }}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Actions */}
      <div style={{
        display: 'flex',
        borderTop: '1px solid rgba(55,53,47,0.09)',
      }}>
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          disabled={isUploading}
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 6,
            padding: 10,
            background: 'none',
            border: 'none',
            borderRight: '1px solid rgba(55,53,47,0.09)',
            cursor: isUploading ? 'wait' : 'pointer',
            fontSize: 13,
            color: '#37352f',
          }}
        >
          <span>📤</span>
          <span>{isUploading ? 'Uploading...' : 'Upload'}</span>
        </button>
        <button
          type="button"
          onClick={() => setShowUrlInput(!showUrlInput)}
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 6,
            padding: 10,
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            fontSize: 13,
            color: '#37352f',
          }}
        >
          <span>🔗</span>
          <span>Embed link</span>
        </button>
      </div>

      {/* Hidden file input */}
      <input
        ref={fileInputRef}
        type="file"
        multiple
        onChange={(e) => handleFileUpload(e.target.files)}
        style={{ display: 'none' }}
        accept="image/*,video/*,audio/*,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.csv,.json,.zip,.rar,.7z"
      />
    </div>
  )
}

// Helper to create a files cell
export function createFilesCell(files: FileAttachment[]): FilesCellType {
  return {
    kind: GridCellKind.Custom,
    allowOverlay: true,
    copyData: files.map(f => f.name).join(', '),
    data: {
      kind: 'files-cell',
      files: files || [],
    },
  }
}
