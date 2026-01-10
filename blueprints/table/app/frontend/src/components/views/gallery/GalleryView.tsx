import { useState, useMemo, useCallback, useEffect } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { TableRecord, Field, Attachment } from '../../../types';
import { RecordModal } from '../RecordModal';
import { GalleryCard } from './GalleryCard';
import { GallerySettings } from './GallerySettings';
import { DEFAULT_GALLERY_CONFIG, CARD_SIZES, type GalleryConfig } from './types';

export function GalleryView() {
  const {
    currentView,
    fields,
    createRecord,
    getSortedRecords,
    updateViewConfig,
  } = useBaseStore();

  const records = getSortedRecords();

  const [expandedRecord, setExpandedRecord] = useState<TableRecord | null>(null);
  const [showSettings, setShowSettings] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState<number>(-1);

  // Load config from view
  const config = useMemo((): GalleryConfig => {
    if (!currentView?.config) return DEFAULT_GALLERY_CONFIG;
    const viewConfig = typeof currentView.config === 'string'
      ? JSON.parse(currentView.config)
      : currentView.config;
    return {
      ...DEFAULT_GALLERY_CONFIG,
      coverField: viewConfig.coverField || null,
      titleField: viewConfig.titleField || null,
      cardFields: viewConfig.cardFields || [],
      cardSize: viewConfig.cardSize || 'medium',
      cardCoverFit: viewConfig.cardCoverFit || 'cover',
      cardColorField: viewConfig.cardColorField || null,
      showEmptyCards: viewConfig.showEmptyCards !== false,
      aspectRatio: viewConfig.aspectRatio || '16:9',
    };
  }, [currentView?.config]);

  // Update config handler
  const handleConfigChange = useCallback((updates: Partial<GalleryConfig>) => {
    const newConfig = { ...config, ...updates };
    const viewConfig = currentView?.config && typeof currentView.config === 'object'
      ? { ...currentView.config }
      : {};
    updateViewConfig({
      ...viewConfig,
      coverField: newConfig.coverField,
      titleField: newConfig.titleField,
      cardFields: newConfig.cardFields,
      cardSize: newConfig.cardSize,
      cardCoverFit: newConfig.cardCoverFit,
      cardColorField: newConfig.cardColorField,
      showEmptyCards: newConfig.showEmptyCards,
      aspectRatio: newConfig.aspectRatio,
    });
  }, [config, currentView?.config, updateViewConfig]);

  // Get cover field
  const coverField = useMemo(() => {
    if (config.coverField) {
      return fields.find((f) => f.id === config.coverField);
    }
    return fields.find((f) => f.type === 'attachment');
  }, [fields, config.coverField]);

  // Auto-set coverField if found an attachment field but not configured
  useEffect(() => {
    if (!config.coverField && coverField) {
      handleConfigChange({ coverField: coverField.id });
    }
  }, [config.coverField, coverField, handleConfigChange]);

  // Get primary/title field
  const primaryField = useMemo(() => {
    if (config.titleField) {
      return fields.find((f) => f.id === config.titleField);
    }
    return fields.find((f) => f.is_primary) ||
      fields.find((f) => f.type === 'text') ||
      fields[0];
  }, [fields, config.titleField]);

  // Get display fields for cards
  const displayFields = useMemo(() => {
    if (config.cardFields.length > 0) {
      return config.cardFields
        .map((id) => fields.find((f) => f.id === id))
        .filter((f): f is Field => Boolean(f));
    }
    return fields
      .filter((f) =>
        !f.is_hidden &&
        !f.is_primary &&
        f.id !== coverField?.id &&
        f.id !== primaryField?.id &&
        f.type !== 'attachment' &&
        f.type !== 'long_text'
      )
      .slice(0, CARD_SIZES[config.cardSize].fieldCount);
  }, [fields, config.cardFields, config.cardSize, coverField, primaryField]);

  // Get color field
  const colorField = useMemo(() => {
    if (!config.cardColorField) return undefined;
    return fields.find((f) => f.id === config.cardColorField);
  }, [fields, config.cardColorField]);

  // Filter records if showEmptyCards is false
  const visibleRecords = useMemo(() => {
    if (config.showEmptyCards || !coverField) return records;
    return records.filter((record) => {
      const attachments = record.values[coverField.id] as Attachment[] | undefined;
      return attachments && attachments.length > 0;
    });
  }, [records, config.showEmptyCards, coverField]);

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (expandedRecord) {
        if (e.key === 'Escape') {
          setExpandedRecord(null);
        }
        return;
      }

      if (showSettings) {
        if (e.key === 'Escape') {
          setShowSettings(false);
        }
        return;
      }

      switch (e.key) {
        case 'j':
        case 'ArrowDown':
          e.preventDefault();
          setSelectedIndex((prev) => Math.min(prev + 1, visibleRecords.length - 1));
          break;
        case 'k':
        case 'ArrowUp':
          e.preventDefault();
          setSelectedIndex((prev) => Math.max(prev - 1, 0));
          break;
        case 'l':
        case 'ArrowRight':
          e.preventDefault();
          setSelectedIndex((prev) => Math.min(prev + 1, visibleRecords.length - 1));
          break;
        case 'h':
        case 'ArrowLeft':
          e.preventDefault();
          setSelectedIndex((prev) => Math.max(prev - 1, 0));
          break;
        case 'Enter':
          e.preventDefault();
          if (selectedIndex >= 0 && selectedIndex < visibleRecords.length) {
            setExpandedRecord(visibleRecords[selectedIndex]);
          }
          break;
        case 'Escape':
          setSelectedIndex(-1);
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [expandedRecord, showSettings, selectedIndex, visibleRecords]);

  const handleAddRecord = async () => {
    const record = await createRecord({});
    setExpandedRecord(record);
  };

  const sizeConfig = CARD_SIZES[config.cardSize];

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-slate-200 bg-white">
        <div className="flex items-center gap-3">
          <span className="text-sm text-gray-600">
            {visibleRecords.length} record{visibleRecords.length !== 1 ? 's' : ''}
            {!config.showEmptyCards && visibleRecords.length !== records.length && (
              <span className="text-slate-400 ml-1">
                ({records.length - visibleRecords.length} hidden)
              </span>
            )}
          </span>
          {coverField && (
            <span className="text-xs text-slate-400">
              Cover: {coverField.name}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2 relative">
          <button
            onClick={() => setShowSettings(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-gray-600 hover:bg-slate-100 rounded-lg transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            Settings
          </button>

          {showSettings && (
            <GallerySettings
              config={config}
              fields={fields}
              onConfigChange={handleConfigChange}
              onClose={() => setShowSettings(false)}
            />
          )}
        </div>
      </div>

      {/* Gallery grid */}
      <div className="flex-1 p-4 overflow-auto">
        {visibleRecords.length > 0 ? (
          <div className={`grid ${sizeConfig.gridCols} ${sizeConfig.gap}`}>
            {visibleRecords.map((record, index) => (
              <GalleryCard
                key={record.id}
                record={record}
                primaryField={primaryField}
                displayFields={displayFields}
                coverField={coverField}
                colorField={colorField}
                config={config}
                onClick={() => setExpandedRecord(record)}
                isSelected={index === selectedIndex}
              />
            ))}

            {/* Add new card */}
            <button
              onClick={handleAddRecord}
              className={`
                bg-slate-50 rounded-xl border-2 border-dashed border-slate-200
                flex items-center justify-center min-h-[200px]
                hover:border-slate-300 hover:bg-slate-100 transition-colors
              `}
            >
              <div className="text-center">
                <svg className="w-8 h-8 mx-auto text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                <span className="text-sm text-gray-500">Add record</span>
              </div>
            </button>
          </div>
        ) : (
          <EmptyState onAddRecord={handleAddRecord} hasRecords={records.length > 0} />
        )}
      </div>

      {/* Keyboard shortcuts hint */}
      {selectedIndex >= 0 && (
        <div className="fixed bottom-4 left-1/2 -translate-x-1/2 bg-gray-900 text-white text-xs px-3 py-1.5 rounded-full shadow-lg">
          <span className="opacity-75">Navigate:</span> ←↑↓→
          <span className="mx-2">|</span>
          <span className="opacity-75">Open:</span> Enter
          <span className="mx-2">|</span>
          <span className="opacity-75">Deselect:</span> Esc
        </div>
      )}

      {/* Record modal */}
      {expandedRecord && (
        <RecordModal
          record={expandedRecord}
          onClose={() => setExpandedRecord(null)}
        />
      )}
    </div>
  );
}

interface EmptyStateProps {
  onAddRecord: () => void;
  hasRecords: boolean;
}

function EmptyState({ onAddRecord, hasRecords }: EmptyStateProps) {
  return (
    <div className="flex-1 flex items-center justify-center text-gray-500 min-h-[400px]">
      <div className="text-center max-w-md">
        <svg className="w-16 h-16 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        {hasRecords ? (
          <>
            <h3 className="text-xl font-semibold text-gray-900 mb-2">No cards with images</h3>
            <p className="text-sm text-gray-500 mb-4">
              All records are hidden because they don't have cover images.
              Enable "Show cards without cover images" in settings.
            </p>
          </>
        ) : (
          <>
            <h3 className="text-xl font-semibold text-gray-900 mb-2">No records yet</h3>
            <p className="text-sm text-gray-500 mb-4">
              Create your first record to see it displayed as a gallery card.
            </p>
          </>
        )}
        <button onClick={onAddRecord} className="btn btn-primary">
          Add record
        </button>
      </div>
    </div>
  );
}
