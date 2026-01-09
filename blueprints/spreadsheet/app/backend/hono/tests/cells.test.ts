import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import type BetterSqlite3 from 'better-sqlite3';
import type { Database } from '../src/db/types.js';
import { createTestDb, createTestApp, registerAndLogin, createWorkbookWithSheet } from './setup.js';

describe('Cells API', () => {
  let db: Database;
  let rawDb: BetterSqlite3.Database;
  let app: ReturnType<typeof createTestApp>;
  let token: string;
  let sheetId: string;

  beforeEach(async () => {
    const result = createTestDb();
    db = result.db;
    rawDb = result.rawDb;
    app = createTestApp(db);

    const auth = await registerAndLogin(app);
    token = auth.token;

    const wb = await createWorkbookWithSheet(app, token);
    sheetId = wb.sheetId;
  });

  afterEach(() => {
    rawDb.close();
  });

  describe('PUT /api/v1/sheets/:sheetId/cells/:row/:col', () => {
    it('should create a cell', async () => {
      const res = await app.request(`/api/v1/sheets/${sheetId}/cells/0/0`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ value: 'Hello' }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { cell: { value: string; row: number; col: number } };
      expect(data.cell.value).toBe('Hello');
      expect(data.cell.row).toBe(0);
      expect(data.cell.col).toBe(0);
    });

    it('should update existing cell', async () => {
      // Create cell
      await app.request(`/api/v1/sheets/${sheetId}/cells/0/0`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ value: 'Original' }),
      });

      // Update cell
      const res = await app.request(`/api/v1/sheets/${sheetId}/cells/0/0`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ value: 'Updated' }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { cell: { value: string } };
      expect(data.cell.value).toBe('Updated');
    });

    it('should store formula', async () => {
      const res = await app.request(`/api/v1/sheets/${sheetId}/cells/0/0`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          value: '10',
          formula: '=5+5',
          display: '10',
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { cell: { formula: string } };
      expect(data.cell.formula).toBe('=5+5');
    });

    it('should store format', async () => {
      const format = JSON.stringify({ bold: true, fontSize: 14 });
      const res = await app.request(`/api/v1/sheets/${sheetId}/cells/0/0`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          value: 'Formatted',
          format,
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { cell: { format: string } };
      expect(data.cell.format).toBe(format);
    });
  });

  describe('GET /api/v1/sheets/:sheetId/cells/:row/:col', () => {
    it('should get cell', async () => {
      // Create cell
      await app.request(`/api/v1/sheets/${sheetId}/cells/5/3`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ value: 'Test Value' }),
      });

      // Get cell
      const res = await app.request(`/api/v1/sheets/${sheetId}/cells/5/3`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { cell: { value: string } };
      expect(data.cell.value).toBe('Test Value');
    });

    it('should return null for non-existent cell', async () => {
      const res = await app.request(`/api/v1/sheets/${sheetId}/cells/99/99`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { cell: null };
      expect(data.cell).toBeNull();
    });
  });

  describe('PUT /api/v1/sheets/:sheetId/cells (batch)', () => {
    it('should update multiple cells', async () => {
      const res = await app.request(`/api/v1/sheets/${sheetId}/cells`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          cells: [
            { row: 0, col: 0, value: 'A1' },
            { row: 0, col: 1, value: 'B1' },
            { row: 1, col: 0, value: 'A2' },
          ],
        }),
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { cells: Array<{ value: string }> };
      expect(data.cells).toHaveLength(3);
    });
  });

  describe('GET /api/v1/sheets/:sheetId/cells', () => {
    it('should get all cells for sheet', async () => {
      // Create some cells
      await app.request(`/api/v1/sheets/${sheetId}/cells`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          cells: [
            { row: 0, col: 0, value: 'A1' },
            { row: 0, col: 1, value: 'B1' },
          ],
        }),
      });

      // Get all cells
      const res = await app.request(`/api/v1/sheets/${sheetId}/cells`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { cells: Record<string, { value: string }> };
      expect(Object.keys(data.cells)).toHaveLength(2);
      expect(data.cells['0,0'].value).toBe('A1');
      expect(data.cells['0,1'].value).toBe('B1');
    });
  });

  describe('DELETE /api/v1/sheets/:sheetId/cells/:row/:col', () => {
    it('should delete cell', async () => {
      // Create cell
      await app.request(`/api/v1/sheets/${sheetId}/cells/0/0`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ value: 'To Delete' }),
      });

      // Delete cell
      const deleteRes = await app.request(`/api/v1/sheets/${sheetId}/cells/0/0`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(deleteRes.status).toBe(200);

      // Verify deleted
      const getRes = await app.request(`/api/v1/sheets/${sheetId}/cells/0/0`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      const data = await getRes.json() as { cell: null };
      expect(data.cell).toBeNull();
    });
  });

  describe('Row operations', () => {
    beforeEach(async () => {
      // Create some cells
      await app.request(`/api/v1/sheets/${sheetId}/cells`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          cells: [
            { row: 0, col: 0, value: 'Row 0' },
            { row: 1, col: 0, value: 'Row 1' },
            { row: 2, col: 0, value: 'Row 2' },
          ],
        }),
      });
    });

    it('should insert rows', async () => {
      const res = await app.request(`/api/v1/sheets/${sheetId}/rows/insert`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ start_row: 1, count: 2 }),
      });

      expect(res.status).toBe(200);

      // Check cells were shifted
      const cellsRes = await app.request(`/api/v1/sheets/${sheetId}/cells`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      const data = await cellsRes.json() as { cells: Record<string, { value: string }> };
      expect(data.cells['0,0'].value).toBe('Row 0');
      expect(data.cells['3,0'].value).toBe('Row 1'); // Shifted from row 1 to row 3
      expect(data.cells['4,0'].value).toBe('Row 2'); // Shifted from row 2 to row 4
    });

    it('should delete rows', async () => {
      const res = await app.request(`/api/v1/sheets/${sheetId}/rows/delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ start_row: 1, count: 1 }),
      });

      expect(res.status).toBe(200);

      // Check cells were shifted
      const cellsRes = await app.request(`/api/v1/sheets/${sheetId}/cells`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      const data = await cellsRes.json() as { cells: Record<string, { value: string }> };
      expect(data.cells['0,0'].value).toBe('Row 0');
      expect(data.cells['1,0'].value).toBe('Row 2'); // Shifted from row 2 to row 1
      expect(data.cells['2,0']).toBeUndefined(); // Row 2 is now empty
    });
  });

  describe('Merge operations', () => {
    it('should merge cells', async () => {
      const res = await app.request(`/api/v1/sheets/${sheetId}/merge`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          start_row: 0,
          start_col: 0,
          end_row: 2,
          end_col: 2,
        }),
      });

      expect(res.status).toBe(201);
      const data = await res.json() as { merge: { startRow: number; endRow: number } };
      expect(data.merge.startRow).toBe(0);
      expect(data.merge.endRow).toBe(2);
    });

    it('should get merged regions', async () => {
      // Create merge
      await app.request(`/api/v1/sheets/${sheetId}/merge`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          start_row: 0,
          start_col: 0,
          end_row: 2,
          end_col: 2,
        }),
      });

      // Get merges
      const res = await app.request(`/api/v1/sheets/${sheetId}/merges`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      expect(res.status).toBe(200);
      const data = await res.json() as { merges: Array<{ id: string }> };
      expect(data.merges).toHaveLength(1);
    });

    it('should prevent overlapping merges', async () => {
      // Create first merge
      await app.request(`/api/v1/sheets/${sheetId}/merge`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          start_row: 0,
          start_col: 0,
          end_row: 2,
          end_col: 2,
        }),
      });

      // Try overlapping merge
      const res = await app.request(`/api/v1/sheets/${sheetId}/merge`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          start_row: 1,
          start_col: 1,
          end_row: 3,
          end_col: 3,
        }),
      });

      expect(res.status).toBe(409);
    });

    it('should unmerge cells', async () => {
      // Create merge
      const mergeRes = await app.request(`/api/v1/sheets/${sheetId}/merge`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          start_row: 0,
          start_col: 0,
          end_row: 2,
          end_col: 2,
        }),
      });

      const { merge } = await mergeRes.json() as { merge: { id: string } };

      // Unmerge
      const res = await app.request(`/api/v1/sheets/${sheetId}/unmerge`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ merge_id: merge.id }),
      });

      expect(res.status).toBe(200);

      // Verify unmerged
      const getRes = await app.request(`/api/v1/sheets/${sheetId}/merges`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      const data = await getRes.json() as { merges: Array<{ id: string }> };
      expect(data.merges).toHaveLength(0);
    });
  });
});
