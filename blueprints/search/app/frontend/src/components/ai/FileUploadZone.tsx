import { useState, useRef, useCallback } from 'react'
import { Upload, X, FileText, Image as ImageIcon } from 'lucide-react'

export interface UploadedFile {
  id: string
  file: File
  type: 'image' | 'pdf'
  preview?: string
}

interface FileUploadZoneProps {
  files: UploadedFile[]
  onFilesChange: (files: UploadedFile[]) => void
  maxFiles?: number
  maxSizeMB?: number
  accept?: string[]
  children: React.ReactNode
}

const DEFAULT_ACCEPT = ['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'application/pdf']
const DEFAULT_MAX_SIZE_MB = 10
const DEFAULT_MAX_FILES = 5

export function FileUploadZone({
  files,
  onFilesChange,
  maxFiles = DEFAULT_MAX_FILES,
  maxSizeMB = DEFAULT_MAX_SIZE_MB,
  accept = DEFAULT_ACCEPT,
  children,
}: FileUploadZoneProps) {
  const [isDragging, setIsDragging] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const validateFile = useCallback(
    (file: File): string | null => {
      if (!accept.includes(file.type)) {
        return `File type ${file.type} is not supported`
      }
      if (file.size > maxSizeMB * 1024 * 1024) {
        return `File size exceeds ${maxSizeMB}MB limit`
      }
      return null
    },
    [accept, maxSizeMB]
  )

  const processFile = useCallback(async (file: File): Promise<UploadedFile | null> => {
    const error = validateFile(file)
    if (error) {
      setError(error)
      return null
    }

    const isImage = file.type.startsWith('image/')
    const type = isImage ? 'image' : 'pdf'

    let preview: string | undefined
    if (isImage) {
      preview = await new Promise<string>((resolve) => {
        const reader = new FileReader()
        reader.onload = () => resolve(reader.result as string)
        reader.readAsDataURL(file)
      })
    }

    return {
      id: crypto.randomUUID(),
      file,
      type,
      preview,
    }
  }, [validateFile])

  const handleFiles = useCallback(
    async (fileList: FileList | File[]) => {
      setError(null)
      const newFiles: UploadedFile[] = []
      const remainingSlots = maxFiles - files.length

      for (let i = 0; i < Math.min(fileList.length, remainingSlots); i++) {
        const file = fileList instanceof FileList ? fileList[i] : fileList[i]
        const processed = await processFile(file)
        if (processed) {
          newFiles.push(processed)
        }
      }

      if (fileList.length > remainingSlots) {
        setError(`Maximum ${maxFiles} files allowed`)
      }

      if (newFiles.length > 0) {
        onFilesChange([...files, ...newFiles])
      }
    },
    [files, maxFiles, onFilesChange, processFile]
  )

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(true)
  }, [])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)
  }, [])

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setIsDragging(false)
      handleFiles(e.dataTransfer.files)
    },
    [handleFiles]
  )

  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (e.target.files) {
        handleFiles(e.target.files)
      }
      // Reset input so same file can be selected again
      e.target.value = ''
    },
    [handleFiles]
  )

  const removeFile = useCallback(
    (id: string) => {
      onFilesChange(files.filter((f) => f.id !== id))
    },
    [files, onFilesChange]
  )

  const openFilePicker = useCallback(() => {
    inputRef.current?.click()
  }, [])

  return (
    <div
      className={`file-upload-zone ${isDragging ? 'dragging' : ''}`}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <input
        ref={inputRef}
        type="file"
        accept={accept.join(',')}
        multiple={maxFiles > 1}
        onChange={handleInputChange}
        className="file-upload-input"
      />

      {/* Drag overlay */}
      {isDragging && (
        <div className="file-upload-overlay">
          <Upload size={32} />
          <span>Drop files here</span>
        </div>
      )}

      {/* Attached files preview */}
      {files.length > 0 && (
        <div className="file-upload-preview">
          {files.map((f) => (
            <div key={f.id} className="file-preview-item">
              {f.type === 'image' && f.preview ? (
                <img src={f.preview} alt={f.file.name} className="file-preview-image" />
              ) : (
                <div className="file-preview-icon">
                  <FileText size={24} />
                </div>
              )}
              <span className="file-preview-name">{f.file.name}</span>
              <button
                type="button"
                onClick={() => removeFile(f.id)}
                className="file-preview-remove"
                aria-label="Remove file"
              >
                <X size={14} />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Error message */}
      {error && <div className="file-upload-error">{error}</div>}

      {/* Upload button (if no children or as additional action) */}
      <div className="file-upload-actions">
        <button
          type="button"
          onClick={openFilePicker}
          className="file-upload-button"
          title="Upload images or PDFs"
          disabled={files.length >= maxFiles}
        >
          <ImageIcon size={18} />
        </button>
      </div>

      {/* Children (the actual input area) */}
      {children}
    </div>
  )
}
