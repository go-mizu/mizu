import { create } from 'zustand';
import type { Cell, CellValue, CellFormat, Selection } from '../types';

export interface UndoAction {
  type: 'cell_edit' | 'cell_format' | 'cell_delete' | 'batch_edit' |
        'row_insert' | 'row_delete' | 'col_insert' | 'col_delete' |
        'merge' | 'unmerge' | 'clear';
  timestamp: number;
  sheetId: string;
  data: {
    before: CellSnapshot[];
    after: CellSnapshot[];
    selection?: Selection;
  };
}

export interface CellSnapshot {
  row: number;
  col: number;
  value: CellValue;
  formula?: string;
  format?: CellFormat;
  display?: string;
}

interface UndoRedoState {
  undoStack: UndoAction[];
  redoStack: UndoAction[];
  maxStackSize: number;
  isUndoing: boolean;
  isRedoing: boolean;

  // Actions
  pushAction: (action: Omit<UndoAction, 'timestamp'>) => void;
  undo: () => UndoAction | null;
  redo: () => UndoAction | null;
  canUndo: () => boolean;
  canRedo: () => boolean;
  clearHistory: () => void;
  setUndoing: (value: boolean) => void;
  setRedoing: (value: boolean) => void;
}

export const useUndoRedoStore = create<UndoRedoState>((set, get) => ({
  undoStack: [],
  redoStack: [],
  maxStackSize: 100,
  isUndoing: false,
  isRedoing: false,

  pushAction: (action) => {
    const { isUndoing, isRedoing } = get();

    // Don't push actions during undo/redo operations
    if (isUndoing || isRedoing) return;

    const fullAction: UndoAction = {
      ...action,
      timestamp: Date.now(),
    };

    set((state) => {
      const newStack = [...state.undoStack, fullAction];

      // Trim to max size
      if (newStack.length > state.maxStackSize) {
        newStack.shift();
      }

      return {
        undoStack: newStack,
        redoStack: [], // Clear redo stack on new action
      };
    });
  },

  undo: () => {
    const { undoStack } = get();
    if (undoStack.length === 0) return null;

    const action = undoStack[undoStack.length - 1];

    set((state) => ({
      undoStack: state.undoStack.slice(0, -1),
      redoStack: [...state.redoStack, action],
    }));

    return action;
  },

  redo: () => {
    const { redoStack } = get();
    if (redoStack.length === 0) return null;

    const action = redoStack[redoStack.length - 1];

    set((state) => ({
      redoStack: state.redoStack.slice(0, -1),
      undoStack: [...state.undoStack, action],
    }));

    return action;
  },

  canUndo: () => get().undoStack.length > 0,
  canRedo: () => get().redoStack.length > 0,

  clearHistory: () => set({ undoStack: [], redoStack: [] }),

  setUndoing: (value) => set({ isUndoing: value }),
  setRedoing: (value) => set({ isRedoing: value }),
}));

// Helper to create cell snapshot from Cell
export function createCellSnapshot(cell: Cell | undefined, row: number, col: number): CellSnapshot {
  if (!cell) {
    return { row, col, value: null };
  }
  return {
    row,
    col,
    value: cell.value,
    formula: cell.formula,
    format: cell.format,
    display: cell.display,
  };
}

// Helper to create empty snapshot for deleted/cleared cells
export function createEmptySnapshot(row: number, col: number): CellSnapshot {
  return { row, col, value: null };
}
