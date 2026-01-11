import { useState, useEffect } from 'react';
import { useBaseStore } from '../../stores/baseStore';
import type { TableRecord, Field, CellValue } from '../../types';

interface RecordModalProps {
  record: TableRecord;
  onClose: () => void;
}

export function RecordModal({ record, onClose }: RecordModalProps) {
  const { fields, updateCellValue, deleteRecord, comments, fetchComments, createComment } = useBaseStore();
  const [activeTab, setActiveTab] = useState<'fields' | 'comments' | 'activity'>('fields');
  const [newComment, setNewComment] = useState('');

  useEffect(() => {
    fetchComments(record.id);
  }, [record.id, fetchComments]);

  const handleFieldChange = async (fieldId: string, value: CellValue) => {
    await updateCellValue(record.id, fieldId, value);
  };

  const handleDelete = async () => {
    if (window.confirm('Delete this record? This cannot be undone.')) {
      await deleteRecord(record.id);
      onClose();
    }
  };

  const handleAddComment = async () => {
    if (!newComment.trim()) return;
    await createComment(record.id, newComment.trim());
    setNewComment('');
  };

  const renderFieldEditor = (field: Field) => {
    const value = record.values[field.id];

    switch (field.type) {
      case 'text':
      case 'single_line_text':
      case 'email':
      case 'url':
      case 'phone':
        return (
          <input
            type={field.type === 'email' ? 'email' : field.type === 'url' ? 'url' : field.type === 'phone' ? 'tel' : 'text'}
            value={(value as string) || ''}
            onChange={(e) => handleFieldChange(field.id, e.target.value || null)}
            className="input"
            placeholder={field.type === 'email' ? 'email@example.com' : field.type === 'url' ? 'https://' : field.type === 'phone' ? '+1 (555) 000-0000' : ''}
          />
        );

      case 'long_text':
      case 'rich_text':
        return (
          <textarea
            value={(value as string) || ''}
            onChange={(e) => handleFieldChange(field.id, e.target.value || null)}
            className="input min-h-[100px]"
            rows={4}
            placeholder={field.type === 'rich_text' ? 'Supports markdown formatting...' : ''}
          />
        );

      case 'number':
      case 'currency':
      case 'percent':
        return (
          <input
            type="number"
            value={(value as number) ?? ''}
            onChange={(e) => handleFieldChange(field.id, e.target.value ? parseFloat(e.target.value) : null)}
            className="input"
          />
        );

      case 'checkbox':
        return (
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={Boolean(value)}
              onChange={(e) => handleFieldChange(field.id, e.target.checked)}
              className="w-5 h-5 rounded border-gray-300 text-primary focus:ring-primary"
            />
            <span className="text-sm text-gray-600">Yes</span>
          </label>
        );

      case 'date':
      case 'datetime':
        return (
          <input
            type={field.type === 'datetime' ? 'datetime-local' : 'date'}
            value={(value as string) || ''}
            onChange={(e) => handleFieldChange(field.id, e.target.value || null)}
            className="input"
          />
        );

      case 'single_select':
        const options = field.options?.choices || [];
        return (
          <select
            value={(value as string) || ''}
            onChange={(e) => handleFieldChange(field.id, e.target.value || null)}
            className="input"
          >
            <option value="">Select...</option>
            {options.map((opt: { id: string; name: string; color: string }) => (
              <option key={opt.id} value={opt.id}>{opt.name}</option>
            ))}
          </select>
        );

      case 'multi_select':
        const multiOptions = field.options?.choices || [];
        const selectedValues = (value as string[]) || [];
        return (
          <div className="space-y-2">
            {multiOptions.map((opt: { id: string; name: string; color: string }) => (
              <label key={opt.id} className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={selectedValues.includes(opt.id)}
                  onChange={(e) => {
                    const newValues = e.target.checked
                      ? [...selectedValues, opt.id]
                      : selectedValues.filter(v => v !== opt.id);
                    handleFieldChange(field.id, newValues.length > 0 ? newValues : null);
                  }}
                  className="w-4 h-4 rounded border-gray-300"
                />
                <span
                  className="px-2 py-0.5 rounded-full text-sm"
                  style={{ backgroundColor: opt.color + '20', color: opt.color }}
                >
                  {opt.name}
                </span>
              </label>
            ))}
          </div>
        );

      case 'rating':
        const maxRating = field.options?.max || 5;
        const currentRating = (value as number) || 0;
        return (
          <div className="flex gap-1">
            {Array.from({ length: maxRating }, (_, i) => (
              <button
                key={i}
                type="button"
                onClick={() => handleFieldChange(field.id, i + 1 === currentRating ? null : i + 1)}
                className="text-yellow-400 hover:scale-110 transition-transform"
              >
                <svg className="w-6 h-6" fill={i < currentRating ? 'currentColor' : 'none'} stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                </svg>
              </button>
            ))}
          </div>
        );

      case 'duration':
        const durationSeconds = (value as number) || 0;
        const hours = Math.floor(durationSeconds / 3600);
        const minutes = Math.floor((durationSeconds % 3600) / 60);
        return (
          <div className="flex gap-3">
            <div className="flex-1">
              <input
                type="number"
                min="0"
                value={hours || ''}
                onChange={(e) => {
                  const h = parseInt(e.target.value) || 0;
                  handleFieldChange(field.id, h * 3600 + minutes * 60);
                }}
                className="input"
                placeholder="0"
              />
              <span className="text-xs text-gray-500 mt-1 block">Hours</span>
            </div>
            <div className="flex-1">
              <input
                type="number"
                min="0"
                max="59"
                value={minutes || ''}
                onChange={(e) => {
                  const m = parseInt(e.target.value) || 0;
                  handleFieldChange(field.id, hours * 3600 + m * 60);
                }}
                className="input"
                placeholder="0"
              />
              <span className="text-xs text-gray-500 mt-1 block">Minutes</span>
            </div>
          </div>
        );

      case 'barcode':
        return (
          <div className="space-y-2">
            <input
              type="text"
              value={(value as string) || ''}
              onChange={(e) => handleFieldChange(field.id, e.target.value || null)}
              className="input font-mono"
              placeholder="Enter barcode value..."
            />
            {value && (
              <div className="flex items-center gap-2 p-2 bg-slate-50 rounded-lg">
                <svg className="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h2M4 12h2m10 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
                </svg>
                <span className="font-mono text-sm">{value as string}</span>
              </div>
            )}
          </div>
        );

      case 'button':
        const buttonUrl = field.options?.url;
        const buttonLabel = field.options?.label || field.name || 'Click';
        const buttonColor = field.options?.color || '#2563eb';
        return (
          <div className="py-2">
            <button
              type="button"
              onClick={() => buttonUrl && window.open(buttonUrl, '_blank', 'noopener,noreferrer')}
              className="px-4 py-2 rounded-lg text-white font-medium transition-opacity hover:opacity-80"
              style={{ backgroundColor: buttonColor }}
            >
              {buttonLabel}
            </button>
            {buttonUrl && (
              <p className="text-xs text-gray-500 mt-2">Opens: {buttonUrl}</p>
            )}
          </div>
        );

      case 'attachment':
        const attachments = (value as { id?: string; filename?: string; url: string; mime_type?: string; size?: number }[]) || [];
        return (
          <div className="space-y-3">
            {attachments.length > 0 && (
              <div className="grid grid-cols-3 gap-2">
                {attachments.map((att, idx) => (
                  <div key={att.id || idx} className="relative group">
                    {att.mime_type?.startsWith('image/') || att.url?.match(/\.(jpg|jpeg|png|gif|webp)/i) ? (
                      <img
                        src={att.url}
                        alt={att.filename || 'attachment'}
                        className="w-full h-24 object-cover rounded-lg border border-slate-200"
                      />
                    ) : (
                      <div className="w-full h-24 bg-slate-100 rounded-lg border border-slate-200 flex flex-col items-center justify-center">
                        <svg className="w-8 h-8 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                        </svg>
                        <span className="text-xs text-slate-500 mt-1 truncate max-w-full px-1">{att.filename}</span>
                      </div>
                    )}
                    <button
                      onClick={() => {
                        const remaining = attachments.filter((_, i) => i !== idx);
                        handleFieldChange(field.id, remaining.length > 0 ? remaining as unknown as CellValue : null);
                      }}
                      className="absolute -top-1 -right-1 w-5 h-5 bg-red-500 text-white rounded-full opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-xs"
                    >
                      ×
                    </button>
                  </div>
                ))}
              </div>
            )}
            <div className="text-sm text-gray-500 italic">
              {attachments.length === 0 ? 'No attachments' : `${attachments.length} file(s)`}
            </div>
          </div>
        );

      case 'formula':
      case 'rollup':
      case 'count':
      case 'lookup':
      case 'autonumber':
      case 'created_time':
      case 'last_modified_time':
      case 'created_by':
      case 'last_modified_by':
        return (
          <div className="text-sm text-gray-500 italic py-2">
            {value !== null && value !== undefined ? String(value) : '—'}
          </div>
        );

      default:
        return (
          <input
            type="text"
            value={value !== null && value !== undefined ? String(value) : ''}
            onChange={(e) => handleFieldChange(field.id, e.target.value || null)}
            className="input"
          />
        );
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal-content max-w-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-header">
          <h3 className="text-lg font-semibold">Record details</h3>
          <div className="flex items-center gap-2">
            <button
              onClick={handleDelete}
              className="p-2 text-gray-400 hover:text-red-500 transition-colors rounded-md hover:bg-red-50"
              title="Delete record"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
            <button onClick={onClose} className="text-gray-400 hover:text-gray-600 rounded-md hover:bg-slate-50 p-1">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="border-b border-slate-200 px-6">
          <div className="flex gap-6">
            <button
              onClick={() => setActiveTab('fields')}
              className={`py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === 'fields'
                  ? 'border-primary text-primary'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              Fields
            </button>
            <button
              onClick={() => setActiveTab('comments')}
              className={`py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === 'comments'
                  ? 'border-primary text-primary'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              Comments ({comments.length})
            </button>
            <button
              onClick={() => setActiveTab('activity')}
              className={`py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === 'activity'
                  ? 'border-primary text-primary'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              Activity
            </button>
          </div>
        </div>

        <div className="modal-body max-h-[60vh] overflow-y-auto">
          {activeTab === 'fields' && (
            <div className="space-y-4">
              {fields.map((field) => (
                <div key={field.id}>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    {field.name}
                  </label>
                  {renderFieldEditor(field)}
                </div>
              ))}
            </div>
          )}

          {activeTab === 'comments' && (
            <div className="space-y-4">
              {/* Add comment */}
              <div className="flex gap-2">
                <textarea
                  value={newComment}
                  onChange={(e) => setNewComment(e.target.value)}
                  placeholder="Add a comment..."
                  className="input flex-1"
                  rows={2}
                />
                <button
                  onClick={handleAddComment}
                  disabled={!newComment.trim()}
                  className="btn btn-primary self-end"
                >
                  Post
                </button>
              </div>

              {/* Comments list */}
              <div className="space-y-3">
                {comments.map((comment) => (
                  <div key={comment.id} className="bg-slate-50 rounded-lg p-3 border border-slate-100">
                    <div className="flex items-center gap-2 mb-2">
                      <div className="w-6 h-6 rounded-full bg-primary-100 flex items-center justify-center text-xs text-primary font-medium">
                        {comment.user?.name?.charAt(0).toUpperCase() || 'U'}
                      </div>
                      <span className="text-sm font-medium text-gray-900">{comment.user?.name || 'Unknown'}</span>
                      <span className="text-xs text-gray-500">
                        {new Date(comment.createdAt).toLocaleString()}
                      </span>
                    </div>
                    <p className="text-sm text-gray-700 whitespace-pre-wrap">{comment.content}</p>
                  </div>
                ))}

                {comments.length === 0 && (
                  <p className="text-sm text-gray-500 text-center py-4">No comments yet</p>
                )}
              </div>
            </div>
          )}

          {activeTab === 'activity' && (
            <div className="space-y-3">
              <div className="text-sm text-gray-500 text-center py-4">
                <p>Created: {new Date(record.createdAt).toLocaleString()}</p>
                <p>Last modified: {new Date(record.updatedAt).toLocaleString()}</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
