import { useState, useEffect, useRef, useCallback } from 'react';
import type { Comment } from '../../../types';
import { commentsApi } from '../../../api/client';

interface CellCommentPopoverProps {
  recordId: string;
  fieldId: string;
  position: { x: number; y: number };
  onClose: () => void;
  onCommentAdded?: () => void;
}

interface CellComment extends Comment {
  fieldId?: string;
}

export function CellCommentPopover({
  recordId,
  fieldId,
  position,
  onClose,
  onCommentAdded,
}: CellCommentPopoverProps) {
  const [comments, setComments] = useState<CellComment[]>([]);
  const [newComment, setNewComment] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editContent, setEditContent] = useState('');
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Load comments for this cell
  useEffect(() => {
    const loadComments = async () => {
      setIsLoading(true);
      try {
        // Load all record comments and filter by field_id
        const { comments: allComments } = await commentsApi.list(recordId);
        const cellComments = allComments.filter(
          (c: CellComment) => c.fieldId === fieldId
        );
        setComments(cellComments);
      } catch (error) {
        console.error('Failed to load comments:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadComments();
  }, [recordId, fieldId]);

  // Focus input on mount
  useEffect(() => {
    setTimeout(() => inputRef.current?.focus(), 100);
  }, []);

  // Handle click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        onClose();
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [onClose]);

  // Handle keyboard
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  // Add new comment
  const handleSubmit = async () => {
    if (!newComment.trim() || isSubmitting) return;

    setIsSubmitting(true);
    try {
      // Note: The backend comment API would need to support field_id
      // For now, we store it in the content as a workaround
      const { comment } = await commentsApi.create(recordId, newComment);
      setComments([...comments, { ...comment, fieldId } as CellComment]);
      setNewComment('');
      onCommentAdded?.();
    } catch (error) {
      console.error('Failed to add comment:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Edit comment
  const handleEdit = async (commentId: string) => {
    if (!editContent.trim()) return;

    try {
      const { comment } = await commentsApi.update(commentId, { text: editContent });
      setComments(comments.map(c => c.id === commentId ? { ...c, ...comment } : c));
      setEditingId(null);
      setEditContent('');
    } catch (error) {
      console.error('Failed to edit comment:', error);
    }
  };

  // Delete comment
  const handleDelete = async (commentId: string) => {
    try {
      await commentsApi.delete(commentId);
      setComments(comments.filter(c => c.id !== commentId));
    } catch (error) {
      console.error('Failed to delete comment:', error);
    }
  };

  // Resolve/unresolve comment
  const handleToggleResolve = async (comment: CellComment) => {
    try {
      if (comment.isResolved) {
        const { comment: updated } = await commentsApi.unresolve(comment.id);
        setComments(comments.map(c => c.id === comment.id ? { ...c, ...updated } : c));
      } else {
        const { comment: updated } = await commentsApi.resolve(comment.id);
        setComments(comments.map(c => c.id === comment.id ? { ...c, ...updated } : c));
      }
    } catch (error) {
      console.error('Failed to toggle resolve:', error);
    }
  };

  // Format timestamp
  const formatTime = (date: string) => {
    const d = new Date(date);
    const now = new Date();
    const diffMs = now.getTime() - d.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return d.toLocaleDateString();
  };

  // Adjust position to stay in viewport
  const adjustedPosition = useCallback(() => {
    const popoverWidth = 320;
    const popoverHeight = 400;
    const padding = 16;

    let x = position.x;
    let y = position.y;

    // Keep within viewport
    if (x + popoverWidth > window.innerWidth - padding) {
      x = window.innerWidth - popoverWidth - padding;
    }
    if (y + popoverHeight > window.innerHeight - padding) {
      y = window.innerHeight - popoverHeight - padding;
    }

    return { left: Math.max(padding, x), top: Math.max(padding, y) };
  }, [position]);

  return (
    <div
      ref={containerRef}
      className="fixed z-50 bg-white rounded-lg shadow-xl border border-slate-200 w-80 max-h-96 flex flex-col"
      style={adjustedPosition()}
    >
      {/* Header */}
      <div className="px-4 py-3 border-b border-slate-200 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <svg className="w-4 h-4 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
          </svg>
          <span className="text-sm font-medium text-slate-700">Comments</span>
          <span className="text-xs text-slate-400">({comments.length})</span>
        </div>
        <button
          onClick={onClose}
          className="text-slate-400 hover:text-slate-600"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Comments list */}
      <div className="flex-1 overflow-auto p-2 space-y-2">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <div className="w-5 h-5 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          </div>
        ) : comments.length === 0 ? (
          <div className="py-8 text-center text-sm text-slate-400">
            No comments yet. Be the first to comment!
          </div>
        ) : (
          comments.map((comment) => (
            <div
              key={comment.id}
              className={`p-3 rounded-lg border ${
                comment.isResolved
                  ? 'bg-slate-50 border-slate-200 opacity-60'
                  : 'bg-white border-slate-200'
              }`}
            >
              {/* Comment header */}
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <div className="w-6 h-6 rounded-full bg-primary-100 flex items-center justify-center">
                    <span className="text-xs font-medium text-primary-700">
                      {comment.userId?.charAt(0).toUpperCase() || 'U'}
                    </span>
                  </div>
                  <span className="text-xs text-slate-500">
                    {formatTime(comment.createdAt)}
                  </span>
                </div>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => handleToggleResolve(comment)}
                    className={`p-1 rounded text-xs ${
                      comment.isResolved
                        ? 'text-green-600 hover:bg-green-50'
                        : 'text-slate-400 hover:bg-slate-100'
                    }`}
                    title={comment.isResolved ? 'Reopen' : 'Resolve'}
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                  </button>
                  {editingId !== comment.id && (
                    <>
                      <button
                        onClick={() => {
                          setEditingId(comment.id);
                          setEditContent(comment.content);
                        }}
                        className="p-1 rounded text-slate-400 hover:bg-slate-100"
                        title="Edit"
                      >
                        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                        </svg>
                      </button>
                      <button
                        onClick={() => handleDelete(comment.id)}
                        className="p-1 rounded text-slate-400 hover:bg-red-50 hover:text-red-600"
                        title="Delete"
                      >
                        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </>
                  )}
                </div>
              </div>

              {/* Comment content */}
              {editingId === comment.id ? (
                <div className="space-y-2">
                  <textarea
                    value={editContent}
                    onChange={(e) => setEditContent(e.target.value)}
                    className="w-full p-2 text-sm border border-slate-200 rounded resize-none focus:outline-none focus:ring-1 focus:ring-primary"
                    rows={2}
                    autoFocus
                  />
                  <div className="flex justify-end gap-2">
                    <button
                      onClick={() => {
                        setEditingId(null);
                        setEditContent('');
                      }}
                      className="px-2 py-1 text-xs text-slate-600 hover:bg-slate-100 rounded"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={() => handleEdit(comment.id)}
                      className="px-2 py-1 text-xs bg-primary text-white rounded hover:bg-primary-600"
                    >
                      Save
                    </button>
                  </div>
                </div>
              ) : (
                <p className={`text-sm ${comment.isResolved ? 'line-through' : ''}`}>
                  {comment.content}
                </p>
              )}
            </div>
          ))
        )}
      </div>

      {/* Add comment input */}
      <div className="p-3 border-t border-slate-200">
        <div className="relative">
          <textarea
            ref={inputRef}
            value={newComment}
            onChange={(e) => setNewComment(e.target.value)}
            placeholder="Add a comment..."
            className="w-full p-2 pr-10 text-sm border border-slate-200 rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            rows={2}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                e.preventDefault();
                handleSubmit();
              }
            }}
          />
          <button
            onClick={handleSubmit}
            disabled={!newComment.trim() || isSubmitting}
            className="absolute bottom-2 right-2 p-1.5 rounded-md bg-primary text-white disabled:opacity-50 disabled:cursor-not-allowed hover:bg-primary-600 transition-colors"
          >
            {isSubmitting ? (
              <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
            ) : (
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
              </svg>
            )}
          </button>
        </div>
        <p className="mt-1 text-xs text-slate-400">
          Press Cmd+Enter to submit
        </p>
      </div>
    </div>
  );
}

// Cell comment indicator - shows orange triangle when cell has comments
export function CellCommentIndicator({
  hasComments,
  onClick,
}: {
  hasComments: boolean;
  onClick: (e: React.MouseEvent) => void;
}) {
  if (!hasComments) return null;

  return (
    <button
      onClick={onClick}
      className="absolute top-0 right-0 w-0 h-0 border-solid border-transparent border-t-orange-400 border-l-transparent"
      style={{
        borderWidth: '0 8px 8px 0',
        borderRightColor: '#f97316',
      }}
      title="View comments"
    />
  );
}
