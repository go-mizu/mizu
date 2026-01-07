import { useCallback, useEffect, useMemo, useState } from 'react';
import DataEditor, {
  GridCell,
  GridCellKind,
  GridColumn,
  Item,
  EditableGridCell,
  GridSelection,
  CompactSelection,
} from '@glideapps/glide-data-grid';
import '@glideapps/glide-data-grid/dist/index.css';
import { useSpreadsheetStore } from './stores/spreadsheet';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { useUndoRedoStore, createCellSnapshot, createEmptySnapshot } from './hooks/useUndoRedo';
import { useClipboardStore, readSystemClipboard } from './hooks/useClipboard';
import { ContextMenu } from './components/ContextMenu';
import { FindReplaceDialog, FindOptions } from './components/FindReplaceDialog';
import { Toolbar } from './components/Toolbar';
import { MenuBar, Menu } from './components/MenuBar';
import { StatusBar } from './components/StatusBar';
import { SheetTabContextMenu } from './components/SheetTabContextMenu';
import type { Cell, CellFormat, CellPosition, Selection, Sheet } from './types';

// Generate column headers A, B, C, ..., Z, AA, AB, etc.
function getColumnLabel(index: number): string {
  let result = '';
  index++;
  while (index > 0) {
    index--;
    result = String.fromCharCode(65 + (index % 26)) + result;
    index = Math.floor(index / 26);
  }
  return result;
}

// Number of rows and columns
const NUM_ROWS = 1000;
const NUM_COLS = 26;

function App() {
  const {
    user,
    isAuthenticated,
    authLoading,
    currentWorkbook,
    currentSheet,
    sheets,
    cells,
    mergedRegions,
    activeCell,
    selection,
    formulaBarValue,
    loading,
    error,
    checkAuth,
    login,
    loadWorkbooks,
    loadWorkbook,
    createWorkbook,
    selectSheet,
    createSheet,
    getCell,
    setCell,
    setCellFormat,
    deleteCell,
    batchUpdateCells,
    insertRows,
    deleteRows,
    insertCols,
    deleteCols,
    mergeCells,
    unmergeCells,
    setActiveCell,
    setSelection,
    setFormulaBarValue,
    clearError,
  } = useSpreadsheetStore();

  const {
    pushAction,
    undo: undoAction,
    redo: redoAction,
    canUndo,
    canRedo,
    setUndoing,
    setRedoing,
  } = useUndoRedoStore();

  const {
    copy: clipboardCopy,
    cut: clipboardCut,
    getData: getClipboardData,
    clearCut,
  } = useClipboardStore();

  // Local state
  const [isEditing, setIsEditing] = useState(false);
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number } | null>(null);
  const [findDialogOpen, setFindDialogOpen] = useState(false);
  const [findDialogMode, setFindDialogMode] = useState<'find' | 'replace'>('find');
  const [columnWidths, setColumnWidths] = useState<Record<string, number>>({});
  const [zoom, setZoom] = useState(100);
  const [sheetTabContextMenu, setSheetTabContextMenu] = useState<{
    x: number;
    y: number;
    sheetId: string;
    sheetName: string;
  } | null>(null);
  const [editingSheetId, setEditingSheetId] = useState<string | null>(null);

  // Generate columns with custom widths
  const columns: GridColumn[] = useMemo(() =>
    Array.from({ length: NUM_COLS }, (_, i) => {
      const id = getColumnLabel(i);
      return {
        id,
        title: id,
        width: columnWidths[id] || 100,
      };
    }), [columnWidths]);

  // Current selection (single cell or range)
  const currentSelection: Selection | null = useMemo(() => {
    if (selection) return selection;
    if (activeCell) {
      return {
        startRow: activeCell.row,
        startCol: activeCell.col,
        endRow: activeCell.row,
        endCol: activeCell.col,
      };
    }
    return null;
  }, [selection, activeCell]);

  // Check auth on mount
  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  // Auto-login in dev mode
  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      login('dev@example.com', 'password').catch(() => {
        // Ignore login error in dev mode
      });
    }
  }, [authLoading, isAuthenticated, login]);

  // Load workbooks once authenticated
  useEffect(() => {
    if (isAuthenticated) {
      loadWorkbooks().then(() => {
        const store = useSpreadsheetStore.getState();
        if (store.workbooks.length > 0) {
          loadWorkbook(store.workbooks[0].id);
        } else {
          createWorkbook('My Spreadsheet').then((wb) => loadWorkbook(wb.id));
        }
      });
    }
  }, [isAuthenticated, loadWorkbooks, loadWorkbook, createWorkbook]);

  // Get cells in a selection range
  const getCellsInRange = useCallback((sel: Selection): Cell[] => {
    const result: Cell[] = [];
    for (let row = sel.startRow; row <= sel.endRow; row++) {
      for (let col = sel.startCol; col <= sel.endCol; col++) {
        const cell = getCell(row, col);
        if (cell) {
          result.push(cell);
        } else {
          // Create virtual cell for empty cells
          result.push({
            id: `${row}:${col}`,
            sheetId: currentSheet?.id || '',
            row,
            col,
            value: null,
            type: 'text',
            updatedAt: new Date().toISOString(),
          });
        }
      }
    }
    return result;
  }, [getCell, currentSheet]);

  // Undo operation
  const handleUndo = useCallback(async () => {
    const action = undoAction();
    if (!action || !currentSheet || action.sheetId !== currentSheet.id) return;

    setUndoing(true);
    try {
      // Restore previous state
      const updates = action.data.before.map(snapshot => ({
        row: snapshot.row,
        col: snapshot.col,
        value: snapshot.value,
        formula: snapshot.formula,
      }));

      if (updates.length === 1) {
        const u = updates[0];
        if (u.value === null && !u.formula) {
          await deleteCell(u.row, u.col);
        } else {
          await setCell(u.row, u.col, u.value, u.formula);
        }
      } else {
        await batchUpdateCells(updates);
      }
    } finally {
      setUndoing(false);
    }
  }, [undoAction, currentSheet, setUndoing, deleteCell, setCell, batchUpdateCells]);

  // Redo operation
  const handleRedo = useCallback(async () => {
    const action = redoAction();
    if (!action || !currentSheet || action.sheetId !== currentSheet.id) return;

    setRedoing(true);
    try {
      // Apply the changes again
      const updates = action.data.after.map(snapshot => ({
        row: snapshot.row,
        col: snapshot.col,
        value: snapshot.value,
        formula: snapshot.formula,
      }));

      if (updates.length === 1) {
        const u = updates[0];
        if (u.value === null && !u.formula) {
          await deleteCell(u.row, u.col);
        } else {
          await setCell(u.row, u.col, u.value, u.formula);
        }
      } else {
        await batchUpdateCells(updates);
      }
    } finally {
      setRedoing(false);
    }
  }, [redoAction, currentSheet, setRedoing, deleteCell, setCell, batchUpdateCells]);

  // Copy operation
  const handleCopy = useCallback(async () => {
    if (!currentSelection || !currentSheet) return;
    const cellsToClip = getCellsInRange(currentSelection);
    await clipboardCopy(cellsToClip, currentSelection, currentSheet.id);
  }, [currentSelection, currentSheet, getCellsInRange, clipboardCopy]);

  // Cut operation
  const handleCut = useCallback(async () => {
    if (!currentSelection || !currentSheet) return;
    const cellsToClip = getCellsInRange(currentSelection);
    await clipboardCut(cellsToClip, currentSelection, currentSheet.id);
  }, [currentSelection, currentSheet, getCellsInRange, clipboardCut]);

  // Paste operation
  const handlePaste = useCallback(async (valuesOnly: boolean = false) => {
    if (!activeCell || !currentSheet) return;

    const clipData = getClipboardData();

    if (clipData && clipData.source === 'internal') {
      // Paste from internal clipboard
      const offsetRow = activeCell.row - clipData.bounds.startRow;
      const offsetCol = activeCell.col - clipData.bounds.startCol;

      const beforeSnapshots = [];
      const afterSnapshots = [];

      for (const cell of clipData.cells) {
        const targetRow = cell.row + offsetRow;
        const targetCol = cell.col + offsetCol;

        if (targetRow >= 0 && targetRow < NUM_ROWS && targetCol >= 0 && targetCol < NUM_COLS) {
          const existingCell = getCell(targetRow, targetCol);
          beforeSnapshots.push(createCellSnapshot(existingCell, targetRow, targetCol));

          afterSnapshots.push({
            row: targetRow,
            col: targetCol,
            value: cell.value,
            formula: valuesOnly ? undefined : cell.formula,
            format: valuesOnly ? undefined : cell.format,
            display: cell.display,
          });
        }
      }

      // Batch update
      await batchUpdateCells(afterSnapshots.map(s => ({
        row: s.row,
        col: s.col,
        value: s.formula ? undefined : s.value,
        formula: s.formula,
      })));

      // Push undo action
      pushAction({
        type: 'batch_edit',
        sheetId: currentSheet.id,
        data: {
          before: beforeSnapshots,
          after: afterSnapshots,
        },
      });

      // If this was a cut operation, clear the source cells
      if (clipData.isCut && clipData.sourceSheetId === currentSheet.id) {
        const clearUpdates = clipData.cells.map(cell => ({
          row: cell.row,
          col: cell.col,
          value: null,
        }));
        await batchUpdateCells(clearUpdates);
        clearCut();
      }
    } else {
      // Try to read from system clipboard
      const systemData = await readSystemClipboard();
      if (systemData && systemData.length > 0) {
        const beforeSnapshots = [];
        const afterSnapshots = [];

        for (let rowIdx = 0; rowIdx < systemData.length; rowIdx++) {
          for (let colIdx = 0; colIdx < systemData[rowIdx].length; colIdx++) {
            const targetRow = activeCell.row + rowIdx;
            const targetCol = activeCell.col + colIdx;

            if (targetRow >= 0 && targetRow < NUM_ROWS && targetCol >= 0 && targetCol < NUM_COLS) {
              const existingCell = getCell(targetRow, targetCol);
              beforeSnapshots.push(createCellSnapshot(existingCell, targetRow, targetCol));

              afterSnapshots.push({
                row: targetRow,
                col: targetCol,
                value: systemData[rowIdx][colIdx],
              });
            }
          }
        }

        await batchUpdateCells(afterSnapshots.map(s => ({
          row: s.row,
          col: s.col,
          value: s.value,
        })));

        pushAction({
          type: 'batch_edit',
          sheetId: currentSheet.id,
          data: {
            before: beforeSnapshots,
            after: afterSnapshots,
          },
        });
      }
    }
  }, [activeCell, currentSheet, getClipboardData, getCell, batchUpdateCells, pushAction, clearCut]);

  // Delete/Clear selection
  const handleClearSelection = useCallback(async () => {
    if (!currentSelection || !currentSheet) return;

    const beforeSnapshots = [];
    const afterSnapshots = [];

    for (let row = currentSelection.startRow; row <= currentSelection.endRow; row++) {
      for (let col = currentSelection.startCol; col <= currentSelection.endCol; col++) {
        const existingCell = getCell(row, col);
        if (existingCell && existingCell.value !== null) {
          beforeSnapshots.push(createCellSnapshot(existingCell, row, col));
          afterSnapshots.push(createEmptySnapshot(row, col));
        }
      }
    }

    if (beforeSnapshots.length > 0) {
      await batchUpdateCells(afterSnapshots.map(s => ({
        row: s.row,
        col: s.col,
        value: null,
      })));

      pushAction({
        type: 'clear',
        sheetId: currentSheet.id,
        data: {
          before: beforeSnapshots,
          after: afterSnapshots,
        },
      });
    }
  }, [currentSelection, currentSheet, getCell, batchUpdateCells, pushAction]);

  // Format cells
  const handleFormatChange = useCallback(async (format: Partial<CellFormat>) => {
    if (!currentSelection || !currentSheet) return;

    for (let row = currentSelection.startRow; row <= currentSelection.endRow; row++) {
      for (let col = currentSelection.startCol; col <= currentSelection.endCol; col++) {
        const existingCell = getCell(row, col);
        const newFormat = { ...existingCell?.format, ...format };
        await setCellFormat(row, col, newFormat);
      }
    }
  }, [currentSelection, currentSheet, getCell, setCellFormat]);

  // Navigation helpers
  const moveActiveCell = useCallback((rowDelta: number, colDelta: number) => {
    if (!activeCell) {
      setActiveCell({ row: 0, col: 0 });
      return;
    }

    const newRow = Math.max(0, Math.min(NUM_ROWS - 1, activeCell.row + rowDelta));
    const newCol = Math.max(0, Math.min(NUM_COLS - 1, activeCell.col + colDelta));
    setActiveCell({ row: newRow, col: newCol });
    setSelection(null);
  }, [activeCell, setActiveCell, setSelection]);

  const extendSelection = useCallback((rowDelta: number, colDelta: number) => {
    if (!activeCell) return;

    const current = currentSelection || {
      startRow: activeCell.row,
      startCol: activeCell.col,
      endRow: activeCell.row,
      endCol: activeCell.col,
    };

    const newEndRow = Math.max(0, Math.min(NUM_ROWS - 1, current.endRow + rowDelta));
    const newEndCol = Math.max(0, Math.min(NUM_COLS - 1, current.endCol + colDelta));

    setSelection({
      startRow: current.startRow,
      startCol: current.startCol,
      endRow: newEndRow,
      endCol: newEndCol,
    });
  }, [activeCell, currentSelection, setSelection]);

  const selectAll = useCallback(() => {
    setSelection({
      startRow: 0,
      startCol: 0,
      endRow: NUM_ROWS - 1,
      endCol: NUM_COLS - 1,
    });
  }, [setSelection]);

  // Start editing current cell
  const startEditing = useCallback(() => {
    setIsEditing(true);
    // Focus formula bar or trigger cell editing
  }, []);

  // Cancel editing
  const cancelEditing = useCallback(() => {
    setIsEditing(false);
    setContextMenu(null);
    if (findDialogOpen) setFindDialogOpen(false);
  }, [findDialogOpen]);

  // Open find dialog
  const openFindDialog = useCallback(() => {
    setFindDialogMode('find');
    setFindDialogOpen(true);
  }, []);

  const openReplaceDialog = useCallback(() => {
    setFindDialogMode('replace');
    setFindDialogOpen(true);
  }, []);

  // Sheet tab context menu handlers
  const handleSheetTabContextMenu = useCallback((event: React.MouseEvent, sheet: Sheet) => {
    event.preventDefault();
    event.stopPropagation();
    setSheetTabContextMenu({
      x: event.clientX,
      y: event.clientY,
      sheetId: sheet.id,
      sheetName: sheet.name,
    });
  }, []);

  const handleDeleteSheet = useCallback(async (sheetId: string) => {
    // Would call deleteSheet from store
    console.log('Delete sheet:', sheetId);
  }, []);

  const handleDuplicateSheet = useCallback(async (sheetId: string) => {
    const sheet = sheets.find(s => s.id === sheetId);
    if (sheet) {
      await createSheet(`${sheet.name} (Copy)`);
    }
  }, [sheets, createSheet]);

  const handleRenameSheet = useCallback((sheetId: string) => {
    setEditingSheetId(sheetId);
  }, []);

  const handleChangeSheetColor = useCallback((sheetId: string, color: string) => {
    // Would call updateSheet with color
    console.log('Change sheet color:', sheetId, color);
  }, []);

  const handleHideSheet = useCallback((sheetId: string) => {
    // Would call updateSheet with hidden: true
    console.log('Hide sheet:', sheetId);
  }, []);

  // Zoom handler
  const handleZoomChange = useCallback((newZoom: number) => {
    setZoom(Math.max(50, Math.min(200, newZoom)));
  }, []);

  // Find functionality
  const handleFindAll = useCallback((searchText: string, options: FindOptions): CellPosition[] => {
    const results: CellPosition[] = [];
    const searchLower = options.matchCase ? searchText : searchText.toLowerCase();

    cells.forEach((cell) => {
      const valueToSearch = options.searchIn === 'formulas'
        ? (cell.formula || '')
        : String(cell.display ?? cell.value ?? '');

      const compareValue = options.matchCase ? valueToSearch : valueToSearch.toLowerCase();

      let matches = false;
      if (options.useRegex) {
        try {
          const regex = new RegExp(searchText, options.matchCase ? '' : 'i');
          matches = options.matchEntireCell
            ? regex.test(valueToSearch) && valueToSearch.match(regex)?.[0] === valueToSearch
            : regex.test(valueToSearch);
        } catch {
          // Invalid regex
        }
      } else {
        matches = options.matchEntireCell
          ? compareValue === searchLower
          : compareValue.includes(searchLower);
      }

      if (matches) {
        results.push({ row: cell.row, col: cell.col });
      }
    });

    // Sort by row then column
    results.sort((a, b) => a.row - b.row || a.col - b.col);
    return results;
  }, [cells]);

  const handleFind = useCallback((searchText: string, options: FindOptions): CellPosition | null => {
    const results = handleFindAll(searchText, options);
    return results.length > 0 ? results[0] : null;
  }, [handleFindAll]);

  const handleReplace = useCallback(async (findText: string, replaceText: string, options: FindOptions): Promise<boolean> => {
    if (!activeCell || !currentSheet) return false;

    const cell = getCell(activeCell.row, activeCell.col);
    if (!cell) return false;

    const valueToSearch = options.searchIn === 'formulas'
      ? (cell.formula || '')
      : String(cell.display ?? cell.value ?? '');

    let newValue: string;
    if (options.useRegex) {
      try {
        const regex = new RegExp(findText, options.matchCase ? 'g' : 'gi');
        newValue = valueToSearch.replace(regex, replaceText);
      } catch {
        return false;
      }
    } else {
      const searchPattern = options.matchCase ? findText : findText.toLowerCase();
      const compareValue = options.matchCase ? valueToSearch : valueToSearch.toLowerCase();

      if (options.matchEntireCell && compareValue !== searchPattern) {
        return false;
      }

      if (!compareValue.includes(searchPattern)) {
        return false;
      }

      newValue = valueToSearch.replace(
        new RegExp(findText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), options.matchCase ? 'g' : 'gi'),
        replaceText
      );
    }

    const beforeSnapshot = createCellSnapshot(cell, activeCell.row, activeCell.col);

    if (options.searchIn === 'formulas' && cell.formula) {
      await setCell(activeCell.row, activeCell.col, null, newValue);
    } else {
      await setCell(activeCell.row, activeCell.col, newValue);
    }

    pushAction({
      type: 'cell_edit',
      sheetId: currentSheet.id,
      data: {
        before: [beforeSnapshot],
        after: [{ row: activeCell.row, col: activeCell.col, value: newValue }],
      },
    });

    return true;
  }, [activeCell, currentSheet, getCell, setCell, pushAction]);

  const handleReplaceAll = useCallback(async (findText: string, replaceText: string, options: FindOptions): Promise<number> => {
    const results = handleFindAll(findText, options);
    if (results.length === 0 || !currentSheet) return 0;

    const beforeSnapshots = [];
    const afterSnapshots = [];

    for (const pos of results) {
      const cell = getCell(pos.row, pos.col);
      if (!cell) continue;

      const valueToSearch = options.searchIn === 'formulas'
        ? (cell.formula || '')
        : String(cell.display ?? cell.value ?? '');

      let newValue: string;
      if (options.useRegex) {
        try {
          const regex = new RegExp(findText, options.matchCase ? 'g' : 'gi');
          newValue = valueToSearch.replace(regex, replaceText);
        } catch {
          continue;
        }
      } else {
        newValue = valueToSearch.replace(
          new RegExp(findText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), options.matchCase ? 'g' : 'gi'),
          replaceText
        );
      }

      beforeSnapshots.push(createCellSnapshot(cell, pos.row, pos.col));
      afterSnapshots.push({
        row: pos.row,
        col: pos.col,
        value: options.searchIn === 'formulas' ? null : newValue,
        formula: options.searchIn === 'formulas' ? newValue : undefined,
      });
    }

    await batchUpdateCells(afterSnapshots.map(s => ({
      row: s.row,
      col: s.col,
      value: s.formula ? undefined : s.value,
      formula: s.formula,
    })));

    pushAction({
      type: 'batch_edit',
      sheetId: currentSheet.id,
      data: {
        before: beforeSnapshots,
        after: afterSnapshots,
      },
    });

    return beforeSnapshots.length;
  }, [handleFindAll, currentSheet, getCell, batchUpdateCells, pushAction]);

  const navigateToResult = useCallback((position: CellPosition) => {
    setActiveCell(position);
    setSelection(null);
  }, [setActiveCell, setSelection]);

  // Row/Column operations
  const handleInsertRowAbove = useCallback(async () => {
    if (!currentSelection) return;
    await insertRows(currentSelection.startRow, 1);
  }, [currentSelection, insertRows]);

  const handleInsertRowBelow = useCallback(async () => {
    if (!currentSelection) return;
    await insertRows(currentSelection.endRow + 1, 1);
  }, [currentSelection, insertRows]);

  const handleDeleteRow = useCallback(async () => {
    if (!currentSelection) return;
    const count = currentSelection.endRow - currentSelection.startRow + 1;
    await deleteRows(currentSelection.startRow, count);
  }, [currentSelection, deleteRows]);

  const handleInsertColLeft = useCallback(async () => {
    if (!currentSelection) return;
    await insertCols(currentSelection.startCol, 1);
  }, [currentSelection, insertCols]);

  const handleInsertColRight = useCallback(async () => {
    if (!currentSelection) return;
    await insertCols(currentSelection.endCol + 1, 1);
  }, [currentSelection, insertCols]);

  const handleDeleteCol = useCallback(async () => {
    if (!currentSelection) return;
    const count = currentSelection.endCol - currentSelection.startCol + 1;
    await deleteCols(currentSelection.startCol, count);
  }, [currentSelection, deleteCols]);

  // Merge cells
  const handleMergeCells = useCallback(async () => {
    if (!currentSelection) return;
    await mergeCells(
      currentSelection.startRow,
      currentSelection.startCol,
      currentSelection.endRow,
      currentSelection.endCol
    );
  }, [currentSelection, mergeCells]);

  const handleUnmergeCells = useCallback(async () => {
    if (!currentSelection) return;
    await unmergeCells(
      currentSelection.startRow,
      currentSelection.startCol,
      currentSelection.endRow,
      currentSelection.endCol
    );
  }, [currentSelection, unmergeCells]);

  // Check if current selection can be merged
  const canMerge = useMemo(() => {
    if (!currentSelection) return false;
    return currentSelection.startRow !== currentSelection.endRow ||
           currentSelection.startCol !== currentSelection.endCol;
  }, [currentSelection]);

  // Check if current selection has merged cells
  const hasMergedCells = useMemo(() => {
    if (!currentSelection || !mergedRegions) return false;
    return mergedRegions.some(region =>
      region.startRow === currentSelection.startRow &&
      region.startCol === currentSelection.startCol &&
      region.endRow === currentSelection.endRow &&
      region.endCol === currentSelection.endCol
    );
  }, [currentSelection, mergedRegions]);

  // Get current cell format
  const currentCellFormat = useMemo(() => {
    if (!activeCell) return undefined;
    const cell = getCell(activeCell.row, activeCell.col);
    return cell?.format;
  }, [activeCell, getCell]);

  // Keyboard shortcuts
  useKeyboardShortcuts({
    onMoveUp: () => moveActiveCell(-1, 0),
    onMoveDown: () => moveActiveCell(1, 0),
    onMoveLeft: () => moveActiveCell(0, -1),
    onMoveRight: () => moveActiveCell(0, 1),
    onTab: () => moveActiveCell(0, 1),
    onShiftTab: () => moveActiveCell(0, -1),
    onMoveToStart: () => setActiveCell({ row: 0, col: 0 }),
    onMoveToEnd: () => {
      // Find last used cell
      let maxRow = 0, maxCol = 0;
      cells.forEach((cell) => {
        if (cell.value !== null) {
          maxRow = Math.max(maxRow, cell.row);
          maxCol = Math.max(maxCol, cell.col);
        }
      });
      setActiveCell({ row: maxRow, col: maxCol });
    },
    onExtendUp: () => extendSelection(-1, 0),
    onExtendDown: () => extendSelection(1, 0),
    onExtendLeft: () => extendSelection(0, -1),
    onExtendRight: () => extendSelection(0, 1),
    onSelectAll: selectAll,
    onEdit: startEditing,
    onDelete: handleClearSelection,
    onEscape: cancelEditing,
    onEnter: () => {
      if (isEditing) {
        setIsEditing(false);
        moveActiveCell(1, 0);
      } else {
        startEditing();
      }
    },
    onCopy: handleCopy,
    onCut: handleCut,
    onPaste: () => handlePaste(false),
    onPasteValues: () => handlePaste(true),
    onUndo: handleUndo,
    onRedo: handleRedo,
    onBold: () => handleFormatChange({ bold: !currentCellFormat?.bold }),
    onItalic: () => handleFormatChange({ italic: !currentCellFormat?.italic }),
    onUnderline: () => handleFormatChange({ underline: !currentCellFormat?.underline }),
    onStrikethrough: () => handleFormatChange({ strikethrough: !currentCellFormat?.strikethrough }),
    onFind: openFindDialog,
    onReplace: openReplaceDialog,
    isEditing: () => isEditing,
  }, true);

  // Get cell content for grid
  const getContent = useCallback(
    (cell: Item): GridCell => {
      const [col, row] = cell;
      const cellData = getCell(row, col);

      if (!cellData) {
        return {
          kind: GridCellKind.Text,
          allowOverlay: true,
          displayData: '',
          data: '',
        };
      }

      // Apply formatting to display
      const themeOverride: Record<string, unknown> = {};
      if (cellData.format) {
        if (cellData.format.bold) themeOverride.baseFontStyle = 'bold';
        if (cellData.format.fontColor) themeOverride.textDark = cellData.format.fontColor;
        if (cellData.format.backgroundColor) themeOverride.bgCell = cellData.format.backgroundColor;
      }

      return {
        kind: GridCellKind.Text,
        allowOverlay: true,
        displayData: cellData.display || String(cellData.value ?? ''),
        data: cellData.formula || String(cellData.value ?? ''),
        themeOverride: Object.keys(themeOverride).length > 0 ? themeOverride : undefined,
      };
    },
    [getCell]
  );

  // Handle cell edit
  const onCellEdited = useCallback(
    async (cell: Item, newValue: EditableGridCell) => {
      if (newValue.kind !== GridCellKind.Text || !currentSheet) return;

      const [col, row] = cell;
      const value = newValue.data;
      const existingCell = getCell(row, col);
      const beforeSnapshot = createCellSnapshot(existingCell, row, col);

      if (value === '') {
        await setCell(row, col, null);
        pushAction({
          type: 'cell_edit',
          sheetId: currentSheet.id,
          data: {
            before: [beforeSnapshot],
            after: [createEmptySnapshot(row, col)],
          },
        });
      } else if (value.startsWith('=')) {
        await setCell(row, col, null, value);
        pushAction({
          type: 'cell_edit',
          sheetId: currentSheet.id,
          data: {
            before: [beforeSnapshot],
            after: [{ row, col, value: null, formula: value }],
          },
        });
      } else {
        const num = parseFloat(value);
        const finalValue = !isNaN(num) && isFinite(num) ? num : value;
        await setCell(row, col, finalValue);
        pushAction({
          type: 'cell_edit',
          sheetId: currentSheet.id,
          data: {
            before: [beforeSnapshot],
            after: [{ row, col, value: finalValue }],
          },
        });
      }
    },
    [currentSheet, getCell, setCell, pushAction]
  );

  // Handle selection change
  const onSelectionChange = useCallback(
    (sel: GridSelection) => {
      if (sel.current?.cell) {
        const [col, row] = sel.current.cell;
        setActiveCell({ row, col });

        // Check if there's a range selection
        if (sel.current.range && (sel.current.range.width > 1 || sel.current.range.height > 1)) {
          setSelection({
            startRow: sel.current.range.y,
            startCol: sel.current.range.x,
            endRow: sel.current.range.y + sel.current.range.height - 1,
            endCol: sel.current.range.x + sel.current.range.width - 1,
          });
        } else {
          setSelection(null);
        }
      }
    },
    [setActiveCell, setSelection]
  );

  // Handle right-click context menu
  const handleContextMenu = useCallback((event: React.MouseEvent) => {
    event.preventDefault();
    setContextMenu({ x: event.clientX, y: event.clientY });
  }, []);

  // Handle formula bar change
  const handleFormulaBarChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setFormulaBarValue(e.target.value);
    },
    [setFormulaBarValue]
  );

  // Handle formula bar submit
  const handleFormulaBarSubmit = useCallback(
    async (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter' && activeCell && currentSheet) {
        const value = formulaBarValue;
        const existingCell = getCell(activeCell.row, activeCell.col);
        const beforeSnapshot = createCellSnapshot(existingCell, activeCell.row, activeCell.col);

        if (value === '') {
          await setCell(activeCell.row, activeCell.col, null);
          pushAction({
            type: 'cell_edit',
            sheetId: currentSheet.id,
            data: {
              before: [beforeSnapshot],
              after: [createEmptySnapshot(activeCell.row, activeCell.col)],
            },
          });
        } else if (value.startsWith('=')) {
          await setCell(activeCell.row, activeCell.col, null, value);
          pushAction({
            type: 'cell_edit',
            sheetId: currentSheet.id,
            data: {
              before: [beforeSnapshot],
              after: [{ row: activeCell.row, col: activeCell.col, value: null, formula: value }],
            },
          });
        } else {
          const num = parseFloat(value);
          const finalValue = !isNaN(num) && isFinite(num) ? num : value;
          await setCell(activeCell.row, activeCell.col, finalValue);
          pushAction({
            type: 'cell_edit',
            sheetId: currentSheet.id,
            data: {
              before: [beforeSnapshot],
              after: [{ row: activeCell.row, col: activeCell.col, value: finalValue }],
            },
          });
        }
      }
    },
    [activeCell, currentSheet, formulaBarValue, getCell, setCell, pushAction]
  );

  // Handle column resize
  const handleColumnResize = useCallback((column: GridColumn, newSize: number) => {
    setColumnWidths(prev => ({
      ...prev,
      [column.id as string]: newSize,
    }));
  }, []);

  // Active cell reference string
  const activeCellRef = useMemo(() => {
    if (!activeCell) return 'A1';
    return `${getColumnLabel(activeCell.col)}${activeCell.row + 1}`;
  }, [activeCell]);

  // Grid selection state
  const gridSelection: GridSelection = useMemo(() => {
    if (!activeCell) {
      return {
        columns: CompactSelection.empty(),
        rows: CompactSelection.empty(),
      };
    }

    const sel = currentSelection || {
      startRow: activeCell.row,
      startCol: activeCell.col,
      endRow: activeCell.row,
      endCol: activeCell.col,
    };

    return {
      columns: CompactSelection.empty(),
      rows: CompactSelection.empty(),
      current: {
        cell: [activeCell.col, activeCell.row],
        range: {
          x: sel.startCol,
          y: sel.startRow,
          width: sel.endCol - sel.startCol + 1,
          height: sel.endRow - sel.startRow + 1,
        },
        rangeStack: [],
      },
    };
  }, [activeCell, currentSelection]);

  // Menu bar configuration
  const menus: Menu[] = useMemo(() => [
    {
      id: 'file',
      label: 'File',
      items: [
        { id: 'new', label: 'New', shortcut: 'Ctrl+N', action: () => createWorkbook('Untitled Spreadsheet') },
        { id: 'divider1', label: '', divider: true },
        { id: 'download', label: 'Download', submenu: [
          { id: 'xlsx', label: 'Microsoft Excel (.xlsx)', action: () => console.log('Download XLSX') },
          { id: 'csv', label: 'CSV (.csv)', action: () => console.log('Download CSV') },
          { id: 'pdf', label: 'PDF (.pdf)', action: () => console.log('Download PDF') },
        ]},
        { id: 'divider2', label: '', divider: true },
        { id: 'print', label: 'Print', shortcut: 'Ctrl+P', action: () => window.print() },
      ],
    },
    {
      id: 'edit',
      label: 'Edit',
      items: [
        { id: 'undo', label: 'Undo', shortcut: 'Ctrl+Z', action: handleUndo, disabled: !canUndo() },
        { id: 'redo', label: 'Redo', shortcut: 'Ctrl+Y', action: handleRedo, disabled: !canRedo() },
        { id: 'divider1', label: '', divider: true },
        { id: 'cut', label: 'Cut', shortcut: 'Ctrl+X', action: handleCut },
        { id: 'copy', label: 'Copy', shortcut: 'Ctrl+C', action: handleCopy },
        { id: 'paste', label: 'Paste', shortcut: 'Ctrl+V', action: () => handlePaste(false) },
        { id: 'paste-values', label: 'Paste values only', shortcut: 'Ctrl+Shift+V', action: () => handlePaste(true) },
        { id: 'divider2', label: '', divider: true },
        { id: 'find', label: 'Find and replace', shortcut: 'Ctrl+H', action: openReplaceDialog },
        { id: 'divider3', label: '', divider: true },
        { id: 'delete', label: 'Delete values', action: handleClearSelection },
      ],
    },
    {
      id: 'view',
      label: 'View',
      items: [
        { id: 'freeze', label: 'Freeze', submenu: [
          { id: 'no-rows', label: 'No rows', action: () => console.log('Unfreeze rows') },
          { id: '1-row', label: '1 row', action: () => console.log('Freeze 1 row') },
          { id: '2-rows', label: '2 rows', action: () => console.log('Freeze 2 rows') },
        ]},
        { id: 'divider1', label: '', divider: true },
        { id: 'zoom', label: 'Zoom', submenu: [
          { id: 'zoom-50', label: '50%', action: () => handleZoomChange(50) },
          { id: 'zoom-75', label: '75%', action: () => handleZoomChange(75) },
          { id: 'zoom-100', label: '100%', action: () => handleZoomChange(100) },
          { id: 'zoom-125', label: '125%', action: () => handleZoomChange(125) },
          { id: 'zoom-150', label: '150%', action: () => handleZoomChange(150) },
          { id: 'zoom-200', label: '200%', action: () => handleZoomChange(200) },
        ]},
      ],
    },
    {
      id: 'insert',
      label: 'Insert',
      items: [
        { id: 'row-above', label: 'Row above', action: handleInsertRowAbove },
        { id: 'row-below', label: 'Row below', action: handleInsertRowBelow },
        { id: 'divider1', label: '', divider: true },
        { id: 'col-left', label: 'Column left', action: handleInsertColLeft },
        { id: 'col-right', label: 'Column right', action: handleInsertColRight },
      ],
    },
    {
      id: 'format',
      label: 'Format',
      items: [
        { id: 'number', label: 'Number', submenu: [
          { id: 'auto', label: 'Automatic', action: () => handleFormatChange({ numberFormat: '' }) },
          { id: 'plain', label: 'Plain text', action: () => handleFormatChange({ numberFormat: '@' }) },
          { id: 'number', label: 'Number', action: () => handleFormatChange({ numberFormat: '#,##0.00' }) },
          { id: 'currency', label: 'Currency', action: () => handleFormatChange({ numberFormat: '$#,##0.00' }) },
          { id: 'percent', label: 'Percent', action: () => handleFormatChange({ numberFormat: '0.00%' }) },
        ]},
        { id: 'divider1', label: '', divider: true },
        { id: 'bold', label: 'Bold', shortcut: 'Ctrl+B', action: () => handleFormatChange({ bold: !currentCellFormat?.bold }) },
        { id: 'italic', label: 'Italic', shortcut: 'Ctrl+I', action: () => handleFormatChange({ italic: !currentCellFormat?.italic }) },
        { id: 'underline', label: 'Underline', shortcut: 'Ctrl+U', action: () => handleFormatChange({ underline: !currentCellFormat?.underline }) },
        { id: 'strikethrough', label: 'Strikethrough', action: () => handleFormatChange({ strikethrough: !currentCellFormat?.strikethrough }) },
        { id: 'divider2', label: '', divider: true },
        { id: 'align', label: 'Align', submenu: [
          { id: 'left', label: 'Left', action: () => handleFormatChange({ hAlign: 'left' }) },
          { id: 'center', label: 'Center', action: () => handleFormatChange({ hAlign: 'center' }) },
          { id: 'right', label: 'Right', action: () => handleFormatChange({ hAlign: 'right' }) },
        ]},
        { id: 'valign', label: 'Vertical align', submenu: [
          { id: 'top', label: 'Top', action: () => handleFormatChange({ vAlign: 'top' }) },
          { id: 'middle', label: 'Middle', action: () => handleFormatChange({ vAlign: 'middle' }) },
          { id: 'bottom', label: 'Bottom', action: () => handleFormatChange({ vAlign: 'bottom' }) },
        ]},
        { id: 'divider3', label: '', divider: true },
        { id: 'merge', label: 'Merge cells', action: handleMergeCells, disabled: !canMerge },
        { id: 'unmerge', label: 'Unmerge cells', action: handleUnmergeCells, disabled: !hasMergedCells },
      ],
    },
    {
      id: 'data',
      label: 'Data',
      items: [
        { id: 'sort-az', label: 'Sort sheet A to Z', action: () => console.log('Sort A-Z') },
        { id: 'sort-za', label: 'Sort sheet Z to A', action: () => console.log('Sort Z-A') },
      ],
    },
    {
      id: 'tools',
      label: 'Tools',
      items: [
        { id: 'autocomplete', label: 'Enable autocomplete', checked: true, action: () => console.log('Toggle autocomplete') },
      ],
    },
    {
      id: 'help',
      label: 'Help',
      items: [
        { id: 'shortcuts', label: 'Keyboard shortcuts', action: () => console.log('Show shortcuts') },
        { id: 'divider1', label: '', divider: true },
        { id: 'help', label: 'Help', action: () => console.log('Show help') },
      ],
    },
  ], [
    createWorkbook, handleUndo, handleRedo, canUndo, canRedo, handleCut, handleCopy, handlePaste,
    openReplaceDialog, handleClearSelection, handleZoomChange, handleInsertRowAbove, handleInsertRowBelow,
    handleInsertColLeft, handleInsertColRight, handleFormatChange, currentCellFormat, handleMergeCells,
    handleUnmergeCells, canMerge, hasMergedCells
  ]);

  if (authLoading) {
    return (
      <div className="app loading">
        <div className="loading-spinner">Loading...</div>
      </div>
    );
  }

  return (
    <div className="app" onContextMenu={handleContextMenu}>
      <header className="header">
        <div className="header-left">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="logo">
            <rect x="3" y="3" width="18" height="18" rx="2" />
            <path d="M3 9h18M9 3v18" />
          </svg>
          <span className="title">{currentWorkbook?.name || 'Untitled Spreadsheet'}</span>
          {loading && <span className="loading-indicator">Saving...</span>}
        </div>
        <div className="header-right">
          {user && <span className="user-name">{user.name}</span>}
          <button className="share-btn">Share</button>
        </div>
      </header>

      <MenuBar menus={menus} />

      {error && (
        <div className="error-banner">
          <span>{error}</span>
          <button onClick={clearError}>Dismiss</button>
        </div>
      )}

      <Toolbar
        onUndo={handleUndo}
        onRedo={handleRedo}
        canUndo={canUndo()}
        canRedo={canRedo()}
        onFormatChange={handleFormatChange}
        currentFormat={currentCellFormat}
        onFind={openFindDialog}
        onMergeCells={handleMergeCells}
        onUnmergeCells={handleUnmergeCells}
        canMerge={canMerge}
        hasMergedCells={hasMergedCells}
        zoom={zoom}
        onZoomChange={handleZoomChange}
      />

      <div className="formula-bar">
        <div className="cell-ref">{activeCellRef}</div>
        <div className="formula-input-container">
          <span className="fx">fx</span>
          <input
            type="text"
            className="formula-input"
            placeholder="Enter formula or value"
            value={formulaBarValue}
            onChange={handleFormulaBarChange}
            onKeyDown={handleFormulaBarSubmit}
          />
        </div>
      </div>

      <div className="grid-container" style={{ transform: `scale(${zoom / 100})`, transformOrigin: 'top left', width: `${10000 / zoom}%`, height: `${10000 / zoom}%` }}>
        <DataEditor
          getCellContent={getContent}
          columns={columns}
          rows={NUM_ROWS}
          onCellEdited={onCellEdited}
          gridSelection={gridSelection}
          onGridSelectionChange={onSelectionChange}
          onColumnResize={handleColumnResize}
          rowMarkers="number"
          rangeSelect="rect"
          columnSelect="single"
          rowSelect="single"
          smoothScrollX
          smoothScrollY
          getCellsForSelection={true}
          theme={{
            bgCell: '#ffffff',
            bgHeader: '#f8f9fa',
            bgHeaderHovered: '#e8eaed',
            textHeader: '#5f6368',
            borderColor: '#e2e2e2',
            accentColor: '#1a73e8',
            accentLight: '#e8f0fe',
          }}
        />
      </div>

      <StatusBar
        cells={cells}
        selection={currentSelection}
        zoom={zoom}
        onZoomChange={handleZoomChange}
      />

      <footer className="sheet-tabs">
        <button className="add-sheet" onClick={() => createSheet(`Sheet${sheets.length + 1}`)}>
          +
        </button>
        {sheets.map((sheet) => (
          <div
            key={sheet.id}
            className={`sheet-tab ${currentSheet?.id === sheet.id ? 'active' : ''}`}
            onClick={() => selectSheet(sheet.id)}
            onContextMenu={(e) => handleSheetTabContextMenu(e, sheet)}
            onDoubleClick={() => setEditingSheetId(sheet.id)}
          >
            {editingSheetId === sheet.id ? (
              <input
                type="text"
                className="sheet-tab-name"
                defaultValue={sheet.name}
                autoFocus
                onBlur={() => setEditingSheetId(null)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    setEditingSheetId(null);
                  } else if (e.key === 'Escape') {
                    setEditingSheetId(null);
                  }
                }}
              />
            ) : (
              sheet.name
            )}
            {sheet.color && (
              <div className="sheet-tab-color" style={{ backgroundColor: sheet.color }} />
            )}
          </div>
        ))}
      </footer>

      {/* Context Menu */}
      {contextMenu && currentSelection && (
        <ContextMenu
          position={contextMenu}
          selection={currentSelection}
          onClose={() => setContextMenu(null)}
          onCut={handleCut}
          onCopy={handleCopy}
          onPaste={() => handlePaste(false)}
          onPasteValuesOnly={() => handlePaste(true)}
          onClearContents={handleClearSelection}
          onInsertRowAbove={handleInsertRowAbove}
          onInsertRowBelow={handleInsertRowBelow}
          onInsertColLeft={handleInsertColLeft}
          onInsertColRight={handleInsertColRight}
          onDeleteRow={handleDeleteRow}
          onDeleteCol={handleDeleteCol}
          onMergeCells={handleMergeCells}
          onUnmergeCells={handleUnmergeCells}
          hasMergedCells={hasMergedCells}
          canMerge={canMerge}
        />
      )}

      {/* Find & Replace Dialog */}
      <FindReplaceDialog
        isOpen={findDialogOpen}
        mode={findDialogMode}
        onClose={() => setFindDialogOpen(false)}
        onFind={handleFind}
        onFindAll={handleFindAll}
        onReplace={handleReplace}
        onReplaceAll={handleReplaceAll}
        onNavigateToResult={navigateToResult}
      />

      {/* Sheet Tab Context Menu */}
      {sheetTabContextMenu && (
        <SheetTabContextMenu
          position={{ x: sheetTabContextMenu.x, y: sheetTabContextMenu.y }}
          sheetId={sheetTabContextMenu.sheetId}
          sheetName={sheetTabContextMenu.sheetName}
          onClose={() => setSheetTabContextMenu(null)}
          onDelete={handleDeleteSheet}
          onDuplicate={handleDuplicateSheet}
          onRename={handleRenameSheet}
          onChangeColor={handleChangeSheetColor}
          onHide={handleHideSheet}
          canDelete={sheets.length > 1}
        />
      )}
    </div>
  );
}

export default App;
