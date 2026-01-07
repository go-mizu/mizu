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

// Chart Types
export type ChartType =
  | 'line'
  | 'bar'
  | 'column'
  | 'pie'
  | 'doughnut'
  | 'area'
  | 'scatter'
  | 'combo'
  | 'stacked_bar'
  | 'stacked_column'
  | 'stacked_area'
  | 'radar'
  | 'bubble'
  | 'waterfall'
  | 'histogram'
  | 'treemap'
  | 'gauge'
  | 'candlestick';

export interface Chart {
  id: string;
  sheetId: string;
  name: string;
  chartType: ChartType;
  position: ChartPosition;
  size: ChartSize;
  dataRanges: DataRange[];
  title?: ChartTitle;
  subtitle?: ChartTitle;
  legend?: LegendConfig;
  axes?: AxesConfig;
  series?: SeriesConfig[];
  options?: ChartOptions;
  createdAt: string;
  updatedAt: string;
}

export interface ChartPosition {
  row: number;
  col: number;
  offsetX: number;
  offsetY: number;
}

export interface ChartSize {
  width: number;
  height: number;
}

export interface DataRange {
  sheetId?: string;
  startRow: number;
  startCol: number;
  endRow: number;
  endCol: number;
  hasHeader: boolean;
}

export interface ChartTitle {
  text: string;
  fontFamily?: string;
  fontSize?: number;
  fontColor?: string;
  bold?: boolean;
  italic?: boolean;
  position?: 'top' | 'bottom' | 'left' | 'right' | 'none';
}

export interface LegendConfig {
  enabled: boolean;
  position: 'top' | 'bottom' | 'left' | 'right' | 'none';
  alignment?: 'start' | 'center' | 'end';
  fontFamily?: string;
  fontSize?: number;
  fontColor?: string;
}

export interface AxesConfig {
  xAxis?: AxisConfig;
  yAxis?: AxisConfig;
  y2Axis?: AxisConfig;
}

export interface AxisConfig {
  title?: ChartTitle;
  min?: number;
  max?: number;
  stepSize?: number;
  gridLines: boolean;
  gridColor?: string;
  tickColor?: string;
  labelFormat?: string;
  logarithmic?: boolean;
  reversed?: boolean;
}

export interface SeriesConfig {
  name: string;
  dataRange?: DataRange;
  chartType?: ChartType;
  color?: string;
  backgroundColor?: string;
  borderColor?: string;
  borderWidth?: number;
  pointStyle?: 'circle' | 'triangle' | 'rect' | 'star' | 'cross';
  pointRadius?: number;
  fill?: boolean;
  tension?: number;
  yAxisId?: string;
  stack?: string;
  dataLabels?: DataLabels;
  trendline?: Trendline;
}

export interface DataLabels {
  enabled: boolean;
  position: 'top' | 'center' | 'bottom' | 'outside';
  format?: string;
  fontSize?: number;
  fontColor?: string;
}

export interface Trendline {
  type: 'linear' | 'exponential' | 'polynomial' | 'moving_average';
  degree?: number;
  period?: number;
  color?: string;
  width?: number;
  showEquation?: boolean;
  showR2?: boolean;
}

export interface ChartOptions {
  backgroundColor?: string;
  borderColor?: string;
  borderWidth?: number;
  borderRadius?: number;
  animated?: boolean;
  animationDuration?: number;
  interactive?: boolean;
  hoverMode?: 'nearest' | 'point' | 'index' | 'dataset';
  tooltipEnabled?: boolean;
  cutoutPercentage?: number;
  startAngle?: number;
  is3D?: boolean;
  sparkline?: boolean;
}

export interface ChartData {
  labels: string[];
  datasets: ChartDataset[];
  metadata?: unknown;
}

export interface ChartDataset {
  label: string;
  data: number[];
  backgroundColor?: string | string[];
  borderColor?: string | string[];
  borderWidth?: number;
  fill?: boolean;
  tension?: number;
  pointRadius?: number;
  pointStyle?: string;
}

export interface CreateChartRequest {
  sheetId: string;
  name: string;
  chartType: ChartType;
  position: ChartPosition;
  size: ChartSize;
  dataRanges: DataRange[];
  title?: ChartTitle;
  subtitle?: ChartTitle;
  legend?: LegendConfig;
  axes?: AxesConfig;
  series?: SeriesConfig[];
  options?: ChartOptions;
}

export interface UpdateChartRequest {
  name?: string;
  chartType?: ChartType;
  position?: ChartPosition;
  size?: ChartSize;
  dataRanges?: DataRange[];
  title?: ChartTitle;
  subtitle?: ChartTitle;
  legend?: LegendConfig;
  axes?: AxesConfig;
  series?: SeriesConfig[];
  options?: ChartOptions;
}
