import { useState, useRef, useEffect, useMemo, Fragment, useCallback } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import { useHistoryStore } from '../../../stores/historyStore';
import type { TableRecord, CellValue } from '../../../types';
import { CellEditor } from './CellEditor';
import { FieldHeader } from './FieldHeader';
import { AddFieldButton } from './AddFieldButton';
import { RecordSidebar } from '../RecordSidebar';
import { SearchBar } from './SearchBar';
import { SummaryBar, type SummaryFunction } from './SummaryBar';
import { FillHandle } from './FillHandle';
import { KeyboardShortcutsModal } from './KeyboardShortcutsModal';
import { normalizeFieldConfig } from './fieldConfig';
import { copyToClipboard, parseClipboardData, stringToCellValue, generateFillSequence } from './clipboardUtils';
import { VirtualizedBody } from './VirtualizedBody';

// Threshold for enabling virtualization
const VIRTUALIZATION_THRESHOLD = 100;

// Row height options
const ROW_HEIGHTS = {
  short: { height: 36, class: 'h-9' },
  medium: { height: 56, class: 'h-14' },
  tall: { height: 96, class: 'h-24' },
  extra_tall: { height: 144, class: 'h-36' },
} as const;

type RowHeightKey = keyof typeof ROW_HEIGHTS;

// Cell range selection interface
interface CellRange {
  startRow: number;
  startCol: number;
  endRow: number;
  endCol: number;
}

export function GridView() {
  const {
    currentTable,
    currentView,
    fields,
    createRecord,
    deleteRecord,
    updateCellValue,
    getSortedRecords,
    getGroupedRecords,
    groupBy,
    updateViewFieldConfig,
    updateViewConfig,
  } = useBaseStore();

  const { pushAction, undo, redo, canUndo, canRedo } = useHistoryStore();

  // Get filtered and sorted records
  const displayRecords = getSortedRecords();
  const groupedRecords = groupBy ? getGroupedRecords() : [{ group: '', records: displayRecords }];

  // Basic state
  const [selectedCell, setSelectedCell] = useState<{ recordId: string; fieldId: string } | null>(null);
  const [editingCell, setEditingCell] = useState<{ recordId: string; fieldId: string } | null>(null);
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set());
  const [expandedRecord, setExpandedRecord] = useState<TableRecord | null>(null);
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; recordId: string } | null>(null);
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set());
  const [columnWidths, setColumnWidths] = useState<Record<string, number>>({});

  // New feature state
  const [showSearch, setShowSearch] = useState(false);
  const [showSummaryBar, setShowSummaryBar] = useState(true);
  const [showKeyboardShortcuts, setShowKeyboardShortcuts] = useState(false);
  const [rowHeight, setRowHeight] = useState<RowHeightKey>('short');
  const [frozenColumnCount, setFrozenColumnCount] = useState(0);
  const [cellRange, setCellRange] = useState<CellRange | null>(null);
  const [rowColorFieldId, setRowColorFieldId] = useState<string | null>(null);
  const [summaryConfig, setSummaryConfig] = useState<Record<string, SummaryFunction>>({});
  const [headerHeight, setHeaderHeight] = useState(36); // Default header height
  const [isResizingHeader, setIsResizingHeader] = useState(false);

  // Use virtualization for large datasets
  const useVirtualization = displayRecords.length > VIRTUALIZATION_THRESHOLD;

  const gridRef = useRef<HTMLDivElement>(null);
  const tableRef = useRef<HTMLTableElement>(null);

  const fieldConfig = useMemo(
    () => normalizeFieldConfig(fields, currentView?.field_config),
    [fields, currentView?.field_config]
  );

  const fieldMap = useMemo(() => {
    return new Map(fields.map((field) => [field.id, field]));
  }, [fields]);

  const orderedFields = useMemo(() => {
    return fieldConfig
      .map((config) => {
        const field = fieldMap.get(config.field_id);
        return field ? { config, field } : null;
      })
      .filter((entry): entry is { config: (typeof fieldConfig)[number]; field: (typeof fields)[number] } => Boolean(entry));
  }, [fieldConfig, fieldMap, fields]);

  const visibleFields = useMemo(() => {
    return orderedFields.filter((entry) => entry.config.visible).map((entry) => entry.field);
  }, [orderedFields]);


  useEffect(() => {
    const nextWidths: Record<string, number> = {};
    fieldConfig.forEach((config) => {
      nextWidths[config.field_id] = config.width;
    });
    setColumnWidths(nextWidths);
  }, [fieldConfig, currentView?.id]);

  useEffect(() => {
    if (!groupBy) {
      setCollapsedGroups(new Set());
    }
  }, [groupBy]);

  const recordIndexMap = useMemo(() => {
    return new Map(displayRecords.map((record, index) => [record.id, index]));
  }, [displayRecords]);

  // Load view settings from config
  useEffect(() => {
    if (currentView?.config) {
      const config = typeof currentView.config === 'string'
        ? JSON.parse(currentView.config)
        : currentView.config;
      if (config.row_height) setRowHeight(config.row_height as RowHeightKey);
      if (config.frozen_columns !== undefined) setFrozenColumnCount(config.frozen_columns);
      if (config.show_summary_bar !== undefined) setShowSummaryBar(config.show_summary_bar);
      if (config.row_color_field_id !== undefined) setRowColorFieldId(config.row_color_field_id);
      if (config.summary_functions) setSummaryConfig(config.summary_functions);
      if (config.header_height !== undefined) setHeaderHeight(config.header_height);
    }
  }, [currentView?.id]);

  // Handle header height resize
  const handleHeaderResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizingHeader(true);
    const startY = e.clientY;
    const startHeight = headerHeight;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const delta = moveEvent.clientY - startY;
      const newHeight = Math.max(36, Math.min(120, startHeight + delta));
      setHeaderHeight(newHeight);
    };

    const handleMouseUp = () => {
      setIsResizingHeader(false);
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      // Save to view config
      const viewConfig = currentView?.config && typeof currentView.config === 'object'
        ? currentView.config as Record<string, unknown>
        : {};
      updateViewConfig({ ...viewConfig, header_height: headerHeight });
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  }, [headerHeight, currentView?.config, updateViewConfig]);

  // Handle summary config changes
  const handleSummaryConfigChange = useCallback((newConfig: Record<string, SummaryFunction>) => {
    setSummaryConfig(newConfig);
    const viewConfig = currentView?.config && typeof currentView.config === 'object'
      ? currentView.config as Record<string, unknown>
      : {};
    updateViewConfig({ ...viewConfig, summary_functions: newConfig });
  }, [currentView?.config, updateViewConfig]);

  // Get row color based on configured color field
  const getRowColor = useCallback((record: TableRecord): string | undefined => {
    if (!rowColorFieldId) return undefined;
    const value = record.values[rowColorFieldId];
    if (!value) return undefined;
    const field = fields.find(f => f.id === rowColorFieldId);
    if (!field || field.type !== 'single_select') return undefined;
    const options = field.options?.choices || [];
    const option = options.find((opt: { id: string }) => opt.id === value);
    return option?.color;
  }, [rowColorFieldId, fields]);

  // Handle bulk delete of selected rows
  const handleBulkDelete = async () => {
    if (selectedRows.size === 0) return;
    const count = selectedRows.size;
    if (!window.confirm(`Delete ${count} record${count > 1 ? 's' : ''}? This cannot be undone.`)) return;

    const deletedRecords = displayRecords.filter(r => selectedRows.has(r.id));
    pushAction({
      type: 'bulk_record_delete',
      payload: { records: deletedRecords.map((r, i) => ({ recordId: r.id, values: r.values, position: i })) },
    });

    for (const recordId of selectedRows) {
      await deleteRecord(recordId);
    }
    setSelectedRows(new Set());
  };

  // Handle copy operation
  const handleCopy = useCallback(async () => {
    if (!selectedCell && !cellRange) return;

    let startRow: number, endRow: number, startCol: number, endCol: number;

    if (cellRange) {
      startRow = Math.min(cellRange.startRow, cellRange.endRow);
      endRow = Math.max(cellRange.startRow, cellRange.endRow);
      startCol = Math.min(cellRange.startCol, cellRange.endCol);
      endCol = Math.max(cellRange.startCol, cellRange.endCol);
    } else if (selectedCell) {
      const rowIdx = displayRecords.findIndex(r => r.id === selectedCell.recordId);
      const colIdx = visibleFields.findIndex(f => f.id === selectedCell.fieldId);
      startRow = endRow = rowIdx;
      startCol = endCol = colIdx;
    } else {
      return;
    }

    const selectedFields = visibleFields.slice(startCol, endCol + 1);

    await copyToClipboard(
      displayRecords.slice(startRow, endRow + 1),
      selectedFields,
      { startRow: 0, startCol: 0, endRow: endRow - startRow, endCol: endCol - startCol }
    );
  }, [selectedCell, cellRange, displayRecords, visibleFields]);

  // Handle paste operation
  const handlePaste = useCallback(async () => {
    if (!selectedCell) return;

    try {
      const text = await navigator.clipboard.readText();
      const parsed = parseClipboardData(text);
      if (parsed.length === 0) return;

      const startRowIdx = displayRecords.findIndex(r => r.id === selectedCell.recordId);
      const startColIdx = visibleFields.findIndex(f => f.id === selectedCell.fieldId);
      if (startRowIdx === -1 || startColIdx === -1) return;

      const updates: { recordId: string; fieldId: string; oldValue: CellValue; newValue: CellValue }[] = [];

      for (let r = 0; r < parsed.length; r++) {
        const recordIdx = startRowIdx + r;
        if (recordIdx >= displayRecords.length) break;
        const record = displayRecords[recordIdx];

        for (let c = 0; c < parsed[r].length; c++) {
          const fieldIdx = startColIdx + c;
          if (fieldIdx >= visibleFields.length) break;
          const field = visibleFields[fieldIdx];

          const oldValue = record.values[field.id] ?? null;
          const newValue = stringToCellValue(parsed[r][c], field);

          updates.push({ recordId: record.id, fieldId: field.id, oldValue, newValue });
          await updateCellValue(record.id, field.id, newValue);
        }
      }

      if (updates.length > 0) {
        pushAction({
          type: 'bulk_cell_update',
          payload: { updates },
        });
      }
    } catch (err) {
      console.error('Paste failed:', err);
    }
  }, [selectedCell, displayRecords, visibleFields, updateCellValue, pushAction]);

  // Handle undo
  const handleUndo = useCallback(async () => {
    const action = undo();
    if (!action) return;

    switch (action.type) {
      case 'cell_update': {
        const p = action.payload as { recordId: string; fieldId: string; oldValue: CellValue };
        await updateCellValue(p.recordId, p.fieldId, p.oldValue);
        break;
      }
      case 'bulk_cell_update': {
        const p = action.payload as { updates: { recordId: string; fieldId: string; oldValue: CellValue }[] };
        for (const u of p.updates) {
          await updateCellValue(u.recordId, u.fieldId, u.oldValue);
        }
        break;
      }
    }
  }, [undo, updateCellValue]);

  // Handle redo
  const handleRedo = useCallback(async () => {
    const action = redo();
    if (!action) return;

    switch (action.type) {
      case 'cell_update': {
        const p = action.payload as { recordId: string; fieldId: string; newValue: CellValue };
        await updateCellValue(p.recordId, p.fieldId, p.newValue);
        break;
      }
      case 'bulk_cell_update': {
        const p = action.payload as { updates: { recordId: string; fieldId: string; newValue: CellValue }[] };
        for (const u of p.updates) {
          await updateCellValue(u.recordId, u.fieldId, u.newValue);
        }
        break;
      }
    }
  }, [redo, updateCellValue]);

  // Handle fill operation from fill handle
  const handleFillEnd = useCallback(async (deltaRows: number, _deltaCols: number) => {
    if (!selectedCell || deltaRows === 0) return;

    const startRowIdx = displayRecords.findIndex(r => r.id === selectedCell.recordId);
    const colIdx = visibleFields.findIndex(f => f.id === selectedCell.fieldId);
    if (startRowIdx === -1 || colIdx === -1) return;

    const field = visibleFields[colIdx];
    const direction = deltaRows > 0 ? 1 : -1;
    const count = Math.abs(deltaRows);

    // Get source values (selected cell or range)
    const sourceValues: CellValue[] = [displayRecords[startRowIdx].values[field.id] ?? null];

    // Generate fill sequence
    const fillValues = generateFillSequence(sourceValues, field, count);
    const updates: { recordId: string; fieldId: string; oldValue: CellValue; newValue: CellValue }[] = [];

    for (let i = 0; i < count; i++) {
      const targetRowIdx = startRowIdx + (i + 1) * direction;
      if (targetRowIdx < 0 || targetRowIdx >= displayRecords.length) break;

      const record = displayRecords[targetRowIdx];
      const oldValue = record.values[field.id] ?? null;
      const newValue = fillValues[i];

      updates.push({ recordId: record.id, fieldId: field.id, oldValue, newValue });
      await updateCellValue(record.id, field.id, newValue);
    }

    if (updates.length > 0) {
      pushAction({
        type: 'bulk_cell_update',
        payload: { updates },
      });
    }
  }, [selectedCell, displayRecords, visibleFields, updateCellValue, pushAction]);

  // Handle duplicate record
  const handleDuplicateRecord = useCallback(async () => {
    if (!selectedCell) return;
    const record = displayRecords.find(r => r.id === selectedCell.recordId);
    if (!record) return;

    const newRecord = await createRecord(record.values);
    pushAction({
      type: 'record_create',
      payload: { recordId: newRecord.id, values: newRecord.values },
    });
  }, [selectedCell, displayRecords, createRecord, pushAction]);

  // Navigate to cell (for search)
  const navigateToCell = useCallback((recordId: string, fieldId: string) => {
    setSelectedCell({ recordId, fieldId });
    // Scroll cell into view
    const rowIdx = displayRecords.findIndex(r => r.id === recordId);
    const cellElement = document.querySelector(`[data-row="${rowIdx}"][data-field="${fieldId}"]`);
    cellElement?.scrollIntoView({ behavior: 'smooth', block: 'center', inline: 'center' });
  }, [displayRecords]);

  // Handle keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const isMod = e.metaKey || e.ctrlKey;

      // Global shortcuts (work regardless of selection)
      if (isMod && e.key === 'f') {
        e.preventDefault();
        setShowSearch(true);
        return;
      }

      if (isMod && e.key === 'z' && !e.shiftKey) {
        e.preventDefault();
        handleUndo();
        return;
      }

      if (isMod && (e.key === 'y' || (e.key === 'z' && e.shiftKey))) {
        e.preventDefault();
        handleRedo();
        return;
      }

      // Show keyboard shortcuts with "?"
      if (e.key === '?' && !editingCell) {
        e.preventDefault();
        setShowKeyboardShortcuts(true);
        return;
      }

      // Close search on Escape
      if (e.key === 'Escape' && showSearch) {
        e.preventDefault();
        setShowSearch(false);
        return;
      }

      // Close keyboard shortcuts on Escape
      if (e.key === 'Escape' && showKeyboardShortcuts) {
        e.preventDefault();
        setShowKeyboardShortcuts(false);
        return;
      }

      // Handle bulk delete with Delete/Backspace when rows are selected
      if ((e.key === 'Delete' || e.key === 'Backspace') && selectedRows.size > 0 && !editingCell && !selectedCell) {
        e.preventDefault();
        handleBulkDelete();
        return;
      }

      // Handle Cmd/Ctrl+A for select all
      if (isMod && e.key === 'a' && !editingCell) {
        e.preventDefault();
        setSelectedRows(new Set(displayRecords.map(r => r.id)));
        return;
      }

      // Copy
      if (isMod && e.key === 'c' && !editingCell) {
        e.preventDefault();
        handleCopy();
        return;
      }

      // Cut
      if (isMod && e.key === 'x' && !editingCell) {
        e.preventDefault();
        handleCopy();
        // Clear selected cells after copy
        if (selectedCell) {
          updateCellValue(selectedCell.recordId, selectedCell.fieldId, null);
        }
        return;
      }

      // Paste
      if (isMod && e.key === 'v' && !editingCell) {
        e.preventDefault();
        handlePaste();
        return;
      }

      // Duplicate record
      if (isMod && e.key === 'd' && !editingCell) {
        e.preventDefault();
        handleDuplicateRecord();
        return;
      }

      // Space to expand record
      if (e.key === ' ' && !editingCell && selectedCell) {
        e.preventDefault();
        const record = displayRecords.find(r => r.id === selectedCell.recordId);
        if (record) setExpandedRecord(record);
        return;
      }

      if (!selectedCell || editingCell) return;

      const currentRecordIndex = displayRecords.findIndex(r => r.id === selectedCell.recordId);
      const currentFieldIndex = visibleFields.findIndex(f => f.id === selectedCell.fieldId);

      // Extend selection with Shift+Arrow
      if (e.shiftKey && ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(e.key)) {
        e.preventDefault();
        const newRange = cellRange || {
          startRow: currentRecordIndex,
          startCol: currentFieldIndex,
          endRow: currentRecordIndex,
          endCol: currentFieldIndex,
        };

        switch (e.key) {
          case 'ArrowUp':
            if (newRange.endRow > 0) newRange.endRow--;
            break;
          case 'ArrowDown':
            if (newRange.endRow < displayRecords.length - 1) newRange.endRow++;
            break;
          case 'ArrowLeft':
            if (newRange.endCol > 0) newRange.endCol--;
            break;
          case 'ArrowRight':
            if (newRange.endCol < visibleFields.length - 1) newRange.endCol++;
            break;
        }
        setCellRange(newRange);
        return;
      }

      switch (e.key) {
        case 'ArrowUp':
          e.preventDefault();
          if (isMod) {
            // Jump to first row
            setSelectedCell({ recordId: displayRecords[0].id, fieldId: selectedCell.fieldId });
          } else if (currentRecordIndex > 0) {
            setSelectedCell({ recordId: displayRecords[currentRecordIndex - 1].id, fieldId: selectedCell.fieldId });
          }
          setCellRange(null);
          break;
        case 'ArrowDown':
          e.preventDefault();
          if (isMod) {
            // Jump to last row
            setSelectedCell({ recordId: displayRecords[displayRecords.length - 1].id, fieldId: selectedCell.fieldId });
          } else if (currentRecordIndex < displayRecords.length - 1) {
            setSelectedCell({ recordId: displayRecords[currentRecordIndex + 1].id, fieldId: selectedCell.fieldId });
          }
          setCellRange(null);
          break;
        case 'ArrowLeft':
          e.preventDefault();
          if (isMod) {
            // Jump to first column
            setSelectedCell({ recordId: selectedCell.recordId, fieldId: visibleFields[0].id });
          } else if (currentFieldIndex > 0) {
            setSelectedCell({ recordId: selectedCell.recordId, fieldId: visibleFields[currentFieldIndex - 1].id });
          }
          setCellRange(null);
          break;
        case 'ArrowRight':
          e.preventDefault();
          if (isMod) {
            // Jump to last column
            setSelectedCell({ recordId: selectedCell.recordId, fieldId: visibleFields[visibleFields.length - 1].id });
          } else if (currentFieldIndex < visibleFields.length - 1) {
            setSelectedCell({ recordId: selectedCell.recordId, fieldId: visibleFields[currentFieldIndex + 1].id });
          }
          setCellRange(null);
          break;
        case 'Tab':
          e.preventDefault();
          if (e.shiftKey) {
            if (currentFieldIndex > 0) {
              setSelectedCell({ recordId: selectedCell.recordId, fieldId: visibleFields[currentFieldIndex - 1].id });
            } else if (currentRecordIndex > 0) {
              setSelectedCell({ recordId: displayRecords[currentRecordIndex - 1].id, fieldId: visibleFields[visibleFields.length - 1].id });
            }
          } else {
            if (currentFieldIndex < visibleFields.length - 1) {
              setSelectedCell({ recordId: selectedCell.recordId, fieldId: visibleFields[currentFieldIndex + 1].id });
            } else if (currentRecordIndex < displayRecords.length - 1) {
              setSelectedCell({ recordId: displayRecords[currentRecordIndex + 1].id, fieldId: visibleFields[0].id });
            }
          }
          setCellRange(null);
          break;
        case 'Enter':
          e.preventDefault();
          if (e.shiftKey) {
            // Insert new row below
            createRecord({});
          } else {
            setEditingCell(selectedCell);
          }
          break;
        case 'Escape':
          e.preventDefault();
          setSelectedCell(null);
          setSelectedRows(new Set());
          setCellRange(null);
          break;
        case 'Delete':
        case 'Backspace':
          if (!editingCell) {
            e.preventDefault();
            const record = displayRecords.find(r => r.id === selectedCell.recordId);
            const field = visibleFields.find(f => f.id === selectedCell.fieldId);
            if (record && field) {
              const oldValue = record.values[field.id] ?? null;
              pushAction({
                type: 'cell_update',
                payload: { recordId: selectedCell.recordId, fieldId: selectedCell.fieldId, oldValue, newValue: null },
              });
            }
            updateCellValue(selectedCell.recordId, selectedCell.fieldId, null);
          }
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [selectedCell, editingCell, displayRecords, visibleFields, updateCellValue, selectedRows, cellRange, showSearch, showKeyboardShortcuts, handleCopy, handlePaste, handleUndo, handleRedo, handleDuplicateRecord, pushAction, createRecord]);

  const handleCellClick = (recordId: string, fieldId: string) => {
    setSelectedCell({ recordId, fieldId });
  };

  const handleCellDoubleClick = (recordId: string, fieldId: string) => {
    setEditingCell({ recordId, fieldId });
  };

  const handleCellChange = async (recordId: string, fieldId: string, value: CellValue) => {
    const record = displayRecords.find(r => r.id === recordId);
    const oldValue = record?.values[fieldId] ?? null;

    pushAction({
      type: 'cell_update',
      payload: { recordId, fieldId, oldValue, newValue: value },
    });

    await updateCellValue(recordId, fieldId, value);
    setEditingCell(null);
  };

  // Check if a cell is in the current selection range
  const isCellInRange = (rowIdx: number, colIdx: number): boolean => {
    if (!cellRange) return false;
    const minRow = Math.min(cellRange.startRow, cellRange.endRow);
    const maxRow = Math.max(cellRange.startRow, cellRange.endRow);
    const minCol = Math.min(cellRange.startCol, cellRange.endCol);
    const maxCol = Math.max(cellRange.startCol, cellRange.endCol);
    return rowIdx >= minRow && rowIdx <= maxRow && colIdx >= minCol && colIdx <= maxCol;
  };

  // Calculate frozen column left offset
  const getFrozenColumnOffset = (colIdx: number): number => {
    let offset = 64; // Row number column width
    for (let i = 0; i < colIdx; i++) {
      const field = visibleFields[i];
      offset += columnWidths[field.id] || field.width || 200;
    }
    return offset;
  };

  const handleAddRow = async () => {
    await createRecord({});
  };

  const handleRowContextMenu = (e: React.MouseEvent, recordId: string) => {
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY, recordId });
  };

  const handleDeleteRow = async () => {
    if (contextMenu) {
      await deleteRecord(contextMenu.recordId);
      setContextMenu(null);
    }
  };

  const handleExpandRecord = (record: TableRecord) => {
    setExpandedRecord(record);
  };

  // Get navigation info for the expanded record
  const getRecordNavigation = useCallback((record: TableRecord | null) => {
    if (!record) return { hasPrev: false, hasNext: false, position: 0, total: 0 };
    const index = recordIndexMap.get(record.id);
    if (index === undefined) return { hasPrev: false, hasNext: false, position: 0, total: 0 };
    return {
      hasPrev: index > 0,
      hasNext: index < displayRecords.length - 1,
      position: index + 1,
      total: displayRecords.length
    };
  }, [recordIndexMap, displayRecords]);

  const handleNavigateRecord = useCallback((direction: 'prev' | 'next') => {
    if (!expandedRecord) return;
    const currentIndex = recordIndexMap.get(expandedRecord.id);
    if (currentIndex === undefined) return;

    const newIndex = direction === 'prev' ? currentIndex - 1 : currentIndex + 1;
    if (newIndex >= 0 && newIndex < displayRecords.length) {
      setExpandedRecord(displayRecords[newIndex]);
    }
  }, [expandedRecord, recordIndexMap, displayRecords]);

  const toggleGroupCollapse = (group: string) => {
    setCollapsedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(group)) {
        next.delete(group);
      } else {
        next.add(group);
      }
      return next;
    });
  };

  const updateFieldConfig = (fieldId: string, updates: Partial<{ visible: boolean; width: number; position: number }>) => {
    const nextConfig = fieldConfig.map((config) => {
      if (config.field_id !== fieldId) return config;
      return { ...config, ...updates };
    });
    updateViewFieldConfig(nextConfig);
  };

  const handleResize = (fieldId: string, width: number) => {
    setColumnWidths((prev) => ({ ...prev, [fieldId]: width }));
  };

  const handleResizeEnd = (fieldId: string, width: number) => {
    updateFieldConfig(fieldId, { width });
  };

  // Auto-fit column width based on content
  const handleAutoFit = useCallback((fieldId: string) => {
    const field = fields.find(f => f.id === fieldId);
    if (!field) return;

    // Calculate max content width
    let maxWidth = field.name.length * 8 + 60; // Header text + padding + icon

    // Check all visible records
    displayRecords.forEach(record => {
      const value = record.values[fieldId];
      if (value === null || value === undefined) return;

      let contentWidth = 60; // Base padding
      if (typeof value === 'string') {
        contentWidth = Math.min(value.length * 7, 400) + 40;
      } else if (typeof value === 'number') {
        contentWidth = String(value).length * 8 + 40;
      } else if (Array.isArray(value)) {
        // For multi-select or attachments
        contentWidth = Math.min(value.length * 80, 400) + 40;
      }

      maxWidth = Math.max(maxWidth, contentWidth);
    });

    // Clamp between min and max
    const finalWidth = Math.max(120, Math.min(maxWidth, 600));

    setColumnWidths(prev => ({ ...prev, [fieldId]: finalWidth }));
    updateFieldConfig(fieldId, { width: finalWidth });
  }, [fields, displayRecords, updateFieldConfig]);

  const handleHideField = (fieldId: string) => {
    updateFieldConfig(fieldId, { visible: false });
  };

  const handleReorderField = (fromId: string, toId: string) => {
    if (fromId === toId) return;
    const visibleOrder = visibleFields.map((field) => field.id);
    const fromIndex = visibleOrder.indexOf(fromId);
    const toIndex = visibleOrder.indexOf(toId);
    if (fromIndex === -1 || toIndex === -1) return;
    const nextVisible = [...visibleOrder];
    const [removed] = nextVisible.splice(fromIndex, 1);
    nextVisible.splice(toIndex, 0, removed);

    const hiddenIds = orderedFields.filter((entry) => !entry.config.visible).map((entry) => entry.field.id);
    const nextOrder = [...nextVisible, ...hiddenIds];
    const nextConfig = nextOrder.map((fieldId, index) => {
      const current = fieldConfig.find((config) => config.field_id === fieldId);
      return {
        field_id: fieldId,
        visible: current?.visible ?? true,
        width: columnWidths[fieldId] ?? current?.width ?? 200,
        position: index,
      };
    });
    updateViewFieldConfig(nextConfig);
  };

  const getCellValue = (record: TableRecord, fieldId: string): CellValue => {
    return record.values[fieldId] ?? null;
  };

  const toggleRowSelection = (recordId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const newSelection = new Set(selectedRows);
    if (e.shiftKey && selectedRows.size > 0) {
      // Range selection
      const lastSelected = Array.from(selectedRows).pop()!;
      const lastIndex = displayRecords.findIndex(r => r.id === lastSelected);
      const currentIndex = displayRecords.findIndex(r => r.id === recordId);
      const start = Math.min(lastIndex, currentIndex);
      const end = Math.max(lastIndex, currentIndex);
      for (let i = start; i <= end; i++) {
        newSelection.add(displayRecords[i].id);
      }
    } else if (e.metaKey || e.ctrlKey) {
      // Toggle selection
      if (newSelection.has(recordId)) {
        newSelection.delete(recordId);
      } else {
        newSelection.add(recordId);
      }
    } else {
      // Single selection
      newSelection.clear();
      newSelection.add(recordId);
    }
    setSelectedRows(newSelection);
  };

  const toggleAllRows = () => {
    if (selectedRows.size === displayRecords.length) {
      setSelectedRows(new Set());
    } else {
      setSelectedRows(new Set(displayRecords.map(r => r.id)));
    }
  };

  if (!currentTable) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500">
        Select a table to view records
      </div>
    );
  }

  const currentRowHeight = ROW_HEIGHTS[rowHeight];

  return (
    <div className="flex-1 overflow-auto flex flex-col relative" ref={gridRef}>
      {/* Search bar */}
      <SearchBar
        records={displayRecords}
        fields={visibleFields}
        isOpen={showSearch}
        onClose={() => setShowSearch(false)}
        onNavigate={navigateToCell}
      />

      {/* Selection bar */}
      {selectedRows.size > 0 && (
        <div className="flex items-center gap-3 px-4 py-2 bg-primary-50 border-b border-primary-200">
          <span className="text-sm font-medium text-primary-700">
            {selectedRows.size} row{selectedRows.size > 1 ? 's' : ''} selected
          </span>
          <button
            onClick={() => setSelectedRows(new Set())}
            className="text-sm text-primary-600 hover:text-primary-800"
          >
            Clear selection
          </button>
          <button
            onClick={handleBulkDelete}
            className="text-sm text-red-600 hover:text-red-800 flex items-center gap-1"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
            Delete selected
          </button>
          <div className="flex items-center gap-2 ml-auto">
            {canUndo() && (
              <button
                onClick={handleUndo}
                className="text-sm text-gray-600 hover:text-gray-800 flex items-center gap-1"
                title="Undo (Cmd/Ctrl+Z)"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
                </svg>
              </button>
            )}
            {canRedo() && (
              <button
                onClick={handleRedo}
                className="text-sm text-gray-600 hover:text-gray-800 flex items-center gap-1"
                title="Redo (Cmd/Ctrl+Shift+Z)"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 10h-10a8 8 0 00-8 8v2M21 10l-6 6m6-6l-6-6" />
                </svg>
              </button>
            )}
            <span className="text-xs text-gray-500">
              Delete/Backspace to delete • Escape to clear
            </span>
          </div>
        </div>
      )}
      <div className="flex-1 overflow-auto">
      <table ref={tableRef} className="w-full border-collapse table-fixed">
        <colgroup>
          <col style={{ width: 64 }} />
          {visibleFields.map((field) => (
            <col key={field.id} style={{ width: columnWidths[field.id] || field.width || 200 }} />
          ))}
          <col style={{ width: 128 }} />
        </colgroup>
        <thead className="sticky top-0 z-10 bg-slate-50 shadow-sm">
          <tr style={{ height: headerHeight }}>
            {/* Row number and checkbox column - frozen */}
            <th
              className="border-b border-r border-slate-200 bg-slate-50 p-0 sticky left-0 z-20 relative"
              style={{ height: headerHeight, width: 64 }}
            >
              <div className="flex items-center justify-center h-full">
                <input
                  type="checkbox"
                  checked={selectedRows.size === displayRecords.length && displayRecords.length > 0}
                  onChange={toggleAllRows}
                  className="w-4 h-4 rounded border-gray-300"
                />
              </div>
            </th>

            {/* Field headers */}
            {visibleFields.map((field, colIdx) => {
              const isFrozen = colIdx < frozenColumnCount;
              return (
                <FieldHeader
                  key={field.id}
                  field={field}
                  width={columnWidths[field.id] || field.width || 200}
                  height={headerHeight}
                  onResize={(width) => handleResize(field.id, width)}
                  onResizeEnd={(width) => handleResizeEnd(field.id, width)}
                  onHide={() => handleHideField(field.id)}
                  onReorder={handleReorderField}
                  isFrozen={isFrozen}
                  frozenOffset={isFrozen ? getFrozenColumnOffset(colIdx) : undefined}
                  onAutoFit={() => handleAutoFit(field.id)}
                />
              );
            })}

            {/* Add field button */}
            <th className="border-b border-slate-200 bg-slate-50 p-0" style={{ height: headerHeight, width: 128 }}>
              <AddFieldButton />
            </th>
          </tr>
          {/* Header resize handle */}
          <tr className="h-0">
            <td
              colSpan={visibleFields.length + 2}
              className="p-0 relative"
            >
              <div
                className={`absolute left-0 right-0 h-1 cursor-ns-resize hover:bg-primary-200 transition-colors ${
                  isResizingHeader ? 'bg-primary-300' : ''
                }`}
                style={{ top: -2 }}
                onMouseDown={handleHeaderResizeStart}
              />
            </td>
          </tr>
        </thead>
      </table>

      {/* Render virtualized or standard body based on record count */}
      {useVirtualization ? (
        <VirtualizedBody
          groupedRecords={groupedRecords}
          visibleFields={visibleFields}
          columnWidths={columnWidths}
          rowHeight={rowHeight}
          frozenColumnCount={frozenColumnCount}
          collapsedGroups={collapsedGroups}
          selectedCell={selectedCell}
          editingCell={editingCell}
          selectedRows={selectedRows}
          groupBy={groupBy}
          showSummaryBar={showSummaryBar}
          recordIndexMap={recordIndexMap}
          onCellClick={handleCellClick}
          onCellDoubleClick={handleCellDoubleClick}
          onCellChange={handleCellChange}
          onCancelEdit={() => setEditingCell(null)}
          onFillEnd={handleFillEnd}
          onToggleRowSelection={toggleRowSelection}
          onExpandRecord={handleExpandRecord}
          onRowContextMenu={handleRowContextMenu}
          onToggleGroupCollapse={toggleGroupCollapse}
          onAddRow={handleAddRow}
          onClearCellRange={() => setCellRange(null)}
          getRowColor={getRowColor}
          getFrozenColumnOffset={getFrozenColumnOffset}
          isCellInRange={isCellInRange}
          renderSummaryBar={() => (
            <SummaryBar
              records={displayRecords}
              fields={visibleFields}
              columnWidths={columnWidths}
              savedConfig={summaryConfig}
              onConfigChange={handleSummaryConfigChange}
            />
          )}
        />
      ) : (
        <table className="w-full border-collapse table-fixed">
          <colgroup>
            <col style={{ width: 64 }} />
            {visibleFields.map((field) => (
              <col key={field.id} style={{ width: columnWidths[field.id] || field.width || 200 }} />
            ))}
            <col style={{ width: 128 }} />
          </colgroup>
          <tbody>
          {groupedRecords.map(({ group, records: groupRecords }) => (
            <Fragment key={group || 'ungrouped'}>
              {groupBy && (
                <tr className="bg-slate-50">
                  <td colSpan={visibleFields.length + 2} className="border-b border-slate-200 px-3 py-2">
                    <button
                      onClick={() => toggleGroupCollapse(group)}
                      className="flex items-center gap-2 text-sm font-semibold text-slate-700"
                    >
                      <svg
                        className={`w-4 h-4 transition-transform ${collapsedGroups.has(group) ? '-rotate-90' : ''}`}
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                      </svg>
                      <span>{group || '(Empty)'}</span>
                      <span className="text-xs text-slate-500 font-normal">{groupRecords.length}</span>
                    </button>
                  </td>
                </tr>
              )}

              {!collapsedGroups.has(group) && groupRecords.map((record) => {
                const rowIndex = recordIndexMap.get(record.id) ?? 0;
                const rowColor = getRowColor(record);
                return (
                  <tr
                    key={record.id}
                    data-row={rowIndex}
                    className={`group ${selectedRows.has(record.id) ? 'bg-primary-50' : 'hover:bg-slate-50'}`}
                    style={{
                      height: currentRowHeight.height,
                      backgroundColor: rowColor && !selectedRows.has(record.id) ? `${rowColor}15` : undefined,
                    }}
                    onContextMenu={(e) => handleRowContextMenu(e, record.id)}
                  >
                    {/* Row number and checkbox - frozen */}
                    <td className="border-b border-r border-slate-200 p-0 sticky left-0 bg-white z-10 relative w-16">
                      {/* Row color indicator bar */}
                      {rowColor && (
                        <div
                          className="absolute left-0 top-0 bottom-0 w-1"
                          style={{ backgroundColor: rowColor }}
                        />
                      )}
                      <div className={`flex items-center justify-center ${currentRowHeight.class} gap-1`}>
                        <input
                          type="checkbox"
                          checked={selectedRows.has(record.id)}
                          onChange={(e) => toggleRowSelection(record.id, e as unknown as React.MouseEvent)}
                          className="w-4 h-4 rounded border-gray-300 opacity-0 group-hover:opacity-100 checked:opacity-100"
                        />
                        <span className="text-xs text-gray-400 w-6 text-center group-hover:hidden">
                          {rowIndex + 1}
                        </span>
                        <button
                          onClick={() => handleExpandRecord(record)}
                          className="hidden group-hover:block text-gray-400 hover:text-gray-600"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
                          </svg>
                        </button>
                      </div>
                    </td>

                    {/* Cells */}
                    {visibleFields.map((field, colIdx) => {
                      const isSelected = selectedCell?.recordId === record.id && selectedCell?.fieldId === field.id;
                      const isEditing = editingCell?.recordId === record.id && editingCell?.fieldId === field.id;
                      const isInRange = isCellInRange(rowIndex, colIdx);
                      const isFrozen = colIdx < frozenColumnCount;
                      const isLastFrozen = colIdx === frozenColumnCount - 1;
                      const value = getCellValue(record, field.id);

                      return (
                        <td
                          key={field.id}
                          data-row={rowIndex}
                          data-field={field.id}
                          className={`border-b border-r border-slate-200 p-0 relative ${
                            isSelected ? 'ring-2 ring-primary ring-inset z-10' : ''
                          } ${isInRange && !isSelected ? 'bg-primary-50/50' : ''} ${
                            isFrozen ? 'sticky bg-white z-10' : ''
                          } ${isLastFrozen && frozenColumnCount > 0 ? 'after:absolute after:right-0 after:top-0 after:bottom-0 after:w-0.5 after:bg-slate-300 after:shadow-sm' : ''}`}
                          style={{
                            width: columnWidths[field.id] || field.width || 200,
                            height: currentRowHeight.height,
                            ...(isFrozen ? { left: getFrozenColumnOffset(colIdx) } : {}),
                          }}
                          onClick={() => {
                            handleCellClick(record.id, field.id);
                            setCellRange(null);
                          }}
                          onDoubleClick={() => handleCellDoubleClick(record.id, field.id)}
                        >
                          <CellEditor
                            field={field}
                            value={value}
                            isEditing={isEditing}
                            onChange={(newValue) => handleCellChange(record.id, field.id, newValue)}
                            onCancel={() => setEditingCell(null)}
                            rowHeight={rowHeight}
                          />
                          {/* Fill handle */}
                          {isSelected && !isEditing && (
                            <FillHandle
                              onFillStart={() => {}}
                              onFillMove={() => {}}
                              onFillEnd={handleFillEnd}
                            />
                          )}
                        </td>
                      );
                    })}

                    {/* Empty cell for add field column */}
                    <td className="border-b border-slate-200" style={{ height: currentRowHeight.height }} />
                  </tr>
                );
              })}
            </Fragment>
          ))}

          {/* Summary bar */}
          {showSummaryBar && (
            <SummaryBar
              records={displayRecords}
              fields={visibleFields}
              columnWidths={columnWidths}
              savedConfig={summaryConfig}
              onConfigChange={handleSummaryConfigChange}
            />
          )}

          {/* Add row button */}
          <tr>
            <td colSpan={visibleFields.length + 2} className="border-b border-slate-200 p-0">
              <button
                onClick={handleAddRow}
                className={`w-full ${currentRowHeight.class} text-left px-4 text-sm text-slate-500 hover:bg-slate-50 flex items-center gap-2`}
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Add row
              </button>
            </td>
          </tr>
          </tbody>
        </table>
      )}

      {/* Context menu */}
      {contextMenu && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setContextMenu(null)} />
          <div
            className="dropdown-menu animate-slide-in"
            style={{ left: contextMenu.x, top: contextMenu.y }}
          >
            <button
              onClick={() => {
                const record = displayRecords.find(r => r.id === contextMenu.recordId);
                if (record) handleExpandRecord(record);
                setContextMenu(null);
              }}
              className="dropdown-item w-full text-left"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
              </svg>
              Expand record
            </button>
            <button
              onClick={async () => {
                const record = displayRecords.find(r => r.id === contextMenu.recordId);
                if (record) {
                  const newRecord = await createRecord(record.values);
                  pushAction({
                    type: 'record_create',
                    payload: { recordId: newRecord.id, values: newRecord.values },
                  });
                }
                setContextMenu(null);
              }}
              className="dropdown-item w-full text-left"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
              </svg>
              Duplicate record
            </button>
            <hr className="my-1" />
            <button
              onClick={handleDeleteRow}
              className="dropdown-item dropdown-item-danger w-full text-left"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              Delete row
            </button>
          </div>
        </>
      )}

      {/* Record sidebar */}
      {expandedRecord && (
        <RecordSidebar
          record={expandedRecord}
          onClose={() => setExpandedRecord(null)}
          onNavigate={handleNavigateRecord}
          hasPrev={getRecordNavigation(expandedRecord).hasPrev}
          hasNext={getRecordNavigation(expandedRecord).hasNext}
          position={getRecordNavigation(expandedRecord).position}
          total={getRecordNavigation(expandedRecord).total}
        />
      )}

      {/* Keyboard shortcuts modal */}
      {showKeyboardShortcuts && (
        <KeyboardShortcutsModal
          onClose={() => setShowKeyboardShortcuts(false)}
        />
      )}
      </div>
    </div>
  );
}
