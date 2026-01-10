import { create } from 'zustand';
import type { CellValue } from '../types';

export type UndoableActionType =
  | 'cell_update'
  | 'record_create'
  | 'record_delete'
  | 'bulk_cell_update'
  | 'bulk_record_delete';

interface CellUpdatePayload {
  recordId: string;
  fieldId: string;
  oldValue: CellValue;
  newValue: CellValue;
}

interface RecordCreatePayload {
  recordId: string;
  values: Record<string, CellValue>;
}

interface RecordDeletePayload {
  recordId: string;
  values: Record<string, CellValue>;
  position: number;
}

interface BulkCellUpdatePayload {
  updates: CellUpdatePayload[];
}

interface BulkRecordDeletePayload {
  records: RecordDeletePayload[];
}

export type UndoableActionPayload =
  | CellUpdatePayload
  | RecordCreatePayload
  | RecordDeletePayload
  | BulkCellUpdatePayload
  | BulkRecordDeletePayload;

export interface UndoableAction {
  type: UndoableActionType;
  payload: UndoableActionPayload;
  timestamp: number;
}

interface HistoryState {
  undoStack: UndoableAction[];
  redoStack: UndoableAction[];
  maxStackSize: number;

  // Actions
  pushAction: (action: Omit<UndoableAction, 'timestamp'>) => void;
  undo: () => UndoableAction | null;
  redo: () => UndoableAction | null;
  canUndo: () => boolean;
  canRedo: () => boolean;
  clear: () => void;
}

export const useHistoryStore = create<HistoryState>((set, get) => ({
  undoStack: [],
  redoStack: [],
  maxStackSize: 100,

  pushAction: (action) => {
    set((state) => {
      const newAction: UndoableAction = {
        ...action,
        timestamp: Date.now(),
      };

      let newUndoStack = [...state.undoStack, newAction];

      // Limit stack size
      if (newUndoStack.length > state.maxStackSize) {
        newUndoStack = newUndoStack.slice(-state.maxStackSize);
      }

      return {
        undoStack: newUndoStack,
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

  clear: () => set({ undoStack: [], redoStack: [] }),
}));

/**
 * Create the inverse action for undo
 */
export function createInverseAction(action: UndoableAction): UndoableAction {
  switch (action.type) {
    case 'cell_update': {
      const payload = action.payload as CellUpdatePayload;
      return {
        type: 'cell_update',
        payload: {
          recordId: payload.recordId,
          fieldId: payload.fieldId,
          oldValue: payload.newValue,
          newValue: payload.oldValue,
        },
        timestamp: Date.now(),
      };
    }
    case 'record_create': {
      const payload = action.payload as RecordCreatePayload;
      return {
        type: 'record_delete',
        payload: {
          recordId: payload.recordId,
          values: payload.values,
          position: 0, // Will need to be tracked
        },
        timestamp: Date.now(),
      };
    }
    case 'record_delete': {
      const payload = action.payload as RecordDeletePayload;
      return {
        type: 'record_create',
        payload: {
          recordId: payload.recordId,
          values: payload.values,
        },
        timestamp: Date.now(),
      };
    }
    case 'bulk_cell_update': {
      const payload = action.payload as BulkCellUpdatePayload;
      return {
        type: 'bulk_cell_update',
        payload: {
          updates: payload.updates.map((u) => ({
            recordId: u.recordId,
            fieldId: u.fieldId,
            oldValue: u.newValue,
            newValue: u.oldValue,
          })),
        },
        timestamp: Date.now(),
      };
    }
    case 'bulk_record_delete': {
      // Inverse would recreate all deleted records
      // This is complex and would need backend support
      return action;
    }
    default:
      return action;
  }
}
