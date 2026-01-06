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
} from '../types';

const API_BASE = '/api/v1';

class ApiClient {
  private token: string | null = null;

  setToken(token: string | null) {
    this.token = token;
    if (token) {
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
    return this.request<Workbook>('GET', `/workbooks/${id}`);
  }

  async createWorkbook(req: CreateWorkbookRequest): Promise<Workbook> {
    return this.request<Workbook>('POST', '/workbooks', req);
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
}

export const api = new ApiClient();
