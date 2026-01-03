import { useState, useEffect, useRef, useCallback } from 'react'
import { X, Send, MoreHorizontal, Check, Trash2, Reply, Edit2, CheckCircle } from 'lucide-react'
import { api, User } from '../api/client'
import { format, parseISO, formatDistanceToNow } from 'date-fns'

interface Comment {
  id: string
  page_id: string
  block_id?: string
  parent_id?: string
  content: { text: string }[]
  author_id: string
  is_resolved: boolean
  created_at: string
  updated_at: string
  author?: User
  replies?: Comment[]
}

interface CommentsPanelProps {
  pageId: string
  blockId?: string
  currentUser: User
  onClose: () => void
}

export function CommentsPanel({ pageId, blockId, currentUser, onClose }: CommentsPanelProps) {
  const [comments, setComments] = useState<Comment[]>([])
  const [loading, setLoading] = useState(true)
  const [newComment, setNewComment] = useState('')
  const [replyingTo, setReplyingTo] = useState<string | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editContent, setEditContent] = useState('')
  const [showResolved, setShowResolved] = useState(false)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  // Load comments
  useEffect(() => {
    const loadComments = async () => {
      setLoading(true)
      try {
        const endpoint = blockId
          ? `/blocks/${blockId}/comments`
          : `/pages/${pageId}/comments`
        const result = await api.get<{ comments: Comment[] }>(endpoint)
        setComments(result.comments || [])
      } catch (err) {
        console.error('Failed to load comments:', err)
        setComments([])
      } finally {
        setLoading(false)
      }
    }

    loadComments()
  }, [pageId, blockId])

  // Filter comments
  const visibleComments = showResolved
    ? comments
    : comments.filter(c => !c.is_resolved)

  // Group comments (top-level only, replies are nested)
  const topLevelComments = visibleComments.filter(c => !c.parent_id)

  // Create comment
  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newComment.trim()) return

    try {
      const result = await api.post<Comment>(`/pages/${pageId}/comments`, {
        block_id: blockId,
        parent_id: replyingTo,
        content: [{ type: 'text', text: newComment.trim() }],
      })

      if (replyingTo) {
        // Add reply to parent
        setComments(comments.map(c =>
          c.id === replyingTo
            ? { ...c, replies: [...(c.replies || []), result] }
            : c
        ))
      } else {
        setComments([result, ...comments])
      }

      setNewComment('')
      setReplyingTo(null)
    } catch (err) {
      console.error('Failed to create comment:', err)
    }
  }, [pageId, blockId, newComment, replyingTo, comments])

  // Update comment
  const handleUpdate = useCallback(async (commentId: string) => {
    if (!editContent.trim()) return

    try {
      await api.put(`/comments/${commentId}`, {
        content: [{ type: 'text', text: editContent.trim() }],
      })

      setComments(comments.map(c =>
        c.id === commentId
          ? { ...c, content: [{ text: editContent.trim() }] }
          : c
      ))

      setEditingId(null)
      setEditContent('')
    } catch (err) {
      console.error('Failed to update comment:', err)
    }
  }, [editContent, comments])

  // Delete comment
  const handleDelete = useCallback(async (commentId: string) => {
    if (!confirm('Delete this comment?')) return

    try {
      await api.delete(`/comments/${commentId}`)
      setComments(comments.filter(c => c.id !== commentId))
    } catch (err) {
      console.error('Failed to delete comment:', err)
    }
  }, [comments])

  // Resolve/unresolve comment
  const handleToggleResolve = useCallback(async (commentId: string, isResolved: boolean) => {
    try {
      const endpoint = isResolved
        ? `/comments/${commentId}/unresolve`
        : `/comments/${commentId}/resolve`
      await api.post(endpoint)

      setComments(comments.map(c =>
        c.id === commentId
          ? { ...c, is_resolved: !isResolved }
          : c
      ))
    } catch (err) {
      console.error('Failed to toggle resolve:', err)
    }
  }, [comments])

  // Start editing
  const startEdit = (comment: Comment) => {
    setEditingId(comment.id)
    setEditContent(comment.content.map(c => c.text).join(''))
  }

  // Start replying
  const startReply = (commentId: string) => {
    setReplyingTo(commentId)
    inputRef.current?.focus()
  }

  return (
    <div className="comments-panel">
      {/* Header */}
      <div className="comments-header">
        <h3>Comments</h3>
        <div className="comments-header-actions">
          <label className="toggle-resolved">
            <input
              type="checkbox"
              checked={showResolved}
              onChange={e => setShowResolved(e.target.checked)}
            />
            <span>Show resolved</span>
          </label>
          <button className="icon-btn" onClick={onClose}>
            <X size={18} />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="comments-content">
        {loading ? (
          <div className="loading-state">
            <div className="spinner" />
          </div>
        ) : topLevelComments.length === 0 ? (
          <div className="empty-state">
            <p>No comments yet</p>
            <span>Start a conversation below</span>
          </div>
        ) : (
          <div className="comments-list">
            {topLevelComments.map(comment => (
              <CommentThread
                key={comment.id}
                comment={comment}
                currentUser={currentUser}
                editingId={editingId}
                editContent={editContent}
                onEditContentChange={setEditContent}
                onStartEdit={startEdit}
                onUpdate={handleUpdate}
                onDelete={handleDelete}
                onReply={startReply}
                onToggleResolve={handleToggleResolve}
                onCancelEdit={() => setEditingId(null)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Input */}
      <div className="comments-input-wrapper">
        {replyingTo && (
          <div className="replying-to">
            <span>Replying to comment</span>
            <button onClick={() => setReplyingTo(null)}>
              <X size={14} />
            </button>
          </div>
        )}
        <form onSubmit={handleSubmit} className="comment-form">
          <textarea
            ref={inputRef}
            value={newComment}
            onChange={e => setNewComment(e.target.value)}
            placeholder="Add a comment..."
            rows={1}
            onKeyDown={e => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                handleSubmit(e)
              }
            }}
          />
          <button type="submit" disabled={!newComment.trim()}>
            <Send size={16} />
          </button>
        </form>
      </div>
    </div>
  )
}

// Comment thread component
interface CommentThreadProps {
  comment: Comment
  currentUser: User
  editingId: string | null
  editContent: string
  onEditContentChange: (content: string) => void
  onStartEdit: (comment: Comment) => void
  onUpdate: (id: string) => void
  onDelete: (id: string) => void
  onReply: (id: string) => void
  onToggleResolve: (id: string, isResolved: boolean) => void
  onCancelEdit: () => void
}

function CommentThread({
  comment,
  currentUser,
  editingId,
  editContent,
  onEditContentChange,
  onStartEdit,
  onUpdate,
  onDelete,
  onReply,
  onToggleResolve,
  onCancelEdit,
}: CommentThreadProps) {
  const [showMenu, setShowMenu] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  const isOwner = comment.author_id === currentUser.id
  const content = comment.content.map(c => c.text).join('')
  const isEditing = editingId === comment.id

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  return (
    <div className={`comment-thread ${comment.is_resolved ? 'resolved' : ''}`}>
      <div className="comment">
        <div className="comment-avatar">
          {comment.author?.avatar ? (
            <img src={comment.author.avatar} alt="" />
          ) : (
            <span>{(comment.author?.name || 'U').charAt(0).toUpperCase()}</span>
          )}
        </div>

        <div className="comment-body">
          <div className="comment-header">
            <span className="comment-author">{comment.author?.name || 'Unknown'}</span>
            <span className="comment-time">
              {formatDistanceToNow(parseISO(comment.created_at), { addSuffix: true })}
            </span>
            {comment.is_resolved && (
              <span className="resolved-badge">
                <CheckCircle size={12} />
                Resolved
              </span>
            )}
          </div>

          {isEditing ? (
            <div className="comment-edit">
              <textarea
                value={editContent}
                onChange={e => onEditContentChange(e.target.value)}
                autoFocus
              />
              <div className="comment-edit-actions">
                <button className="btn-secondary" onClick={onCancelEdit}>
                  Cancel
                </button>
                <button className="btn-primary" onClick={() => onUpdate(comment.id)}>
                  Save
                </button>
              </div>
            </div>
          ) : (
            <div className="comment-content">{content}</div>
          )}

          <div className="comment-actions">
            <button className="comment-action" onClick={() => onReply(comment.id)}>
              <Reply size={14} />
              <span>Reply</span>
            </button>
            <button
              className="comment-action"
              onClick={() => onToggleResolve(comment.id, comment.is_resolved)}
            >
              <Check size={14} />
              <span>{comment.is_resolved ? 'Unresolve' : 'Resolve'}</span>
            </button>

            {isOwner && (
              <div className="comment-menu-wrapper" ref={menuRef}>
                <button
                  className="comment-action"
                  onClick={() => setShowMenu(!showMenu)}
                >
                  <MoreHorizontal size={14} />
                </button>
                {showMenu && (
                  <div className="comment-menu">
                    <button onClick={() => { onStartEdit(comment); setShowMenu(false) }}>
                      <Edit2 size={14} />
                      <span>Edit</span>
                    </button>
                    <button className="danger" onClick={() => onDelete(comment.id)}>
                      <Trash2 size={14} />
                      <span>Delete</span>
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Replies */}
      {comment.replies && comment.replies.length > 0 && (
        <div className="comment-replies">
          {comment.replies.map(reply => (
            <CommentThread
              key={reply.id}
              comment={reply}
              currentUser={currentUser}
              editingId={editingId}
              editContent={editContent}
              onEditContentChange={onEditContentChange}
              onStartEdit={onStartEdit}
              onUpdate={onUpdate}
              onDelete={onDelete}
              onReply={onReply}
              onToggleResolve={onToggleResolve}
              onCancelEdit={onCancelEdit}
            />
          ))}
        </div>
      )}
    </div>
  )
}
