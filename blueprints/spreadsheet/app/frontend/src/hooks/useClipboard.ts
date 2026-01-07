import { create } from 'zustand';
import type { Cell, CellValue, CellFormat, Selection } from '../types';

export interface ClipboardCell {
  row: number;
  col: number;
  value: CellValue;
  formula?: string;
  format?: CellFormat;
  display?: string;
}

export interface ClipboardData {
  cells: ClipboardCell[];
  bounds: Selection;
  source: 'internal' | 'external';
  isCut: boolean;
  sourceSheetId?: string;
}

interface ClipboardState {
  data: ClipboardData | null;
  cutSelection: Selection | null;
  cutSheetId: string | null;

  // Actions
  copy: (cells: Cell[], selection: Selection, sheetId: string) => Promise<void>;
  cut: (cells: Cell[], selection: Selection, sheetId: string) => Promise<void>;
  paste: () => ClipboardData | null;
  getData: () => ClipboardData | null;
  clearCut: () => void;
  isCutActive: (sheetId: string) => boolean;
  getCutSelection: () => Selection | null;
}

// Convert cells to TSV for system clipboard
function cellsToTSV(cells: ClipboardCell[], bounds: Selection): string {
  const rows: string[][] = [];
  const numRows = bounds.endRow - bounds.startRow + 1;
  const numCols = bounds.endCol - bounds.startCol + 1;

  // Initialize grid
  for (let i = 0; i < numRows; i++) {
    rows.push(new Array(numCols).fill(''));
  }

  // Fill in values
  for (const cell of cells) {
    const rowIdx = cell.row - bounds.startRow;
    const colIdx = cell.col - bounds.startCol;
    if (rowIdx >= 0 && rowIdx < numRows && colIdx >= 0 && colIdx < numCols) {
      // Use display value if available, otherwise raw value
      const value = cell.display ?? cell.value ?? '';
      // Escape tabs and newlines
      let strValue = String(value);
      if (strValue.includes('\t') || strValue.includes('\n') || strValue.includes('"')) {
        strValue = '"' + strValue.replace(/"/g, '""') + '"';
      }
      rows[rowIdx][colIdx] = strValue;
    }
  }

  return rows.map(row => row.join('\t')).join('\n');
}

// Parse TSV from system clipboard
export function parseTSV(text: string): CellValue[][] {
  const lines = text.split('\n');
  const result: CellValue[][] = [];

  for (const line of lines) {
    if (line.trim() === '') continue;

    const values: CellValue[] = [];
    let current = '';
    let inQuotes = false;
    let i = 0;

    while (i < line.length) {
      const char = line[i];

      if (inQuotes) {
        if (char === '"') {
          if (line[i + 1] === '"') {
            current += '"';
            i += 2;
          } else {
            inQuotes = false;
            i++;
          }
        } else {
          current += char;
          i++;
        }
      } else {
        if (char === '"') {
          inQuotes = true;
          i++;
        } else if (char === '\t') {
          values.push(parseValue(current));
          current = '';
          i++;
        } else {
          current += char;
          i++;
        }
      }
    }
    values.push(parseValue(current));
    result.push(values);
  }

  return result;
}

// Parse string value to appropriate type
function parseValue(str: string): CellValue {
  const trimmed = str.trim();
  if (trimmed === '') return null;

  // Try to parse as number
  const num = parseFloat(trimmed);
  if (!isNaN(num) && isFinite(num) && String(num) === trimmed) {
    return num;
  }

  // Check for boolean
  if (trimmed.toLowerCase() === 'true') return true;
  if (trimmed.toLowerCase() === 'false') return false;

  return str;
}

export const useClipboardStore = create<ClipboardState>((set, get) => ({
  data: null,
  cutSelection: null,
  cutSheetId: null,

  copy: async (cells, selection, sheetId) => {
    const clipboardCells: ClipboardCell[] = cells.map(cell => ({
      row: cell.row,
      col: cell.col,
      value: cell.value,
      formula: cell.formula,
      format: cell.format,
      display: cell.display,
    }));

    const data: ClipboardData = {
      cells: clipboardCells,
      bounds: selection,
      source: 'internal',
      isCut: false,
      sourceSheetId: sheetId,
    };

    set({ data, cutSelection: null, cutSheetId: null });

    // Copy to system clipboard as TSV
    try {
      const tsv = cellsToTSV(clipboardCells, selection);
      await navigator.clipboard.writeText(tsv);
    } catch (e) {
      console.warn('Failed to copy to system clipboard:', e);
    }
  },

  cut: async (cells, selection, sheetId) => {
    const clipboardCells: ClipboardCell[] = cells.map(cell => ({
      row: cell.row,
      col: cell.col,
      value: cell.value,
      formula: cell.formula,
      format: cell.format,
      display: cell.display,
    }));

    const data: ClipboardData = {
      cells: clipboardCells,
      bounds: selection,
      source: 'internal',
      isCut: true,
      sourceSheetId: sheetId,
    };

    set({
      data,
      cutSelection: selection,
      cutSheetId: sheetId,
    });

    // Copy to system clipboard as TSV
    try {
      const tsv = cellsToTSV(clipboardCells, selection);
      await navigator.clipboard.writeText(tsv);
    } catch (e) {
      console.warn('Failed to copy to system clipboard:', e);
    }
  },

  paste: () => {
    return get().data;
  },

  getData: () => get().data,

  clearCut: () => {
    set((state) => {
      if (state.data?.isCut) {
        return {
          data: { ...state.data, isCut: false },
          cutSelection: null,
          cutSheetId: null,
        };
      }
      return { cutSelection: null, cutSheetId: null };
    });
  },

  isCutActive: (sheetId) => {
    const { cutSelection, cutSheetId } = get();
    return cutSelection !== null && cutSheetId === sheetId;
  },

  getCutSelection: () => get().cutSelection,
}));

// Helper to read from system clipboard
export async function readSystemClipboard(): Promise<CellValue[][] | null> {
  try {
    const text = await navigator.clipboard.readText();
    if (!text) return null;
    return parseTSV(text);
  } catch (e) {
    console.warn('Failed to read system clipboard:', e);
    return null;
  }
}
