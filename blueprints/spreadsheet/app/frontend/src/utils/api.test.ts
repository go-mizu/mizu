import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock fetch globally
const mockFetch = vi.fn();
global.fetch = mockFetch;

// Import after mocking fetch
import { api } from './api';

describe('API Client', () => {
  beforeEach(() => {
    mockFetch.mockReset();
    localStorage.clear();
  });

  describe('authentication', () => {
    it('should login and store token', async () => {
      const mockResponse = {
        user: { id: '1', email: 'test@example.com', name: 'Test User' },
        token: 'test-token-123',
      };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await api.login({ email: 'test@example.com', password: 'password' });

      expect(result).toEqual(mockResponse);
      expect(localStorage.getItem('auth_token')).toBe('test-token-123');
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: 'test@example.com', password: 'password' }),
      });
    });

    it('should register and store token', async () => {
      const mockResponse = {
        user: { id: '1', email: 'test@example.com', name: 'Test User' },
        token: 'test-token-123',
      };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      });

      const result = await api.register({
        email: 'test@example.com',
        password: 'password',
        name: 'Test User',
      });

      expect(result).toEqual(mockResponse);
      expect(localStorage.getItem('auth_token')).toBe('test-token-123');
    });

    it('should logout and clear token', async () => {
      localStorage.setItem('auth_token', 'test-token');
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 204,
      });

      await api.logout();

      expect(localStorage.getItem('auth_token')).toBeNull();
    });

    it('should get current user', async () => {
      localStorage.setItem('auth_token', 'test-token');
      const mockUser = { id: '1', email: 'test@example.com', name: 'Test User' };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockUser),
      });

      const result = await api.me();

      expect(result).toEqual(mockUser);
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/auth/me', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer test-token',
        },
        body: undefined,
      });
    });
  });

  describe('workbooks', () => {
    beforeEach(() => {
      localStorage.setItem('auth_token', 'test-token');
    });

    it('should list workbooks', async () => {
      const mockWorkbooks = [
        { id: '1', name: 'Workbook 1' },
        { id: '2', name: 'Workbook 2' },
      ];
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockWorkbooks),
      });

      const result = await api.listWorkbooks();

      expect(result).toEqual(mockWorkbooks);
    });

    it('should get workbook by id', async () => {
      const mockWorkbook = { id: '1', name: 'Test Workbook', ownerId: 'user1' };
      const mockSheets = [
        { id: 'sh1', name: 'Sheet1', workbookId: '1' },
        { id: 'sh2', name: 'Sheet2', workbookId: '1' },
      ];
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ workbook: mockWorkbook, sheets: mockSheets }),
      });

      const result = await api.getWorkbook('1');

      expect(result).toEqual(mockWorkbook);
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/workbooks/1', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer test-token',
        },
        body: undefined,
      });
    });

    it('should create workbook', async () => {
      const mockWorkbook = { id: '1', name: 'New Workbook' };
      const mockSheet = { id: 'sh1', name: 'Sheet1' };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ workbook: mockWorkbook, sheet: mockSheet }),
      });

      const result = await api.createWorkbook({ name: 'New Workbook' });

      expect(result).toEqual(mockWorkbook);
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/workbooks', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer test-token',
        },
        body: JSON.stringify({ name: 'New Workbook' }),
      });
    });

    it('should delete workbook', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 204,
      });

      await api.deleteWorkbook('1');

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/workbooks/1', {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer test-token',
        },
        body: undefined,
      });
    });
  });

  describe('cells', () => {
    beforeEach(() => {
      localStorage.setItem('auth_token', 'test-token');
    });

    it('should set cell value', async () => {
      const mockCell = { id: '1', sheetId: 'sheet1', row: 0, col: 0, value: 'test' };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockCell),
      });

      const result = await api.setCell('sheet1', 0, 0, { value: 'test' });

      expect(result).toEqual(mockCell);
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/sheets/sheet1/cells/0/0', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer test-token',
        },
        body: JSON.stringify({ value: 'test' }),
      });
    });

    it('should set cell formula', async () => {
      const mockCell = { id: '1', sheetId: 'sheet1', row: 0, col: 0, formula: '=A1+B1' };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockCell),
      });

      const result = await api.setCell('sheet1', 0, 0, { formula: '=A1+B1' });

      expect(result).toEqual(mockCell);
    });

    it('should get cells in range', async () => {
      const mockCells = [
        { id: '1', row: 0, col: 0, value: 'A1' },
        { id: '2', row: 0, col: 1, value: 'B1' },
      ];
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockCells),
      });

      const result = await api.getCells('sheet1', {
        startRow: 0,
        startCol: 0,
        endRow: 1,
        endCol: 1,
      });

      expect(result).toEqual(mockCells);
    });
  });

  describe('export', () => {
    beforeEach(() => {
      localStorage.setItem('auth_token', 'test-token');
    });

    it('should export workbook as CSV', async () => {
      const mockBlob = new Blob(['col1,col2\nval1,val2'], { type: 'text/csv' });
      mockFetch.mockResolvedValueOnce({
        ok: true,
        blob: () => Promise.resolve(mockBlob),
      });

      const result = await api.exportWorkbook('wb1', 'csv', {});

      expect(result).toBeInstanceOf(Blob);
      expect(mockFetch).toHaveBeenCalled();
      const call = mockFetch.mock.calls[0];
      expect(call[0]).toContain('/api/v1/workbooks/wb1/export');
      expect(call[0]).toContain('format=csv');
    });

    it('should export workbook as XLSX with options', async () => {
      const mockBlob = new Blob(['xlsx content'], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' });
      mockFetch.mockResolvedValueOnce({
        ok: true,
        blob: () => Promise.resolve(mockBlob),
      });

      const options = { formatting: true, formulas: true };
      const result = await api.exportWorkbook('wb1', 'xlsx', options);

      expect(result).toBeInstanceOf(Blob);
      expect(mockFetch).toHaveBeenCalled();
      const call = mockFetch.mock.calls[0];
      expect(call[0]).toContain('format=xlsx');
      expect(call[0]).toContain('formatting=true');
      expect(call[0]).toContain('formulas=true');
    });

    it('should export sheet', async () => {
      const mockBlob = new Blob(['json content'], { type: 'application/json' });
      mockFetch.mockResolvedValueOnce({
        ok: true,
        blob: () => Promise.resolve(mockBlob),
      });

      const result = await api.exportSheet('sh1', 'json', { metadata: true });

      expect(result).toBeInstanceOf(Blob);
      expect(mockFetch).toHaveBeenCalled();
      const call = mockFetch.mock.calls[0];
      expect(call[0]).toContain('/api/v1/sheets/sh1/export');
      expect(call[0]).toContain('format=json');
    });

    it('should handle export error', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        text: () => Promise.resolve('Unsupported format'),
      });

      await expect(api.exportWorkbook('wb1', 'invalid' as any, {})).rejects.toThrow();
    });
  });

  describe('import', () => {
    beforeEach(() => {
      localStorage.setItem('auth_token', 'test-token');
    });

    it('should import CSV to workbook', async () => {
      const mockData = {
        sheetId: 'new-sheet-1',
        rowsImported: 100,
        colsImported: 5,
        cellsImported: 500,
        warnings: [],
      };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: mockData }),
      });

      const file = new File(['col1,col2\nval1,val2'], 'test.csv', { type: 'text/csv' });
      const result = await api.importToWorkbook('wb1', file, {});

      expect(result).toEqual(mockData);
      expect(mockFetch).toHaveBeenCalled();

      // Verify FormData was used
      const call = mockFetch.mock.calls[0];
      expect(call[0]).toBe('/api/v1/workbooks/wb1/import');
      expect(call[1].method).toBe('POST');
      expect(call[1].body).toBeInstanceOf(FormData);
    });

    it('should import XLSX to sheet with options', async () => {
      const mockData = {
        sheetId: 'sh1',
        rowsImported: 50,
        colsImported: 10,
        cellsImported: 400,
        warnings: ['Some formulas could not be imported'],
      };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: mockData }),
      });

      const file = new File(['xlsx content'], 'test.xlsx', {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
      });
      const options = { hasHeaders: true, importFormatting: true };
      const result = await api.importToSheet('sh1', file, options);

      expect(result).toEqual(mockData);
      expect(result.warnings).toHaveLength(1);
    });

    it('should handle import error', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        text: () => Promise.resolve('Invalid CSV format'),
      });

      const file = new File(['invalid data'], 'test.csv', { type: 'text/csv' });

      await expect(api.importToWorkbook('wb1', file, {})).rejects.toThrow();
    });

    it('should import JSON file', async () => {
      const mockData = {
        sheetId: 'new-sheet-1',
        rowsImported: 10,
        colsImported: 3,
        cellsImported: 30,
        warnings: [],
      };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: mockData }),
      });

      const jsonData = JSON.stringify({
        version: '1.0',
        sheets: [{ name: 'Sheet1', cells: [] }],
      });
      const file = new File([jsonData], 'data.json', { type: 'application/json' });
      const result = await api.importToWorkbook('wb1', file, { importFormulas: true });

      expect(result).toEqual(mockData);
    });
  });

  describe('error handling', () => {
    it('should throw error on non-ok response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        json: () => Promise.resolve({ error: 'unauthorized', message: 'Invalid token' }),
      });

      await expect(api.me()).rejects.toEqual({
        error: 'unauthorized',
        message: 'Invalid token',
      });
    });

    it('should handle network errors', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      await expect(api.me()).rejects.toThrow('Network error');
    });
  });
});
