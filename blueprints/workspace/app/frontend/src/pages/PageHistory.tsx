import { useState, useEffect, useCallback } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  History,
  Clock,
  User,
  ChevronRight,
  RotateCcw,
  Eye,
  X,
  AlertCircle,
  Loader2,
  Calendar,
} from 'lucide-react'
import { api } from '../api/client'

interface PageVersion {
  id: string
  createdAt: string
  createdBy: {
    id: string
    name: string
    email: string
    avatarUrl?: string
  }
  summary: string
  blockCount: number
}

interface PageHistoryProps {
  pageId: string
  isOpen: boolean
  onClose: () => void
  onRestore: (versionId: string) => void
  onPreview: (versionId: string) => void
}

export function PageHistory({
  pageId,
  isOpen,
  onClose,
  onRestore,
  onPreview,
}: PageHistoryProps) {
  const [versions, setVersions] = useState<PageVersion[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedVersion, setSelectedVersion] = useState<string | null>(null)
  const [isRestoring, setIsRestoring] = useState(false)

  // Fetch page versions
  const fetchVersions = useCallback(async () => {
    if (!pageId) return

    setIsLoading(true)
    setError(null)

    try {
      const response = await api.get<{ versions: PageVersion[] }>(
        `/pages/${pageId}/versions`
      )
      setVersions(response.versions || [])
    } catch (err) {
      console.error('Failed to fetch page versions:', err)
      setError('Failed to load version history')
    } finally {
      setIsLoading(false)
    }
  }, [pageId])

  useEffect(() => {
    if (isOpen) {
      fetchVersions()
    }
  }, [isOpen, fetchVersions])

  // Handle restore
  const handleRestore = useCallback(async (versionId: string) => {
    setIsRestoring(true)
    try {
      await api.post(`/pages/${pageId}/versions/${versionId}/restore`)
      onRestore(versionId)
      onClose()
    } catch (err) {
      console.error('Failed to restore version:', err)
      setError('Failed to restore version')
    } finally {
      setIsRestoring(false)
    }
  }, [pageId, onRestore, onClose])

  // Format relative time
  const formatRelativeTime = (dateString: string): string => {
    const date = new Date(dateString)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMs / 3600000)
    const diffDays = Math.floor(diffMs / 86400000)

    if (diffMins < 1) return 'Just now'
    if (diffMins < 60) return `${diffMins} minutes ago`
    if (diffHours < 24) return `${diffHours} hours ago`
    if (diffDays < 7) return `${diffDays} days ago`
    return date.toLocaleDateString()
  }

  // Group versions by date
  const groupedVersions = versions.reduce((groups, version) => {
    const date = new Date(version.createdAt).toDateString()
    if (!groups[date]) {
      groups[date] = []
    }
    groups[date].push(version)
    return groups
  }, {} as Record<string, PageVersion[]>)

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
          justifyContent: 'flex-end',
        }}
        onClick={onClose}
      >
        <motion.div
          initial={{ x: '100%' }}
          animate={{ x: 0 }}
          exit={{ x: '100%' }}
          transition={{ type: 'spring', damping: 25, stiffness: 300 }}
          onClick={(e) => e.stopPropagation()}
          style={{
            width: '400px',
            height: '100%',
            background: 'var(--bg-primary)',
            borderLeft: '1px solid var(--border-color)',
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          }}
        >
          {/* Header */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '16px 20px',
              borderBottom: '1px solid var(--border-color)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
              <History size={20} style={{ color: 'var(--text-secondary)' }} />
              <span style={{ fontSize: '16px', fontWeight: 600, color: 'var(--text-primary)' }}>
                Page History
              </span>
            </div>
            <button
              onClick={onClose}
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
          <div style={{ flex: 1, overflowY: 'auto', padding: '16px 0' }}>
            {isLoading ? (
              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  padding: '48px',
                  color: 'var(--text-tertiary)',
                }}
              >
                <Loader2
                  size={24}
                  style={{ animation: 'spin 1s linear infinite', marginBottom: '12px' }}
                />
                <span style={{ fontSize: '14px' }}>Loading history...</span>
              </div>
            ) : error ? (
              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  padding: '48px',
                  color: 'var(--danger-color)',
                }}
              >
                <AlertCircle size={24} style={{ marginBottom: '12px' }} />
                <span style={{ fontSize: '14px' }}>{error}</span>
                <button
                  onClick={fetchVersions}
                  style={{
                    marginTop: '12px',
                    padding: '6px 12px',
                    background: 'var(--bg-secondary)',
                    border: '1px solid var(--border-color)',
                    borderRadius: '4px',
                    fontSize: '13px',
                    cursor: 'pointer',
                  }}
                >
                  Retry
                </button>
              </div>
            ) : versions.length === 0 ? (
              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  padding: '48px',
                  color: 'var(--text-tertiary)',
                }}
              >
                <Clock size={32} style={{ marginBottom: '12px', opacity: 0.5 }} />
                <span style={{ fontSize: '14px', fontWeight: 500, marginBottom: '4px' }}>
                  No version history
                </span>
                <span style={{ fontSize: '13px', textAlign: 'center' }}>
                  Page versions will appear here as you make changes.
                </span>
              </div>
            ) : (
              Object.entries(groupedVersions).map(([date, dateVersions]) => (
                <div key={date} style={{ marginBottom: '16px' }}>
                  {/* Date header */}
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '8px',
                      padding: '8px 20px',
                      fontSize: '12px',
                      fontWeight: 500,
                      color: 'var(--text-tertiary)',
                      textTransform: 'uppercase',
                      letterSpacing: '0.5px',
                    }}
                  >
                    <Calendar size={12} />
                    {date === new Date().toDateString()
                      ? 'Today'
                      : date === new Date(Date.now() - 86400000).toDateString()
                      ? 'Yesterday'
                      : new Date(date).toLocaleDateString(undefined, {
                          weekday: 'short',
                          month: 'short',
                          day: 'numeric',
                        })}
                  </div>

                  {/* Versions for this date */}
                  {dateVersions.map((version) => (
                    <button
                      key={version.id}
                      onClick={() => setSelectedVersion(
                        selectedVersion === version.id ? null : version.id
                      )}
                      style={{
                        width: '100%',
                        padding: '12px 20px',
                        background: selectedVersion === version.id
                          ? 'var(--accent-bg)'
                          : 'none',
                        border: 'none',
                        textAlign: 'left',
                        cursor: 'pointer',
                        transition: 'background 0.1s',
                      }}
                      onMouseEnter={(e) => {
                        if (selectedVersion !== version.id) {
                          e.currentTarget.style.background = 'var(--bg-hover)'
                        }
                      }}
                      onMouseLeave={(e) => {
                        if (selectedVersion !== version.id) {
                          e.currentTarget.style.background = 'none'
                        }
                      }}
                    >
                      <div
                        style={{
                          display: 'flex',
                          alignItems: 'flex-start',
                          gap: '12px',
                        }}
                      >
                        {/* User avatar */}
                        <div
                          style={{
                            width: '32px',
                            height: '32px',
                            borderRadius: '50%',
                            background: version.createdBy.avatarUrl
                              ? `url(${version.createdBy.avatarUrl}) center/cover`
                              : 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontSize: '13px',
                            fontWeight: 600,
                            color: 'white',
                            flexShrink: 0,
                          }}
                        >
                          {!version.createdBy.avatarUrl &&
                            version.createdBy.name.charAt(0).toUpperCase()}
                        </div>

                        <div style={{ flex: 1, minWidth: 0 }}>
                          {/* Author and time */}
                          <div
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: '8px',
                              marginBottom: '4px',
                            }}
                          >
                            <span
                              style={{
                                fontSize: '14px',
                                fontWeight: 500,
                                color: 'var(--text-primary)',
                              }}
                            >
                              {version.createdBy.name}
                            </span>
                            <span
                              style={{
                                fontSize: '12px',
                                color: 'var(--text-tertiary)',
                              }}
                            >
                              {formatRelativeTime(version.createdAt)}
                            </span>
                          </div>

                          {/* Summary */}
                          <p
                            style={{
                              fontSize: '13px',
                              color: 'var(--text-secondary)',
                              margin: 0,
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap',
                            }}
                          >
                            {version.summary || `${version.blockCount} blocks`}
                          </p>
                        </div>

                        <ChevronRight
                          size={16}
                          style={{
                            color: 'var(--text-tertiary)',
                            transform: selectedVersion === version.id ? 'rotate(90deg)' : 'none',
                            transition: 'transform 0.15s',
                          }}
                        />
                      </div>

                      {/* Expanded actions */}
                      <AnimatePresence>
                        {selectedVersion === version.id && (
                          <motion.div
                            initial={{ height: 0, opacity: 0 }}
                            animate={{ height: 'auto', opacity: 1 }}
                            exit={{ height: 0, opacity: 0 }}
                            style={{
                              overflow: 'hidden',
                              marginTop: '12px',
                              display: 'flex',
                              gap: '8px',
                            }}
                          >
                            <button
                              onClick={(e) => {
                                e.stopPropagation()
                                onPreview(version.id)
                              }}
                              style={{
                                flex: 1,
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                gap: '6px',
                                padding: '8px 12px',
                                background: 'var(--bg-primary)',
                                border: '1px solid var(--border-color)',
                                borderRadius: '6px',
                                fontSize: '13px',
                                fontWeight: 500,
                                color: 'var(--text-secondary)',
                                cursor: 'pointer',
                              }}
                            >
                              <Eye size={14} />
                              Preview
                            </button>
                            <button
                              onClick={(e) => {
                                e.stopPropagation()
                                handleRestore(version.id)
                              }}
                              disabled={isRestoring}
                              style={{
                                flex: 1,
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                gap: '6px',
                                padding: '8px 12px',
                                background: 'var(--accent-color)',
                                border: 'none',
                                borderRadius: '6px',
                                fontSize: '13px',
                                fontWeight: 500,
                                color: 'white',
                                cursor: isRestoring ? 'wait' : 'pointer',
                                opacity: isRestoring ? 0.7 : 1,
                              }}
                            >
                              {isRestoring ? (
                                <Loader2 size={14} style={{ animation: 'spin 1s linear infinite' }} />
                              ) : (
                                <RotateCcw size={14} />
                              )}
                              Restore
                            </button>
                          </motion.div>
                        )}
                      </AnimatePresence>
                    </button>
                  ))}
                </div>
              ))
            )}
          </div>

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
