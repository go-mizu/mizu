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
import { RecordSidebar } from '../RecordSidebar';
import { KanbanColumn } from './KanbanColumn';
import { KanbanCardOverlay } from './KanbanCard';
import { KanbanSettings } from './KanbanSettings';
import { DEFAULT_KANBAN_CONFIG, type KanbanConfig, type KanbanColumn as KanbanColumnType } from './types';

// Predefined colors for new stacks
const STACK_COLORS = [
  '#3b82f6', // blue
  '#10b981', // green
  '#f59e0b', // amber
  '#ef4444', // red
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#06b6d4', // cyan
  '#84cc16', // lime
  '#f97316', // orange
  '#6366f1', // indigo
];

export function KanbanView() {
  const {
    currentView,
    fields,
    createRecord,
    updateCellValue,
    getSortedRecords,
    updateViewConfig,
    createSelectOption,
  } = useBaseStore();

  // Get filtered and sorted records
  const records = getSortedRecords();

  // State
  const [expandedRecord, setExpandedRecord] = useState<TableRecord | null>(null);
  const [showSettings, setShowSettings] = useState(false);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [overId, setOverId] = useState<string | null>(null);
  const [showAddStack, setShowAddStack] = useState(false);
  const [newStackName, setNewStackName] = useState('');
  const [newStackColor, setNewStackColor] = useState('#3b82f6');

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
      keepSorted: viewConfig.keepSorted || false,
      hideEmptyFields: viewConfig.hideEmptyFields || false,
      cardPositions: viewConfig.cardPositions || {},
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
      keepSorted: newConfig.keepSorted,
      hideEmptyFields: newConfig.hideEmptyFields,
      cardPositions: newConfig.cardPositions,
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

      // Apply manual ordering if not keepSorted
      if (!config.keepSorted && config.cardPositions) {
        result.forEach((col) => {
          const manualOrder = config.cardPositions[col.id];
          if (manualOrder && manualOrder.length > 0) {
            col.records.sort((a, b) => {
              const aIdx = manualOrder.indexOf(a.id);
              const bIdx = manualOrder.indexOf(b.id);
              if (aIdx === -1 && bIdx === -1) return 0;
              if (aIdx === -1) return 1;
              if (bIdx === -1) return -1;
              return aIdx - bIdx;
            });
          }
        });
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

      // Apply manual ordering if not keepSorted
      if (!config.keepSorted && config.cardPositions) {
        result.forEach((col) => {
          const manualOrder = config.cardPositions[col.id];
          if (manualOrder && manualOrder.length > 0) {
            col.records.sort((a, b) => {
              const aIdx = manualOrder.indexOf(a.id);
              const bIdx = manualOrder.indexOf(b.id);
              if (aIdx === -1 && bIdx === -1) return 0;
              if (aIdx === -1) return 1;
              if (bIdx === -1) return -1;
              return aIdx - bIdx;
            });
          }
        });
      }

      if (config.hideEmptyColumns) {
        return result.filter((col) => col.records.length > 0);
      }

      return result;
    }

    // For linked record fields (single-link only), build from unique linked records
    if (groupByField.type === 'link') {
      const linkedRecordValues = new Map<string, { id: string; name: string }>();

      // Extract unique linked records from all records
      records.forEach((record) => {
        const value = record.values[groupByField.id];
        if (Array.isArray(value) && value.length > 0) {
          const linkedRec = value[0] as { id: string; name: string };
          if (linkedRec.id && linkedRec.name) {
            linkedRecordValues.set(linkedRec.id, linkedRec);
          }
        }
      });

      // Create uncategorized column
      columnMap.set('__uncategorized__', {
        id: '__uncategorized__',
        name: 'Unlinked',
        color: '#6b7280',
        records: [],
        isCollapsed: config.collapsedColumns.includes('__uncategorized__'),
      });

      // Create columns from linked records
      linkedRecordValues.forEach((linkedRec, id) => {
        columnMap.set(id, {
          id,
          name: linkedRec.name,
          color: stringToColor(id),
          records: [],
          isCollapsed: config.collapsedColumns.includes(id),
        });
      });

      // Assign records to columns
      records.forEach((record) => {
        const value = record.values[groupByField.id];
        if (Array.isArray(value) && value.length > 0) {
          const linkedRec = value[0] as { id: string; name: string };
          columnMap.get(linkedRec.id)?.records.push(record);
        } else {
          columnMap.get('__uncategorized__')?.records.push(record);
        }
      });

      const result = Array.from(columnMap.values());

      // Apply manual ordering if not keepSorted
      if (!config.keepSorted && config.cardPositions) {
        result.forEach((col) => {
          const manualOrder = config.cardPositions[col.id];
          if (manualOrder && manualOrder.length > 0) {
            col.records.sort((a, b) => {
              const aIdx = manualOrder.indexOf(a.id);
              const bIdx = manualOrder.indexOf(b.id);
              if (aIdx === -1 && bIdx === -1) return 0;
              if (aIdx === -1) return 1;
              if (bIdx === -1) return -1;
              return aIdx - bIdx;
            });
          }
        });
      }

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
  }, [groupByField, records, config.collapsedColumns, config.hideEmptyColumns, config.keepSorted, config.cardPositions]);

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

  // Reorder card within the same column
  const reorderWithinColumn = useCallback((
    columnId: string,
    recordId: string,
    targetIndex: number
  ) => {
    const column = columns.find((c) => c.id === columnId);
    if (!column) return;

    // Get current order or initialize from column records
    const currentOrder = config.cardPositions[columnId] || column.records.map((r) => r.id);
    const fromIndex = currentOrder.indexOf(recordId);
    if (fromIndex === -1) return;

    // Reorder
    const newOrder = [...currentOrder];
    newOrder.splice(fromIndex, 1);
    newOrder.splice(targetIndex, 0, recordId);

    // Update config
    handleConfigChange({
      cardPositions: {
        ...config.cardPositions,
        [columnId]: newOrder,
      },
    });
  }, [columns, config.cardPositions, handleConfigChange]);

  // Add card to a specific position in a column
  const addToColumnPosition = useCallback((
    columnId: string,
    recordId: string,
    targetIndex?: number
  ) => {
    const column = columns.find((c) => c.id === columnId);
    if (!column) return;

    const currentOrder = config.cardPositions[columnId] || column.records.map((r) => r.id);
    const newOrder = currentOrder.filter((id) => id !== recordId); // Remove if exists
    const insertAt = targetIndex ?? newOrder.length;
    newOrder.splice(insertAt, 0, recordId);

    handleConfigChange({
      cardPositions: {
        ...config.cardPositions,
        [columnId]: newOrder,
      },
    });
  }, [columns, config.cardPositions, handleConfigChange]);

  const handleDragEnd = useCallback(async (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveId(null);
    setOverId(null);

    if (!over || !groupByField) return;

    const activeData = active.data.current;
    const overData = over.data.current;

    if (!activeData?.record) return;

    const draggedRecord = activeData.record as TableRecord;
    const currentColumnId = findColumnForRecord(draggedRecord, groupByField);

    // Determine target column and position
    let targetColumnId: string;
    let targetIndex: number | undefined;

    if (overData?.type === 'column') {
      targetColumnId = over.id as string;
      targetIndex = undefined; // Append to end
    } else if (overData?.type === 'card') {
      const overRecord = overData.record as TableRecord;
      targetColumnId = findColumnForRecord(overRecord, groupByField);
      // Find position of the over card in its column
      const targetColumn = columns.find((c) => c.id === targetColumnId);
      targetIndex = targetColumn?.records.findIndex((r) => r.id === overRecord.id);
    } else {
      return;
    }

    // Same column: reorder within column (if manual ordering enabled)
    if (currentColumnId === targetColumnId) {
      if (!config.keepSorted && targetIndex !== undefined) {
        reorderWithinColumn(currentColumnId, draggedRecord.id, targetIndex);
      }
      return;
    }

    // Different column: update field value
    const newValue = targetColumnId === '__uncategorized__' ? null : targetColumnId;

    // For linked records, need to set as array
    if (groupByField.type === 'link') {
      const linkedColumn = columns.find((c) => c.id === targetColumnId);
      if (linkedColumn && targetColumnId !== '__uncategorized__') {
        await updateCellValue(draggedRecord.id, groupByField.id, [{ id: targetColumnId, name: linkedColumn.name }]);
      } else {
        await updateCellValue(draggedRecord.id, groupByField.id, []);
      }
    } else {
      await updateCellValue(draggedRecord.id, groupByField.id, newValue);
    }

    // Update position in new column if manual ordering
    if (!config.keepSorted) {
      addToColumnPosition(targetColumnId, draggedRecord.id, targetIndex);
    }
  }, [groupByField, updateCellValue, columns, config.keepSorted, reorderWithinColumn, addToColumnPosition]);

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

  // Add new stack (creates a new select option)
  const handleAddStack = useCallback(async () => {
    if (!groupByField || groupByField.type !== 'single_select' || !newStackName.trim()) return;

    await createSelectOption(groupByField.id, newStackName.trim(), newStackColor);
    setNewStackName('');
    setNewStackColor('#3b82f6');
    setShowAddStack(false);
  }, [groupByField, newStackName, newStackColor, createSelectOption]);

  // No grouping field available
  if (!groupByField) {
    return (
      <div className="flex-1 flex items-center justify-center bg-[var(--at-bg)]">
        <div className="empty-state animate-fade-in">
          <div className="empty-state-icon-wrapper">
            <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7m0 10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2h-2a2 2 0 00-2 2" />
            </svg>
          </div>
          <h3 className="empty-state-title">No stacking field</h3>
          <p className="empty-state-description">
            Kanban view requires a Single Select, User, or Link field to group cards into columns.
          </p>
          <button
            onClick={() => setShowSettings(true)}
            className="btn btn-primary mt-2"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            Configure Kanban
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Toolbar */}
      <div className="view-toolbar">
        <div className="view-toolbar-left">
          <span className="text-sm text-[var(--at-text-secondary)]">
            Stacked by: <span className="font-medium text-[var(--at-text)]">{groupByField.name}</span>
          </span>
          <span className="view-toolbar-count">
            {records.length} record{records.length !== 1 ? 's' : ''}
          </span>
        </div>
        <div className="view-toolbar-right relative">
          <button
            onClick={() => setShowSettings(true)}
            className="toolbar-btn"
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

            {/* Add Stack button (only for single_select fields) */}
            {groupByField?.type === 'single_select' && (
              <button
                onClick={() => setShowAddStack(true)}
                className="flex-shrink-0 w-72 min-h-[200px] bg-[var(--at-surface-muted)] rounded-xl border-2 border-dashed border-[var(--at-border-strong)] flex flex-col items-center justify-center gap-2 text-[var(--at-muted)] hover:bg-[var(--at-surface-hover)] hover:border-[var(--at-primary)] hover:text-[var(--at-primary)] transition-colors"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                <span className="text-sm font-medium">Add Stack</span>
              </button>
            )}
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

      {/* Record sidebar */}
      {expandedRecord && (
        <RecordSidebar
          record={expandedRecord}
          onClose={() => setExpandedRecord(null)}
          onNavigate={(direction) => {
            const currentIndex = records.findIndex(r => r.id === expandedRecord.id);
            const newIndex = direction === 'prev' ? currentIndex - 1 : currentIndex + 1;
            if (newIndex >= 0 && newIndex < records.length) {
              setExpandedRecord(records[newIndex]);
            }
          }}
          hasPrev={records.findIndex(r => r.id === expandedRecord.id) > 0}
          hasNext={records.findIndex(r => r.id === expandedRecord.id) < records.length - 1}
          position={records.findIndex(r => r.id === expandedRecord.id) + 1}
          total={records.length}
        />
      )}

      {/* Add Stack Modal */}
      {showAddStack && (
        <div className="modal-overlay">
          <div className="modal-content" style={{ maxWidth: '400px' }}>
            <div className="modal-header">
              <h3>Add New Stack</h3>
              <button onClick={() => setShowAddStack(false)} className="modal-close">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="modal-body space-y-4">
              <div>
                <label className="field-label">Stack Name</label>
                <input
                  type="text"
                  value={newStackName}
                  onChange={(e) => setNewStackName(e.target.value)}
                  placeholder="Enter stack name..."
                  className="input"
                  autoFocus
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && newStackName.trim()) {
                      handleAddStack();
                    } else if (e.key === 'Escape') {
                      setShowAddStack(false);
                    }
                  }}
                />
              </div>
              <div>
                <label className="field-label">Color</label>
                <div className="flex gap-2 flex-wrap">
                  {STACK_COLORS.map((color) => (
                    <button
                      key={color}
                      onClick={() => setNewStackColor(color)}
                      className={`color-swatch color-swatch-lg ${newStackColor === color ? 'color-swatch-active' : ''}`}
                      style={{ backgroundColor: color }}
                    />
                  ))}
                </div>
              </div>
            </div>
            <div className="modal-footer">
              <button onClick={() => { setShowAddStack(false); setNewStackName(''); }} className="btn btn-secondary">
                Cancel
              </button>
              <button onClick={handleAddStack} disabled={!newStackName.trim()} className="btn btn-primary">
                Add Stack
              </button>
            </div>
          </div>
        </div>
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
