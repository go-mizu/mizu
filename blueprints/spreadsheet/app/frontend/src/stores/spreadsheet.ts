import { create } from 'zustand';
import { api } from '../utils/api';
import type {
  User,
  Workbook,
  Sheet,
  Cell,
  CellValue,
  CellFormat,
  Selection,
  CellPosition,
  MergedRegion,
  Chart,
  CreateChartRequest,
  UpdateChartRequest,
} from '../types';

interface SpreadsheetState {
  // Auth
  user: User | null;
  isAuthenticated: boolean;
  authLoading: boolean;

  // Data
  workbooks: Workbook[];
  currentWorkbook: Workbook | null;
  sheets: Sheet[];
  currentSheet: Sheet | null;
  cells: Map<string, Cell>;
  mergedRegions: MergedRegion[];
  charts: Chart[];
  selectedChart: Chart | null;
  chartEditorOpen: boolean;
  editingChart: Chart | null;

  // UI State
  selection: Selection | null;
  activeCell: CellPosition | null;
  editingCell: CellPosition | null;
  formulaBarValue: string;
  loading: boolean;
  error: string | null;

  // Actions - Auth
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => Promise<void>;
  checkAuth: () => Promise<void>;

  // Actions - Workbooks
  loadWorkbooks: () => Promise<void>;
  loadWorkbook: (id: string) => Promise<void>;
  createWorkbook: (name: string) => Promise<Workbook>;
  renameWorkbook: (id: string, name: string) => Promise<void>;
  deleteWorkbook: (id: string) => Promise<void>;

  // Actions - Sheets
  loadSheets: (workbookId: string) => Promise<void>;
  selectSheet: (sheetId: string) => Promise<void>;
  createSheet: (name: string) => Promise<Sheet>;
  renameSheet: (id: string, name: string) => Promise<void>;
  deleteSheet: (id: string) => Promise<void>;

  // Actions - Cells
  loadCells: (startRow: number, startCol: number, endRow: number, endCol: number) => Promise<void>;
  getCell: (row: number, col: number) => Cell | undefined;
  setCell: (row: number, col: number, value: CellValue, formula?: string) => Promise<void>;
  setCellFormat: (row: number, col: number, format: CellFormat) => Promise<void>;
  deleteCell: (row: number, col: number) => Promise<void>;
  batchUpdateCells: (updates: Array<{ row: number; col: number; value?: CellValue; formula?: string }>) => Promise<void>;
  batchSetCellFormat: (updates: Array<{ row: number; col: number; format: CellFormat }>) => Promise<void>;

  // Actions - Row/Column
  insertRows: (rowIndex: number, count: number) => Promise<void>;
  deleteRows: (startRow: number, count: number) => Promise<void>;
  insertCols: (colIndex: number, count: number) => Promise<void>;
  deleteCols: (startCol: number, count: number) => Promise<void>;

  // Actions - Merges
  loadMerges: () => Promise<void>;
  mergeCells: (startRow: number, startCol: number, endRow: number, endCol: number) => Promise<void>;
  unmergeCells: (startRow: number, startCol: number, endRow: number, endCol: number) => Promise<void>;

  // Actions - Charts
  loadCharts: () => Promise<void>;
  createChart: (data: CreateChartRequest) => Promise<Chart>;
  updateChart: (id: string, data: UpdateChartRequest) => Promise<void>;
  deleteChart: (id: string) => Promise<void>;
  duplicateChart: (id: string) => Promise<void>;
  selectChart: (chart: Chart | null) => void;
  openChartEditor: (chart?: Chart) => void;
  closeChartEditor: () => void;

  // Actions - Selection
  setSelection: (selection: Selection | null) => void;
  setActiveCell: (position: CellPosition | null) => void;
  setEditingCell: (position: CellPosition | null) => void;
  setFormulaBarValue: (value: string) => void;

  // Actions - Utility
  clearError: () => void;
}

const cellKey = (row: number, col: number) => `${row}:${col}`;

export const useSpreadsheetStore = create<SpreadsheetState>((set, get) => ({
  // Initial state
  user: null,
  isAuthenticated: false,
  authLoading: true,

  workbooks: [],
  currentWorkbook: null,
  sheets: [],
  currentSheet: null,
  cells: new Map(),
  mergedRegions: [],
  charts: [],
  selectedChart: null,
  chartEditorOpen: false,
  editingChart: null,

  selection: null,
  activeCell: null,
  editingCell: null,
  formulaBarValue: '',
  loading: false,
  error: null,

  // Auth actions
  login: async (email, password) => {
    set({ loading: true, error: null });
    try {
      const { user } = await api.login({ email, password });
      set({ user, isAuthenticated: true, loading: false });
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Login failed', loading: false });
      throw e;
    }
  },

  register: async (email, password, name) => {
    set({ loading: true, error: null });
    try {
      const { user } = await api.register({ email, password, name });
      set({ user, isAuthenticated: true, loading: false });
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Registration failed', loading: false });
      throw e;
    }
  },

  logout: async () => {
    try {
      await api.logout();
    } finally {
      set({
        user: null,
        isAuthenticated: false,
        workbooks: [],
        currentWorkbook: null,
        sheets: [],
        currentSheet: null,
        cells: new Map(),
        mergedRegions: [],
        charts: [],
        selectedChart: null,
        chartEditorOpen: false,
        editingChart: null,
      });
    }
  },

  checkAuth: async () => {
    set({ authLoading: true });
    try {
      const user = await api.me();
      set({ user, isAuthenticated: true, authLoading: false });
    } catch {
      set({ user: null, isAuthenticated: false, authLoading: false });
    }
  },

  // Workbook actions
  loadWorkbooks: async () => {
    set({ loading: true, error: null });
    try {
      const workbooks = await api.listWorkbooks();
      set({ workbooks, loading: false });
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to load workbooks', loading: false });
    }
  },

  loadWorkbook: async (id) => {
    set({ loading: true, error: null });
    try {
      const workbook = await api.getWorkbook(id);
      const sheets = await api.listSheets(id);
      set({ currentWorkbook: workbook, sheets, loading: false });

      // Auto-select first sheet
      if (sheets.length > 0) {
        await get().selectSheet(sheets[0].id);
      }
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to load workbook', loading: false });
    }
  },

  createWorkbook: async (name) => {
    set({ loading: true, error: null });
    try {
      const workbook = await api.createWorkbook({ name });
      set((state) => ({
        workbooks: [...state.workbooks, workbook],
        loading: false,
      }));
      return workbook;
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to create workbook', loading: false });
      throw e;
    }
  },

  renameWorkbook: async (id, name) => {
    try {
      const workbook = await api.updateWorkbook(id, { name });
      set((state) => ({
        workbooks: state.workbooks.map((w) => (w.id === id ? workbook : w)),
        currentWorkbook: state.currentWorkbook?.id === id ? workbook : state.currentWorkbook,
      }));
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to rename workbook' });
    }
  },

  deleteWorkbook: async (id) => {
    try {
      await api.deleteWorkbook(id);
      set((state) => ({
        workbooks: state.workbooks.filter((w) => w.id !== id),
        currentWorkbook: state.currentWorkbook?.id === id ? null : state.currentWorkbook,
      }));
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to delete workbook' });
    }
  },

  // Sheet actions
  loadSheets: async (workbookId) => {
    try {
      const sheets = await api.listSheets(workbookId);
      set({ sheets });
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to load sheets' });
    }
  },

  selectSheet: async (sheetId) => {
    const { sheets } = get();
    const sheet = sheets.find((s) => s.id === sheetId);
    if (!sheet) return;

    set({ currentSheet: sheet, cells: new Map(), mergedRegions: [], charts: [], selectedChart: null });

    // Load initial cells, merges, and charts
    await Promise.all([
      get().loadCells(0, 0, 100, 26),
      get().loadMerges(),
      get().loadCharts(),
    ]);
  },

  createSheet: async (name) => {
    const { currentWorkbook } = get();
    if (!currentWorkbook) throw new Error('No workbook selected');

    try {
      const sheet = await api.createSheet({ workbookId: currentWorkbook.id, name });
      set((state) => ({ sheets: [...state.sheets, sheet] }));
      return sheet;
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to create sheet' });
      throw e;
    }
  },

  renameSheet: async (id, name) => {
    try {
      const sheet = await api.updateSheet(id, { name });
      set((state) => ({
        sheets: state.sheets.map((s) => (s.id === id ? sheet : s)),
        currentSheet: state.currentSheet?.id === id ? sheet : state.currentSheet,
      }));
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to rename sheet' });
    }
  },

  deleteSheet: async (id) => {
    const { sheets, currentSheet } = get();
    if (sheets.length <= 1) {
      set({ error: 'Cannot delete the last sheet' });
      return;
    }

    try {
      await api.deleteSheet(id);
      const newSheets = sheets.filter((s) => s.id !== id);
      set({ sheets: newSheets });

      if (currentSheet?.id === id && newSheets.length > 0) {
        await get().selectSheet(newSheets[0].id);
      }
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to delete sheet' });
    }
  },

  // Cell actions
  loadCells: async (startRow, startCol, endRow, endCol) => {
    const { currentSheet } = get();
    if (!currentSheet) return;

    try {
      const cellsArray = await api.getCells(currentSheet.id, { startRow, startCol, endRow, endCol });
      set((state) => {
        const cells = new Map(state.cells);
        for (const cell of cellsArray) {
          cells.set(cellKey(cell.row, cell.col), cell);
        }
        return { cells };
      });
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to load cells' });
    }
  },

  getCell: (row, col) => {
    return get().cells.get(cellKey(row, col));
  },

  setCell: async (row, col, value, formula) => {
    const { currentSheet } = get();
    if (!currentSheet) return;

    try {
      const cell = await api.setCell(currentSheet.id, row, col, {
        value: formula ? undefined : value,
        formula,
      });
      set((state) => {
        const cells = new Map(state.cells);
        cells.set(cellKey(row, col), cell);
        return { cells };
      });
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to set cell' });
    }
  },

  setCellFormat: async (row, col, format) => {
    const { currentSheet, cells } = get();
    if (!currentSheet) return;

    try {
      const existingCell = cells.get(cellKey(row, col));
      const cell = await api.setCell(currentSheet.id, row, col, {
        value: existingCell?.value,
        formula: existingCell?.formula,
        format,
      });
      set((state) => {
        const newCells = new Map(state.cells);
        newCells.set(cellKey(row, col), cell);
        return { cells: newCells };
      });
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to format cell' });
    }
  },

  deleteCell: async (row, col) => {
    const { currentSheet } = get();
    if (!currentSheet) return;

    try {
      await api.deleteCell(currentSheet.id, row, col);
      set((state) => {
        const cells = new Map(state.cells);
        cells.delete(cellKey(row, col));
        return { cells };
      });
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to delete cell' });
    }
  },

  batchUpdateCells: async (updates) => {
    const { currentSheet } = get();
    if (!currentSheet) return;

    try {
      const cellsArray = await api.batchUpdateCells(currentSheet.id, {
        cells: updates.map((u) => ({
          row: u.row,
          col: u.col,
          value: u.formula ? undefined : u.value,
          formula: u.formula,
        })),
      });
      set((state) => {
        const cells = new Map(state.cells);
        for (const cell of cellsArray) {
          cells.set(cellKey(cell.row, cell.col), cell);
        }
        return { cells };
      });
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to update cells' });
    }
  },

  // Batch set cell format - optimized to send all format changes in one request
  batchSetCellFormat: async (updates) => {
    const { currentSheet, cells } = get();
    if (!currentSheet || updates.length === 0) return;

    try {
      // Build batch update request with existing cell values and new formats
      const batchUpdates = updates.map((u) => {
        const existingCell = cells.get(cellKey(u.row, u.col));
        return {
          row: u.row,
          col: u.col,
          value: existingCell?.value,
          formula: existingCell?.formula,
          format: u.format,
        };
      });

      const cellsArray = await api.batchUpdateCells(currentSheet.id, {
        cells: batchUpdates,
      });

      set((state) => {
        const newCells = new Map(state.cells);
        for (const cell of cellsArray) {
          newCells.set(cellKey(cell.row, cell.col), cell);
        }
        return { cells: newCells };
      });
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to format cells' });
    }
  },

  // Row/Column actions
  insertRows: async (rowIndex, count) => {
    const { currentSheet } = get();
    if (!currentSheet) return;

    try {
      await api.insertRows(currentSheet.id, { rowIndex, count });
      // Reload cells after row insertion
      set({ cells: new Map() });
      await get().loadCells(0, 0, 100, 26);
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to insert rows' });
    }
  },

  deleteRows: async (startRow, count) => {
    const { currentSheet } = get();
    if (!currentSheet) return;

    try {
      await api.deleteRows(currentSheet.id, { startRow, count });
      set({ cells: new Map() });
      await get().loadCells(0, 0, 100, 26);
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to delete rows' });
    }
  },

  insertCols: async (colIndex, count) => {
    const { currentSheet } = get();
    if (!currentSheet) return;

    try {
      await api.insertCols(currentSheet.id, { colIndex, count });
      set({ cells: new Map() });
      await get().loadCells(0, 0, 100, 26);
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to insert columns' });
    }
  },

  deleteCols: async (startCol, count) => {
    const { currentSheet } = get();
    if (!currentSheet) return;

    try {
      await api.deleteCols(currentSheet.id, { startCol, count });
      set({ cells: new Map() });
      await get().loadCells(0, 0, 100, 26);
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to delete columns' });
    }
  },

  // Merge actions
  loadMerges: async () => {
    const { currentSheet } = get();
    if (!currentSheet) return;

    try {
      const mergedRegions = await api.getMerges(currentSheet.id);
      set({ mergedRegions });
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to load merged regions' });
    }
  },

  mergeCells: async (startRow, startCol, endRow, endCol) => {
    const { currentSheet } = get();
    if (!currentSheet) return;

    try {
      const region = await api.merge(currentSheet.id, { startRow, startCol, endRow, endCol });
      set((state) => ({ mergedRegions: [...state.mergedRegions, region] }));
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to merge cells' });
    }
  },

  unmergeCells: async (startRow, startCol, endRow, endCol) => {
    const { currentSheet } = get();
    if (!currentSheet) return;

    try {
      await api.unmerge(currentSheet.id, { startRow, startCol, endRow, endCol });
      set((state) => ({
        mergedRegions: state.mergedRegions.filter(
          (r) =>
            !(r.startRow === startRow && r.startCol === startCol && r.endRow === endRow && r.endCol === endCol)
        ),
      }));
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to unmerge cells' });
    }
  },

  // Chart actions
  loadCharts: async () => {
    const { currentSheet } = get();
    if (!currentSheet) return;

    try {
      const charts = await api.listCharts(currentSheet.id);
      set({ charts });
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to load charts' });
    }
  },

  createChart: async (data) => {
    try {
      const chart = await api.createChart(data);
      set((state) => ({ charts: [...state.charts, chart] }));
      return chart;
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to create chart' });
      throw e;
    }
  },

  updateChart: async (id, data) => {
    try {
      const chart = await api.updateChart(id, data);
      set((state) => ({
        charts: state.charts.map((c) => (c.id === id ? chart : c)),
        selectedChart: state.selectedChart?.id === id ? chart : state.selectedChart,
      }));
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to update chart' });
    }
  },

  deleteChart: async (id) => {
    try {
      await api.deleteChart(id);
      set((state) => ({
        charts: state.charts.filter((c) => c.id !== id),
        selectedChart: state.selectedChart?.id === id ? null : state.selectedChart,
      }));
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to delete chart' });
    }
  },

  duplicateChart: async (id) => {
    try {
      const chart = await api.duplicateChart(id);
      set((state) => ({ charts: [...state.charts, chart] }));
    } catch (e) {
      const err = e as { message?: string };
      set({ error: err.message || 'Failed to duplicate chart' });
    }
  },

  selectChart: (chart) => set({ selectedChart: chart }),

  openChartEditor: (chart) => set({
    chartEditorOpen: true,
    editingChart: chart || null,
  }),

  closeChartEditor: () => set({
    chartEditorOpen: false,
    editingChart: null,
  }),

  // Selection actions
  setSelection: (selection) => set({ selection }),
  setActiveCell: (position) => {
    set({ activeCell: position });
    if (position) {
      const cell = get().getCell(position.row, position.col);
      set({ formulaBarValue: cell?.formula || cell?.display || '' });
    }
  },
  setEditingCell: (position) => set({ editingCell: position }),
  setFormulaBarValue: (value) => set({ formulaBarValue: value }),

  // Utility
  clearError: () => set({ error: null }),
}));
