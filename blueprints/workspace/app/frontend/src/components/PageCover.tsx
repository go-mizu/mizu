import { useState, useRef, useCallback } from 'react'
import { Image, Upload, Link, X, Move } from 'lucide-react'
import { api } from '../api/client'

interface PageCoverProps {
  pageId: string
  coverUrl?: string
  coverPosition?: number // 0-100, vertical position
  onChange?: (coverUrl: string | null, position?: number) => void
}

// Predefined gradient backgrounds
const GRADIENTS = [
  { id: 'gradient-1', value: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)' },
  { id: 'gradient-2', value: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)' },
  { id: 'gradient-3', value: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)' },
  { id: 'gradient-4', value: 'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)' },
  { id: 'gradient-5', value: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)' },
  { id: 'gradient-6', value: 'linear-gradient(135deg, #a8edea 0%, #fed6e3 100%)' },
  { id: 'gradient-7', value: 'linear-gradient(135deg, #d299c2 0%, #fef9d7 100%)' },
  { id: 'gradient-8', value: 'linear-gradient(135deg, #89f7fe 0%, #66a6ff 100%)' },
]

// Predefined solid colors
const SOLID_COLORS = [
  { id: 'red', value: '#e03e3e' },
  { id: 'orange', value: '#d9730d' },
  { id: 'yellow', value: '#cb912f' },
  { id: 'green', value: '#448361' },
  { id: 'blue', value: '#337ea9' },
  { id: 'purple', value: '#9065b0' },
  { id: 'pink', value: '#c14c8a' },
  { id: 'gray', value: '#787774' },
]

export function PageCover({ pageId, coverUrl, coverPosition = 50, onChange }: PageCoverProps) {
  const [showPicker, setShowPicker] = useState(false)
  const [isRepositioning, setIsRepositioning] = useState(false)
  const [position, setPosition] = useState(coverPosition)
  const [isUploading, setIsUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const coverRef = useRef<HTMLDivElement>(null)

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setIsUploading(true)
    try {
      // For now, use object URL - in production, upload to server
      const url = URL.createObjectURL(file)
      onChange?.(url, position)
      setShowPicker(false)
    } catch (err) {
      console.error('Failed to upload cover:', err)
    } finally {
      setIsUploading(false)
    }
  }

  const handleUrlInput = () => {
    const url = prompt('Enter image URL:')
    if (url) {
      onChange?.(url, position)
      setShowPicker(false)
    }
  }

  const handleGradientSelect = (gradient: string) => {
    onChange?.(gradient, position)
    setShowPicker(false)
  }

  const handleColorSelect = (color: string) => {
    onChange?.(color, position)
    setShowPicker(false)
  }

  const handleRemoveCover = async () => {
    try {
      await api.delete(`/pages/${pageId}/cover`)
      onChange?.(null)
      setShowPicker(false)
    } catch (err) {
      console.error('Failed to remove cover:', err)
    }
  }

  const handleRepositionStart = () => {
    setIsRepositioning(true)
  }

  const handleRepositionEnd = useCallback(() => {
    setIsRepositioning(false)
    onChange?.(coverUrl || '', position)
  }, [coverUrl, position, onChange])

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (!isRepositioning || !coverRef.current) return

      const rect = coverRef.current.getBoundingClientRect()
      const y = e.clientY - rect.top
      const newPosition = Math.max(0, Math.min(100, (y / rect.height) * 100))
      setPosition(newPosition)
    },
    [isRepositioning]
  )

  // Determine if cover is a URL, gradient, or color
  const isGradient = coverUrl?.startsWith('linear-gradient')
  const isColor = coverUrl?.startsWith('#')
  const isImage = coverUrl && !isGradient && !isColor

  const coverStyle = {
    ...(isGradient && { background: coverUrl }),
    ...(isColor && { backgroundColor: coverUrl }),
    ...(isImage && {
      backgroundImage: `url(${coverUrl})`,
      backgroundPosition: `center ${position}%`,
      backgroundSize: 'cover',
    }),
  }

  if (!coverUrl) {
    return (
      <div className="page-cover-wrapper empty">
        <button
          className="add-cover-btn"
          onClick={() => setShowPicker(true)}
        >
          <Image size={16} />
          <span>Add cover</span>
        </button>

        {showPicker && (
          <>
            <div className="cover-picker-overlay" onClick={() => setShowPicker(false)} />
            <div className="cover-picker">
              <CoverPickerContent
                onFileSelect={() => fileInputRef.current?.click()}
                onUrlInput={handleUrlInput}
                onGradientSelect={handleGradientSelect}
                onColorSelect={handleColorSelect}
              />
            </div>
          </>
        )}

        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          onChange={handleFileUpload}
          style={{ display: 'none' }}
        />
      </div>
    )
  }

  return (
    <div
      ref={coverRef}
      className={`page-cover-wrapper ${isRepositioning ? 'repositioning' : ''}`}
      onMouseMove={handleMouseMove}
      onMouseUp={handleRepositionEnd}
      onMouseLeave={handleRepositionEnd}
    >
      <div className="page-cover" style={coverStyle}>
        {isRepositioning && (
          <div className="reposition-overlay">
            <span>Drag image to reposition</span>
          </div>
        )}
      </div>

      <div className="cover-actions">
        <button className="cover-action-btn" onClick={() => setShowPicker(true)}>
          <Image size={14} />
          <span>Change cover</span>
        </button>
        {isImage && (
          <button className="cover-action-btn" onMouseDown={handleRepositionStart}>
            <Move size={14} />
            <span>Reposition</span>
          </button>
        )}
      </div>

      {showPicker && (
        <>
          <div className="cover-picker-overlay" onClick={() => setShowPicker(false)} />
          <div className="cover-picker">
            <CoverPickerContent
              onFileSelect={() => fileInputRef.current?.click()}
              onUrlInput={handleUrlInput}
              onGradientSelect={handleGradientSelect}
              onColorSelect={handleColorSelect}
              onRemove={handleRemoveCover}
              hasExisting={!!coverUrl}
            />
          </div>
        </>
      )}

      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        onChange={handleFileUpload}
        style={{ display: 'none' }}
      />

      {isUploading && (
        <div className="cover-uploading">
          <div className="spinner" />
          <span>Uploading...</span>
        </div>
      )}
    </div>
  )
}

function CoverPickerContent({
  onFileSelect,
  onUrlInput,
  onGradientSelect,
  onColorSelect,
  onRemove,
  hasExisting = false,
}: {
  onFileSelect: () => void
  onUrlInput: () => void
  onGradientSelect: (gradient: string) => void
  onColorSelect: (color: string) => void
  onRemove?: () => void
  hasExisting?: boolean
}) {
  return (
    <div className="cover-picker-content">
      <div className="picker-section">
        <div className="picker-header">Upload</div>
        <div className="upload-options">
          <button className="upload-btn" onClick={onFileSelect}>
            <Upload size={16} />
            <span>Upload image</span>
          </button>
          <button className="upload-btn" onClick={onUrlInput}>
            <Link size={16} />
            <span>Link to image</span>
          </button>
        </div>
      </div>

      <div className="picker-section">
        <div className="picker-header">Gradients</div>
        <div className="gradient-grid">
          {GRADIENTS.map((g) => (
            <button
              key={g.id}
              className="gradient-option"
              style={{ background: g.value }}
              onClick={() => onGradientSelect(g.value)}
            />
          ))}
        </div>
      </div>

      <div className="picker-section">
        <div className="picker-header">Solid colors</div>
        <div className="color-grid">
          {SOLID_COLORS.map((c) => (
            <button
              key={c.id}
              className="color-option"
              style={{ backgroundColor: c.value }}
              onClick={() => onColorSelect(c.value)}
            />
          ))}
        </div>
      </div>

      {hasExisting && onRemove && (
        <div className="picker-section">
          <button className="remove-cover-btn" onClick={onRemove}>
            <X size={14} />
            <span>Remove cover</span>
          </button>
        </div>
      )}
    </div>
  )
}
