import type {
  User,
  Workbook,
  Sheet,
  Cell,
  MergedRegion,
  LoginRequest,
  RegisterRequest,
  AuthResponse,
  CreateWorkbookRequest,
  UpdateWorkbookRequest,
  CreateSheetRequest,
  UpdateSheetRequest,
  SetCellRequest,
  BatchUpdateRequest,
  InsertRowsRequest,
  DeleteRowsRequest,
  InsertColsRequest,
  DeleteColsRequest,
  MergeRequest,
  GetRangeParams,
  EvaluateFormulaRequest,
  EvaluateFormulaResponse,
  ApiError,
  Chart,
  ChartData,
  CreateChartRequest,
  UpdateChartRequest,
} from '../types';

const API_BASE = '/api/v1';

/**
 * SECURITY NOTE: Authentication tokens are stored in localStorage.
 *
 * This approach has the following security implications:
 * - Tokens are vulnerable to XSS attacks (any JS on the page can access them)
 * - Tokens persist across browser sessions
 *
 * For production applications, consider:
 * - Using HttpOnly cookies for token storage
 * - Implementing CSRF protection
 * - Using short-lived access tokens with refresh tokens
 * - Adding token revocation mechanisms
 */
class ApiClient {
  private token: string | null = null;

  setToken(token: string | null) {
    this.token = token;
    if (token) {
      // SECURITY: localStorage is accessible to any JavaScript on the page
      localStorage.setItem('auth_token', token);
    } else {
      localStorage.removeItem('auth_token');
    }
  }

  getToken(): string | null {
    if (!this.token) {
      this.token = localStorage.getItem('auth_token');
    }
    return this.token;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    const token = this.getToken();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      let error: ApiError;
      try {
        error = await response.json();
      } catch {
        error = {
          error: 'request_failed',
          message: response.statusText,
          status: response.status,
        };
      }
      throw error;
    }

    if (response.status === 204) {
      return undefined as T;
    }

    return response.json();
  }

  // Auth
  async register(req: RegisterRequest): Promise<AuthResponse> {
    const res = await this.request<AuthResponse>('POST', '/auth/register', req);
    this.setToken(res.token);
    return res;
  }

  async login(req: LoginRequest): Promise<AuthResponse> {
    const res = await this.request<AuthResponse>('POST', '/auth/login', req);
    this.setToken(res.token);
    return res;
  }

  async logout(): Promise<void> {
    await this.request<void>('POST', '/auth/logout');
    this.setToken(null);
  }

  async me(): Promise<User> {
    return this.request<User>('GET', '/auth/me');
  }

  // Workbooks
  async listWorkbooks(): Promise<Workbook[]> {
    return this.request<Workbook[]>('GET', '/workbooks');
  }

  async getWorkbook(id: string): Promise<Workbook> {
    const response = await this.request<{ workbook: Workbook; sheets: Sheet[] }>('GET', `/workbooks/${id}`);
    return response.workbook;
  }

  async createWorkbook(req: CreateWorkbookRequest): Promise<Workbook> {
    const response = await this.request<{ workbook: Workbook; sheet: Sheet }>('POST', '/workbooks', req);
    return response.workbook;
  }

  async updateWorkbook(id: string, req: UpdateWorkbookRequest): Promise<Workbook> {
    return this.request<Workbook>('PATCH', `/workbooks/${id}`, req);
  }

  async deleteWorkbook(id: string): Promise<void> {
    return this.request<void>('DELETE', `/workbooks/${id}`);
  }

  async listSheets(workbookId: string): Promise<Sheet[]> {
    return this.request<Sheet[]>('GET', `/workbooks/${workbookId}/sheets`);
  }

  // Sheets
  async getSheet(id: string): Promise<Sheet> {
    return this.request<Sheet>('GET', `/sheets/${id}`);
  }

  async createSheet(req: CreateSheetRequest): Promise<Sheet> {
    return this.request<Sheet>('POST', '/sheets', req);
  }

  async updateSheet(id: string, req: UpdateSheetRequest): Promise<Sheet> {
    return this.request<Sheet>('PATCH', `/sheets/${id}`, req);
  }

  async deleteSheet(id: string): Promise<void> {
    return this.request<void>('DELETE', `/sheets/${id}`);
  }

  // Cells
  async getCell(sheetId: string, row: number, col: number): Promise<Cell | null> {
    try {
      return await this.request<Cell>('GET', `/sheets/${sheetId}/cells/${row}/${col}`);
    } catch (e) {
      const err = e as ApiError;
      if (err.status === 404) return null;
      throw e;
    }
  }

  async getCells(sheetId: string, params: GetRangeParams): Promise<Cell[]> {
    const query = new URLSearchParams({
      startRow: params.startRow.toString(),
      startCol: params.startCol.toString(),
      endRow: params.endRow.toString(),
      endCol: params.endCol.toString(),
    });
    return this.request<Cell[]>('GET', `/sheets/${sheetId}/cells?${query}`);
  }

  async setCell(sheetId: string, row: number, col: number, req: SetCellRequest): Promise<Cell> {
    return this.request<Cell>('PUT', `/sheets/${sheetId}/cells/${row}/${col}`, req);
  }

  async deleteCell(sheetId: string, row: number, col: number): Promise<void> {
    return this.request<void>('DELETE', `/sheets/${sheetId}/cells/${row}/${col}`);
  }

  async batchUpdateCells(sheetId: string, req: BatchUpdateRequest): Promise<Cell[]> {
    return this.request<Cell[]>('PUT', `/sheets/${sheetId}/cells`, req);
  }

  // Row/Column operations
  async insertRows(sheetId: string, req: InsertRowsRequest): Promise<void> {
    return this.request<void>('POST', `/sheets/${sheetId}/rows/insert`, req);
  }

  async deleteRows(sheetId: string, req: DeleteRowsRequest): Promise<void> {
    return this.request<void>('POST', `/sheets/${sheetId}/rows/delete`, req);
  }

  async insertCols(sheetId: string, req: InsertColsRequest): Promise<void> {
    return this.request<void>('POST', `/sheets/${sheetId}/cols/insert`, req);
  }

  async deleteCols(sheetId: string, req: DeleteColsRequest): Promise<void> {
    return this.request<void>('POST', `/sheets/${sheetId}/cols/delete`, req);
  }

  // Merges
  async getMerges(sheetId: string): Promise<MergedRegion[]> {
    return this.request<MergedRegion[]>('GET', `/sheets/${sheetId}/merges`);
  }

  async merge(sheetId: string, req: MergeRequest): Promise<MergedRegion> {
    return this.request<MergedRegion>('POST', `/sheets/${sheetId}/merge`, req);
  }

  async unmerge(sheetId: string, req: MergeRequest): Promise<void> {
    return this.request<void>('POST', `/sheets/${sheetId}/unmerge`, req);
  }

  // Formula
  async evaluateFormula(req: EvaluateFormulaRequest): Promise<EvaluateFormulaResponse> {
    return this.request<EvaluateFormulaResponse>('POST', '/formula/evaluate', req);
  }

  // Import/Export
  async exportWorkbook(
    workbookId: string,
    format: string,
    options?: ExportOptions
  ): Promise<Blob> {
    const params = new URLSearchParams({ format });
    if (options?.formatting) params.append('formatting', 'true');
    if (options?.formulas) params.append('formulas', 'true');
    if (options?.headers) params.append('headers', 'true');
    if (options?.gridlines) params.append('gridlines', 'true');
    if (options?.orientation) params.append('orientation', options.orientation);
    if (options?.paperSize) params.append('paperSize', options.paperSize);
    if (options?.compact) params.append('compact', 'true');
    if (options?.metadata) params.append('metadata', 'true');

    const token = this.getToken();
    const headers: Record<string, string> = {};
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/workbooks/${workbookId}/export?${params}`, {
      method: 'GET',
      headers,
    });

    if (!response.ok) {
      throw new Error(`Export failed: ${response.statusText}`);
    }

    return response.blob();
  }

  async exportSheet(
    sheetId: string,
    format: string,
    options?: ExportOptions
  ): Promise<Blob> {
    const params = new URLSearchParams({ format });
    if (options?.formatting) params.append('formatting', 'true');
    if (options?.formulas) params.append('formulas', 'true');
    if (options?.headers) params.append('headers', 'true');

    const token = this.getToken();
    const headers: Record<string, string> = {};
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/sheets/${sheetId}/export?${params}`, {
      method: 'GET',
      headers,
    });

    if (!response.ok) {
      throw new Error(`Export failed: ${response.statusText}`);
    }

    return response.blob();
  }

  async importToWorkbook(
    workbookId: string,
    file: File,
    options?: ImportOptions,
    onProgress?: (progress: ImportProgress) => void
  ): Promise<ImportResult> {
    const formData = new FormData();
    formData.append('file', file);

    if (options?.hasHeaders) formData.append('hasHeaders', 'true');
    if (options?.skipEmptyRows) formData.append('skipEmptyRows', 'true');
    if (options?.trimWhitespace) formData.append('trimWhitespace', 'true');
    if (options?.autoDetectTypes !== false) formData.append('autoDetectTypes', 'true');
    if (options?.importFormatting) formData.append('importFormatting', 'true');
    if (options?.importFormulas) formData.append('importFormulas', 'true');
    if (options?.importSheet) formData.append('importSheet', options.importSheet);
    if (options?.sheetName) formData.append('sheetName', options.sheetName);
    if (options?.format) formData.append('format', options.format);

    const token = this.getToken();

    // Use XMLHttpRequest for progress tracking
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      const startTime = Date.now();
      let lastLoaded = 0;
      let lastTime = startTime;

      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable && onProgress) {
          const now = Date.now();
          const timeDelta = (now - lastTime) / 1000; // seconds
          const loadedDelta = e.loaded - lastLoaded;
          const speed = timeDelta > 0 ? loadedDelta / timeDelta : 0;

          lastLoaded = e.loaded;
          lastTime = now;

          onProgress({
            phase: 'uploading',
            loaded: e.loaded,
            total: e.total,
            speed,
          });
        }
      });

      xhr.upload.addEventListener('loadend', () => {
        if (onProgress) {
          onProgress({
            phase: 'processing',
            loaded: file.size,
            total: file.size,
            speed: 0,
          });
        }
      });

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            const result = JSON.parse(xhr.responseText);
            resolve(result.data);
          } catch {
            reject(new Error('Invalid response format'));
          }
        } else {
          try {
            const error = JSON.parse(xhr.responseText);
            reject(new Error(error.message || error.error || 'Import failed'));
          } catch {
            reject(new Error(`Import failed: ${xhr.statusText}`));
          }
        }
      });

      xhr.addEventListener('error', () => {
        reject(new Error('Network error during import'));
      });

      xhr.addEventListener('abort', () => {
        reject(new Error('Import was cancelled'));
      });

      xhr.open('POST', `${API_BASE}/workbooks/${workbookId}/import`);

      if (token) {
        xhr.setRequestHeader('Authorization', `Bearer ${token}`);
      }

      xhr.send(formData);
    });
  }

  async importToSheet(
    sheetId: string,
    file: File,
    options?: ImportOptions
  ): Promise<ImportResult> {
    const formData = new FormData();
    formData.append('file', file);

    if (options?.hasHeaders) formData.append('hasHeaders', 'true');
    if (options?.skipEmptyRows) formData.append('skipEmptyRows', 'true');
    if (options?.trimWhitespace) formData.append('trimWhitespace', 'true');
    if (options?.autoDetectTypes !== false) formData.append('autoDetectTypes', 'true');
    if (options?.importFormatting) formData.append('importFormatting', 'true');
    if (options?.importFormulas) formData.append('importFormulas', 'true');
    if (options?.format) formData.append('format', options.format);

    const token = this.getToken();
    const headers: Record<string, string> = {};
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/sheets/${sheetId}/import`, {
      method: 'POST',
      headers,
      body: formData,
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || error.error || 'Import failed');
    }

    const result = await response.json();
    return result.data;
  }

  async getSupportedFormats(): Promise<{ import: string[]; export: string[] }> {
    return this.request<{ import: string[]; export: string[] }>('GET', '/formats');
  }

  // Charts
  async createChart(req: CreateChartRequest): Promise<Chart> {
    return this.request<Chart>('POST', '/charts', req);
  }

  async getChart(id: string): Promise<Chart> {
    return this.request<Chart>('GET', `/charts/${id}`);
  }

  async updateChart(id: string, req: UpdateChartRequest): Promise<Chart> {
    return this.request<Chart>('PATCH', `/charts/${id}`, req);
  }

  async deleteChart(id: string): Promise<void> {
    return this.request<void>('DELETE', `/charts/${id}`);
  }

  async duplicateChart(id: string): Promise<Chart> {
    return this.request<Chart>('POST', `/charts/${id}/duplicate`);
  }

  async getChartData(id: string): Promise<ChartData> {
    return this.request<ChartData>('GET', `/charts/${id}/data`);
  }

  async listCharts(sheetId: string): Promise<Chart[]> {
    return this.request<Chart[]>('GET', `/sheets/${sheetId}/charts`);
  }
}

// Export types
export interface ExportOptions {
  formatting?: boolean;
  formulas?: boolean;
  headers?: boolean;
  gridlines?: boolean;
  orientation?: 'portrait' | 'landscape';
  paperSize?: 'letter' | 'a4' | 'legal';
  compact?: boolean;
  metadata?: boolean;
}

export interface ImportOptions {
  hasHeaders?: boolean;
  skipEmptyRows?: boolean;
  trimWhitespace?: boolean;
  autoDetectTypes?: boolean;
  importFormatting?: boolean;
  importFormulas?: boolean;
  importSheet?: string;
  sheetName?: string;
  format?: string;
}

export interface ImportProgress {
  phase: 'uploading' | 'processing';
  loaded: number;
  total: number;
  speed: number; // bytes per second
}

export interface ImportResult {
  sheetId: string;
  rowsImported: number;
  colsImported: number;
  cellsImported: number;
  warnings?: string[];
}

export const api = new ApiClient();
