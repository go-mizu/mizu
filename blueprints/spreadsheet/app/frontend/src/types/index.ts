// API Types

export interface User {
  id: string;
  email: string;
  name: string;
  avatar?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Workbook {
  id: string;
  name: string;
  ownerId: string;
  settings?: WorkbookSettings;
  createdAt: string;
  updatedAt: string;
}

export interface WorkbookSettings {
  theme?: string;
  autoSave?: boolean;
}

export interface Sheet {
  id: string;
  workbookId: string;
  name: string;
  index: number;
  hidden: boolean;
  color?: string;
  gridColor: string;
  frozenRows: number;
  frozenCols: number;
  defaultRowHeight: number;
  defaultColWidth: number;
  rowHeights?: Record<number, number>;
  colWidths?: Record<number, number>;
  hiddenRows?: number[];
  hiddenCols?: number[];
  createdAt: string;
  updatedAt: string;
}

export interface Cell {
  id: string;
  sheetId: string;
  row: number;
  col: number;
  value: CellValue;
  formula?: string;
  display?: string;
  type: CellType;
  format?: CellFormat;
  hyperlink?: Hyperlink;
  note?: string;
  updatedAt: string;
}

export type CellValue = string | number | boolean | null;

export type CellType = 'text' | 'number' | 'bool' | 'formula' | 'error' | 'date';

export interface CellFormat {
  fontFamily?: string;
  fontSize?: number;
  fontColor?: string;
  bold?: boolean;
  italic?: boolean;
  underline?: boolean;
  strikethrough?: boolean;
  backgroundColor?: string;
  hAlign?: 'left' | 'center' | 'right';
  vAlign?: 'top' | 'middle' | 'bottom';
  wrapText?: boolean;
  textRotation?: number;
  indent?: number;
  borderTop?: Border;
  borderRight?: Border;
  borderBottom?: Border;
  borderLeft?: Border;
  numberFormat?: string;
}

export interface Border {
  style: 'thin' | 'medium' | 'thick' | 'dashed' | 'dotted' | 'double';
  color: string;
}

export interface Hyperlink {
  url: string;
  text?: string;
}

export interface MergedRegion {
  id: string;
  sheetId: string;
  startRow: number;
  startCol: number;
  endRow: number;
  endCol: number;
}

// API Request/Response types

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  name: string;
}

export interface AuthResponse {
  user: User;
  token: string;
}

export interface CreateWorkbookRequest {
  name: string;
}

export interface UpdateWorkbookRequest {
  name?: string;
  settings?: WorkbookSettings;
}

export interface CreateSheetRequest {
  workbookId: string;
  name: string;
}

export interface UpdateSheetRequest {
  name?: string;
  hidden?: boolean;
  color?: string;
  frozenRows?: number;
  frozenCols?: number;
}

export interface SetCellRequest {
  value?: CellValue;
  formula?: string;
  format?: CellFormat;
}

export interface BatchUpdateRequest {
  cells: CellUpdate[];
}

export interface CellUpdate {
  row: number;
  col: number;
  value?: CellValue;
  formula?: string;
  format?: CellFormat;
}

export interface InsertRowsRequest {
  rowIndex: number;
  count: number;
}

export interface DeleteRowsRequest {
  startRow: number;
  count: number;
}

export interface InsertColsRequest {
  colIndex: number;
  count: number;
}

export interface DeleteColsRequest {
  startCol: number;
  count: number;
}

export interface MergeRequest {
  startRow: number;
  startCol: number;
  endRow: number;
  endCol: number;
}

export interface GetRangeParams {
  startRow: number;
  startCol: number;
  endRow: number;
  endCol: number;
}

export interface EvaluateFormulaRequest {
  sheetId: string;
  formula: string;
}

export interface EvaluateFormulaResponse {
  result: CellValue;
  display: string;
}

// API Error
export interface ApiError {
  error: string;
  message: string;
  status: number;
}

// Selection types
export interface Selection {
  startRow: number;
  startCol: number;
  endRow: number;
  endCol: number;
}

export interface CellPosition {
  row: number;
  col: number;
}
