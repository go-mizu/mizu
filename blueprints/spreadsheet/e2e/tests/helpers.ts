import { APIRequestContext, Page, expect } from '@playwright/test';

const API_BASE = '/api/v1';

// Types
export interface User {
  id: string;
  email: string;
  name: string;
  avatar?: string;
}

export interface Workbook {
  id: string;
  name: string;
  ownerId: string;
  settings?: Record<string, unknown>;
}

export interface Sheet {
  id: string;
  workbookId: string;
  name: string;
  index: number;
  hidden?: boolean;
  color?: string;
  frozenRows?: number;
  frozenCols?: number;
}

export interface Cell {
  id: string;
  sheetId: string;
  row: number;
  col: number;
  value: unknown;
  formula?: string;
  display?: string;
  type?: string;
}

export interface MergedRegion {
  id: string;
  sheetId: string;
  startRow: number;
  startCol: number;
  endRow: number;
  endCol: number;
}

// API Client for direct API testing
export class APIClient {
  private token: string | null = null;

  constructor(private request: APIRequestContext) {}

  setToken(token: string) {
    this.token = token;
  }

  clearToken() {
    this.token = null;
  }

  private headers() {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }
    return headers;
  }

  // Auth
  async register(email: string, password: string, name: string) {
    const res = await this.request.post(`${API_BASE}/auth/register`, {
      headers: this.headers(),
      data: { email, password, name },
    });
    return res;
  }

  async login(email: string, password: string) {
    const res = await this.request.post(`${API_BASE}/auth/login`, {
      headers: this.headers(),
      data: { email, password },
    });
    return res;
  }

  async me() {
    const res = await this.request.get(`${API_BASE}/auth/me`, {
      headers: this.headers(),
    });
    return res;
  }

  async logout() {
    const res = await this.request.post(`${API_BASE}/auth/logout`, {
      headers: this.headers(),
    });
    return res;
  }

  // Workbooks
  async listWorkbooks() {
    const res = await this.request.get(`${API_BASE}/workbooks`, {
      headers: this.headers(),
    });
    return res;
  }

  async createWorkbook(name: string) {
    const res = await this.request.post(`${API_BASE}/workbooks`, {
      headers: this.headers(),
      data: { name },
    });
    return res;
  }

  async getWorkbook(id: string) {
    const res = await this.request.get(`${API_BASE}/workbooks/${id}`, {
      headers: this.headers(),
    });
    return res;
  }

  async updateWorkbook(id: string, data: { name?: string }) {
    const res = await this.request.patch(`${API_BASE}/workbooks/${id}`, {
      headers: this.headers(),
      data,
    });
    return res;
  }

  async deleteWorkbook(id: string) {
    const res = await this.request.delete(`${API_BASE}/workbooks/${id}`, {
      headers: this.headers(),
    });
    return res;
  }

  async listSheets(workbookId: string) {
    const res = await this.request.get(`${API_BASE}/workbooks/${workbookId}/sheets`, {
      headers: this.headers(),
    });
    return res;
  }

  // Sheets
  async createSheet(workbookId: string, name: string, options?: { index?: number; color?: string }) {
    const res = await this.request.post(`${API_BASE}/sheets`, {
      headers: this.headers(),
      data: { workbookId, name, ...options },
    });
    return res;
  }

  async getSheet(id: string) {
    const res = await this.request.get(`${API_BASE}/sheets/${id}`, {
      headers: this.headers(),
    });
    return res;
  }

  async updateSheet(id: string, data: { name?: string; color?: string; frozenRows?: number; frozenCols?: number }) {
    const res = await this.request.patch(`${API_BASE}/sheets/${id}`, {
      headers: this.headers(),
      data,
    });
    return res;
  }

  async deleteSheet(id: string) {
    const res = await this.request.delete(`${API_BASE}/sheets/${id}`, {
      headers: this.headers(),
    });
    return res;
  }

  // Cells
  async getCells(sheetId: string, range: { startRow: number; startCol: number; endRow: number; endCol: number }) {
    const params = new URLSearchParams({
      startRow: range.startRow.toString(),
      startCol: range.startCol.toString(),
      endRow: range.endRow.toString(),
      endCol: range.endCol.toString(),
    });
    const res = await this.request.get(`${API_BASE}/sheets/${sheetId}/cells?${params}`, {
      headers: this.headers(),
    });
    return res;
  }

  async getCell(sheetId: string, row: number, col: number) {
    const res = await this.request.get(`${API_BASE}/sheets/${sheetId}/cells/${row}/${col}`, {
      headers: this.headers(),
    });
    return res;
  }

  async setCell(sheetId: string, row: number, col: number, data: { value?: unknown; formula?: string }) {
    const res = await this.request.put(`${API_BASE}/sheets/${sheetId}/cells/${row}/${col}`, {
      headers: this.headers(),
      data,
    });
    return res;
  }

  async batchUpdateCells(sheetId: string, cells: Array<{ row: number; col: number; value?: unknown; formula?: string }>) {
    const res = await this.request.put(`${API_BASE}/sheets/${sheetId}/cells`, {
      headers: this.headers(),
      data: { cells },
    });
    return res;
  }

  async deleteCell(sheetId: string, row: number, col: number) {
    const res = await this.request.delete(`${API_BASE}/sheets/${sheetId}/cells/${row}/${col}`, {
      headers: this.headers(),
    });
    return res;
  }

  // Row/Column Operations
  async insertRows(sheetId: string, rowIndex: number, count: number) {
    const res = await this.request.post(`${API_BASE}/sheets/${sheetId}/rows/insert`, {
      headers: this.headers(),
      data: { rowIndex, count },
    });
    return res;
  }

  async deleteRows(sheetId: string, startRow: number, count: number) {
    const res = await this.request.post(`${API_BASE}/sheets/${sheetId}/rows/delete`, {
      headers: this.headers(),
      data: { startRow, count },
    });
    return res;
  }

  async insertCols(sheetId: string, colIndex: number, count: number) {
    const res = await this.request.post(`${API_BASE}/sheets/${sheetId}/cols/insert`, {
      headers: this.headers(),
      data: { colIndex, count },
    });
    return res;
  }

  async deleteCols(sheetId: string, startCol: number, count: number) {
    const res = await this.request.post(`${API_BASE}/sheets/${sheetId}/cols/delete`, {
      headers: this.headers(),
      data: { startCol, count },
    });
    return res;
  }

  // Merges
  async getMerges(sheetId: string) {
    const res = await this.request.get(`${API_BASE}/sheets/${sheetId}/merges`, {
      headers: this.headers(),
    });
    return res;
  }

  async merge(sheetId: string, region: { startRow: number; startCol: number; endRow: number; endCol: number }) {
    const res = await this.request.post(`${API_BASE}/sheets/${sheetId}/merge`, {
      headers: this.headers(),
      data: region,
    });
    return res;
  }

  async unmerge(sheetId: string, region: { startRow: number; startCol: number; endRow: number; endCol: number }) {
    const res = await this.request.post(`${API_BASE}/sheets/${sheetId}/unmerge`, {
      headers: this.headers(),
      data: region,
    });
    return res;
  }

  // Formula
  async evaluateFormula(formula: string, context?: { sheetId?: string; row?: number; col?: number }) {
    const res = await this.request.post(`${API_BASE}/formula/evaluate`, {
      headers: this.headers(),
      data: { formula, context },
    });
    return res;
  }
}

// Test Fixtures Helper
export function generateUniqueEmail(): string {
  return `test-${Date.now()}-${Math.random().toString(36).substring(7)}@example.com`;
}

export async function registerAndLogin(api: APIClient, email?: string, password = 'testpass123', name = 'Test User') {
  const userEmail = email || generateUniqueEmail();

  // Register
  const registerRes = await api.register(userEmail, password, name);
  expect(registerRes.ok()).toBeTruthy();

  const registerData = await registerRes.json();
  api.setToken(registerData.token);

  return { user: registerData.user as User, token: registerData.token as string, email: userEmail, password };
}

export async function createTestWorkbook(api: APIClient, name = 'Test Workbook') {
  const res = await api.createWorkbook(name);
  expect(res.ok()).toBeTruthy();
  const data = await res.json();
  // API returns { workbook, sheet } structure
  return (data.workbook || data) as Workbook;
}

export async function getFirstSheet(api: APIClient, workbookId: string) {
  const res = await api.listSheets(workbookId);
  expect(res.ok()).toBeTruthy();
  const sheets = (await res.json()) as Sheet[];
  expect(sheets.length).toBeGreaterThan(0);
  return sheets[0];
}
