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

    it('should create workbook', async () => {
      const mockWorkbook = { id: '1', name: 'New Workbook' };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockWorkbook),
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
