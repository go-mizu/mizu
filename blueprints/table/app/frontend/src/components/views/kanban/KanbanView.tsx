import { useState, useMemo, useCallback, useEffect } from 'react';
import {
  DndContext,
  DragOverlay,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragEndEvent,
  type DragOverEvent,
} from '@dnd-kit/core';
import { sortableKeyboardCoordinates } from '@dnd-kit/sortable';
import { useBaseStore } from '../../../stores/baseStore';
import type { TableRecord, Field } from '../../../types';
import { RecordModal } from '../RecordModal';
import { KanbanColumn } from './KanbanColumn';
import { KanbanCardOverlay } from './KanbanCard';
import { KanbanSettings } from './KanbanSettings';
import { DEFAULT_KANBAN_CONFIG, type KanbanConfig, type KanbanColumn as KanbanColumnType } from './types';

export function KanbanView() {
  const {
    currentView,
    fields,
    createRecord,
    updateCellValue,
    getSortedRecords,
    updateViewConfig,
  } = useBaseStore();

  // Get filtered and sorted records
  const records = getSortedRecords();

  // State
  const [expandedRecord, setExpandedRecord] = useState<TableRecord | null>(null);
  const [showSettings, setShowSettings] = useState(false);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [overId, setOverId] = useState<string | null>(null);

  // Load config from view
  const config = useMemo((): KanbanConfig => {
    if (!currentView?.config) return DEFAULT_KANBAN_CONFIG;
    const viewConfig = typeof currentView.config === 'string'
      ? JSON.parse(currentView.config)
      : currentView.config;
    return {
      ...DEFAULT_KANBAN_CONFIG,
      groupBy: viewConfig.groupBy || null,
      cardFields: viewConfig.cardFields || [],
      coverField: viewConfig.coverField || null,
      cardSize: viewConfig.cardSize || 'medium',
      cardCoverFit: viewConfig.cardCoverFit || 'cover',
      hideEmptyColumns: viewConfig.hideEmptyColumns || false,
      collapsedColumns: viewConfig.collapsedColumns || [],
      cardColorField: viewConfig.cardColorField || null,
    };
  }, [currentView?.config]);

  // Update config handler
  const handleConfigChange = useCallback((updates: Partial<KanbanConfig>) => {
    const newConfig = { ...config, ...updates };
    const viewConfig = currentView?.config && typeof currentView.config === 'object'
      ? { ...currentView.config }
      : {};
    updateViewConfig({
      ...viewConfig,
      groupBy: newConfig.groupBy,
      cardFields: newConfig.cardFields,
      coverField: newConfig.coverField,
      cardSize: newConfig.cardSize,
      cardCoverFit: newConfig.cardCoverFit,
      hideEmptyColumns: newConfig.hideEmptyColumns,
      collapsedColumns: newConfig.collapsedColumns,
      cardColorField: newConfig.cardColorField,
    });
  }, [config, currentView?.config, updateViewConfig]);

  // Get the field used for grouping
  const groupByField = useMemo(() => {
    if (config.groupBy) {
      return fields.find((f) => f.id === config.groupBy);
    }
    // Fallback: find first single_select field
    return fields.find((f) => f.type === 'single_select');
  }, [fields, config.groupBy]);

  // Auto-set groupBy if not set but a suitable field exists
  useEffect(() => {
    if (!config.groupBy && groupByField) {
      handleConfigChange({ groupBy: groupByField.id });
    }
  }, [config.groupBy, groupByField, handleConfigChange]);

  // Get primary field for card titles
  const primaryField = useMemo(() => {
    return fields.find((f) => f.is_primary) || fields.find((f) => f.type === 'text') || fields[0];
  }, [fields]);

  // Get display fields for cards
  const displayFields = useMemo(() => {
    if (config.cardFields.length > 0) {
      return config.cardFields
        .map((id) => fields.find((f) => f.id === id))
        .filter((f): f is Field => Boolean(f));
    }
    // Default: first 3 visible, non-primary, non-groupBy fields
    return fields
      .filter((f) => !f.is_hidden && !f.is_primary && f.id !== groupByField?.id)
      .slice(0, 3);
  }, [fields, config.cardFields, groupByField]);

  // Get cover field
  const coverField = useMemo(() => {
    if (!config.coverField) return undefined;
    return fields.find((f) => f.id === config.coverField);
  }, [fields, config.coverField]);

  // Get color field
  const colorField = useMemo(() => {
    if (!config.cardColorField) return undefined;
    return fields.find((f) => f.id === config.cardColorField);
  }, [fields, config.cardColorField]);

  // Build columns from groupBy field
  const columns = useMemo((): KanbanColumnType[] => {
    if (!groupByField) {
      return [{
        id: '__all__',
        name: 'All Records',
        color: '#6b7280',
        records,
        isCollapsed: false,
      }];
    }

    const columnMap = new Map<string, KanbanColumnType>();

    // For single_select or multi_select fields, build from choices
    if (groupByField.type === 'single_select' || groupByField.type === 'multi_select') {
      const choices = groupByField.options?.choices || [];

      // Create uncategorized column
      columnMap.set('__uncategorized__', {
        id: '__uncategorized__',
        name: 'Uncategorized',
        color: '#6b7280',
        records: [],
        isCollapsed: config.collapsedColumns.includes('__uncategorized__'),
      });

      // Create columns for each choice
      choices.forEach((choice: { id: string; name: string; color: string }) => {
        columnMap.set(choice.id, {
          id: choice.id,
          name: choice.name,
          color: choice.color,
          records: [],
          isCollapsed: config.collapsedColumns.includes(choice.id),
        });
      });

      // Assign records to columns
      records.forEach((record) => {
        const value = record.values[groupByField.id] as string | undefined;
        if (value && columnMap.has(value)) {
          columnMap.get(value)!.records.push(record);
        } else {
          columnMap.get('__uncategorized__')!.records.push(record);
        }
      });

      // Build result: choices in order, then uncategorized if non-empty
      const result: KanbanColumnType[] = choices.map(
        (choice: { id: string }) => columnMap.get(choice.id)!
      );
      const uncategorized = columnMap.get('__uncategorized__')!;
      if (uncategorized.records.length > 0) {
        result.unshift(uncategorized);
      }

      // Filter empty columns if configured
      if (config.hideEmptyColumns) {
        return result.filter((col) => col.records.length > 0);
      }

      return result;
    }

    // For user fields, build from unique values in records
    if (groupByField.type === 'user') {
      const userValues = new Set<string>();
      records.forEach((record) => {
        const value = record.values[groupByField.id];
        if (value && typeof value === 'string') {
          userValues.add(value);
        }
      });

      // Create uncategorized column
      columnMap.set('__uncategorized__', {
        id: '__uncategorized__',
        name: 'Unassigned',
        color: '#6b7280',
        records: [],
        isCollapsed: config.collapsedColumns.includes('__uncategorized__'),
      });

      // Create columns for each unique user
      Array.from(userValues).forEach((userId) => {
        columnMap.set(userId, {
          id: userId,
          name: userId, // TODO: Resolve user name
          color: stringToColor(userId),
          records: [],
          isCollapsed: config.collapsedColumns.includes(userId),
        });
      });

      // Assign records to columns
      records.forEach((record) => {
        const value = record.values[groupByField.id] as string | undefined;
        if (value && columnMap.has(value)) {
          columnMap.get(value)!.records.push(record);
        } else {
          columnMap.get('__uncategorized__')!.records.push(record);
        }
      });

      const result = Array.from(columnMap.values());

      if (config.hideEmptyColumns) {
        return result.filter((col) => col.records.length > 0);
      }

      return result;
    }

    // Fallback: all records in one column
    return [{
      id: '__all__',
      name: 'All Records',
      color: '#6b7280',
      records,
      isCollapsed: false,
    }];
  }, [groupByField, records, config.collapsedColumns, config.hideEmptyColumns]);

  // Get active record for drag overlay
  const activeRecord = useMemo(() => {
    if (!activeId) return null;
    return records.find((r) => r.id === activeId) || null;
  }, [activeId, records]);

  // DnD sensors
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 5,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  // DnD handlers
  const handleDragStart = useCallback((event: DragStartEvent) => {
    setActiveId(event.active.id as string);
  }, []);

  const handleDragOver = useCallback((event: DragOverEvent) => {
    const { over } = event;
    if (over) {
      // Check if over a column or a card
      const overData = over.data.current;
      if (overData?.type === 'column') {
        setOverId(over.id as string);
      } else if (overData?.type === 'card') {
        // Find which column this card belongs to
        const cardRecord = overData.record as TableRecord;
        const columnId = findColumnForRecord(cardRecord, groupByField);
        setOverId(columnId);
      }
    } else {
      setOverId(null);
    }
  }, [groupByField]);

  const handleDragEnd = useCallback(async (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveId(null);
    setOverId(null);

    if (!over || !groupByField) return;

    const activeData = active.data.current;
    const overData = over.data.current;

    // Determine target column
    let targetColumnId: string | null = null;
    if (overData?.type === 'column') {
      targetColumnId = over.id as string;
    } else if (overData?.type === 'card') {
      const cardRecord = overData.record as TableRecord;
      targetColumnId = findColumnForRecord(cardRecord, groupByField);
    }

    if (!targetColumnId || !activeData?.record) return;

    const draggedRecord = activeData.record as TableRecord;
    const currentColumnId = findColumnForRecord(draggedRecord, groupByField);

    // Only update if moved to a different column
    if (currentColumnId !== targetColumnId) {
      const newValue = targetColumnId === '__uncategorized__' ? null : targetColumnId;
      await updateCellValue(draggedRecord.id, groupByField.id, newValue);
    }
  }, [groupByField, updateCellValue]);

  const handleDragCancel = useCallback(() => {
    setActiveId(null);
    setOverId(null);
  }, []);

  // Add card to column
  const handleAddCard = useCallback(async (columnId: string) => {
    if (!groupByField) {
      await createRecord({});
      return;
    }

    const initialValues = columnId !== '__uncategorized__'
      ? { [groupByField.id]: columnId }
      : {};
    const record = await createRecord(initialValues);
    setExpandedRecord(record);
  }, [groupByField, createRecord]);

  // Toggle column collapse
  const handleToggleCollapse = useCallback((columnId: string) => {
    const newCollapsed = config.collapsedColumns.includes(columnId)
      ? config.collapsedColumns.filter((id) => id !== columnId)
      : [...config.collapsedColumns, columnId];
    handleConfigChange({ collapsedColumns: newCollapsed });
  }, [config.collapsedColumns, handleConfigChange]);

  // No grouping field available
  if (!groupByField) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500">
        <div className="text-center max-w-md">
          <svg className="w-16 h-16 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7m0 10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2h-2a2 2 0 00-2 2" />
          </svg>
          <h3 className="text-xl font-semibold text-gray-900 mb-2">No stacking field</h3>
          <p className="text-sm text-gray-500 mb-4">
            Kanban view requires a Single Select or User field to group cards into columns.
          </p>
          <button
            onClick={() => setShowSettings(true)}
            className="px-4 py-2 bg-primary text-white rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
          >
            Configure Kanban
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-slate-200 bg-white">
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-600">
            Stacked by: <span className="font-medium">{groupByField.name}</span>
          </span>
          <span className="text-xs text-slate-400">
            {records.length} record{records.length !== 1 ? 's' : ''}
          </span>
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
            <KanbanSettings
              config={config}
              fields={fields}
              onConfigChange={handleConfigChange}
              onClose={() => setShowSettings(false)}
            />
          )}
        </div>
      </div>

      {/* Kanban board */}
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragStart={handleDragStart}
        onDragOver={handleDragOver}
        onDragEnd={handleDragEnd}
        onDragCancel={handleDragCancel}
      >
        <div className="flex-1 overflow-x-auto p-4">
          <div className="flex gap-4 min-h-full">
            {columns.map((column) => (
              <KanbanColumn
                key={column.id}
                column={column}
                primaryField={primaryField}
                displayFields={displayFields}
                coverField={coverField}
                colorField={colorField}
                config={config}
                onAddCard={handleAddCard}
                onCardClick={setExpandedRecord}
                onToggleCollapse={handleToggleCollapse}
                isOver={overId === column.id}
              />
            ))}
          </div>
        </div>

        <DragOverlay>
          {activeRecord && (
            <KanbanCardOverlay
              record={activeRecord}
              primaryField={primaryField}
              displayFields={displayFields}
              coverField={coverField}
              config={config}
            />
          )}
        </DragOverlay>
      </DndContext>

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

// Helper: find column ID for a record
function findColumnForRecord(record: TableRecord, groupByField: Field | undefined): string {
  if (!groupByField) return '__all__';
  const value = record.values[groupByField.id] as string | undefined;
  return value || '__uncategorized__';
}

// Helper: generate consistent color from string
function stringToColor(str: string): string {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = hash % 360;
  return `hsl(${hue}, 60%, 50%)`;
}
