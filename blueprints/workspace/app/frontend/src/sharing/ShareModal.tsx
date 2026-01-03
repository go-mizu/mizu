import { useState, useEffect, useRef, useCallback } from 'react'
import { createPortal } from 'react-dom'
import {
  X, Link2, Copy, Check, Globe, Lock, Users,
  ChevronDown, Trash2, Mail, Clock
} from 'lucide-react'
import { api, User } from '../api/client'

interface Share {
  id: string
  page_id: string
  type: 'user' | 'link' | 'domain' | 'public'
  permission: 'read' | 'comment' | 'edit' | 'full_access'
  user_id?: string
  token?: string
  expires_at?: string
  domain?: string
  user?: User
}

interface ShareModalProps {
  pageId: string
  pageTitle: string
  onClose: () => void
}

const PERMISSION_LABELS = {
  read: 'Can view',
  comment: 'Can comment',
  edit: 'Can edit',
  full_access: 'Full access',
}

const PERMISSION_DESCRIPTIONS = {
  read: 'View only',
  comment: 'View and comment',
  edit: 'View, comment, and edit',
  full_access: 'Full access including sharing',
}

export function ShareModal({ pageId, pageTitle, onClose }: ShareModalProps) {
  const [shares, setShares] = useState<Share[]>([])
  const [loading, setLoading] = useState(true)
  const [email, setEmail] = useState('')
  const [permission, setPermission] = useState<Share['permission']>('read')
  const [isPublic, setIsPublic] = useState(false)
  const [shareLink, setShareLink] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [showPermissionMenu, setShowPermissionMenu] = useState<string | null>(null)
  const modalRef = useRef<HTMLDivElement>(null)

  // Load shares
  useEffect(() => {
    const loadShares = async () => {
      setLoading(true)
      try {
        const result = await api.get<{ shares: Share[] }>(`/pages/${pageId}/shares`)
        setShares(result.shares || [])

        // Check if public
        const publicShare = result.shares?.find(s => s.type === 'public')
        setIsPublic(!!publicShare)

        // Get share link
        const linkShare = result.shares?.find(s => s.type === 'link')
        if (linkShare?.token) {
          setShareLink(`${window.location.origin}/share/${linkShare.token}`)
        }
      } catch (err) {
        console.error('Failed to load shares:', err)
      } finally {
        setLoading(false)
      }
    }

    loadShares()
  }, [pageId])

  // Close on escape
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  // Invite by email
  const handleInvite = useCallback(async (e: React.FormEvent) => {
    e.preventDefault()
    if (!email.trim()) return

    try {
      const result = await api.post<Share>(`/pages/${pageId}/shares`, {
        type: 'user',
        email: email.trim(),
        permission,
      })
      setShares([...shares, result])
      setEmail('')
    } catch (err) {
      console.error('Failed to invite:', err)
    }
  }, [pageId, email, permission, shares])

  // Update permission
  const handleUpdatePermission = useCallback(async (shareId: string, newPermission: Share['permission']) => {
    try {
      await api.put(`/shares/${shareId}`, { permission: newPermission })
      setShares(shares.map(s =>
        s.id === shareId ? { ...s, permission: newPermission } : s
      ))
      setShowPermissionMenu(null)
    } catch (err) {
      console.error('Failed to update permission:', err)
    }
  }, [shares])

  // Remove share
  const handleRemove = useCallback(async (shareId: string) => {
    try {
      await api.delete(`/shares/${shareId}`)
      setShares(shares.filter(s => s.id !== shareId))
    } catch (err) {
      console.error('Failed to remove share:', err)
    }
  }, [shares])

  // Toggle public
  const handleTogglePublic = useCallback(async () => {
    try {
      if (isPublic) {
        // Disable public
        const publicShare = shares.find(s => s.type === 'public')
        if (publicShare) {
          await api.delete(`/shares/${publicShare.id}`)
          setShares(shares.filter(s => s.type !== 'public'))
        }
        setIsPublic(false)
      } else {
        // Enable public
        const result = await api.post<Share>(`/pages/${pageId}/shares`, {
          type: 'public',
          permission: 'read',
        })
        setShares([...shares, result])
        setIsPublic(true)
      }
    } catch (err) {
      console.error('Failed to toggle public:', err)
    }
  }, [pageId, isPublic, shares])

  // Generate/regenerate share link
  const handleGenerateLink = useCallback(async () => {
    try {
      const result = await api.post<{ token: string }>(`/pages/${pageId}/share-link`, {
        permission: 'read',
      })
      const link = `${window.location.origin}/share/${result.token}`
      setShareLink(link)

      // Update shares list
      const linkShare = shares.find(s => s.type === 'link')
      if (linkShare) {
        setShares(shares.map(s =>
          s.id === linkShare.id ? { ...s, token: result.token } : s
        ))
      } else {
        setShares([...shares, { id: result.token, page_id: pageId, type: 'link', permission: 'read', token: result.token }])
      }
    } catch (err) {
      console.error('Failed to generate link:', err)
    }
  }, [pageId, shares])

  // Copy link
  const handleCopyLink = useCallback(async () => {
    if (!shareLink) {
      await handleGenerateLink()
    }

    if (shareLink) {
      await navigator.clipboard.writeText(shareLink)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }, [shareLink, handleGenerateLink])

  // User shares only
  const userShares = shares.filter(s => s.type === 'user')

  return createPortal(
    <div className="modal-overlay" onClick={onClose}>
      <div
        ref={modalRef}
        className="share-modal"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="share-header">
          <h2>Share "{pageTitle}"</h2>
          <button className="icon-btn" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        {/* Content */}
        <div className="share-content">
          {/* Invite form */}
          <form onSubmit={handleInvite} className="invite-form">
            <div className="invite-input-wrapper">
              <Mail size={16} />
              <input
                type="email"
                placeholder="Add people by email..."
                value={email}
                onChange={e => setEmail(e.target.value)}
              />
            </div>
            <div className="invite-permission">
              <select
                value={permission}
                onChange={e => setPermission(e.target.value as Share['permission'])}
              >
                {Object.entries(PERMISSION_LABELS).map(([key, label]) => (
                  <option key={key} value={key}>{label}</option>
                ))}
              </select>
            </div>
            <button type="submit" className="btn-primary" disabled={!email.trim()}>
              Invite
            </button>
          </form>

          {/* Shared with */}
          {userShares.length > 0 && (
            <div className="shared-list">
              <h4>Shared with</h4>
              {userShares.map(share => (
                <div key={share.id} className="share-item">
                  <div className="share-user">
                    <div className="share-avatar">
                      {share.user?.avatar ? (
                        <img src={share.user.avatar} alt="" />
                      ) : (
                        <span>{(share.user?.name || 'U').charAt(0).toUpperCase()}</span>
                      )}
                    </div>
                    <div className="share-user-info">
                      <span className="share-user-name">{share.user?.name || 'Unknown'}</span>
                      <span className="share-user-email">{share.user?.email}</span>
                    </div>
                  </div>
                  <div className="share-actions">
                    <div className="permission-dropdown">
                      <button
                        className="permission-btn"
                        onClick={() => setShowPermissionMenu(
                          showPermissionMenu === share.id ? null : share.id
                        )}
                      >
                        <span>{PERMISSION_LABELS[share.permission]}</span>
                        <ChevronDown size={14} />
                      </button>
                      {showPermissionMenu === share.id && (
                        <div className="permission-menu">
                          {Object.entries(PERMISSION_LABELS).map(([key, label]) => (
                            <button
                              key={key}
                              className={share.permission === key ? 'active' : ''}
                              onClick={() => handleUpdatePermission(share.id, key as Share['permission'])}
                            >
                              <span className="permission-label">{label}</span>
                              <span className="permission-desc">
                                {PERMISSION_DESCRIPTIONS[key as keyof typeof PERMISSION_DESCRIPTIONS]}
                              </span>
                            </button>
                          ))}
                          <hr />
                          <button className="danger" onClick={() => handleRemove(share.id)}>
                            <Trash2 size={14} />
                            <span>Remove access</span>
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          <hr className="share-divider" />

          {/* Link sharing */}
          <div className="share-link-section">
            <div className="share-option">
              <div className="share-option-icon">
                <Link2 size={18} />
              </div>
              <div className="share-option-info">
                <span className="share-option-title">Share link</span>
                <span className="share-option-desc">
                  Anyone with the link can view
                </span>
              </div>
              <button
                className={`btn-secondary ${copied ? 'copied' : ''}`}
                onClick={handleCopyLink}
              >
                {copied ? <Check size={14} /> : <Copy size={14} />}
                <span>{copied ? 'Copied!' : 'Copy link'}</span>
              </button>
            </div>
          </div>

          {/* Public access */}
          <div className="share-public-section">
            <div className="share-option">
              <div className="share-option-icon">
                {isPublic ? <Globe size={18} /> : <Lock size={18} />}
              </div>
              <div className="share-option-info">
                <span className="share-option-title">
                  {isPublic ? 'Published to web' : 'Publish to web'}
                </span>
                <span className="share-option-desc">
                  {isPublic
                    ? 'Anyone on the internet can view'
                    : 'Make this page visible to everyone'}
                </span>
              </div>
              <label className="toggle-switch">
                <input
                  type="checkbox"
                  checked={isPublic}
                  onChange={handleTogglePublic}
                />
                <span className="toggle-slider" />
              </label>
            </div>
          </div>
        </div>
      </div>
    </div>,
    document.body
  )
}
