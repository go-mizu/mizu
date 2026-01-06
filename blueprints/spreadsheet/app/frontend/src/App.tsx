import { useCallback, useEffect, useMemo } from 'react';
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

// Generate columns
const columns: GridColumn[] = Array.from({ length: NUM_COLS }, (_, i) => ({
  id: getColumnLabel(i),
  title: getColumnLabel(i),
  width: 100,
}));

function App() {
  const {
    user,
    isAuthenticated,
    authLoading,
    currentWorkbook,
    currentSheet,
    sheets,
    activeCell,
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
    setActiveCell,
    setFormulaBarValue,
    clearError,
  } = useSpreadsheetStore();

  // Check auth on mount
  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  // Auto-login in dev mode
  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      // Try dev login
      login('dev@example.com', 'password').catch(() => {
        // Ignore login error in dev mode
      });
    }
  }, [authLoading, isAuthenticated, login]);

  // Load workbooks once authenticated
  useEffect(() => {
    if (isAuthenticated) {
      loadWorkbooks().then(() => {
        // Auto-create or load first workbook
        const store = useSpreadsheetStore.getState();
        if (store.workbooks.length > 0) {
          loadWorkbook(store.workbooks[0].id);
        } else {
          createWorkbook('My Spreadsheet').then((wb) => loadWorkbook(wb.id));
        }
      });
    }
  }, [isAuthenticated, loadWorkbooks, loadWorkbook, createWorkbook]);

  // Get cell content
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

      return {
        kind: GridCellKind.Text,
        allowOverlay: true,
        displayData: cellData.display || String(cellData.value ?? ''),
        data: cellData.formula || String(cellData.value ?? ''),
      };
    },
    [getCell]
  );

  // Handle cell edit
  const onCellEdited = useCallback(
    (cell: Item, newValue: EditableGridCell) => {
      if (newValue.kind !== GridCellKind.Text) return;

      const [col, row] = cell;
      const value = newValue.data;

      if (value === '') {
        // Clear cell
        setCell(row, col, null);
      } else if (value.startsWith('=')) {
        // Formula
        setCell(row, col, null, value);
      } else {
        // Try to parse as number
        const num = parseFloat(value);
        if (!isNaN(num) && isFinite(num)) {
          setCell(row, col, num);
        } else {
          setCell(row, col, value);
        }
      }
    },
    [setCell]
  );

  // Handle selection change
  const onSelectionChange = useCallback(
    (selection: GridSelection) => {
      if (selection.current?.cell) {
        const [col, row] = selection.current.cell;
        setActiveCell({ row, col });
      }
    },
    [setActiveCell]
  );

  // Handle formula bar change
  const handleFormulaBarChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setFormulaBarValue(e.target.value);
    },
    [setFormulaBarValue]
  );

  // Handle formula bar submit
  const handleFormulaBarSubmit = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter' && activeCell) {
        const value = formulaBarValue;
        if (value === '') {
          setCell(activeCell.row, activeCell.col, null);
        } else if (value.startsWith('=')) {
          setCell(activeCell.row, activeCell.col, null, value);
        } else {
          const num = parseFloat(value);
          if (!isNaN(num) && isFinite(num)) {
            setCell(activeCell.row, activeCell.col, num);
          } else {
            setCell(activeCell.row, activeCell.col, value);
          }
        }
      }
    },
    [activeCell, formulaBarValue, setCell]
  );

  // Active cell reference string
  const activeCellRef = useMemo(() => {
    if (!activeCell) return 'A1';
    return `${getColumnLabel(activeCell.col)}${activeCell.row + 1}`;
  }, [activeCell]);

  // Grid selection
  const gridSelection: GridSelection = useMemo(() => {
    if (!activeCell) {
      return {
        columns: CompactSelection.empty(),
        rows: CompactSelection.empty(),
      };
    }
    return {
      columns: CompactSelection.empty(),
      rows: CompactSelection.empty(),
      current: {
        cell: [activeCell.col, activeCell.row],
        range: {
          x: activeCell.col,
          y: activeCell.row,
          width: 1,
          height: 1,
        },
        rangeStack: [],
      },
    };
  }, [activeCell]);

  if (authLoading) {
    return (
      <div className="app loading">
        <div className="loading-spinner">Loading...</div>
      </div>
    );
  }

  return (
    <div className="app">
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

      {error && (
        <div className="error-banner">
          <span>{error}</span>
          <button onClick={clearError}>Dismiss</button>
        </div>
      )}

      <div className="toolbar">
        <div className="toolbar-group">
          <button title="Undo">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M3 10h10a5 5 0 0 1 5 5v2M3 10l5-5M3 10l5 5" />
            </svg>
          </button>
          <button title="Redo">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M21 10H11a5 5 0 0 0-5 5v2M21 10l-5-5M21 10l-5 5" />
            </svg>
          </button>
        </div>
        <div className="toolbar-divider" />
        <div className="toolbar-group">
          <select className="font-select" defaultValue="Arial">
            <option value="Arial">Arial</option>
            <option value="Helvetica">Helvetica</option>
            <option value="Times New Roman">Times New Roman</option>
            <option value="Courier New">Courier New</option>
          </select>
          <select className="size-select" defaultValue="10">
            <option value="8">8</option>
            <option value="9">9</option>
            <option value="10">10</option>
            <option value="11">11</option>
            <option value="12">12</option>
            <option value="14">14</option>
            <option value="18">18</option>
            <option value="24">24</option>
          </select>
        </div>
        <div className="toolbar-divider" />
        <div className="toolbar-group">
          <button title="Bold" className="format-btn">
            <strong>B</strong>
          </button>
          <button title="Italic" className="format-btn">
            <em>I</em>
          </button>
          <button title="Underline" className="format-btn">
            <u>U</u>
          </button>
        </div>
      </div>

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

      <div className="grid-container">
        <DataEditor
          getCellContent={getContent}
          columns={columns}
          rows={NUM_ROWS}
          onCellEdited={onCellEdited}
          gridSelection={gridSelection}
          onGridSelectionChange={onSelectionChange}
          rowMarkers="number"
          smoothScrollX
          smoothScrollY
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

      <footer className="sheet-tabs">
        <button className="add-sheet" onClick={() => createSheet(`Sheet${sheets.length + 1}`)}>
          +
        </button>
        {sheets.map((sheet) => (
          <div
            key={sheet.id}
            className={`sheet-tab ${currentSheet?.id === sheet.id ? 'active' : ''}`}
            onClick={() => selectSheet(sheet.id)}
          >
            {sheet.name}
          </div>
        ))}
      </footer>
    </div>
  );
}

export default App;
