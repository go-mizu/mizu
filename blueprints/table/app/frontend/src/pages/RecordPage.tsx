import React, { useState, useEffect, useCallback } from 'react';
import { useBaseStore } from '../stores/baseStore';
import type { TableRecord, Field, CellValue } from '../types';

interface RecordPageProps {
  recordId: string;
  tableId: string;
  viewId?: string;
  onBack: () => void;
}

export function RecordPage({ recordId, tableId, viewId, onBack }: RecordPageProps) {
  const {
    fields,
    records,
    updateCellValue,
    deleteRecord,
    comments,
    fetchComments,
    createComment,
    getSortedRecords,
    loadRecords,
    selectTable,
    selectView,
    currentTable,
    currentView
  } = useBaseStore();

  const [newComment, setNewComment] = useState('');
  const [isLoading, setIsLoading] = useState(true);

  // Get current record and sorted records for navigation
  const sortedRecords = getSortedRecords();
  const currentRecord = records.find(r => r.id === recordId);
  const currentIndex = sortedRecords.findIndex(r => r.id === recordId);
  const hasPrev = currentIndex > 0;
  const hasNext = currentIndex < sortedRecords.length - 1;

  // Load data if necessary
  useEffect(() => {
    const initData = async () => {
      setIsLoading(true);
      if (currentTable?.id !== tableId) {
        await selectTable(tableId);
      }
      if (viewId && currentView?.id !== viewId) {
        await selectView(viewId);
      }
      if (records.length === 0) {
        await loadRecords(tableId);
      }
      await fetchComments(recordId);
      setIsLoading(false);
    };
    initData();
  }, [tableId, viewId, recordId, currentTable?.id, currentView?.id, selectTable, selectView, loadRecords, fetchComments]);

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return;
      }

      if (e.key === 'Escape') {
        onBack();
      } else if ((e.key === 'ArrowLeft' || e.key === 'k') && hasPrev) {
        e.preventDefault();
        navigateRecord('prev');
      } else if ((e.key === 'ArrowRight' || e.key === 'j') && hasNext) {
        e.preventDefault();
        navigateRecord('next');
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [hasPrev, hasNext, onBack]);

  const navigateRecord = useCallback((direction: 'prev' | 'next') => {
    const newIndex = direction === 'prev' ? currentIndex - 1 : currentIndex + 1;
    if (newIndex >= 0 && newIndex < sortedRecords.length) {
      const newRecord = sortedRecords[newIndex];
      // Update URL without full page reload
      window.history.replaceState(
        {},
        '',
        `/record/${newRecord.id}${viewId ? `?view=${viewId}` : ''}`
      );
      // Trigger re-render by updating comments
      fetchComments(newRecord.id);
    }
  }, [currentIndex, sortedRecords, viewId, fetchComments]);

  const handleFieldChange = async (fieldId: string, value: CellValue) => {
    await updateCellValue(recordId, fieldId, value);
  };

  const handleDelete = async () => {
    if (window.confirm('Delete this record? This cannot be undone.')) {
      await deleteRecord(recordId);
      onBack();
    }
  };

  const handleAddComment = async () => {
    if (!newComment.trim()) return;
    await createComment(recordId, newComment.trim());
    setNewComment('');
  };

  // Get primary field value for title
  const primaryField = fields.find(f => f.is_primary);
  const titleValue = currentRecord && primaryField
    ? (currentRecord.values[primaryField.id] as string) || 'Untitled'
    : 'Untitled';

  const getFieldIcon = (type: string) => {
    const icons: Record<string, React.ReactNode> = {
      text: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" />,
      single_line_text: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" />,
      long_text: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />,
      rich_text: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />,
      number: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 20l4-16m2 16l4-16M6 9h14M4 15h14" />,
      currency: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />,
      percent: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />,
      checkbox: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />,
      date: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />,
      datetime: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />,
      single_select: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l4-4 4 4m0 6l-4 4-4-4" />,
      multi_select: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />,
      rating: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />,
      attachment: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />,
      link: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />,
      email: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />,
      url: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />,
      phone: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />,
      duration: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />,
      barcode: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h2M4 12h2m10 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />,
      button: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />,
      formula: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />,
      rollup: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />,
      count: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 20l4-16m2 16l4-16M6 9h14M4 15h14" />,
      lookup: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />,
      autonumber: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 20l4-16m2 16l4-16M6 9h14M4 15h14" />,
      created_time: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />,
      last_modified_time: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />,
      created_by: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />,
      last_modified_by: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />,
      user: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />,
      collaborator: <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />,
    };
    return (
      <svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        {icons[type] || <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" />}
      </svg>
    );
  };

  const renderFieldEditor = (field: Field, record: TableRecord) => {
    const value = record.values[field.id];

    switch (field.type) {
      case 'text':
      case 'single_line_text':
      case 'email':
      case 'url':
      case 'phone':
        return (
          <div className="relative">
            <input
              type={field.type === 'email' ? 'email' : field.type === 'url' ? 'url' : field.type === 'phone' ? 'tel' : 'text'}
              value={(value as string) || ''}
              onChange={(e) => handleFieldChange(field.id, e.target.value || null)}
              className="w-full px-3 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors"
              placeholder={field.type === 'email' ? 'email@example.com' : field.type === 'url' ? 'https://' : field.type === 'phone' ? '+1 (555) 000-0000' : 'Enter text...'}
            />
            {field.type === 'url' && value && (
              <a
                href={value as string}
                target="_blank"
                rel="noopener noreferrer"
                className="absolute right-2 top-1/2 -translate-y-1/2 text-primary hover:text-primary-dark"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                </svg>
              </a>
            )}
          </div>
        );

      case 'long_text':
      case 'rich_text':
        return (
          <textarea
            value={(value as string) || ''}
            onChange={(e) => handleFieldChange(field.id, e.target.value || null)}
            className="w-full px-3 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary min-h-[150px] resize-y transition-colors"
            rows={6}
            placeholder={field.type === 'rich_text' ? 'Supports markdown formatting...' : 'Enter long text...'}
          />
        );

      case 'number':
      case 'currency':
      case 'percent':
        return (
          <div className="relative">
            {field.type === 'currency' && (
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 text-sm">
                {field.options?.currency_symbol || '$'}
              </span>
            )}
            <input
              type="number"
              value={(value as number) ?? ''}
              onChange={(e) => handleFieldChange(field.id, e.target.value ? parseFloat(e.target.value) : null)}
              className={`w-full px-3 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary text-right transition-colors ${field.type === 'currency' ? 'pl-8' : ''}`}
              placeholder="0"
            />
            {field.type === 'percent' && (
              <span className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 text-sm">%</span>
            )}
          </div>
        );

      case 'checkbox':
        return (
          <label className="flex items-center gap-3 cursor-pointer py-2">
            <div className="relative">
              <input
                type="checkbox"
                checked={Boolean(value)}
                onChange={(e) => handleFieldChange(field.id, e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-slate-200 rounded-full peer-checked:bg-primary transition-colors"></div>
              <div className="absolute left-1 top-1 w-4 h-4 bg-white rounded-full transition-transform peer-checked:translate-x-5"></div>
            </div>
            <span className="text-sm text-gray-600">{value ? 'Yes' : 'No'}</span>
          </label>
        );

      case 'date':
      case 'datetime':
        return (
          <input
            type={field.type === 'datetime' ? 'datetime-local' : 'date'}
            value={(value as string) || ''}
            onChange={(e) => handleFieldChange(field.id, e.target.value || null)}
            className="w-full px-3 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors"
          />
        );

      case 'single_select':
        const options = field.options?.choices || [];
        const selectedOption = options.find((opt: { id: string }) => opt.id === value);
        return (
          <div className="space-y-3">
            {selectedOption && (
              <div className="flex flex-wrap gap-2">
                <span
                  className="inline-flex items-center gap-1 px-3 py-1.5 rounded-full text-sm font-medium"
                  style={{ backgroundColor: selectedOption.color + '20', color: selectedOption.color }}
                >
                  {selectedOption.name}
                  <button
                    onClick={() => handleFieldChange(field.id, null)}
                    className="ml-1 hover:bg-black/10 rounded-full p-0.5"
                  >
                    <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </span>
              </div>
            )}
            <select
              value={(value as string) || ''}
              onChange={(e) => handleFieldChange(field.id, e.target.value || null)}
              className="w-full px-3 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors bg-white"
            >
              <option value="">Select an option...</option>
              {options.map((opt: { id: string; name: string; color: string }) => (
                <option key={opt.id} value={opt.id}>{opt.name}</option>
              ))}
            </select>
          </div>
        );

      case 'multi_select':
        const multiOptions = field.options?.choices || [];
        const selectedValues = (value as string[]) || [];
        return (
          <div className="space-y-3">
            <div className="flex flex-wrap gap-2 min-h-[40px] p-2 border border-slate-200 rounded-lg bg-slate-50">
              {selectedValues.length > 0 ? selectedValues.map((valId) => {
                const opt = multiOptions.find((o: { id: string }) => o.id === valId);
                return opt ? (
                  <span
                    key={opt.id}
                    className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-sm font-medium"
                    style={{ backgroundColor: opt.color + '20', color: opt.color }}
                  >
                    {opt.name}
                    <button
                      onClick={() => {
                        const newValues = selectedValues.filter(v => v !== opt.id);
                        handleFieldChange(field.id, newValues.length > 0 ? newValues : null);
                      }}
                      className="ml-1 hover:bg-black/10 rounded-full p-0.5"
                    >
                      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </span>
                ) : null;
              }) : (
                <span className="text-sm text-gray-400 py-1">No options selected</span>
              )}
            </div>
            <div className="border border-slate-200 rounded-lg p-2 space-y-1 max-h-48 overflow-y-auto">
              {multiOptions.map((opt: { id: string; name: string; color: string }) => (
                <label key={opt.id} className="flex items-center gap-3 p-2 rounded hover:bg-slate-50 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={selectedValues.includes(opt.id)}
                    onChange={(e) => {
                      const newValues = e.target.checked
                        ? [...selectedValues, opt.id]
                        : selectedValues.filter(v => v !== opt.id);
                      handleFieldChange(field.id, newValues.length > 0 ? newValues : null);
                    }}
                    className="w-4 h-4 rounded border-gray-300 text-primary focus:ring-primary"
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
          </div>
        );

      case 'rating':
        const maxRating = field.options?.max || 5;
        const currentRating = (value as number) || 0;
        return (
          <div className="flex gap-1.5 py-2">
            {Array.from({ length: maxRating }, (_, i) => (
              <button
                key={i}
                type="button"
                onClick={() => handleFieldChange(field.id, i + 1 === currentRating ? null : i + 1)}
                className="text-yellow-400 hover:scale-110 transition-transform"
              >
                <svg className="w-8 h-8" fill={i < currentRating ? 'currentColor' : 'none'} stroke="currentColor" viewBox="0 0 24 24">
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
          <div className="flex gap-4">
            <div className="flex-1">
              <input
                type="number"
                min="0"
                value={hours || ''}
                onChange={(e) => {
                  const h = parseInt(e.target.value) || 0;
                  handleFieldChange(field.id, h * 3600 + minutes * 60);
                }}
                className="w-full px-3 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors"
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
                className="w-full px-3 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors"
                placeholder="0"
              />
              <span className="text-xs text-gray-500 mt-1 block">Minutes</span>
            </div>
          </div>
        );

      case 'barcode':
        return (
          <div className="space-y-3">
            <input
              type="text"
              value={(value as string) || ''}
              onChange={(e) => handleFieldChange(field.id, e.target.value || null)}
              className="w-full px-3 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary font-mono transition-colors"
              placeholder="Enter barcode value..."
            />
            {value && (
              <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-lg">
                <svg className="w-6 h-6 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
              className="px-5 py-2.5 rounded-lg text-white font-medium transition-opacity hover:opacity-80"
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
          <div className="space-y-4">
            {attachments.length > 0 && (
              <div className="grid grid-cols-4 gap-3">
                {attachments.map((att, idx) => (
                  <div key={att.id || idx} className="relative group">
                    {att.mime_type?.startsWith('image/') || att.url?.match(/\.(jpg|jpeg|png|gif|webp)/i) ? (
                      <img
                        src={att.url}
                        alt={att.filename || 'attachment'}
                        className="w-full h-28 object-cover rounded-lg border border-slate-200"
                      />
                    ) : (
                      <div className="w-full h-28 bg-slate-100 rounded-lg border border-slate-200 flex flex-col items-center justify-center">
                        <svg className="w-10 h-10 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                        </svg>
                        <span className="text-xs text-slate-500 mt-2 truncate max-w-full px-2">{att.filename}</span>
                      </div>
                    )}
                    <button
                      onClick={() => {
                        const remaining = attachments.filter((_, i) => i !== idx);
                        handleFieldChange(field.id, remaining.length > 0 ? remaining as unknown as CellValue : null);
                      }}
                      className="absolute -top-2 -right-2 w-6 h-6 bg-red-500 text-white rounded-full opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-sm shadow-lg"
                    >
                      ×
                    </button>
                    {att.filename && (
                      <p className="text-xs text-gray-500 mt-1 truncate" title={att.filename}>{att.filename}</p>
                    )}
                  </div>
                ))}
              </div>
            )}
            <button className="flex items-center gap-2 px-4 py-3 border border-dashed border-slate-300 rounded-lg text-sm text-gray-500 hover:border-primary hover:text-primary transition-colors w-full justify-center">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
              </svg>
              Attach file
            </button>
          </div>
        );

      case 'link':
        const linkedRecords = (value as unknown as { id: string; primary_value: string }[]) || [];
        return (
          <div className="space-y-3">
            <div className="flex flex-wrap gap-2 min-h-[40px] p-2 border border-slate-200 rounded-lg bg-slate-50">
              {linkedRecords.length > 0 ? linkedRecords.map((rec) => (
                <span
                  key={rec.id}
                  className="inline-flex items-center gap-1 px-3 py-1 bg-blue-50 text-blue-700 rounded-lg text-sm font-medium"
                >
                  {rec.primary_value}
                  <button
                    onClick={() => {
                      const remaining = linkedRecords.filter(r => r.id !== rec.id);
                      handleFieldChange(field.id, remaining.length > 0 ? remaining as unknown as CellValue : null);
                    }}
                    className="ml-1 hover:bg-blue-100 rounded-full p-0.5"
                  >
                    <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </span>
              )) : (
                <span className="text-sm text-gray-400 py-1">No linked records</span>
              )}
            </div>
            <button className="flex items-center gap-2 px-4 py-2 border border-dashed border-slate-300 rounded-lg text-sm text-gray-500 hover:border-primary hover:text-primary transition-colors">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Link record
            </button>
          </div>
        );

      case 'user':
      case 'collaborator':
        const users = (value as { id: string; name: string }[]) || [];
        return (
          <div className="space-y-3">
            <div className="flex flex-wrap gap-2">
              {users.map((user) => (
                <span key={user.id} className="inline-flex items-center gap-2 px-3 py-1.5 bg-slate-100 rounded-full text-sm">
                  <div className="w-6 h-6 rounded-full bg-primary-100 flex items-center justify-center text-xs text-primary font-medium">
                    {user.name.charAt(0).toUpperCase()}
                  </div>
                  {user.name}
                </span>
              ))}
            </div>
            {users.length === 0 && (
              <span className="text-sm text-gray-400">No users assigned</span>
            )}
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
        let displayValue = '—';
        if (value !== null && value !== undefined) {
          if (field.type === 'created_time' || field.type === 'last_modified_time') {
            displayValue = new Date(value as string).toLocaleString();
          } else if (field.type === 'created_by' || field.type === 'last_modified_by') {
            displayValue = (value as unknown as { name: string })?.name || String(value);
          } else {
            displayValue = String(value);
          }
        }
        return (
          <div className="px-4 py-3 bg-slate-50 rounded-lg text-sm text-gray-500 italic">
            {displayValue}
          </div>
        );

      default:
        return (
          <input
            type="text"
            value={value !== null && value !== undefined ? String(value) : ''}
            onChange={(e) => handleFieldChange(field.id, e.target.value || null)}
            className="w-full px-3 py-2.5 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors"
          />
        );
    }
  };

  if (isLoading || !currentRecord) {
    return (
      <div className="min-h-screen bg-[#f6f7fb] flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#f6f7fb]">
      {/* Header */}
      <div className="bg-white border-b border-slate-200 sticky top-0 z-10">
        <div className="max-w-6xl mx-auto px-6 h-14 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <button
              onClick={onBack}
              className="flex items-center gap-2 text-gray-600 hover:text-gray-900 transition-colors"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
              </svg>
              Back to table
            </button>
            <span className="text-sm text-gray-400">|</span>
            <span className="text-sm text-gray-500">
              Record {currentIndex + 1} of {sortedRecords.length}
            </span>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => navigateRecord('prev')}
              disabled={!hasPrev}
              className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-gray-600 hover:text-gray-900 hover:bg-slate-100 rounded-lg disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
              Prev
            </button>
            <button
              onClick={() => navigateRecord('next')}
              disabled={!hasNext}
              className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-gray-600 hover:text-gray-900 hover:bg-slate-100 rounded-lg disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              Next
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </button>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="max-w-6xl mx-auto px-6 py-8">
        <div className="flex gap-8">
          {/* Main content - Fields */}
          <div className="flex-1">
            <div className="bg-white rounded-xl shadow-sm border border-slate-200">
              {/* Record Title */}
              <div className="px-8 py-6 border-b border-slate-100">
                <input
                  type="text"
                  value={titleValue}
                  onChange={(e) => primaryField && handleFieldChange(primaryField.id, e.target.value || null)}
                  className="w-full text-2xl font-semibold border-0 outline-none focus:ring-0 placeholder-gray-400"
                  placeholder="Untitled"
                />
              </div>

              {/* Fields */}
              <div className="divide-y divide-slate-100">
                {fields.filter(f => !f.is_primary).map((field) => (
                  <div key={field.id} className="px-8 py-5">
                    <div className="flex items-start gap-8">
                      <div className="w-40 flex-shrink-0 pt-2">
                        <div className="flex items-center gap-2 text-sm font-medium text-gray-500">
                          {getFieldIcon(field.type)}
                          {field.name}
                        </div>
                      </div>
                      <div className="flex-1">
                        {renderFieldEditor(field, currentRecord)}
                      </div>
                    </div>
                  </div>
                ))}
              </div>

              {/* Actions */}
              <div className="px-8 py-4 border-t border-slate-100 bg-slate-50 rounded-b-xl">
                <button
                  onClick={handleDelete}
                  className="flex items-center gap-2 text-sm text-red-600 hover:text-red-700 transition-colors"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                  Delete record
                </button>
              </div>
            </div>
          </div>

          {/* Sidebar - Comments */}
          <div className="w-80 flex-shrink-0">
            <div className="bg-white rounded-xl shadow-sm border border-slate-200 sticky top-20">
              <div className="px-5 py-4 border-b border-slate-100">
                <h3 className="text-sm font-semibold text-gray-900">Comments</h3>
              </div>

              <div className="p-5">
                {/* Add comment */}
                <div className="flex gap-3 mb-6">
                  <div className="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center text-sm text-primary font-medium flex-shrink-0">
                    U
                  </div>
                  <div className="flex-1">
                    <textarea
                      value={newComment}
                      onChange={(e) => setNewComment(e.target.value)}
                      placeholder="Add a comment..."
                      className="w-full px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary resize-none"
                      rows={2}
                    />
                    <div className="flex justify-end mt-2">
                      <button
                        onClick={handleAddComment}
                        disabled={!newComment.trim()}
                        className="px-3 py-1.5 bg-primary text-white text-sm font-medium rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-primary-dark transition-colors"
                      >
                        Post
                      </button>
                    </div>
                  </div>
                </div>

                {/* Comments list */}
                <div className="space-y-4 max-h-96 overflow-y-auto">
                  {comments.map((comment) => (
                    <div key={comment.id} className="flex gap-3">
                      <div className="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center text-sm text-primary font-medium flex-shrink-0">
                        {comment.user?.name?.charAt(0).toUpperCase() || 'U'}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-baseline gap-2 flex-wrap">
                          <span className="text-sm font-medium text-gray-900">{comment.user?.name || 'Unknown'}</span>
                          <span className="text-xs text-gray-400">
                            {new Date(comment.createdAt).toLocaleDateString()}
                          </span>
                        </div>
                        <p className="text-sm text-gray-700 mt-1 whitespace-pre-wrap break-words">{comment.content}</p>
                      </div>
                    </div>
                  ))}

                  {comments.length === 0 && (
                    <p className="text-sm text-gray-400 text-center py-4">No comments yet</p>
                  )}
                </div>
              </div>

              {/* Activity section */}
              <div className="px-5 py-4 border-t border-slate-100 bg-slate-50 rounded-b-xl">
                <h4 className="text-xs font-medium text-gray-500 uppercase mb-3">Activity</h4>
                <div className="space-y-2 text-xs text-gray-500">
                  <div className="flex items-center gap-2">
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                    </svg>
                    Created {new Date(currentRecord.createdAt).toLocaleDateString()}
                  </div>
                  {currentRecord.updatedAt !== currentRecord.createdAt && (
                    <div className="flex items-center gap-2">
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                      Modified {new Date(currentRecord.updatedAt).toLocaleDateString()}
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
